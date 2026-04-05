package module

import (
	"slices"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
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

// --- helpers for inspecting base module definitions ---

// defNames returns the names of all definitions in the module.
func defNames(mod *Module) []string {
	var names []string
	for d := range mod.AllDefinitions() {
		names = append(names, d.DefinitionName())
	}
	return names
}

// --- CreateBaseModules ---

func TestCreateBaseModules(t *testing.T) {
	modules := CreateBaseModules()
	testutil.Len(t, modules, 7, "module count")

	wantNames := []string{
		"SNMPv2-SMI", "SNMPv2-TC", "SNMPv2-CONF",
		"RFC1155-SMI", "RFC1065-SMI", "RFC-1212", "RFC-1215",
	}
	for i, mod := range modules {
		testutil.Equal(t, wantNames[i], mod.Name, "module[%d] name", i)
	}
}

func TestCreateBaseModules_Languages(t *testing.T) {
	modules := CreateBaseModules()

	for _, mod := range modules {
		switch mod.Name {
		case "SNMPv2-SMI", "SNMPv2-TC", "SNMPv2-CONF":
			testutil.Equal(t, types.LanguageSMIv2, mod.Language, "%s language", mod.Name)
		case "RFC1155-SMI", "RFC1065-SMI", "RFC-1212", "RFC-1215":
			testutil.Equal(t, types.LanguageSMIv1, mod.Language, "%s language", mod.Name)
		}
	}
}

func TestCreateBaseModules_MacroOnlyModulesEmpty(t *testing.T) {
	modules := CreateBaseModules()
	for _, mod := range modules {
		switch mod.Name {
		case "SNMPv2-CONF", "RFC-1212", "RFC-1215":
			testutil.Equal(t, 0, mod.DefinitionCount(), "%s definitions", mod.Name)
		}
	}
}

// --- GetBaseModule / AllBaseModules ---

func TestGetBaseModule(t *testing.T) {
	for _, name := range BaseModuleNames() {
		mod := GetBaseModule(name)
		testutil.NotNil(t, mod, "GetBaseModule(%q)", name)
		testutil.Equal(t, name, mod.Name, "GetBaseModule(%q).Name", name)
	}

	testutil.Nil(t, GetBaseModule("NONEXISTENT"), "GetBaseModule for unknown name")
	testutil.Nil(t, GetBaseModule("SNMPv2-MIB"), "GetBaseModule for non-base module")
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

// --- SNMPv2-SMI OID definitions ---

func TestSNMPv2SMI_OIDTree(t *testing.T) {
	mod := GetBaseModule("SNMPv2-SMI")

	// Verify all expected OID assignments exist and have correct components.
	tests := []struct {
		name       string
		wantParent string // expected first component (name reference)
		wantArc    uint32 // expected second component (numeric arc)
	}{
		// Root assignments use a single numeric component.
		// These are tested separately below.

		// { parent, arc } style assignments:
		{"org", "iso", 3},
		{"dod", "org", 6},
		{"internet", "dod", 1},
		{"directory", "internet", 1},
		{"mgmt", "internet", 2},
		{"experimental", "internet", 3},
		{"private", "internet", 4},
		{"enterprises", "private", 1},
		{"mib-2", "mgmt", 1},
		{"transmission", "mib-2", 10},
		{"security", "internet", 5},
		{"snmpV2", "internet", 6},
		{"snmpDomains", "snmpV2", 1},
		{"snmpProxys", "snmpV2", 2},
		{"snmpModules", "snmpV2", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			va := requireDef[*ValueAssignment](t, mod, tt.name)
			testutil.Len(t, va.Oid.Components, 2, "%s component count", tt.name)

			nameComp, ok := va.Oid.Components[0].(*OidComponentName)
			testutil.True(t, ok, "%s[0] should be OidComponentName", tt.name)
			testutil.Equal(t, tt.wantParent, nameComp.NameValue, "%s parent", tt.name)

			numComp, ok := va.Oid.Components[1].(*OidComponentNumber)
			testutil.True(t, ok, "%s[1] should be OidComponentNumber", tt.name)
			testutil.Equal(t, tt.wantArc, numComp.Value, "%s arc", tt.name)
		})
	}
}

func TestSNMPv2SMI_RootOIDs(t *testing.T) {
	mod := GetBaseModule("SNMPv2-SMI")

	// Root-level OIDs have a single numeric component.
	roots := []struct {
		name string
		arc  uint32
	}{
		{"ccitt", 0},
		{"iso", 1},
		{"joint-iso-ccitt", 2},
	}

	for _, tt := range roots {
		t.Run(tt.name, func(t *testing.T) {
			va := requireDef[*ValueAssignment](t, mod, tt.name)
			testutil.Len(t, va.Oid.Components, 1, "%s component count", tt.name)

			numComp, ok := va.Oid.Components[0].(*OidComponentNumber)
			testutil.True(t, ok, "%s[0] should be OidComponentNumber", tt.name)
			testutil.Equal(t, tt.arc, numComp.Value, "%s arc", tt.name)
		})
	}
}

func TestSNMPv2SMI_ZeroDotZero(t *testing.T) {
	mod := GetBaseModule("SNMPv2-SMI")
	va := requireDef[*ValueAssignment](t, mod, "zeroDotZero")
	testutil.Len(t, va.Oid.Components, 2, "zeroDotZero component count")

	c0, ok := va.Oid.Components[0].(*OidComponentNumber)
	testutil.True(t, ok, "zeroDotZero[0] should be OidComponentNumber")
	testutil.Equal(t, uint32(0), c0.Value, "zeroDotZero[0]")

	c1, ok := va.Oid.Components[1].(*OidComponentNumber)
	testutil.True(t, ok, "zeroDotZero[1] should be OidComponentNumber")
	testutil.Equal(t, uint32(0), c1.Value, "zeroDotZero[1]")
}

// --- SNMPv2-SMI type definitions ---

func TestSNMPv2SMI_BaseTypes(t *testing.T) {
	mod := GetBaseModule("SNMPv2-SMI")

	tests := []struct {
		name     string
		wantBase types.BaseType
	}{
		{"Integer32", types.BaseInteger32},
		{"Counter32", types.BaseCounter32},
		{"Counter64", types.BaseCounter64},
		{"Gauge32", types.BaseGauge32},
		{"Unsigned32", types.BaseUnsigned32},
		{"TimeTicks", types.BaseTimeTicks},
		{"IpAddress", types.BaseIpAddress},
		{"Opaque", types.BaseOpaque},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := requireDef[*TypeDef](t, mod, tt.name)
			testutil.NotNil(t, td.BaseType, "%s.BaseType should be set", tt.name)
			testutil.Equal(t, tt.wantBase, *td.BaseType, "%s base type", tt.name)
			testutil.False(t, td.IsTextualConvention, "%s should not be a TC", tt.name)
		})
	}
}

