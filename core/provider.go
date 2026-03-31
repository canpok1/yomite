package core

import (
	"encoding/json"
	"fmt"
	"strings"
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

	user = fmt.Sprintf(`
これから読む対象を教えます。
教えるのは文書全体の中の一文だけですが、それを読んで、頭に思い浮かべたことや次に読む場所、覚えておきたいことを出力してください。
出力は必ずJSON形式で行い、JSON以外のテキストは含めないこと。

## ルール
- 基本的にすべての文を順番に読み進めてください。通常は next_index に現在の位置+1 を指定します。
- next_index を null にできるのは、最後の文を読み終えた場合のみです。
- 疑問や混乱が生じた場合は、前の文に戻って読み返すこともできます（next_index に現在より小さい値を指定）。
- 途中で読了（null）を返さないでください。必ず最後の文まで読み切ってください。

## 読む対象
- 読む文の位置: %d
- 全体の文数: %d
- 今回の文: %s

## 覚えていること
%s

## 出力形式
- current_index: 読んだ文の位置
- next_index: 次に読む文の位置。読了する場合のみ null
- note: 感想。感想がなければ null
- note.type: 感想の種類（QUESTION, RESOLVED, CONFUSION）
- note.content: 文を読んだ感想（頭に思い浮かべたこと）
- memory: 覚えておきたいこと。なお次の文を読む際、memoryの値が「今覚えていること」の値として利用される

`+"```"+`
{
  "current_index": <今読んだ文の位置（整数）>,
  "next_index": <次に読む文の位置（整数）。読み終えた場合のみnull>,
  "note": <感想がある場合は{"type": "QUESTION"|"RESOLVED"|"CONFUSION", "content": "感想の内容"}、なければnull>,
  "memory": "<次の文を読む際に覚えていたいこと（自由テキスト）>"
}`+"```",
		req.CurrentIndex,
		req.TotalSentences,
		req.CurrentSentence,
		memorySection)

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
