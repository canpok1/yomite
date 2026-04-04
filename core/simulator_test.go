package core

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
)

// discardLogger はテスト用の出力を破棄するロガー。
var discardLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

// mockProvider はテスト用のProviderモック。
type mockProvider struct {
	responses []SimulationResponse
	errors    []error
	calls     []SimulationRequest
	callIdx   int
}

func (m *mockProvider) Execute(req SimulationRequest) (SimulationResponse, error) {
	m.calls = append(m.calls, req)
	idx := m.callIdx
	m.callIdx++
	if idx < len(m.errors) && m.errors[idx] != nil {
		return SimulationResponse{}, m.errors[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return SimulationResponse{}, errors.New("no more mock responses")
}

func intPtr(v int) *int {
	return &v
}

func TestRunSimulation_NormalCompletion(t *testing.T) {
	// 3文のドキュメントを順に読み、2文目でnilを返して終了
	doc := Document{
		ID:      "test",
		RawText: "文1。文2。文3。",
		Sentences: []Sentence{
			{Index: 0, Content: "文1。"},
			{Index: 1, Content: "文2。"},
			{Index: 2, Content: "文3。"},
		},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "テスト用",
		MemoryCapacity: 100,
		MaxSteps:       10,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(1), Note: nil, Memory: "文1を読んだ"},
			{CurrentIndex: 1, NextIndex: intPtr(2), Note: &Note{Type: NoteTypeQuestion, Content: "なぜ？"}, Memory: "文1と文2を読んだ"},
			{CurrentIndex: 2, NextIndex: nil, Note: nil, Memory: "全部読んだ"},
		},
	}

	steps, err := RunSimulation(doc, persona, mock, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}

	// Step 0
	if steps[0].Step != 0 || steps[0].SentenceIdx != 0 {
		t.Errorf("step 0: got Step=%d, SentenceIdx=%d", steps[0].Step, steps[0].SentenceIdx)
	}
	if steps[0].TargetIdx == nil || *steps[0].TargetIdx != 1 {
		t.Errorf("step 0: expected TargetIdx=1, got %v", steps[0].TargetIdx)
	}
	if steps[0].Note != nil {
		t.Errorf("step 0: expected no note")
	}

	// Step 1
	if steps[1].Step != 1 || steps[1].SentenceIdx != 1 {
		t.Errorf("step 1: got Step=%d, SentenceIdx=%d", steps[1].Step, steps[1].SentenceIdx)
	}
	if steps[1].Note == nil || steps[1].Note.Type != NoteTypeQuestion {
		t.Errorf("step 1: expected QUESTION note")
	}

	// Step 2
	if steps[2].Step != 2 || steps[2].SentenceIdx != 2 {
		t.Errorf("step 2: got Step=%d, SentenceIdx=%d", steps[2].Step, steps[2].SentenceIdx)
	}
	if steps[2].TargetIdx != nil {
		t.Errorf("step 2: expected nil TargetIdx for completion")
	}
}

func TestRunSimulation_MaxStepsTermination(t *testing.T) {
	doc := Document{
		ID:      "test",
		RawText: "文1。文2。",
		Sentences: []Sentence{
			{Index: 0, Content: "文1。"},
			{Index: 1, Content: "文2。"},
		},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "テスト用",
		MemoryCapacity: 100,
		MaxSteps:       3, // 明示的にmax_stepsを指定
	}
	// 永遠にループするレスポンス
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(1), Memory: "m1"},
			{CurrentIndex: 1, NextIndex: intPtr(0), Memory: "m2"},
			{CurrentIndex: 0, NextIndex: intPtr(1), Memory: "m3"},
			{CurrentIndex: 1, NextIndex: intPtr(0), Memory: "m4"}, // ここには到達しない
		},
	}

	steps, err := RunSimulation(doc, persona, mock, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(steps) != 3 {
		t.Fatalf("expected 3 steps (max_steps), got %d", len(steps))
	}
}

func TestRunSimulation_DefaultMaxSteps(t *testing.T) {
	// MaxSteps=0 の場合、デフォルト (文数×3) が使用される
	doc := Document{
		ID:      "test",
		RawText: "文1。",
		Sentences: []Sentence{
			{Index: 0, Content: "文1。"},
		},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "テスト用",
		MemoryCapacity: 100,
		MaxSteps:       0, // デフォルト: 1×3=3
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(0), Memory: "m1"},
			{CurrentIndex: 0, NextIndex: intPtr(0), Memory: "m2"},
			{CurrentIndex: 0, NextIndex: intPtr(0), Memory: "m3"},
			{CurrentIndex: 0, NextIndex: intPtr(0), Memory: "m4"}, // 到達しない
		},
	}

	steps, err := RunSimulation(doc, persona, mock, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(steps) != 3 {
		t.Fatalf("expected 3 steps (default max_steps=sentences*3), got %d", len(steps))
	}
}

