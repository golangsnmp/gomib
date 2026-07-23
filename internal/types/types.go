// Package types provides internal types shared across gomib packages.
package types

import (
	"context"
	"log/slog"
	"math"
	"sort"
)

// LevelTrace is a custom log level more verbose than Debug.
// Use for per-item iteration logging (tokens, OID nodes, imports).
// Enable with: &slog.HandlerOptions{Level: slog.Level(-8)}
const LevelTrace = slog.Level(-8)

// noCtx is a background context used for slog calls that don't need cancellation.
var noCtx = context.Background() //nolint:gochecknoglobals // package-level context for internal use

// Logger wraps slog.Logger with nil-safe convenience methods.
type Logger struct {
	L *slog.Logger
}

// Enabled reports whether logging is active at the given level.
func (l *Logger) Enabled(level slog.Level) bool {
	return l.L != nil && l.L.Enabled(noCtx, level)
}

// Log emits a structured log message at the given level. No-op if nil.
func (l *Logger) Log(level slog.Level, msg string, attrs ...slog.Attr) {
	if l.L != nil && l.L.Enabled(noCtx, level) {
		l.L.LogAttrs(noCtx, level, msg, attrs...)
	}
}

// TraceEnabled reports whether trace-level logging is active.
func (l *Logger) TraceEnabled() bool {
	return l.Enabled(LevelTrace)
}

// Trace emits a log message at the custom trace level.
func (l *Logger) Trace(msg string, attrs ...slog.Attr) {
	l.Log(LevelTrace, msg, attrs...)
}

// ByteOffset is a byte position in source text.
type ByteOffset uint32

// Span represents a range in source text.
type Span struct {
	Start ByteOffset // inclusive
	End   ByteOffset // exclusive
}

// Synthetic is a span for compiler-generated constructs (base modules,
// built-in types). Uses MaxUint32 to distinguish from uninitialized zero-value spans.
//
//nolint:gochecknoglobals // package-level sentinel
var Synthetic = Span{Start: ByteOffset(math.MaxUint32), End: ByteOffset(math.MaxUint32)}

// NewSpan creates a Span from start and end byte offsets.
func NewSpan(start, end ByteOffset) Span {
	return Span{Start: start, End: end}
}

// Contains reports whether offset falls within this span [Start, End).
// Returns false for zero-value spans and synthetic spans.
func (s Span) Contains(offset ByteOffset) bool {
	if s.Start == 0 && s.End == 0 {
		return false
	}
	if s == Synthetic {
		return false
	}
	return offset >= s.Start && offset < s.End
}

// SpanDiagnostic is an internal diagnostic from the lexer or parser.
// Converted to Diagnostic during parsing with module name and
// line/column info.
type SpanDiagnostic struct {
	Severity Severity
	Code     string // Diagnostic code (e.g., "identifier-underscore")
	Span     Span
	Message  string
}

// BuildLineTable scans source bytes and returns a table mapping line numbers
// to byte offsets. Entry i is the byte offset where line i+1 starts.
// Line 1 always starts at offset 0.
func BuildLineTable(source []byte) []int {
	// Pre-count newlines for a single allocation.
	n := 1
	for _, b := range source {
		if b == '\n' {
			n++
		}
	}
	table := make([]int, 0, n)
	table = append(table, 0) // line 1 starts at offset 0
	for i, b := range source {
		if b == '\n' {
			table = append(table, i+1)
		}
	}
	return table
}

// LineColFromTable converts a byte offset to 1-based line and column numbers
// using a precomputed line table. Returns (0, 0) if the table is empty or
// the offset is from a synthetic span.
func LineColFromTable(table []int, offset ByteOffset) (line, col int) {
	if len(table) == 0 || offset == ByteOffset(math.MaxUint32) {
		return 0, 0
	}
	off := int(offset)
	// Find the last line start <= offset.
	lo := sort.Search(len(table), func(i int) bool { return table[i] > off }) - 1
	return lo + 1, off - table[lo] + 1
}

// LineColRangeFromTable converts a span to 1-based start and end positions.
// The start is inclusive and the end is exclusive. A zero-length, reversed,
// synthetic, or otherwise unavailable end is returned as (0, 0).
func LineColRangeFromTable(table []int, span Span) (line, col, endLine, endCol int) {
	line, col = LineColFromTable(table, span.Start)
	if span.End <= span.Start {
		return line, col, 0, 0
	}
	endLine, endCol = LineColFromTable(table, span.End)
	return line, col, endLine, endCol
}

// OffsetFromTable converts 1-based line and column numbers to a byte offset
// using a precomputed line table. Returns (0, false) if the table is empty
// or the line/col is out of range.
func OffsetFromTable(table []int, line, col int) (ByteOffset, bool) {
	if len(table) == 0 || line < 1 || line > len(table) || col < 1 {
		return 0, false
	}
	return ByteOffset(table[line-1] + col - 1), true
}