func TestSNMPv2SMI_Integer32Range(t *testing.T) {
	mod := GetBaseModule("SNMPv2-SMI")
	td := requireDef[*TypeDef](t, mod, "Integer32")

	constrained, ok := td.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, ok, "Integer32 syntax should be constrained")

	rangeConstraint, ok := constrained.Constraint.(*ConstraintRange)
	testutil.True(t, ok, "Integer32 constraint should be ConstraintRange")
	testutil.Len(t, rangeConstraint.Ranges, 1, "Integer32 range count")

	r := rangeConstraint.Ranges[0]
	minVal, ok := r.Min.(*RangeValueSigned)
	testutil.True(t, ok, "Integer32 min should be signed")
	testutil.Equal(t, int64(-2147483648), minVal.Value, "Integer32 min")

	maxVal, ok := r.Max.(*RangeValueSigned)
	testutil.True(t, ok, "Integer32 max should be signed")
	testutil.Equal(t, int64(2147483647), maxVal.Value, "Integer32 max")
}

func TestSNMPv2SMI_UnsignedRanges(t *testing.T) {
	mod := GetBaseModule("SNMPv2-SMI")

	tests := []struct {
		name   string
		maxVal uint64
	}{
		{"Counter32", 4294967295},
		{"Counter64", 18446744073709551615},
		{"Gauge32", 4294967295},
		{"Unsigned32", 4294967295},
		{"TimeTicks", 4294967295},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := requireDef[*TypeDef](t, mod, tt.name)
			constrained, ok := td.Syntax.(*TypeSyntaxConstrained)
			testutil.True(t, ok, "%s syntax should be constrained", tt.name)

			rangeConstraint, ok := constrained.Constraint.(*ConstraintRange)
			testutil.True(t, ok, "%s constraint should be ConstraintRange", tt.name)
			testutil.Len(t, rangeConstraint.Ranges, 1, "%s range count", tt.name)

			r := rangeConstraint.Ranges[0]
			minVal, ok := r.Min.(*RangeValueUnsigned)
			testutil.True(t, ok, "%s min should be unsigned", tt.name)
			testutil.Equal(t, uint64(0), minVal.Value, "%s min", tt.name)

			maxVal, ok := r.Max.(*RangeValueUnsigned)
			testutil.True(t, ok, "%s max should be unsigned", tt.name)
			testutil.Equal(t, tt.maxVal, maxVal.Value, "%s max", tt.name)
		})
	}
}

