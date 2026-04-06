package syntax_test

import (
	"strings"
	"testing"

	"github.com/golangsnmp/gomib/syntax"
)

var cursorTestMIB = []byte(`EXAMPLE-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, Integer32
        FROM SNMPv2-SMI
    DisplayString
        FROM SNMPv2-TC;

exampleMIB MODULE-IDENTITY
    LAST-UPDATED "200601010000Z"
    ORGANIZATION "Example Inc."
    CONTACT-INFO "support@example.com"
    DESCRIPTION  "An example MIB."
    ::= { enterprises 99999 }

exampleString OBJECT-TYPE
    SYNTAX      DisplayString (SIZE (0..255))
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "An example string object."
    ::= { exampleMIB 1 }

-- a trailing comment
END
`)

// findOffset returns the byte offset of the first occurrence of needle in source.
func findOffset(source []byte, needle string) syntax.ByteOffset {
	idx := strings.Index(string(source), needle)
	if idx < 0 {
		panic("needle not found: " + needle)
	}
	return syntax.ByteOffset(idx)
}

// findNthOffset returns the byte offset of the n-th (1-based) occurrence.
func findNthOffset(source []byte, needle string, n int) syntax.ByteOffset {
	s := string(source)
	idx := 0
	for i := range n {
		pos := strings.Index(s[idx:], needle)
		if pos < 0 {
			panic("needle not found")
		}
		if i < n-1 {
			idx += pos + len(needle)
		} else {
			idx += pos
		}
	}
	return syntax.ByteOffset(idx)
}

func TestCursorContextAt_OutsideModule(t *testing.T) {
	// Put together source with content before the module.
	source := []byte("   \nEXAMPLE-MIB DEFINITIONS ::= BEGIN\nEND\n")
	file, _ := syntax.Parse(source)

	ctx := syntax.CursorContextAt(file, 0) // in leading whitespace before module
	if ctx.Module != nil {
		t.Error("expected Module nil for offset before module start")
	}
}

func TestCursorContextAt_ModuleName(t *testing.T) {
	file, _ := syntax.Parse(cursorTestMIB)

	// Offset on "EXAMPLE-MIB" at the very start.
	offset := findOffset(cursorTestMIB, "EXAMPLE-MIB")
	ctx := syntax.CursorContextAt(file, offset)

	if ctx.Module == nil {
		t.Fatal("expected Module non-nil")
	}
	name := string(cursorTestMIB[ctx.Module.Name.Span.Start:ctx.Module.Name.Span.End])
	if name != "EXAMPLE-MIB" {
		t.Errorf("Module.Name = %q, want %q", name, "EXAMPLE-MIB")
	}
	if ctx.Definition != nil {
		t.Error("expected Definition nil on module name")
	}
	if ctx.Imports != nil {
		t.Error("expected Imports nil on module name")
	}
}

func TestCursorContextAt_ImportsSection(t *testing.T) {
	file, _ := syntax.Parse(cursorTestMIB)

	// On "MODULE-IDENTITY" inside imports - should be in imports, in a group.
	offset := findOffset(cursorTestMIB, "MODULE-IDENTITY, Integer32")
	ctx := syntax.CursorContextAt(file, offset)

	if ctx.Module == nil {
		t.Fatal("expected Module non-nil")
	}
	if ctx.Imports == nil {
		t.Fatal("expected Imports non-nil")
	}
	if ctx.ImportGroup == nil {
		t.Fatal("expected ImportGroup non-nil")
	}
	if ctx.Definition != nil {
		t.Error("expected Definition nil inside imports")
	}
}

func TestCursorContextAt_ImportGroupAfterFROM(t *testing.T) {
	file, _ := syntax.Parse(cursorTestMIB)

	// On "SNMPv2-SMI" (the module name after FROM) in the first import group.
	offset := findOffset(cursorTestMIB, "SNMPv2-SMI")
	ctx := syntax.CursorContextAt(file, offset)

	if ctx.ImportGroup == nil {
		t.Fatal("expected ImportGroup non-nil")
	}
	// Verify we can tell we're after FROM.
	if ctx.ImportGroup.From.IsZero() {
		t.Error("expected FROM token non-zero")
	}
	if offset < ctx.ImportGroup.From.Span.End {
		t.Error("expected offset to be after FROM token")
	}
}

func TestCursorContextAt_ImportGroupBeforeFROM(t *testing.T) {
	file, _ := syntax.Parse(cursorTestMIB)

	// On "DisplayString" in the second import group - before FROM.
	offset := findNthOffset(cursorTestMIB, "DisplayString", 1)
	ctx := syntax.CursorContextAt(file, offset)

	if ctx.ImportGroup == nil {
		t.Fatal("expected ImportGroup non-nil")
	}
	if ctx.ImportGroup.From.IsZero() {
		t.Fatal("expected FROM token non-zero")
	}
	if offset >= ctx.ImportGroup.From.Span.Start {
		t.Error("expected offset to be before FROM token")
	}
}

