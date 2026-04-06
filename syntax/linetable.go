package syntax

import "github.com/golangsnmp/gomib/internal/types"

// LineTable provides bidirectional conversion between byte offsets and
// 1-based line/column numbers.
type LineTable struct {
	table []int
}

// BuildLineTable scans source bytes and returns a LineTable for
// bidirectional offset/position conversion.
func BuildLineTable(source []byte) LineTable {
	return LineTable{table: types.BuildLineTable(source)}
}

// LineCol converts a byte offset to 1-based line and column numbers.
// Returns (0, 0) for synthetic offsets or empty tables.
func (lt LineTable) LineCol(offset ByteOffset) (line, col int) {
	return types.LineColFromTable(lt.table, offset)
}

// Offset converts 1-based line and column numbers to a byte offset.
// Returns (0, false) for out-of-range positions or empty tables.
func (lt LineTable) Offset(line, col int) (ByteOffset, bool) {
	return types.OffsetFromTable(lt.table, line, col)
}
