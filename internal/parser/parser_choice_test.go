package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestParseChoiceTypeAssignment(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestChoice ::= CHOICE {
			internet Integer32,
			raw DisplayString
		}
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.TypeAssignmentDef)
	testutil.True(t, ok, "expected TypeAssignmentDef, got %T", module.Body[0])
	testutil.Equal(t, "TestChoice", def.Name.Name, "type name")

	choice, ok := def.Syntax.(*ast.TypeSyntaxChoice)
	testutil.True(t, ok, "expected TypeSyntaxChoice, got %T", def.Syntax)
	testutil.Len(t, choice.Alternatives, 2, "alternatives count")
	testutil.Equal(t, "internet", choice.Alternatives[0].Name.Name, "first alternative name")
	testutil.Equal(t, "raw", choice.Alternatives[1].Name.Name, "second alternative name")
}

func TestParseChoiceWithBuiltinTypes(t *testing.T) {
	// Test CHOICE with builtin types like INTEGER and OCTET STRING
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestBuiltinChoice ::= CHOICE {
			num INTEGER,
			str OCTET STRING
		}
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.TypeAssignmentDef)
	testutil.True(t, ok, "expected TypeAssignmentDef, got %T", module.Body[0])

	choice, ok := def.Syntax.(*ast.TypeSyntaxChoice)
	testutil.True(t, ok, "expected TypeSyntaxChoice, got %T", def.Syntax)
	testutil.Len(t, choice.Alternatives, 2, "alternatives count")
	testutil.Equal(t, "num", choice.Alternatives[0].Name.Name, "first alternative name")
	testutil.Equal(t, "str", choice.Alternatives[1].Name.Name, "second alternative name")
}

func TestParseChoiceSingleAlternative(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		SingleChoice ::= CHOICE {
			only Integer32
		}
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.TypeAssignmentDef)
	testutil.True(t, ok, "expected TypeAssignmentDef, got %T", module.Body[0])

	choice, ok := def.Syntax.(*ast.TypeSyntaxChoice)
	testutil.True(t, ok, "expected TypeSyntaxChoice, got %T", def.Syntax)
	testutil.Len(t, choice.Alternatives, 1, "alternatives count")
	testutil.Equal(t, "only", choice.Alternatives[0].Name.Name, "alternative name")
}

func TestParseChoiceAlternativeSyntax(t *testing.T) {
	// Verify that alternative Syntax fields are populated correctly.
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestChoice ::= CHOICE {
			num INTEGER,
			str OCTET STRING,
			ref DisplayString
		}
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def := module.Body[0].(*ast.TypeAssignmentDef)
	choice := def.Syntax.(*ast.TypeSyntaxChoice)
	testutil.Len(t, choice.Alternatives, 3, "alternatives count")

	// INTEGER alternative should have a builtin syntax
	if choice.Alternatives[0].Syntax == nil {
		t.Fatal("first alternative should have non-nil Syntax")
	}
	// OCTET STRING alternative
	if choice.Alternatives[1].Syntax == nil {
		t.Fatal("second alternative should have non-nil Syntax")
	}
	// DisplayString is a type reference
	ref, ok := choice.Alternatives[2].Syntax.(*ast.TypeSyntaxTypeRef)
	testutil.True(t, ok, "third alternative Syntax: expected TypeSyntaxTypeRef, got %T", choice.Alternatives[2].Syntax)
	testutil.Equal(t, "DisplayString", ref.Name.Name, "type ref name")
}

func TestParseChoiceWithNamedNumbers(t *testing.T) {
	// CHOICE alternative with INTEGER that has named number values.
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestChoice ::= CHOICE {
			status INTEGER { up(1), down(2) },
			name OCTET STRING
		}
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def := module.Body[0].(*ast.TypeAssignmentDef)
	choice := def.Syntax.(*ast.TypeSyntaxChoice)
	testutil.Len(t, choice.Alternatives, 2, "alternatives count")

	// First alternative should have an IntegerEnum syntax with named numbers
	intEnum, ok := choice.Alternatives[0].Syntax.(*ast.TypeSyntaxIntegerEnum)
	testutil.True(t, ok, "expected TypeSyntaxIntegerEnum, got %T", choice.Alternatives[0].Syntax)
	testutil.Len(t, intEnum.NamedNumbers, 2, "named numbers count")
}

func TestParseChoiceNested(t *testing.T) {
	// Nested CHOICE inside a CHOICE alternative.
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestChoice ::= CHOICE {
			inner CHOICE {
				x INTEGER,
				y OCTET STRING
			},
			other Integer32
		}
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def := module.Body[0].(*ast.TypeAssignmentDef)
	outer := def.Syntax.(*ast.TypeSyntaxChoice)
	testutil.Len(t, outer.Alternatives, 2, "outer alternatives count")

	// First alternative's syntax should be a nested CHOICE
	inner, ok := outer.Alternatives[0].Syntax.(*ast.TypeSyntaxChoice)
	testutil.True(t, ok, "expected nested TypeSyntaxChoice, got %T", outer.Alternatives[0].Syntax)
	testutil.Len(t, inner.Alternatives, 2, "inner alternatives count")
	testutil.Equal(t, "x", inner.Alternatives[0].Name.Name, "inner first alternative")
	testutil.Equal(t, "y", inner.Alternatives[1].Name.Name, "inner second alternative")
}

func TestParseChoiceInObjectTypeSyntax(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX CHOICE {
				alpha INTEGER,
				beta OCTET STRING
			}
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test"
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])

	choice, ok := def.Syntax.Syntax.(*ast.TypeSyntaxChoice)
	testutil.True(t, ok, "expected TypeSyntaxChoice, got %T", def.Syntax.Syntax)
	testutil.Len(t, choice.Alternatives, 2, "alternatives count")
}
