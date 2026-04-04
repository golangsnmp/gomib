package types

import (
	"testing"
)

func TestSpanDiagnosticCollector_EmitDiagnostic(t *testing.T) {
	config := DefaultConfig()
	c := NewSpanDiagnosticCollector(config)

	span := NewSpan(10, 20)
	c.EmitDiagnostic(DiagParseError, span, "unexpected token")

	if c.DiagnosticCount() != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", c.DiagnosticCount())
	}
	diags := c.Diagnostics()
	if diags[0].Code != DiagParseError {
		t.Errorf("expected code %q, got %q", DiagParseError, diags[0].Code)
	}
	if diags[0].Span != span {
		t.Errorf("expected span %v, got %v", span, diags[0].Span)
	}
	if diags[0].Message != "unexpected token" {
		t.Errorf("expected message %q, got %q", "unexpected token", diags[0].Message)
	}
	if diags[0].Severity != SeverityForCode(DiagParseError) {
		t.Errorf("expected severity %v, got %v", SeverityForCode(DiagParseError), diags[0].Severity)
	}
}

func TestSpanDiagnosticCollector_EmitFiltered(t *testing.T) {
	config := SilentConfig()
	c := NewSpanDiagnosticCollector(config)

	// Non-fatal diagnostic should be filtered under silent config.
	c.EmitDiagnostic(DiagIdentifierUnderscore, NewSpan(0, 5), "has underscore")

	if c.DiagnosticCount() != 0 {
		t.Fatalf("expected 0 diagnostics under silent config, got %d", c.DiagnosticCount())
	}
}

func TestSpanDiagnosticCollector_RecordDiagnostic(t *testing.T) {
	config := SilentConfig()
	c := NewSpanDiagnosticCollector(config)

	// RecordDiagnostic bypasses filtering.
	diag := SpanDiagnostic{
		Severity: SeverityMinor,
		Code:     DiagIdentifierUnderscore,
		Span:     NewSpan(0, 5),
		Message:  "has underscore",
	}
	c.RecordDiagnostic(diag)

	if c.DiagnosticCount() != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", c.DiagnosticCount())
	}
	got := c.Diagnostics()
	if got[0] != diag {
		t.Errorf("expected %v, got %v", diag, got[0])
	}
}

func TestSpanDiagnosticCollector_DiagnosticsFrom(t *testing.T) {
	config := DefaultConfig()
	c := NewSpanDiagnosticCollector(config)

	c.EmitDiagnostic(DiagParseError, NewSpan(0, 5), "first")
	c.EmitDiagnostic(DiagParseError, NewSpan(10, 15), "second")
	c.EmitDiagnostic(DiagParseError, NewSpan(20, 25), "third")

	from1 := c.DiagnosticsFrom(1)
	if len(from1) != 2 {
		t.Fatalf("expected 2 diagnostics from index 1, got %d", len(from1))
	}
	if from1[0].Message != "second" {
		t.Errorf("expected %q, got %q", "second", from1[0].Message)
	}
	if from1[1].Message != "third" {
		t.Errorf("expected %q, got %q", "third", from1[1].Message)
	}
}

func TestSpanDiagnosticCollector_DiagnosticsClone(t *testing.T) {
	config := DefaultConfig()
	c := NewSpanDiagnosticCollector(config)

	c.EmitDiagnostic(DiagParseError, NewSpan(0, 5), "original")

	diags := c.Diagnostics()
	diags[0].Message = "mutated"

	// Original should be unaffected.
	got := c.Diagnostics()
	if got[0].Message != "original" {
		t.Errorf("Diagnostics() should return a clone; original was mutated")
	}
}

func TestSpanDiagnosticCollector_ResetDiagnostics(t *testing.T) {
	config := DefaultConfig()
	c := NewSpanDiagnosticCollector(config)

	c.EmitDiagnostic(DiagParseError, NewSpan(0, 5), "before reset")
	c.ResetDiagnostics()

	if c.DiagnosticCount() != 0 {
		t.Fatalf("expected 0 after reset, got %d", c.DiagnosticCount())
	}
}
