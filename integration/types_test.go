package integration

import (
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/internal/testutil"
)

// TypeTestCase defines a test case for object type resolution.
// Verify expected values with: snmptranslate -m <MODULE> -Td <name>
type TypeTestCase struct {
	Name     string         // object name
	Module   string         // module name
	TypeName string         // expected type name (e.g., "DisplayString", "InterfaceIndex")
	BaseType gomib.BaseType // expected base type
	NetSnmp  string         // snmptranslate command used for verification
}

// typeTests contains all type resolution test cases.
//
// Verified against net-snmp 5.9.4: snmptranslate -Td -m SYNTHETIC-MIB SYNTHETIC-MIB::<name>
// Note: TypeName is the TEXTUAL CONVENTION name, not the base SYNTAX.
var typeTests = []TypeTestCase{
	// === Base types (no TC) ===
	{Name: "syntheticSystemObjectID", Module: "SYNTHETIC-MIB", TypeName: "OBJECT IDENTIFIER", BaseType: gomib.BaseObjectIdentifier,
		NetSnmp: "SYNTAX OBJECT IDENTIFIER"},
	{Name: "syntheticSystemUpTime", Module: "SYNTHETIC-MIB", TypeName: "TimeTicks", BaseType: gomib.BaseTimeTicks,
		NetSnmp: "SYNTAX TimeTicks"},
	{Name: "syntheticSimpleData", Module: "SYNTHETIC-MIB", TypeName: "Counter32", BaseType: gomib.BaseCounter32,
		NetSnmp: "SYNTAX Counter32"},
	{Name: "syntheticAugmentHCData", Module: "SYNTHETIC-MIB", TypeName: "Counter64", BaseType: gomib.BaseCounter64,
		NetSnmp: "SYNTAX Counter64"},
	{Name: "syntheticComplexAddress", Module: "SYNTHETIC-MIB", TypeName: "IpAddress", BaseType: gomib.BaseIpAddress,
		NetSnmp: "SYNTAX IpAddress"},
	{Name: "syntheticComplexValue", Module: "SYNTHETIC-MIB", TypeName: "Gauge32", BaseType: gomib.BaseGauge32,
		NetSnmp: "SYNTAX Gauge32"},
	{Name: "syntheticConnProcessId", Module: "SYNTHETIC-MIB", TypeName: "Unsigned32", BaseType: gomib.BaseUnsigned32,
		NetSnmp: "SYNTAX Unsigned32"},

	// === Standard TCs from SNMPv2-TC ===
	{Name: "syntheticSystemDescription", Module: "SYNTHETIC-MIB", TypeName: "DisplayString", BaseType: gomib.BaseOctetString,
		NetSnmp: "TEXTUAL CONVENTION DisplayString"},
	{Name: "syntheticConfigSerial", Module: "SYNTHETIC-MIB", TypeName: "TestAndIncr", BaseType: gomib.BaseInteger32,
		NetSnmp: "TEXTUAL CONVENTION TestAndIncr"},
	{Name: "syntheticLastChange", Module: "SYNTHETIC-MIB", TypeName: "TimeStamp", BaseType: gomib.BaseTimeTicks,
		NetSnmp: "TEXTUAL CONVENTION TimeStamp"},
	{Name: "syntheticBootStatus", Module: "SYNTHETIC-MIB", TypeName: "TruthValue", BaseType: gomib.BaseInteger32,
		NetSnmp: "TEXTUAL CONVENTION TruthValue"},
	{Name: "syntheticDeviceType", Module: "SYNTHETIC-MIB", TypeName: "AutonomousType", BaseType: gomib.BaseObjectIdentifier,
		NetSnmp: "TEXTUAL CONVENTION AutonomousType"},
	{Name: "syntheticInstallDate", Module: "SYNTHETIC-MIB", TypeName: "DateAndTime", BaseType: gomib.BaseOctetString,
		NetSnmp: "TEXTUAL CONVENTION DateAndTime"},
	{Name: "syntheticAugmentPhysAddress", Module: "SYNTHETIC-MIB", TypeName: "PhysAddress", BaseType: gomib.BaseOctetString,
		NetSnmp: "TEXTUAL CONVENTION PhysAddress"},
	{Name: "syntheticSimpleRowStatus", Module: "SYNTHETIC-MIB", TypeName: "RowStatus", BaseType: gomib.BaseInteger32,
		NetSnmp: "TEXTUAL CONVENTION RowStatus"},
	{Name: "syntheticComplexTimestamp", Module: "SYNTHETIC-MIB", TypeName: "TimeStamp", BaseType: gomib.BaseTimeTicks,
		NetSnmp: "TEXTUAL CONVENTION TimeStamp"},
	{Name: "syntheticFdbAddress", Module: "SYNTHETIC-MIB", TypeName: "MacAddress", BaseType: gomib.BaseOctetString,
		NetSnmp: "TEXTUAL CONVENTION MacAddress"},

	// === Custom TCs from SYNTHETIC-MIB ===
	{Name: "syntheticSimpleStatus", Module: "SYNTHETIC-MIB", TypeName: "SyntheticStatus", BaseType: gomib.BaseInteger32,
		NetSnmp: "TEXTUAL CONVENTION SyntheticStatus"},
	{Name: "syntheticMemorySize", Module: "SYNTHETIC-MIB", TypeName: "SyntheticKBytes", BaseType: gomib.BaseInteger32,
		NetSnmp: "TEXTUAL CONVENTION SyntheticKBytes"},
	{Name: "syntheticErrorState", Module: "SYNTHETIC-MIB", TypeName: "SyntheticBitmask", BaseType: gomib.BaseBits,
		NetSnmp: "TEXTUAL CONVENTION SyntheticBitmask"},
	{Name: "syntheticAugmentName", Module: "SYNTHETIC-MIB", TypeName: "SyntheticName", BaseType: gomib.BaseOctetString,
		NetSnmp: "TEXTUAL CONVENTION SyntheticName"},
	{Name: "syntheticFixedId", Module: "SYNTHETIC-MIB", TypeName: "SyntheticFixedOctetString", BaseType: gomib.BaseOctetString,
		NetSnmp: "TEXTUAL CONVENTION SyntheticFixedOctetString"},
	{Name: "syntheticFdbEntryStatus", Module: "SYNTHETIC-MIB", TypeName: "SyntheticFdbStatus", BaseType: gomib.BaseInteger32,
		NetSnmp: "TEXTUAL CONVENTION SyntheticFdbStatus"},
	{Name: "syntheticConnLocalAddress", Module: "SYNTHETIC-MIB", TypeName: "SyntheticIndexInetAddress", BaseType: gomib.BaseOctetString,
		NetSnmp: "TEXTUAL CONVENTION SyntheticIndexInetAddress"},

	// === Custom TCs from SYNTHETICTYPES-MIB ===
	{Name: "syntheticTypeCodeValue", Module: "SYNTHETIC-MIB", TypeName: "SyntheticTypeCode", BaseType: gomib.BaseInteger32,
		NetSnmp: "TEXTUAL CONVENTION SyntheticTypeCode"},
	{Name: "syntheticConnLocalAddressType", Module: "SYNTHETIC-MIB", TypeName: "SyntheticInetAddressType", BaseType: gomib.BaseInteger32,
		NetSnmp: "TEXTUAL CONVENTION SyntheticInetAddressType"},
	{Name: "syntheticConnLocalPort", Module: "SYNTHETIC-MIB", TypeName: "SyntheticInetPortNumber", BaseType: gomib.BaseUnsigned32,
		NetSnmp: "TEXTUAL CONVENTION SyntheticInetPortNumber"},
}

