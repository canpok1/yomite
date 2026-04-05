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

// collectSteps はコールバックで受け取ったステップをスライスに集めるヘルパー。
func collectSteps() (onStep func(SimulationStep) error, getSteps func() []SimulationStep) {
	var steps []SimulationStep
	onStep = func(s SimulationStep) error {
		steps = append(steps, s)
		return nil
	}
	getSteps = func() []SimulationStep {
		return steps
	}
	return
}

func TestRunSimulation_NormalCompletion(t *testing.T) {
	// 3文のドキュメントを順に読み、最後の文でnilを返して終了
	// 各ステップで note + memory の2回呼び出し = 6レスポンス
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
			// Step 1: note
			{CurrentIndex: 0, NextIndex: intPtr(1), Note: nil},
			// Step 1: memory
			{Memory: "文1を読んだ"},
			// Step 2: note
			{CurrentIndex: 1, NextIndex: intPtr(2), Note: &Note{Type: NoteTypeQuestion, Content: "なぜ？"}},
			// Step 2: memory
			{Memory: "文1と文2を読んだ"},
			// Step 3: note
			{CurrentIndex: 2, NextIndex: nil, Note: nil},
			// Step 3: memory
			{Memory: "全部読んだ"},
		},
	}

	onStep, getSteps := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	steps := getSteps()
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}

	// Step 1
	if steps[0].Step != 1 || steps[0].SentenceIdx != 0 {
		t.Errorf("step 1: got Step=%d, SentenceIdx=%d", steps[0].Step, steps[0].SentenceIdx)
	}
	if steps[0].TargetIdx == nil || *steps[0].TargetIdx != 1 {
		t.Errorf("step 1: expected TargetIdx=1, got %v", steps[0].TargetIdx)
	}
	if steps[0].Note != nil {
		t.Errorf("step 1: expected no note")
	}

	// Step 2
	if steps[1].Step != 2 || steps[1].SentenceIdx != 1 {
		t.Errorf("step 2: got Step=%d, SentenceIdx=%d", steps[1].Step, steps[1].SentenceIdx)
	}
	if steps[1].Note == nil || steps[1].Note.Type != NoteTypeQuestion {
		t.Errorf("step 2: expected QUESTION note")
	}

	// Step 3
	if steps[2].Step != 3 || steps[2].SentenceIdx != 2 {
		t.Errorf("step 3: got Step=%d, SentenceIdx=%d", steps[2].Step, steps[2].SentenceIdx)
	}
	if steps[2].TargetIdx != nil {
		t.Errorf("step 3: expected nil TargetIdx for completion")
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
		MaxSteps:       3,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			// Step 1: note, memory
			{CurrentIndex: 0, NextIndex: intPtr(1)},
			{Memory: "m1"},
			// Step 2: note, memory
			{CurrentIndex: 1, NextIndex: intPtr(0)},
			{Memory: "m2"},
			// Step 3: note, memory
			{CurrentIndex: 0, NextIndex: intPtr(1)},
			{Memory: "m3"},
			// Step 4 (won't reach): note, memory
			{CurrentIndex: 1, NextIndex: intPtr(0)},
			{Memory: "m4"},
		},
	}

	onStep, getSteps := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	steps := getSteps()
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps (max_steps), got %d", len(steps))
	}
}

func TestRunSimulation_DefaultMaxSteps(t *testing.T) {
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
		MaxSteps:       0,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			// Step 1: note, memory
			{CurrentIndex: 0, NextIndex: intPtr(0)},
			{Memory: "m1"},
			// Step 2: note, memory
			{CurrentIndex: 0, NextIndex: intPtr(0)},
			{Memory: "m2"},
			// Step 3: note, memory
			{CurrentIndex: 0, NextIndex: intPtr(0)},
			{Memory: "m3"},
			// Step 4 (won't reach): note, memory
			{CurrentIndex: 0, NextIndex: intPtr(0)},
			{Memory: "m4"},
		},
	}

	onStep, getSteps := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	steps := getSteps()
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps (default max_steps=sentences*3), got %d", len(steps))
	}
}

func TestRunSimulation_ProviderErrorOnNote(t *testing.T) {
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
			// Step 1: note, memory
			{CurrentIndex: 0, NextIndex: intPtr(1)},
			{Memory: "m1"},
		},
		errors: []error{
			nil, nil,
			errors.New("LLM connection failed"), // Step 2 note fails
		},
	}

	onStep, _ := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err == nil {
		t.Fatal("expected error from provider failure")
	}
	if !strings.Contains(err.Error(), "note") {
		t.Errorf("error should mention note phase: %v", err)
	}
}

