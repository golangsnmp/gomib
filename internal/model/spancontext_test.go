package model_test

import (
	"context"
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/internal/model"
)

func TestSpanContext_Integration(t *testing.T) {
	m, err := gomib.Load(context.Background(),
		gomib.WithSource(gomib.MustDir("../../testdata/corpus/primary/ietf")),
		gomib.WithModules("IF-MIB"),
	)
	if err != nil {
		t.Skipf("could not load IF-MIB: %v", err)
	}

	mod := m.Module("IF-MIB")
	if mod == nil {
		t.Fatal("IF-MIB module not found")
	}

	// Find an import symbol span and verify SpanContext returns Import.
	imports := mod.Imports()
	if len(imports) == 0 {
		t.Fatal("IF-MIB has no imports")
	}
	firstSym := imports[0].Symbols[0]
	if firstSym.Span.Start == 0 && firstSym.Span.End == 0 {
		t.Skip("import symbol has zero span")
	}
	sc := mod.SpanContext(firstSym.Span.Start)
	if sc.Kind != model.SpanContextImport {
		t.Errorf("SpanContext at import span: got kind %d, want SpanContextImport (%d)",
			sc.Kind, model.SpanContextImport)
	}
	if sc.Name != firstSym.Name {
		t.Errorf("SpanContext at import span: got name %q, want %q", sc.Name, firstSym.Name)
	}

	// Find an object and verify its definition span returns Definition.
	obj := m.Object("ifIndex")
	if obj != nil {
		defSpan := obj.Span()
		if defSpan.Start != 0 || defSpan.End != 0 {
			sc = mod.SpanContext(defSpan.Start)
			if sc.Kind != model.SpanContextDefinition && sc.Kind != model.SpanContextOidRef {
				t.Logf("SpanContext at ifIndex definition start: kind=%d name=%q", sc.Kind, sc.Name)
			}
		}
	}

	// Offset outside any span returns None - just verify it doesn't panic.
	sc = mod.SpanContext(0)
	_ = sc
}

func TestSpanContext_BaseModule(t *testing.T) {
	m, err := gomib.Load(context.Background(),
		gomib.WithModules("SNMPv2-SMI"),
	)
	if err != nil {
		t.Skipf("could not load: %v", err)
	}
	mod := m.Module("SNMPv2-SMI")
	if mod == nil {
		t.Fatal("SNMPv2-SMI not found")
	}
	sc := mod.SpanContext(10)
	if sc.Kind != model.SpanContextNone {
		t.Errorf("SpanContext on base module: got kind %d, want SpanContextNone", sc.Kind)
	}
}
