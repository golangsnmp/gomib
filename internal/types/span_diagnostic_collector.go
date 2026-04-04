package types

import "slices"

// SpanDiagnosticCollector collects SpanDiagnostic values with config-based
// filtering. Embedded by the lexer and parser to share diagnostic collection
// logic.
type SpanDiagnosticCollector struct {
	diagnostics []SpanDiagnostic
	DiagConfig  DiagnosticConfig
}

// NewSpanDiagnosticCollector returns a collector with the given config.
func NewSpanDiagnosticCollector(config DiagnosticConfig) SpanDiagnosticCollector {
	return SpanDiagnosticCollector{DiagConfig: config}
}

// EmitDiagnostic records a diagnostic if the config reports it.
func (c *SpanDiagnosticCollector) EmitDiagnostic(code string, span Span, message string) {
	sev := SeverityForCode(code)
	if !c.DiagConfig.ShouldReport(code, sev) {
		return
	}
	c.diagnostics = append(c.diagnostics, SpanDiagnostic{
		Severity: sev,
		Code:     code,
		Span:     span,
		Message:  message,
	})
}

// RecordDiagnostic appends a pre-built diagnostic without filtering.
func (c *SpanDiagnosticCollector) RecordDiagnostic(diag SpanDiagnostic) {
	c.diagnostics = append(c.diagnostics, diag)
}

// Diagnostics returns a clone of all collected diagnostics.
func (c *SpanDiagnosticCollector) Diagnostics() []SpanDiagnostic {
	return slices.Clone(c.diagnostics)
}

// DiagnosticsFrom returns a clone of diagnostics collected since index n.
func (c *SpanDiagnosticCollector) DiagnosticsFrom(n int) []SpanDiagnostic {
	return slices.Clone(c.diagnostics[n:])
}

// DiagnosticCount returns the number of collected diagnostics.
func (c *SpanDiagnosticCollector) DiagnosticCount() int {
	return len(c.diagnostics)
}

// ResetDiagnostics clears collected diagnostics, retaining the underlying buffer.
func (c *SpanDiagnosticCollector) ResetDiagnostics() {
	c.diagnostics = c.diagnostics[:0]
}
