package core

import (
	"fmt"
	"log/slog"
	"unicode/utf8"
)

// RunSimulation はドキュメントに対してAI読者シミュレーションを実行し、全ステップを返す。
func RunSimulation(doc Document, persona Persona, provider Provider, logger *slog.Logger) ([]SimulationStep, error) {
	totalSentences := len(doc.Sentences)
	if totalSentences == 0 {
		return nil, nil
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

	steps := make([]SimulationStep, 0, maxSteps)
	var memory string
	currentIdx := 0

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
			return nil, fmt.Errorf("step %d: provider error: %w", step, err)
		}

		var nextIdx int
		hasNext := resp.NextIndex != nil
		if hasNext {
			nextIdx = *resp.NextIndex
			if nextIdx < 0 || nextIdx >= totalSentences {
				return nil, &ErrIndexOutOfRange{
					Field: "next_index",
					Index: nextIdx,
					Max:   totalSentences,
				}
			}
		}
		// NOTE: ParseResponse が next_index==totalSentences を nil に変換済みなので
		// ここでは追加の境界補正は不要。

		steps = append(steps, SimulationStep{
			Step:        step,
			SentenceIdx: currentIdx,
			TargetIdx:   resp.NextIndex,
			Note:        resp.Note,
		})

		logger.Info("step completed",
			"step", step,
			"current_index", currentIdx,
			"next_index", resp.NextIndex,
		)

		// 記憶バッファを更新（memory_capacity で文字数制限）
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
		"total_steps", len(steps),
	)

	return steps, nil
}
