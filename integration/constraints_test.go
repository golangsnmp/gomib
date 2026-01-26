package integration

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// SizeTestCase defines a test case for SIZE constraint verification.
// Verify expected values with: snmptranslate -m <MODULE> -Td <name>
type SizeTestCase struct {
	Name    string // object name
	Module  string // module name
	MinSize int64  // expected minimum size
	MaxSize int64  // expected maximum size
	NetSnmp string // net-snmp output for reference
}

// sizeTests contains SIZE constraint test cases.
//
// Verified against net-snmp 5.9.4: snmptranslate -Td -m SYNTHETIC-MIB SYNTHETIC-MIB::<name>
// Note: Tests use the effective size constraint from the type chain.
var sizeTests = []SizeTestCase{
	// === TC-level constraints (these work) ===

	// syntheticSystemDescription uses DisplayString: OCTET STRING (0..255)
	{Name: "syntheticSystemDescription", Module: "SYNTHETIC-MIB", MinSize: 0, MaxSize: 255,
		NetSnmp: "SYNTAX OCTET STRING (0..255)"},

	// syntheticAugmentName uses SyntheticName: OCTET STRING (SIZE (0..64))
	{Name: "syntheticAugmentName", Module: "SYNTHETIC-MIB", MinSize: 0, MaxSize: 64,
		NetSnmp: "SYNTAX OCTET STRING (0..64)"},

	// syntheticFixedId uses SyntheticFixedOctetString: OCTET STRING (SIZE (8))
	{Name: "syntheticFixedId", Module: "SYNTHETIC-MIB", MinSize: 8, MaxSize: 8,
		NetSnmp: "SYNTAX OCTET STRING (8)"},

	// syntheticFdbAddress uses MacAddress: OCTET STRING (SIZE (6))
	{Name: "syntheticFdbAddress", Module: "SYNTHETIC-MIB", MinSize: 6, MaxSize: 6,
		NetSnmp: "SYNTAX OCTET STRING (6)"},

	// === Inline constraints ===

	// syntheticPortBitmask: OCTET STRING (SIZE (0..64)) - inline, not from TC
	{Name: "syntheticPortBitmask", Module: "SYNTHETIC-MIB", MinSize: 0, MaxSize: 64,
		NetSnmp: "SYNTAX OCTET STRING (0..64)"},
}

func TestSizeConstraints(t *testing.T) {
	if len(sizeTests) == 0 {
		t.Skip("no SIZE test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range sizeTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.Name)
			testutil.NotNil(t, obj.Type(), "object should have a resolved type")

			// Size is pre-computed on the object (effective value from inline or type chain)
			testutil.NotEmpty(t, obj.EffectiveSizes(), "should have a SIZE constraint")

			// Use first range (most common case)
			minSize := obj.EffectiveSizes()[0].Min
			maxSize := obj.EffectiveSizes()[0].Max

			testutil.Equal(t, tc.MinSize, minSize, "min size mismatch")
			testutil.Equal(t, tc.MaxSize, maxSize, "max size mismatch")
		})
	}
}

// RangeTestCase defines a test case for value range constraint verification.
type RangeTestCase struct {
	Name     string // object name
	Module   string // module name
	MinValue int64  // expected minimum value
	MaxValue int64  // expected maximum value
	NetSnmp  string // net-snmp output for reference
}

// rangeTests contains value range test cases.
//
// Verified against net-snmp 5.9.4: snmptranslate -Td -m SYNTHETIC-MIB SYNTHETIC-MIB::<name>
var rangeTests = []RangeTestCase{
	// === TC-level constraints (these work) ===

	// syntheticMemorySize uses SyntheticKBytes: Integer32 (0..2147483647)
	{Name: "syntheticMemorySize", Module: "SYNTHETIC-MIB", MinValue: 0, MaxValue: 2147483647,
		NetSnmp: "SYNTAX Integer32 (0..2147483647)"},

	// syntheticConfigSerial uses TestAndIncr: INTEGER (0..2147483647)
	{Name: "syntheticConfigSerial", Module: "SYNTHETIC-MIB", MinValue: 0, MaxValue: 2147483647,
		NetSnmp: "SYNTAX INTEGER (0..2147483647)"},

	// syntheticTypeCodeValue uses SyntheticTypeCode: Integer32 (0..255)
	{Name: "syntheticTypeCodeValue", Module: "SYNTHETIC-MIB", MinValue: 0, MaxValue: 255,
		NetSnmp: "SYNTAX Integer32 (0..255)"},

	// syntheticConnLocalPort uses SyntheticInetPortNumber: Unsigned32 (0..65535)
	{Name: "syntheticConnLocalPort", Module: "SYNTHETIC-MIB", MinValue: 0, MaxValue: 65535,
		NetSnmp: "SYNTAX Unsigned32 (0..65535)"},

	// === Inline constraints (fixed - stored on object, queried via EffectiveValueRange) ===

	// syntheticSimpleIndex: Unsigned32 (1..65535) - inline
	{Name: "syntheticSimpleIndex", Module: "SYNTHETIC-MIB", MinValue: 1, MaxValue: 65535,
		NetSnmp: "SYNTAX Unsigned32 (1..65535)"},

	// syntheticComplexGroup: Integer32 (1..255) - inline
	{Name: "syntheticComplexGroup", Module: "SYNTHETIC-MIB", MinValue: 1, MaxValue: 255,
		NetSnmp: "SYNTAX Integer32 (1..255)"},

	// syntheticFdbPort: Integer32 (0..65535) - inline
	{Name: "syntheticFdbPort", Module: "SYNTHETIC-MIB", MinValue: 0, MaxValue: 65535,
		NetSnmp: "SYNTAX Integer32 (0..65535)"},

	// syntheticOidIndex: Integer32 (1..2147483647) - inline
	{Name: "syntheticOidIndex", Module: "SYNTHETIC-MIB", MinValue: 1, MaxValue: 2147483647,
		NetSnmp: "SYNTAX Integer32 (1..2147483647)"},

	// syntheticSWRunIndex: Integer32 (1..2147483647) - inline
	{Name: "syntheticSWRunIndex", Module: "SYNTHETIC-MIB", MinValue: 1, MaxValue: 2147483647,
		NetSnmp: "SYNTAX Integer32 (1..2147483647)"},
}

