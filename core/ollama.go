package core

// OllamaProvider はOllama APIを使用するLLMプロバイダ実装。
type OllamaProvider struct {
	origin string
	model  string
}

// NewOllamaProvider は新しいOllamaProviderを生成する。
func NewOllamaProvider(origin, model string) *OllamaProvider {
	return &OllamaProvider{
		origin: origin,
		model:  model,
	}
}
