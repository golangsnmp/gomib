package module

import (
	"slices"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestBaseModuleNames_DefensiveCopy(t *testing.T) {
	original := BaseModuleNames()
	saved := slices.Clone(original)

	// Mutate the returned slice.
	original[0] = "CORRUPTED"

	// A second call must still return the original names.
	got := BaseModuleNames()
	testutil.SliceEqual(t, saved, got, "BaseModuleNames() was mutated by caller")
}

func TestAllBaseModules(t *testing.T) {
	all := AllBaseModules()
	testutil.Len(t, all, 7, "AllBaseModules count")

	for i, bm := range all {
		testutil.Equal(t, BaseModule(i), bm, "AllBaseModules[%d]", i)
	}
}

func TestBaseModule_SMIClassification(t *testing.T) {
	smiv2 := []BaseModule{BaseModuleSNMPv2SMI, BaseModuleSNMPv2TC, BaseModuleSNMPv2CONF}
	smiv1 := []BaseModule{BaseModuleRFC1155SMI, BaseModuleRFC1065SMI, BaseModuleRFC1212, BaseModuleRFC1215}

	for _, bm := range smiv2 {
		testutil.True(t, bm.IsSMIv2(), "%s.IsSMIv2()", bm.Name())
		testutil.False(t, bm.IsSMIv1(), "%s.IsSMIv1()", bm.Name())
	}
	for _, bm := range smiv1 {
		testutil.True(t, bm.IsSMIv1(), "%s.IsSMIv1()", bm.Name())
		testutil.False(t, bm.IsSMIv2(), "%s.IsSMIv2()", bm.Name())
	}
}

func TestBaseModuleFromName(t *testing.T) {
	bm, ok := BaseModuleFromName("SNMPv2-SMI")
	testutil.True(t, ok, "BaseModuleFromName SNMPv2-SMI ok")
	testutil.Equal(t, BaseModuleSNMPv2SMI, bm, "BaseModuleFromName SNMPv2-SMI")

	_, ok = BaseModuleFromName("NONEXISTENT")
	testutil.False(t, ok, "BaseModuleFromName NONEXISTENT ok")
}

func TestIsBaseModule(t *testing.T) {
	testutil.True(t, IsBaseModule("SNMPv2-SMI"), "SNMPv2-SMI")
	testutil.True(t, IsBaseModule("RFC1155-SMI"), "RFC1155-SMI")
	testutil.False(t, IsBaseModule("IF-MIB"), "IF-MIB")
	testutil.False(t, IsBaseModule("SNMPv2-MIB"), "SNMPv2-MIB")
}

func TestBaseModule_Name_OutOfRange(t *testing.T) {
	invalid := BaseModule(999)
	testutil.Equal(t, "", invalid.Name(), "out-of-range BaseModule.Name()")
}
