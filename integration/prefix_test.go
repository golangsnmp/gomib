package integration

import (
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/internal/testutil"
)

// LongestPrefixTestCase defines a test case for longest prefix matching.
type LongestPrefixTestCase struct {
	Query        string     // OID to query (may include instance suffix)
	ExpectedOid  string     // expected OID of the matched node
	ExpectedName string     // expected name of the matched node
	ExpectedKind gomib.Kind // expected kind of the matched node
	Description  string     // what this test case covers
}

var longestPrefixTests = []LongestPrefixTestCase{
	// Column instances - the most common use case for SNMP polling
	{
		Query:        "1.3.6.1.2.1.999.2.1.1.1.5",
		ExpectedOid:  "1.3.6.1.2.1.999.2.1.1.1",
		ExpectedName: "syntheticSimpleIndex",
		ExpectedKind: gomib.KindColumn,
		Description:  "column instance with single index",
	},
	{
		Query:        "1.3.6.1.2.1.999.2.1.1.3.42",
		ExpectedOid:  "1.3.6.1.2.1.999.2.1.1.3",
		ExpectedName: "syntheticSimpleData",
		ExpectedKind: gomib.KindColumn,
		Description:  "column instance with different index",
	},

	// Scalar instances (scalar.0)
	{
		Query:        "1.3.6.1.2.1.999.1.1.0",
		ExpectedOid:  "1.3.6.1.2.1.999.1.1",
		ExpectedName: "syntheticSystemDescription",
		ExpectedKind: gomib.KindScalar,
		Description:  "scalar instance (.0 suffix)",
	},
	{
		Query:        "1.3.6.1.2.1.999.1.3.0",
		ExpectedOid:  "1.3.6.1.2.1.999.1.3",
		ExpectedName: "syntheticSystemUpTime",
		ExpectedKind: gomib.KindScalar,
		Description:  "another scalar instance",
	},

	// Exact match - should return the node itself
	{
		Query:        "1.3.6.1.2.1.999.2.1.1.1",
		ExpectedOid:  "1.3.6.1.2.1.999.2.1.1.1",
		ExpectedName: "syntheticSimpleIndex",
		ExpectedKind: gomib.KindColumn,
		Description:  "exact match returns the node",
	},
	{
		Query:        "1.3.6.1.2.1.999",
		ExpectedOid:  "1.3.6.1.2.1.999",
		ExpectedName: "syntheticMIB",
		ExpectedKind: gomib.KindNode,
		Description:  "exact match of module identity",
	},

	// Multi-component instance index
	{
		Query:        "1.3.6.1.2.1.999.4.1.1.3.1.192.168.1.1",
		ExpectedOid:  "1.3.6.1.2.1.999.4.1.1.3",
		ExpectedName: "syntheticComplexValue",
		ExpectedKind: gomib.KindColumn,
		Description:  "column with multi-component index (group + IP address)",
	},

	// Deep subtree - should find deepest matching node
	{
		Query:        "1.3.6.1.2.1.999.2.1.1.999.999.999",
		ExpectedOid:  "1.3.6.1.2.1.999.2.1.1",
		ExpectedName: "syntheticSimpleEntry",
		ExpectedKind: gomib.KindRow,
		Description:  "non-existent column under row returns row",
	},

	// Partial tree match
	{
		Query:        "1.3.6.1.2.1.999.999.999",
		ExpectedOid:  "1.3.6.1.2.1.999",
		ExpectedName: "syntheticMIB",
		ExpectedKind: gomib.KindNode,
		Description:  "non-existent subtree returns deepest known node",
	},
}

func TestLongestPrefix(t *testing.T) {
	m := loadCorpus(t)

	for _, tc := range longestPrefixTests {
		t.Run(tc.Description, func(t *testing.T) {
			oid, err := gomib.ParseOID(tc.Query)
			testutil.NoError(t, err, "parse OID")

			node := m.LongestPrefixByOID(oid)
			testutil.NotNil(t, node, "should find a matching prefix for %s", tc.Query)

			testutil.Equal(t, tc.ExpectedOid, node.OID().String(), "OID mismatch")
			testutil.Equal(t, tc.ExpectedName, node.Name(), "name mismatch")
			testutil.Equal(t, tc.ExpectedKind, node.Kind(), "kind mismatch")
		})
	}
}

func TestLongestPrefix_NoMatch(t *testing.T) {
	m := loadCorpus(t)

	// OID that doesn't exist at all (wrong top-level arc)
	oid, _ := gomib.ParseOID("9.9.9.9.9")
	node := m.LongestPrefixByOID(oid)
	testutil.Nil(t, node, "should return nil for completely unknown OID tree")
}

func TestLongestPrefix_EmptyOid(t *testing.T) {
	m := loadCorpus(t)

	node := m.LongestPrefixByOID(nil)
	testutil.Nil(t, node, "should return nil for empty OID")
}

func TestLongestPrefix_InvalidOid(t *testing.T) {
	m := loadCorpus(t)

	// Invalid OID string will parse to nil
	oid, _ := gomib.ParseOID("not.an.oid")
	node := m.LongestPrefixByOID(oid)
	testutil.Nil(t, node, "should return nil for invalid OID")
}

func TestLongestPrefixByOID(t *testing.T) {
	m := loadCorpus(t)

	// Test the Oid type variant
	oid := gomib.Oid{1, 3, 6, 1, 2, 1, 999, 2, 1, 1, 1, 5}
	node := m.LongestPrefixByOID(oid)

	testutil.NotNil(t, node, "should find node")
	testutil.Equal(t, "syntheticSimpleIndex", node.Name(), "name mismatch")
	testutil.Equal(t, "1.3.6.1.2.1.999.2.1.1.1", node.OID().String(), "OID mismatch")
}

func TestLongestPrefixByOID_Empty(t *testing.T) {
	m := loadCorpus(t)

	node := m.LongestPrefixByOID(nil)
	testutil.Nil(t, node, "should return nil for nil OID")

	node = m.LongestPrefixByOID(gomib.Oid{})
	testutil.Nil(t, node, "should return nil for empty OID")
}
