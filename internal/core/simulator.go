package core

import (
	"fmt"
	"log/slog"
)

// RunSimulation はドキュメントに対してAI読者シミュレーションを実行する。
// 各ステップが完了するたびに onStep コールバックを呼び出し、結果を逐次的に返す。
// onStep がエラーを返した場合、シミュレーションを中断しそのエラーを返す。
func RunSimulation(doc Document, persona Persona, provider Provider, logger *slog.Logger, onStep func(SimulationStep) error) error {
	totalSentences := len(doc.Sentences)
	if totalSentences == 0 {
		return nil
	}

	maxSteps := persona.MaxSteps
	if maxSteps <= 0 {
		maxSteps = totalSentences * 3
	}

	logger.Info("simulation started",
		"file", doc.ID,
		"total_sentences", totalSentences,
		"max_steps", maxSteps,
	)

	var memory string
	currentIdx := 0
	completedSteps := 0

	for step := 0; step < maxSteps; step++ {
		// Phase 1: 感想生成（note + next_index）
		noteReq := SimulationRequest{
			Phase:           PhaseNote,
			SystemPrompt:    persona.SystemPrompt,
			CurrentSentence: doc.Sentences[currentIdx].Content,
			CurrentIndex:    currentIdx,
			TotalSentences:  totalSentences,
			Memory:          memory,
			MemoryCapacity:  persona.MemoryCapacity,
		}

		noteResp, err := provider.Execute(noteReq)
		if err != nil {
			return fmt.Errorf("step %d (note): provider error: %w", step, err)
		}

		// Phase 2: メモリ生成（memory）
		memReq := SimulationRequest{
			Phase:           PhaseMemory,
			SystemPrompt:    persona.SystemPrompt,
			CurrentSentence: doc.Sentences[currentIdx].Content,
			CurrentIndex:    currentIdx,
			TotalSentences:  totalSentences,
			Memory:          memory,
			MemoryCapacity:  persona.MemoryCapacity,
			Note:            noteResp.Note,
		}

		memResp, err := provider.Execute(memReq)
		if err != nil {
			return fmt.Errorf("step %d (memory): provider error: %w", step, err)
		}

		// NOTE: next_index の範囲検証は ParseNoteResponse でも実施済みだが、
		// Provider 実装が ParseNoteResponse を経由しない場合に備えた防御的チェック。
		hasNext := noteResp.NextIndex != nil
		if hasNext {
			nextIdx := *noteResp.NextIndex
			if nextIdx < 0 || nextIdx >= totalSentences {
				return &ErrIndexOutOfRange{
					Field: "next_index",
					Index: nextIdx,
					Max:   totalSentences,
				}
			}
		}

		s := SimulationStep{
			Step:        step,
			SentenceIdx: currentIdx,
			TargetIdx:   noteResp.NextIndex,
			Note:        noteResp.Note,
		}
		if err := onStep(s); err != nil {
			return fmt.Errorf("step %d: callback error: %w", step, err)
		}
		completedSteps++

		logger.Info("step completed",
			"step", step,
			"current_index", currentIdx,
			"next_index", noteResp.NextIndex,
		)

		memory = memResp.Memory
		runes := []rune(memory)
		memoryLen := len(runes)
		if persona.MemoryCapacity > 0 && memoryLen > persona.MemoryCapacity {
			memory = string(runes[:persona.MemoryCapacity])
			logger.Warn("memory truncated",
				"step", step,
				"original_length", memoryLen,
				"truncated_length", persona.MemoryCapacity,
			)
		}

		if !hasNext {
			break
		}

		currentIdx = *noteResp.NextIndex
	}

	logger.Info("simulation finished",
		"total_steps", completedSteps,
	)

	return nil
}