func TestSNMPv2SMI_IpAddress_SizeConstraint(t *testing.T) {
	mod := GetBaseModule("SNMPv2-SMI")
	td := requireDef[*TypeDef](t, mod, "IpAddress")

	constrained, ok := td.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, ok, "IpAddress syntax should be constrained")

	_, ok = constrained.Base.(*TypeSyntaxOctetString)
	testutil.True(t, ok, "IpAddress base should be OCTET STRING")

	sizeConstraint, ok := constrained.Constraint.(*ConstraintSize)
	testutil.True(t, ok, "IpAddress constraint should be ConstraintSize")
	testutil.Len(t, sizeConstraint.Ranges, 1, "IpAddress size range count")

	r := sizeConstraint.Ranges[0]
	minVal, ok := r.Min.(*RangeValueUnsigned)
	testutil.True(t, ok, "IpAddress size min type")
	testutil.Equal(t, uint64(4), minVal.Value, "IpAddress SIZE")
	testutil.Nil(t, r.Max, "IpAddress size should be a fixed value (no max)")
}

func TestSNMPv2SMI_Opaque_NoConstraint(t *testing.T) {
	mod := GetBaseModule("SNMPv2-SMI")
	td := requireDef[*TypeDef](t, mod, "Opaque")

	_, ok := td.Syntax.(*TypeSyntaxOctetString)
	testutil.True(t, ok, "Opaque syntax should be plain OCTET STRING")
}

func TestSNMPv2SMI_OidTypes(t *testing.T) {
	mod := GetBaseModule("SNMPv2-SMI")

	for _, name := range []string{"ObjectName", "NotificationName"} {
		t.Run(name, func(t *testing.T) {
			td := requireDef[*TypeDef](t, mod, name)
			_, ok := td.Syntax.(*TypeSyntaxObjectIdentifier)
			testutil.True(t, ok, "%s syntax should be OBJECT IDENTIFIER", name)
			testutil.Nil(t, td.BaseType, "%s should have no explicit base type", name)
		})
	}
}

