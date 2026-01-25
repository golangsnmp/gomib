// Package types provides shared types used across the gomib packages.
package types

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
)

// LevelTrace is a custom log level more verbose than Debug.
// Use for per-item iteration logging (tokens, OID nodes, imports).
// Users enable with: &slog.HandlerOptions{Level: slog.Level(-8)}
const LevelTrace = slog.Level(-8)

// ctx is a package-level context for logging (avoids repeated allocations).
var ctx = context.Background()

// Logger wraps slog.Logger with nil-safe helpers.
// Embed this in components that need logging.
type Logger struct {
	L *slog.Logger
}

// Enabled returns true if logging is enabled at the given level.
// Use this to guard expensive attribute computation.
func (l *Logger) Enabled(level slog.Level) bool {
	return l.L != nil && l.L.Enabled(ctx, level)
}

// Log emits a log message if logging is enabled.
// WARNING: Arguments are evaluated even if logging is disabled.
// Only use with cheap attributes, or guard with Enabled().
func (l *Logger) Log(level slog.Level, msg string, attrs ...slog.Attr) {
	if l.L != nil && l.L.Enabled(ctx, level) {
		l.L.LogAttrs(ctx, level, msg, attrs...)
	}
}

// TraceEnabled returns true if trace-level logging is enabled.
func (l *Logger) TraceEnabled() bool {
	return l.Enabled(LevelTrace)
}

// Trace emits a trace-level log. Only use with cheap attributes.
func (l *Logger) Trace(msg string, attrs ...slog.Attr) {
	l.Log(LevelTrace, msg, attrs...)
}

// ByteOffset is a byte position in source text.
// Uses uint32 to match Rust implementation (sources limited to ~4GB).
type ByteOffset uint32

// Span represents a range in source text.
type Span struct {
	Start ByteOffset // inclusive
	End   ByteOffset // exclusive
}

// Synthetic is a span for compiler-generated constructs that don't come from source.
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

// Severity is a diagnostic severity level.
type Severity int

const (
	// SeverityError blocks progress; the input may be malformed.
	SeverityError Severity = iota
	// SeverityWarning is informational; parsing continues.
	SeverityWarning
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	default:
		return fmt.Sprintf("Severity(%d)", s)
	}
}

// Diagnostic is a message from the lexer or parser.
type Diagnostic struct {
	Severity Severity
	Span     Span
	Message  string
}

// NewError creates an error diagnostic.
func NewError(span Span, message string) Diagnostic {
	return Diagnostic{Severity: SeverityError, Span: span, Message: message}
}

// NewWarning creates a warning diagnostic.
func NewWarning(span Span, message string) Diagnostic {
	return Diagnostic{Severity: SeverityWarning, Span: span, Message: message}
}

// DiagnosticCollector accumulates diagnostics during parsing/resolution.
type DiagnosticCollector struct {
	diagnostics []Diagnostic
}

// Add adds a diagnostic with the given severity, span, and message.
func (c *DiagnosticCollector) Add(sev Severity, span Span, msg string) {
	c.diagnostics = append(c.diagnostics, Diagnostic{
		Severity: sev,
		Span:     span,
		Message:  msg,
	})
}

// Error adds an error diagnostic.
func (c *DiagnosticCollector) Error(span Span, msg string) {
	c.Add(SeverityError, span, msg)
}

// Warning adds a warning diagnostic.
func (c *DiagnosticCollector) Warning(span Span, msg string) {
	c.Add(SeverityWarning, span, msg)
}

// Diagnostics returns a copy of all collected diagnostics.
// The returned slice is owned by the caller.
func (c *DiagnosticCollector) Diagnostics() []Diagnostic {
	return slices.Clone(c.diagnostics)
}

// HasErrors returns true if any error diagnostics were collected.
func (c *DiagnosticCollector) HasErrors() bool {
	return slices.ContainsFunc(c.diagnostics, func(d Diagnostic) bool {
		return d.Severity == SeverityError
	})
}

// Status represents the status of a MIB definition.
type Status int

const (
	StatusCurrent Status = iota
	StatusDeprecated
	StatusObsolete
)

func (s Status) String() string {
	switch s {
	case StatusCurrent:
		return "current"
	case StatusDeprecated:
		return "deprecated"
	case StatusObsolete:
		return "obsolete"
	default:
		return fmt.Sprintf("Status(%d)", s)
	}
}

// Access represents the access level of a MIB object.
type Access int

const (
	AccessNotAccessible Access = iota
	AccessAccessibleForNotify
	AccessReadOnly
	AccessReadWrite
	AccessReadCreate
	AccessWriteOnly
)

func (a Access) String() string {
	switch a {
	case AccessNotAccessible:
		return "not-accessible"
	case AccessAccessibleForNotify:
		return "accessible-for-notify"
	case AccessReadOnly:
		return "read-only"
	case AccessReadWrite:
		return "read-write"
	case AccessReadCreate:
		return "read-create"
	case AccessWriteOnly:
		return "write-only"
	default:
		return fmt.Sprintf("Access(%d)", a)
	}
}

// SmiLanguage represents the SMI language version.
//
// Detected from imports during lowering:
//   - SMIv2 if imports from SNMPv2-SMI, SNMPv2-TC, or SNMPv2-CONF
//   - SMIv1 otherwise (default)
type SmiLanguage int

const (
	// SmiLanguageUnknown indicates language not yet determined.
	SmiLanguageUnknown SmiLanguage = iota
	// SmiLanguageSMIv1 is SMIv1 (RFC 1155, 1212, 1215).
	SmiLanguageSMIv1
	// SmiLanguageSMIv2 is SMIv2 (RFC 2578, 2579, 2580).
	SmiLanguageSMIv2
	// SmiLanguageSPPI is SPPI Policy Information Base (RFC 3159).
	SmiLanguageSPPI
)

func (l SmiLanguage) String() string {
	switch l {
	case SmiLanguageUnknown:
		return "Unknown"
	case SmiLanguageSMIv1:
		return "SMIv1"
	case SmiLanguageSMIv2:
		return "SMIv2"
	case SmiLanguageSPPI:
		return "SPPI"
	default:
		return fmt.Sprintf("SmiLanguage(%d)", l)
	}
}

// BaseType represents the base SMI type.
type BaseType int

const (
	BaseInteger32 BaseType = iota
	BaseUnsigned32
	BaseCounter32
	BaseCounter64
	BaseGauge32
	BaseTimeTicks
	BaseIpAddress
	BaseOctetString
	BaseObjectIdentifier
	BaseOpaque
	BaseBits
	BaseSequence
)

func (b BaseType) String() string {
	switch b {
	case BaseInteger32:
		return "Integer32"
	case BaseUnsigned32:
		return "Unsigned32"
	case BaseCounter32:
		return "Counter32"
	case BaseCounter64:
		return "Counter64"
	case BaseGauge32:
		return "Gauge32"
	case BaseTimeTicks:
		return "TimeTicks"
	case BaseIpAddress:
		return "IpAddress"
	case BaseOctetString:
		return "OCTET STRING"
	case BaseObjectIdentifier:
		return "OBJECT IDENTIFIER"
	case BaseOpaque:
		return "Opaque"
	case BaseBits:
		return "BITS"
	case BaseSequence:
		return "SEQUENCE"
	default:
		return fmt.Sprintf("BaseType(%d)", b)
	}
}
