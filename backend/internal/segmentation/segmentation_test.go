package segmentation

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"taotale/backend/internal/sourcetext"
)

type vector struct {
	Name, Language, Text string
	Result
}

func readVectors(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func TestFrozenLegacy(t *testing.T) {
	var data struct{ Segmentation []vector }
	readVectors(t, "../../../testdata/legacy/mandarin-v1.json", &data)
	if len(data.Segmentation) != 8 {
		t.Fatal("missing legacy vectors")
	}
	for _, v := range data.Segmentation {
		t.Run(v.Name, func(t *testing.T) {
			checkVector(t, Key{"zh-CN", "zh-cn-legacy-v1", Version1}, v)
		})
	}
}

func TestPhraseV1(t *testing.T) {
	var data []vector
	readVectors(t, "testdata/phrase-v1.json", &data)
	if len(data) != 10 {
		t.Fatal("missing phrase vectors")
	}
	for _, v := range data {
		t.Run(v.Language+"/"+v.Name, func(t *testing.T) {
			key, err := Default(v.Language)
			if err != nil {
				t.Fatal(err)
			}
			checkVector(t, key, v)
		})
	}
}

func checkVector(t *testing.T, key Key, v vector) {
	t.Helper()
	got, err := SegmentSource(key, v.Text)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, v.Result) {
		t.Fatalf("got %+v, want %+v", got, v.Result)
	}
	spans := append([]Span{}, got.Gaps...)
	for _, segment := range got.Segments {
		spans = append(spans, segment.Span)
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].TextStart < spans[j].TextStart })
	index, err := sourcetext.NewIndex(v.Text)
	if err != nil {
		t.Fatal(err)
	}
	var rebuilt strings.Builder
	cursor := 0
	for _, span := range spans {
		a, aOK := index.AtUTF16(span.TextStart)
		b, bOK := index.AtUTF16(span.TextEnd)
		if !aOK || !bOK || span.TextStart != cursor || b.Byte <= a.Byte || v.Text[a.Byte:b.Byte] != span.Text {
			t.Fatalf("invalid source span %+v", span)
		}
		cursor = span.TextEnd
		rebuilt.WriteString(span.Text)
	}
	if rebuilt.String() != v.Text {
		t.Fatal("spans and gaps did not reconstruct the source")
	}
}

func TestRegistryAndBounds(t *testing.T) {
	if _, err := Default("ja-JP"); !errors.Is(err, ErrUnknownStrategy) {
		t.Fatalf("Japanese default: %v", err)
	}
	for _, invalid := range []Key{
		{"ko-KR", "zh-cn-phrase", Version1},
		{"zh-CN", "zh-cn-phrase", "2.0.0"},
	} {
		if _, err := SegmentSource(invalid, "你好"); !errors.Is(err, ErrUnknownStrategy) {
			t.Fatalf("accepted %+v: %v", invalid, err)
		}
	}
	korean, _ := Default("ko-KR")
	line := strings.Repeat("한", 1498) + "𠀀"
	got, err := SegmentSource(korean, line)
	if err != nil || len(got.Segments) != 1 || got.Segments[0].TextEnd != 1500 {
		t.Fatalf("at limit: %+v, %v", got, err)
	}
	if _, err := SegmentSource(korean, line+"글"); !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("past limit: %v", err)
	}
	if _, err := SegmentSource(korean, "\xff"); err == nil {
		t.Fatal("accepted malformed UTF-8")
	}
}