func TestObjectType(t *testing.T) {
	if len(typeTests) == 0 {
		t.Skip("no type test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range typeTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.Name)

			testutil.NotNil(t, obj.Type(), "object should have a resolved type")
			testutil.Equal(t, tc.TypeName, obj.Type().Name(), "type name mismatch")
			testutil.Equal(t, tc.BaseType, obj.Type().Base(), "base type mismatch")
		})
	}
}

// TextualConventionTestCase defines a test case for textual convention verification.
type TextualConventionTestCase struct {
	Name        string // TC name
	Module      string // module name
	BaseType    gomib.BaseType
	Hint        string // expected display hint (empty if none)
	Description string // substring expected in description (empty to skip)
	NetSnmp     string
}

// tcTests contains textual convention test cases.
//
// Verified against net-snmp 5.9.4 via snmptranslate -Td on objects using these TCs.
// The DISPLAY-HINT comes from the TC definition.
var tcTests = []TextualConventionTestCase{
	// TCs from SYNTHETIC-MIB
	{Name: "SyntheticName", Module: "SYNTHETIC-MIB", BaseType: gomib.BaseOctetString, Hint: "64a",
		NetSnmp: "DISPLAY-HINT 64a (syntheticAugmentName)"},
	{Name: "SyntheticPath", Module: "SYNTHETIC-MIB", BaseType: gomib.BaseOctetString, Hint: "128a",
		NetSnmp: "DISPLAY-HINT 128a (syntheticSWRunPath)"},
	{Name: "SyntheticKBytes", Module: "SYNTHETIC-MIB", BaseType: gomib.BaseInteger32, Hint: "d",
		NetSnmp: "DISPLAY-HINT d (syntheticMemorySize)"},
	{Name: "SyntheticFixedOctetString", Module: "SYNTHETIC-MIB", BaseType: gomib.BaseOctetString, Hint: "8x",
		NetSnmp: "DISPLAY-HINT 8x (syntheticFixedId)"},
	{Name: "SyntheticIndexInetAddress", Module: "SYNTHETIC-MIB", BaseType: gomib.BaseOctetString, Hint: "1x:",
		NetSnmp: "DISPLAY-HINT 1x: (syntheticConnLocalAddress)"},

	// TCs from SYNTHETICTYPES-MIB
	{Name: "SyntheticTypeCode", Module: "SYNTHETICTYPES-MIB", BaseType: gomib.BaseInteger32, Hint: "d",
		NetSnmp: "DISPLAY-HINT d (syntheticTypeCodeValue)"},
	{Name: "SyntheticInetPortNumber", Module: "SYNTHETICTYPES-MIB", BaseType: gomib.BaseUnsigned32, Hint: "d",
		NetSnmp: "DISPLAY-HINT d (syntheticConnLocalPort)"},
}

func TestTextualConventions(t *testing.T) {
	if len(tcTests) == 0 {
		t.Skip("no textual convention test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range tcTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			// Find the type by name in the module
			mod := m.Module(tc.Module)
			testutil.NotNil(t, mod, "module %s should exist", tc.Module)

			// Search for the type in the module's types
			var found gomib.Type
			for _, typ := range mod.Types() {
				if typ.Name() == tc.Name {
					found = typ
					break
				}
			}
			testutil.NotNil(t, found, "type %s should exist in module %s", tc.Name, tc.Module)

			testutil.Equal(t, tc.BaseType, found.Base(), "base type mismatch")
			if tc.Hint != "" {
				testutil.Equal(t, tc.Hint, found.DisplayHint(), "display hint mismatch")
			}
			if tc.Description != "" {
				testutil.Contains(t, found.Description(), tc.Description, "description mismatch")
			}
		})
	}
}
