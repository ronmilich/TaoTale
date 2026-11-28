package vocabmatch

import (
	"slices"
	"testing"
)

func TestMandarinAnnotations(t *testing.T) {
	for _, test := range []struct {
		name, source string
		words        []Word
		want         []Annotation
	}{
		{
			name:   "longer match wins even when it starts later",
			source: "𠀀甲乙丙丁\r\n 𠀀甲乙丙丁",
			words:  []Word{{"early", "𠀀甲"}, {"long", "甲乙丙丁"}, {"short", "乙丙"}, {"rare", "𠀀"}},
			want:   []Annotation{{"rare", 0, 2}, {"long", 2, 6}, {"rare", 9, 11}, {"long", 11, 15}},
		},
		{
			name:   "earliest and stable ID break ties, with overlapping repetitions",
			source: "哈哈哈哈",
			words:  []Word{{"z", "哈哈哈"}, {"a", "哈哈哈"}, {"last", "哈"}},
			want:   []Annotation{{"a", 0, 3}, {"last", 3, 4}},
		},
		{
			name:   "case, Unicode composition and whitespace are literal",
			source: "ABC abc e\u0301 é 甲  乙",
			words:  []Word{{"case", "abc"}, {"normalization", "é"}, {"spacing", "甲 乙"}, {"exact", "甲  乙"}, {"empty", ""}},
			want:   []Annotation{{"case", 4, 7}, {"normalization", 11, 12}, {"exact", 13, 17}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Mandarin(test.source, test.words)
			if !slices.Equal(got, test.want) {
				t.Fatalf("got %+v, want %+v", got, test.want)
			}
			slices.Reverse(test.words)
			if reversed := Mandarin(test.source, test.words); !slices.Equal(reversed, got) {
				t.Fatalf("dictionary order changed matches: %+v", reversed)
			}
		})
	}
	if !Enabled("zh-CN") || Enabled("ko-KR") || Enabled("ja-JP") {
		t.Fatal("only Mandarin has automatic matching")
	}
}
