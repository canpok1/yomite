package core

import (
	"encoding/json"
	"strings"
	"testing"
)

// テストリスト（シンプル→複雑の順）
//
// 型定義テスト:
// TODO: SimulationRequest の各フィールドが正しく設定できる
// TODO: SimulationResponse のJSONラウンドトリップ（全フィールドあり）
// TODO: SimulationResponse のJSONラウンドトリップ（NextIndex=nil, Note=nil）
//
// プロンプト構築テスト:
// TODO: BuildPrompt がシステムプロンプトを返す
// TODO: BuildPrompt がユーザーメッセージにJSON出力フォーマット指示を含む
// TODO: BuildPrompt がユーザーメッセージに現在の文・位置・記憶を含む
// TODO: BuildPrompt で記憶が空の場合の動作
//
// レスポンスパーステスト:
// TODO: ParseResponse が正常なJSONをパースできる
// TODO: ParseResponse でNextIndex=null, Note=nullのパース
// TODO: ParseResponse で不正なJSONに対してErrInvalidJSONを返す
// TODO: ParseResponse で空文字列に対してErrInvalidJSONを返す
// TODO: ParseResponse でcurrent_indexが範囲外の場合にErrIndexOutOfRangeを返す
// TODO: ParseResponse でnext_indexが範囲外の場合にErrIndexOutOfRangeを返す
// TODO: ParseResponse でnext_indexが負数の場合にErrIndexOutOfRangeを返す

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
	if !contains(jsonStr, `"next_index":null`) {
		t.Errorf("expected next_index:null in JSON, got %s", jsonStr)
	}
	if !contains(jsonStr, `"note":null`) {
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
