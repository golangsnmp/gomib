// Package module provides a normalized representation of MIB modules.
//
// This package transforms AST structures into a simplified module representation
// independent of whether the source was SMIv1 or SMIv2. Key transformations:
//
//   - Language detection from imports
//   - Import flattening (one symbol per import)
//   - Unified notification type (TRAP-TYPE and NOTIFICATION-TYPE)
//
// # What Lowering Does NOT Do (per V2 design)
//
// Status and Access values are preserved without normalization:
//   - STATUS: mandatory, optional preserved (not mapped to current/deprecated)
//   - ACCESS: SPPI values preserved (install, install-notify, report-only)
//   - ACCESS keyword preserved (ACCESS vs MAX-ACCESS vs PIB-ACCESS)
//
// These are resolver responsibilities:
//   - OID resolution (keeps OID components as symbols)
//   - Type resolution (keeps type references as symbols)
//   - Nodekind inference (requires resolved OID tree)
//   - Import resolution (just normalize; actual lookup is resolver's job)
//   - Built-in type injection
package module

import (
	"iter"

	"github.com/golangsnmp/gomib/internal/types"
)

// Module is a normalized, language-independent MIB module.
type Module struct {
	Name        string
	Language    types.Language
	Imports     []Import
	Definitions []Definition
	Span        types.Span
	Diagnostics []types.Diagnostic

	// LineTable maps line numbers to byte offsets of line starts.
	// Entry i holds the byte offset where line i+1 begins (0-indexed).
	// Used by the resolver to convert spans to line/column numbers
	// after the raw source bytes have been released.
	LineTable []int
}

// NewModule returns a Module with the given name and no definitions.
func NewModule(name string, span types.Span) *Module {
	return &Module{
		Name:        name,
		Language:    types.LanguageUnknown,
		Imports:     nil,
		Definitions: nil,
		Span:        span,
		Diagnostics: nil,
	}
}

// HasErrors reports whether this module has any error-level diagnostics.
func (m *Module) HasErrors() bool {
	for _, d := range m.Diagnostics {
		if d.Severity.AtLeast(types.SeverityError) {
			return true
		}
	}
	return false
}

// DefinitionNames returns an iterator over the names of all definitions.
func (m *Module) DefinitionNames() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, def := range m.Definitions {
			if !yield(def.DefinitionName()) {
				return
			}
		}
	}
}

// Import is a single imported symbol, flattened from the AST's grouped format.
type Import struct {
	Module string
	Symbol string
	Span   types.Span
}

// NewImport returns an Import for the given symbol from the given module.
func NewImport(module, symbol string, span types.Span) Import {
	return Import{
		Module: module,
		Symbol: symbol,
		Span:   span,
	}
}
