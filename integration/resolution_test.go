package integration

import (
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/internal/testutil"
)

// IndexResolutionTestCase tests that INDEX clauses resolve correctly across imports.
type IndexResolutionTestCase struct {
	Name      string   // row entry name
	Module    string   // module name
	WantIndex []string // expected index object names
}

// indexResolutionTests tests INDEX resolution for objects that import index columns
// from other modules (e.g., RADLAN-MIB importing dot1dBasePort from BRIDGE-MIB).
var indexResolutionTests = []IndexResolutionTestCase{
	{Name: "rlPortGvrpErrorStatisticsEntry", Module: "RADLAN-MIB", WantIndex: []string{"dot1dBasePort"}},
	{Name: "rldot1dStpPortBpduGuardEntry", Module: "RADLAN-MIB", WantIndex: []string{"dot1dBasePort"}},
	{Name: "rlStormCtrlGroupEntry", Module: "RADLAN-MIB", WantIndex: []string{"dot1dBasePort"}},
	{Name: "rldot1dPriorityPortGroupEntry", Module: "RADLAN-MIB", WantIndex: []string{"dot1dBasePort"}},
	{Name: "rlPortGvrpRegistrationModeEntry", Module: "RADLAN-MIB", WantIndex: []string{"dot1dBasePort"}},
}

func TestIndexResolution(t *testing.T) {
	m := loadCorpus(t)

	for _, tc := range indexResolutionTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			mod := m.Module(tc.Module)
			if mod == nil {
				t.Skipf("module %s not in corpus", tc.Module)
			}
			obj := mod.Object(tc.Name)
			if obj == nil {
				t.Skipf("object %s not found", tc.Name)
			}

			idx := obj.Index()
			testutil.Len(t, idx, len(tc.WantIndex), "INDEX count mismatch")

			for i, wantName := range tc.WantIndex {
				if i >= len(idx) {
					continue
				}
				testutil.NotNil(t, idx[i].Object, "INDEX[%d] should be resolved", i)
				testutil.Equal(t, wantName, idx[i].Object.Name(), "INDEX[%d] name mismatch", i)
			}
		})
	}
}

// NotificationObjectsTestCase tests that notification OBJECTS resolve correctly.
type NotificationObjectsTestCase struct {
	Name        string   // notification name
	Module      string   // module name
	WantObjects []string // expected object names
}

// notificationObjectsTests tests OBJECTS resolution for notifications that reference
// objects not explicitly imported by the defining module.
var notificationObjectsTests = []NotificationObjectsTestCase{
	{Name: "autoUpgradeTrap", Module: "ES4552BH2-MIB",
		WantObjects: []string{"fileCopyFileType", "trapAutoUpgradeResult", "trapAutoUpgradeNewVer"}},
	{Name: "lbdDetectionTrap", Module: "ES4552BH2-MIB",
		WantObjects: []string{"trapIfIndex", "trapVlanId"}},
}

func TestNotificationObjects(t *testing.T) {
	m := loadCorpus(t)

	for _, tc := range notificationObjectsTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			mod := m.Module(tc.Module)
			if mod == nil {
				t.Skipf("module %s not in corpus", tc.Module)
			}

			var notif gomib.Notification
			for _, n := range mod.Notifications() {
				if n.Name() == tc.Name {
					notif = n
					break
				}
			}
			if notif == nil {
				t.Skipf("notification %s not found", tc.Name)
			}

			objects := notif.Objects()
			testutil.Len(t, objects, len(tc.WantObjects), "OBJECTS count mismatch")

			for i, wantName := range tc.WantObjects {
				if i >= len(objects) {
					continue
				}
				testutil.NotNil(t, objects[i], "OBJECTS[%d] should be resolved", i)
				testutil.Equal(t, wantName, objects[i].Name(), "OBJECTS[%d] name mismatch", i)
			}
		})
	}
}

// BaseTypeTestCase tests that object base types resolve correctly through type chains.
type BaseTypeTestCase struct {
	Name     string         // object name
	Module   string         // module name
	WantBase gomib.BaseType // expected base type
}

// baseTypeTests tests base type resolution for objects using TCs that derive from
// constrained built-in types (e.g., TmnxHigh32 -> Unsigned32).
var baseTypeTests = []BaseTypeTestCase{
	{Name: "sapIngQosSCIRHi", Module: "TIMETRA-SAP-MIB", WantBase: gomib.BaseUnsigned32},
	{Name: "sapEgrQosSCIRHi", Module: "TIMETRA-SAP-MIB", WantBase: gomib.BaseUnsigned32},
	{Name: "tSapIngPolicerAdminPIRHi", Module: "TIMETRA-QOS-MIB", WantBase: gomib.BaseUnsigned32},
	{Name: "tSapEgrPolicerAdminCIRHi", Module: "TIMETRA-QOS-MIB", WantBase: gomib.BaseUnsigned32},
	{Name: "tIPv6FilterSharedPccRuleInsrtSiz", Module: "TIMETRA-FILTER-MIB", WantBase: gomib.BaseUnsigned32},
}

