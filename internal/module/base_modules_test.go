package module

import (
	"testing"
)

func TestBaseModuleCount(t *testing.T) {
	modules := AllBaseModules()
	if len(modules) != 7 {
		t.Errorf("got %d base modules, want 7", len(modules))
	}
}

func TestBaseModuleRoundtrip(t *testing.T) {
	for _, m := range AllBaseModules() {
		name := m.Name()
		parsed, ok := BaseModuleFromName(name)
		if !ok {
			t.Errorf("BaseModuleFromName(%q) returned false", name)
			continue
		}
		if parsed != m {
			t.Errorf("roundtrip failed for %s: got %v, want %v", name, parsed, m)
		}
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
		if got != tt.expected {
			t.Errorf("IsBaseModule(%q) = %v, want %v", tt.name, got, tt.expected)
		}
	}
}

func TestCreateBaseModules(t *testing.T) {
	modules := CreateBaseModules()
	if len(modules) != 7 {
		t.Fatalf("got %d modules, want 7", len(modules))
	}

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
		if modules[i].Name != name {
			t.Errorf("modules[%d].Name = %q, want %q", i, modules[i].Name, name)
		}
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
		if !defNames[name] {
			t.Errorf("SNMPv2-SMI missing OID definition: %s", name)
		}
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
		if !defNames[name] {
			t.Errorf("SNMPv2-SMI missing base type: %s", name)
		}
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
		if !defNames[name] {
			t.Errorf("SNMPv2-TC missing textual convention: %s", name)
		}
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
			if len(va.Oid.Components) != 1 {
				t.Errorf("%s has %d components, want 1", exp.name, len(va.Oid.Components))
				continue
			}
			num, ok := va.Oid.Components[0].(*OidComponentNumber)
			if !ok {
				t.Errorf("%s component is not a number", exp.name)
				continue
			}
			if num.Value != exp.arcNum {
				t.Errorf("%s arc = %d, want %d", exp.name, num.Value, exp.arcNum)
			}
		}
		if !found {
			t.Errorf("root OID %s not found", exp.name)
		}
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
		if !defNames[name] {
			t.Errorf("RFC1155-SMI missing type: %s", name)
		}
	}

	// Check OID roots
	expectedOids := []string{"internet", "enterprises"}
	for _, name := range expectedOids {
		if !defNames[name] {
			t.Errorf("RFC1155-SMI missing OID: %s", name)
		}
	}
}

func TestEmptyModules(t *testing.T) {
	modules := CreateBaseModules()

	// SNMPv2-CONF, RFC-1212, RFC-1215 should have no definitions
	emptyModules := []int{2, 5, 6} // indices

	for _, idx := range emptyModules {
		if len(modules[idx].Definitions) != 0 {
			t.Errorf("%s has %d definitions, want 0",
				modules[idx].Name, len(modules[idx].Definitions))
		}
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

		if len(va.Oid.Components) != 2 {
			t.Errorf("enterprises has %d components, want 2", len(va.Oid.Components))
			return
		}

		nameComp, ok := va.Oid.Components[0].(*OidComponentName)
		if !ok {
			t.Errorf("enterprises[0] is not a name component")
			return
		}
		if nameComp.NameValue != "private" {
			t.Errorf("enterprises[0] = %q, want %q", nameComp.NameValue, "private")
		}

		numComp, ok := va.Oid.Components[1].(*OidComponentNumber)
		if !ok {
			t.Errorf("enterprises[1] is not a number component")
			return
		}
		if numComp.Value != 1 {
			t.Errorf("enterprises[1] = %d, want 1", numComp.Value)
		}
		return
	}
	t.Error("enterprises not found")
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
		if tt.module.IsSMIv1() != tt.isSMIv1 {
			t.Errorf("%s.IsSMIv1() = %v, want %v", tt.module.Name(), tt.module.IsSMIv1(), tt.isSMIv1)
		}
		if tt.module.IsSMIv2() != tt.isSMIv2 {
			t.Errorf("%s.IsSMIv2() = %v, want %v", tt.module.Name(), tt.module.IsSMIv2(), tt.isSMIv2)
		}
	}
}
