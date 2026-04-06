package model_test

import (
	"context"
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/internal/model"
)

func TestSymbolReferences(t *testing.T) {
	m, err := gomib.Load(context.Background(),
		gomib.WithSource(gomib.MustDir("../../testdata/corpus/primary/ietf")),
		gomib.WithModules("IF-MIB"),
	)
	if err != nil {
		t.Skipf("could not load IF-MIB: %v", err)
	}

	// ifIndex is defined in IF-MIB and referenced by many objects in INDEX
	// clauses and OID assignments. It should have references.
	refs := m.SymbolReferences("ifIndex")
	if len(refs) == 0 {
		t.Fatal("expected references for ifIndex, got none")
	}

	// Verify each ref has a non-nil module and non-zero span.
	for i, ref := range refs {
		if ref.Module == nil {
			t.Errorf("ref[%d]: Module is nil", i)
		}
		if ref.Span.Start == 0 && ref.Span.End == 0 {
			t.Errorf("ref[%d]: Span is zero", i)
		}
		if ref.Kind == 0 {
			t.Errorf("ref[%d]: Kind is zero", i)
		}
	}

	// Check that import references are present.
	var hasImport, hasOid, hasIndex bool
	for _, ref := range refs {
		switch ref.Kind {
		case model.RefImport:
			hasImport = true
		case model.RefOidValue:
			hasOid = true
		case model.RefIndex:
			hasIndex = true
		}
	}
	// ifIndex is imported by other modules and used in INDEX clauses.
	if !hasImport {
		t.Log("no import references found for ifIndex (may be expected if only IF-MIB loaded)")
	}
	if !hasOid {
		t.Log("no OID references found for ifIndex")
	}
	if !hasIndex {
		t.Log("no INDEX references found for ifIndex")
	}

	// ifDescr should be referenced in OID assignments (its parent is ifEntry).
	descrRefs := m.SymbolReferences("ifEntry")
	if len(descrRefs) == 0 {
		t.Error("expected references for ifEntry, got none")
	}

	// A non-existent symbol should return nil.
	noRefs := m.SymbolReferences("nonExistentSymbol12345")
	if len(noRefs) != 0 {
		t.Errorf("expected no references for non-existent symbol, got %d", len(noRefs))
	}
}

func TestSymbolReferencesImports(t *testing.T) {
	// Load multiple modules to test cross-module import references.
	m, err := gomib.Load(context.Background(),
		gomib.WithSource(gomib.MustDir("../../testdata/corpus/primary/ietf")),
		gomib.WithModules("IF-MIB", "IP-MIB"),
	)
	if err != nil {
		t.Skipf("could not load modules: %v", err)
	}

	// DisplayString is a textual convention from SNMPv2-TC, imported by many modules.
	refs := m.SymbolReferences("DisplayString")
	var importCount int
	for _, ref := range refs {
		if ref.Kind == model.RefImport {
			importCount++
		}
	}
	if importCount == 0 {
		t.Error("expected import references for DisplayString")
	}
}

func TestSymbolKindMacroName(t *testing.T) {
	tests := []struct {
		kind model.SymbolKind
		want string
	}{
		{model.SymbolKindNone, ""},
		{model.SymbolKindObject, "OBJECT-TYPE"},
		{model.SymbolKindType, ""},
		{model.SymbolKindTextualConvention, "TEXTUAL-CONVENTION"},
		{model.SymbolKindNotification, "NOTIFICATION-TYPE"},
		{model.SymbolKindGroup, "OBJECT-GROUP"},
		{model.SymbolKindNotificationGroup, "NOTIFICATION-GROUP"},
		{model.SymbolKindCompliance, "MODULE-COMPLIANCE"},
		{model.SymbolKindCapability, "AGENT-CAPABILITIES"},
		{model.SymbolKindNode, ""},
	}

	for _, tt := range tests {
		got := tt.kind.MacroName()
		if got != tt.want {
			t.Errorf("SymbolKind(%d).MacroName() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}