func TestSNMPv2SMI_ExtUTCTime(t *testing.T) {
	mod := GetBaseModule("SNMPv2-SMI")
	td := requireDef[*TypeDef](t, mod, "ExtUTCTime")

	testutil.Equal(t, types.StatusCurrent, td.Status, "ExtUTCTime status")

	constrained, ok := td.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, ok, "ExtUTCTime syntax should be constrained")

	sizeConstraint, ok := constrained.Constraint.(*ConstraintSize)
	testutil.True(t, ok, "ExtUTCTime constraint should be ConstraintSize")
	testutil.Len(t, sizeConstraint.Ranges, 2, "ExtUTCTime should have two size alternatives")

	// SIZE (11 | 13)
	r0Min, ok := sizeConstraint.Ranges[0].Min.(*RangeValueUnsigned)
	testutil.True(t, ok, "size[0] min type")
	testutil.Equal(t, uint64(11), r0Min.Value, "size[0]")

	r1Min, ok := sizeConstraint.Ranges[1].Min.(*RangeValueUnsigned)
	testutil.True(t, ok, "size[1] min type")
	testutil.Equal(t, uint64(13), r1Min.Value, "size[1]")
}

// --- SNMPv2-TC definitions ---

func TestSNMPv2TC_Imports(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")
	testutil.Len(t, mod.Imports, 1, "SNMPv2-TC import count")
	testutil.Equal(t, "SNMPv2-SMI", mod.Imports[0].Module, "import module")
	testutil.Equal(t, "TimeTicks", mod.Imports[0].Symbol, "import symbol")
}

func TestSNMPv2TC_AllDefinitionsPresent(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")

	wantTCs := []string{
		"DisplayString", "PhysAddress", "MacAddress",
		"TruthValue", "RowStatus", "StorageType",
		"TimeStamp", "TimeInterval", "DateAndTime",
		"TestAndIncr", "AutonomousType", "InstancePointer",
		"VariablePointer", "RowPointer", "TDomain", "TAddress",
	}

	names := defNames(mod)
	for _, want := range wantTCs {
		testutil.True(t, slices.Contains(names, want), "TC %q should be defined", want)
	}
	testutil.Equal(t, len(wantTCs), mod.DefinitionCount(), "TC count")
}

func TestSNMPv2TC_AllAreTextualConventions(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")
	for _, td := range mod.TypeDefs {
		testutil.True(t, td.IsTextualConvention, "%s should be a TC", td.Name)
	}
}

func TestSNMPv2TC_DisplayHints(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")

	tests := []struct {
		name string
		hint string
	}{
		{"DisplayString", "255a"},
		{"PhysAddress", "1x:"},
		{"MacAddress", "1x:"},
		{"DateAndTime", "2d-1d-1d,1d:1d:1d.1d,1a1d:1d"},
		// These TCs have no display hint.
		{"TruthValue", ""},
		{"RowStatus", ""},
		{"StorageType", ""},
		{"TimeStamp", ""},
		{"TimeInterval", ""},
		{"AutonomousType", ""},
		{"TAddress", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := requireDef[*TypeDef](t, mod, tt.name)
			testutil.Equal(t, tt.hint, td.DisplayHint, "%s display hint", tt.name)
		})
	}
}

func TestSNMPv2TC_TruthValue(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")
	td := requireDef[*TypeDef](t, mod, "TruthValue")

	enumSyntax, ok := td.Syntax.(*TypeSyntaxIntegerEnum)
	testutil.True(t, ok, "TruthValue syntax should be IntegerEnum")
	testutil.Len(t, enumSyntax.NamedNumbers, 2, "TruthValue enum count")
	testutil.Equal(t, "true", enumSyntax.NamedNumbers[0].Name, "TruthValue[0] name")
	testutil.Equal(t, int64(1), enumSyntax.NamedNumbers[0].Value, "TruthValue[0] value")
	testutil.Equal(t, "false", enumSyntax.NamedNumbers[1].Name, "TruthValue[1] name")
	testutil.Equal(t, int64(2), enumSyntax.NamedNumbers[1].Value, "TruthValue[1] value")
}