func TestCursorContextAt_Definition(t *testing.T) {
	file, _ := syntax.Parse(cursorTestMIB)

	// On "OBJECT-TYPE" keyword inside the exampleString definition.
	offset := findOffset(cursorTestMIB, "OBJECT-TYPE")
	ctx := syntax.CursorContextAt(file, offset)

	if ctx.Module == nil {
		t.Fatal("expected Module non-nil")
	}
	if ctx.Definition == nil {
		t.Fatal("expected Definition non-nil")
	}
	if _, ok := ctx.Definition.(*syntax.ObjectTypeNode); !ok {
		t.Errorf("expected ObjectTypeNode, got %T", ctx.Definition)
	}
	if ctx.Imports != nil {
		t.Error("expected Imports nil inside definition")
	}
}

func TestCursorContextAt_OidValue(t *testing.T) {
	file, _ := syntax.Parse(cursorTestMIB)

	// On "enterprises" inside "::= { enterprises 99999 }".
	offset := findOffset(cursorTestMIB, "enterprises")
	ctx := syntax.CursorContextAt(file, offset)

	if ctx.Definition == nil {
		t.Fatal("expected Definition non-nil")
	}
	if ctx.OidValue == nil {
		t.Fatal("expected OidValue non-nil")
	}
}

func TestCursorContextAt_OidValue_ObjectType(t *testing.T) {
	file, _ := syntax.Parse(cursorTestMIB)

	// On "exampleMIB" inside "::= { exampleMIB 1 }" of the OBJECT-TYPE.
	// This is the second occurrence of "exampleMIB" in the source.
	offset := findNthOffset(cursorTestMIB, "exampleMIB", 2)
	ctx := syntax.CursorContextAt(file, offset)

	if ctx.Definition == nil {
		t.Fatal("expected Definition non-nil")
	}
	if _, ok := ctx.Definition.(*syntax.ObjectTypeNode); !ok {
		t.Errorf("expected ObjectTypeNode, got %T", ctx.Definition)
	}
	if ctx.OidValue == nil {
		t.Fatal("expected OidValue non-nil")
	}
}

func TestCursorContextAt_InString(t *testing.T) {
	file, _ := syntax.Parse(cursorTestMIB)

	// Inside the "An example MIB." string.
	offset := findOffset(cursorTestMIB, "An example MIB")
	ctx := syntax.CursorContextAt(file, offset)

	if !ctx.InString {
		t.Error("expected InString true")
	}
	if ctx.InComment {
		t.Error("expected InComment false")
	}
}

func TestCursorContextAt_InComment(t *testing.T) {
	file, _ := syntax.Parse(cursorTestMIB)

	// Inside the "-- a trailing comment".
	offset := findOffset(cursorTestMIB, "a trailing comment")
	ctx := syntax.CursorContextAt(file, offset)

	if !ctx.InComment {
		t.Error("expected InComment true")
	}
	if ctx.InString {
		t.Error("expected InString false")
	}
}

func TestCursorContextAt_NotInCommentOrString(t *testing.T) {
	file, _ := syntax.Parse(cursorTestMIB)

	// On "read-only" - a keyword, not in comment or string.
	offset := findOffset(cursorTestMIB, "read-only")
	ctx := syntax.CursorContextAt(file, offset)

	if ctx.InComment {
		t.Error("expected InComment false")
	}
	if ctx.InString {
		t.Error("expected InString false")
	}
}

func TestCursorContextAt_BetweenDefinitions(t *testing.T) {
	file, _ := syntax.Parse(cursorTestMIB)

	// Find the blank line between the two definitions.
	// The MODULE-IDENTITY ends with "::= { enterprises 99999 }" and then
	// there's a blank line before "exampleString OBJECT-TYPE".
	miEnd := findOffset(cursorTestMIB, "exampleString OBJECT-TYPE")
	// Go back a few bytes to be on the blank line.
	offset := miEnd - 1
	ctx := syntax.CursorContextAt(file, offset)

	if ctx.Module == nil {
		t.Fatal("expected Module non-nil")
	}
	// Between definitions, Definition might be nil (cursor is in whitespace
	// between defs) or might be in the preceding definition depending on span
	// boundaries. The key test is that Imports is nil and we're in the module.
	if ctx.Imports != nil {
		t.Error("expected Imports nil between definitions")
	}
}

func TestCursorContextAt_NotOidWhenOutsideBraces(t *testing.T) {
	file, _ := syntax.Parse(cursorTestMIB)

	// On "SYNTAX" keyword - inside definition but not in OID.
	offset := findOffset(cursorTestMIB, "SYNTAX      DisplayString")
	ctx := syntax.CursorContextAt(file, offset)

	if ctx.OidValue != nil {
		t.Error("expected OidValue nil on SYNTAX keyword")
	}
}
