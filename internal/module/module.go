package module

import (
	"iter"

	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

// Module is a normalized MIB module.
type Module struct {
	// Name is the module name.
	Name string
	// Language is the detected SMI language.
	Language Language
	// Imports are the imports (flattened to individual symbols).
	Imports []Import
	// Definitions are the definitions.
	Definitions []Definition
	// Span is the source span for diagnostics.
	Span types.Span
	// Diagnostics are the lowering diagnostics.
	Diagnostics []mib.Diagnostic
}

// NewModule creates a new module.
func NewModule(name string, span types.Span) *Module {
	return &Module{
		Name:        name,
		Language:    LanguageUnknown,
		Imports:     nil,
		Definitions: nil,
		Span:        span,
		Diagnostics: nil,
	}
}

// HasErrors returns true if this module has any error diagnostics.
func (m *Module) HasErrors() bool {
	for _, d := range m.Diagnostics {
		if d.Severity <= mib.SeverityError {
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

// Import is a single imported symbol from a module.
// Each import is flattened from the AST's grouped format.
type Import struct {
	// Module is the source module name.
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