func TestSNMPv2TC_RowStatus(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")
	td := requireDef[*TypeDef](t, mod, "RowStatus")

	enumSyntax, ok := td.Syntax.(*TypeSyntaxIntegerEnum)
	testutil.True(t, ok, "RowStatus syntax should be IntegerEnum")

	wantEnums := []struct {
		name  string
		value int64
	}{
		{"active", 1},
		{"notInService", 2},
		{"notReady", 3},
		{"createAndGo", 4},
		{"createAndWait", 5},
		{"destroy", 6},
	}
	testutil.Len(t, enumSyntax.NamedNumbers, len(wantEnums), "RowStatus enum count")
	for i, want := range wantEnums {
		testutil.Equal(t, want.name, enumSyntax.NamedNumbers[i].Name, "RowStatus[%d] name", i)
		testutil.Equal(t, want.value, enumSyntax.NamedNumbers[i].Value, "RowStatus[%d] value", i)
	}
}

func TestSNMPv2TC_StorageType(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")
	td := requireDef[*TypeDef](t, mod, "StorageType")

	enumSyntax, ok := td.Syntax.(*TypeSyntaxIntegerEnum)
	testutil.True(t, ok, "StorageType syntax should be IntegerEnum")

	wantEnums := []struct {
		name  string
		value int64
	}{
		{"other", 1},
		{"volatile", 2},
		{"nonVolatile", 3},
		{"permanent", 4},
		{"readOnly", 5},
	}
	testutil.Len(t, enumSyntax.NamedNumbers, len(wantEnums), "StorageType enum count")
	for i, want := range wantEnums {
		testutil.Equal(t, want.name, enumSyntax.NamedNumbers[i].Name, "StorageType[%d] name", i)
		testutil.Equal(t, want.value, enumSyntax.NamedNumbers[i].Value, "StorageType[%d] value", i)
	}
}

func TestSNMPv2TC_DisplayString_SizeConstraint(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")
	td := requireDef[*TypeDef](t, mod, "DisplayString")

	constrained, ok := td.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, ok, "DisplayString syntax should be constrained")

	sizeConstraint, ok := constrained.Constraint.(*ConstraintSize)
	testutil.True(t, ok, "DisplayString constraint should be ConstraintSize")
	testutil.Len(t, sizeConstraint.Ranges, 1, "DisplayString size range count")

	r := sizeConstraint.Ranges[0]
	minVal, ok := r.Min.(*RangeValueUnsigned)
	testutil.True(t, ok, "DisplayString size min type")
	testutil.Equal(t, uint64(0), minVal.Value, "DisplayString size min")

	maxVal, ok := r.Max.(*RangeValueUnsigned)
	testutil.True(t, ok, "DisplayString size max type")
	testutil.Equal(t, uint64(255), maxVal.Value, "DisplayString size max")
}

func TestSNMPv2TC_MacAddress_FixedSize(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")
	td := requireDef[*TypeDef](t, mod, "MacAddress")

	constrained, ok := td.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, ok, "MacAddress syntax should be constrained")

	sizeConstraint, ok := constrained.Constraint.(*ConstraintSize)
	testutil.True(t, ok, "MacAddress constraint should be ConstraintSize")
	testutil.Len(t, sizeConstraint.Ranges, 1, "MacAddress size range count")

	r := sizeConstraint.Ranges[0]
	minVal, ok := r.Min.(*RangeValueUnsigned)
	testutil.True(t, ok, "MacAddress size value type")
	testutil.Equal(t, uint64(6), minVal.Value, "MacAddress SIZE")
	testutil.Nil(t, r.Max, "MacAddress size should be fixed (no max)")
}

func TestSNMPv2TC_DateAndTime_SizeConstraint(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")
	td := requireDef[*TypeDef](t, mod, "DateAndTime")

	constrained, ok := td.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, ok, "DateAndTime syntax should be constrained")

	sizeConstraint, ok := constrained.Constraint.(*ConstraintSize)
	testutil.True(t, ok, "DateAndTime constraint should be ConstraintSize")
	testutil.Len(t, sizeConstraint.Ranges, 2, "DateAndTime size alternatives")

	// SIZE (8 | 11)
	r0Min, ok := sizeConstraint.Ranges[0].Min.(*RangeValueUnsigned)
	testutil.True(t, ok, "DateAndTime size[0] type")
	testutil.Equal(t, uint64(8), r0Min.Value, "DateAndTime size[0]")

	r1Min, ok := sizeConstraint.Ranges[1].Min.(*RangeValueUnsigned)
	testutil.True(t, ok, "DateAndTime size[1] type")
	testutil.Equal(t, uint64(11), r1Min.Value, "DateAndTime size[1]")
}

