package gomib_test

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func loadIndexDecodeMIB(t testing.TB) *mib.Mib {
	t.Helper()
	return loadProblemMIB(t, "PROBLEM-INDEX-DECODE-MIB")
}

func TestDecodeSuffixInteger(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "simpleEntry")
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{42})

	testutil.Equal(t, 1, len(decoded), "decoded count")
	testutil.Equal(t, "simpleIndex", decoded[0].Name(), "name")
	testutil.Equal(t, mib.IndexValueInteger, decoded[0].Value().Kind(), "kind")
	n, ok := decoded[0].Value().Integer()
	testutil.True(t, ok, "Integer() ok")
	testutil.Equal(t, uint32(42), n, "value")
}

func TestDecodeSuffixIpAddress(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "ipEntry")
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{10, 0, 1, 99})

	testutil.Equal(t, 1, len(decoded), "decoded count")
	testutil.Equal(t, "ipAddr", decoded[0].Name(), "name")
	ip, ok := decoded[0].Value().IpAddress()
	testutil.True(t, ok, "IpAddress() ok")
	testutil.Equal(t, [4]byte{10, 0, 1, 99}, ip, "value")
	testutil.Equal(t, "10.0.1.99", decoded[0].Value().String(), "string")
}

func TestDecodeSuffixFixedString(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "fixedEntry")
	// MAC address AA:BB:CC:DD:EE:FF -> 6 arcs
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF})

	testutil.Equal(t, 1, len(decoded), "decoded count")
	testutil.Equal(t, "fixedAddr", decoded[0].Name(), "name")
	b, ok := decoded[0].Value().Bytes()
	testutil.True(t, ok, "Bytes() ok")
	testutil.Equal(t, 6, len(b), "length")
	testutil.Equal(t, "AA:BB:CC:DD:EE:FF", decoded[0].Value().String(), "string")
}

func TestDecodeSuffixVariableStringLengthPrefixed(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "varEntry")
	// "eth0" = 4 chars, length-prefixed: [4, 101, 116, 104, 48]
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{4, 101, 116, 104, 48})

	testutil.Equal(t, 1, len(decoded), "decoded count")
	testutil.Equal(t, "varName", decoded[0].Name(), "name")
	b, ok := decoded[0].Value().Bytes()
	testutil.True(t, ok, "Bytes() ok")
	testutil.SliceEqual(t, []byte("eth0"), b, "value")
	testutil.Equal(t, "eth0", decoded[0].Value().String(), "string")
}

func TestDecodeSuffixVariableStringEmpty(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "varEntry")
	// Empty string: length prefix = 0
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{0})

	testutil.Equal(t, 1, len(decoded), "decoded count")
	testutil.Equal(t, "varName", decoded[0].Name(), "name")
	b, ok := decoded[0].Value().Bytes()
	testutil.True(t, ok, "Bytes() ok")
	testutil.Equal(t, 0, len(b), "empty bytes")
}

func TestDecodeSuffixMultiIndex(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "multiEntry")
	// slot=3, addr=192.168.1.1
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{3, 192, 168, 1, 1})

	testutil.Equal(t, 2, len(decoded), "decoded count")
	testutil.Equal(t, "multiSlot", decoded[0].Name(), "name[0]")
	n, _ := decoded[0].Value().Integer()
	testutil.Equal(t, uint32(3), n, "value[0]")
	testutil.Equal(t, "multiAddr", decoded[1].Name(), "name[1]")
	ip, _ := decoded[1].Value().IpAddress()
	testutil.Equal(t, [4]byte{192, 168, 1, 1}, ip, "value[1]")
}

func TestDecodeSuffixImpliedString(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "impliedEntry")
	// "test" without length prefix (IMPLIED)
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{116, 101, 115, 116})

	testutil.Equal(t, 1, len(decoded), "decoded count")
	testutil.Equal(t, "impliedName", decoded[0].Name(), "name")
	b, ok := decoded[0].Value().Bytes()
	testutil.True(t, ok, "Bytes() ok")
	testutil.SliceEqual(t, []byte("test"), b, "value")
	testutil.Equal(t, "test", decoded[0].Value().String(), "string")
}

