package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
)

func newTestParser(source string) *Parser {
	return New([]byte(source), nil, types.DiagnosticConfig{})
}

func roundTrip(t *testing.T, source string) {
	t.Helper()
	p := newTestParser(source)
	file := p.ParseModule()
	got := string(cst.ReconstructText(file, []byte(source)))
	if got != source {
		t.Errorf("round-trip mismatch\ngot:\n%s\nwant:\n%s", got, source)
	}
}

func TestParseEmptyModule(t *testing.T) {
	source := "TEST-MIB DEFINITIONS ::= BEGIN\nEND\n"
	p := newTestParser(source)
	file := p.ParseModule()

	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}
	mod := file.Modules[0]
	if mod.Name.Kind != lexer.TokUppercaseIdent {
		t.Errorf("expected uppercase ident for name, got %v", mod.Name.Kind)
	}
	if p.text(mod.Name.Span) != "TEST-MIB" {
		t.Errorf("expected name TEST-MIB, got %q", p.text(mod.Name.Span))
	}
	if mod.Imports != nil {
		t.Error("expected no imports")
	}
	if len(mod.Body) != 0 {
		t.Errorf("expected no definitions, got %d", len(mod.Body))
	}
}

func TestRoundTripEmptyModule(t *testing.T) {
	roundTrip(t, "TEST-MIB DEFINITIONS ::= BEGIN\nEND\n")
}

func TestRoundTripModuleWithImports(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Integer32
        FROM SNMPv2-SMI
    DisplayString
        FROM SNMPv2-TC;

END
`
	roundTrip(t, source)
}

func TestParseImports(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE
        FROM SNMPv2-SMI
    DisplayString
        FROM SNMPv2-TC;

END
`
	p := newTestParser(source)
	file := p.ParseModule()

	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}
	mod := file.Modules[0]
	if mod.Imports == nil {
		t.Fatal("expected imports, got nil")
	}
	if len(mod.Imports.Groups) != 2 {
		t.Fatalf("expected 2 import groups, got %d", len(mod.Imports.Groups))
	}

	// First group: MODULE-IDENTITY, OBJECT-TYPE FROM SNMPv2-SMI
	g1 := mod.Imports.Groups[0]
	if len(g1.Symbols) != 2 {
		t.Errorf("expected 2 symbols in group 1, got %d", len(g1.Symbols))
	}
	if p.text(g1.Module.Span) != "SNMPv2-SMI" {
		t.Errorf("expected module SNMPv2-SMI, got %q", p.text(g1.Module.Span))
	}

	// Second group: DisplayString FROM SNMPv2-TC
	g2 := mod.Imports.Groups[1]
	if len(g2.Symbols) != 1 {
		t.Errorf("expected 1 symbol in group 2, got %d", len(g2.Symbols))
	}
}

func TestRoundTripObjectType(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testObject OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "A test object."
    ::= { testMIB 1 }

END
`
	roundTrip(t, source)
}

func TestParseObjectType(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testObject OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "A test object."
    ::= { testMIB 1 }

END
`
	p := newTestParser(source)
	file := p.ParseModule()

	mod := file.Modules[0]
	if len(mod.Body) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(mod.Body))
	}

	obj, ok := mod.Body[0].(*cst.ObjectTypeNode)
	if !ok {
		t.Fatalf("expected ObjectTypeNode, got %T", mod.Body[0])
	}

	if p.text(obj.Name.Span) != "testObject" {
		t.Errorf("expected name testObject, got %q", p.text(obj.Name.Span))
	}
	if obj.Syntax == nil {
		t.Error("expected syntax clause")
	}
	if obj.Access == nil {
		t.Error("expected access clause")
	}
	if obj.Status == nil {
		t.Error("expected status clause")
	}
	if obj.Description == nil {
		t.Error("expected description clause")
	}
	if obj.Oid == nil {
		t.Error("expected OID")
	}
}

