package tts

import (
	"context"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestMock(t *testing.T) {
	m := Mock{}
	for _, lang := range []string{"zh-CN", "ko-KR"} {
		catalog, err := m.ListVoices(t.Context(), lang)
		if err != nil || len(catalog) != 1 || catalog[0].Language != lang ||
			!strings.HasSuffix(catalog[0].DisplayName, " (mock)") || !slices.Equal(catalog[0].Rates, []float64{0.75, 1, 1.25}) {
			t.Fatalf("catalog %v, %v", catalog, err)
		}
	}
	request := Request{"ko-KR", "ko-KR-mock", "😀 한글 & Latin", 1}
	a, err := m.Synthesize(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	b, err := m.Synthesize(t.Context(), request)
	if err != nil || !reflect.DeepEqual(a, b) || len(a.MP3) == 0 || a.BoundaryText != request.Source {
		t.Fatal("mock output is missing or nondeterministic")
	}
	if len(a.Boundaries) != 2 || a.Boundaries[0].Text != "한글" || a.Boundaries[0].TextOffset != 2 ||
		a.Boundaries[1].Text != "Latin" || a.Boundaries[1].AudioTicks != 5_000_000 {
		t.Fatalf("unexpected boundaries: %+v", a.Boundaries)
	}
}

func TestProviderErrors(t *testing.T) {
	m := Mock{}
	for _, request := range []Request{
		{"ja-JP", "ja-JP-mock", "甲", 1},
		{"ko-KR", "zh-CN-mock", "한", 1},
		{"ko-KR", "ko-KR-mock", "한", 1.2},
		{"ko-KR", "ko-KR-mock", "한", math.NaN()},
		{"ko-KR", "ko-KR-mock", "한\x00", 1},
		{"ko-KR", "ko-KR-mock", "\xff", 1},
		{"ko-KR", "ko-KR-mock", "😀", 1},
		{"ko-KR", "ko-KR-mock", "한" + strings.Repeat("𠀀", 750), 1},
	} {
		if _, err := m.Synthesize(t.Context(), request); CodeOf(err) != Invalid {
			t.Fatalf("expected validation error, got %v", err)
		}
	}
	azure := NewAzure("", "", "", 0)
	request := Request{"ko-KR", "ko-KR-SunHiNeural", "한", 1}
	if _, err := azure.Synthesize(t.Context(), request); CodeOf(err) != Unavailable {
		t.Fatalf("missing credentials: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := azure.Synthesize(ctx, request); CodeOf(err) != Canceled {
		t.Fatalf("canceled: %v", err)
	}
	ctx, cancel = context.WithTimeout(t.Context(), 0)
	defer cancel()
	if _, err := azure.Synthesize(ctx, request); CodeOf(err) != Timeout {
		t.Fatalf("deadline: %v", err)
	}
}

func TestAzureSSML(t *testing.T) {
	doc := ssml(Request{"zh-CN", "zh-CN-XiaoxiaoNeural", "\"𠀀\" & 3 < 5\r\n", .75})
	if !strings.Contains(doc, `xml:lang="zh-CN"><voice name="zh-CN-XiaoxiaoNeural"><prosody rate="0.75">`) ||
		!strings.Contains(doc, "&quot;𠀀&quot; &amp; 3 &lt; 5&#13;\n</prosody>") {
		t.Fatal(doc)
	}
}
