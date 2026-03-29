package core

import (
	"fmt"
	"unicode/utf8"
)

// RunSimulation はドキュメントに対してAI読者シミュレーションを実行し、全ステップを返す。
func RunSimulation(doc Document, persona Persona, provider Provider) ([]SimulationStep, error) {
	totalSentences := len(doc.Sentences)
	if totalSentences == 0 {
		return nil, nil
	}

	maxSteps := persona.MaxSteps
	if maxSteps <= 0 {
		maxSteps = totalSentences * 3
	}

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

		steps = append(steps, SimulationStep{
			Step:        step,
			SentenceIdx: currentIdx,
			TargetIdx:   resp.NextIndex,
			Note:        resp.Note,
		})

		// 記憶バッファを更新（memory_capacity で文字数制限）
		memory = resp.Memory
		if persona.MemoryCapacity > 0 && utf8.RuneCountInString(memory) > persona.MemoryCapacity {
			memory = string([]rune(memory)[:persona.MemoryCapacity])
		}

		if !hasNext {
			break
		}

		currentIdx = nextIdx
	}

	return steps, nil
}
