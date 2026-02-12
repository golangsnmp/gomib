package module

import (
	"iter"

	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

// Module is a normalized, language-independent MIB module.
type Module struct {
	Name        string
	Language    Language
	Imports     []Import
	Definitions []Definition
	Span        types.Span
	Diagnostics []mib.Diagnostic
}

// NewModule returns a Module with the given name and no definitions.
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

// HasErrors reports whether this module has any error-level diagnostics.
func (m *Module) HasErrors() bool {
	for _, d := range m.Diagnostics {
		if d.Severity <= mib.SeverityError {
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
