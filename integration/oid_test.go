package integration

import (
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/internal/testutil"
)

// OidTestCase defines a test case for OID resolution.
// Verify expected values with: snmptranslate -m <MODULE> -On <name>
type OidTestCase struct {
	Name    string     // object/node name
	Module  string     // module name
	Oid     string     // expected OID in dotted notation
	Kind    gomib.Kind // expected node kind
	NetSnmp string     // snmptranslate command used for verification
}

// oidTests contains all OID resolution test cases.
// Add new cases after verifying with snmptranslate.
//
// Verified against net-snmp 5.9.4 with:
//
//	MIBDIRS="+.../synthetic:.../ietf:.../iana" MIBS="SYNTHETIC-MIB:SYNTHETICTYPES-MIB"
var oidTests = []OidTestCase{
	// === SYNTHETIC-MIB Module Identity ===
	{Name: "syntheticMIB", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999", Kind: gomib.KindNode,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticMIB -> .1.3.6.1.2.1.999"},

	// === Scalars ===
	{Name: "syntheticSystemDescription", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.1", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticSystemDescription -> .1.3.6.1.2.1.999.1.1"},
	{Name: "syntheticSystemObjectID", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.2", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticSystemObjectID -> .1.3.6.1.2.1.999.1.2"},
	{Name: "syntheticSystemUpTime", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.3", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticSystemUpTime -> .1.3.6.1.2.1.999.1.3"},
	{Name: "syntheticConfigSerial", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.4", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticConfigSerial -> .1.3.6.1.2.1.999.1.4"},
	{Name: "syntheticTrapEnable", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.5", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticTrapEnable -> .1.3.6.1.2.1.999.1.5"},
	{Name: "syntheticMemorySize", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.6", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticMemorySize -> .1.3.6.1.2.1.999.1.6"},
	{Name: "syntheticLastChange", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.7", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticLastChange -> .1.3.6.1.2.1.999.1.7"},
	{Name: "syntheticErrorState", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.8", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticErrorState -> .1.3.6.1.2.1.999.1.8"},
	{Name: "syntheticBootStatus", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.9", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticBootStatus -> .1.3.6.1.2.1.999.1.9"},
	{Name: "syntheticDeviceType", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.10", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticDeviceType -> .1.3.6.1.2.1.999.1.10"},
	{Name: "syntheticInstallDate", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.11", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticInstallDate -> .1.3.6.1.2.1.999.1.11"},
	{Name: "syntheticTypeCodeValue", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.12", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticTypeCodeValue -> .1.3.6.1.2.1.999.1.12"},
	{Name: "syntheticFixedId", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.1.13", Kind: gomib.KindScalar,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticFixedId -> .1.3.6.1.2.1.999.1.13"},

	// === Simple Table ===
	{Name: "syntheticSimpleTable", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.2.1", Kind: gomib.KindTable,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticSimpleTable -> .1.3.6.1.2.1.999.2.1"},
	{Name: "syntheticSimpleEntry", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.2.1.1", Kind: gomib.KindRow,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticSimpleEntry -> .1.3.6.1.2.1.999.2.1.1"},
	{Name: "syntheticSimpleIndex", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.2.1.1.1", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticSimpleIndex -> .1.3.6.1.2.1.999.2.1.1.1"},
	{Name: "syntheticSimpleStatus", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.2.1.1.2", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticSimpleStatus -> .1.3.6.1.2.1.999.2.1.1.2"},
	{Name: "syntheticSimpleData", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.2.1.1.3", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticSimpleData -> .1.3.6.1.2.1.999.2.1.1.3"},
	{Name: "syntheticSimpleRowStatus", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.2.1.1.4", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticSimpleRowStatus -> .1.3.6.1.2.1.999.2.1.1.4"},
	{Name: "syntheticPortBitmask", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.2.1.1.5", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticPortBitmask -> .1.3.6.1.2.1.999.2.1.1.5"},

	// === Augment Table ===
	{Name: "syntheticAugmentTable", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.3.1", Kind: gomib.KindTable,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticAugmentTable -> .1.3.6.1.2.1.999.3.1"},
	{Name: "syntheticAugmentEntry", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.3.1.1", Kind: gomib.KindRow,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticAugmentEntry -> .1.3.6.1.2.1.999.3.1.1"},
	{Name: "syntheticAugmentName", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.3.1.1.1", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticAugmentName -> .1.3.6.1.2.1.999.3.1.1.1"},
	{Name: "syntheticAugmentHCData", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.3.1.1.2", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticAugmentHCData -> .1.3.6.1.2.1.999.3.1.1.2"},
	{Name: "syntheticAugmentPhysAddress", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.3.1.1.3", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticAugmentPhysAddress -> .1.3.6.1.2.1.999.3.1.1.3"},

	// === Complex Table (multi-part index) ===
	{Name: "syntheticComplexTable", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.4.1", Kind: gomib.KindTable,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticComplexTable -> .1.3.6.1.2.1.999.4.1"},
	{Name: "syntheticComplexEntry", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.4.1.1", Kind: gomib.KindRow,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticComplexEntry -> .1.3.6.1.2.1.999.4.1.1"},
	{Name: "syntheticComplexGroup", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.4.1.1.1", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticComplexGroup -> .1.3.6.1.2.1.999.4.1.1.1"},
	{Name: "syntheticComplexAddress", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.4.1.1.2", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticComplexAddress -> .1.3.6.1.2.1.999.4.1.1.2"},
	{Name: "syntheticComplexValue", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.4.1.1.3", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticComplexValue -> .1.3.6.1.2.1.999.4.1.1.3"},
	{Name: "syntheticComplexTimestamp", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.4.1.1.4", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticComplexTimestamp -> .1.3.6.1.2.1.999.4.1.1.4"},

	// === Connection Table (6-part index) ===
	{Name: "syntheticConnectionTable", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.7.1", Kind: gomib.KindTable,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticConnectionTable -> .1.3.6.1.2.1.999.7.1"},
	{Name: "syntheticConnectionEntry", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.7.1.1", Kind: gomib.KindRow,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticConnectionEntry -> .1.3.6.1.2.1.999.7.1.1"},
	{Name: "syntheticConnLocalAddressType", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.7.1.1.1", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticConnLocalAddressType -> .1.3.6.1.2.1.999.7.1.1.1"},
	{Name: "syntheticConnLocalAddress", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.7.1.1.2", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticConnLocalAddress -> .1.3.6.1.2.1.999.7.1.1.2"},
	{Name: "syntheticConnLocalPort", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.7.1.1.3", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticConnLocalPort -> .1.3.6.1.2.1.999.7.1.1.3"},
	{Name: "syntheticConnRemAddressType", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.7.1.1.4", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticConnRemAddressType -> .1.3.6.1.2.1.999.7.1.1.4"},
	{Name: "syntheticConnRemAddress", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.7.1.1.5", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticConnRemAddress -> .1.3.6.1.2.1.999.7.1.1.5"},
	{Name: "syntheticConnRemPort", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.7.1.1.6", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticConnRemPort -> .1.3.6.1.2.1.999.7.1.1.6"},
	{Name: "syntheticConnState", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.7.1.1.7", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticConnState -> .1.3.6.1.2.1.999.7.1.1.7"},
	{Name: "syntheticConnProcessId", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.7.1.1.8", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticConnProcessId -> .1.3.6.1.2.1.999.7.1.1.8"},

	// === FDB Table (MacAddress index) ===
	{Name: "syntheticFdbTable", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.8.1", Kind: gomib.KindTable,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticFdbTable -> .1.3.6.1.2.1.999.8.1"},
	{Name: "syntheticFdbEntry", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.8.1.1", Kind: gomib.KindRow,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticFdbEntry -> .1.3.6.1.2.1.999.8.1.1"},
	{Name: "syntheticFdbAddress", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.8.1.1.1", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticFdbAddress -> .1.3.6.1.2.1.999.8.1.1.1"},
	{Name: "syntheticFdbPort", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.8.1.1.2", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticFdbPort -> .1.3.6.1.2.1.999.8.1.1.2"},
	{Name: "syntheticFdbEntryStatus", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.8.1.1.3", Kind: gomib.KindColumn,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticFdbEntryStatus -> .1.3.6.1.2.1.999.8.1.1.3"},

	// === Notifications ===
	{Name: "syntheticConfigChange", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.14.0.1", Kind: gomib.KindNotification,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticConfigChange -> .1.3.6.1.2.1.999.14.0.1"},
	{Name: "syntheticFailure", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.14.0.2", Kind: gomib.KindNotification,
		NetSnmp: "snmptranslate -On SYNTHETIC-MIB::syntheticFailure -> .1.3.6.1.2.1.999.14.0.2"},

	// === SYNTHETICTYPES-MIB ===
	{Name: "syntheticTypesMIB", Module: "SYNTHETICTYPES-MIB", Oid: "1.3.6.1.3.998", Kind: gomib.KindNode,
		NetSnmp: "snmptranslate -On SYNTHETICTYPES-MIB::syntheticTypesMIB -> .1.3.6.1.3.998"},
}

func TestOidResolution(t *testing.T) {
	if len(oidTests) == 0 {
		t.Skip("no OID test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range oidTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			node := getNode(t, m, tc.Module, tc.Name)

			got := node.OID().String()
			testutil.Equal(t, tc.Oid, got, "OID mismatch")
			testutil.Equal(t, tc.Kind, node.Kind, "kind mismatch")
		})
	}
}
