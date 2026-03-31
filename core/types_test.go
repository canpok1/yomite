package core

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNoteTypeConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		noteType NoteType
		want     string
	}{
		{"QUESTION", NoteTypeQuestion, "QUESTION"},
		{"RESOLVED", NoteTypeResolved, "RESOLVED"},
		{"CONFUSION", NoteTypeConfusion, "CONFUSION"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.noteType) != tt.want {
				t.Errorf("got %q, want %q", tt.noteType, tt.want)
			}
		})
	}
}

func TestNoteJSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := Note{
		Type:    NoteTypeQuestion,
		Content: "What does this mean?",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Note
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded != original {
		t.Errorf("round trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestSentenceJSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := Sentence{
		Index:   0,
		Content: "Hello world.",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Sentence
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded != original {
		t.Errorf("round trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestDocumentJSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := Document{
		ID:      "doc-1",
		RawText: "Hello world. Goodbye world.",
		Sentences: []Sentence{
			{Index: 0, Content: "Hello world."},
			{Index: 1, Content: "Goodbye world."},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Document
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(decoded, original) {
		t.Errorf("round trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestSimulationStepJSONRoundTripWithValues(t *testing.T) {
	t.Parallel()

	targetIdx := 3
	note := &Note{
		Type:    NoteTypeConfusion,
		Content: "This is confusing.",
	}
	original := SimulationStep{
		Step:        1,
		SentenceIdx: 2,
		TargetIdx:   &targetIdx,
		Note:        note,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded SimulationStep
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Step != original.Step {
		t.Errorf("Step: got %d, want %d", decoded.Step, original.Step)
	}
	if decoded.SentenceIdx != original.SentenceIdx {
		t.Errorf("SentenceIdx: got %d, want %d", decoded.SentenceIdx, original.SentenceIdx)
	}
	if decoded.TargetIdx == nil {
		t.Fatal("TargetIdx: got nil, want non-nil")
	}
	if *decoded.TargetIdx != *original.TargetIdx {
		t.Errorf("TargetIdx: got %d, want %d", *decoded.TargetIdx, *original.TargetIdx)
	}
	if decoded.Note == nil {
		t.Fatal("Note: got nil, want non-nil")
	}
	if *decoded.Note != *original.Note {
		t.Errorf("Note: got %+v, want %+v", *decoded.Note, *original.Note)
	}
}

func TestSimulationStepJSONRoundTripWithNil(t *testing.T) {
	t.Parallel()

	original := SimulationStep{
		Step:        0,
		SentenceIdx: 0,
		TargetIdx:   nil,
		Note:        nil,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify null is in JSON output
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}
	if raw["next_index"] != nil {
		t.Errorf("next_index should be null in JSON, got %v", raw["next_index"])
	}
	if raw["note"] != nil {
		t.Errorf("note should be null in JSON, got %v", raw["note"])
	}

	var decoded SimulationStep
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.TargetIdx != nil {
		t.Errorf("TargetIdx: got %v, want nil", decoded.TargetIdx)
	}
	if decoded.Note != nil {
		t.Errorf("Note: got %v, want nil", decoded.Note)
	}
}
