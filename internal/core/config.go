package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const defaultOrigin = "http://localhost:11434"

// Config はアプリケーション全体の設定を保持する。
type Config struct {
	Log             LogConfig                 `json:"log"`
	DefaultProvider string                    `json:"default_provider"`
	DefaultPersona  string                    `json:"default_persona"`
	Providers       map[string]ProviderConfig `json:"providers"`
	Personas        map[string]Persona        `json:"personas"`
}

// LogConfig はログ出力の設定を保持する。
type LogConfig struct {
	Level string `json:"level"` // "debug", "info", "warn"
	Path  string `json:"path"`  // ログ出力先ファイルパス
}

// ProviderConfig はLLMプロバイダの接続情報を表す。
type ProviderConfig struct {
	Type   string `json:"type"`
	Model  string `json:"model"`
	Origin string `json:"origin"`
}

// Persona はAI読者の人格設定を表す。
type Persona struct {
	DisplayName    string `json:"display_name"`
	SystemPrompt   string `json:"system_prompt"`
	MemoryCapacity int    `json:"memory_capacity"`
	MaxSteps       int    `json:"max_steps"`
}

// LoadConfig は設定ファイルを読み込み、バリデーション済みの Config を返す。
// explicitPath が空でない場合、そのパスのみを読み込む。
// 空の場合、ローカル(./yomite.json)→グローバル(~/.config/yomite/config.json)の順で探索・マージする。
func LoadConfig(explicitPath string) (*Config, error) {
	if explicitPath != "" {
		cfg, err := loadConfigFile(explicitPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", explicitPath, err)
		}
		return finalizeConfig(cfg)
	}

	localPath := "yomite.json"
	globalPath, err := globalConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine global config path: %w", err)
	}

	return loadConfigFromPaths(localPath, globalPath)
}

// loadConfigFromPaths はローカルとグローバルのパスから設定を読み込み、マージする。
func loadConfigFromPaths(localPath, globalPath string) (*Config, error) {
	localCfg, localErr := loadConfigFile(localPath)
	globalCfg, globalErr := loadConfigFile(globalPath)

	if localErr != nil && !errors.Is(localErr, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to load local config %s: %w", localPath, localErr)
	}
	if globalErr != nil && !errors.Is(globalErr, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to load global config %s: %w", globalPath, globalErr)
	}
	if localErr != nil && globalErr != nil {
		return nil, fmt.Errorf("no config file found: checked %s and %s", localPath, globalPath)
	}

	var cfg *Config
	switch {
	case localErr == nil && globalErr == nil:
		cfg = mergeConfig(globalCfg, localCfg)
	case localErr == nil:
		cfg = localCfg
	default:
		cfg = globalCfg
	}

	return finalizeConfig(cfg)
}

func finalizeConfig(cfg *Config) (*Config, error) {
	applyDefaults(cfg)
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return &cfg, nil
}

func globalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "yomite", "config.json"), nil
}

func applyDefaults(cfg *Config) {
	for name, p := range cfg.Providers {
		if p.Origin == "" {
			p.Origin = defaultOrigin
			cfg.Providers[name] = p
		}
	}
}

func validate(cfg *Config) error {
	if cfg.Log.Path == "" {
		return fmt.Errorf("log.path is required")
	}
	switch cfg.Log.Level {
	case "debug", "info", "warn":
		// valid
	default:
		return fmt.Errorf("log.level must be one of: debug, info, warn (got %q)", cfg.Log.Level)
	}

	if cfg.DefaultProvider != "" {
		if _, ok := cfg.Providers[cfg.DefaultProvider]; !ok {
			return fmt.Errorf("default_provider %q not found in providers", cfg.DefaultProvider)
		}
	}
	if cfg.DefaultPersona != "" {
		if _, ok := cfg.Personas[cfg.DefaultPersona]; !ok {
			return fmt.Errorf("default_persona %q not found in personas", cfg.DefaultPersona)
		}
	}
	return nil
}

// ToSlogLevel は設定ファイルのログレベル文字列を slog.Level に変換する。
func ToSlogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	default:
		return slog.LevelWarn
	}
}

func mergeConfig(base, override *Config) *Config {
	merged := Config{
		Log:             base.Log,
		DefaultProvider: base.DefaultProvider,
		DefaultPersona:  base.DefaultPersona,
		Providers:       make(map[string]ProviderConfig, len(base.Providers)),
		Personas:        make(map[string]Persona, len(base.Personas)),
	}
	for k, v := range base.Providers {
		merged.Providers[k] = v
	}
	for k, v := range base.Personas {
		merged.Personas[k] = v
	}

	if override.Log.Path != "" {
		merged.Log = override.Log
	}
	if override.DefaultProvider != "" {
		merged.DefaultProvider = override.DefaultProvider
	}
	if override.DefaultPersona != "" {
		merged.DefaultPersona = override.DefaultPersona
	}

	for k, v := range override.Providers {
		merged.Providers[k] = v
	}
	for k, v := range override.Personas {
		merged.Personas[k] = v
	}

	return &merged
}
