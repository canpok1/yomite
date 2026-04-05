package core

import (
	"reflect"
	"testing"
)

func TestSplitSentences_EmptyString(t *testing.T) {
	t.Parallel()

	doc := Document{RawText: ""}
	got := doc.SplitSentences()

	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestSplitSentences_SingleSentenceWithPeriod(t *testing.T) {
	t.Parallel()

	doc := Document{RawText: "これはテストです。"}
	got := doc.SplitSentences()

	want := []Sentence{{Index: 0, Content: "これはテストです。"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSentences_FullWidthDelimiters(t *testing.T) {
	t.Parallel()

	doc := Document{RawText: "驚いた！本当に？そうだ。"}
	got := doc.SplitSentences()

	want := []Sentence{
		{Index: 0, Content: "驚いた！"},
		{Index: 1, Content: "本当に？"},
		{Index: 2, Content: "そうだ。"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSentences_ClosingQuote(t *testing.T) {
	t.Parallel()

	doc := Document{RawText: "「はい」そうだ。"}
	got := doc.SplitSentences()

	want := []Sentence{
		{Index: 0, Content: "「はい」"},
		{Index: 1, Content: "そうだ。"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSentences_HalfWidthDelimiters(t *testing.T) {
	t.Parallel()

	doc := Document{RawText: "Hello. World! Really? Yes."}
	got := doc.SplitSentences()

	want := []Sentence{
		{Index: 0, Content: "Hello."},
		{Index: 1, Content: "World!"},
		{Index: 2, Content: "Really?"},
		{Index: 3, Content: "Yes."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSentences_HalfWidthPeriodNoSpace(t *testing.T) {
	t.Parallel()

	doc := Document{RawText: "値は3.14です。"}
	got := doc.SplitSentences()

	want := []Sentence{{Index: 0, Content: "値は3.14です。"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSentences_OpeningQuote(t *testing.T) {
	t.Parallel()

	doc := Document{RawText: "彼は「はい」と言った。"}
	got := doc.SplitSentences()

	want := []Sentence{
		{Index: 0, Content: "彼は"},
		{Index: 1, Content: "「はい」"},
		{Index: 2, Content: "と言った。"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSentences_ParagraphBreak(t *testing.T) {
	t.Parallel()

	doc := Document{RawText: "第一段落\n\n第二段落"}
	got := doc.SplitSentences()

	want := []Sentence{
		{Index: 0, Content: "第一段落"},
		{Index: 1, Content: "第二段落"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSentences_EmptySentenceFiltering(t *testing.T) {
	t.Parallel()

	doc := Document{RawText: "はい。\n\n\n\nいいえ。"}
	got := doc.SplitSentences()

	want := []Sentence{
		{Index: 0, Content: "はい。"},
		{Index: 1, Content: "いいえ。"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSentences_SpecExample(t *testing.T) {
	t.Parallel()

	doc := Document{RawText: "彼は「了解。すぐ行く。」と言った。"}
	got := doc.SplitSentences()

	want := []Sentence{
		{Index: 0, Content: "彼は"},
		{Index: 1, Content: "「了解。"},
		{Index: 2, Content: "すぐ行く。」"},
		{Index: 3, Content: "と言った。"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitSentences_NestedQuotes(t *testing.T) {
	t.Parallel()

	doc := Document{RawText: "「彼は「了解」と言った」"}
	got := doc.SplitSentences()

	want := []Sentence{
		{Index: 0, Content: "「彼は"},
		{Index: 1, Content: "「了解」"},
		{Index: 2, Content: "と言った」"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
