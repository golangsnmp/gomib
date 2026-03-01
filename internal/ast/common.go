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

// QuotedString holds a string literal value with its source span.
type QuotedString struct {
	Value string
	Span  types.Span
}

// NamedNumber is a named number in an enumeration or BITS definition.
type NamedNumber struct {
	Name  Ident
	Value int64
	Span  types.Span
}
