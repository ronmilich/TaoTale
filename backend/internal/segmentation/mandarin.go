package segmentation

import (
	"strings"
	"unicode"
)

const mandarinClosers = "”’」』\"'"

func legacyMandarin(line []rune) []run {
	return mandarinRuns(line, func(i int) bool {
		switch line[i] {
		case '，', '。':
			return true
		case '-':
			left, right := i-1, i+1
			for left >= 0 && whitespace(line[left]) {
				left--
			}
			for right < len(line) && whitespace(line[right]) {
				right++
			}
			return left >= 0 && right < len(line) &&
				unicode.Is(unicode.Han, line[left]) && unicode.Is(unicode.Han, line[right])
		default:
			return false
		}
	})
}

func mandarin(line []rune) []run {
	return mandarinRuns(line, func(i int) bool { return strings.ContainsRune("，。！？；", line[i]) })
}

func mandarinRuns(line []rune, delimiter func(int) bool) []run {
	var runs []run
	start := 0
	for i := 0; i < len(line); i++ {
		if !delimiter(i) {
			continue
		}
		end := i + 1
		for end < len(line) && (delimiter(end) || strings.ContainsRune(mandarinClosers, line[end])) {
			end++
		}
		runs = appendRun(runs, line, start, end)
		start, i = end, end-1
	}
	return appendRun(runs, line, start, len(line))
}
