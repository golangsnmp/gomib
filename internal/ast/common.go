// Package ast provides Abstract Syntax Tree types for parsed MIB modules.
package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// Ident is an identifier with source location.
type Ident struct {
	Name string
	Span types.Span
}

// NewIdent creates a new identifier.
func NewIdent(name string, span types.Span) Ident {
	return Ident{Name: name, Span: span}
}

// IsUppercase returns true if this is an uppercase identifier.
func (i Ident) IsUppercase() bool {
	if len(i.Name) == 0 {
		return false
	}
	c := i.Name[0]
	return c >= 'A' && c <= 'Z'
}

// IsLowercase returns true if this is a lowercase identifier.
func (i Ident) IsLowercase() bool {
	if len(i.Name) == 0 {
		return false
	}
	c := i.Name[0]
	return c >= 'a' && c <= 'z'
}

// QuotedString is a quoted string literal with source location.
type QuotedString struct {
	Value string
	Span  types.Span
}

// NewQuotedString creates a new quoted string.
func NewQuotedString(value string, span types.Span) QuotedString {
	return QuotedString{Value: value, Span: span}
}

// NamedNumber is a named number in an enumeration or BITS definition.
type NamedNumber struct {
	Name  Ident
	Value int64
	Span  types.Span
}

// NewNamedNumber creates a new named number.
func NewNamedNumber(name Ident, value int64, span types.Span) NamedNumber {
	return NamedNumber{Name: name, Value: value, Span: span}
}
