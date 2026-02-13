// Package types provides internal types shared across gomib packages.
package types

import (
	"context"
	"log/slog"

	"github.com/golangsnmp/gomib/mib"
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

// Diagnostic is an internal diagnostic from the lexer or parser.
// Converted to mib.Diagnostic during lowering with module name and
// line/column info.
type Diagnostic struct {
	Severity mib.Severity
	Code     string // Diagnostic code (e.g., "identifier-underscore")
	Span     Span
	Message  string
}
