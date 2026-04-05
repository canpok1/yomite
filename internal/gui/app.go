package gui

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/canpok1/yomite/internal/core"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventSimulationStep  = "simulation:step"
	eventSimulationDone  = "simulation:done"
	eventSimulationError = "simulation:error"
)

// App はWailsバインディング層。core/ パッケージの機能をフロントエンドに公開する。
type App struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	providerFactory func(cfg core.ProviderConfig) core.Provider
	emitEvent       func(eventName string, data ...any)
}

func NewApp() *App {
	a := &App{}
	a.providerFactory = func(cfg core.ProviderConfig) core.Provider {
		return core.NewOllamaProvider(cfg.Origin, cfg.Model)
	}
	a.emitEvent = func(eventName string, data ...any) {
		wailsRuntime.EventsEmit(a.ctx, eventName, data...)
	}
	return a
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// LoadDocument はテキストを文分割して返す。
func (a *App) LoadDocument(rawText string) []core.Sentence {
	doc := core.Document{RawText: rawText}
	return doc.SplitSentences()
}

// GetConfig は設定ファイルを読み込んで返す。
func (a *App) GetConfig() (*core.Config, error) {
	return core.LoadConfig("")
}

// ListProviders は利用可能なプロバイダID一覧を返す。
func (a *App) ListProviders() ([]string, error) {
	cfg, err := core.LoadConfig("")
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(cfg.Providers))
	for id := range cfg.Providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

// ListPersonas はペルソナID→表示名のマッピングを返す。
func (a *App) ListPersonas() (map[string]string, error) {
	cfg, err := core.LoadConfig("")
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(cfg.Personas))
	for id, p := range cfg.Personas {
		result[id] = p.DisplayName
	}
	return result, nil
}

// StartSimulation はgoroutineでシミュレーションを実行し、Wails Eventsで逐次通知する。
func (a *App) StartSimulation(rawText, providerID, personaID string) error {
	cfg, err := core.LoadConfig("")
	if err != nil {
		return err
	}

	providerCfg, ok := cfg.Providers[providerID]
	if !ok {
		return fmt.Errorf("プロバイダ %q が設定に存在しません", providerID)
	}

	persona, ok := cfg.Personas[personaID]
	if !ok {
		return fmt.Errorf("ペルソナ %q が設定に存在しません", personaID)
	}

	doc := core.Document{ID: "gui-input", RawText: rawText}
	doc.Sentences = doc.SplitSentences()

	provider := a.providerFactory(providerCfg)

	// NOTE: cancel の読み書きのみをロックで保護する。
	// goroutine起動をロック内に入れると呼び出し元がブロックされるため、ロック外で起動する。
	a.mu.Lock()
	if a.cancel != nil {
		a.mu.Unlock()
		return fmt.Errorf("シミュレーションが既に実行中です")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel
	a.mu.Unlock()

	go a.runSimulation(ctx, doc, persona, provider)

	return nil
}

// runSimulation はシミュレーションを実行し、結果をWails Eventsで通知する。
func (a *App) runSimulation(ctx context.Context, doc core.Document, persona core.Persona, provider core.Provider) {
	defer func() {
		a.mu.Lock()
		a.cancel = nil
		a.mu.Unlock()
	}()

	logger := slog.Default()

	onStep := func(s core.SimulationStep) error {
		a.emitEvent(eventSimulationStep, s)
		return ctx.Err()
	}

	if err := core.RunSimulation(doc, persona, provider, logger, onStep); err != nil {
		// NOTE: キャンセル由来のエラーはフロントエンドに通知しない。
		// StopSimulation() 呼び出し元がキャンセルを把握済みのため。
		if ctx.Err() != nil {
			return
		}
		a.emitEvent(eventSimulationError, err.Error())
		return
	}

	a.emitEvent(eventSimulationDone)
}

// StopSimulation は実行中のシミュレーションをキャンセルする。
func (a *App) StopSimulation() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}
