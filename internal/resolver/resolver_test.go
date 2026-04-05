package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestResolveNilOrEmptyInput(t *testing.T) {
	t.Run("nil modules", func(t *testing.T) {
		m := Resolve(nil, nil, nil, nil)
		testutil.NotNil(t, m, "Resolve returned nil Mib")
		testutil.NotEmpty(t, m.Modules(), "expected at least base modules, got 0")
	})

	t.Run("empty modules", func(t *testing.T) {
		m := Resolve([]*module.Module{}, nil, nil, nil)
		testutil.NotNil(t, m, "Resolve returned nil Mib")
		testutil.NotEmpty(t, m.Modules(), "expected at least base modules, got 0")
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := model.VerboseConfig()
		m := Resolve(nil, nil, nil, &cfg)
		testutil.NotNil(t, m, "Resolve returned nil Mib")
		testutil.NotEmpty(t, m.Modules(), "expected at least base modules, got 0")
	})
}

func TestResolveBaseModulesRegistered(t *testing.T) {
	m := Resolve(nil, nil, nil, nil)

	expectedModules := []string{
		"SNMPv2-SMI",
		"SNMPv2-TC",
		"SNMPv2-CONF",
		"RFC1155-SMI",
		"RFC1065-SMI",
		"RFC-1212",
		"RFC-1215",
	}
	for _, name := range expectedModules {
		testutil.NotNil(t, m.Module(name), "base module  not found in Mib")
	}
}

func TestResolveBaseModulePrimitiveTypes(t *testing.T) {
	m := Resolve(nil, nil, nil, nil)

	// The 4 ASN.1 primitives should be seeded in SNMPv2-SMI.
	primitives := []struct {
		name string
		base model.BaseType
	}{
		{"INTEGER", model.BaseInteger32},
		{"OCTET STRING", model.BaseOctetString},
		{"OBJECT IDENTIFIER", model.BaseObjectIdentifier},
		{"BITS", model.BaseBits},
	}

	smiMod := m.Module("SNMPv2-SMI")
	testutil.NotNil(t, smiMod, "SNMPv2-SMI module not found")

	for _, p := range primitives {
		typ := smiMod.Type(p.name)
		testutil.NotNil(t, typ, "primitive type %q not found in SNMPv2-SMI", p.name)
		testutil.Equal(t, p.base, typ.Base(), "primitive type  base")
	}
}

func TestResolveBaseModuleNodes(t *testing.T) {
	m := Resolve(nil, nil, nil, nil)

	// Base modules define well-known OID roots.
	expectedNodes := []string{
		"iso",
		"org",
		"dod",
		"internet",
		"mgmt",
		"mib-2",
		"enterprises",
		"private",
		"experimental",
	}
	for _, name := range expectedNodes {
		nd := m.Node(name)
		testutil.NotNil(t, nd, "base node %q not found", name)
		testutil.NotEmpty(t, nd.OID(), "base node  has empty OID")
	}
}

func TestResolveBaseModuleNodeOIDValues(t *testing.T) {
	m := Resolve(nil, nil, nil, nil)

	tests := []struct {
		name string
		oid  model.OID
	}{
		{"iso", model.OID{1}},
		{"org", model.OID{1, 3}},
		{"dod", model.OID{1, 3, 6}},
		{"internet", model.OID{1, 3, 6, 1}},
		{"mgmt", model.OID{1, 3, 6, 1, 2}},
		{"mib-2", model.OID{1, 3, 6, 1, 2, 1}},
		{"enterprises", model.OID{1, 3, 6, 1, 4, 1}},
	}

	for _, tt := range tests {
		nd := m.Node(tt.name)
		testutil.NotNil(t, nd, "node %q not found", tt.name)
		got := nd.OID()
		testutil.Equal(t, len(tt.oid), len(got), "node %q OID length", tt.name)
		for i := range got {
			testutil.Equal(t, tt.oid[i], got[i], "node  OID[]")
		}
	}
}