func TestSNMPv2TC_TAddress_SizeConstraint(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")
	td := requireDef[*TypeDef](t, mod, "TAddress")

	constrained, ok := td.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, ok, "TAddress syntax should be constrained")

	sizeConstraint, ok := constrained.Constraint.(*ConstraintSize)
	testutil.True(t, ok, "TAddress constraint should be ConstraintSize")
	testutil.Len(t, sizeConstraint.Ranges, 1, "TAddress size range count")

	r := sizeConstraint.Ranges[0]
	minVal, ok := r.Min.(*RangeValueUnsigned)
	testutil.True(t, ok, "TAddress size min type")
	testutil.Equal(t, uint64(1), minVal.Value, "TAddress size min")

	maxVal, ok := r.Max.(*RangeValueUnsigned)
	testutil.True(t, ok, "TAddress size max type")
	testutil.Equal(t, uint64(255), maxVal.Value, "TAddress size max")
}

func TestSNMPv2TC_TimeStamp_BasedOnTimeTicks(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")
	td := requireDef[*TypeDef](t, mod, "TimeStamp")

	ref, ok := td.Syntax.(*TypeSyntaxTypeRef)
	testutil.True(t, ok, "TimeStamp syntax should be TypeRef")
	testutil.Equal(t, "TimeTicks", ref.Name, "TimeStamp base type reference")
}

func TestSNMPv2TC_OidBasedTypes(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")

	oidTypes := []string{"AutonomousType", "InstancePointer", "VariablePointer", "RowPointer", "TDomain"}
	for _, name := range oidTypes {
		t.Run(name, func(t *testing.T) {
			td := requireDef[*TypeDef](t, mod, name)
			_, ok := td.Syntax.(*TypeSyntaxObjectIdentifier)
			testutil.True(t, ok, "%s syntax should be OBJECT IDENTIFIER", name)
		})
	}
}

func TestSNMPv2TC_InstancePointer_Obsolete(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")
	td := requireDef[*TypeDef](t, mod, "InstancePointer")
	testutil.Equal(t, types.StatusObsolete, td.Status, "InstancePointer status")
}

func TestSNMPv2TC_TimeInterval_Range(t *testing.T) {
	mod := GetBaseModule("SNMPv2-TC")
	td := requireDef[*TypeDef](t, mod, "TimeInterval")

	constrained, ok := td.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, ok, "TimeInterval syntax should be constrained")

	rangeConstraint, ok := constrained.Constraint.(*ConstraintRange)
	testutil.True(t, ok, "TimeInterval constraint should be ConstraintRange")
	testutil.Len(t, rangeConstraint.Ranges, 1, "TimeInterval range count")

	r := rangeConstraint.Ranges[0]
	minVal, ok := r.Min.(*RangeValueSigned)
	testutil.True(t, ok, "TimeInterval min type")
	testutil.Equal(t, int64(0), minVal.Value, "TimeInterval min")

	maxVal, ok := r.Max.(*RangeValueSigned)
	testutil.True(t, ok, "TimeInterval max type")
	testutil.Equal(t, int64(2147483647), maxVal.Value, "TimeInterval max")
}

// --- SMIv1 base modules ---

