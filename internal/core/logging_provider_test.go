package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestLoggingProvider_LogsRequestAndResponse(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	inner := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: intPtr(1), Memory: "memo", RawResponseText: `{"current_index":0,"next_index":1,"memory":"memo"}`},
		},
	}

	lp := NewLoggingProvider(inner, logger)
	req := SimulationRequest{
		SystemPrompt:    "テスト用",
		CurrentSentence: "文1。",
		CurrentIndex:    0,
		TotalSentences:  2,
		Memory:          "",
	}

	resp, err := lp.Execute(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CurrentIndex != 0 {
		t.Errorf("CurrentIndex: got %d, want 0", resp.CurrentIndex)
	}

	logs := buf.String()
	if !strings.Contains(logs, "llm request") {
		t.Errorf("expected 'llm request' log entry, got: %s", logs)
	}
	if !strings.Contains(logs, "llm response") {
		t.Errorf("expected 'llm response' log entry, got: %s", logs)
	}
	if !strings.Contains(logs, "duration_ms") {
		t.Errorf("expected 'duration_ms' in log, got: %s", logs)
	}
}

func TestLoggingProvider_LogsOnError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	inner := &mockProvider{
		errors: []error{errors.New("connection failed")},
	}

	lp := NewLoggingProvider(inner, logger)
	req := SimulationRequest{
		SystemPrompt:    "テスト用",
		CurrentSentence: "文1。",
		CurrentIndex:    0,
		TotalSentences:  1,
	}

	_, err := lp.Execute(req)
	if err == nil {
		t.Fatal("expected error")
	}

	logs := buf.String()
	if !strings.Contains(logs, "llm request") {
		t.Errorf("expected 'llm request' log entry even on error")
	}
	if !strings.Contains(logs, "llm error") {
		t.Errorf("expected 'llm error' log entry, got: %s", logs)
	}
}

func TestLoggingProvider_NoLogAtInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	inner := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: nil, Memory: "m"},
		},
	}

	lp := NewLoggingProvider(inner, logger)
	req := SimulationRequest{
		SystemPrompt:    "テスト用",
		CurrentSentence: "文1。",
		CurrentIndex:    0,
		TotalSentences:  1,
	}

	_, err := lp.Execute(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// debug レベルのログは info ハンドラーでは出力されない
	if strings.Contains(buf.String(), "llm request") {
		t.Errorf("expected no debug logs at info level, got: %s", buf.String())
	}
}

func TestLoggingProvider_RequestFieldsInLog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	inner := &mockProvider{
		responses: []SimulationResponse{
			{CurrentIndex: 0, NextIndex: nil, Memory: "m", RawResponseText: "raw"},
		},
	}

	lp := NewLoggingProvider(inner, logger)
	req := SimulationRequest{
		SystemPrompt:    "システムプロンプト",
		CurrentSentence: "テスト文。",
		CurrentIndex:    2,
		TotalSentences:  5,
		Memory:          "記憶",
	}

	_, _ = lp.Execute(req)

	// Parse log lines to verify fields
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log lines, got %d", len(lines))
	}

	// Check request log
	var reqLog map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &reqLog); err != nil {
		t.Fatalf("failed to parse request log: %v", err)
	}
	if reqLog["msg"] != "llm request" {
		t.Errorf("expected 'llm request', got %v", reqLog["msg"])
	}
	if int(reqLog["current_index"].(float64)) != 2 {
		t.Errorf("expected current_index=2, got %v", reqLog["current_index"])
	}

	// Check response log
	var respLog map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &respLog); err != nil {
		t.Fatalf("failed to parse response log: %v", err)
	}
	if respLog["msg"] != "llm response" {
		t.Errorf("expected 'llm response', got %v", respLog["msg"])
	}
	if respLog["raw_response"] != "raw" {
		t.Errorf("expected raw_response='raw', got %v", respLog["raw_response"])
	}
}
