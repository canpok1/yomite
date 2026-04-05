package core

import (
	"strings"
)

// SplitSentences は Document.RawText を文分割ルールに従って []Sentence に分割する。
//
// 文分割ルール:
//  1. 全角区切り文字（。！？」）の直後で分割。区切り文字は直前の文に含める。
//  2. 半角区切り文字（. ! ?）+ 半角スペースの直後で分割。区切り文字は直前の文に含め、半角スペースは除去。
//  3. 開き引用符（「）の直前で分割。「は新しい文の先頭になる。
//  4. 連続改行（空行）は文の区切りとして扱う。
//  5. 空白のみ・空文字の文はフィルタリングして除外する。
func (d *Document) SplitSentences() []Sentence {
	if d.RawText == "" {
		return []Sentence{}
	}

	// まず段落区切り（空行）で分割
	paragraphs := splitByEmptyLines(d.RawText)

	var rawParts []string
	for _, para := range paragraphs {
		parts := splitParagraph(para)
		rawParts = append(rawParts, parts...)
	}

	// 空文除外してSentenceスライスを構築
	// NOTE: nilではなく空スライスを返すことで、JSON出力が"null"ではなく"[]"になることを保証する。
	sentences := make([]Sentence, 0)
	for _, part := range rawParts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		sentences = append(sentences, Sentence{
			Index:   len(sentences),
			Content: trimmed,
		})
	}

	return sentences
}

// splitByEmptyLines は連続改行（空行）でテキストを分割する。
func splitByEmptyLines(text string) []string {
	var result []string
	var current strings.Builder

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
			continue
		}
		if current.Len() > 0 && i > 0 && strings.TrimSpace(lines[i-1]) != "" {
			current.WriteString("\n")
		}
		current.WriteString(line)
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// splitParagraph は段落内のテキストを文分割ルール1〜3に従って分割する。
func splitParagraph(text string) []string {
	runes := []rune(text)
	var parts []string
	var current []rune

	i := 0
	for i < len(runes) {
		ch := runes[i]

		// ルール3: 開き引用符「の直前で分割
		if ch == '「' && len(current) > 0 {
			parts = append(parts, string(current))
			current = nil
		}

		current = append(current, ch)

		// ルール1: 全角区切り文字（。！？」）の直後で分割
		// ただし。！？の直後に」が続く場合は」も同じ文に含める
		if ch == '。' || ch == '！' || ch == '？' {
			if i+1 < len(runes) && runes[i+1] == '」' {
				current = append(current, '」')
				i++
			}
			parts = append(parts, string(current))
			current = nil
			i++
			continue
		}
		if ch == '」' {
			parts = append(parts, string(current))
			current = nil
			i++
			continue
		}

		// ルール2: 半角区切り文字 + スペースの直後で分割
		if (ch == '.' || ch == '!' || ch == '?') && i+1 < len(runes) && runes[i+1] == ' ' {
			parts = append(parts, string(current))
			current = nil
			i += 2 // スペースをスキップ
			continue
		}

		i++
	}

	if len(current) > 0 {
		parts = append(parts, string(current))
	}

	return parts
}
