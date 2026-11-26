// Package segmentation chooses source runs without assigning audio timing.
package segmentation

import (
	"errors"
	"strings"
	"unicode"

	"taotale/backend/internal/sourcetext"
)

const Version1 = "1.0.0"

type Key struct {
	LanguageCode, StrategyID, Version string
}

var (
	ErrUnknownStrategy = errors.New("unknown segmentation strategy")
	ErrLineTooLong     = errors.New("Korean source line exceeds 1500 UTF-16 units")
)

func Default(languageCode string) (Key, error) {
	switch languageCode {
	case "zh-CN":
		return Key{languageCode, "zh-cn-phrase", Version1}, nil
	case "ko-KR":
		return Key{languageCode, "ko-kr-phrase", Version1}, nil
	default:
		return Key{}, ErrUnknownStrategy
	}
}

type Span struct {
	TextStart int
	TextEnd   int
	Text      string
}

type Segment struct {
	Index int
	Span
}

type Result struct {
	Segments []Segment
	Gaps     []Span
}

type run struct{ start, end int }
type strategy func(line []rune) []run

var registry = map[Key]strategy{
	{"zh-CN", "zh-cn-legacy-v1", Version1}: legacyMandarin,
	{"zh-CN", "zh-cn-phrase", Version1}:    mandarin,
	{"ko-KR", "ko-kr-phrase", Version1}:    korean,
}

func SegmentSource(key Key, source string) (Result, error) {
	segmentLine, ok := registry[key]
	if !ok {
		return Result{}, ErrUnknownStrategy
	}
	index, err := sourcetext.NewIndex(source)
	if err != nil {
		return Result{}, err
	}
	runes := []rune(source)
	result := Result{Segments: []Segment{}, Gaps: []Span{}}
	span := func(start, end int) Span {
		a, _ := index.AtScalar(start)
		b, _ := index.AtScalar(end)
		return Span{a.UTF16, b.UTF16, source[a.Byte:b.Byte]}
	}
	cursor := 0
	for start := 0; start < len(runes); {
		end := start
		for end < len(runes) && runes[end] != '\r' && runes[end] != '\n' {
			end++
		}
		if key.LanguageCode == "ko-KR" {
			a, _ := index.AtScalar(start)
			b, _ := index.AtScalar(end)
			if b.UTF16-a.UTF16 > 1500 {
				return Result{}, ErrLineTooLong
			}
		}
		for _, part := range segmentLine(runes[start:end]) {
			left, right := start+part.start, start+part.end
			if cursor < left {
				result.Gaps = append(result.Gaps, span(cursor, left))
			}
			result.Segments = append(result.Segments, Segment{len(result.Segments), span(left, right)})
			cursor = right
		}
		start = end + 1
	}
	if cursor < len(runes) {
		result.Gaps = append(result.Gaps, span(cursor, len(runes)))
	}
	return result, nil
}

// Use ECMAScript whitespace for identical legacy trimming and Han adjacency.
func whitespace(r rune) bool {
	return unicode.Is(unicode.Zs, r) || strings.ContainsRune("\t\n\v\f\r\u2028\u2029\ufeff", r)
}

func appendRun(runs []run, line []rune, start, end int) []run {
	for start < end && whitespace(line[start]) {
		start++
	}
	for end > start && whitespace(line[end-1]) {
		end--
	}
	for _, r := range line[start:end] {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return append(runs, run{start, end})
		}
	}
	return runs
}
