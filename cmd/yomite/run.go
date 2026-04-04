package yomite

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/canpok1/yomite/core"
)

var providerFactory func(cfg core.ProviderConfig) core.Provider

func init() {
	providerFactory = func(cfg core.ProviderConfig) core.Provider {
		return core.NewOllamaProvider(cfg.Origin, cfg.Model)
	}
}

// Run は yomite run サブコマンドを実行する。
func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		filePath   string
		providerID string
		personaID  string
		jsonOutput bool
		configPath string
	)

	fs.StringVar(&filePath, "f", "", "入力テキストファイルのパス（必須）")
	fs.StringVar(&providerID, "provider", "", "プロバイダID指定")
	fs.StringVar(&personaID, "persona", "", "ペルソナID指定")
	fs.BoolVar(&jsonOutput, "json", false, "出力をJSON形式に切替")
	fs.StringVar(&configPath, "config", "", "設定ファイルのパスを明示指定")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if filePath == "" {
		_, _ = fmt.Fprintln(stderr, "エラー: -f フラグは必須です（入力テキストファイルのパスを指定してください）")
		return 1
	}

	rawText, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintf(stderr, "エラー: ファイルが見つかりません: %s\n", filePath)
		} else {
			_, _ = fmt.Fprintf(stderr, "エラー: ファイルの読み込みに失敗しました: %v\n", err)
		}
		return 1
	}

	cfg, err := core.LoadConfig(configPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "エラー: 設定ファイルの読み込みに失敗しました: %v\n", err)
		return 1
	}

	// ログファイルを開く
	logFile, err := os.OpenFile(cfg.Log.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "エラー: ログファイルを開けませんでした: %v\n", err)
		return 1
	}
	defer func() { _ = logFile.Close() }()

	handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: core.ToSlogLevel(cfg.Log.Level),
	})
	logger := slog.New(handler)

	logger.Info("config loaded",
		"log_level", cfg.Log.Level,
		"log_path", cfg.Log.Path,
		"provider", providerID,
		"persona", personaID,
	)

	if providerID == "" {
		providerID = cfg.DefaultProvider
	}
	if providerID == "" {
		_, _ = fmt.Fprintln(stderr, "エラー: プロバイダが指定されていません（--provider フラグまたは設定ファイルの default_provider を設定してください）")
		return 1
	}
	providerCfg, ok := cfg.Providers[providerID]
	if !ok {
		_, _ = fmt.Fprintf(stderr, "エラー: プロバイダ %q が設定に存在しません\n", providerID)
		return 1
	}

	if personaID == "" {
		personaID = cfg.DefaultPersona
	}
	if personaID == "" {
		_, _ = fmt.Fprintln(stderr, "エラー: ペルソナが指定されていません（--persona フラグまたは設定ファイルの default_persona を設定してください）")
		return 1
	}
	persona, ok := cfg.Personas[personaID]
	if !ok {
		_, _ = fmt.Fprintf(stderr, "エラー: ペルソナ %q が設定に存在しません\n", personaID)
		return 1
	}

	doc := core.Document{
		ID:      filePath,
		RawText: string(rawText),
	}
	doc.Sentences = doc.SplitSentences()

	provider := core.NewLoggingProvider(providerFactory(providerCfg), logger)

	onStep := func(s core.SimulationStep) error {
		if jsonOutput {
			return outputStepJSON(stdout, s)
		}
		return outputStepText(stdout, s, doc)
	}

	if err := core.RunSimulation(doc, persona, provider, logger, onStep); err != nil {
		_, _ = fmt.Fprintf(stderr, "エラー: シミュレーション実行に失敗しました: %v\n", err)
		return 1
	}

	return 0
}

// outputStepJSON は1ステップをJSON Lines形式で出力する。
func outputStepJSON(w io.Writer, s core.SimulationStep) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

// outputStepText は1ステップをテキスト形式で出力する。
func outputStepText(w io.Writer, s core.SimulationStep, doc core.Document) error {
	sentence := ""
	if s.SentenceIdx >= 0 && s.SentenceIdx < len(doc.Sentences) {
		sentence = doc.Sentences[s.SentenceIdx].Content
	}

	direction := ""
	if s.TargetIdx != nil {
		target := *s.TargetIdx
		switch {
		case target > s.SentenceIdx:
			direction = fmt.Sprintf("→ 先読み (→%d)", target)
		case target < s.SentenceIdx:
			direction = fmt.Sprintf("← 読み返し (→%d)", target)
		default:
			direction = fmt.Sprintf("● 再読 (→%d)", target)
		}
	} else {
		direction = "■ 読了"
	}

	if _, err := fmt.Fprintf(w, "[Step %d] 文%d: %s\n", s.Step, s.SentenceIdx, sentence); err != nil {
		return err
	}

	if s.Note != nil {
		if _, err := fmt.Fprintf(w, "  Note[%s]: %s\n", s.Note.Type, s.Note.Content); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(w, "  %s\n", direction)
	return err
}
