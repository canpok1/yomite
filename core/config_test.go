package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigJSONRoundTrip(t *testing.T) {
	original := Config{
		DefaultProvider: "local_ollama",
		DefaultPersona:  "beginner",
		Providers: map[string]Provider{
			"local_ollama": {
				Type:   "ollama",
				Model:  "gemma2",
				Origin: "http://localhost:11434",
			},
		},
		Personas: map[string]Persona{
			"beginner": {
				DisplayName:    "初学者",
				SystemPrompt:   "あなたは初学者です。",
				MemoryCapacity: 200,
				MaxSteps:       100,
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored Config
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.DefaultProvider != original.DefaultProvider {
		t.Errorf("DefaultProvider: got %q, want %q", restored.DefaultProvider, original.DefaultProvider)
	}
	if restored.DefaultPersona != original.DefaultPersona {
		t.Errorf("DefaultPersona: got %q, want %q", restored.DefaultPersona, original.DefaultPersona)
	}

	p, ok := restored.Providers["local_ollama"]
	if !ok {
		t.Fatal("Provider 'local_ollama' not found")
	}
	if p.Type != "ollama" || p.Model != "gemma2" || p.Origin != "http://localhost:11434" {
		t.Errorf("Provider mismatch: %+v", p)
	}

	persona, ok := restored.Personas["beginner"]
	if !ok {
		t.Fatal("Persona 'beginner' not found")
	}
	if persona.DisplayName != "初学者" || persona.MemoryCapacity != 200 || persona.MaxSteps != 100 {
		t.Errorf("Persona mismatch: %+v", persona)
	}
}

func TestLoadConfig_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	content := `{
		"default_provider": "local",
		"default_persona": "test",
		"providers": {
			"local": {"type": "ollama", "model": "gemma2"}
		},
		"personas": {
			"test": {"display_name": "テスト", "system_prompt": "テスト用", "memory_capacity": 100, "max_steps": 50}
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.DefaultProvider != "local" {
		t.Errorf("DefaultProvider: got %q, want %q", cfg.DefaultProvider, "local")
	}

	// Origin が未指定の場合、デフォルト値が設定されること
	p := cfg.Providers["local"]
	if p.Origin != "http://localhost:11434" {
		t.Errorf("Origin default: got %q, want %q", p.Origin, "http://localhost:11434")
	}
}

func TestLoadConfig_ExplicitPathNotFound(t *testing.T) {
	nonExistentPath := filepath.Join(t.TempDir(), "not-found.json")
	_, err := LoadConfig(nonExistentPath)
	if err == nil {
		t.Fatal("expected error for nonexistent config path")
	}
}

func TestLoadConfig_LocalOnly(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "yomite.json")
	writeTestConfig(t, localPath, "local_only", "persona_a", map[string]Provider{
		"local_only": {Type: "ollama", Model: "gemma2"},
	}, map[string]Persona{
		"persona_a": {DisplayName: "A", SystemPrompt: "a", MemoryCapacity: 100, MaxSteps: 10},
	})

	cfg, err := loadConfigFromPaths(localPath, filepath.Join(dir, "nonexistent.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultProvider != "local_only" {
		t.Errorf("DefaultProvider: got %q, want %q", cfg.DefaultProvider, "local_only")
	}
}

func TestLoadConfig_GlobalOnly(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "config.json")
	writeTestConfig(t, globalPath, "global_only", "persona_b", map[string]Provider{
		"global_only": {Type: "ollama", Model: "llama3"},
	}, map[string]Persona{
		"persona_b": {DisplayName: "B", SystemPrompt: "b", MemoryCapacity: 200, MaxSteps: 20},
	})

	cfg, err := loadConfigFromPaths(filepath.Join(dir, "nonexistent.json"), globalPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultProvider != "global_only" {
		t.Errorf("DefaultProvider: got %q, want %q", cfg.DefaultProvider, "global_only")
	}
}

func TestLoadConfig_MergeBothExist(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "yomite.json")
	globalPath := filepath.Join(dir, "config.json")

	// グローバル: provider=global_p, persona=global_persona
	writeTestConfig(t, globalPath, "global_p", "global_persona", map[string]Provider{
		"global_p": {Type: "ollama", Model: "gemma2", Origin: "http://remote:11434"},
	}, map[string]Persona{
		"global_persona": {DisplayName: "Global", SystemPrompt: "global", MemoryCapacity: 100, MaxSteps: 10},
	})

	// ローカル: provider=local_p, persona=local_persona（上書き）
	writeTestConfig(t, localPath, "local_p", "local_persona", map[string]Provider{
		"local_p": {Type: "ollama", Model: "llama3"},
	}, map[string]Persona{
		"local_persona": {DisplayName: "Local", SystemPrompt: "local", MemoryCapacity: 300, MaxSteps: 30},
	})

	cfg, err := loadConfigFromPaths(localPath, globalPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ローカルで上書き
	if cfg.DefaultProvider != "local_p" {
		t.Errorf("DefaultProvider: got %q, want %q", cfg.DefaultProvider, "local_p")
	}
	if cfg.DefaultPersona != "local_persona" {
		t.Errorf("DefaultPersona: got %q, want %q", cfg.DefaultPersona, "local_persona")
	}

	// グローバルのproviderも残る
	if _, ok := cfg.Providers["global_p"]; !ok {
		t.Error("global provider should still exist after merge")
	}
	// ローカルのproviderも追加される
	if _, ok := cfg.Providers["local_p"]; !ok {
		t.Error("local provider should exist after merge")
	}

	// グローバルのpersonaも残る
	if _, ok := cfg.Personas["global_persona"]; !ok {
		t.Error("global persona should still exist after merge")
	}
	// ローカルのpersonaも追加される
	if _, ok := cfg.Personas["local_persona"]; !ok {
		t.Error("local persona should exist after merge")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	invalidPath := filepath.Join(dir, "yomite.json")
	if err := os.WriteFile(invalidPath, []byte("{invalid json}"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfigFromPaths(invalidPath, filepath.Join(dir, "nonexistent.json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON config")
	}
	// エラーメッセージがJSONパースエラーであること（"no config file found"ではない）
	if got := err.Error(); !strings.Contains(got, "failed to load local config") {
		t.Errorf("expected local config error, got: %s", got)
	}
}

func TestLoadConfig_NeitherExists(t *testing.T) {
	dir := t.TempDir()
	_, err := loadConfigFromPaths(
		filepath.Join(dir, "nonexistent1.json"),
		filepath.Join(dir, "nonexistent2.json"),
	)
	if err == nil {
		t.Fatal("expected error when neither config exists")
	}
}

func TestValidate_InvalidDefaultProvider(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	writeTestConfig(t, configPath, "nonexistent", "test", map[string]Provider{
		"local": {Type: "ollama", Model: "gemma2"},
	}, map[string]Persona{
		"test": {DisplayName: "T", SystemPrompt: "t", MemoryCapacity: 100, MaxSteps: 10},
	})

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("expected validation error for invalid default_provider")
	}
}

func TestValidate_InvalidDefaultPersona(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	writeTestConfig(t, configPath, "local", "nonexistent", map[string]Provider{
		"local": {Type: "ollama", Model: "gemma2"},
	}, map[string]Persona{
		"test": {DisplayName: "T", SystemPrompt: "t", MemoryCapacity: 100, MaxSteps: 10},
	})

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("expected validation error for invalid default_persona")
	}
}

// writeTestConfig はテスト用の設定ファイルを書き出すヘルパー。
func writeTestConfig(t *testing.T, path, defaultProvider, defaultPersona string, providers map[string]Provider, personas map[string]Persona) {
	t.Helper()
	cfg := Config{
		DefaultProvider: defaultProvider,
		DefaultPersona:  defaultPersona,
		Providers:       providers,
		Personas:        personas,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