func TestDecodeSuffixOIDLengthPrefixed(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "oidEntry")
	// OID 1.3.6.1 = 4 arcs, length-prefixed: [4, 1, 3, 6, 1]
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{4, 1, 3, 6, 1})

	testutil.Equal(t, 1, len(decoded), "decoded count")
	testutil.Equal(t, "oidIndex", decoded[0].Name(), "name")
	oid, ok := decoded[0].Value().OID()
	testutil.True(t, ok, "OID() ok")
	testutil.SliceEqual(t, mib.OID{1, 3, 6, 1}, oid, "value")
	testutil.Equal(t, "1.3.6.1", decoded[0].Value().String(), "string")
}

func TestDecodeSuffixImpliedOID(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "impliedOidEntry")
	// OID 1.3.6.1.4 without length prefix (IMPLIED)
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{1, 3, 6, 1, 4})

	testutil.Equal(t, 1, len(decoded), "decoded count")
	testutil.Equal(t, "impliedOidIndex", decoded[0].Name(), "name")
	oid, ok := decoded[0].Value().OID()
	testutil.True(t, ok, "OID() ok")
	testutil.SliceEqual(t, mib.OID{1, 3, 6, 1, 4}, oid, "value")
}

func TestDecodeSuffixInsufficientArcs(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "ipEntry")
	// IpAddress needs 4 arcs, only 2 provided
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{10, 0})
	testutil.Equal(t, 0, len(decoded), "should be empty")
}

func TestDecodeSuffixEmpty(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "simpleEntry")
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{})
	testutil.Equal(t, 0, len(decoded), "should be empty")
}

func TestDecodeSuffixExtraArcsIgnored(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "simpleEntry")
	// One integer index, extra arcs beyond it
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{42, 99, 100})
	testutil.Equal(t, 1, len(decoded), "decoded count")
	n, _ := decoded[0].Value().Integer()
	testutil.Equal(t, uint32(42), n, "value")
}

func TestDecodeSuffixMultiIndexPartial(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "multiEntry")
	// slot=3, then only 2 arcs for IP (needs 4) -> partial
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{3, 10, 0})
	testutil.Equal(t, 1, len(decoded), "decoded count")
	n, _ := decoded[0].Value().Integer()
	testutil.Equal(t, uint32(3), n, "value")
}

func TestDecodeSuffixLengthPrefixedTruncated(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	row := requireObject(t, m, "varEntry")
	// Length says 10 but only 3 data arcs follow
	decoded := mib.DecodeSuffix(row.EffectiveIndexes(), mib.OID{10, 65, 66, 67})
	testutil.Equal(t, 0, len(decoded), "should be empty")
}

func TestLookupInstanceDecodeIndexes(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	oid, err := m.ResolveOID("simpleValue.42")
	if err != nil {
		t.Fatalf("ResolveOID: %v", err)
	}
	lookup := m.LookupInstance(oid)
	testutil.Equal(t, "simpleValue", lookup.Node().Name(), "node name")
	testutil.SliceEqual(t, mib.OID{42}, lookup.Suffix(), "suffix")

	decoded := lookup.DecodeIndexes()
	testutil.Equal(t, 1, len(decoded), "decoded count")
	testutil.Equal(t, "simpleIndex", decoded[0].Name(), "name")
	n, _ := decoded[0].Value().Integer()
	testutil.Equal(t, uint32(42), n, "value")
}

func TestLookupInstanceDecodeNonColumn(t *testing.T) {
	m := loadIndexDecodeMIB(t)
	oid, err := m.ResolveOID("simpleTable")
	if err != nil {
		t.Fatalf("ResolveOID: %v", err)
	}
	lookup := m.LookupInstance(oid)
	decoded := lookup.DecodeIndexes()
	testutil.Equal(t, 0, len(decoded), "should be empty for non-column")
}