func TestRangeConstraints(t *testing.T) {
	if len(rangeTests) == 0 {
		t.Skip("no range test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range rangeTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.Name)
			testutil.NotNil(t, obj.Type(), "object should have a resolved type")

			// ValueRange is pre-computed on the object (effective value from inline or type chain)
			testutil.NotEmpty(t, obj.EffectiveRanges(), "should have a value range constraint")

			// Use first range (most common case)
			minVal := obj.EffectiveRanges()[0].Min
			maxVal := obj.EffectiveRanges()[0].Max

			testutil.Equal(t, tc.MinValue, minVal, "min value mismatch")
			testutil.Equal(t, tc.MaxValue, maxVal, "max value mismatch")
		})
	}
}

// DisplayHintTestCase defines a test case for DISPLAY-HINT verification.
type DisplayHintTestCase struct {
	Name    string // object name
	Module  string // module name
	Hint    string // expected display hint
	NetSnmp string
}

// hintTests contains DISPLAY-HINT test cases.
//
// Verified against net-snmp 5.9.4: snmptranslate -Td -m SYNTHETIC-MIB SYNTHETIC-MIB::<name>
// The DISPLAY-HINT is inherited from the TEXTUAL-CONVENTION.
var hintTests = []DisplayHintTestCase{
	// syntheticSystemDescription uses DisplayString: DISPLAY-HINT "255a"
	{Name: "syntheticSystemDescription", Module: "SYNTHETIC-MIB", Hint: "255a",
		NetSnmp: "DISPLAY-HINT 255a"},

	// syntheticAugmentName uses SyntheticName: DISPLAY-HINT "64a"
	{Name: "syntheticAugmentName", Module: "SYNTHETIC-MIB", Hint: "64a",
		NetSnmp: "DISPLAY-HINT 64a"},

	// syntheticMemorySize uses SyntheticKBytes: DISPLAY-HINT "d"
	{Name: "syntheticMemorySize", Module: "SYNTHETIC-MIB", Hint: "d",
		NetSnmp: "DISPLAY-HINT d"},

	// syntheticTypeCodeValue uses SyntheticTypeCode: DISPLAY-HINT "d"
	{Name: "syntheticTypeCodeValue", Module: "SYNTHETIC-MIB", Hint: "d",
		NetSnmp: "DISPLAY-HINT d"},

	// syntheticFixedId uses SyntheticFixedOctetString: DISPLAY-HINT "8x"
	{Name: "syntheticFixedId", Module: "SYNTHETIC-MIB", Hint: "8x",
		NetSnmp: "DISPLAY-HINT 8x"},

	// syntheticConnLocalPort uses SyntheticInetPortNumber: DISPLAY-HINT "d"
	{Name: "syntheticConnLocalPort", Module: "SYNTHETIC-MIB", Hint: "d",
		NetSnmp: "DISPLAY-HINT d"},

	// syntheticConnLocalAddress uses SyntheticIndexInetAddress: DISPLAY-HINT "1x:"
	{Name: "syntheticConnLocalAddress", Module: "SYNTHETIC-MIB", Hint: "1x:",
		NetSnmp: "DISPLAY-HINT 1x:"},

	// syntheticAugmentPhysAddress uses PhysAddress: DISPLAY-HINT "1x:"
	{Name: "syntheticAugmentPhysAddress", Module: "SYNTHETIC-MIB", Hint: "1x:",
		NetSnmp: "DISPLAY-HINT 1x:"},

	// syntheticFdbAddress uses MacAddress: DISPLAY-HINT "1x:"
	{Name: "syntheticFdbAddress", Module: "SYNTHETIC-MIB", Hint: "1x:",
		NetSnmp: "DISPLAY-HINT 1x:"},

	// syntheticInstallDate uses DateAndTime: DISPLAY-HINT "2d-1d-1d,1d:1d:1d.1d,1a1d:1d"
	{Name: "syntheticInstallDate", Module: "SYNTHETIC-MIB", Hint: "2d-1d-1d,1d:1d:1d.1d,1a1d:1d",
		NetSnmp: "DISPLAY-HINT 2d-1d-1d,1d:1d:1d.1d,1a1d:1d"},
}

func TestDisplayHints(t *testing.T) {
	if len(hintTests) == 0 {
		t.Skip("no display hint test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range hintTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.Name)
			testutil.NotNil(t, obj.Type(), "object should have a resolved type")

			// Hint is pre-computed on the object (effective value from inline or type chain)
			testutil.Equal(t, tc.Hint, obj.EffectiveDisplayHint(), "display hint mismatch")
		})
	}
}
