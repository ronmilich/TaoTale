// Package vocabmatch defines vocabulary matching independently of speech segmentation.
package vocabmatch

import (
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"

	"taotale/backend/internal/sourcetext"
)

func Enabled(language string) bool { return language == "zh-CN" }

type Word struct {
	ID   string
	Text string
}

type Annotation struct {
	VocabularyID string `json:"vocabularyId"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
}

// Mandarin finds exact display spellings, including all overlapping occurrences.
// Longer UTF-16 ranges win first, followed by earlier starts and then stable IDs.
// The accepted ranges are returned in source order.
func Mandarin(source string, words []Word) []Annotation {
	result := []Annotation{}
	index, err := sourcetext.NewIndex(source)
	if err != nil {
		return result
	}
	var candidates []Annotation
	for _, word := range words {
		if word.Text == "" || !utf8.ValidString(word.Text) {
			continue
		}
		for offset := 0; offset < len(source); {
			match := strings.Index(source[offset:], word.Text)
			if match < 0 {
				break
			}
			start := offset + match
			left, _ := index.AtByte(start)
			right, _ := index.AtByte(start + len(word.Text))
			// Both strings are valid UTF-8, so a byte-exact match has scalar boundaries.
			candidates = append(candidates, Annotation{word.ID, left.UTF16, right.UTF16})
			offset = start + 1
		}
	}
	slices.SortFunc(candidates, func(a, b Annotation) int {
		if byLength := cmp.Compare(b.End-b.Start, a.End-a.Start); byLength != 0 {
			return byLength
		}
		if byStart := cmp.Compare(a.Start, b.Start); byStart != 0 {
			return byStart
		}
		return cmp.Compare(a.VocabularyID, b.VocabularyID)
	})
	end, _ := index.AtByte(len(source))
	claimed := make([]bool, end.UTF16)
	for _, match := range candidates {
		if slices.Contains(claimed[match.Start:match.End], true) {
			continue
		}
		for i := match.Start; i < match.End; i++ {
			claimed[i] = true
		}
		result = append(result, match)
	}
	slices.SortFunc(result, func(a, b Annotation) int { return cmp.Compare(a.Start, b.Start) })
	return result
}
