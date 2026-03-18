package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestIndexValueString(t *testing.T) {
	tests := []struct {
		name  string
		value IndexValue
		want  string
	}{
		{"integer", IndexValue{kind: IndexValueInteger, value: uint32(42)}, "42"},
		{"ip", IndexValue{kind: IndexValueIpAddress, value: [4]byte{192, 168, 1, 1}}, "192.168.1.1"},
		{"printable string", IndexValue{kind: IndexValueOctetString, value: []byte("eth0")}, "eth0"},
		{"binary string", IndexValue{kind: IndexValueOctetString, value: []byte{0x00, 0xFF}}, "00:FF"},
		{"oid", IndexValue{kind: IndexValueObjectIdentifier, value: OID{1, 3, 6, 1}}, "1.3.6.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Equal(t, tt.want, tt.value.String(), "String()")
		})
	}
}

func TestIndexValueAccessors(t *testing.T) {
	t.Run("Integer", func(t *testing.T) {
		v := IndexValue{kind: IndexValueInteger, value: uint32(42)}
		n, ok := v.Integer()
		testutil.True(t, ok, "Integer() ok")
		testutil.Equal(t, uint32(42), n, "Integer() value")

		_, ok = v.IpAddress()
		testutil.True(t, !ok, "IpAddress() should be false")
		_, ok = v.Bytes()
		testutil.True(t, !ok, "Bytes() should be false")
		_, ok = v.OID()
		testutil.True(t, !ok, "OID() should be false")
	})

	t.Run("IpAddress", func(t *testing.T) {
		v := IndexValue{kind: IndexValueIpAddress, value: [4]byte{10, 0, 1, 99}}
		ip, ok := v.IpAddress()
		testutil.True(t, ok, "IpAddress() ok")
		testutil.Equal(t, [4]byte{10, 0, 1, 99}, ip, "IpAddress() value")
		testutil.NotNil(t, v.IP(), "IP() should not be nil")
	})

	t.Run("Bytes", func(t *testing.T) {
		v := IndexValue{kind: IndexValueOctetString, value: []byte{1, 2, 3}}
		b, ok := v.Bytes()
		testutil.True(t, ok, "Bytes() ok")
		testutil.Equal(t, 3, len(b), "Bytes() length")
	})

	t.Run("OID", func(t *testing.T) {
		v := IndexValue{kind: IndexValueObjectIdentifier, value: OID{1, 3, 6, 1}}
		oid, ok := v.OID()
		testutil.True(t, ok, "OID() ok")
		testutil.Equal(t, 4, len(oid), "OID() length")
	})
}

func TestDecodedIndexString(t *testing.T) {
	d := DecodedIndex{
		name:  "ifIndex",
		value: IndexValue{kind: IndexValueInteger, value: uint32(7)},
	}
	testutil.Equal(t, "ifIndex=7", d.String(), "DecodedIndex.String()")
}