func TestRunSimulation_ProviderError(t *testing.T) {
	doc := Document{
		ID:      "test",
		RawText: "文1。文2。",
		Sentences: []Sentence{
			{Index: 0, Content: "文1。"},
			{Index: 1, Content: "文2。"},
		},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "テスト用",
		MemoryCapacity: 100,
		MaxSteps:       10,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(1), Memory: "m1"},
		},
		errors: []error{
			nil,
			errors.New("LLM connection failed"),
		},
	}

	_, err := RunSimulation(doc, persona, mock, discardLogger)
	if err == nil {
		t.Fatal("expected error from provider failure")
	}
}

func TestRunSimulation_OutOfRangeNextIndex(t *testing.T) {
	doc := Document{
		ID:      "test",
		RawText: "文1。文2。",
		Sentences: []Sentence{
			{Index: 0, Content: "文1。"},
			{Index: 1, Content: "文2。"},
		},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "テスト用",
		MemoryCapacity: 100,
		MaxSteps:       10,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(99), Memory: "m1"}, // 範囲外
		},
	}

	_, err := RunSimulation(doc, persona, mock, discardLogger)
	if err == nil {
		t.Fatal("expected error for out-of-range next_index")
	}
	var idxErr *ErrIndexOutOfRange
	if !errors.As(err, &idxErr) {
		t.Errorf("expected ErrIndexOutOfRange, got %T: %v", err, err)
	}
}

func TestRunSimulation_MemoryCapacityLimit(t *testing.T) {
	doc := Document{
		ID:      "test",
		RawText: "文1。",
		Sentences: []Sentence{
			{Index: 0, Content: "文1。"},
		},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "テスト用",
		MemoryCapacity: 5, // 5文字まで
		MaxSteps:       10,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: nil, Memory: "これは長い記憶バッファです"}, // 12文字 → 5文字に切り詰め
		},
	}

	steps, err := RunSimulation(doc, persona, mock, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}

	// 次のリクエストに渡されるmemoryが制限されていることを確認
	// この場合はステップ1つで終了するので、mockへのリクエストのmemoryを検証
	// 最初のリクエストのmemoryは空
	if mock.calls[0].Memory != "" {
		t.Errorf("first request should have empty memory, got %q", mock.calls[0].Memory)
	}
}

func TestRunSimulation_MemoryCapacityAppliedToNextStep(t *testing.T) {
	doc := Document{
		ID:      "test",
		RawText: "文1。文2。",
		Sentences: []Sentence{
			{Index: 0, Content: "文1。"},
			{Index: 1, Content: "文2。"},
		},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "テスト用",
		MemoryCapacity: 3, // 3文字まで
		MaxSteps:       10,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(1), Memory: "あいうえお"}, // 5文字 → 3文字に切り詰め
			{CurrentIndex: 1, NextIndex: nil, Memory: "ok"},
		},
	}

	_, err := RunSimulation(doc, persona, mock, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2回目のリクエストで、memoryが3文字に切り詰められていることを確認
	if len(mock.calls) < 2 {
		t.Fatalf("expected at least 2 calls, got %d", len(mock.calls))
	}
	if mock.calls[1].Memory != "あいう" {
		t.Errorf("second request memory should be truncated to 3 chars, got %q", mock.calls[1].Memory)
	}
}

func TestRunSimulation_Backtracking(t *testing.T) {
	// AIが後戻りするケース
	doc := Document{
		ID:      "test",
		RawText: "文1。文2。文3。",
		Sentences: []Sentence{
			{Index: 0, Content: "文1。"},
			{Index: 1, Content: "文2。"},
			{Index: 2, Content: "文3。"},
		},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "テスト用",
		MemoryCapacity: 100,
		MaxSteps:       10,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(1), Memory: "m1"},
			{CurrentIndex: 1, NextIndex: intPtr(0), Memory: "m2"}, // 後戻り
			{CurrentIndex: 0, NextIndex: intPtr(2), Memory: "m3"}, // スキップ
			{CurrentIndex: 2, NextIndex: nil, Memory: "m4"},
		},
	}

	steps, err := RunSimulation(doc, persona, mock, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(steps))
	}

	// 後戻り確認
	if steps[1].TargetIdx == nil || *steps[1].TargetIdx != 0 {
		t.Errorf("step 1 should backtrack to 0")
	}
	if steps[2].SentenceIdx != 0 {
		t.Errorf("step 2 should be at sentence 0 after backtrack")
	}
}

