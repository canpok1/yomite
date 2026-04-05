package gui

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/canpok1/yomite/internal/core"
)

// mockProvider はテスト用のProviderモック。
type mockProvider struct {
	responses []core.SimulationResponse
	errors    []error
	callIdx   int
	delay     time.Duration
}

func (m *mockProvider) Execute(req core.SimulationRequest) (core.SimulationResponse, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	idx := m.callIdx
	m.callIdx++
	if idx < len(m.errors) && m.errors[idx] != nil {
		return core.SimulationResponse{}, m.errors[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return core.SimulationResponse{}, errors.New("no more mock responses")
}

func intPtr(v int) *int {
	return &v
}

// eventRecord は発火されたイベントを記録する。
type eventRecord struct {
	Name string
	Data []any
}

// newTestApp はテスト用のAppを生成する。providerを注入し、イベント発火を記録する。
func newTestApp(provider core.Provider) (*App, *[]eventRecord) {
	var mu sync.Mutex
	var events []eventRecord

	a := &App{}
	a.ctx = context.Background()
	a.providerFactory = func(cfg core.ProviderConfig) core.Provider {
		return provider
	}
	a.emitEvent = func(eventName string, data ...any) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, eventRecord{Name: eventName, Data: data})
	}
	return a, &events
}

// setupTestEnv はテスト用の設定ファイルを配置し、環境を整える。
func setupTestEnv(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	config := `{
		"log": {"level": "info", "path": "` + dir + `/test.log"},
		"default_provider": "test-provider",
		"default_persona": "test-persona",
		"providers": {
			"test-provider": {
				"type": "ollama",
				"model": "test-model",
				"origin": "http://localhost:11434"
			}
		},
		"personas": {
			"test-persona": {
				"display_name": "テスト読者",
				"system_prompt": "test prompt",
				"memory_capacity": 100,
				"max_steps": 10
			}
		}
	}`
	if err := os.WriteFile(dir+"/yomite.json", []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	t.Setenv("HOME", dir)
	t.Chdir(dir)
}

// waitForSimulationDone はシミュレーションgoroutineの終了を待つ。
func waitForSimulationDone(t *testing.T, app *App) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for simulation to finish")
			return
		default:
			app.mu.Lock()
			done := app.cancel == nil
			app.mu.Unlock()
			if done {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestLoadDocument(t *testing.T) {
	app := NewApp("")

	sentences := app.LoadDocument("文1。文2。文3。")

	if len(sentences) != 3 {
		t.Fatalf("expected 3 sentences, got %d", len(sentences))
	}
	if sentences[0].Content != "文1。" {
		t.Errorf("expected first sentence '文1。', got %q", sentences[0].Content)
	}
	if sentences[2].Content != "文3。" {
		t.Errorf("expected third sentence '文3。', got %q", sentences[2].Content)
	}
}

func TestLoadDocument_Empty(t *testing.T) {
	app := NewApp("")

	sentences := app.LoadDocument("")

	if sentences != nil {
		t.Errorf("expected nil for empty input, got %v", sentences)
	}
}

func TestStartSimulation_EmitsStepsAndDone(t *testing.T) {
	mock := &mockProvider{
		responses: []core.SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(1)},
			{Memory: "mem0"},
			{CurrentIndex: 1, NextIndex: nil},
			{Memory: "mem1"},
		},
	}

	setupTestEnv(t)
	app, events := newTestApp(mock)

	err := app.StartSimulation("文1。文2。", "test-provider", "test-persona")
	if err != nil {
		t.Fatalf("StartSimulation returned error: %v", err)
	}

	waitForSimulationDone(t, app)

	var stepCount, doneCount int
	for _, e := range *events {
		switch e.Name {
		case eventSimulationStep:
			stepCount++
		case eventSimulationDone:
			doneCount++
		case eventSimulationError:
			t.Fatalf("unexpected simulation:error event: %v", e.Data)
		}
	}

	if stepCount != 2 {
		t.Errorf("expected 2 step events, got %d", stepCount)
	}
	if doneCount != 1 {
		t.Errorf("expected 1 done event, got %d", doneCount)
	}
}

func TestStartSimulation_ProviderError_EmitsError(t *testing.T) {
	mock := &mockProvider{
		errors: []error{errors.New("provider failure")},
	}

	setupTestEnv(t)
	app, events := newTestApp(mock)

	err := app.StartSimulation("テスト文。", "test-provider", "test-persona")
	if err != nil {
		t.Fatalf("StartSimulation returned error: %v", err)
	}

	waitForSimulationDone(t, app)

	var errorCount int
	for _, e := range *events {
		if e.Name == eventSimulationError {
			errorCount++
		}
	}

	if errorCount != 1 {
		t.Errorf("expected 1 error event, got %d", errorCount)
	}
}

func TestStartSimulation_DuplicateCall(t *testing.T) {
	var responses []core.SimulationResponse
	for i := 0; i < 100; i++ {
		responses = append(responses,
			core.SimulationResponse{CurrentIndex: 0, NextIndex: intPtr(0)},
			core.SimulationResponse{Memory: "mem"},
		)
	}
	mock := &mockProvider{responses: responses, delay: 50 * time.Millisecond}

	setupTestEnv(t)
	app, _ := newTestApp(mock)

	err := app.StartSimulation("テスト文。", "test-provider", "test-persona")
	if err != nil {
		t.Fatalf("first StartSimulation returned error: %v", err)
	}

	time.Sleep(30 * time.Millisecond)

	err = app.StartSimulation("テスト文。", "test-provider", "test-persona")
	if err == nil {
		t.Fatal("expected error for duplicate StartSimulation, got nil")
	}

	app.StopSimulation()
	waitForSimulationDone(t, app)
}

func TestStopSimulation_CancelsRunning(t *testing.T) {
	var responses []core.SimulationResponse
	for i := 0; i < 1000; i++ {
		responses = append(responses,
			core.SimulationResponse{CurrentIndex: 0, NextIndex: intPtr(0)},
			core.SimulationResponse{Memory: "mem"},
		)
	}
	mock := &mockProvider{responses: responses, delay: 10 * time.Millisecond}

	setupTestEnv(t)
	app, _ := newTestApp(mock)

	err := app.StartSimulation("テスト文。", "test-provider", "test-persona")
	if err != nil {
		t.Fatalf("StartSimulation returned error: %v", err)
	}

	time.Sleep(30 * time.Millisecond)
	app.StopSimulation()
	waitForSimulationDone(t, app)

	app.mu.Lock()
	if app.cancel != nil {
		t.Error("expected cancel to be nil after simulation stopped")
	}
	app.mu.Unlock()
}

func TestStopSimulation_NoOp(t *testing.T) {
	app := NewApp("")
	app.StopSimulation()
}

func TestStartSimulation_InvalidProvider(t *testing.T) {
	setupTestEnv(t)
	app, _ := newTestApp(&mockProvider{})

	err := app.StartSimulation("テスト。", "nonexistent", "test-persona")
	if err == nil {
		t.Fatal("expected error for invalid provider, got nil")
	}
}

func TestStartSimulation_InvalidPersona(t *testing.T) {
	setupTestEnv(t)
	app, _ := newTestApp(&mockProvider{})

	err := app.StartSimulation("テスト。", "test-provider", "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid persona, got nil")
	}
}
