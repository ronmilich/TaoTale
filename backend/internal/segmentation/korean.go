package segmentation

import (
	"strings"
	"unicode"
)

const koreanClosers = "”’」』»\"')]）］】〉》}"
const koreanEndings = ".?!。？！．"

func korean(line []rune) []run {
	var runs []run
	start := 0
	nested := koreanParentheticals(line)
	for i := 0; i < len(line); i++ {
		if !strings.ContainsRune(koreanEndings, line[i]) || nested[i] || protectedPeriod(line, i) {
			continue
		}
		end := i + 1
		for end < len(line) && strings.ContainsRune(koreanEndings+"…"+koreanClosers, line[end]) {
			end++
		}
		runs = appendRun(runs, line, start, end)
		start, i = end, end-1
	}
	return appendRun(runs, line, start, len(line))
}

func protectedPeriod(line []rune, i int) bool {
	if line[i] != '.' && line[i] != '．' {
		return false
	}
	if i > 0 && i+1 < len(line) && unicode.IsDigit(line[i-1]) && unicode.IsDigit(line[i+1]) {
		return true
	}
	// A repeated period or ellipsis alone does not start a new sentence.
	if (i > 0 && line[i-1] == line[i]) || (i+1 < len(line) && line[i+1] == line[i]) {
		return true
	}
	if line[i] != '.' {
		return false
	}
	left, right := i, i+1
	for left > 0 && latinAbbreviationRune(line[left-1]) {
		left--
	}
	for right < len(line) && latinAbbreviationRune(line[right]) {
		right++
	}
	switch strings.ToLower(string(line[left:right])) {
	case "mr.", "mrs.", "ms.", "dr.", "prof.", "vs.", "etc.", "e.g.", "i.e.", "a.m.", "p.m.", "u.s.":
		return true
	default:
		return false
	}
}

func latinAbbreviationRune(r rune) bool {
	return r == '.' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// Only successfully matched bracket pairs suppress internal punctuation.
func koreanParentheticals(line []rune) []bool {
	pairs := map[rune]rune{'(': ')', '[': ']', '（': '）', '［': '］', '【': '】', '〈': '〉', '《': '》', '{': '}'}
	type opener struct {
		pos   int
		close rune
	}
	var stack []opener
	delta := make([]int, len(line)+1)
	for i, r := range line {
		if close, ok := pairs[r]; ok {
			stack = append(stack, opener{i, close})
		} else if n := len(stack); n > 0 && r == stack[n-1].close {
			delta[stack[n-1].pos+1]++
			delta[i]--
			stack = stack[:n-1]
		}
	}
	nested := make([]bool, len(line))
	depth := 0
	for i := range line {
		depth += delta[i]
		nested[i] = depth > 0
	}
	return nested
}
