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

// ctx is a package-level context for logging.
var ctx = context.Background()

// Logger wraps slog.Logger with nil-safe helpers.
type Logger struct {
	L *slog.Logger
}

// Enabled returns true if logging is enabled at the given level.
func (l *Logger) Enabled(level slog.Level) bool {
	return l.L != nil && l.L.Enabled(ctx, level)
}

// Log emits a log message if logging is enabled.
func (l *Logger) Log(level slog.Level, msg string, attrs ...slog.Attr) {
	if l.L != nil && l.L.Enabled(ctx, level) {
		l.L.LogAttrs(ctx, level, msg, attrs...)
	}
}

// TraceEnabled returns true if trace-level logging is enabled.
func (l *Logger) TraceEnabled() bool {
	return l.Enabled(LevelTrace)
}

// Trace emits a trace-level log.
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

// NewSpan creates a new span.
func NewSpan(start, end ByteOffset) Span {
	return Span{Start: start, End: end}
}

// Len returns the length of the span in bytes.
func (s Span) Len() ByteOffset {
	return s.End - s.Start
}

// IsEmpty returns true if the span is empty.
func (s Span) IsEmpty() bool {
	return s.Start == s.End
}

// IsSynthetic returns true if this is a synthetic span.
func (s Span) IsSynthetic() bool {
	return s.Start == 0 && s.End == 0
}

// Severity is a diagnostic severity level (internal use during parsing).
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

// Diagnostic is a message from the lexer or parser (internal use).
type Diagnostic struct {
	Severity Severity
	Span     Span
	Message  string
}
