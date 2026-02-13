package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

// Module is the top-level AST node for a parsed MIB module.
type Module struct {
	Name            Ident
	DefinitionsKind DefinitionsKind
	Imports         []ImportClause
	Exports         *ExportsClause
	Body            []Definition
	Span            types.Span
	Diagnostics     []types.Diagnostic
}

// NewModule creates a Module with nil imports, body, and diagnostics.
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

// HasErrors reports whether any diagnostic has error severity or worse.
func (m *Module) HasErrors() bool {
	for _, d := range m.Diagnostics {
		if d.Severity <= mib.SeverityError {
			return true
		}
	}
	return false
}

// DefinitionsKind distinguishes DEFINITIONS from PIB-DEFINITIONS.
type DefinitionsKind int

const (
	DefinitionsKindDefinitions DefinitionsKind = iota
	DefinitionsKindPibDefinitions
)

// ImportClause groups symbols imported from a single source module.
type ImportClause struct {
	Symbols    []Ident
	FromModule Ident
	Span       types.Span
}

// NewImportClause creates an ImportClause from its components.
func NewImportClause(symbols []Ident, fromModule Ident, span types.Span) ImportClause {
	return ImportClause{
		Symbols:    symbols,
		FromModule: fromModule,
		Span:       span,
	}
}

// ExportsClause records the presence of an EXPORTS clause (SMIv1 only).
// The exported symbols are not tracked since EXPORTS is skipped.
type ExportsClause struct {
	Span types.Span
}
