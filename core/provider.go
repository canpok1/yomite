package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/canpok1/yomite/core/templates"
)

// Provider はLLMプロバイダのインターフェースを定義する。
type Provider interface {
	Execute(req SimulationRequest) (SimulationResponse, error)
}

// SimulationRequest はシミュレーションの1ステップでAIに送る入力を表す。
type SimulationRequest struct {
	SystemPrompt    string // ペルソナのsystem_prompt
	CurrentSentence string // 現在の文
	CurrentIndex    int    // 文のインデックス
	TotalSentences  int    // 文の総数
	Memory          string // 記憶バッファ
}

// SimulationResponse はAIからの1ステップの出力を表す。
type SimulationResponse struct {
	CurrentIndex    int    `json:"current_index"`
	NextIndex       *int   `json:"next_index"` // nil = 読了
	Note            *Note  `json:"note"`       // nil = 感想なし
	Memory          string `json:"memory"`
	RawResponseText string `json:"-"` // デバッグ用。ユーザー向けJSON出力には含まれない
}

// userPromptTmpl はユーザープロンプトのテンプレート。ビルド時に埋め込まれたファイルからパースする。
var userPromptTmpl = template.Must(
	template.ParseFS(templates.FS, "user_prompt.tmpl"),
)

// promptData はユーザープロンプトテンプレートに渡すデータ。
type promptData struct {
	CurrentIndex    int
	TotalSentences  int
	CurrentSentence string
	MemorySection   string
}

// BuildPrompt はSimulationRequestからLLMに送るシステムプロンプトとユーザーメッセージを構築する。
func BuildPrompt(req SimulationRequest) (system string, user string) {
	system = req.SystemPrompt

	memorySection := "（なし）"
	if req.Memory != "" {
		memorySection = req.Memory
	}

	var buf bytes.Buffer
	if err := userPromptTmpl.Execute(&buf, promptData{
		CurrentIndex:    req.CurrentIndex,
		TotalSentences:  req.TotalSentences,
		CurrentSentence: req.CurrentSentence,
		MemorySection:   memorySection,
	}); err != nil {
		panic(fmt.Sprintf("failed to execute user prompt template: %v", err))
	}
	user = buf.String()

	return system, user
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

// ParseResponse はAIのテキスト出力からSimulationResponseをパースし、インデックスの範囲を検証する。
func ParseResponse(text string, totalSentences int) (SimulationResponse, error) {
	var resp SimulationResponse

	resp.RawResponseText = text

	cleaned := stripMarkdownCodeBlock(text)
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return resp, &ErrInvalidJSON{Raw: text, Err: err}
	}

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
