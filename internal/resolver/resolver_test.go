package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

func TestResolveNilModulesNilLoggerNilConfig(t *testing.T) {
	m := Resolve(nil, nil, nil)
	if m == nil {
		t.Fatal("Resolve returned nil Mib")
	}
	// Should have base modules registered even with nil input.
	if m.ModuleCount() == 0 {
		t.Error("expected at least base modules, got 0")
	}
}

func TestResolveEmptyModulesNilLoggerNilConfig(t *testing.T) {
	m := Resolve([]*module.Module{}, nil, nil)
	if m == nil {
		t.Fatal("Resolve returned nil Mib")
	}
	// Empty user modules still gets base modules.
	if m.ModuleCount() == 0 {
		t.Error("expected at least base modules, got 0")
	}
}

func TestResolveNilModulesWithCustomConfig(t *testing.T) {
	cfg := mib.StrictConfig()
	m := Resolve(nil, nil, &cfg)
	if m == nil {
		t.Fatal("Resolve returned nil Mib")
	}
	// With strict config, the pipeline should still complete.
	if m.ModuleCount() == 0 {
		t.Error("expected at least base modules, got 0")
	}
}

func TestResolveBaseModulesRegistered(t *testing.T) {
	m := Resolve(nil, nil, nil)

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
		if m.Module(name) == nil {
			t.Errorf("base module %q not found in Mib", name)
		}
	}
}

func TestResolveBaseModulePrimitiveTypes(t *testing.T) {
	m := Resolve(nil, nil, nil)

	// The 4 ASN.1 primitives should be seeded in SNMPv2-SMI.
	primitives := []struct {
		name string
		base mib.BaseType
	}{
		{"INTEGER", mib.BaseInteger32},
		{"OCTET STRING", mib.BaseOctetString},
		{"OBJECT IDENTIFIER", mib.BaseObjectIdentifier},
		{"BITS", mib.BaseBits},
	}

	smiMod := m.Module("SNMPv2-SMI")
	if smiMod == nil {
		t.Fatal("SNMPv2-SMI module not found")
	}

	for _, p := range primitives {
		typ := smiMod.Type(p.name)
		if typ == nil {
			t.Errorf("primitive type %q not found in SNMPv2-SMI", p.name)
			continue
		}
		if typ.Base() != p.base {
			t.Errorf("primitive type %q base = %v, want %v", p.name, typ.Base(), p.base)
		}
	}
}

func TestResolveBaseModuleNodes(t *testing.T) {
	m := Resolve(nil, nil, nil)

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
		nd := m.FindNode(name)
		if nd == nil {
			t.Errorf("base node %q not found", name)
			continue
		}
		if len(nd.OID()) == 0 {
			t.Errorf("base node %q has empty OID", name)
		}
	}
}

func TestResolveBaseModuleNodeOIDValues(t *testing.T) {
	m := Resolve(nil, nil, nil)

	tests := []struct {
		name string
		oid  mib.Oid
	}{
		{"iso", mib.Oid{1}},
		{"org", mib.Oid{1, 3}},
		{"dod", mib.Oid{1, 3, 6}},
		{"internet", mib.Oid{1, 3, 6, 1}},
		{"mgmt", mib.Oid{1, 3, 6, 1, 2}},
		{"mib-2", mib.Oid{1, 3, 6, 1, 2, 1}},
		{"enterprises", mib.Oid{1, 3, 6, 1, 4, 1}},
	}

	for _, tt := range tests {
		nd := m.FindNode(tt.name)
		if nd == nil {
			t.Errorf("node %q not found", tt.name)
			continue
		}
		got := nd.OID()
		if len(got) != len(tt.oid) {
			t.Errorf("node %q OID length = %d, want %d", tt.name, len(got), len(tt.oid))
			continue
		}
		for i := range got {
			if got[i] != tt.oid[i] {
				t.Errorf("node %q OID[%d] = %d, want %d", tt.name, i, got[i], tt.oid[i])
			}
		}
	}
}

func TestResolveBaseModuleSMITypes(t *testing.T) {
	m := Resolve(nil, nil, nil)

	// SMI types defined as TypeDefs in SNMPv2-SMI should be resolved.
	smiTypes := []struct {
		name string
		base mib.BaseType
	}{
		{"Integer32", mib.BaseInteger32},
		{"Counter32", mib.BaseCounter32},
		{"Counter64", mib.BaseCounter64},
		{"Gauge32", mib.BaseGauge32},
		{"Unsigned32", mib.BaseUnsigned32},
		{"TimeTicks", mib.BaseTimeTicks},
		{"IpAddress", mib.BaseIpAddress},
		{"Opaque", mib.BaseOpaque},
	}

	for _, tt := range smiTypes {
		typ := m.FindType(tt.name)
		if typ == nil {
			t.Errorf("SMI type %q not found", tt.name)
			continue
		}
		if typ.Base() != tt.base {
			t.Errorf("SMI type %q base = %v, want %v", tt.name, typ.Base(), tt.base)
		}
	}
}

