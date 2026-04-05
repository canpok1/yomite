package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSimulationResponseJSONRoundTrip(t *testing.T) {
	nextIdx := 6
	resp := SimulationResponse{
		CurrentIndex: 5,
		NextIndex:    &nextIdx,
		Note: &Note{
			Type:    NoteTypeQuestion,
			Content: "この用語の定義がまだ出てきていない",
		},
		Memory: "第3文で認知科学が定義された。",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var got SimulationResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.CurrentIndex != resp.CurrentIndex {
		t.Errorf("CurrentIndex: got %d, want %d", got.CurrentIndex, resp.CurrentIndex)
	}
	if got.NextIndex == nil || *got.NextIndex != *resp.NextIndex {
		t.Errorf("NextIndex: got %v, want %v", got.NextIndex, resp.NextIndex)
	}
	if got.Note == nil || got.Note.Type != NoteTypeQuestion || got.Note.Content != resp.Note.Content {
		t.Errorf("Note: got %+v, want %+v", got.Note, resp.Note)
	}
	if got.Memory != resp.Memory {
		t.Errorf("Memory: got %q, want %q", got.Memory, resp.Memory)
	}
}

func TestSimulationResponseJSONRoundTripNil(t *testing.T) {
	resp := SimulationResponse{
		CurrentIndex: 3,
		NextIndex:    nil,
		Note:         nil,
		Memory:       "",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// null がJSONに含まれることを確認
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"next_index":null`) {
		t.Errorf("expected next_index:null in JSON, got %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"note":null`) {
		t.Errorf("expected note:null in JSON, got %s", jsonStr)
	}

	var got SimulationResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if got.NextIndex != nil {
		t.Errorf("NextIndex: got %v, want nil", got.NextIndex)
	}
	if got.Note != nil {
		t.Errorf("Note: got %+v, want nil", got.Note)
	}
}

func TestBuildPromptSystemPrompt(t *testing.T) {
	req := SimulationRequest{
		SystemPrompt:    "あなたは初学者の読者です。",
		CurrentSentence: "テスト文。",
		CurrentIndex:    0,
		TotalSentences:  5,
		Memory:          "",
	}

	system, _ := BuildPrompt(req)

	if !strings.Contains(system, req.SystemPrompt) {
		t.Errorf("system prompt should contain persona prompt, got %q", system)
	}
}

func TestBuildNotePromptContainsNewFormat(t *testing.T) {
	req := SimulationRequest{
		Phase:           PhaseNote,
		SystemPrompt:    "あなたは初学者の読者です。",
		CurrentSentence: "テスト文。",
		CurrentIndex:    0,
		TotalSentences:  5,
		Memory:          "",
	}

	_, user := BuildNotePrompt(req)

	if !strings.Contains(user, "next_action") {
		t.Error("user message should contain next_action field instruction")
	}
	if !strings.Contains(user, "feeling") {
		t.Error("user message should contain feeling field instruction")
	}
	if !strings.Contains(user, "feeling_type") {
		t.Error("user message should contain feeling_type field instruction")
	}
	if !strings.Contains(user, "JSON") {
		t.Error("user message should mention JSON format")
	}
	// current_index / next_index はLLMに返させないのでプロンプトの出力形式に含まれない
}

func TestBuildNotePromptContainsLastIndex(t *testing.T) {
	req := SimulationRequest{
		Phase:           PhaseNote,
		SystemPrompt:    "テスト",
		CurrentSentence: "テスト文。",
		CurrentIndex:    0,
		TotalSentences:  10,
		Memory:          "",
	}

	_, user := BuildNotePrompt(req)

	if !strings.Contains(user, "最後の文は9") {
		t.Error("user message should contain last index (TotalSentences - 1)")
	}
}

func TestBuildMemoryPromptContainsPlainTextInstruction(t *testing.T) {
	req := SimulationRequest{
		Phase:           PhaseMemory,
		SystemPrompt:    "あなたは初学者の読者です。",
		CurrentSentence: "テスト文。",
		CurrentIndex:    0,
		TotalSentences:  5,
		Memory:          "",
		MemoryCapacity:  100,
		Note:            &Note{Type: NoteTypeQuestion, Content: "テスト感想"},
	}

	_, user := BuildMemoryPrompt(req)

	if !strings.Contains(user, "プレーンテキスト") {
		t.Error("user message should mention plain text format")
	}
	if !strings.Contains(user, "テスト感想") {
		t.Error("user message should contain note content")
	}
}

func TestBuildNotePromptContainsContext(t *testing.T) {
	req := SimulationRequest{
		Phase:           PhaseNote,
		SystemPrompt:    "あなたは初学者の読者です。",
		CurrentSentence: "これはテスト文です。",
		CurrentIndex:    2,
		TotalSentences:  10,
		Memory:          "前の文で重要な概念が出た。",
	}

	_, user := BuildNotePrompt(req)

	if !strings.Contains(user, req.CurrentSentence) {
		t.Error("user message should contain current sentence")
	}
	if !strings.Contains(user, "2") {
		t.Error("user message should contain current index")
	}
	if !strings.Contains(user, "10") {
		t.Error("user message should contain total sentences")
	}
	if !strings.Contains(user, req.Memory) {
		t.Error("user message should contain memory")
	}
}

func TestBuildNotePromptEmptyMemory(t *testing.T) {
	req := SimulationRequest{
		Phase:           PhaseNote,
		SystemPrompt:    "あなたは初学者の読者です。",
		CurrentSentence: "テスト文。",
		CurrentIndex:    0,
		TotalSentences:  5,
		Memory:          "",
	}

	_, user := BuildNotePrompt(req)

	if user == "" {
		t.Error("user message should not be empty even with empty memory")
	}
}

func TestBuildMemoryPromptContainsContext(t *testing.T) {
	req := SimulationRequest{
		Phase:           PhaseMemory,
		SystemPrompt:    "あなたは初学者の読者です。",
		CurrentSentence: "これはテスト文です。",
		CurrentIndex:    2,
		TotalSentences:  10,
		Memory:          "前の文で重要な概念が出た。",
		MemoryCapacity:  100,
		Note:            nil,
	}

	_, user := BuildMemoryPrompt(req)

	if !strings.Contains(user, req.CurrentSentence) {
		t.Error("user message should contain current sentence")
	}
	if !strings.Contains(user, req.Memory) {
		t.Error("user message should contain memory")
	}
	if !strings.Contains(user, "（なし）") {
		t.Error("user message should contain '（なし）' for nil note")
	}
}

func TestParseMemoryResponsePlainText(t *testing.T) {
	input := "第3文で認知科学が定義された。"

	resp, err := ParseMemoryResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Memory != "第3文で認知科学が定義された。" {
		t.Errorf("Memory: got %q", resp.Memory)
	}
}

func TestParseMemoryResponseWithWhitespace(t *testing.T) {
	input := "  テスト記憶  \n"

	resp, err := ParseMemoryResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Memory != "テスト記憶" {
		t.Errorf("Memory: got %q, want %q", resp.Memory, "テスト記憶")
	}
}

func TestParseMemoryResponseMarkdownCodeBlock(t *testing.T) {
	input := "```\nテスト記憶\n```"

	resp, err := ParseMemoryResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Memory != "テスト記憶" {
		t.Errorf("Memory: got %q, want %q", resp.Memory, "テスト記憶")
	}
}

func TestParseNoteResponse_NextAction(t *testing.T) {
	input := `{"next_action": "next", "feeling": "この用語の定義がまだ出てきていない", "feeling_type": "question"}`

	resp, err := ParseNoteResponse(input, 5, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CurrentIndex != 5 {
		t.Errorf("CurrentIndex: got %d, want 5", resp.CurrentIndex)
	}
	if resp.NextIndex == nil || *resp.NextIndex != 6 {
		t.Errorf("NextIndex: got %v, want 6", resp.NextIndex)
	}
	if resp.Note == nil || resp.Note.Type != NoteTypeQuestion {
		t.Errorf("Note.Type: got %v, want QUESTION", resp.Note)
	}
	if resp.Note.Content != "この用語の定義がまだ出てきていない" {
		t.Errorf("Note.Content: got %q", resp.Note.Content)
	}
}

func TestParseNoteResponse_NullFeeling(t *testing.T) {
	input := `{"next_action": "finish", "feeling": null, "feeling_type": null}`

	resp, err := ParseNoteResponse(input, 9, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NextIndex != nil {
		t.Errorf("NextIndex: got %v, want nil", resp.NextIndex)
	}
	if resp.Note != nil {
		t.Errorf("Note: got %v, want nil", resp.Note)
	}
}

func TestParseNoteResponse_BackAction(t *testing.T) {
	input := `{"next_action": "back:2", "feeling": "混乱した", "feeling_type": "confusion"}`

	resp, err := ParseNoteResponse(input, 5, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NextIndex == nil || *resp.NextIndex != 3 {
		t.Errorf("NextIndex: got %v, want 3 (5-2)", resp.NextIndex)
	}
	if resp.Note == nil || resp.Note.Type != NoteTypeConfusion {
		t.Errorf("Note.Type: got %v, want CONFUSION", resp.Note)
	}
}

func TestParseNoteResponse_BackActionClampToZero(t *testing.T) {
	input := `{"next_action": "back:10", "feeling": null, "feeling_type": null}`

	resp, err := ParseNoteResponse(input, 2, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NextIndex == nil || *resp.NextIndex != 0 {
		t.Errorf("NextIndex: got %v, want 0 (clamped)", resp.NextIndex)
	}
}

func TestParseNoteResponse_NextAtLastSentence(t *testing.T) {
	// 最後の文で "next" を返した場合は読了
	input := `{"next_action": "next", "feeling": null, "feeling_type": null}`

	resp, err := ParseNoteResponse(input, 9, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NextIndex != nil {
		t.Errorf("NextIndex: got %v, want nil (should be treated as end-of-reading)", resp.NextIndex)
	}
}

func TestParseNoteResponse_InvalidJSON(t *testing.T) {
	_, err := ParseNoteResponse("not json at all", 0, 10)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !isErrInvalidJSON(err) {
		t.Errorf("expected ErrInvalidJSON, got %T: %v", err, err)
	}
}

func TestParseNoteResponse_EmptyString(t *testing.T) {
	_, err := ParseNoteResponse("", 0, 10)
	if err == nil {
		t.Fatal("expected error for empty string")
	}
	if !isErrInvalidJSON(err) {
		t.Errorf("expected ErrInvalidJSON, got %T: %v", err, err)
	}
}

func TestParseNoteResponse_InvalidNextAction(t *testing.T) {
	input := `{"next_action": "jump:5", "feeling": null, "feeling_type": null}`

	_, err := ParseNoteResponse(input, 0, 10)
	if err == nil {
		t.Fatal("expected error for invalid next_action")
	}
	if !isErrInvalidNextAction(err) {
		t.Errorf("expected ErrInvalidNextAction, got %T: %v", err, err)
	}
}

func TestParseNoteResponse_BackZero(t *testing.T) {
	input := `{"next_action": "back:0", "feeling": null, "feeling_type": null}`

	_, err := ParseNoteResponse(input, 5, 10)
	if err == nil {
		t.Fatal("expected error for back:0")
	}
}

func TestParseNoteResponse_ResolvedType(t *testing.T) {
	input := `{"next_action": "next", "feeling": "疑問が解消された", "feeling_type": "resolved"}`

	resp, err := ParseNoteResponse(input, 3, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Note == nil || resp.Note.Type != NoteTypeResolved {
		t.Errorf("Note.Type: got %v, want RESOLVED", resp.Note)
	}
}

func TestParseNoteResponse_MarkdownCodeBlock(t *testing.T) {
	input := "```json\n{\"next_action\": \"next\", \"feeling\": null, \"feeling_type\": null}\n```"

	resp, err := ParseNoteResponse(input, 0, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CurrentIndex != 0 {
		t.Errorf("CurrentIndex: got %d, want 0", resp.CurrentIndex)
	}
	if resp.NextIndex == nil || *resp.NextIndex != 1 {
		t.Errorf("NextIndex: got %v, want 1", resp.NextIndex)
	}
}

func TestParseNoteResponse_MarkdownCodeBlockNoLang(t *testing.T) {
	input := "```\n{\"next_action\": \"next\", \"feeling\": null, \"feeling_type\": null}\n```"

	resp, err := ParseNoteResponse(input, 0, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CurrentIndex != 0 {
		t.Errorf("CurrentIndex: got %d, want 0", resp.CurrentIndex)
	}
}

func TestParseNoteResponse_FeelingWithoutType(t *testing.T) {
	// feeling があるが feeling_type が null の場合、デフォルトで QUESTION
	input := `{"next_action": "next", "feeling": "気になる", "feeling_type": null}`

	resp, err := ParseNoteResponse(input, 0, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Note == nil || resp.Note.Type != NoteTypeQuestion {
		t.Errorf("Note.Type: got %v, want QUESTION (default)", resp.Note)
	}
}

func TestStripMarkdownCodeBlock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain JSON",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "with json lang",
			input: "```json\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "without lang",
			input: "```\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "with surrounding whitespace",
			input: "  ```json\n{\"key\": \"value\"}\n```  ",
			want:  `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMarkdownCodeBlock(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseNextAction(t *testing.T) {
	tests := []struct {
		name           string
		action         string
		currentIdx     int
		totalSentences int
		wantNextIdx    *int
		wantErr        bool
	}{
		{
			name:           "next from middle",
			action:         "next",
			currentIdx:     3,
			totalSentences: 10,
			wantNextIdx:    intPtr(4),
		},
		{
			name:           "next from last sentence",
			action:         "next",
			currentIdx:     9,
			totalSentences: 10,
			wantNextIdx:    nil,
		},
		{
			name:           "finish",
			action:         "finish",
			currentIdx:     5,
			totalSentences: 10,
			wantNextIdx:    nil,
		},
		{
			name:           "back:1",
			action:         "back:1",
			currentIdx:     5,
			totalSentences: 10,
			wantNextIdx:    intPtr(4),
		},
		{
			name:           "back:3",
			action:         "back:3",
			currentIdx:     5,
			totalSentences: 10,
			wantNextIdx:    intPtr(2),
		},
		{
			name:           "back clamp to 0",
			action:         "back:10",
			currentIdx:     2,
			totalSentences: 10,
			wantNextIdx:    intPtr(0),
		},
		{
			name:           "invalid action",
			action:         "skip",
			currentIdx:     0,
			totalSentences: 10,
			wantErr:        true,
		},
		{
			name:           "back:0 is invalid",
			action:         "back:0",
			currentIdx:     5,
			totalSentences: 10,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNextAction(tt.action, tt.currentIdx, tt.totalSentences)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNextIdx == nil {
				if got != nil {
					t.Errorf("got %v, want nil", *got)
				}
			} else {
				if got == nil || *got != *tt.wantNextIdx {
					t.Errorf("got %v, want %d", got, *tt.wantNextIdx)
				}
			}
		})
	}
}

func isErrInvalidJSON(err error) bool {
	_, ok := err.(*ErrInvalidJSON)
	return ok
}

func isErrInvalidNextAction(err error) bool {
	_, ok := err.(*ErrInvalidNextAction)
	return ok
}
