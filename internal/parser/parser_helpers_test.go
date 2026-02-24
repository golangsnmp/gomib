package parser

import (
	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/types"
)

// parseModule parses source with PermissiveConfig and returns the AST module.
func parseModule(source string) *ast.Module {
	return parseModuleWith(source, types.PermissiveConfig())
}

// parseModuleWith parses source with the given diagnostic config.
func parseModuleWith(source string, config types.DiagnosticConfig) *ast.Module {
	p := New([]byte(source), nil, config)
	return p.ParseModule()
}

// countDiagnostics returns how many diagnostics in the slice have the given code.
func countDiagnostics(diags []types.SpanDiagnostic, code string) int {
	n := 0
	for _, d := range diags {
		if d.Code == code {
			n++
		}
	}
	return n
}