func TestRunSimulation_ProviderErrorOnMemory(t *testing.T) {
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
			{CurrentIndex: 0, NextIndex: nil}, // note ok
		},
		errors: []error{
			nil,
			errors.New("LLM connection failed"), // memory fails
		},
	}

	onStep, _ := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err == nil {
		t.Fatal("expected error from provider failure")
	}
	if !strings.Contains(err.Error(), "memory") {
		t.Errorf("error should mention memory phase: %v", err)
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
			{CurrentIndex: 0, NextIndex: intPtr(99)}, // note with bad index
			{Memory: "m1"},                           // memory (won't be used)
		},
	}

	onStep, _ := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
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
		MemoryCapacity: 5,
		MaxSteps:       10,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: nil}, // note
			{Memory: "これは長い記憶バッファです"},         // memory
		},
	}

	onStep, getSteps := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	steps := getSteps()
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}

	// 最初のnoteリクエストのmemoryは空
	if mock.calls[0].Memory != "" {
		t.Errorf("first note request should have empty memory, got %q", mock.calls[0].Memory)
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
		MemoryCapacity: 3,
		MaxSteps:       10,
	}
	mock := &mockProvider{
		responses: []SimulationResponse{
			// Step 1
			{CurrentIndex: 0, NextIndex: intPtr(1)}, // note
			{Memory: "あいうえお"},                       // memory (will be truncated to 3 chars)
			// Step 2
			{CurrentIndex: 1, NextIndex: nil}, // note
			{Memory: "ok"},                    // memory
		},
	}

	onStep, _ := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.calls) < 3 {
		t.Fatalf("expected at least 3 calls, got %d", len(mock.calls))
	}
	// Step 1のnoteリクエスト（calls[2]）のmemoryが切り詰められていること
	if mock.calls[2].Memory != "あいう" {
		t.Errorf("step 1 note request memory should be truncated to 3 chars, got %q", mock.calls[2].Memory)
	}
}

func TestRunSimulation_Backtracking(t *testing.T) {
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
			// Step 1: note, memory
			{CurrentIndex: 0, NextIndex: intPtr(1)},
			{Memory: "m1"},
			// Step 2: note (backtrack), memory
			{CurrentIndex: 1, NextIndex: intPtr(0)},
			{Memory: "m2"},
			// Step 3: note, memory
			{CurrentIndex: 0, NextIndex: intPtr(2)},
			{Memory: "m3"},
			// Step 4: note (done), memory
			{CurrentIndex: 2, NextIndex: nil},
			{Memory: "m4"},
		},
	}

	onStep, getSteps := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	steps := getSteps()
	if len(steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(steps))
	}

	if steps[1].TargetIdx == nil || *steps[1].TargetIdx != 0 {
		t.Errorf("step 1 should backtrack to 0")
	}
	if steps[2].SentenceIdx != 0 {
		t.Errorf("step 2 should be at sentence 0 after backtrack")
	}
}

func TestRunSimulation_RequestFields(t *testing.T) {
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
			// Step 1: note, memory
			{CurrentIndex: 0, NextIndex: intPtr(1)},
			{Memory: "記憶1"},
			// Step 2: note, memory
			{CurrentIndex: 1, NextIndex: nil},
			{Memory: "記憶2"},
		},
	}

	onStep, _ := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 steps × 2 phases = 4 calls
	if len(mock.calls) != 4 {
		t.Fatalf("expected 4 calls, got %d", len(mock.calls))
	}

	// Step 1 note request (calls[0])
	noteReq0 := mock.calls[0]
	if noteReq0.Phase != PhaseNote {
		t.Errorf("calls[0].Phase: got %d, want PhaseNote", noteReq0.Phase)
	}
	if noteReq0.SystemPrompt != "あなたはテスト用AIです" {
		t.Errorf("calls[0].SystemPrompt: got %q", noteReq0.SystemPrompt)
	}
	if noteReq0.CurrentSentence != "文1。" {
		t.Errorf("calls[0].CurrentSentence: got %q", noteReq0.CurrentSentence)
	}
	if noteReq0.CurrentIndex != 0 {
		t.Errorf("calls[0].CurrentIndex: got %d", noteReq0.CurrentIndex)
	}
	if noteReq0.TotalSentences != 2 {
		t.Errorf("calls[0].TotalSentences: got %d", noteReq0.TotalSentences)
	}
	if noteReq0.Memory != "" {
		t.Errorf("calls[0].Memory: got %q, want empty", noteReq0.Memory)
	}

	// Step 1 memory request (calls[1])
	memReq0 := mock.calls[1]
	if memReq0.Phase != PhaseMemory {
		t.Errorf("calls[1].Phase: got %d, want PhaseMemory", memReq0.Phase)
	}

	// Step 2 note request (calls[2])
	noteReq1 := mock.calls[2]
	if noteReq1.Phase != PhaseNote {
		t.Errorf("calls[2].Phase: got %d, want PhaseNote", noteReq1.Phase)
	}
	if noteReq1.CurrentSentence != "文2。" {
		t.Errorf("calls[2].CurrentSentence: got %q", noteReq1.CurrentSentence)
	}
	if noteReq1.CurrentIndex != 1 {
		t.Errorf("calls[2].CurrentIndex: got %d", noteReq1.CurrentIndex)
	}
	if noteReq1.Memory != "記憶1" {
		t.Errorf("calls[2].Memory: got %q, want %q", noteReq1.Memory, "記憶1")
	}
}

