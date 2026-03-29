package core

import (
	"testing"
)

// テストリスト（シンプル→複雑の順）:
// TODO: NewOllamaProvider がorigin, modelを保持する
// TODO: OllamaProvider が Provider インターフェースを実装する
// TODO: 正常なレスポンスでSimulationResponseを返す
// TODO: リクエストボディが正しい形式で送信される（model, system, user message）
// TODO: HTTP非200レスポンスでエラーを返す
// TODO: 接続拒否でエラーを返す
// TODO: レスポンスのJSONパースエラーでエラーを返す
// TODO: LLM出力のSimulationResponseパースエラーでエラーを返す

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
