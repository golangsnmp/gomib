package integration

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// EnumTestCase defines a test case for enumerated INTEGER verification.
// Verify expected values with: snmptranslate -m <MODULE> -Td <name>
type EnumTestCase struct {
	Name    string           // object name with enum type
	Module  string           // module name
	Values  map[string]int64 // expected enum label -> value mapping
	NetSnmp string
}

// enumTests contains enumerated INTEGER test cases.
//
// Verified against net-snmp 5.9.4: snmptranslate -Td -m SYNTHETIC-MIB SYNTHETIC-MIB::<name>
var enumTests = []EnumTestCase{
	// syntheticTrapEnable: INTEGER {enabled(1), disabled(2)}
	{Name: "syntheticTrapEnable", Module: "SYNTHETIC-MIB",
		Values:  map[string]int64{"enabled": 1, "disabled": 2},
		NetSnmp: "SYNTAX INTEGER {enabled(1), disabled(2)}"},

	// syntheticSimpleStatus uses SyntheticStatus TC: INTEGER {enabled(1), disabled(2), testing(3), unknown(4), down(5)}
	{Name: "syntheticSimpleStatus", Module: "SYNTHETIC-MIB",
		Values:  map[string]int64{"enabled": 1, "disabled": 2, "testing": 3, "unknown": 4, "down": 5},
		NetSnmp: "SYNTAX INTEGER {enabled(1), disabled(2), testing(3), unknown(4), down(5)}"},

	// syntheticBootStatus uses TruthValue TC: INTEGER {true(1), false(2)}
	{Name: "syntheticBootStatus", Module: "SYNTHETIC-MIB",
		Values:  map[string]int64{"true": 1, "false": 2},
		NetSnmp: "SYNTAX INTEGER {true(1), false(2)}"},

	// syntheticSimpleRowStatus uses RowStatus TC
	{Name: "syntheticSimpleRowStatus", Module: "SYNTHETIC-MIB",
		Values: map[string]int64{
			"active": 1, "notInService": 2, "notReady": 3,
			"createAndGo": 4, "createAndWait": 5, "destroy": 6,
		},
		NetSnmp: "SYNTAX INTEGER {active(1), notInService(2), notReady(3), createAndGo(4), createAndWait(5), destroy(6)}"},

	// syntheticConnLocalAddressType uses SyntheticInetAddressType TC: INTEGER {unknown(0), ipv4(1), ipv6(2)}
	{Name: "syntheticConnLocalAddressType", Module: "SYNTHETIC-MIB",
		Values:  map[string]int64{"unknown": 0, "ipv4": 1, "ipv6": 2},
		NetSnmp: "SYNTAX INTEGER {unknown(0), ipv4(1), ipv6(2)}"},

	// syntheticConnState: INTEGER {closed(1), listen(2), synSent(3), established(5)}
	{Name: "syntheticConnState", Module: "SYNTHETIC-MIB",
		Values:  map[string]int64{"closed": 1, "listen": 2, "synSent": 3, "established": 5},
		NetSnmp: "SYNTAX INTEGER {closed(1), listen(2), synSent(3), established(5)}"},

	// syntheticFdbEntryStatus uses SyntheticFdbStatus TC: INTEGER {other(1), invalid(2), learned(3), self(4), mgmt(5)}
	{Name: "syntheticFdbEntryStatus", Module: "SYNTHETIC-MIB",
		Values:  map[string]int64{"other": 1, "invalid": 2, "learned": 3, "self": 4, "mgmt": 5},
		NetSnmp: "SYNTAX INTEGER {other(1), invalid(2), learned(3), self(4), mgmt(5)}"},

	// syntheticSWRunStatus: INTEGER {running(1), runnable(2), notRunnable(3), invalid(4)}
	{Name: "syntheticSWRunStatus", Module: "SYNTHETIC-MIB",
		Values:  map[string]int64{"running": 1, "runnable": 2, "notRunnable": 3, "invalid": 4},
		NetSnmp: "SYNTAX INTEGER {running(1), runnable(2), notRunnable(3), invalid(4)}"},
}

func TestEnumValues(t *testing.T) {
	if len(enumTests) == 0 {
		t.Skip("no enum test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range enumTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.Name)

			// NamedValues contains the effective enum values (pre-computed from inline or type chain)
			testutil.NotEmpty(t, obj.EffectiveEnums(), "should have enum values")

			// Verify expected values are present
			for expectedLabel, expectedValue := range tc.Values {
				found := false
				for _, nv := range obj.EffectiveEnums() {
					if nv.Label == expectedLabel {
						testutil.Equal(t, expectedValue, nv.Value, "enum value mismatch for %s", expectedLabel)
						found = true
						break
					}
				}
				testutil.True(t, found, "enum label %s not found", expectedLabel)
			}
		})
	}
}

// BitsTestCase defines a test case for BITS type verification.
type BitsTestCase struct {
	Name      string           // object name with BITS type
	Module    string           // module name
	Positions map[string]int64 // expected bit label -> position mapping
	NetSnmp   string
}

// bitsTests contains BITS type test cases.
//
// Verified against net-snmp 5.9.4: snmptranslate -Td -m SYNTHETIC-MIB SYNTHETIC-MIB::syntheticErrorState
// shows: SYNTAX BITS {bitZero(0), bitOne(1), bitTwo(2), bitSeven(7)}
var bitsTests = []BitsTestCase{
	// syntheticErrorState uses SyntheticBitmask TC: BITS {bitZero(0), bitOne(1), bitTwo(2), bitSeven(7)}
	{Name: "syntheticErrorState", Module: "SYNTHETIC-MIB",
		Positions: map[string]int64{"bitZero": 0, "bitOne": 1, "bitTwo": 2, "bitSeven": 7},
		NetSnmp:   "SYNTAX BITS {bitZero(0), bitOne(1), bitTwo(2), bitSeven(7)}"},
}

func TestBitsDefinitions(t *testing.T) {
	if len(bitsTests) == 0 {
		t.Skip("no BITS test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range bitsTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.Name)

			// NamedValues contains the effective BITS positions (pre-computed from inline or type chain)
			testutil.NotEmpty(t, obj.EffectiveEnums(), "should have BITS definitions")

			// Verify expected positions are present
			for expectedLabel, expectedPos := range tc.Positions {
				found := false
				for _, nv := range obj.EffectiveEnums() {
					if nv.Label == expectedLabel {
						testutil.Equal(t, expectedPos, nv.Value, "bit position mismatch for %s", expectedLabel)
						found = true
						break
					}
				}
				testutil.True(t, found, "bit label %s not found", expectedLabel)
			}
		})
	}
}
