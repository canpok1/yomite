package core

// Config はアプリケーション全体の設定を保持する。
type Config struct {
	DefaultProvider string              `json:"default_provider"`
	DefaultPersona  string              `json:"default_persona"`
	Providers       map[string]Provider `json:"providers"`
	Personas        map[string]Persona  `json:"personas"`
}

// Provider はLLMプロバイダの接続情報を表す。
type Provider struct {
	Type   string `json:"type"`
	Model  string `json:"model"`
	Origin string `json:"origin"`
}

// Persona はAI読者の人格設定を表す。
type Persona struct {
	DisplayName    string `json:"display_name"`
	SystemPrompt   string `json:"system_prompt"`
	MemoryCapacity int    `json:"memory_capacity"`
	MaxSteps       int    `json:"max_steps"`
}
