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

func TestTokenAt(t *testing.T) {
	file, _ := syntax.Parse(testMIB)

	// "EXAMPLE-MIB" is the first token in the module.
	tok, ok := syntax.TokenAt(file, 1) // byte 1 is inside the module name
	if !ok {
		t.Fatal("expected token at offset 1")
	}
	name := string(testMIB[tok.Span.Start:tok.Span.End])
	if name != "EXAMPLE-MIB" {
		t.Errorf("token text = %q, want %q", name, "EXAMPLE-MIB")
	}
	if tok.Kind != syntax.TokUppercaseIdent {
		t.Errorf("token kind = %v, want TokUppercaseIdent", tok.Kind)
	}

	// Offset inside whitespace/trivia should return not-found.
	modNode := file.Modules[0]
	// The space between module name and DEFINITIONS keyword is trivia.
	nameEnd := modNode.Name.Span.End
	_, ok = syntax.TokenAt(file, nameEnd)
	if ok {
		t.Error("expected no token in whitespace trivia")
	}
}

func TestTokenAt_Empty(t *testing.T) {
	file, _ := syntax.Parse(nil)
	_, ok := syntax.TokenAt(file, 0)
	if ok {
		t.Error("expected no token for empty source")
	}
}

func TestTokenAt_IdentifiesKeywords(t *testing.T) {
	file, _ := syntax.Parse(testMIB)

	// Find the IMPORTS keyword by walking until we hit it.
	var importsOffset syntax.ByteOffset
	file.WalkTokens(func(tok syntax.SyntaxToken) {
		if tok.Kind == syntax.TokKwImports {
			importsOffset = tok.Span.Start
		}
	})

	tok, ok := syntax.TokenAt(file, importsOffset)
	if !ok {
		t.Fatal("expected token at IMPORTS offset")
	}
	if tok.Kind != syntax.TokKwImports {
		t.Errorf("token kind = %v, want TokKwImports", tok.Kind)
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