func TestResolveBaseModuleSMITypes(t *testing.T) {
	m := Resolve(nil, nil, nil, nil)

	// SMI types defined as TypeDefs in SNMPv2-SMI should be resolved.
	smiTypes := []struct {
		name string
		base model.BaseType
	}{
		{"Integer32", model.BaseInteger32},
		{"Counter32", model.BaseCounter32},
		{"Counter64", model.BaseCounter64},
		{"Gauge32", model.BaseGauge32},
		{"Unsigned32", model.BaseUnsigned32},
		{"TimeTicks", model.BaseTimeTicks},
		{"IpAddress", model.BaseIpAddress},
		{"Opaque", model.BaseOpaque},
	}

	for _, tt := range smiTypes {
		typ := m.Type(tt.name)
		testutil.NotNil(t, typ, "SMI type %q not found", tt.name)
		testutil.Equal(t, tt.base, typ.Base(), "SMI type  base")
	}
}

func TestResolveUnresolvedImportProducesDiagnostic(t *testing.T) {
	// Create a module that imports a symbol from a non-existent module.
	mod := module.NewModule("BAD-IMPORT-MIB", types.Span{})
	mod.Language = types.LanguageSMIv2
	mod.Imports = []module.Import{
		module.NewImport("NONEXISTENT-MIB", "fakeObject", types.Span{}),
	}

	m := Resolve([]*module.Module{mod}, nil, nil, nil)
	testutil.NotNil(t, m, "Resolve returned nil Mib")

	// The module should still be registered.
	testutil.NotNil(t, m.Module("BAD-IMPORT-MIB"), "BAD-IMPORT-MIB not found in resolved Mib")

	// Should have unresolved references.
	unresolved := m.Unresolved()
	testutil.NotEmpty(t, unresolved, "expected unresolved references, got none")

	found := false
	for _, u := range unresolved {
		if u.Kind == model.UnresolvedImport && u.Symbol == "fakeObject" {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected unresolved import for fakeObject, got:")

	// Diagnostics should contain the import failure with module-not-found code.
	diags := m.Diagnostics()
	foundDiag := false
	for _, d := range diags {
		if d.Code == "import-module-not-found" && d.Module == "BAD-IMPORT-MIB" {
			foundDiag = true
			break
		}
	}
	testutil.True(t, foundDiag, "expected diagnostic with code import-module-not-found for BAD-IMPORT-MIB")
}

func TestResolveNoUserModulesNodeCount(t *testing.T) {
	m := Resolve(nil, nil, nil, nil)
	// Base modules define OID nodes (iso, org, dod, internet, etc.).
	// There should be a reasonable number of nodes from base modules alone.
	testutil.Greater(t, m.NodeCount(), 0, "expected non-zero node count from base modules")
}

func TestResolveNoUserModulesTypeCount(t *testing.T) {
	m := Resolve(nil, nil, nil, nil)
	// At minimum: 4 ASN.1 primitives + SMI types + TCs
	testutil.True(t, len(m.Types()) >= 4, "expected at least 4 types (ASN.1 primitives), got %d", len(m.Types()))
}

func TestResolveBaseOnlyIsClean(t *testing.T) {
	m := Resolve(nil, nil, nil, nil)
	testutil.Len(t, m.Unresolved(), 0, "expected no unresolved references for base-only resolution, got")
	testutil.False(t, m.HasErrors(), "expected no errors for base-only resolution, diagnostics:")
}

func TestResolveUserModuleDuplicatingBaseModuleIsDropped(t *testing.T) {
	// If a user module has the same name as a base module, the base module
	// takes priority and the user module is dropped.
	userMod := module.NewModule("SNMPv2-SMI", types.Span{})
	userMod.Language = types.LanguageSMIv2

	m := Resolve([]*module.Module{userMod}, nil, nil, nil)
	testutil.NotNil(t, m, "Resolve returned nil Mib")

	// The SNMPv2-SMI module should still have its types (from the real base).
	smiMod := m.Module("SNMPv2-SMI")
	testutil.NotNil(t, smiMod, "SNMPv2-SMI not found")
	testutil.NotEmpty(t, smiMod.Types(), "SNMPv2-SMI should have types from the base module, not the empty user module")
}
