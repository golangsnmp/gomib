package module

import (
	"iter"

	"github.com/golangsnmp/gomib/internal/types"
)

// Module is a normalized MIB module.
type Module struct {
	// Name is the module name.
	Name string
	// Language is the detected SMI language.
	Language SmiLanguage
	// Imports are the imports.
	Imports []Import
	// Definitions are the definitions.
	Definitions []Definition
	// Span is the source span for diagnostics.
	Span types.Span
	// Diagnostics are the lowering diagnostics.
	Diagnostics []types.Diagnostic
}

// NewModule creates a new module.
func NewModule(name string, span types.Span) *Module {
	return &Module{
		Name:        name,
		Language:    SmiLanguageUnknown,
		Imports:     nil,
		Definitions: nil,
		Span:        span,
		Diagnostics: nil,
	}
}

// HasErrors returns true if this module has any error diagnostics.
func (m *Module) HasErrors() bool {
	for _, d := range m.Diagnostics {
		if d.Severity == types.SeverityError {
			return true
		}
	}
	return false
}

// DefinitionNames returns an iterator over all definition names.
func (m *Module) DefinitionNames() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, def := range m.Definitions {
			if !yield(def.DefinitionName()) {
				return
			}
		}
	}
}

// Import is an import.
//
// Each import is flattened to individual symbols.
type Import struct {
	// Module is the module name.
	Module string
	// Symbol is the symbol name.
	Symbol string
	// Span is the original source span.
	Span types.Span
}

// NewImport creates a new import.
func NewImport(module, symbol string, span types.Span) Import {
	return Import{
		Module: module,
		Symbol: symbol,
		Span:   span,
	}
}
