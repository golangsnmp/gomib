// Package ast provides Abstract Syntax Tree types for parsed MIB modules.
//
// The AST captures syntactic structure as-written, preserving source locations
// for diagnostics. Semantic analysis (resolution, normalization) happens in
// later phases (module lowering and resolver).
package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// Ident is an identifier with source location.
//
// Identifiers in SMI are case-sensitive. Uppercase identifiers denote
// module names and type references; lowercase identifiers denote object
// names and enum labels.
type Ident struct {
	// Name is the identifier text.
	Name string
	// Span is the source location.
	Span types.Span
}

// NewIdent creates a new identifier.
func NewIdent(name string, span types.Span) Ident {
	return Ident{Name: name, Span: span}
}

// IsUppercase returns true if this is an uppercase identifier (module/type name).
func (i Ident) IsUppercase() bool {
	if len(i.Name) == 0 {
		return false
	}
	c := i.Name[0]
	return c >= 'A' && c <= 'Z'
}

// IsLowercase returns true if this is a lowercase identifier (object/enum name).
func (i Ident) IsLowercase() bool {
	if len(i.Name) == 0 {
		return false
	}
	c := i.Name[0]
	return c >= 'a' && c <= 'z'
}

// QuotedString is a quoted string literal with source location.
//
// The value contains the string content with quotes stripped.
// MIB strings can contain non-ASCII characters (often Latin-1 encoded).
type QuotedString struct {
	// Value is the string content (quotes stripped).
	Value string
	// Span is the source location (includes quotes).
	Span types.Span
}

// NewQuotedString creates a new quoted string.
func NewQuotedString(value string, span types.Span) QuotedString {
	return QuotedString{Value: value, Span: span}
}

// NamedNumber is a named number in an enumeration or BITS definition.
//
// Examples:
//   - up(1) in INTEGER { up(1), down(2) }
//   - bit0(0) in BITS { bit0(0), bit1(1) }
type NamedNumber struct {
	// Name is the label name.
	Name Ident
	// Value is the numeric value.
	Value int64
	// Span is the source location (covers name(value)).
	Span types.Span
}

// NewNamedNumber creates a new named number.
func NewNamedNumber(name Ident, value int64, span types.Span) NamedNumber {
	return NamedNumber{Name: name, Value: value, Span: span}
}
