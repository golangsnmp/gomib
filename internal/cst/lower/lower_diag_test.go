package lower_test

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/cst/lower"
	cstparser "github.com/golangsnmp/gomib/internal/cst/parser"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

func parseMods(t *testing.T, source string) []*module.Module {
	t.Helper()
	src := []byte(source)
	// Use VerboseConfig so style-level diagnostics are reported.
	cfg := types.VerboseConfig()
	p := cstparser.New(src, nil, cfg)
	file := p.ParseModule()
	return lower.Lower(file, src, p.Diagnostics(), cfg)
}

func hasDiag(mods []*module.Module, code string) bool {
	for _, m := range mods {
		for _, d := range m.Diagnostics {
			if d.Code == code {
				return true
			}
		}
	}
	return false
}

func TestEmptyDescriptionObjectType(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, OBJECT-TYPE, Integer32 FROM SNMPv2-SMI;
testObj OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION ""
    ::= { testObj 1 }
END
`)
	if !hasDiag(mods, types.DiagEmptyDescription) {
		t.Error("expected DiagEmptyDescription for empty OBJECT-TYPE DESCRIPTION")
	}
}

func TestEmptyOrganization(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY FROM SNMPv2-SMI;
testMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION ""
    CONTACT-INFO "test"
    DESCRIPTION "test"
    ::= { testMIB 1 }
END
`)
	if !hasDiag(mods, types.DiagEmptyOrganization) {
		t.Error("expected DiagEmptyOrganization")
	}
}

func TestEmptyContact(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY FROM SNMPv2-SMI;
testMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "test"
    CONTACT-INFO ""
    DESCRIPTION "test"
    ::= { testMIB 1 }
END
`)
	if !hasDiag(mods, types.DiagEmptyContact) {
		t.Error("expected DiagEmptyContact")
	}
}

func TestEmptyReferenceObjectType(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, OBJECT-TYPE, Integer32 FROM SNMPv2-SMI;
testObj OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "ok"
    REFERENCE ""
    ::= { testObj 1 }
END
`)
	if !hasDiag(mods, types.DiagEmptyReference) {
		t.Error("expected DiagEmptyReference")
	}
}

func TestChoiceNotAllowed(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
MyType ::= CHOICE { a INTEGER, b OCTET STRING }
END
`)
	if !hasDiag(mods, types.DiagChoiceNotAllowed) {
		t.Error("expected DiagChoiceNotAllowed outside base module")
	}
}

func TestTaggedTypeNotAllowed(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
MyType ::= [APPLICATION 0] IMPLICIT OCTET STRING
END
`)
	if !hasDiag(mods, types.DiagTaggedTypeNotAllowed) {
		t.Error("expected DiagTaggedTypeNotAllowed outside base module")
	}
}

func TestNonEmptyDescriptionNoDiag(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, OBJECT-TYPE, Integer32 FROM SNMPv2-SMI;
testObj OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "A valid description"
    ::= { testObj 1 }
END
`)
	if hasDiag(mods, types.DiagEmptyDescription) {
		t.Error("should not emit DiagEmptyDescription for non-empty description")
	}
}

func TestEnumNameRedefinition(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, OBJECT-TYPE FROM SNMPv2-SMI;
testObj OBJECT-TYPE
    SYNTAX INTEGER { up(1), up(2) }
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "test"
    ::= { testObj 1 }
END
`)
	if !hasDiag(mods, types.DiagEnumNameRedefinition) {
		t.Error("expected DiagEnumNameRedefinition for duplicate enum name")
	}
}

func TestEnumValueRedefinition(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, OBJECT-TYPE FROM SNMPv2-SMI;
testObj OBJECT-TYPE
    SYNTAX INTEGER { up(1), down(1) }
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "test"
    ::= { testObj 1 }
END
`)
	if !hasDiag(mods, types.DiagEnumValueRedefinition) {
		t.Error("expected DiagEnumValueRedefinition for duplicate enum value")
	}
}

func TestBitsNameRedefinition(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, OBJECT-TYPE FROM SNMPv2-SMI;
testObj OBJECT-TYPE
    SYNTAX BITS { flag(0), flag(1) }
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "test"
    ::= { testObj 1 }
END
`)
	if !hasDiag(mods, types.DiagBitsNameRedefinition) {
		t.Error("expected DiagBitsNameRedefinition for duplicate BITS name")
	}
}

func TestBitsValueRedefinition(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, OBJECT-TYPE FROM SNMPv2-SMI;
testObj OBJECT-TYPE
    SYNTAX BITS { a(0), b(0) }
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "test"
    ::= { testObj 1 }
END
`)
	if !hasDiag(mods, types.DiagBitsValueRedefinition) {
		t.Error("expected DiagBitsValueRedefinition for duplicate BITS position")
	}
}

func TestBadIdentifierCaseTypeRef(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, OBJECT-TYPE FROM SNMPv2-SMI;
testObj OBJECT-TYPE
    SYNTAX myType
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "test"
    ::= { testObj 1 }
END
`)
	if !hasDiag(mods, types.DiagBadIdentifierCase) {
		t.Error("expected DiagBadIdentifierCase for lowercase type reference")
	}
}

func TestBadIdentifierCaseTypeAssignment(t *testing.T) {
	mods := parseMods(t, `
TEST-MIB DEFINITIONS ::= BEGIN
myType ::= INTEGER
END
`)
	if !hasDiag(mods, types.DiagBadIdentifierCase) {
		t.Error("expected DiagBadIdentifierCase for lowercase type assignment name")
	}
}
