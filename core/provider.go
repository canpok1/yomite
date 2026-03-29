package core

import (
	"encoding/json"
	"fmt"
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
	CurrentIndex int    `json:"current_index"`
	NextIndex    *int   `json:"next_index"` // nil = 読了
	Note         *Note  `json:"note"`       // nil = 感想なし
	Memory       string `json:"memory"`
}

// BuildPrompt はSimulationRequestからLLMに送るシステムプロンプトとユーザーメッセージを構築する。
func BuildPrompt(req SimulationRequest) (system string, user string) {
	system = req.SystemPrompt

	memorySection := "（なし）"
	if req.Memory != "" {
		memorySection = req.Memory
	}

	user = fmt.Sprintf(`## 現在の文
位置: %d / %d（0始まり）
内容: %s

## 記憶バッファ
%s

## 指示
上記の文を読み、以下のJSON形式で応答してください。JSON以外のテキストは含めないでください。

{
  "current_index": %d,
  "next_index": <次に読む文のインデックス（整数）。読了する場合はnull>,
  "note": <感想がある場合は{"type": "QUESTION"|"RESOLVED"|"CONFUSION", "content": "感想の内容"}、なければnull>,
  "memory": "<記憶バッファの更新内容（自由テキスト）>"
}`, req.CurrentIndex, req.TotalSentences, req.CurrentSentence,
		memorySection,
		req.CurrentIndex)

	return system, user
}

// ErrInvalidJSON はAIレスポンスが不正なJSONの場合のエラーを表す。
type ErrInvalidJSON struct {
	Raw string
	Err error
}

func (e *ErrInvalidJSON) Error() string {
	return fmt.Sprintf("invalid JSON response: %v", e.Err)
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

// ParseResponse はAIのテキスト出力からSimulationResponseをパースし、インデックスの範囲を検証する。
func ParseResponse(text string, totalSentences int) (SimulationResponse, error) {
	var resp SimulationResponse

	if err := json.Unmarshal([]byte(text), &resp); err != nil {
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
		if idx < 0 || idx >= totalSentences {
			return resp, &ErrIndexOutOfRange{
				Field: "next_index",
				Index: idx,
				Max:   totalSentences,
			}
		}
	}

	return resp, nil
}
