package module

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

const defvalTestMIB = `
DEFVAL-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Integer32, Counter64, enterprises
        FROM SNMPv2-SMI;

defvalTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

intObj OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-write
    STATUS      current
    DESCRIPTION "Test"
    DEFVAL { 42 }
    ::= { defvalTest 1 }

negIntObj OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-write
    STATUS      current
    DESCRIPTION "Test"
    DEFVAL { -1 }
    ::= { defvalTest 2 }

strObj OBJECT-TYPE
    SYNTAX      OCTET STRING
    MAX-ACCESS  read-write
    STATUS      current
    DESCRIPTION "Test"
    DEFVAL { "public" }
    ::= { defvalTest 3 }

hexObj OBJECT-TYPE
    SYNTAX      OCTET STRING
    MAX-ACCESS  read-write
    STATUS      current
    DESCRIPTION "Test"
    DEFVAL { 'FF00'H }
    ::= { defvalTest 4 }

enumObj OBJECT-TYPE
    SYNTAX      INTEGER { enabled(1), disabled(2) }
    MAX-ACCESS  read-write
    STATUS      current
    DESCRIPTION "Test"
    DEFVAL { enabled }
    ::= { defvalTest 5 }

bitsObj OBJECT-TYPE
    SYNTAX      BITS { flag1(0), flag2(1), flag3(2) }
    MAX-ACCESS  read-write
    STATUS      current
    DESCRIPTION "Test"
    DEFVAL { { flag1, flag3 } }
    ::= { defvalTest 6 }

oidObj OBJECT-TYPE
    SYNTAX      OBJECT IDENTIFIER
    MAX-ACCESS  read-write
    STATUS      current
    DESCRIPTION "Test"
    DEFVAL { { 1 3 6 1 } }
    ::= { defvalTest 7 }

noDefvalObj OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { defvalTest 8 }

END
`

func TestLower_DefVal(t *testing.T) {
	mod := lowerModule(t, defvalTestMIB)

	t.Run("integer", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "intObj")
		testutil.NotNil(t, obj.DefVal, "intObj should have a DEFVAL")
		dv, ok := obj.DefVal.(*DefValInteger)
		testutil.True(t, ok, "expected *DefValInteger, got %T", obj.DefVal)
		testutil.Equal(t, int64(42), dv.Value, "integer value")
	})

	t.Run("negative-integer", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "negIntObj")
		testutil.NotNil(t, obj.DefVal, "negIntObj should have a DEFVAL")
		dv, ok := obj.DefVal.(*DefValInteger)
		testutil.True(t, ok, "expected *DefValInteger, got %T", obj.DefVal)
		testutil.Equal(t, int64(-1), dv.Value, "negative integer value")
	})

	t.Run("string", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "strObj")
		testutil.NotNil(t, obj.DefVal, "strObj should have a DEFVAL")
		dv, ok := obj.DefVal.(*DefValString)
		testutil.True(t, ok, "expected *DefValString, got %T", obj.DefVal)
		testutil.Equal(t, "public", dv.Value, "string value")
	})

	t.Run("hex-string", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "hexObj")
		testutil.NotNil(t, obj.DefVal, "hexObj should have a DEFVAL")
		dv, ok := obj.DefVal.(*DefValHexString)
		testutil.True(t, ok, "expected *DefValHexString, got %T", obj.DefVal)
		testutil.Equal(t, "FF00", dv.Value, "hex string value")
	})

	t.Run("enum", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "enumObj")
		testutil.NotNil(t, obj.DefVal, "enumObj should have a DEFVAL")
		dv, ok := obj.DefVal.(*DefValEnum)
		testutil.True(t, ok, "expected *DefValEnum, got %T", obj.DefVal)
		testutil.Equal(t, "enabled", dv.Name, "enum label")
	})

	t.Run("bits", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "bitsObj")
		testutil.NotNil(t, obj.DefVal, "bitsObj should have a DEFVAL")
		dv, ok := obj.DefVal.(*DefValBits)
		testutil.True(t, ok, "expected *DefValBits, got %T", obj.DefVal)
		testutil.Len(t, dv.Labels, 2, "bits label count")
		testutil.Equal(t, "flag1", dv.Labels[0], "first bits label")
		testutil.Equal(t, "flag3", dv.Labels[1], "second bits label")
	})

	t.Run("oid", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "oidObj")
		testutil.NotNil(t, obj.DefVal, "oidObj should have a DEFVAL")
		dv, ok := obj.DefVal.(*DefValOidValue)
		testutil.True(t, ok, "expected *DefValOidValue, got %T", obj.DefVal)
		testutil.Len(t, dv.Components, 4, "OID component count")

		expected := []uint32{1, 3, 6, 1}
		for i, want := range expected {
			comp, ok := dv.Components[i].(*OidComponentNumber)
			testutil.True(t, ok, "component %d: expected *OidComponentNumber, got %T", i, dv.Components[i])
			testutil.Equal(t, want, comp.Value, "component %d value", i)
		}
	})

	t.Run("absent", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "noDefvalObj")
		testutil.Nil(t, obj.DefVal, "noDefvalObj should not have a DEFVAL")
	})
}
