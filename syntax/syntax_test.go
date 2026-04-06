package syntax_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/golangsnmp/gomib/syntax"
)

var testMIB = []byte(`
EXAMPLE-MIB DEFINITIONS ::= BEGIN

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

END
`)

func TestParse_BasicStructure(t *testing.T) {
	file, diags := syntax.Parse(testMIB)
	if file == nil {
		t.Fatal("Parse returned nil ModuleFile")
	}
	if len(file.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(file.Modules))
	}
	mod := file.Modules[0]
	name := string(testMIB[mod.Name.Span.Start:mod.Name.Span.End])
	if name != "EXAMPLE-MIB" {
		t.Errorf("module name = %q, want %q", name, "EXAMPLE-MIB")
	}
	if len(mod.Body) < 2 {
		t.Errorf("expected at least 2 body definitions, got %d", len(mod.Body))
	}
	for _, d := range diags {
		if d.Severity <= 1 {
			t.Errorf("unexpected severe diagnostic: %s: %s", d.Code, d.Message)
		}
	}
}

func TestParse_RoundTrip(t *testing.T) {
	file, _ := syntax.Parse(testMIB)
	reconstructed := syntax.ReconstructText(file, testMIB)
	if !bytes.Equal(reconstructed, testMIB) {
		t.Errorf("round-trip failed: reconstructed text differs from original")
	}
}

func TestParse_RoundTrip_RealMIB(t *testing.T) {
	source, err := os.ReadFile("../testdata/corpus/primary/ietf/IF-MIB")
	if err != nil {
		t.Skipf("could not read IF-MIB: %v", err)
	}
	file, _ := syntax.Parse(source)
	reconstructed := syntax.ReconstructText(file, source)
	if !bytes.Equal(reconstructed, source) {
		t.Errorf("round-trip failed for IF-MIB")
	}
}

func TestTokenize(t *testing.T) {
	tokens := syntax.Tokenize(testMIB)
	if len(tokens) == 0 {
		t.Fatal("Tokenize returned no tokens")
	}
	last := tokens[len(tokens)-1]
	if last.Kind != syntax.TokEOF {
		t.Errorf("last token kind = %v, want EOF", last.Kind)
	}
	for _, tok := range tokens {
		if tok.Kind == syntax.TokUppercaseIdent {
			text := string(testMIB[tok.Span.Start:tok.Span.End])
			if text != "EXAMPLE-MIB" {
				t.Errorf("first ident = %q, want %q", text, "EXAMPLE-MIB")
			}
			break
		}
	}
}

func TestIdentifierAt(t *testing.T) {
	src := []byte("ifDescr OBJECT-TYPE\n    SYNTAX DisplayString\n")

	tests := []struct {
		offset int
		want   string
	}{
		{0, "ifDescr"},
		{3, "ifDescr"},
		{6, "ifDescr"},
		{7, ""},                                  // space
		{8, "OBJECT-TYPE"},
		{14, "OBJECT-TYPE"},                       // on the hyphen
		{len("ifDescr OBJECT-TYPE"), ""},           // newline
		{24, "SYNTAX"},
		{31, "DisplayString"},
		{len(src), ""},
		{len(src) - 1, ""},                        // trailing newline
	}
	for _, tt := range tests {
		got := syntax.IdentifierAt(src, syntax.ByteOffset(tt.offset))
		if got != tt.want {
			t.Errorf("IdentifierAt(src, %d) = %q, want %q", tt.offset, got, tt.want)
		}
	}
}

func TestIdentifierAtEmpty(t *testing.T) {
	got := syntax.IdentifierAt(nil, 0)
	if got != "" {
		t.Errorf("IdentifierAt(nil, 0) = %q, want empty", got)
	}
	got = syntax.IdentifierAt([]byte{}, 0)
	if got != "" {
		t.Errorf("IdentifierAt(empty, 0) = %q, want empty", got)
	}
}

func TestIdentifierAtUnderscores(t *testing.T) {
	src := []byte("some_ident_with_underscores")
	got := syntax.IdentifierAt(src, 5)
	if got != "some_ident_with_underscores" {
		t.Errorf("got %q, want full identifier with underscores", got)
	}
}

func TestParse_DefinitionNodeTypes(t *testing.T) {
	file, _ := syntax.Parse(testMIB)
	mod := file.Modules[0]

	var hasMI, hasOT bool
	for _, def := range mod.Body {
		switch def.(type) {
		case *syntax.ModuleIdentityNode:
			hasMI = true
		case *syntax.ObjectTypeNode:
			hasOT = true
		}
	}
	if !hasMI {
		t.Error("expected ModuleIdentityNode in body")
	}
	if !hasOT {
		t.Error("expected ObjectTypeNode in body")
	}
}
