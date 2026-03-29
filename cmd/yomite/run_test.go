package yomite

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/yomite/core"
)

// mockProvider はテスト用のProviderモック。
type mockProvider struct {
	responses []core.SimulationResponse
	callIdx   int
}

func (m *mockProvider) Execute(_ core.SimulationRequest) (core.SimulationResponse, error) {
	if m.callIdx >= len(m.responses) {
		return core.SimulationResponse{}, nil
	}
	resp := m.responses[m.callIdx]
	m.callIdx++
	return resp, nil
}

func intPtr(v int) *int {
	return &v
}

// writeTestConfig は一時ディレクトリにテスト用設定ファイルを作成する。
func writeTestConfig(t *testing.T) string {
	t.Helper()
	cfg := core.Config{
		DefaultProvider: "test-provider",
		DefaultPersona:  "test-persona",
		Providers: map[string]core.ProviderConfig{
			"test-provider": {
				Type:   "ollama",
				Model:  "test-model",
				Origin: "http://localhost:11434",
			},
		},
		Personas: map[string]core.Persona{
			"test-persona": {
				DisplayName:    "テスト読者",
				SystemPrompt:   "テスト用プロンプト",
				MemoryCapacity: 100,
				MaxSteps:       10,
			},
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeTestInput は一時ディレクトリにテスト用入力ファイルを作成する。
func writeTestInput(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRun_Integration_TextOutput(t *testing.T) {
	inputPath := writeTestInput(t, "テスト文1。テスト文2。")
	configPath := writeTestConfig(t)

	mock := &mockProvider{
		responses: []core.SimulationResponse{
			{
				CurrentIndex: 0,
				NextIndex:    intPtr(1),
				Note:         &core.Note{Type: core.NoteTypeQuestion, Content: "これは何？"},
				Memory:       "テスト文1を読んだ",
			},
			{
				CurrentIndex: 1,
				NextIndex:    nil,
				Note:         &core.Note{Type: core.NoteTypeResolved, Content: "分かった"},
				Memory:       "テスト文2を読んだ",
			},
		},
	}

	origFactory := providerFactory
	providerFactory = func(_ core.ProviderConfig) core.Provider { return mock }
	defer func() { providerFactory = origFactory }()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"-f", inputPath, "--config", configPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "[Step 0]") {
		t.Errorf("expected Step 0 in output, got: %s", output)
	}
	if !strings.Contains(output, "Note[QUESTION]: これは何？") {
		t.Errorf("expected QUESTION note, got: %s", output)
	}
	if !strings.Contains(output, "読了") {
		t.Errorf("expected finish marker, got: %s", output)
	}
}

func TestRun_Integration_JSONOutput(t *testing.T) {
	inputPath := writeTestInput(t, "テスト文1。テスト文2。")
	configPath := writeTestConfig(t)

	mock := &mockProvider{
		responses: []core.SimulationResponse{
			{
				CurrentIndex: 0,
				NextIndex:    intPtr(1),
				Note:         nil,
				Memory:       "memo",
			},
			{
				CurrentIndex: 1,
				NextIndex:    nil,
				Note:         nil,
				Memory:       "",
			},
		},
	}

	origFactory := providerFactory
	providerFactory = func(_ core.ProviderConfig) core.Provider { return mock }
	defer func() { providerFactory = origFactory }()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"-f", inputPath, "--config", configPath, "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", code, stderr.String())
	}

	var steps []core.SimulationStep
	if err := json.Unmarshal(stdout.Bytes(), &steps); err != nil {
		t.Fatalf("invalid JSON: %v; output: %s", err, stdout.String())
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].TargetIdx == nil || *steps[0].TargetIdx != 1 {
		t.Errorf("expected target_idx=1 in step 0")
	}
	if steps[1].TargetIdx != nil {
		t.Errorf("expected nil target_idx in step 1")
	}
}

func TestRun_MissingFileFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "-f フラグは必須") {
		t.Errorf("expected error about -f flag, got: %s", stderr.String())
	}
}

func TestRun_FileNotFound(t *testing.T) {
	configPath := writeTestConfig(t)
	var stderr bytes.Buffer
	code := Run([]string{"-f", "/nonexistent/file.txt", "--config", configPath}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "ファイルが見つかりません") {
		t.Errorf("expected file not found error, got: %s", stderr.String())
	}
}

