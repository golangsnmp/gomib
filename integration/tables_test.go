package integration

import (
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/internal/testutil"
)

// TableTestCase defines a test case for table structure verification.
// Verify expected values with: snmptranslate -m <MODULE> -Td <rowName>
type TableTestCase struct {
	TableName  string   // table name
	RowName    string   // row entry name
	Module     string   // module name
	IndexNames []string // expected index column names in order
	HasImplied bool     // whether last index is IMPLIED
	NetSnmp    string   // snmptranslate command used for verification
}

// tableTests contains all table structure test cases.
//
// Verified against net-snmp 5.9.4: snmptranslate -Td -m SYNTHETIC-MIB SYNTHETIC-MIB::<entryName>
var tableTests = []TableTestCase{
	// Simple table with single index
	{TableName: "syntheticSimpleTable", RowName: "syntheticSimpleEntry", Module: "SYNTHETIC-MIB",
		IndexNames: []string{"syntheticSimpleIndex"}, HasImplied: false,
		NetSnmp: "INDEX { syntheticSimpleIndex }"},

	// Complex table with 2-part index (Integer32 + IpAddress)
	{TableName: "syntheticComplexTable", RowName: "syntheticComplexEntry", Module: "SYNTHETIC-MIB",
		IndexNames: []string{"syntheticComplexGroup", "syntheticComplexAddress"}, HasImplied: false,
		NetSnmp: "INDEX { syntheticComplexGroup, syntheticComplexAddress }"},

	// OID table with single index
	{TableName: "syntheticOidTable", RowName: "syntheticOidEntry", Module: "SYNTHETIC-MIB",
		IndexNames: []string{"syntheticOidIndex"}, HasImplied: false,
		NetSnmp: "INDEX { syntheticOidIndex }"},

	// Software table with single index
	{TableName: "syntheticSWRunTable", RowName: "syntheticSWRunEntry", Module: "SYNTHETIC-MIB",
		IndexNames: []string{"syntheticSWRunIndex"}, HasImplied: false,
		NetSnmp: "INDEX { syntheticSWRunIndex }"},

	// Connection table with 6-part index
	{TableName: "syntheticConnectionTable", RowName: "syntheticConnectionEntry", Module: "SYNTHETIC-MIB",
		IndexNames: []string{
			"syntheticConnLocalAddressType", "syntheticConnLocalAddress", "syntheticConnLocalPort",
			"syntheticConnRemAddressType", "syntheticConnRemAddress", "syntheticConnRemPort",
		}, HasImplied: false,
		NetSnmp: "INDEX { syntheticConnLocalAddressType, syntheticConnLocalAddress, syntheticConnLocalPort, syntheticConnRemAddressType, syntheticConnRemAddress, syntheticConnRemPort }"},

	// FDB table with MacAddress index
	{TableName: "syntheticFdbTable", RowName: "syntheticFdbEntry", Module: "SYNTHETIC-MIB",
		IndexNames: []string{"syntheticFdbAddress"}, HasImplied: false,
		NetSnmp: "INDEX { syntheticFdbAddress }"},
}

func TestTableStructure(t *testing.T) {
	if len(tableTests) == 0 {
		t.Skip("no table test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range tableTests {
		t.Run(tc.Module+"::"+tc.TableName, func(t *testing.T) {
			// Verify table node
			tableNode := getNode(t, m, tc.Module, tc.TableName)
			testutil.Equal(t, gomib.KindTable, tableNode.Kind, "should be a table")

			// Verify row node
			rowNode := getNode(t, m, tc.Module, tc.RowName)
			testutil.Equal(t, gomib.KindRow, rowNode.Kind, "should be a row")

			// Verify row is child of table
			testutil.True(t, tableNode == rowNode.Parent, "row should be child of table")

			// Verify index
			obj := getObject(t, m, tc.Module, tc.RowName)
			testutil.NotEmpty(t, obj.Index, "row should have an INDEX clause")
			testutil.Len(t, obj.Index, len(tc.IndexNames), "index count mismatch")

			for i, expectedName := range tc.IndexNames {
				testutil.NotNil(t, obj.Index[i].Object, "index %d should be resolved", i)
				testutil.Equal(t, expectedName, obj.Index[i].Object.Name, "index %d name mismatch", i)
			}

			// Verify IMPLIED
			if tc.HasImplied {
				lastIdx := len(obj.Index) - 1
				hasImplied := false
				for _, idx := range obj.Index {
					if idx.Implied {
						hasImplied = true
						break
					}
				}
				testutil.True(t, hasImplied, "should have IMPLIED index")
				testutil.True(t, obj.Index[lastIdx].Implied, "last index should be IMPLIED")
			}
		})
	}
}

// AugmentsTestCase defines a test case for AUGMENTS verification.
type AugmentsTestCase struct {
	RowName     string // row that augments another
	Module      string
	AugmentsRow string // name of the augmented row
	AugmentsMod string // module of the augmented row
	NetSnmp     string
}

// augmentsTests contains AUGMENTS test cases.
//
// Verified against net-snmp 5.9.4: snmptranslate -Td -m SYNTHETIC-MIB SYNTHETIC-MIB::syntheticAugmentEntry
// shows: AUGMENTS { syntheticSimpleEntry }
var augmentsTests = []AugmentsTestCase{
	{RowName: "syntheticAugmentEntry", Module: "SYNTHETIC-MIB",
		AugmentsRow: "syntheticSimpleEntry", AugmentsMod: "SYNTHETIC-MIB",
		NetSnmp: "AUGMENTS { syntheticSimpleEntry }"},
}

func TestAugments(t *testing.T) {
	if len(augmentsTests) == 0 {
		t.Skip("no AUGMENTS test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range augmentsTests {
		t.Run(tc.Module+"::"+tc.RowName, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.RowName)
			testutil.NotNil(t, obj.Augments, "should have AUGMENTS")

			augObj := obj.Augments
			testutil.Equal(t, tc.AugmentsRow, augObj.Name, "augmented row name mismatch")
		})
	}
}

