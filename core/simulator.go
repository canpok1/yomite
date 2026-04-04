package core

import (
	"fmt"
	"log/slog"
	"unicode/utf8"
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
		req := SimulationRequest{
			SystemPrompt:    persona.SystemPrompt,
			CurrentSentence: doc.Sentences[currentIdx].Content,
			CurrentIndex:    currentIdx,
			TotalSentences:  totalSentences,
			Memory:          memory,
		}

		resp, err := provider.Execute(req)
		if err != nil {
			return fmt.Errorf("step %d: provider error: %w", step, err)
		}

		var nextIdx int
		hasNext := resp.NextIndex != nil
		if hasNext {
			nextIdx = *resp.NextIndex
			if nextIdx < 0 || nextIdx >= totalSentences {
				return &ErrIndexOutOfRange{
					Field: "next_index",
					Index: nextIdx,
					Max:   totalSentences,
				}
			}
		}
		// NOTE: ParseResponse が next_index==totalSentences を nil に変換済みなので
		// ここでは追加の境界補正は不要。

		s := SimulationStep{
			Step:        step,
			SentenceIdx: currentIdx,
			TargetIdx:   resp.NextIndex,
			Note:        resp.Note,
		}
		if err := onStep(s); err != nil {
			return fmt.Errorf("step %d: callback error: %w", step, err)
		}
		completedSteps++

		logger.Info("step completed",
			"step", step,
			"current_index", currentIdx,
			"next_index", resp.NextIndex,
		)

		memory = resp.Memory
		memoryLen := utf8.RuneCountInString(memory)
		if persona.MemoryCapacity > 0 && memoryLen > persona.MemoryCapacity {
			memory = string([]rune(memory)[:persona.MemoryCapacity])
			logger.Warn("memory truncated",
				"step", step,
				"original_length", memoryLen,
				"truncated_length", persona.MemoryCapacity,
			)
		}

		if !hasNext {
			break
		}

		currentIdx = nextIdx
	}

	logger.Info("simulation finished",
		"total_steps", completedSteps,
	)

	return nil
}
