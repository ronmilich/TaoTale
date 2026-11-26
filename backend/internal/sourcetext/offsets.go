// Package sourcetext indexes exact UTF-8 source at Unicode scalar boundaries.
package sourcetext

import (
	"cmp"
	"errors"
	"sort"
	"unicode/utf16"
	"unicode/utf8"
)

// Boundary describes one position, including the position just past the last scalar.
// Only UTF16 is used for persisted source offsets.
type Boundary struct {
	Byte, Scalar, UTF16 int
}

type Index struct {
	boundaries []Boundary
}

func NewIndex(source string) (*Index, error) {
	if !utf8.ValidString(source) {
		return nil, errors.New("source contains invalid UTF-8")
	}
	var boundaries []Boundary
	var next Boundary
	for _, r := range source {
		boundaries = append(boundaries, next)
		next.Byte += utf8.RuneLen(r)
		next.UTF16 += utf16.RuneLen(r)
		next.Scalar++
	}
	return &Index{append(boundaries, next)}, nil
}

func (idx *Index) AtScalar(offset int) (Boundary, bool) {
	if offset < 0 || offset >= len(idx.boundaries) {
		return Boundary{}, false
	}
	return idx.boundaries[offset], true
}

func (idx *Index) AtByte(offset int) (Boundary, bool) {
	i, ok := sort.Find(len(idx.boundaries), func(i int) int { return cmp.Compare(offset, idx.boundaries[i].Byte) })
	if !ok {
		return Boundary{}, false
	}
	return idx.boundaries[i], true
}

func (idx *Index) AtUTF16(offset int) (Boundary, bool) {
	i, ok := sort.Find(len(idx.boundaries), func(i int) int { return cmp.Compare(offset, idx.boundaries[i].UTF16) })
	if !ok {
		return Boundary{}, false
	}
	return idx.boundaries[i], true
}
