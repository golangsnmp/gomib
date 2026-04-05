package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestParseDefValOidNamed(t *testing.T) {
	// DEFVAL with named OID components: { sysName 0 }
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX OBJECT IDENTIFIER
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { sysName 0 } }
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")

	oidVal, ok := def.DefVal.(*module.DefValOidValue)
	testutil.True(t, ok, "expected DefValOidValue, got %T", def.DefVal)
	testutil.Equal(t, 2, len(oidVal.Components), "OID component count")
}

func TestParseDefValOidNumeric(t *testing.T) {
	// DEFVAL with all-numeric OID: { 1 3 6 1 }
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX OBJECT IDENTIFIER
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { 1 3 6 1 } }
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")

	oidVal, ok := def.DefVal.(*module.DefValOidValue)
	testutil.True(t, ok, "expected DefValOidValue, got %T", def.DefVal)
	testutil.Equal(t, 4, len(oidVal.Components), "OID component count")
}

func TestParseDefValOidNamedNumber(t *testing.T) {
	// DEFVAL with named-number OID components: { iso 3 6 1 }
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX OBJECT IDENTIFIER
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { iso 3 6 1 } }
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")

	oidVal, ok := def.DefVal.(*module.DefValOidValue)
	testutil.True(t, ok, "expected DefValOidValue, got %T", def.DefVal)
	// First component is a name, next 3 are numbers
	testutil.Equal(t, 4, len(oidVal.Components), "OID component count")

	// Verify first component is a name reference
	_, ok = oidVal.Components[0].(*module.OidComponentName)
	testutil.True(t, ok, "first component should be OidComponentName, got %T", oidVal.Components[0])
}

func TestParseDefValOidEmpty(t *testing.T) {
	// DEFVAL with empty braces treated as empty BITS, not OID
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX BITS { a(0) }
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { } }
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")

	bitsVal, ok := def.DefVal.(*module.DefValBits)
	testutil.True(t, ok, "expected DefValBits for empty braces, got %T", def.DefVal)
	testutil.Len(t, bitsVal.Labels, 0, "empty braces should have no labels")
}
