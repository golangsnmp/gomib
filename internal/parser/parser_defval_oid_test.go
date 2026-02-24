package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestParseDefValOidNamed(t *testing.T) {
	// DEFVAL with named OID components: { sysName 0 }
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX OBJECT IDENTIFIER
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { sysName 0 } }
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")

	oidVal, ok := def.DefVal.Value.(*ast.DefValContentObjectIdentifier)
	testutil.True(t, ok, "expected DefValContentObjectIdentifier, got %T", def.DefVal.Value)
	testutil.Equal(t, 2, len(oidVal.Components), "OID component count")
}

func TestParseDefValOidNumeric(t *testing.T) {
	// DEFVAL with all-numeric OID: { 1 3 6 1 }
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX OBJECT IDENTIFIER
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { 1 3 6 1 } }
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")

	oidVal, ok := def.DefVal.Value.(*ast.DefValContentObjectIdentifier)
	testutil.True(t, ok, "expected DefValContentObjectIdentifier, got %T", def.DefVal.Value)
	testutil.Equal(t, 4, len(oidVal.Components), "OID component count")
}

func TestParseDefValOidNamedNumber(t *testing.T) {
	// DEFVAL with named-number OID components: { iso 3 6 1 }
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX OBJECT IDENTIFIER
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { iso 3 6 1 } }
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")

	oidVal, ok := def.DefVal.Value.(*ast.DefValContentObjectIdentifier)
	testutil.True(t, ok, "expected DefValContentObjectIdentifier, got %T", def.DefVal.Value)
	// First component is a name, next 3 are numbers
	testutil.Equal(t, 4, len(oidVal.Components), "OID component count")

	// Verify first component is a name reference
	_, ok = oidVal.Components[0].(*ast.OidComponentName)
	testutil.True(t, ok, "first component should be OidComponentName, got %T", oidVal.Components[0])
}

func TestParseDefValOidEmpty(t *testing.T) {
	// DEFVAL with empty braces treated as empty BITS, not OID
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX BITS { a(0) }
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { } }
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")

	bitsVal, ok := def.DefVal.Value.(*ast.DefValContentBits)
	testutil.True(t, ok, "expected DefValContentBits for empty braces, got %T", def.DefVal.Value)
	testutil.Len(t, bitsVal.Labels, 0, "empty braces should have no labels")
}
