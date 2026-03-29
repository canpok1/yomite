package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaProvider はOllama APIを使用するLLMプロバイダ実装。
type OllamaProvider struct {
	origin     string
	model      string
	httpClient *http.Client
}

// NewOllamaProvider は新しいOllamaProviderを生成する。
func NewOllamaProvider(origin, model string) *OllamaProvider {
	return &OllamaProvider{
		origin: origin,
		model:  model,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

// Execute はOllama /api/chat にリクエストを送り、SimulationResponseを返す。
func (p *OllamaProvider) Execute(req SimulationRequest) (SimulationResponse, error) {
	system, user := BuildPrompt(req)

	chatReq := ollamaChatRequest{
		Model: p.model,
		Messages: []ollamaMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Stream: false,
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return SimulationResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := p.httpClient.Post(p.origin+"/api/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return SimulationResponse{}, fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return SimulationResponse{}, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return SimulationResponse{}, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	return ParseResponse(chatResp.Message.Content, req.TotalSentences)
}
