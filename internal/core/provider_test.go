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

func TestBuildNotePromptContainsJSONFormat(t *testing.T) {
	req := SimulationRequest{
		Phase:           PhaseNote,
		SystemPrompt:    "あなたは初学者の読者です。",
		CurrentSentence: "テスト文。",
		CurrentIndex:    0,
		TotalSentences:  5,
		Memory:          "",
	}

	_, user := BuildNotePrompt(req)

	if !strings.Contains(user, "current_index") {
		t.Error("user message should contain current_index field instruction")
	}
	if !strings.Contains(user, "next_index") {
		t.Error("user message should contain next_index field instruction")
	}
	if !strings.Contains(user, "note") {
		t.Error("user message should contain note field instruction")
	}
	if !strings.Contains(user, "JSON") {
		t.Error("user message should mention JSON format")
	}
}

func TestBuildMemoryPromptContainsJSONFormat(t *testing.T) {
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

	if !strings.Contains(user, "memory") {
		t.Error("user message should contain memory field instruction")
	}
	if !strings.Contains(user, "JSON") {
		t.Error("user message should mention JSON format")
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

func TestParseMemoryResponseValid(t *testing.T) {
	input := `{"memory": "第3文で認知科学が定義された。"}`

	resp, err := ParseMemoryResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Memory != "第3文で認知科学が定義された。" {
		t.Errorf("Memory: got %q", resp.Memory)
	}
}

func TestParseMemoryResponseInvalidJSON(t *testing.T) {
	_, err := ParseMemoryResponse("not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !isErrInvalidJSON(err) {
		t.Errorf("expected ErrInvalidJSON, got %T: %v", err, err)
	}
}

func TestParseMemoryResponseMarkdownCodeBlock(t *testing.T) {
	input := "```json\n{\"memory\": \"テスト記憶\"}\n```"

	resp, err := ParseMemoryResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Memory != "テスト記憶" {
		t.Errorf("Memory: got %q, want %q", resp.Memory, "テスト記憶")
	}
}

func TestParseNoteResponse_Valid(t *testing.T) {
	input := `{
		"current_index": 5,
		"next_index": 6,
		"note": {"type": "QUESTION", "content": "この用語の定義がまだ出てきていない"},
		"memory": "第3文で認知科学が定義された。"
	}`

	resp, err := ParseNoteResponse(input, 10)
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
}

func TestParseNoteResponse_NullFields(t *testing.T) {
	input := `{"current_index": 3, "next_index": null, "note": null, "memory": ""}`

	resp, err := ParseNoteResponse(input, 10)
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

func TestParseNoteResponse_InvalidJSON(t *testing.T) {
	_, err := ParseNoteResponse("not json at all", 10)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !isErrInvalidJSON(err) {
		t.Errorf("expected ErrInvalidJSON, got %T: %v", err, err)
	}
}

func TestParseNoteResponse_EmptyString(t *testing.T) {
	_, err := ParseNoteResponse("", 10)
	if err == nil {
		t.Fatal("expected error for empty string")
	}
	if !isErrInvalidJSON(err) {
		t.Errorf("expected ErrInvalidJSON, got %T: %v", err, err)
	}
}

func TestParseNoteResponse_CurrentIndexOutOfRange(t *testing.T) {
	input := `{"current_index": 10, "next_index": 0, "note": null, "memory": ""}`

	_, err := ParseNoteResponse(input, 10) // totalSentences=10, valid range 0-9
	if err == nil {
		t.Fatal("expected error for out-of-range current_index")
	}
	if !isErrIndexOutOfRange(err) {
		t.Errorf("expected ErrIndexOutOfRange, got %T: %v", err, err)
	}
}

func TestParseNoteResponse_NextIndexEqualsTotalSentences(t *testing.T) {
	// NOTE: next_index == totalSentences はLLMが最終文の次へ進もうとしたケース。
	// 範囲外エラーではなく読了（nil）として扱う。
	input := `{"current_index": 9, "next_index": 10, "note": null, "memory": ""}`

	resp, err := ParseNoteResponse(input, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NextIndex != nil {
		t.Errorf("NextIndex: got %v, want nil (should be treated as end-of-reading)", resp.NextIndex)
	}
}

func TestParseNoteResponse_NextIndexOutOfRange(t *testing.T) {
	input := `{"current_index": 0, "next_index": 11, "note": null, "memory": ""}`

	_, err := ParseNoteResponse(input, 10)
	if err == nil {
		t.Fatal("expected error for out-of-range next_index")
	}
	if !isErrIndexOutOfRange(err) {
		t.Errorf("expected ErrIndexOutOfRange, got %T: %v", err, err)
	}
}

func TestParseNoteResponse_NegativeNextIndex(t *testing.T) {
	input := `{"current_index": 0, "next_index": -1, "note": null, "memory": ""}`

	_, err := ParseNoteResponse(input, 10)
	if err == nil {
		t.Fatal("expected error for negative next_index")
	}
	if !isErrIndexOutOfRange(err) {
		t.Errorf("expected ErrIndexOutOfRange, got %T: %v", err, err)
	}
}

func TestParseNoteResponse_MarkdownCodeBlock(t *testing.T) {
	input := "```json\n{\"current_index\": 0, \"next_index\": 1, \"note\": null, \"memory\": null}\n```"

	resp, err := ParseNoteResponse(input, 5)
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
	input := "```\n{\"current_index\": 0, \"next_index\": 1, \"note\": null, \"memory\": null}\n```"

	resp, err := ParseNoteResponse(input, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CurrentIndex != 0 {
		t.Errorf("CurrentIndex: got %d, want 0", resp.CurrentIndex)
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

func isErrInvalidJSON(err error) bool {
	_, ok := err.(*ErrInvalidJSON)
	return ok
}

func isErrIndexOutOfRange(err error) bool {
	_, ok := err.(*ErrIndexOutOfRange)
	return ok
}
