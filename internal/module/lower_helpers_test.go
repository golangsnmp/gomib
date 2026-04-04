package module

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/parser"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

// lowerModule parses source and returns the lowered Module.
func lowerModule(t *testing.T, source string) *Module {
	t.Helper()
	b := []byte(source)
	p := parser.New(b, nil, types.VerboseConfig())
	mods := p.ParseModule()
	testutil.Greater(t, len(mods), 0, "parse returned no modules")
	mod := Lower(mods[0], b, nil, types.VerboseConfig())
	testutil.NotNil(t, mod, "lower returned nil")
	return mod
}

// requireDef finds the first definition of the given type with the given name.
// Fails the test if not found or wrong type.
func requireDef[T Definition](t *testing.T, mod *Module, name string) T {
	t.Helper()
	for _, d := range mod.Definitions {
		if d.DefinitionName() == name {
			v, ok := d.(T)
			if !ok {
				t.Fatalf("%s is %T, not %T", name, d, *new(T))
			}
			return v
		}
	}
	t.Fatalf("definition %q not found", name)
	return *new(T)
}

// lowerAndFindDiagnostic parses source, lowers the AST, and returns the
// first diagnostic matching code. Returns nil if no matching diagnostic
// is found.
func lowerAndFindDiagnostic(t *testing.T, source []byte, config types.DiagnosticConfig, code string) *types.Diagnostic {
	t.Helper()

	p := parser.New(source, nil, config)
	mods := p.ParseModule()
	testutil.Greater(t, len(mods), 0, "parse returned no modules")

	mod := Lower(mods[0], source, nil, config)
	testutil.NotNil(t, mod, "lower returned nil")

	for _, d := range mod.Diagnostics {
		if d.Code == code {
			return &d
		}
	}
	return nil
}
