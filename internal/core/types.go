package core

// NoteType はシミュレーション中のノートの種類を表す。
type NoteType string

const (
	NoteTypeQuestion  NoteType = "QUESTION"
	NoteTypeResolved  NoteType = "RESOLVED"
	NoteTypeConfusion NoteType = "CONFUSION"
)

// Document は入力テキストとその文分割結果を保持する。
type Document struct {
	ID        string     `json:"id"`
	RawText   string     `json:"raw_text"`
	Sentences []Sentence `json:"sentences"`
}

// Sentence はテキスト中の1文を表す。
type Sentence struct {
	Index   int    `json:"index"`
	Content string `json:"content"`
}

// SimulationStep はシミュレーションの1ステップの結果を表す。
type SimulationStep struct {
	Step        int    `json:"step"`
	SentenceIdx int    `json:"current_index"`
	TargetIdx   *int   `json:"next_index"`
	Note        *Note  `json:"note"`
	Memory      string `json:"memory"`
}

// Note はシミュレーション中の読者の感想を表す。
type Note struct {
	Type    NoteType `json:"type"`
	Content string   `json:"content"`
}
