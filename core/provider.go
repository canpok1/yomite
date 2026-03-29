package core

import "fmt"

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
