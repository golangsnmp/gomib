package ast

import (
	"github.com/golangsnmp/gomib/internal/types"
)

// Module is a parsed MIB module.
//
// Represents the top-level structure of a MIB file:
//
//	ModuleName DEFINITIONS ::= BEGIN
//	    IMPORTS ... ;
//	    <definitions>
//	END
type Module struct {
	// Name is the module name (e.g., IF-MIB, SNMPv2-SMI).
	Name Ident
	// DefinitionsKind is the kind of definitions (DEFINITIONS or PIB-DEFINITIONS).
	DefinitionsKind DefinitionsKind
	// Imports are the import clauses.
	Imports []ImportClause
	// Exports is the export clause (SMIv1 only, rare).
	Exports *ExportsClause
	// Body contains the module body definitions.
	Body []Definition
	// Span is the source location (entire module).
	Span types.Span
	// Diagnostics are the parse diagnostics (errors and warnings).
	Diagnostics []types.Diagnostic
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
		if d.Severity == types.SeverityError {
			return true
		}
	}
	return false
}

// DefinitionsKind is the kind of module definition.
type DefinitionsKind int

const (
	// DefinitionsKindDefinitions is a standard MIB module: DEFINITIONS ::= BEGIN
	DefinitionsKindDefinitions DefinitionsKind = iota
	// DefinitionsKindPibDefinitions is a SPPI PIB module: PIB-DEFINITIONS ::= BEGIN
	DefinitionsKindPibDefinitions
)

// ImportClause is an import clause specifying symbols imported from another module.
//
// Example:
//
//	IMPORTS
//	    MODULE-IDENTITY, OBJECT-TYPE
//	        FROM SNMPv2-SMI
//	    DisplayString
//	        FROM SNMPv2-TC;
//
// Each ImportClause represents one <symbols> FROM <module> group.
type ImportClause struct {
	// Symbols are the symbols being imported.
	Symbols []Ident
	// FromModule is the source module name.
	FromModule Ident
	// Span is the source location (covers <symbols> FROM <module>).
	Span types.Span
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
//
// The EXPORTS keyword is handled by the lexer skip state, so this type
// only records that an EXPORTS clause was present.
type ExportsClause struct {
	// Span is the source location.
	Span types.Span
}
