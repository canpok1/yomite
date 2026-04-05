package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/canpok1/yomite/internal/core/templates"
)

// Phase はLLM呼び出しの段階を表す。
type Phase int

const (
	// PhaseNote は感想生成段階（note + next_index を生成）。
	PhaseNote Phase = iota
	// PhaseMemory はメモリ生成段階（memory を生成）。
	PhaseMemory
)

// Provider はLLMプロバイダのインターフェースを定義する。
type Provider interface {
	Execute(req SimulationRequest) (SimulationResponse, error)
}

// SimulationRequest はシミュレーションの1ステップでAIに送る入力を表す。
type SimulationRequest struct {
	Phase           Phase  // LLM呼び出しの段階
	SystemPrompt    string // ペルソナのsystem_prompt
	CurrentSentence string // 現在の文
	CurrentIndex    int    // 文のインデックス
	TotalSentences  int    // 文の総数
	Memory          string // 記憶バッファ
	MemoryCapacity  int    // 記憶バッファの最大文字数
	// NOTE: Note は PhaseMemory 時にのみ参照される。PhaseNote 時は nil であること。
	Note *Note
}

// SimulationResponse はAIからの1ステップの出力を表す。
type SimulationResponse struct {
	CurrentIndex    int    `json:"current_index"`
	NextIndex       *int   `json:"next_index"` // nil = 読了
	Note            *Note  `json:"note"`       // nil = 感想なし
	Memory          string `json:"memory"`
	RawResponseText string `json:"-"` // デバッグ用。ユーザー向けJSON出力には含まれない
}

// notePromptTmpl は感想生成段階のテンプレート。
var notePromptTmpl = template.Must(
	template.ParseFS(templates.FS, "note_prompt.tmpl"),
)

// memoryPromptTmpl はメモリ生成段階のテンプレート。
var memoryPromptTmpl = template.Must(
	template.ParseFS(templates.FS, "memory_prompt.tmpl"),
)

// promptData はプロンプトテンプレートに渡す共通データ。
type promptData struct {
	CurrentIndex    int
	TotalSentences  int
	CurrentSentence string
	MemorySection   string
	MemoryCapacity  int
	NoteSection     string // メモリ生成テンプレートでのみ使用
}

// BuildPrompt はSimulationRequestのPhaseに応じて適切なプロンプトを構築する。
func BuildPrompt(req SimulationRequest) (system string, user string) {
	switch req.Phase {
	case PhaseNote:
		return BuildNotePrompt(req)
	case PhaseMemory:
		return BuildMemoryPrompt(req)
	default:
		panic(fmt.Sprintf("unknown phase: %d", req.Phase))
	}
}

func placeholderIfEmpty(s string) string {
	if s == "" {
		return "（なし）"
	}
	return s
}

func noteSection(note *Note) string {
	if note == nil {
		return "（なし）"
	}
	return fmt.Sprintf("[%s] %s", note.Type, note.Content)
}

func executeTemplate(tmpl *template.Template, data any) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(fmt.Sprintf("failed to execute template: %v", err))
	}
	return buf.String()
}

// BuildNotePrompt は感想生成段階のプロンプトを構築する。
func BuildNotePrompt(req SimulationRequest) (system string, user string) {
	return req.SystemPrompt, executeTemplate(notePromptTmpl, promptData{
		CurrentIndex:    req.CurrentIndex,
		TotalSentences:  req.TotalSentences,
		CurrentSentence: req.CurrentSentence,
		MemorySection:   placeholderIfEmpty(req.Memory),
		MemoryCapacity:  req.MemoryCapacity,
	})
}

// BuildMemoryPrompt はメモリ生成段階のプロンプトを構築する。
func BuildMemoryPrompt(req SimulationRequest) (system string, user string) {
	return req.SystemPrompt, executeTemplate(memoryPromptTmpl, promptData{
		CurrentIndex:    req.CurrentIndex,
		TotalSentences:  req.TotalSentences,
		CurrentSentence: req.CurrentSentence,
		MemorySection:   placeholderIfEmpty(req.Memory),
		MemoryCapacity:  req.MemoryCapacity,
		NoteSection:     noteSection(req.Note),
	})
}

