package lower_test

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/types"
)

func TestComplianceSyntaxRefinement(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-COMPLIANCE, OBJECT-GROUP FROM SNMPv2-CONF
        MODULE-IDENTITY FROM SNMPv2-SMI;
testCompliance MODULE-COMPLIANCE
    STATUS current
    DESCRIPTION "test"
    MODULE
        MANDATORY-GROUPS { testGroup }
        OBJECT testObj
            SYNTAX INTEGER (0..100)
            DESCRIPTION "refined"
    ::= { testCompliance 1 }
END
`)

	if len(mods) == 0 || len(mods[0].Compliances) == 0 {
		t.Fatal("expected at least one compliance")
	}
	comp := mods[0].Compliances[0]
	if len(comp.Modules) == 0 || len(comp.Modules[0].Objects) == 0 {
		t.Fatal("expected compliance module with objects")
	}
	obj := comp.Modules[0].Objects[0]
	if obj.Syntax == nil {
		t.Error("expected syntax refinement on compliance object")
	}
}

func TestComplianceNamedModule(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-COMPLIANCE, OBJECT-GROUP FROM SNMPv2-CONF
        MODULE-IDENTITY FROM SNMPv2-SMI;
testCompliance MODULE-COMPLIANCE
    STATUS current
    DESCRIPTION "test"
    MODULE SNMPv2-MIB
        MANDATORY-GROUPS { systemGroup }
    ::= { testCompliance 1 }
END
`)

	if len(mods) == 0 || len(mods[0].Compliances) == 0 {
		t.Fatal("expected at least one compliance")
	}
	comp := mods[0].Compliances[0]
	if len(comp.Modules) == 0 {
		t.Fatal("expected compliance module")
	}
	if comp.Modules[0].ModuleName != "SNMPv2-MIB" {
		t.Errorf("expected module name SNMPv2-MIB, got %q", comp.Modules[0].ModuleName)
	}
}

func TestMultiModuleDiagnosticIsolation(t *testing.T) {
	mods := parseMods(t, `
MOD-A DEFINITIONS ::= BEGIN
MyType ::= INTEGER { up(1), up(1) }
END

MOD-B DEFINITIONS ::= BEGIN
MyType ::= INTEGER
END
`)

	if len(mods) < 2 {
		t.Fatal("expected two modules")
	}
	hasA := false
	for _, d := range mods[0].Diagnostics {
		if d.Code == types.DiagEnumNameRedefinition {
			hasA = true
		}
	}
	if !hasA {
		t.Error("expected DiagEnumNameRedefinition in MOD-A")
	}
	for _, d := range mods[1].Diagnostics {
		if d.Code == types.DiagEnumNameRedefinition {
			t.Error("DiagEnumNameRedefinition should not leak to MOD-B")
		}
	}
}
