package module

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestBaseModuleCount(t *testing.T) {
	modules := AllBaseModules()
	testutil.Len(t, modules, 7, "base modules count")
}

func TestBaseModuleRoundtrip(t *testing.T) {
	for _, m := range AllBaseModules() {
		name := m.Name()
		parsed, ok := BaseModuleFromName(name)
		testutil.True(t, ok, "BaseModuleFromName(%q) returned false", name)
		testutil.Equal(t, m, parsed, "roundtrip failed for %s", name)
	}
}

func TestIsBaseModule(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"SNMPv2-SMI", true},
		{"SNMPv2-TC", true},
		{"SNMPv2-CONF", true},
		{"RFC1155-SMI", true},
		{"RFC1065-SMI", true},
		{"RFC-1212", true},
		{"RFC-1215", true},
		{"IF-MIB", false},
		{"TCP-MIB", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsBaseModule(tt.name)
		testutil.Equal(t, tt.expected, got, "IsBaseModule(%q)", tt.name)
	}
}

func TestCreateBaseModules(t *testing.T) {
	modules := CreateBaseModules()
	testutil.Len(t, modules, 7, "modules count")

	expectedNames := []string{
		"SNMPv2-SMI",
		"SNMPv2-TC",
		"SNMPv2-CONF",
		"RFC1155-SMI",
		"RFC1065-SMI",
		"RFC-1212",
		"RFC-1215",
	}

	for i, name := range expectedNames {
		testutil.Equal(t, name, modules[i].Name, "modules[%d].Name", i)
	}
}

func TestSNMPv2SMIHasOidDefinitions(t *testing.T) {
	modules := CreateBaseModules()
	smi := modules[0] // SNMPv2-SMI

	expectedOids := []string{
		"ccitt", "iso", "joint-iso-ccitt",
		"internet", "enterprises", "mib-2", "zeroDotZero",
	}

	defNames := make(map[string]bool)
	for _, def := range smi.Definitions {
		defNames[def.DefinitionName()] = true
	}

	for _, name := range expectedOids {
		testutil.True(t, defNames[name], "SNMPv2-SMI missing OID definition: %s", name)
	}
}

func TestSNMPv2SMIHasBaseTypes(t *testing.T) {
	modules := CreateBaseModules()
	smi := modules[0] // SNMPv2-SMI

	expectedTypes := []string{
		"Integer32", "Counter32", "Counter64", "Gauge32",
		"Unsigned32", "TimeTicks", "IpAddress", "Opaque",
	}

	defNames := make(map[string]bool)
	for _, def := range smi.Definitions {
		defNames[def.DefinitionName()] = true
	}

	for _, name := range expectedTypes {
		testutil.True(t, defNames[name], "SNMPv2-SMI missing base type: %s", name)
	}
}

func TestSNMPv2TCHasTCs(t *testing.T) {
	modules := CreateBaseModules()
	tc := modules[1] // SNMPv2-TC

	expectedTCs := []string{
		"DisplayString", "TruthValue", "RowStatus", "MacAddress",
	}

	defNames := make(map[string]bool)
	for _, def := range tc.Definitions {
		defNames[def.DefinitionName()] = true
	}

	for _, name := range expectedTCs {
		testutil.True(t, defNames[name], "SNMPv2-TC missing textual convention: %s", name)
	}
}

func TestRootOidArcs(t *testing.T) {
	modules := CreateBaseModules()
	smi := modules[0] // SNMPv2-SMI

	// Find root OIDs and verify their numeric values
	type oidCheck struct {
		name   string
		arcNum uint32
	}

	expected := []oidCheck{
		{"ccitt", 0},
		{"iso", 1},
		{"joint-iso-ccitt", 2},
	}

	for _, exp := range expected {
		found := false
		for _, def := range smi.Definitions {
			va, ok := def.(*ValueAssignment)
			if !ok || va.Name != exp.name {
				continue
			}
			found = true
			testutil.Len(t, va.Oid.Components, 1, "%s components", exp.name)
			num, ok := va.Oid.Components[0].(*OidComponentNumber)
			testutil.True(t, ok, "%s component is not a number", exp.name)
			testutil.Equal(t, exp.arcNum, num.Value, "%s arc", exp.name)
		}
		testutil.True(t, found, "root OID %s not found", exp.name)
	}
}

func TestRFC1155SMIHasTypes(t *testing.T) {
	modules := CreateBaseModules()
	rfc1155 := modules[3] // RFC1155-SMI

	expectedTypes := []string{
		"Counter", "Gauge", "NetworkAddress", "IpAddress", "TimeTicks", "Opaque",
	}

	defNames := make(map[string]bool)
	for _, def := range rfc1155.Definitions {
		defNames[def.DefinitionName()] = true
	}

	for _, name := range expectedTypes {
		testutil.True(t, defNames[name], "RFC1155-SMI missing type: %s", name)
	}

	// Check OID roots
	expectedOids := []string{"internet", "enterprises"}
	for _, name := range expectedOids {
		testutil.True(t, defNames[name], "RFC1155-SMI missing OID: %s", name)
	}
}

func TestEmptyModules(t *testing.T) {
	modules := CreateBaseModules()

	// SNMPv2-CONF, RFC-1212, RFC-1215 should have no definitions
	emptyModules := []int{2, 5, 6} // indices

	for _, idx := range emptyModules {
		testutil.Len(t, modules[idx].Definitions, 0, "%s definitions", modules[idx].Name)
	}
}

func TestOidChain(t *testing.T) {
	modules := CreateBaseModules()
	smi := modules[0] // SNMPv2-SMI

	// Find enterprises definition - should have { private 1 }
	for _, def := range smi.Definitions {
		va, ok := def.(*ValueAssignment)
		if !ok || va.Name != "enterprises" {
			continue
		}

		testutil.Len(t, va.Oid.Components, 2, "enterprises components")

		nameComp, ok := va.Oid.Components[0].(*OidComponentName)
		testutil.True(t, ok, "enterprises[0] is not a name component")
		testutil.Equal(t, "private", nameComp.NameValue, "enterprises[0]")

		numComp, ok := va.Oid.Components[1].(*OidComponentNumber)
		testutil.True(t, ok, "enterprises[1] is not a number component")
		testutil.Equal(t, uint32(1), numComp.Value, "enterprises[1]")
		return
	}
	testutil.Fail(t, "enterprises not found")
}

func TestBaseModuleSMIVersions(t *testing.T) {
	tests := []struct {
		module  BaseModule
		isSMIv1 bool
		isSMIv2 bool
	}{
		{BaseModuleSNMPv2SMI, false, true},
		{BaseModuleSNMPv2TC, false, true},
		{BaseModuleSNMPv2CONF, false, true},
		{BaseModuleRFC1155SMI, true, false},
		{BaseModuleRFC1065SMI, true, false},
		{BaseModuleRFC1212, true, false},
		{BaseModuleRFC1215, true, false},
	}

	for _, tt := range tests {
		testutil.Equal(t, tt.isSMIv1, tt.module.IsSMIv1(), "%s.IsSMIv1()", tt.module.Name())
		testutil.Equal(t, tt.isSMIv2, tt.module.IsSMIv2(), "%s.IsSMIv2()", tt.module.Name())
	}
}
