package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canpok1/yomite/internal/core"
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
	logPath := filepath.Join(t.TempDir(), "test.log")
	cfg := core.Config{
		Log:             core.LogConfig{Level: "warn", Path: logPath},
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
	if !strings.Contains(output, "[Step 1]") {
		t.Errorf("expected Step 1 in output, got: %s", output)
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

	// JSON Lines形式: 1行1JSONオブジェクト
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSON Lines, got %d; output: %s", len(lines), stdout.String())
	}

	var step0 core.SimulationStep
	if err := json.Unmarshal([]byte(lines[0]), &step0); err != nil {
		t.Fatalf("invalid JSON on line 0: %v; line: %s", err, lines[0])
	}
	if step0.TargetIdx == nil || *step0.TargetIdx != 1 {
		t.Errorf("expected target_idx=1 in step 0")
	}

	var step1 core.SimulationStep
	if err := json.Unmarshal([]byte(lines[1]), &step1); err != nil {
		t.Fatalf("invalid JSON on line 1: %v; line: %s", err, lines[1])
	}
	if step1.TargetIdx != nil {
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

func TestOutputStepJSON(t *testing.T) {
	step := core.SimulationStep{
		Step:        1,
		SentenceIdx: 0,
		TargetIdx:   intPtr(1),
		Note:        &core.Note{Type: core.NoteTypeQuestion, Content: "テスト疑問"},
	}

	var buf bytes.Buffer
	if err := outputStepJSON(&buf, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var result core.SimulationStep
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON output: %v; output: %s", err, output)
	}
	if result.Note == nil || result.Note.Type != core.NoteTypeQuestion {
		t.Errorf("expected QUESTION note")
	}
	if result.TargetIdx == nil || *result.TargetIdx != 1 {
		t.Errorf("expected target_idx=1")
	}
}

func TestOutputStepJSON_NilFields(t *testing.T) {
	step := core.SimulationStep{
		Step:        1,
		SentenceIdx: 1,
		TargetIdx:   nil,
		Note:        nil,
	}

	var buf bytes.Buffer
	if err := outputStepJSON(&buf, step); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var result core.SimulationStep
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result.TargetIdx != nil {
		t.Errorf("expected nil target_idx")
	}
	if result.Note != nil {
		t.Errorf("expected nil note")
	}
}

func TestOutputStepText(t *testing.T) {
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
			Step:        1,
			SentenceIdx: 0,
			TargetIdx:   intPtr(1),
			Note:        &core.Note{Type: core.NoteTypeQuestion, Content: "疑問"},
		},
		{
			Step:        2,
			SentenceIdx: 1,
			TargetIdx:   intPtr(0),
			Note:        nil,
		},
		{
			Step:        3,
			SentenceIdx: 0,
			TargetIdx:   nil,
			Note:        &core.Note{Type: core.NoteTypeResolved, Content: "解消"},
		},
	}

	var stdout bytes.Buffer
	for _, s := range steps {
		if err := outputStepText(&stdout, s, doc); err != nil {
			t.Fatalf("unexpected error at step %d: %v", s.Step, err)
		}
	}

	output := stdout.String()

	// Step 1: 先読み
	if !strings.Contains(output, "[Step 1] 文0: 文1。") {
		t.Errorf("expected step 1 header, got: %s", output)
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

func TestOutputStepText_Reread(t *testing.T) {
	doc := core.Document{
		Sentences: []core.Sentence{
			{Index: 0, Content: "テスト。"},
		},
	}

	step := core.SimulationStep{
		Step:        1,
		SentenceIdx: 0,
		TargetIdx:   intPtr(0),
	}

	var stdout bytes.Buffer
	if err := outputStepText(&stdout, step, doc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "再読") {
		t.Errorf("expected reread direction, got: %s", stdout.String())
	}
}

func TestRun_Integration_LogFileCreated(t *testing.T) {
	inputPath := writeTestInput(t, "テスト文1。テスト文2。")
	logPath := filepath.Join(t.TempDir(), "test.log")
	cfg := core.Config{
		Log:             core.LogConfig{Level: "info", Path: logPath},
		DefaultProvider: "test-provider",
		DefaultPersona:  "test-persona",
		Providers: map[string]core.ProviderConfig{
			"test-provider": {Type: "ollama", Model: "test", Origin: "http://localhost:11434"},
		},
		Personas: map[string]core.Persona{
			"test-persona": {DisplayName: "T", SystemPrompt: "テスト用", MemoryCapacity: 100, MaxSteps: 10},
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	mock := &mockProvider{
		responses: []core.SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(1), Memory: "m1"},
			{CurrentIndex: 1, NextIndex: nil, Memory: "m2"},
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

	// ログファイルが作成されていること
	logData, err := os.ReadFile(cfg.Log.Path)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	logContent := string(logData)

	if !strings.Contains(logContent, "simulation started") {
		t.Errorf("expected 'simulation started' in log file")
	}
	if !strings.Contains(logContent, "simulation finished") {
		t.Errorf("expected 'simulation finished' in log file")
	}
}

func TestRun_Integration_DebugLogIncludesLLM(t *testing.T) {
	inputPath := writeTestInput(t, "テスト文。")
	logPath := filepath.Join(t.TempDir(), "debug.log")
	cfg := core.Config{
		Log:             core.LogConfig{Level: "debug", Path: logPath},
		DefaultProvider: "test-provider",
		DefaultPersona:  "test-persona",
		Providers: map[string]core.ProviderConfig{
			"test-provider": {Type: "ollama", Model: "test", Origin: "http://localhost:11434"},
		},
		Personas: map[string]core.Persona{
			"test-persona": {DisplayName: "T", SystemPrompt: "テスト用", MemoryCapacity: 100, MaxSteps: 10},
		},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	mock := &mockProvider{
		responses: []core.SimulationResponse{
			{CurrentIndex: 0, NextIndex: nil, Memory: "m", RawResponseText: `{"current_index":0}`},
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

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	logContent := string(logData)

	if !strings.Contains(logContent, "llm request") {
		t.Errorf("expected 'llm request' in debug log")
	}
	if !strings.Contains(logContent, "llm response") {
		t.Errorf("expected 'llm response' in debug log")
	}
}