// ErrInvalidJSON はAIレスポンスが不正なJSONの場合のエラーを表す。
type ErrInvalidJSON struct {
	Raw string
	Err error
}

func (e *ErrInvalidJSON) Error() string {
	return fmt.Sprintf("LLMが不正なJSON応答を返しました: %v\n応答内容: %s", e.Err, e.Raw)
}

func (e *ErrInvalidJSON) Unwrap() error {
	return e.Err
}

// ErrIndexOutOfRange はAIレスポンスのインデックスが範囲外の場合のエラーを表す。
type ErrIndexOutOfRange struct {
	Field string
	Index int
	Max   int
}

func (e *ErrIndexOutOfRange) Error() string {
	return fmt.Sprintf("index out of range: %s=%d (valid range: 0-%d)", e.Field, e.Index, e.Max-1)
}

// stripMarkdownCodeBlock はLLM応答からmarkdownコードブロック（```json ... ```）を除去する。
// NOTE: LLMはプロンプトで「JSON以外のテキストは含めないでください」と指示しても
// markdownコードブロックで応答をラップすることがあるため、パース前に除去する。
func stripMarkdownCodeBlock(text string) string {
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "```") {
		// 最初の行（```json や ```）を除去
		if idx := strings.Index(trimmed, "\n"); idx != -1 {
			trimmed = trimmed[idx+1:]
		}
		// 末尾の ``` を除去
		trimmed, _ = strings.CutSuffix(trimmed, "```")
		return strings.TrimSpace(trimmed)
	}
	return trimmed
}

// ParseResponse はPhaseに応じて適切なパース処理を呼び出す。
func ParseResponse(text string, totalSentences int, phase Phase) (SimulationResponse, error) {
	switch phase {
	case PhaseNote:
		return ParseNoteResponse(text, totalSentences)
	case PhaseMemory:
		return ParseMemoryResponse(text)
	default:
		panic(fmt.Sprintf("unknown phase: %d", phase))
	}
}

// noteResponse は感想生成段階のLLM応答をパースするための構造体。
type noteResponse struct {
	CurrentIndex int   `json:"current_index"`
	NextIndex    *int  `json:"next_index"`
	Note         *Note `json:"note"`
}

// ParseNoteResponse は感想生成段階のAI出力をパースし、インデックスの範囲を検証する。
func ParseNoteResponse(text string, totalSentences int) (SimulationResponse, error) {
	var resp SimulationResponse
	resp.RawResponseText = text

	cleaned := stripMarkdownCodeBlock(text)
	var nr noteResponse
	if err := json.Unmarshal([]byte(cleaned), &nr); err != nil {
		return resp, &ErrInvalidJSON{Raw: text, Err: err}
	}

	resp.CurrentIndex = nr.CurrentIndex
	resp.NextIndex = nr.NextIndex
	resp.Note = nr.Note

	if resp.CurrentIndex < 0 || resp.CurrentIndex >= totalSentences {
		return resp, &ErrIndexOutOfRange{
			Field: "current_index",
			Index: resp.CurrentIndex,
			Max:   totalSentences,
		}
	}

	if resp.NextIndex != nil {
		idx := *resp.NextIndex
		// NOTE: LLMが最後の文を読んだ後に next_index=totalSentences を返すことがある。
		// これは「次の文へ進む」意図だが範囲外なので、読了（null）として扱う。
		if idx == totalSentences {
			resp.NextIndex = nil
		} else if idx < 0 || idx >= totalSentences {
			return resp, &ErrIndexOutOfRange{
				Field: "next_index",
				Index: idx,
				Max:   totalSentences,
			}
		}
	}

	return resp, nil
}

// memoryResponse はメモリ生成段階のLLM応答をパースするための構造体。
type memoryResponse struct {
	Memory string `json:"memory"`
}

// ParseMemoryResponse はメモリ生成段階のAI出力をパースする。
func ParseMemoryResponse(text string) (SimulationResponse, error) {
	var resp SimulationResponse
	resp.RawResponseText = text

	cleaned := stripMarkdownCodeBlock(text)
	var mr memoryResponse
	if err := json.Unmarshal([]byte(cleaned), &mr); err != nil {
		return resp, &ErrInvalidJSON{Raw: text, Err: err}
	}

	resp.Memory = mr.Memory

	return resp, nil
}