func TestResolveUnresolvedImportProducesDiagnostic(t *testing.T) {
	// Create a module that imports a symbol from a non-existent module.
	mod := module.NewModule("BAD-IMPORT-MIB", types.Span{})
	mod.Language = module.LanguageSMIv2
	mod.Imports = []module.Import{
		module.NewImport("NONEXISTENT-MIB", "fakeObject", types.Span{}),
	}

	m := Resolve([]*module.Module{mod}, nil, nil)
	if m == nil {
		t.Fatal("Resolve returned nil Mib")
	}

	// The module should still be registered.
	if m.Module("BAD-IMPORT-MIB") == nil {
		t.Error("BAD-IMPORT-MIB not found in resolved Mib")
	}

	// Should have unresolved references.
	unresolved := m.Unresolved()
	if len(unresolved) == 0 {
		t.Fatal("expected unresolved references, got none")
	}

	found := false
	for _, u := range unresolved {
		if u.Kind == mib.UnresolvedImport && u.Symbol == "fakeObject" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected unresolved import for fakeObject, got: %v", unresolved)
	}

	// Diagnostics should contain the import failure with module-not-found code.
	diags := m.Diagnostics()
	foundDiag := false
	for _, d := range diags {
		if d.Code == "import-module-not-found" && d.Module == "BAD-IMPORT-MIB" {
			foundDiag = true
			break
		}
	}
	if !foundDiag {
		t.Error("expected diagnostic with code import-module-not-found for BAD-IMPORT-MIB")
	}
}

func TestResolvePermissiveConfig(t *testing.T) {
	cfg := mib.PermissiveConfig()
	m := Resolve(nil, nil, &cfg)
	if m == nil {
		t.Fatal("Resolve returned nil Mib")
	}
	// Should still produce base modules.
	if m.ModuleCount() == 0 {
		t.Error("expected at least base modules")
	}
}

func TestResolveNoUserModulesNodeCount(t *testing.T) {
	m := Resolve(nil, nil, nil)
	// Base modules define OID nodes (iso, org, dod, internet, etc.).
	// There should be a reasonable number of nodes from base modules alone.
	if m.NodeCount() == 0 {
		t.Error("expected non-zero node count from base modules")
	}
}

func TestResolveNoUserModulesTypeCount(t *testing.T) {
	m := Resolve(nil, nil, nil)
	// At minimum: 4 ASN.1 primitives + SMI types + TCs
	if m.TypeCount() < 4 {
		t.Errorf("expected at least 4 types (ASN.1 primitives), got %d", m.TypeCount())
	}
}

func TestResolveNoUserModulesHasNoUnresolved(t *testing.T) {
	// With only base modules and no user modules, there should be no
	// unresolved references.
	m := Resolve(nil, nil, nil)
	if len(m.Unresolved()) != 0 {
		t.Errorf("expected no unresolved references for base-only resolution, got: %v", m.Unresolved())
	}
}

func TestResolveBaseOnlyHasNoErrors(t *testing.T) {
	m := Resolve(nil, nil, nil)
	if m.HasErrors() {
		t.Errorf("expected no errors for base-only resolution, diagnostics: %v", m.Diagnostics())
	}
}

func TestResolveUserModuleDuplicatingBaseModuleIsDropped(t *testing.T) {
	// If a user module has the same name as a base module, the base module
	// takes priority and the user module is dropped.
	userMod := module.NewModule("SNMPv2-SMI", types.Span{})
	userMod.Language = module.LanguageSMIv2

	m := Resolve([]*module.Module{userMod}, nil, nil)
	if m == nil {
		t.Fatal("Resolve returned nil Mib")
	}

	// The SNMPv2-SMI module should still have its types (from the real base).
	smiMod := m.Module("SNMPv2-SMI")
	if smiMod == nil {
		t.Fatal("SNMPv2-SMI not found")
	}
	if len(smiMod.Types()) == 0 {
		t.Error("SNMPv2-SMI should have types from the base module, not the empty user module")
	}
}