func TestSMIv1Base_TypeDefinitions(t *testing.T) {
	// Both RFC1155-SMI and RFC1065-SMI should produce the same type definitions.
	for _, name := range []string{"RFC1155-SMI", "RFC1065-SMI"} {
		t.Run(name, func(t *testing.T) {
			mod := GetBaseModule(name)

			tests := []struct {
				typeName string
				wantBase types.BaseType
			}{
				{"Counter", types.BaseCounter32},
				{"Gauge", types.BaseGauge32},
				{"IpAddress", types.BaseIpAddress},
				{"NetworkAddress", types.BaseIpAddress},
				{"TimeTicks", types.BaseTimeTicks},
				{"Opaque", types.BaseOpaque},
			}

			for _, tt := range tests {
				t.Run(tt.typeName, func(t *testing.T) {
					td := requireDef[*TypeDef](t, mod, tt.typeName)
					testutil.NotNil(t, td.BaseType, "%s.BaseType", tt.typeName)
					testutil.Equal(t, tt.wantBase, *td.BaseType, "%s base type", tt.typeName)
					testutil.False(t, td.IsTextualConvention, "%s should not be TC", tt.typeName)
				})
			}
		})
	}
}

func TestSMIv1Base_NetworkAddress_RefsIpAddress(t *testing.T) {
	mod := GetBaseModule("RFC1155-SMI")
	td := requireDef[*TypeDef](t, mod, "NetworkAddress")

	ref, ok := td.Syntax.(*TypeSyntaxTypeRef)
	testutil.True(t, ok, "NetworkAddress syntax should be TypeRef")
	testutil.Equal(t, "IpAddress", ref.Name, "NetworkAddress references IpAddress")
}

func TestSMIv1Base_OIDTree(t *testing.T) {
	// SMIv1 modules share the core OID tree (iso through enterprises)
	// but not the SMIv2-specific nodes (mib-2, snmpV2, etc.).
	mod := GetBaseModule("RFC1155-SMI")

	coreOIDs := []string{"iso", "org", "dod", "internet", "directory", "mgmt", "experimental", "private", "enterprises"}
	names := defNames(mod)
	for _, name := range coreOIDs {
		testutil.True(t, slices.Contains(names, name), "%s: OID %q should be defined", mod.Name, name)
	}

	// SMIv2-specific OIDs should NOT be present in SMIv1 modules.
	smiv2Only := []string{"mib-2", "transmission", "snmpV2", "snmpDomains", "snmpProxys", "snmpModules", "zeroDotZero", "snmp", "ccitt", "joint-iso-ccitt", "security"}
	for _, name := range smiv2Only {
		testutil.False(t, slices.Contains(names, name), "%s: SMIv2 OID %q should not be in SMIv1 module", mod.Name, name)
	}
}

func TestSMIv1Base_ObjectName(t *testing.T) {
	mod := GetBaseModule("RFC1155-SMI")
	td := requireDef[*TypeDef](t, mod, "ObjectName")

	_, ok := td.Syntax.(*TypeSyntaxObjectIdentifier)
	testutil.True(t, ok, "ObjectName syntax should be OBJECT IDENTIFIER")
	testutil.Nil(t, td.BaseType, "ObjectName should have no explicit base type")
}

func TestSMIv1Base_CounterRange(t *testing.T) {
	mod := GetBaseModule("RFC1155-SMI")
	td := requireDef[*TypeDef](t, mod, "Counter")

	constrained, ok := td.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, ok, "Counter syntax should be constrained")

	rangeConstraint, ok := constrained.Constraint.(*ConstraintRange)
	testutil.True(t, ok, "Counter constraint should be ConstraintRange")
	testutil.Len(t, rangeConstraint.Ranges, 1, "Counter range count")

	r := rangeConstraint.Ranges[0]
	minVal, ok := r.Min.(*RangeValueUnsigned)
	testutil.True(t, ok, "Counter min type")
	testutil.Equal(t, uint64(0), minVal.Value, "Counter min")

	maxVal, ok := r.Max.(*RangeValueUnsigned)
	testutil.True(t, ok, "Counter max type")
	testutil.Equal(t, uint64(4294967295), maxVal.Value, "Counter max")
}

// --- BaseModule.Name() edge case ---

func TestBaseModule_Name_OutOfRange(t *testing.T) {
	invalid := BaseModule(999)
	testutil.Equal(t, "", invalid.Name(), "out-of-range BaseModule.Name()")
}
