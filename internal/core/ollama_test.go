package core

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newOllamaChatServer はテスト用のOllama /api/chat サーバーを返す。
// content にはLLMの出力テキストを指定する。
func newOllamaChatServer(content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"message": map[string]any{
				"role":    "assistant",
				"content": content,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestNewOllamaProvider(t *testing.T) {
	origin := "http://localhost:11434"
	model := "gemma2"

	p := NewOllamaProvider(origin, model)

	if p.origin != origin {
		t.Errorf("origin: got %q, want %q", p.origin, origin)
	}
	if p.model != model {
		t.Errorf("model: got %q, want %q", p.model, model)
	}
}

func TestOllamaProvider_ImplementsProvider(t *testing.T) {
	var _ Provider = (*OllamaProvider)(nil)
}

func TestOllamaProvider_Execute_NotePhase(t *testing.T) {
	content := `{"next_action": "next", "feeling": null, "feeling_type": null}`

	server := newOllamaChatServer(content)
	defer server.Close()

	p := NewOllamaProvider(server.URL, "gemma2")
	req := SimulationRequest{
		Phase:           PhaseNote,
		SystemPrompt:    "あなたは初学者です。",
		CurrentSentence: "これはテスト文です。",
		CurrentIndex:    0,
		TotalSentences:  3,
		Memory:          "",
	}

	got, err := p.Execute(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.CurrentIndex != 0 {
		t.Errorf("CurrentIndex: got %d, want 0", got.CurrentIndex)
	}
	if got.NextIndex == nil || *got.NextIndex != 1 {
		t.Errorf("NextIndex: got %v, want 1", got.NextIndex)
	}
}

func TestOllamaProvider_Execute_MemoryPhase(t *testing.T) {
	content := "テスト記憶"

	server := newOllamaChatServer(content)
	defer server.Close()

	p := NewOllamaProvider(server.URL, "gemma2")
	req := SimulationRequest{
		Phase:           PhaseMemory,
		SystemPrompt:    "あなたは初学者です。",
		CurrentSentence: "これはテスト文です。",
		CurrentIndex:    0,
		TotalSentences:  3,
		Memory:          "",
		MemoryCapacity:  100,
	}

	got, err := p.Execute(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Memory != "テスト記憶" {
		t.Errorf("Memory: got %q, want %q", got.Memory, "テスト記憶")
	}
}

func TestOllamaProvider_Execute_RequestFormat(t *testing.T) {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method: got %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/chat" {
			t.Errorf("Path: got %s, want /api/chat", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		resp := map[string]any{
			"message": map[string]any{
				"role":    "assistant",
				"content": `{"next_action": "next", "feeling": null, "feeling_type": null}`,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL, "gemma2")
	req := SimulationRequest{
		SystemPrompt:    "システムプロンプト",
		CurrentSentence: "テスト文",
		CurrentIndex:    0,
		TotalSentences:  2,
		Memory:          "記憶",
	}

	_, _ = p.Execute(req)

	if model, ok := receivedBody["model"].(string); !ok || model != "gemma2" {
		t.Errorf("model: got %v, want %q", receivedBody["model"], "gemma2")
	}

	if stream, ok := receivedBody["stream"].(bool); !ok || stream {
		t.Errorf("stream: got %v, want false", receivedBody["stream"])
	}

	messages, ok := receivedBody["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages: got %v, want 2 messages", receivedBody["messages"])
	}

	sysMsg := messages[0].(map[string]any)
	if sysMsg["role"] != "system" {
		t.Errorf("messages[0].role: got %v, want system", sysMsg["role"])
	}

	userMsg := messages[1].(map[string]any)
	if userMsg["role"] != "user" {
		t.Errorf("messages[1].role: got %v, want user", userMsg["role"])
	}
}

func TestOllamaProvider_Execute_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL, "nonexistent")
	req := SimulationRequest{
		SystemPrompt:    "test",
		CurrentSentence: "test",
		TotalSentences:  1,
	}

	_, err := p.Execute(req)
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should contain status code 404: got %v", err)
	}
}

func TestOllamaProvider_Execute_ConnectionRefused(t *testing.T) {
	p := NewOllamaProvider("http://127.0.0.1:1", "gemma2")
	req := SimulationRequest{
		SystemPrompt:    "test",
		CurrentSentence: "test",
		TotalSentences:  1,
	}

	_, err := p.Execute(req)
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
	if !strings.Contains(err.Error(), "failed to send request to Ollama") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOllamaProvider_Execute_InvalidOllamaJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL, "gemma2")
	req := SimulationRequest{
		SystemPrompt:    "test",
		CurrentSentence: "test",
		TotalSentences:  1,
	}

	_, err := p.Execute(req)
	if err == nil {
		t.Fatal("expected error for invalid Ollama response JSON")
	}
	if !strings.Contains(err.Error(), "failed to decode Ollama response") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestOllamaProvider_Execute_InvalidSimulationResponseJSON(t *testing.T) {
	server := newOllamaChatServer("これはJSONではありません")
	defer server.Close()

	p := NewOllamaProvider(server.URL, "gemma2")
	req := SimulationRequest{
		SystemPrompt:    "test",
		CurrentSentence: "test",
		TotalSentences:  1,
	}

	_, err := p.Execute(req)
	if err == nil {
		t.Fatal("expected error for invalid simulation response JSON")
	}
	var errInvalidJSON *ErrInvalidJSON
	if !errors.As(err, &errInvalidJSON) {
		t.Errorf("expected ErrInvalidJSON, got: %T: %v", err, err)
	}
}