func TestRunSimulation_MemoryPhaseReceivesNote(t *testing.T) {
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
	expectedNote := &Note{Type: NoteTypeQuestion, Content: "なぜ？"}
	mock := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: nil, Note: expectedNote}, // note
			{Memory: "m1"}, // memory
		},
	}

	onStep, _ := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// memory phase request should include the note from note phase
	memReq := mock.calls[1]
	if memReq.Note == nil {
		t.Fatal("memory phase request should include note")
	}
	if memReq.Note.Type != NoteTypeQuestion || memReq.Note.Content != "なぜ？" {
		t.Errorf("memory phase note: got %+v, want %+v", memReq.Note, expectedNote)
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

	onStep, getSteps := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(getSteps()) != 0 {
		t.Errorf("expected 0 steps for empty document, got %d", len(getSteps()))
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
			{CurrentIndex: 0, NextIndex: intPtr(-1)}, // note with bad index
			{Memory: "m1"},                           // memory (won't be reached)
		},
	}

	onStep, _ := collectSteps()
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
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
			// Step 1: note, memory
			{CurrentIndex: 0, NextIndex: intPtr(1)},
			{Memory: "m1"},
			// Step 2: note, memory
			{CurrentIndex: 1, NextIndex: nil},
			{Memory: "m2"},
		},
	}

	onStep, _ := collectSteps()
	err := RunSimulation(doc, persona, mock, logger, onStep)
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
			{CurrentIndex: 0, NextIndex: nil}, // note
			{Memory: "あいうえお"},                 // memory (will be truncated)
		},
	}

	onStep, _ := collectSteps()
	err := RunSimulation(doc, persona, mock, logger, onStep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "memory truncated") {
		t.Errorf("expected 'memory truncated' warn log, got: %s", logs)
	}
}

func TestRunSimulation_CallbackCalledPerStep(t *testing.T) {
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
			// Step 1: note, memory
			{CurrentIndex: 0, NextIndex: intPtr(1)},
			{Memory: "m1"},
			// Step 2: note, memory
			{CurrentIndex: 1, NextIndex: nil},
			{Memory: "m2"},
		},
	}

	var callOrder []int
	onStep := func(s SimulationStep) error {
		callOrder = append(callOrder, s.Step)
		return nil
	}
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(callOrder) != 2 {
		t.Fatalf("expected 2 callback calls, got %d", len(callOrder))
	}
	if callOrder[0] != 1 || callOrder[1] != 2 {
		t.Errorf("expected callback order [1, 2], got %v", callOrder)
	}
}

func TestRunSimulation_CallbackError(t *testing.T) {
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
			// Step 1: note, memory
			{CurrentIndex: 0, NextIndex: intPtr(1)},
			{Memory: "m1"},
		},
	}

	callbackErr := errors.New("output failed")
	onStep := func(s SimulationStep) error {
		if s.Step == 1 {
			return callbackErr
		}
		return nil
	}
	err := RunSimulation(doc, persona, mock, discardLogger, onStep)
	if err == nil {
		t.Fatal("expected error from callback failure")
	}
	if !errors.Is(err, callbackErr) {
		t.Errorf("expected callback error to be wrapped, got: %v", err)
	}
	// note + memory の2回呼ばれた後にコールバックエラーで中断
	if mock.callIdx != 2 {
		t.Errorf("expected provider to be called twice (note + memory), got %d", mock.callIdx)
	}
}
