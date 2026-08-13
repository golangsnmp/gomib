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
	// Use VerboseConfig so style-level diagnostics are reported.
	return parseModsWithConfig(t, source, types.VerboseConfig())
}

func parseModsWithConfig(t *testing.T, source string, cfg types.DiagnosticConfig) []*module.Module {
	t.Helper()
	src := []byte(source)
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

func TestWholeFileLexerDiagnosticAssignedByAbsoluteSpan(t *testing.T) {
	const source = "FIRST-MIB DEFINITIONS ::= BEGIN\nEND\n" +
		"SECOND-MIB DEFINITIONS ::= BEGIN\n@\nEND\n"

	mods := parseMods(t, source)
	if len(mods) != 2 {
		t.Fatalf("got %d modules, want 2", len(mods))
	}
	if mods[0].Name != "FIRST-MIB" || mods[1].Name != "SECOND-MIB" {
		t.Fatalf("module names = %q, %q, want FIRST-MIB, SECOND-MIB", mods[0].Name, mods[1].Name)
	}
	if len(mods[0].Diagnostics) != 0 {
		t.Fatalf("FIRST-MIB diagnostics = %v, want none", mods[0].Diagnostics)
	}
	if len(mods[1].Diagnostics) != 1 {
		t.Fatalf("SECOND-MIB diagnostics = %v, want one lexer diagnostic", mods[1].Diagnostics)
	}

	got := mods[1].Diagnostics[0]
	if got.Code != types.DiagUnexpectedCharacter || got.Module != "SECOND-MIB" ||
		got.Line != 4 || got.Column != 1 {
		t.Errorf("SECOND-MIB diagnostic = %#v, want unexpected-character at SECOND-MIB:4:1", got)
	}
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

func TestEnumValueInteger32Bounds(t *testing.T) {
	const source = `
TEST-MIB DEFINITIONS ::= BEGIN
Boundary ::= INTEGER {
    minimum(-2147483648),
    maximum(2147483647),
    below(-2147483649),
    vendorSentinel(4294967295)
}
END
`
	mods := parseMods(t, source)
	if len(mods) != 1 {
		t.Fatalf("got %d modules, want 1", len(mods))
	}
	if len(mods[0].TypeDefs) != 1 {
		t.Fatalf("got %d type definitions, want 1", len(mods[0].TypeDefs))
	}
	syntax, ok := mods[0].TypeDefs[0].Syntax.(*module.TypeSyntaxIntegerEnum)
	if !ok {
		t.Fatalf("syntax type = %T, want *module.TypeSyntaxIntegerEnum", mods[0].TypeDefs[0].Syntax)
	}
	want := []int64{-2147483648, 2147483647, -2147483649, 4294967295}
	if len(syntax.NamedNumbers) != len(want) {
		t.Fatalf("got %d named numbers, want %d", len(syntax.NamedNumbers), len(want))
	}
	for i, value := range want {
		if syntax.NamedNumbers[i].Value != value {
			t.Errorf("named number %q value = %d, want %d", syntax.NamedNumbers[i].Name, syntax.NamedNumbers[i].Value, value)
		}
	}

	var boundsDiags []types.Diagnostic
	for _, diag := range mods[0].Diagnostics {
		if diag.Code == types.DiagEnumValueOutOfRange {
			boundsDiags = append(boundsDiags, diag)
		}
	}
	if len(boundsDiags) != 2 {
		t.Fatalf("got %d bounds diagnostics, want 2", len(boundsDiags))
	}
	for _, diag := range boundsDiags {
		if diag.Severity != types.SeverityWarning {
			t.Errorf("bounds diagnostic severity = %v, want %v", diag.Severity, types.SeverityWarning)
		}
	}
}

func TestEnumValueBoundsDiagnosticConfig(t *testing.T) {
	const source = `
TEST-MIB DEFINITIONS ::= BEGIN
VendorEnum ::= INTEGER { vendorSentinel(4294967295) }
END
`

	if hasDiag(parseModsWithConfig(t, source, types.DefaultConfig()), types.DiagEnumValueOutOfRange) {
		t.Error("default reporting should suppress warning-level enum bounds diagnostic")
	}

	overridden := types.DefaultConfig()
	overridden.Overrides = map[string]types.Severity{types.DiagEnumValueOutOfRange: types.SeverityMinor}
	mods := parseModsWithConfig(t, source, overridden)
	if !hasDiag(mods, types.DiagEnumValueOutOfRange) {
		t.Error("severity override should enable enum bounds diagnostic at default reporting")
	}
	for _, diag := range mods[0].Diagnostics {
		if diag.Code == types.DiagEnumValueOutOfRange && diag.Severity != types.SeverityMinor {
			t.Errorf("overridden severity = %v, want %v", diag.Severity, types.SeverityMinor)
		}
	}

	ignored := types.VerboseConfig()
	ignored.Ignore = []string{types.DiagEnumValueOutOfRange}
	if hasDiag(parseModsWithConfig(t, source, ignored), types.DiagEnumValueOutOfRange) {
		t.Error("ignore configuration should suppress enum bounds diagnostic")
	}
}

func TestHexRangeLiteralLowering(t *testing.T) {
	mods := parseMods(t, "\nTEST-MIB DEFINITIONS ::= BEGIN\n"+
		"Whitespace ::= INTEGER ('7f ff'H | '7f\tff'H | '7f\rff'H | '7f\nff'H)\n"+
		"Malformed ::= INTEGER ('0G'H)\n"+
		"Overflow ::= INTEGER ('10000000000000000'H)\n"+
		"END\n")
	if len(mods) != 1 {
		t.Fatalf("got %d modules, want 1", len(mods))
	}
	if len(mods[0].TypeDefs) != 3 {
		t.Fatalf("got %d type definitions, want 3", len(mods[0].TypeDefs))
	}

	constraints := func(index int) []module.Range {
		t.Helper()
		syntax, ok := mods[0].TypeDefs[index].Syntax.(*module.TypeSyntaxConstrained)
		if !ok {
			t.Fatalf("type %d syntax = %T, want constrained", index, mods[0].TypeDefs[index].Syntax)
		}
		return syntax.Constraint.(*module.ConstraintRange).Ranges
	}

	valid := constraints(0)
	if len(valid) != 4 {
		t.Fatalf("got %d whitespace ranges, want 4", len(valid))
	}
	for index, r := range valid {
		min, ok := r.Min.(*module.RangeValueUnsigned)
		if !ok || min.Value != 0x7fff {
			t.Errorf("whitespace range %d endpoint = %#v, want 32767", index, r.Min)
		}
	}
	for index, want := range []string{"'0G'H", "'10000000000000000'H"} {
		got, ok := constraints(index + 1)[0].Min.(*module.RangeValueRaw)
		if !ok || got.Value != want {
			t.Errorf("type %d endpoint = %#v, want raw %q", index+1, got, want)
		}
	}

	var invalid int
	for _, diag := range mods[0].Diagnostics {
		if diag.Code == types.DiagInvalidHexRange {
			invalid++
		}
	}
	if invalid != 2 {
		t.Errorf("got %d invalid hex range diagnostics, want 2", invalid)
	}
}

func TestInvalidRangeEndpointsPreserveRawSource(t *testing.T) {
	const source = `
TEST-MIB DEFINITIONS ::= BEGIN
Overflow ::= INTEGER (18446744073709551616)
Underflow ::= INTEGER (-9223372036854775809)
Symbolic ::= INTEGER (VENDOR-MIN)
END
`
	mods := parseMods(t, source)
	if len(mods) != 1 {
		t.Fatalf("modules = %d, want 1", len(mods))
	}
	if len(mods[0].TypeDefs) != 3 {
		t.Fatalf("type definitions = %d, want 3", len(mods[0].TypeDefs))
	}

	want := []string{"18446744073709551616", "-9223372036854775809", "VENDOR-MIN"}
	for i, raw := range want {
		syntax, ok := mods[0].TypeDefs[i].Syntax.(*module.TypeSyntaxConstrained)
		if !ok {
			t.Fatalf("type %d syntax = %T, want constrained", i, mods[0].TypeDefs[i].Syntax)
		}
		got, ok := syntax.Constraint.(*module.ConstraintRange).Ranges[0].Min.(*module.RangeValueRaw)
		if !ok || got.Value != raw {
			t.Errorf("type %d endpoint = %#v, want raw %q", i, got, raw)
		}
	}

	var unknown int
	for _, diag := range mods[0].Diagnostics {
		if diag.Code == types.DiagUnknownRangeValue {
			unknown++
		}
	}
	if unknown != len(want) {
		t.Errorf("got %d unknown range diagnostics, want %d", unknown, len(want))
	}
}

func TestInvalidRangeEndpointDiagnosticConfig(t *testing.T) {
	const source = `
TEST-MIB DEFINITIONS ::= BEGIN
Unknown ::= INTEGER (VENDOR-MIN)
END
`
	if hasDiag(parseModsWithConfig(t, source, types.DefaultConfig()), types.DiagUnknownRangeValue) {
		t.Error("default reporting should suppress warning-level unknown range diagnostic")
	}

	overridden := types.DefaultConfig()
	overridden.Overrides = map[string]types.Severity{types.DiagUnknownRangeValue: types.SeverityMinor}
	if !hasDiag(parseModsWithConfig(t, source, overridden), types.DiagUnknownRangeValue) {
		t.Error("severity override should enable unknown range diagnostic")
	}

	ignored := types.VerboseConfig()
	ignored.Ignore = []string{types.DiagUnknownRangeValue}
	if hasDiag(parseModsWithConfig(t, source, ignored), types.DiagUnknownRangeValue) {
		t.Error("ignore configuration should suppress unknown range diagnostic")
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
