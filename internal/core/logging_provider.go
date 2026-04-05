package core

import (
	"context"
	"log/slog"
	"time"
)

// LoggingProvider は Provider をラップし、LLMリクエスト/レスポンスをログに記録するデコレータ。
type LoggingProvider struct {
	inner  Provider
	logger *slog.Logger
}

// NewLoggingProvider は LoggingProvider を生成する。
func NewLoggingProvider(inner Provider, logger *slog.Logger) *LoggingProvider {
	return &LoggingProvider{inner: inner, logger: logger}
}

// Execute は内部プロバイダの Execute を呼び出し、リクエストとレスポンスをログに記録する。
func (p *LoggingProvider) Execute(req SimulationRequest) (SimulationResponse, error) {
	debugEnabled := p.logger.Enabled(context.Background(), slog.LevelDebug)

	if debugEnabled {
		system, user := BuildPrompt(req)
		p.logger.Debug("llm request",
			"current_index", req.CurrentIndex,
			"total_sentences", req.TotalSentences,
			"system_prompt", system,
			"user_prompt", user,
		)
	}

	start := time.Now()
	resp, err := p.inner.Execute(req)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		p.logger.Warn("llm error",
			"current_index", req.CurrentIndex,
			"error", err.Error(),
			"duration_ms", durationMs,
		)
		return resp, err
	}

	if debugEnabled {
		p.logger.Debug("llm response",
			"current_index", req.CurrentIndex,
			"raw_response", resp.RawResponseText,
			"duration_ms", durationMs,
		)
	}

	return resp, nil
}
