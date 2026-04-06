package syntax_test

import (
	"testing"

	"github.com/golangsnmp/gomib/syntax"
)

func TestLineTable_LineCol(t *testing.T) {
	lt := syntax.BuildLineTable([]byte("abc\ndef\nghi"))
	tests := []struct {
		offset   syntax.ByteOffset
		wantLine int
		wantCol  int
	}{
		{0, 1, 1},
		{2, 1, 3},
		{4, 2, 1},
		{8, 3, 1},
		{10, 3, 3},
	}
	for _, tt := range tests {
		line, col := lt.LineCol(tt.offset)
		if line != tt.wantLine || col != tt.wantCol {
			t.Errorf("LineCol(%d) = (%d, %d), want (%d, %d)",
				tt.offset, line, col, tt.wantLine, tt.wantCol)
		}
	}
}

func TestLineTable_Offset(t *testing.T) {
	lt := syntax.BuildLineTable([]byte("abc\ndef\nghi"))
	tests := []struct {
		line   int
		col    int
		want   syntax.ByteOffset
		wantOK bool
	}{
		{1, 1, 0, true},
		{1, 3, 2, true},
		{2, 1, 4, true},
		{3, 1, 8, true},
		{3, 3, 10, true},
		{0, 1, 0, false},
		{1, 0, 0, false},
		{4, 1, 0, false},
	}
	for _, tt := range tests {
		got, ok := lt.Offset(tt.line, tt.col)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("Offset(%d, %d) = (%d, %v), want (%d, %v)",
				tt.line, tt.col, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestLineTable_RoundTrip(t *testing.T) {
	source := []byte("first line\nsecond line\nthird")
	lt := syntax.BuildLineTable(source)
	offsets := []syntax.ByteOffset{0, 5, 11, 15, 22, 25}
	for _, off := range offsets {
		line, col := lt.LineCol(off)
		got, ok := lt.Offset(line, col)
		if !ok || got != off {
			t.Errorf("round-trip %d -> (%d,%d) -> (%d, %v)",
				off, line, col, got, ok)
		}
	}
}

func TestLineTable_EmptySource(t *testing.T) {
	lt := syntax.BuildLineTable([]byte{})
	line, col := lt.LineCol(0)
	if line != 1 || col != 1 {
		t.Errorf("LineCol(0) on empty source = (%d, %d), want (1, 1)", line, col)
	}
	off, ok := lt.Offset(1, 1)
	if !ok || off != 0 {
		t.Errorf("Offset(1,1) on empty source = (%d, %v), want (0, true)", off, ok)
	}
}
