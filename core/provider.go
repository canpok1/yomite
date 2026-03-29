package core

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
