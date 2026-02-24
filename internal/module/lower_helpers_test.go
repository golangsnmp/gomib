package module

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/parser"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

// lowerAndFindDiagnostic parses source, lowers the AST, and returns the
// first diagnostic matching code. Returns nil if no matching diagnostic
// is found.
func lowerAndFindDiagnostic(t *testing.T, source []byte, config types.DiagnosticConfig, code string) *types.Diagnostic {
	t.Helper()

	p := parser.New(source, nil, config)
	ast := p.ParseModule()
	testutil.NotNil(t, ast, "parse returned nil")

	mod := Lower(ast, source, nil, config)
	testutil.NotNil(t, mod, "lower returned nil")

	for _, d := range mod.Diagnostics {
		if d.Code == code {
			return &d
		}
	}
	return nil
}