func TestRunSimulation_RequestFields(t *testing.T) {
	// Provider.Execute に正しいリクエストが渡されることを確認
	doc := Document{
		ID:      "test",
		RawText: "文1。文2。",
		Sentences: []Sentence{
			{Index: 0, Content: "文1。"},
			{Index: 1, Content: "文2。"},
		},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "あなたはテスト用AIです",
		MemoryCapacity: 100,
		MaxSteps:       10,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(1), Memory: "記憶1"},
			{CurrentIndex: 1, NextIndex: nil, Memory: "記憶2"},
		},
	}

	_, err := RunSimulation(doc, persona, mock, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(mock.calls))
	}

	// 1回目のリクエスト
	req0 := mock.calls[0]
	if req0.SystemPrompt != "あなたはテスト用AIです" {
		t.Errorf("req0.SystemPrompt: got %q", req0.SystemPrompt)
	}
	if req0.CurrentSentence != "文1。" {
		t.Errorf("req0.CurrentSentence: got %q", req0.CurrentSentence)
	}
	if req0.CurrentIndex != 0 {
		t.Errorf("req0.CurrentIndex: got %d", req0.CurrentIndex)
	}
	if req0.TotalSentences != 2 {
		t.Errorf("req0.TotalSentences: got %d", req0.TotalSentences)
	}
	if req0.Memory != "" {
		t.Errorf("req0.Memory: got %q, want empty", req0.Memory)
	}

	// 2回目のリクエスト
	req1 := mock.calls[1]
	if req1.CurrentSentence != "文2。" {
		t.Errorf("req1.CurrentSentence: got %q", req1.CurrentSentence)
	}
	if req1.CurrentIndex != 1 {
		t.Errorf("req1.CurrentIndex: got %d", req1.CurrentIndex)
	}
	if req1.Memory != "記憶1" {
		t.Errorf("req1.Memory: got %q, want %q", req1.Memory, "記憶1")
	}
}

func TestRunSimulation_EmptyDocument(t *testing.T) {
	doc := Document{
		ID:        "test",
		RawText:   "",
		Sentences: []Sentence{},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "テスト用",
		MemoryCapacity: 100,
		MaxSteps:       10,
	}
	mock := &mockProvider{}

	steps, err := RunSimulation(doc, persona, mock, discardLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps for empty document, got %d", len(steps))
	}
}

func TestRunSimulation_NegativeNextIndex(t *testing.T) {
	doc := Document{
		ID:      "test",
		RawText: "文1。",
		Sentences: []Sentence{
			{Index: 0, Content: "文1。"},
		},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "テスト用",
		MemoryCapacity: 100,
		MaxSteps:       10,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(-1), Memory: "m1"}, // 負のインデックス
		},
	}

	_, err := RunSimulation(doc, persona, mock, discardLogger)
	if err == nil {
		t.Fatal("expected error for negative next_index")
	}
	var idxErr *ErrIndexOutOfRange
	if !errors.As(err, &idxErr) {
		t.Errorf("expected ErrIndexOutOfRange, got %T: %v", err, err)
	}
}

func TestRunSimulation_LogOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	doc := Document{
		ID:      "test.txt",
		RawText: "文1。文2。",
		Sentences: []Sentence{
			{Index: 0, Content: "文1。"},
			{Index: 1, Content: "文2。"},
		},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "テスト用",
		MemoryCapacity: 100,
		MaxSteps:       10,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(1), Memory: "m1"},
			{CurrentIndex: 1, NextIndex: nil, Memory: "m2"},
		},
	}

	_, err := RunSimulation(doc, persona, mock, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "simulation started") {
		t.Errorf("expected 'simulation started' log")
	}
	if !strings.Contains(logs, "step completed") {
		t.Errorf("expected 'step completed' log")
	}
	if !strings.Contains(logs, "simulation finished") {
		t.Errorf("expected 'simulation finished' log")
	}
}

func TestRunSimulation_MemoryTruncationLog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	doc := Document{
		ID:      "test.txt",
		RawText: "文1。",
		Sentences: []Sentence{
			{Index: 0, Content: "文1。"},
		},
	}
	persona := Persona{
		DisplayName:    "テスト",
		SystemPrompt:   "テスト用",
		MemoryCapacity: 3,
		MaxSteps:       10,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: nil, Memory: "あいうえお"},
		},
	}

	_, err := RunSimulation(doc, persona, mock, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "memory truncated") {
		t.Errorf("expected 'memory truncated' warn log, got: %s", logs)
	}
}
