package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
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
	LastIndex       int // TotalSentences - 1（テンプレートで最後の文の位置を表示するため）
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
		LastIndex:       req.TotalSentences - 1,
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
		LastIndex:       req.TotalSentences - 1,
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
func ParseResponse(text string, currentIdx, totalSentences int, phase Phase) (SimulationResponse, error) {
	switch phase {
	case PhaseNote:
		return ParseNoteResponse(text, currentIdx, totalSentences)
	case PhaseMemory:
		return ParseMemoryResponse(text)
	default:
		panic(fmt.Sprintf("unknown phase: %d", phase))
	}
}

// noteResponse は感想生成段階のLLM応答をパースするための構造体。
// NOTE: LLMにはJSON全体ではなくフィールド値のみを返させ、
// current_index や next_index の計算はプログラム側で行う。
type noteResponse struct {
	NextAction  string  `json:"next_action"`
	Feeling     *string `json:"feeling"`
	FeelingType *string `json:"feeling_type"`
}

// ErrInvalidNextAction はnext_actionの値が不正な場合のエラーを表す。
// NOTE: 内部エラーをラップしないため Unwrap は実装しない。
type ErrInvalidNextAction struct {
	Value string
}

func (e *ErrInvalidNextAction) Error() string {
	return fmt.Sprintf("invalid next_action value: %q (expected \"next\", \"back:N\", or \"finish\")", e.Value)
}

// feelingTypeToNoteType はLLMのfeeling_type文字列をNoteTypeに変換する。
func feelingTypeToNoteType(ft string) (NoteType, error) {
	switch strings.ToUpper(ft) {
	case "QUESTION":
		return NoteTypeQuestion, nil
	case "RESOLVED":
		return NoteTypeResolved, nil
	case "CONFUSION":
		return NoteTypeConfusion, nil
	default:
		return "", fmt.Errorf("unknown feeling_type: %q", ft)
	}
}

// parseNextAction はnext_action文字列をcurrentIdxとtotalSentencesから次のインデックスに変換する。
// 読了の場合はnilを返す。
func parseNextAction(action string, currentIdx, totalSentences int) (*int, error) {
	switch {
	case action == "next":
		next := currentIdx + 1
		if next >= totalSentences {
			return nil, nil // 読了
		}
		return &next, nil
	case action == "finish":
		return nil, nil // 読了
	case strings.HasPrefix(action, "back:"):
		nStr := strings.TrimPrefix(action, "back:")
		n, err := strconv.Atoi(nStr)
		if err != nil || n <= 0 {
			return nil, &ErrInvalidNextAction{Value: action}
		}
		next := max(currentIdx-n, 0)
		return &next, nil
	default:
		return nil, &ErrInvalidNextAction{Value: action}
	}
}

// ParseNoteResponse は感想生成段階のAI出力をパースし、プログラム側でSimulationResponseに組み立てる。
func ParseNoteResponse(text string, currentIdx, totalSentences int) (SimulationResponse, error) {
	var resp SimulationResponse
	resp.RawResponseText = text
	resp.CurrentIndex = currentIdx

	cleaned := stripMarkdownCodeBlock(text)
	var nr noteResponse
	if err := json.Unmarshal([]byte(cleaned), &nr); err != nil {
		return resp, &ErrInvalidJSON{Raw: text, Err: err}
	}

	nextIdx, err := parseNextAction(nr.NextAction, currentIdx, totalSentences)
	if err != nil {
		return resp, err
	}
	resp.NextIndex = nextIdx

	if nr.Feeling == nil || *nr.Feeling == "" {
		return resp, nil
	}

	ft := "question" // デフォルト（feelingTypeToNoteType内でToUpperされる）
	if nr.FeelingType != nil && *nr.FeelingType != "" {
		ft = *nr.FeelingType
	}
	noteType, err := feelingTypeToNoteType(ft)
	if err != nil {
		return resp, &ErrInvalidJSON{Raw: text, Err: err}
	}
	resp.Note = &Note{
		Type:    noteType,
		Content: *nr.Feeling,
	}
	return resp, nil
}

// ParseMemoryResponse はメモリ生成段階のAI出力をパースする。
// NOTE: LLMにはプレーンテキストで応答させるため、JSON解析は行わない。
// markdownコードブロックが混入した場合のみ除去する。
func ParseMemoryResponse(text string) (SimulationResponse, error) {
	var resp SimulationResponse
	resp.RawResponseText = text
	resp.Memory = stripMarkdownCodeBlock(text)
	return resp, nil
}
