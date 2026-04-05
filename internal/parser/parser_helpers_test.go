package parser

import (
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// parseModule parses source with DefaultConfig and returns the first module.
func parseModule(source string) *module.Module {
	return parseModuleWith(source, types.DefaultConfig())
}

// parseModuleWith parses source with the given diagnostic config and returns
// the first module.
func parseModuleWith(source string, config types.DiagnosticConfig) *module.Module {
	p := New([]byte(source), nil, config)
	return p.ParseModule()[0]
}

// countDiagnostics returns how many diagnostics in the slice have the given code.
func countDiagnostics(diags []types.Diagnostic, code string) int {
	n := 0
	for _, d := range diags {
		if d.Code == code {
			n++
		}
	}
	return n
}