func TestRun_ConfigNotFound(t *testing.T) {
	inputPath := writeTestInput(t, "テスト。")
	var stderr bytes.Buffer
	code := Run([]string{"-f", inputPath, "--config", "/nonexistent/config.json"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "設定ファイル") {
		t.Errorf("expected config error, got: %s", stderr.String())
	}
}

func TestRun_ProviderNotFound(t *testing.T) {
	inputPath := writeTestInput(t, "テスト。")
	configPath := writeTestConfig(t)
	var stderr bytes.Buffer
	code := Run([]string{"-f", inputPath, "--config", configPath, "--provider", "unknown"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "プロバイダ") && !strings.Contains(stderr.String(), "存在しません") {
		t.Errorf("expected provider not found error, got: %s", stderr.String())
	}
}

func TestRun_PersonaNotFound(t *testing.T) {
	inputPath := writeTestInput(t, "テスト。")
	configPath := writeTestConfig(t)
	var stderr bytes.Buffer
	code := Run([]string{"-f", inputPath, "--config", configPath, "--persona", "unknown"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "ペルソナ") && !strings.Contains(stderr.String(), "存在しません") {
		t.Errorf("expected persona not found error, got: %s", stderr.String())
	}
}

func TestOutputJSON(t *testing.T) {
	steps := []core.SimulationStep{
		{
			Step:        0,
			SentenceIdx: 0,
			TargetIdx:   intPtr(1),
			Note:        &core.Note{Type: core.NoteTypeQuestion, Content: "テスト疑問"},
		},
		{
			Step:        1,
			SentenceIdx: 1,
			TargetIdx:   nil,
			Note:        nil,
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := outputJSON(&stdout, &stderr, steps)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	var result []core.SimulationStep
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(result))
	}
	if result[0].Note == nil || result[0].Note.Type != core.NoteTypeQuestion {
		t.Errorf("expected QUESTION note in step 0")
	}
	if result[1].TargetIdx != nil {
		t.Errorf("expected nil target_idx in step 1")
	}
}

func TestOutputText(t *testing.T) {
	doc := core.Document{
		ID:      "test",
		RawText: "文1。文2。",
		Sentences: []core.Sentence{
			{Index: 0, Content: "文1。"},
			{Index: 1, Content: "文2。"},
		},
	}

	steps := []core.SimulationStep{
		{
			Step:        0,
			SentenceIdx: 0,
			TargetIdx:   intPtr(1),
			Note:        &core.Note{Type: core.NoteTypeQuestion, Content: "疑問"},
		},
		{
			Step:        1,
			SentenceIdx: 1,
			TargetIdx:   intPtr(0),
			Note:        nil,
		},
		{
			Step:        2,
			SentenceIdx: 0,
			TargetIdx:   nil,
			Note:        &core.Note{Type: core.NoteTypeResolved, Content: "解消"},
		},
	}

	var stdout bytes.Buffer
	code := outputText(&stdout, steps, doc)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()

	// Step 0: 先読み
	if !strings.Contains(output, "[Step 0] 文0: 文1。") {
		t.Errorf("expected step 0 header, got: %s", output)
	}
	if !strings.Contains(output, "Note[QUESTION]: 疑問") {
		t.Errorf("expected QUESTION note, got: %s", output)
	}
	if !strings.Contains(output, "先読み") {
		t.Errorf("expected forward direction, got: %s", output)
	}

	// Step 1: 読み返し
	if !strings.Contains(output, "読み返し") {
		t.Errorf("expected backward direction, got: %s", output)
	}

	// Step 2: 読了
	if !strings.Contains(output, "読了") {
		t.Errorf("expected finish marker, got: %s", output)
	}
	if !strings.Contains(output, "Note[RESOLVED]: 解消") {
		t.Errorf("expected RESOLVED note, got: %s", output)
	}
}

func TestOutputText_Reread(t *testing.T) {
	doc := core.Document{
		Sentences: []core.Sentence{
			{Index: 0, Content: "テスト。"},
		},
	}

	steps := []core.SimulationStep{
		{
			Step:        0,
			SentenceIdx: 0,
			TargetIdx:   intPtr(0),
		},
	}

	var stdout bytes.Buffer
	code := outputText(&stdout, steps, doc)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "再読") {
		t.Errorf("expected reread direction, got: %s", stdout.String())
	}
}