// ColumnTestCase defines a test case for column verification within a table.
type ColumnTestCase struct {
	Name      string // column name
	Module    string
	TableName string // parent table name
	NetSnmp   string
}

// columnTests contains column test cases.
//
// Verified via OID hierarchy in net-snmp output.
var columnTests = []ColumnTestCase{
	// Simple table columns
	{Name: "syntheticSimpleIndex", Module: "SYNTHETIC-MIB", TableName: "syntheticSimpleTable",
		NetSnmp: "::= { syntheticSimpleEntry 1 }"},
	{Name: "syntheticSimpleStatus", Module: "SYNTHETIC-MIB", TableName: "syntheticSimpleTable",
		NetSnmp: "::= { syntheticSimpleEntry 2 }"},
	{Name: "syntheticSimpleData", Module: "SYNTHETIC-MIB", TableName: "syntheticSimpleTable",
		NetSnmp: "::= { syntheticSimpleEntry 3 }"},
	{Name: "syntheticSimpleRowStatus", Module: "SYNTHETIC-MIB", TableName: "syntheticSimpleTable",
		NetSnmp: "::= { syntheticSimpleEntry 4 }"},
	{Name: "syntheticPortBitmask", Module: "SYNTHETIC-MIB", TableName: "syntheticSimpleTable",
		NetSnmp: "::= { syntheticSimpleEntry 5 }"},

	// Augment table columns
	{Name: "syntheticAugmentName", Module: "SYNTHETIC-MIB", TableName: "syntheticAugmentTable",
		NetSnmp: "::= { syntheticAugmentEntry 1 }"},
	{Name: "syntheticAugmentHCData", Module: "SYNTHETIC-MIB", TableName: "syntheticAugmentTable",
		NetSnmp: "::= { syntheticAugmentEntry 2 }"},
	{Name: "syntheticAugmentPhysAddress", Module: "SYNTHETIC-MIB", TableName: "syntheticAugmentTable",
		NetSnmp: "::= { syntheticAugmentEntry 3 }"},

	// Complex table columns
	{Name: "syntheticComplexGroup", Module: "SYNTHETIC-MIB", TableName: "syntheticComplexTable",
		NetSnmp: "::= { syntheticComplexEntry 1 }"},
	{Name: "syntheticComplexAddress", Module: "SYNTHETIC-MIB", TableName: "syntheticComplexTable",
		NetSnmp: "::= { syntheticComplexEntry 2 }"},
	{Name: "syntheticComplexValue", Module: "SYNTHETIC-MIB", TableName: "syntheticComplexTable",
		NetSnmp: "::= { syntheticComplexEntry 3 }"},
	{Name: "syntheticComplexTimestamp", Module: "SYNTHETIC-MIB", TableName: "syntheticComplexTable",
		NetSnmp: "::= { syntheticComplexEntry 4 }"},

	// Connection table columns
	{Name: "syntheticConnState", Module: "SYNTHETIC-MIB", TableName: "syntheticConnectionTable",
		NetSnmp: "::= { syntheticConnectionEntry 7 }"},
	{Name: "syntheticConnProcessId", Module: "SYNTHETIC-MIB", TableName: "syntheticConnectionTable",
		NetSnmp: "::= { syntheticConnectionEntry 8 }"},

	// FDB table columns
	{Name: "syntheticFdbAddress", Module: "SYNTHETIC-MIB", TableName: "syntheticFdbTable",
		NetSnmp: "::= { syntheticFdbEntry 1 }"},
	{Name: "syntheticFdbPort", Module: "SYNTHETIC-MIB", TableName: "syntheticFdbTable",
		NetSnmp: "::= { syntheticFdbEntry 2 }"},
	{Name: "syntheticFdbEntryStatus", Module: "SYNTHETIC-MIB", TableName: "syntheticFdbTable",
		NetSnmp: "::= { syntheticFdbEntry 3 }"},
}

func TestColumns(t *testing.T) {
	if len(columnTests) == 0 {
		t.Skip("no column test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range columnTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			node := getNode(t, m, tc.Module, tc.Name)
			testutil.Equal(t, gomib.KindColumn, node.Kind, "should be a column")

			// Verify ancestry: column -> row -> table
			testutil.NotNil(t, node.Parent, "column should have parent (row)")
			testutil.Equal(t, gomib.KindRow, node.Parent.Kind, "parent should be row")
			testutil.NotNil(t, node.Parent.Parent, "row should have parent (table)")
			testutil.Equal(t, gomib.KindTable, node.Parent.Parent.Kind, "grandparent should be table")
			testutil.Equal(t, tc.TableName, node.Parent.Parent.Name, "table name mismatch")
		})
	}
}
