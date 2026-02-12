// Package ast provides Abstract Syntax Tree types for parsed MIB modules.
package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// Ident represents a named reference in MIB source with its span.
type Ident struct {
	Name string
	Span types.Span
}

// NewIdent creates an Ident from a name and source span.
func NewIdent(name string, span types.Span) Ident {
	return Ident{Name: name, Span: span}
}

// IsUppercase reports whether the identifier starts with A-Z.
func (i Ident) IsUppercase() bool {
	if len(i.Name) == 0 {
		return false
	}
	c := i.Name[0]
	return c >= 'A' && c <= 'Z'
}

// IsLowercase reports whether the identifier starts with a-z.
func (i Ident) IsLowercase() bool {
	if len(i.Name) == 0 {
		return false
	}
	c := i.Name[0]
	return c >= 'a' && c <= 'z'
}

// QuotedString holds a string literal value with its source span.
type QuotedString struct {
	Value string
	Span  types.Span
}

// NewQuotedString creates a QuotedString from a value and source span.
func NewQuotedString(value string, span types.Span) QuotedString {
	return QuotedString{Value: value, Span: span}
}

// NamedNumber is a named number in an enumeration or BITS definition.
type NamedNumber struct {
	Name  Ident
	Value int64
	Span  types.Span
}

// NewNamedNumber creates a NamedNumber from its components.
func NewNamedNumber(name Ident, value int64, span types.Span) NamedNumber {
	return NamedNumber{Name: name, Value: value, Span: span}
}