func TestRoundTripValueAssignment(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testMIB OBJECT IDENTIFIER ::= { enterprises 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripModuleIdentity(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testMIB MODULE-IDENTITY
    LAST-UPDATED "200601010000Z"
    ORGANIZATION "Test Corp"
    CONTACT-INFO "contact@test.com"
    DESCRIPTION  "Test MIB module."
    REVISION     "200601010000Z"
    DESCRIPTION  "Initial version."
    ::= { enterprises 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripObjectIdentity(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testIdentity OBJECT-IDENTITY
    STATUS  current
    DESCRIPTION "A test identity."
    ::= { testMIB 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripNotificationType(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testNotif NOTIFICATION-TYPE
    OBJECTS { testObject }
    STATUS  current
    DESCRIPTION "A test notification."
    ::= { testMIB 2 }

END
`
	roundTrip(t, source)
}

func TestRoundTripTextualConvention(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

TestTC TEXTUAL-CONVENTION
    STATUS  current
    DESCRIPTION "A test TC."
    SYNTAX  Integer32 (0..100)

END
`
	roundTrip(t, source)
}

func TestRoundTripTextualConventionWithAssignment(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

TestTC ::= TEXTUAL-CONVENTION
    STATUS  current
    DESCRIPTION "A test TC."
    SYNTAX  Integer32 (0..100)

END
`
	roundTrip(t, source)
}

func TestRoundTripTypeAssignment(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

TestType ::= Integer32 (0..255)

END
`
	roundTrip(t, source)
}

func TestRoundTripObjectGroup(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testGroup OBJECT-GROUP
    OBJECTS { testObject }
    STATUS  current
    DESCRIPTION "A test group."
    ::= { testMIB 3 }

END
`
	roundTrip(t, source)
}

func TestRoundTripNotificationGroup(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testNotifGroup NOTIFICATION-GROUP
    NOTIFICATIONS { testNotif }
    STATUS  current
    DESCRIPTION "A test notification group."
    ::= { testMIB 4 }

END
`
	roundTrip(t, source)
}

func TestRoundTripTrapType(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testTrap TRAP-TYPE
    ENTERPRISE testEnterprise
    VARIABLES { testVar1, testVar2 }
    DESCRIPTION "A test trap."
    ::= 1

END
`
	roundTrip(t, source)
}

func TestRoundTripModuleCompliance(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testCompliance MODULE-COMPLIANCE
    STATUS  current
    DESCRIPTION "Test compliance."
    MODULE
        MANDATORY-GROUPS { testGroup }
    ::= { testMIB 5 }

END
`
	roundTrip(t, source)
}

func TestRoundTripAgentCapabilities(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testCaps AGENT-CAPABILITIES
    PRODUCT-RELEASE "1.0"
    STATUS  current
    DESCRIPTION "Test capabilities."
    SUPPORTS TEST-MIB
        INCLUDES { testGroup }
    ::= { testMIB 6 }

END
`
	roundTrip(t, source)
}

func TestRoundTripWithComments(t *testing.T) {
	source := `-- Module header comment
TEST-MIB DEFINITIONS ::= BEGIN

-- Import section
IMPORTS
    Integer32 -- inline comment
        FROM SNMPv2-SMI;

-- A value assignment
testMIB OBJECT IDENTIFIER ::= { enterprises 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripIntegerEnum(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testObj OBJECT-TYPE
    SYNTAX  INTEGER { up(1), down(2), testing(3) }
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test."
    ::= { testMIB 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripBitsSyntax(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testObj OBJECT-TYPE
    SYNTAX  BITS { flag1(0), flag2(1), flag3(2) }
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test."
    ::= { testMIB 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripOctetStringConstrained(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testObj OBJECT-TYPE
    SYNTAX  OCTET STRING (SIZE (0..255))
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test."
    ::= { testMIB 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripSequenceOf(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

TestTable ::= SEQUENCE OF TestEntry

END
`
	roundTrip(t, source)
}

func TestRoundTripSequence(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

TestEntry ::= SEQUENCE {
    testIndex   Integer32,
    testValue   OCTET STRING
}

END
`
	roundTrip(t, source)
}

func TestRoundTripObjectTypeWithIndex(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testEntry OBJECT-TYPE
    SYNTAX      TestEntry
    MAX-ACCESS  not-accessible
    STATUS      current
    DESCRIPTION "A row."
    INDEX { testIndex }
    ::= { testTable 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripObjectTypeWithAugments(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testAugEntry OBJECT-TYPE
    SYNTAX      TestAugEntry
    MAX-ACCESS  not-accessible
    STATUS      current
    DESCRIPTION "Augmented row."
    AUGMENTS { testEntry }
    ::= { testTable 2 }

END
`
	roundTrip(t, source)
}

func TestRoundTripObjectTypeWithDefVal(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testObj OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-write
    STATUS      current
    DESCRIPTION "Test."
    DEFVAL { 42 }
    ::= { testMIB 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripObjectTypeWithReference(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testObj OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test."
    REFERENCE   "RFC 1234, Section 5."
    ::= { testMIB 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripObjectTypeWithUnits(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testObj OBJECT-TYPE
    SYNTAX      Integer32
    UNITS       "seconds"
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test."
    ::= { testMIB 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripOidNamedNumber(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testObj OBJECT IDENTIFIER ::= { iso org(3) dod(6) 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripRangeConstraint(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

TestType ::= Integer32 (0..100 | 200..300)

END
`
	roundTrip(t, source)
}

func TestRoundTripSizeConstraint(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

TestString ::= OCTET STRING (SIZE (1..128))

END
`
	roundTrip(t, source)
}

func TestRoundTripDefValBits(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testObj OBJECT-TYPE
    SYNTAX      BITS { flag1(0), flag2(1) }
    MAX-ACCESS  read-write
    STATUS      current
    DESCRIPTION "Test."
    DEFVAL { { flag1 } }
    ::= { testMIB 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripDisplayHintTC(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

TestHint TEXTUAL-CONVENTION
    DISPLAY-HINT "255a"
    STATUS  current
    DESCRIPTION "A test TC with display hint."
    SYNTAX  OCTET STRING (SIZE (0..255))

END
`
	roundTrip(t, source)
}

func TestRoundTripComplianceWithGroupsAndObjects(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testCompliance MODULE-COMPLIANCE
    STATUS  current
    DESCRIPTION "Test compliance."
    MODULE
        MANDATORY-GROUPS { testGroup }
        GROUP testOptionalGroup
            DESCRIPTION "Optional group."
        OBJECT testObj
            SYNTAX Integer32 (0..100)
            DESCRIPTION "Restricted."
    ::= { testMIB 5 }

END
`
	roundTrip(t, source)
}

func TestRoundTripAgentCapabilitiesWithVariation(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testCaps AGENT-CAPABILITIES
    PRODUCT-RELEASE "1.0"
    STATUS  current
    DESCRIPTION "Test capabilities."
    SUPPORTS TEST-MIB
        INCLUDES { testGroup }
        VARIATION testObj
            SYNTAX Integer32 (0..50)
            ACCESS read-only
            DESCRIPTION "Restricted."
    ::= { testMIB 6 }

END
`
	roundTrip(t, source)
}

func TestRoundTripMultipleDefinitions(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testMIB OBJECT IDENTIFIER ::= { enterprises 1 }

testObj OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test."
    ::= { testMIB 1 }

TestType ::= Integer32 (0..100)

END
`
	roundTrip(t, source)
}

func TestEOFHasTrailingTrivia(t *testing.T) {
	// Trailing whitespace after END should be attached to EOF trivia.
	source := "TEST-MIB DEFINITIONS ::= BEGIN\nEND\n\n"
	p := newTestParser(source)
	file := p.ParseModule()
	got := string(cst.ReconstructText(file, []byte(source)))
	if got != source {
		t.Errorf("round-trip mismatch\ngot:  %q\nwant: %q", got, source)
	}
}

func TestRoundTripObjectIdentifierSyntax(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testObj OBJECT-TYPE
    SYNTAX      OBJECT IDENTIFIER
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test."
    ::= { testMIB 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripQualifiedOidComponent(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

testObj OBJECT IDENTIFIER ::= { SNMPv2-SMI.internet 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripMultiModuleWithMacro(t *testing.T) {
	source := `FIRST-MIB DEFINITIONS ::= BEGIN

FOO MACRO ::=
BEGIN
    TYPE NOTATION ::= body
END

testObj OBJECT IDENTIFIER ::= { enterprises 1 }

END

SECOND-MIB DEFINITIONS ::= BEGIN

END
`
	p := newTestParser(source)
	file := p.ParseModule()
	got := string(cst.ReconstructText(file, []byte(source)))
	if got != source {
		minLen := min(len(got), len(source))
		diffPos := minLen
		for i := range minLen {
			if got[i] != source[i] {
				diffPos = i
				break
			}
		}
		t.Errorf("round-trip mismatch at byte %d\ngot len=%d, want len=%d\ngot:  %q\nwant: %q",
			diffPos, len(got), len(source), got, source)
	}
}

func TestRoundTripMultipleMacroDefinitions(t *testing.T) {
	// Tests the case where a module contains multiple MACRO definitions
	// with keyword names, like in SNMPv2-SMI.
	source := `TEST-MIB DEFINITIONS ::= BEGIN

-- first macro
MODULE-IDENTITY MACRO ::=
BEGIN
    TYPE NOTATION ::= body1
END

-- second macro
OBJECT-TYPE MACRO ::=
BEGIN
    TYPE NOTATION ::= body2
END

-- a regular definition
testObj OBJECT IDENTIFIER ::= { enterprises 1 }

END
`
	roundTrip(t, source)
}

func TestRoundTripKeywordMacro(t *testing.T) {
	// NOTIFICATION-TYPE MACRO in particular.
	source := `TEST-MIB DEFINITIONS ::= BEGIN

NOTIFICATION-TYPE MACRO ::=
BEGIN
    TYPE NOTATION ::= body
END

END
`
	roundTrip(t, source)
}

func TestRoundTripMacroThenTypeAssignment(t *testing.T) {
	// MACRO definition followed by OBJECT IDENTIFIER type assignment.
	source := `TEST-MIB DEFINITIONS ::= BEGIN

MODULE-IDENTITY MACRO ::=
BEGIN
    Text ::= value(IA5String)
END

ObjectName ::=
    OBJECT IDENTIFIER

END
`
	roundTrip(t, source)
}

func TestRoundTripChoiceType(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

ObjectSyntax ::=
    CHOICE {
        simple
            SimpleSyntax,
        complex
            ApplicationSyntax
    }

END
`
	roundTrip(t, source)
}

func TestRoundTripTaggedType(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

IpAddress ::=
    [APPLICATION 0]
        IMPLICIT OCTET STRING (SIZE (4))

END
`
	roundTrip(t, source)
}

func TestRoundTripNegativeRange(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

Integer32 ::=
        INTEGER (-2147483648..2147483647)

END
`
	roundTrip(t, source)
}

func TestRoundTripLargeRange(t *testing.T) {
	source := `TEST-MIB DEFINITIONS ::= BEGIN

Counter64 ::=
    [APPLICATION 6]
        IMPLICIT INTEGER (0..18446744073709551615)

END
`
	roundTrip(t, source)
}

// TestCorpusRoundTrip tests the round-trip invariant on real MIB files from
// the corpus. Any MIB that can be parsed without fatal errors must round-trip.
func TestCorpusRoundTrip(t *testing.T) {
	corpusDir := filepath.Join("..", "..", "..", "testdata", "corpus", "primary")
	if _, err := os.Stat(corpusDir); os.IsNotExist(err) {
		t.Skip("corpus not found")
	}

	// A selection of representative MIBs covering different definition types.
	mibs := []string{
		"ietf/IF-MIB.mib",
		"ietf/SNMPv2-MIB.mib",
		"ietf/SNMPv2-SMI.mib",
		"ietf/SNMPv2-TC.mib",
		"ietf/SNMPv2-CONF.mib",
		"ietf/HOST-RESOURCES-MIB.mib",
		"ietf/IP-MIB.mib",
		"ietf/BRIDGE-MIB.mib",
		"iana/IANAifType-MIB.mib",
	}

	for _, mib := range mibs {
		t.Run(mib, func(t *testing.T) {
			path := filepath.Join(corpusDir, mib)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Skipf("cannot read %s: %v", path, err)
				return
			}

			p := New(source, nil, types.DiagnosticConfig{})
			file := p.ParseModule()
			got := cst.ReconstructText(file, source)
			if string(got) != string(source) {
				// Find first difference for debugging.
				minLen := min(len(got), len(source))
				diffPos := minLen
				for i := range minLen {
					if got[i] != source[i] {
						diffPos = i
						break
					}
				}
				start := max(diffPos-40, 0)
				end := min(diffPos+40, len(source))
				endGot := min(diffPos+40, len(got))
				t.Errorf("round-trip mismatch at byte %d (len got=%d, want=%d)\nsource[%d:%d]: %q\ngot[%d:%d]:    %q",
					diffPos, len(got), len(source),
					start, end, source[start:end],
					start, endGot, got[start:endGot])
			}
		})
	}
}
