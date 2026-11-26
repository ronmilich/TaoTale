package sourcetext

import "testing"

func TestF3Boundaries(t *testing.T) {
	for _, tc := range []struct {
		source       string
		bytes, units []int
	}{
		{"", []int{0}, []int{0}},
		{"汉𠀀😀e\u0301한，\t\r\nA\r�", []int{0, 3, 7, 11, 12, 14, 17, 20, 21, 22, 23, 24, 25, 28}, []int{0, 1, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}},
	} {
		idx, err := NewIndex(tc.source)
		if err != nil {
			t.Fatal(err)
		}
		for _, unit := range []struct {
			name    string
			offsets []int
			lookup  func(int) (Boundary, bool)
		}{
			{"byte", tc.bytes, idx.AtByte}, {"utf16", tc.units, idx.AtUTF16},
		} {
			want := map[int]Boundary{}
			for i, pos := range unit.offsets {
				b := Boundary{tc.bytes[i], i, tc.units[i]}
				want[pos] = b
				if got, ok := idx.AtScalar(i); !ok || got != b {
					t.Fatalf("scalar %d: %v %v, want %v", i, got, ok, b)
				}
			}
			for i := -1; i <= unit.offsets[len(unit.offsets)-1]+1; i++ {
				got, ok := unit.lookup(i)
				b, valid := want[i]
				if ok != valid || got != b {
					t.Fatalf("%s %d: %v %v, want %v %v", unit.name, i, got, ok, b, valid)
				}
			}
		}
		for _, i := range []int{-1, len(tc.bytes)} {
			if _, ok := idx.AtScalar(i); ok {
				t.Fatalf("accepted scalar %d", i)
			}
		}
	}
}

func TestMalformedUTF8(t *testing.T) {
	for _, source := range []string{"a\x80", "\xf0\x9f", "\xed\xa0\x80", "\xc0\x80"} {
		if _, err := NewIndex(source); err == nil {
			t.Fatalf("accepted malformed source %x", source)
		}
	}
}
