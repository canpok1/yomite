package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// テストリスト（シンプル→複雑の順）:
//
// 1. Config構造体のJSONラウンドトリップ（Provider/Persona含む）
// 2. Provider.Origin のデフォルト値設定
// 3. 明示パス指定でファイルを読み込める
// 4. 明示パスのファイルが存在しない場合エラー
// 5. ローカル yomite.json のみ存在 → 読み込み成功
// 6. グローバル config.json のみ存在 → 読み込み成功
// 7. 両方存在 → グローバルベースにローカルで上書きマージ
// 8. どちらも存在しない → エラー
// 9. default_provider が providers に存在しない → バリデーションエラー
// 10. default_persona が personas に存在しない → バリデーションエラー
// 11. バリデーション成功ケース

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
	_, err := LoadConfig("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for nonexistent config path")
	}
}
