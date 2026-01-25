package integration

import (
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/stretchr/testify/require"
)

// AccessTestCase defines a test case for object access verification.
// Verify expected values with: snmptranslate -m <MODULE> -Td <name>
type AccessTestCase struct {
	Name    string       // object name
	Module  string       // module name
	Access  gomib.Access // expected access level
	NetSnmp string       // snmptranslate command used for verification
}

// accessTests contains all access level test cases.
//
// Verified against net-snmp 5.9.4: snmptranslate -Td -m SYNTHETIC-MIB SYNTHETIC-MIB::<name>
var accessTests = []AccessTestCase{
	// === Scalars - read-only ===
	{Name: "syntheticSystemDescription", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticSystemObjectID", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticSystemUpTime", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticMemorySize", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticLastChange", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticErrorState", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticBootStatus", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticDeviceType", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticInstallDate", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticTypeCodeValue", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticFixedId", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},

	// === Scalars - read-write ===
	{Name: "syntheticConfigSerial", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadWrite,
		NetSnmp: "MAX-ACCESS read-write"},
	{Name: "syntheticTrapEnable", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadWrite,
		NetSnmp: "MAX-ACCESS read-write"},

	// === Table columns - not-accessible (index columns) ===
	{Name: "syntheticSimpleIndex", Module: "SYNTHETIC-MIB", Access: gomib.AccessNotAccessible,
		NetSnmp: "MAX-ACCESS not-accessible"},
	{Name: "syntheticComplexGroup", Module: "SYNTHETIC-MIB", Access: gomib.AccessNotAccessible,
		NetSnmp: "MAX-ACCESS not-accessible"},
	{Name: "syntheticComplexAddress", Module: "SYNTHETIC-MIB", Access: gomib.AccessNotAccessible,
		NetSnmp: "MAX-ACCESS not-accessible"},
	{Name: "syntheticConnLocalAddressType", Module: "SYNTHETIC-MIB", Access: gomib.AccessNotAccessible,
		NetSnmp: "MAX-ACCESS not-accessible"},
	{Name: "syntheticConnLocalAddress", Module: "SYNTHETIC-MIB", Access: gomib.AccessNotAccessible,
		NetSnmp: "MAX-ACCESS not-accessible"},
	{Name: "syntheticConnLocalPort", Module: "SYNTHETIC-MIB", Access: gomib.AccessNotAccessible,
		NetSnmp: "MAX-ACCESS not-accessible"},
	{Name: "syntheticFdbAddress", Module: "SYNTHETIC-MIB", Access: gomib.AccessNotAccessible,
		NetSnmp: "MAX-ACCESS not-accessible"},

	// === Table columns - read-create ===
	{Name: "syntheticSimpleStatus", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadCreate,
		NetSnmp: "MAX-ACCESS read-create"},
	{Name: "syntheticSimpleRowStatus", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadCreate,
		NetSnmp: "MAX-ACCESS read-create"},
	{Name: "syntheticPortBitmask", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadCreate,
		NetSnmp: "MAX-ACCESS read-create"},
	{Name: "syntheticAugmentName", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadCreate,
		NetSnmp: "MAX-ACCESS read-create"},

	// === Table columns - read-only ===
	{Name: "syntheticSimpleData", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticAugmentHCData", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticAugmentPhysAddress", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticComplexValue", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticComplexTimestamp", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticConnState", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticConnProcessId", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticFdbPort", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
	{Name: "syntheticFdbEntryStatus", Module: "SYNTHETIC-MIB", Access: gomib.AccessReadOnly,
		NetSnmp: "MAX-ACCESS read-only"},
}

func TestObjectAccess(t *testing.T) {
	if len(accessTests) == 0 {
		t.Skip("no access test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range accessTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.Name)
			require.Equal(t, tc.Access, obj.Access, "access mismatch")
		})
	}
}

// StatusTestCase defines a test case for object status verification.
type StatusTestCase struct {
	Name    string       // object name
	Module  string       // module name
	Status  gomib.Status // expected status
	NetSnmp string
}

// statusTests contains all status test cases.
//
// All SYNTHETIC-MIB objects have STATUS current (verified via snmptranslate -Td).
var statusTests = []StatusTestCase{
	{Name: "syntheticSystemDescription", Module: "SYNTHETIC-MIB", Status: gomib.StatusCurrent,
		NetSnmp: "STATUS current"},
	{Name: "syntheticTrapEnable", Module: "SYNTHETIC-MIB", Status: gomib.StatusCurrent,
		NetSnmp: "STATUS current"},
	{Name: "syntheticSimpleTable", Module: "SYNTHETIC-MIB", Status: gomib.StatusCurrent,
		NetSnmp: "STATUS current"},
	{Name: "syntheticSimpleEntry", Module: "SYNTHETIC-MIB", Status: gomib.StatusCurrent,
		NetSnmp: "STATUS current"},
	{Name: "syntheticSimpleStatus", Module: "SYNTHETIC-MIB", Status: gomib.StatusCurrent,
		NetSnmp: "STATUS current"},
	{Name: "syntheticAugmentEntry", Module: "SYNTHETIC-MIB", Status: gomib.StatusCurrent,
		NetSnmp: "STATUS current"},
	{Name: "syntheticConnectionEntry", Module: "SYNTHETIC-MIB", Status: gomib.StatusCurrent,
		NetSnmp: "STATUS current"},
	{Name: "syntheticFdbEntry", Module: "SYNTHETIC-MIB", Status: gomib.StatusCurrent,
		NetSnmp: "STATUS current"},
}

func TestObjectStatus(t *testing.T) {
	if len(statusTests) == 0 {
		t.Skip("no status test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range statusTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.Name)
			require.Equal(t, tc.Status, obj.Status, "status mismatch")
		})
	}
}
