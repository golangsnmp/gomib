package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestParseMultipleModules(t *testing.T) {
	source := `
A-MIB DEFINITIONS ::= BEGIN
aRoot OBJECT IDENTIFIER ::= { 1 3 6 1 4 1 99990 }
END

B-MIB DEFINITIONS ::= BEGIN
IMPORTS aRoot FROM A-MIB;
bRoot OBJECT IDENTIFIER ::= { aRoot 1 }
END
`
	p := New([]byte(source), nil, types.PermissiveConfig())
	mods := p.ParseModule()

	testutil.Equal(t, 2, len(mods), "should parse two modules")
	testutil.Equal(t, "A-MIB", mods[0].Name.Name, "first module name")
	testutil.Equal(t, "B-MIB", mods[1].Name.Name, "second module name")

	testutil.Greater(t, len(mods[0].Body), 0, "first module should have definitions")
	testutil.Greater(t, len(mods[1].Body), 0, "second module should have definitions")
	testutil.Equal(t, 1, len(mods[1].Imports), "second module should have one import")
}

func TestParseSingleModuleUnchanged(t *testing.T) {
	source := `
X-MIB DEFINITIONS ::= BEGIN
xRoot OBJECT IDENTIFIER ::= { 1 3 6 1 4 1 99991 }
END
`
	p := New([]byte(source), nil, types.PermissiveConfig())
	mods := p.ParseModule()

	testutil.Equal(t, 1, len(mods), "single module file should return one module")
	testutil.Equal(t, "X-MIB", mods[0].Name.Name, "module name")
}

func TestParseMultiModuleDiagnosticsIsolated(t *testing.T) {
	// First module has an underscore identifier (generates diagnostic in strict mode).
	// Second module is clean. Diagnostics should be isolated.
	source := `
A_MIB DEFINITIONS ::= BEGIN
a_root OBJECT IDENTIFIER ::= { 1 3 6 1 4 1 99992 }
END

CLEAN-MIB DEFINITIONS ::= BEGIN
cleanRoot OBJECT IDENTIFIER ::= { 1 3 6 1 4 1 99993 }
END
`
	p := New([]byte(source), nil, types.StrictConfig())
	mods := p.ParseModule()

	testutil.Equal(t, 2, len(mods), "should parse two modules")

	// First module should have underscore diagnostics.
	underscores := countDiagnostics(mods[0].Diagnostics, types.DiagIdentifierUnderscore)
	testutil.Greater(t, underscores, 0, "first module should have underscore diagnostics")

	// Second module should be clean of underscore diagnostics.
	underscores = countDiagnostics(mods[1].Diagnostics, types.DiagIdentifierUnderscore)
	testutil.Equal(t, 0, underscores, "second module should have no underscore diagnostics")
}