func TestBaseTypeResolution(t *testing.T) {
	m := loadCorpus(t)

	for _, tc := range baseTypeTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.Name)
			testutil.NotNil(t, obj.Type(), "object should have resolved type")
			testutil.Equal(t, tc.WantBase, obj.Type().Base(), "base type mismatch")
		})
	}
}

// TypeNameTestCase tests that object type names are preserved correctly.
type TypeNameTestCase struct {
	Name         string // object name
	Module       string // module name
	WantTypeName string // expected type name
}

// typeNameTests tests type name preservation for objects using TCs with inline
// enum restrictions (e.g., TPSPRateType { kbps(1), percentLocal(2) }).
var typeNameTests = []TypeNameTestCase{
	{Name: "tmnxSubAuthPlcyRadAuthAlgorithm", Module: "TIMETRA-SUBSCRIBER-MGMT-MIB", WantTypeName: "TmnxSubRadServAlgorithm"},
	{Name: "tPortSchedPlcyLvl4RateType", Module: "TIMETRA-QOS-MIB", WantTypeName: "TPSPRateType"},
	{Name: "tPortSchedPlcyLvl6RateType", Module: "TIMETRA-QOS-MIB", WantTypeName: "TPSPRateType"},
	{Name: "tPortSchedPlcyMaxRateType", Module: "TIMETRA-QOS-MIB", WantTypeName: "TPSPRateType"},
}

func TestTypeNameResolution(t *testing.T) {
	m := loadCorpus(t)

	for _, tc := range typeNameTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.Name)
			testutil.NotNil(t, obj.Type(), "object should have resolved type")
			testutil.Equal(t, tc.WantTypeName, obj.Type().Name(), "type name mismatch")
		})
	}
}

// EffectiveEnumsTestCase tests that effective enums are inherited from type chains.
type EffectiveEnumsTestCase struct {
	Name      string           // object name
	Module    string           // module name
	WantEnums map[string]int64 // expected enum label -> value mapping
}

// effectiveEnumsTests tests effective enum inheritance for objects using TCs like RowStatus.
var effectiveEnumsTests = []EffectiveEnumsTestCase{
	{Name: "syntheticSimpleRowStatus", Module: "SYNTHETIC-MIB",
		WantEnums: map[string]int64{
			"active": 1, "notInService": 2, "notReady": 3,
			"createAndGo": 4, "createAndWait": 5, "destroy": 6,
		}},
	{Name: "hwpingUdpServerRowStatus", Module: "HUAWEI-DISMAN-PING-MIB",
		WantEnums: map[string]int64{
			"active": 1, "notInService": 2, "notReady": 3,
			"createAndGo": 4, "createAndWait": 5, "destroy": 6,
		}},
}

func TestEffectiveEnums(t *testing.T) {
	m := loadCorpus(t)

	for _, tc := range effectiveEnumsTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.Name)

			enums := obj.EffectiveEnums()
			testutil.NotEmpty(t, enums, "should have effective enums")

			enumMap := make(map[string]int64)
			for _, e := range enums {
				enumMap[e.Label] = e.Value
			}

			for name, value := range tc.WantEnums {
				got, ok := enumMap[name]
				testutil.True(t, ok, "missing enum %s", name)
				if ok {
					testutil.Equal(t, value, got, "enum %s value mismatch", name)
				}
			}
		})
	}
}

// EffectiveBitsTestCase tests that effective bits are inherited from type chains.
type EffectiveBitsTestCase struct {
	Name     string   // object name
	Module   string   // module name
	WantBits []string // expected bit labels
}

// effectiveBitsTests tests effective bits inheritance for objects using BITS TCs.
var effectiveBitsTests = []EffectiveBitsTestCase{
	{Name: "rcLldpPortCdpRemCapabilities", Module: "RAPID-CITY",
		WantBits: []string{"other", "repeater", "bridge", "wlanAccessPoint",
			"router", "telephone", "docsisCableDevice", "stationOnly"}},
}

func TestEffectiveBits(t *testing.T) {
	m := loadCorpus(t)

	for _, tc := range effectiveBitsTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			mod := m.Module(tc.Module)
			if mod == nil {
				t.Skipf("module %s not in corpus", tc.Module)
			}
			obj := mod.Object(tc.Name)
			if obj == nil {
				t.Skipf("object %s not found", tc.Name)
			}

			bits := obj.EffectiveBits()
			testutil.NotEmpty(t, bits, "should have effective bits")

			bitNames := make(map[string]bool)
			for _, b := range bits {
				bitNames[b.Label] = true
			}

			for _, expected := range tc.WantBits {
				testutil.True(t, bitNames[expected], "missing bit %s", expected)
			}
		})
	}
}
