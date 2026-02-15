// Package types provides internal types shared across gomib packages.
package types

import (
	"context"
	"log/slog"
)

// LevelTrace is a custom log level more verbose than Debug.
// Use for per-item iteration logging (tokens, OID nodes, imports).
// Enable with: &slog.HandlerOptions{Level: slog.Level(-8)}
const LevelTrace = slog.Level(-8)

// noCtx is a background context used for slog calls that don't need cancellation.
var noCtx = context.Background() //nolint:gochecknoglobals

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

// Synthetic is a span for compiler-generated constructs.
var Synthetic = Span{Start: 0, End: 0}

// NewSpan creates a Span from start and end byte offsets.
func NewSpan(start, end ByteOffset) Span {
	return Span{Start: start, End: end}
}

// SpanDiagnostic is an internal diagnostic from the lexer or parser.
// Converted to Diagnostic during lowering with module name and
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
// using a precomputed line table. Returns (0, 0) if the table is nil or the
// offset cannot be resolved.
func LineColFromTable(table []int, offset ByteOffset) (line, col int) {
	if len(table) == 0 || offset == 0 {
		return 0, 0
	}
	off := int(offset)
	// Binary search for the last line start <= offset.
	lo, hi := 0, len(table)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if table[mid] <= off {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1, off - table[lo] + 1
}
