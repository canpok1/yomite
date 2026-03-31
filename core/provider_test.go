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

func TestBuildPromptUserMessageContainsJSONFormat(t *testing.T) {
	req := SimulationRequest{
		SystemPrompt:    "あなたは初学者の読者です。",
		CurrentSentence: "テスト文。",
		CurrentIndex:    0,
		TotalSentences:  5,
		Memory:          "",
	}

	_, user := BuildPrompt(req)

	// JSONフォーマット指示が含まれることを確認
	if !strings.Contains(user, "current_index") {
		t.Error("user message should contain current_index field instruction")
	}
	if !strings.Contains(user, "next_index") {
		t.Error("user message should contain next_index field instruction")
	}
	if !strings.Contains(user, "note") {
		t.Error("user message should contain note field instruction")
	}
	if !strings.Contains(user, "memory") {
		t.Error("user message should contain memory field instruction")
	}
	if !strings.Contains(user, "JSON") {
		t.Error("user message should mention JSON format")
	}
}

func TestBuildPromptUserMessageContainsContext(t *testing.T) {
	req := SimulationRequest{
		SystemPrompt:    "あなたは初学者の読者です。",
		CurrentSentence: "これはテスト文です。",
		CurrentIndex:    2,
		TotalSentences:  10,
		Memory:          "前の文で重要な概念が出た。",
	}

	_, user := BuildPrompt(req)

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

func TestBuildPromptEmptyMemory(t *testing.T) {
	req := SimulationRequest{
		SystemPrompt:    "あなたは初学者の読者です。",
		CurrentSentence: "テスト文。",
		CurrentIndex:    0,
		TotalSentences:  5,
		Memory:          "",
	}

	_, user := BuildPrompt(req)

	// 空の記憶でもエラーなくプロンプトが生成されることを確認
	if user == "" {
		t.Error("user message should not be empty even with empty memory")
	}
}

func TestParseResponseValid(t *testing.T) {
	input := `{
		"current_index": 5,
		"next_index": 6,
		"note": {"type": "QUESTION", "content": "この用語の定義がまだ出てきていない"},
		"memory": "第3文で認知科学が定義された。"
	}`

	resp, err := ParseResponse(input, 10)
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
	if resp.Memory != "第3文で認知科学が定義された。" {
		t.Errorf("Memory: got %q", resp.Memory)
	}
}

func TestParseResponseNullFields(t *testing.T) {
	input := `{"current_index": 3, "next_index": null, "note": null, "memory": ""}`

	resp, err := ParseResponse(input, 10)
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

func TestParseResponseInvalidJSON(t *testing.T) {
	_, err := ParseResponse("not json at all", 10)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !isErrInvalidJSON(err) {
		t.Errorf("expected ErrInvalidJSON, got %T: %v", err, err)
	}
}

func TestParseResponseEmptyString(t *testing.T) {
	_, err := ParseResponse("", 10)
	if err == nil {
		t.Fatal("expected error for empty string")
	}
	if !isErrInvalidJSON(err) {
		t.Errorf("expected ErrInvalidJSON, got %T: %v", err, err)
	}
}

func TestParseResponseCurrentIndexOutOfRange(t *testing.T) {
	input := `{"current_index": 10, "next_index": 0, "note": null, "memory": ""}`

	_, err := ParseResponse(input, 10) // totalSentences=10, valid range 0-9
	if err == nil {
		t.Fatal("expected error for out-of-range current_index")
	}
	if !isErrIndexOutOfRange(err) {
		t.Errorf("expected ErrIndexOutOfRange, got %T: %v", err, err)
	}
}

func TestParseResponseNextIndexEqualsTotalSentences(t *testing.T) {
	// NOTE: next_index == totalSentences はLLMが最終文の次へ進もうとしたケース。
	// 範囲外エラーではなく読了（nil）として扱う。
	input := `{"current_index": 9, "next_index": 10, "note": null, "memory": ""}`

	resp, err := ParseResponse(input, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NextIndex != nil {
		t.Errorf("NextIndex: got %v, want nil (should be treated as end-of-reading)", resp.NextIndex)
	}
}

func TestParseResponseNextIndexOutOfRange(t *testing.T) {
	input := `{"current_index": 0, "next_index": 11, "note": null, "memory": ""}`

	_, err := ParseResponse(input, 10)
	if err == nil {
		t.Fatal("expected error for out-of-range next_index")
	}
	if !isErrIndexOutOfRange(err) {
		t.Errorf("expected ErrIndexOutOfRange, got %T: %v", err, err)
	}
}

func TestParseResponseNegativeNextIndex(t *testing.T) {
	input := `{"current_index": 0, "next_index": -1, "note": null, "memory": ""}`

	_, err := ParseResponse(input, 10)
	if err == nil {
		t.Fatal("expected error for negative next_index")
	}
	if !isErrIndexOutOfRange(err) {
		t.Errorf("expected ErrIndexOutOfRange, got %T: %v", err, err)
	}
}

func TestParseResponseMarkdownCodeBlock(t *testing.T) {
	input := "```json\n{\"current_index\": 0, \"next_index\": 1, \"note\": null, \"memory\": null}\n```"

	resp, err := ParseResponse(input, 5)
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

func TestParseResponseMarkdownCodeBlockNoLang(t *testing.T) {
	input := "```\n{\"current_index\": 0, \"next_index\": 1, \"note\": null, \"memory\": null}\n```"

	resp, err := ParseResponse(input, 5)
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
