package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestParseChoiceTypeAssignment(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestChoice ::= CHOICE {
			internet Integer32,
			raw DisplayString
		}
		END`)

	testutil.Len(t, mod.TypeDefs, 1, "definitions count")
	def := mod.TypeDefs[0]
	testutil.Equal(t, "TestChoice", def.Name, "type name")

	// CHOICE is normalized to the first alternative during parsing.
	ref, ok := def.Syntax.(*module.TypeSyntaxTypeRef)
	testutil.True(t, ok, "expected TypeSyntaxTypeRef (first alternative), got %T", def.Syntax)
	testutil.Equal(t, "Integer32", ref.Name, "first alternative type")
}

func TestParseChoiceWithBuiltinTypes(t *testing.T) {
	// Test CHOICE with builtin types like INTEGER and OCTET STRING
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestBuiltinChoice ::= CHOICE {
			num INTEGER,
			str OCTET STRING
		}
		END`)

	testutil.Len(t, mod.TypeDefs, 1, "definitions count")
	def := mod.TypeDefs[0]

	// CHOICE normalized to first alternative: INTEGER -> TypeSyntaxTypeRef
	ref, ok := def.Syntax.(*module.TypeSyntaxTypeRef)
	testutil.True(t, ok, "expected TypeSyntaxTypeRef (first alternative), got %T", def.Syntax)
	testutil.Equal(t, "INTEGER", ref.Name, "first alternative type")
}

func TestParseChoiceSingleAlternative(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		SingleChoice ::= CHOICE {
			only Integer32
		}
		END`)

	testutil.Len(t, mod.TypeDefs, 1, "definitions count")
	def := mod.TypeDefs[0]

	// Single alternative normalized to its type.
	ref, ok := def.Syntax.(*module.TypeSyntaxTypeRef)
	testutil.True(t, ok, "expected TypeSyntaxTypeRef, got %T", def.Syntax)
	testutil.Equal(t, "Integer32", ref.Name, "alternative type")
}

func TestParseChoiceAlternativeSyntax(t *testing.T) {
	// Verify CHOICE normalization picks the first alternative's syntax.
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestChoice ::= CHOICE {
			num INTEGER,
			str OCTET STRING,
			ref DisplayString
		}
		END`)

	testutil.Len(t, mod.TypeDefs, 1, "definitions count")
	def := mod.TypeDefs[0]

	// CHOICE is normalized to the first alternative (INTEGER).
	ref, ok := def.Syntax.(*module.TypeSyntaxTypeRef)
	testutil.True(t, ok, "expected TypeSyntaxTypeRef (normalized from CHOICE), got %T", def.Syntax)
	testutil.Equal(t, "INTEGER", ref.Name, "first alternative type")
}

func TestParseChoiceWithNamedNumbers(t *testing.T) {
	// CHOICE alternative with INTEGER that has named number values.
	// Normalized to the first alternative.
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestChoice ::= CHOICE {
			status INTEGER { up(1), down(2) },
			name OCTET STRING
		}
		END`)

	testutil.Len(t, mod.TypeDefs, 1, "definitions count")
	def := mod.TypeDefs[0]

	// First alternative is an INTEGER enum.
	intEnum, ok := def.Syntax.(*module.TypeSyntaxIntegerEnum)
	testutil.True(t, ok, "expected TypeSyntaxIntegerEnum (first alternative), got %T", def.Syntax)
	testutil.Len(t, intEnum.NamedNumbers, 2, "named numbers count")
}

func TestParseChoiceNested(t *testing.T) {
	// Nested CHOICE inside a CHOICE alternative.
	// Outer CHOICE normalizes to first alternative, which is an inner CHOICE
	// that also normalizes to its first alternative.
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestChoice ::= CHOICE {
			inner CHOICE {
				x INTEGER,
				y OCTET STRING
			},
			other Integer32
		}
		END`)

	testutil.Len(t, mod.TypeDefs, 1, "definitions count")
	def := mod.TypeDefs[0]

	// Outer CHOICE -> first alternative's syntax = inner CHOICE -> first alternative = INTEGER
	ref, ok := def.Syntax.(*module.TypeSyntaxTypeRef)
	testutil.True(t, ok, "expected TypeSyntaxTypeRef (nested CHOICE normalized), got %T", def.Syntax)
	testutil.Equal(t, "INTEGER", ref.Name, "doubly-nested first alternative type")
}

func TestParseChoiceInObjectTypeSyntax(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
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

	testutil.Len(t, mod.ObjectTypes, 1, "definitions count")
	def := mod.ObjectTypes[0]

	// CHOICE normalized to first alternative (INTEGER).
	ref, ok := def.Syntax.(*module.TypeSyntaxTypeRef)
	testutil.True(t, ok, "expected TypeSyntaxTypeRef (CHOICE normalized), got %T", def.Syntax)
	testutil.Equal(t, "INTEGER", ref.Name, "first alternative type")
}
