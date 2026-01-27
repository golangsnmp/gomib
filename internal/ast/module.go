package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// Module is a parsed MIB module.
type Module struct {
	Name            Ident
	DefinitionsKind DefinitionsKind
	Imports         []ImportClause
	Exports         *ExportsClause
	Body            []Definition
	Span            types.Span
	Diagnostics     []types.Diagnostic
}

// NewModule creates a new module.
func NewModule(name Ident, definitionsKind DefinitionsKind, span types.Span) *Module {
	return &Module{
		Name:            name,
		DefinitionsKind: definitionsKind,
		Imports:         nil,
		Exports:         nil,
		Body:            nil,
		Span:            span,
		Diagnostics:     nil,
	}
}

// HasErrors returns true if this module has parse errors.
func (m *Module) HasErrors() bool {
	for _, d := range m.Diagnostics {
		if d.Severity <= types.SeverityError {
			return true
		}
	}
	return false
}

// DefinitionsKind is the kind of module definition.
type DefinitionsKind int

const (
	DefinitionsKindDefinitions DefinitionsKind = iota
	DefinitionsKindPibDefinitions
)

// ImportClause is an import clause.
type ImportClause struct {
	Symbols    []Ident
	FromModule Ident
	Span       types.Span
}

// NewImportClause creates a new import clause.
func NewImportClause(symbols []Ident, fromModule Ident, span types.Span) ImportClause {
	return ImportClause{
		Symbols:    symbols,
		FromModule: fromModule,
		Span:       span,
	}
}

// ExportsClause is an exports clause (SMIv1 only).
type ExportsClause struct {
	Span types.Span
}
