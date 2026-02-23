package types

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestBuildLineTable_Empty(t *testing.T) {
	table := BuildLineTable(nil)
	testutil.SliceEqual(t, []int{0}, table, "empty input")
}

func TestBuildLineTable_SingleLine(t *testing.T) {
	table := BuildLineTable([]byte("hello"))
	testutil.SliceEqual(t, []int{0}, table, "single line")
}

func TestBuildLineTable_MultipleLines(t *testing.T) {
	// "ab\ncd\nef"
	// Line 1 starts at 0, line 2 at 3, line 3 at 6
	table := BuildLineTable([]byte("ab\ncd\nef"))
	testutil.SliceEqual(t, []int{0, 3, 6}, table, "multiple lines")
}

func TestBuildLineTable_TrailingNewline(t *testing.T) {
	// "abc\n" has 2 lines: line 1 at 0, line 2 at 4 (empty last line)
	table := BuildLineTable([]byte("abc\n"))
	testutil.SliceEqual(t, []int{0, 4}, table, "trailing newline")
}

func TestBuildLineTable_ConsecutiveNewlines(t *testing.T) {
	// "\n\n" has 3 lines: line 1 at 0, line 2 at 1, line 3 at 2
	table := BuildLineTable([]byte("\n\n"))
	testutil.SliceEqual(t, []int{0, 1, 2}, table, "consecutive newlines")
}

func TestLineColFromTable_EmptyTable(t *testing.T) {
	line, col := LineColFromTable(nil, 0)
	testutil.Equal(t, 0, line, "empty table line")
	testutil.Equal(t, 0, col, "empty table col")
}

func TestLineColFromTable_SingleLine(t *testing.T) {
	// "hello" - single line, table = [0]
	table := BuildLineTable([]byte("hello"))

	tests := []struct {
		offset   ByteOffset
		wantLine int
		wantCol  int
	}{
		{0, 1, 1}, // start of line 1
		{2, 1, 3}, // middle of line 1
		{4, 1, 5}, // end of line 1
	}
	for _, tt := range tests {
		line, col := LineColFromTable(table, tt.offset)
		testutil.Equal(t, tt.wantLine, line, "offset %d line", tt.offset)
		testutil.Equal(t, tt.wantCol, col, "offset %d col", tt.offset)
	}
}

func TestLineColFromTable_MultipleLines(t *testing.T) {
	// "ab\ncd\nef"
	// Line 1: offset 0-2 ("ab\n")
	// Line 2: offset 3-5 ("cd\n")
	// Line 3: offset 6-7 ("ef")
	table := BuildLineTable([]byte("ab\ncd\nef"))

	tests := []struct {
		offset   ByteOffset
		wantLine int
		wantCol  int
	}{
		{0, 1, 1}, // start of line 1
		{1, 1, 2}, // 'b' on line 1
		{2, 1, 3}, // '\n' on line 1
		{3, 2, 1}, // start of line 2
		{4, 2, 2}, // 'd' on line 2
		{5, 2, 3}, // '\n' on line 2
		{6, 3, 1}, // start of line 3
		{7, 3, 2}, // 'f' on line 3
	}
	for _, tt := range tests {
		line, col := LineColFromTable(table, tt.offset)
		testutil.Equal(t, tt.wantLine, line, "offset %d line", tt.offset)
		testutil.Equal(t, tt.wantCol, col, "offset %d col", tt.offset)
	}
}

func TestLineColFromTable_PositionAtNewline(t *testing.T) {
	// "\n" - just a newline: table = [0, 1]
	table := BuildLineTable([]byte("\n"))

	line, col := LineColFromTable(table, 0) // the newline character itself
	testutil.Equal(t, 1, line, "newline char line")
	testutil.Equal(t, 1, col, "newline char col")

	line, col = LineColFromTable(table, 1) // start of line 2
	testutil.Equal(t, 2, line, "after newline line")
	testutil.Equal(t, 1, col, "after newline col")
}
