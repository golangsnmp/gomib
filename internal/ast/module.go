package ast

import (
	"slices"

	"github.com/golangsnmp/gomib/internal/types"
)

// Module is the top-level AST node for a parsed MIB module.
type Module struct {
	Name        Ident
	Imports     []ImportClause
	Body        []Definition
	Span        types.Span
	Diagnostics []types.SpanDiagnostic
}

// NewModule creates a Module with the given name and span.
func NewModule(name Ident, span types.Span) *Module {
	return &Module{
		Name: name,
		Span: span,
	}
}

// HasErrors reports whether any diagnostic has error severity or worse.
func (m *Module) HasErrors() bool {
	return slices.ContainsFunc(m.Diagnostics, func(d types.SpanDiagnostic) bool {
		return d.Severity.AtLeast(types.SeverityError)
	})
}

// ImportClause groups symbols imported from a single source module.
type ImportClause struct {
	Symbols    []Ident
	FromModule Ident
	Span       types.Span
}
