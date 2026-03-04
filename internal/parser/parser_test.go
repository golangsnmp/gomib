package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestParseEmptyModule(t *testing.T) {
	module := parseModule("TEST-MIB DEFINITIONS ::= BEGIN END")

	testutil.Equal(t, "TEST-MIB", module.Name.Name, "module name")
	testutil.Len(t, module.Body, 0, "body should be empty")
}

func TestParseModuleWithImports(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		IMPORTS
			MODULE-IDENTITY, OBJECT-TYPE FROM SNMPv2-SMI
			DisplayString FROM SNMPv2-TC;
		END`)

	testutil.Equal(t, "TEST-MIB", module.Name.Name, "module name")
	testutil.Len(t, module.Imports, 2, "imports count")
	testutil.Equal(t, "SNMPv2-SMI", module.Imports[0].FromModule.Name, "first import module")
	testutil.Len(t, module.Imports[0].Symbols, 2, "first import symbols count")
	testutil.Equal(t, "SNMPv2-TC", module.Imports[1].FromModule.Name, "second import module")
}

func TestParseValueAssignment(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObject OBJECT IDENTIFIER ::= { iso 3 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ValueAssignmentDef)
	testutil.True(t, ok, "expected ValueAssignmentDef, got %T", module.Body[0])
	testutil.Equal(t, "testObject", def.Name.Name, "definition name")
	testutil.Len(t, def.OidAssignment.Components, 2, "OID components count")
}

func TestParseSimpleObjectType(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testIndex OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test description"
			::= { testEntry 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.Equal(t, "testIndex", def.Name.Name, "definition name")
	testutil.Equal(t, types.AccessReadOnly, def.Access.Value, "access value")
	testutil.NotNil(t, def.Status, "status should be set")
	testutil.Equal(t, types.StatusCurrent, def.Status.Value, "status value")
	testutil.NotNil(t, def.Description, "description should be set")
	testutil.Equal(t, "Test description", def.Description.Value, "description value")
}

func TestParseIntegerEnum(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testStatus OBJECT-TYPE
			SYNTAX INTEGER { up(1), down(2), testing(3) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test status"
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	enumSyntax, ok := def.Syntax.Syntax.(*ast.TypeSyntaxIntegerEnum)
	testutil.True(t, ok, "expected IntegerEnum syntax, got %T", def.Syntax.Syntax)
	testutil.Len(t, enumSyntax.NamedNumbers, 3, "named numbers count")
	testutil.Equal(t, "up", enumSyntax.NamedNumbers[0].Name.Name, "first named number name")
	testutil.Equal(t, int64(1), enumSyntax.NamedNumbers[0].Value, "first named number value")
}

func TestParseModuleIdentity(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testMIB MODULE-IDENTITY
			LAST-UPDATED "200001010000Z"
			ORGANIZATION "Test Org"
			CONTACT-INFO "test@test.com"
			DESCRIPTION "Test MIB"
			::= { enterprises 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ModuleIdentityDef)
	testutil.True(t, ok, "expected ModuleIdentityDef, got %T", module.Body[0])
	testutil.Equal(t, "testMIB", def.Name.Name, "definition name")
	testutil.Equal(t, "200001010000Z", def.LastUpdated.Value, "last-updated value")
	testutil.Equal(t, "Test Org", def.Organization.Value, "organization value")
}

func TestParseTextualConvention(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestString TEXTUAL-CONVENTION
			DISPLAY-HINT "255a"
			STATUS current
			DESCRIPTION "Test string type"
			SYNTAX OCTET STRING (SIZE (0..255))
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.TextualConventionDef)
	testutil.True(t, ok, "expected TextualConventionDef, got %T", module.Body[0])
	testutil.Equal(t, "TestString", def.Name.Name, "definition name")
	testutil.NotNil(t, def.DisplayHint, "display-hint should be set")
	testutil.Equal(t, "255a", def.DisplayHint.Value, "display-hint value")
	constrained, ok := def.Syntax.Syntax.(*ast.TypeSyntaxConstrained)
	testutil.True(t, ok, "expected constrained syntax, got %T", def.Syntax.Syntax)
	sizeConstraint, ok := constrained.Constraint.(*ast.ConstraintSize)
	testutil.True(t, ok, "expected SIZE constraint, got %T", constrained.Constraint)
	testutil.Len(t, sizeConstraint.Ranges, 1, "size constraint ranges count")
}

func TestParseObjectGroup(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testGroup OBJECT-GROUP
			OBJECTS { testObject1, testObject2 }
			STATUS current
			DESCRIPTION "Test group"
			::= { testConformance 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectGroupDef)
	testutil.True(t, ok, "expected ObjectGroupDef, got %T", module.Body[0])
	testutil.Equal(t, "testGroup", def.Name.Name, "definition name")
	testutil.Len(t, def.Objects, 2, "objects count")
}

func TestParseTypeAssignment(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestEntry ::= SEQUENCE {
			testIndex Integer32,
			testName DisplayString
		}
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.TypeAssignmentDef)
	testutil.True(t, ok, "expected TypeAssignmentDef, got %T", module.Body[0])
	testutil.Equal(t, "TestEntry", def.Name.Name, "definition name")
	seq, ok := def.Syntax.(*ast.TypeSyntaxSequence)
	testutil.True(t, ok, "expected SEQUENCE syntax, got %T", def.Syntax)
	testutil.Len(t, seq.Fields, 2, "sequence fields count")
}

func TestParseTaggedType(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		MyCounter ::= [APPLICATION 1] IMPLICIT INTEGER (0..4294967295)
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.TypeAssignmentDef)
	testutil.True(t, ok, "expected TypeAssignmentDef, got %T", module.Body[0])
	testutil.Equal(t, "MyCounter", def.Name.Name, "definition name")
	tagged, ok := def.Syntax.(*ast.TypeSyntaxTagged)
	testutil.True(t, ok, "expected TypeSyntaxTagged, got %T", def.Syntax)
	// The underlying type should be a constrained INTEGER
	_, ok = tagged.Underlying.(*ast.TypeSyntaxConstrained)
	testutil.True(t, ok, "expected constrained underlying type, got %T", tagged.Underlying)
}

func TestParseDefVal(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testDefault OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { 42 }
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	intVal, ok := def.DefVal.Value.(*ast.DefValContentInteger)
	testutil.True(t, ok, "expected integer DEFVAL, got %T", def.DefVal.Value)
	testutil.Equal(t, int64(42), intVal.Value, "DEFVAL value")
}

func TestParseIndex(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testEntry OBJECT-TYPE
			SYNTAX TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "Test entry"
			INDEX { testIndex, IMPLIED testName }
			::= { testTable 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.True(t, def.Index != nil, "INDEX should be set")
	indexClause, ok := def.Index.(*ast.IndexClauseIndex)
	testutil.True(t, ok, "expected IndexClauseIndex, got %T", def.Index)
	testutil.Len(t, indexClause.Items, 2, "index items count")
	testutil.False(t, indexClause.Items[0].Implied, "first index should not be IMPLIED")
	testutil.True(t, indexClause.Items[1].Implied, "second index should be IMPLIED")
}

func TestParseIndexBareOctetString(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testEntry OBJECT-TYPE
			SYNTAX TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "Test entry"
			INDEX { OCTET STRING, testOther }
			::= { testTable 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.True(t, def.Index != nil, "INDEX should be set")
	indexClause, ok := def.Index.(*ast.IndexClauseIndex)
	testutil.True(t, ok, "expected IndexClauseIndex, got %T", def.Index)
	testutil.Len(t, indexClause.Items, 2, "OCTET STRING should be one index item, not two")
	testutil.Equal(t, "OCTET STRING", indexClause.Items[0].Object.Name, "first index should be OCTET STRING")
	testutil.Equal(t, "testOther", indexClause.Items[1].Object.Name, "second index should be testOther")
}

func TestParseErrorRecovery(t *testing.T) {
	// This source has an error in the first definition but should parse the second
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		badObject OBJECT-TYPE
			SYNTAX
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Bad"
			::= { test 1 }
		goodObject OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Good"
			::= { test 2 }
		END`)

	// Should have some diagnostics from the error
	testutil.Greater(t, len(module.Diagnostics), 0, "expected diagnostics from parse error")

	// Should have recovered and parsed the second definition
	testutil.Greater(t, len(module.Body), 0, "expected at least one definition after recovery")

	// Verify the recovered definition is goodObject
	var found bool
	for _, def := range module.Body {
		if objDef, ok := def.(*ast.ObjectTypeDef); ok && objDef.Name.Name == "goodObject" {
			found = true
			break
		}
	}
	testutil.True(t, found, "goodObject should be parsed after error recovery")
}

// === SMIv1-specific constructs ===

func TestParseTrapType(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantName   string
		wantNumber uint32
		wantVars   int
		wantDesc   bool
		wantRef    bool
	}{
		{
			name: "full",
			source: `TEST-MIB DEFINITIONS ::= BEGIN
		testTrap TRAP-TYPE
			ENTERPRISE testEnterprise
			VARIABLES { testObject1, testObject2 }
			DESCRIPTION "Test trap"
			REFERENCE "RFC 1215"
			::= 5
		END`,
			wantName:   "testTrap",
			wantNumber: 5,
			wantVars:   2,
			wantDesc:   true,
			wantRef:    true,
		},
		{
			name: "minimal",
			source: `TEST-MIB DEFINITIONS ::= BEGIN
		minTrap TRAP-TYPE
			ENTERPRISE testEnterprise
			::= 1
		END`,
			wantName:   "minTrap",
			wantNumber: 1,
			wantVars:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module := parseModule(tt.source)
			if len(module.Body) == 0 {
				t.Fatal("expected definitions in module body")
			}
			def, ok := module.Body[0].(*ast.TrapTypeDef)
			testutil.True(t, ok, "expected TrapTypeDef, got %T", module.Body[0])
			testutil.Equal(t, tt.wantName, def.Name.Name, "trap name")
			testutil.Equal(t, "testEnterprise", def.Enterprise.Name, "enterprise")
			testutil.Equal(t, tt.wantNumber, def.TrapNumber, "trap number")
			testutil.Len(t, def.Variables, tt.wantVars, "variables count")
			if tt.wantDesc {
				testutil.NotNil(t, def.Description, "description should be set")
			}
			if tt.wantRef {
				testutil.NotNil(t, def.Reference, "reference should be set")
			}
		})
	}
}

// === Conformance constructs ===

func TestParseNotificationType(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		wantObjects int
	}{
		{
			name: "with objects",
			source: `TEST-MIB DEFINITIONS ::= BEGIN
		testNotification NOTIFICATION-TYPE
			OBJECTS { testObject1, testObject2 }
			STATUS current
			DESCRIPTION "Test notification"
			::= { testNotifications 1 }
		END`,
			wantObjects: 2,
		},
		{
			name: "no objects",
			source: `TEST-MIB DEFINITIONS ::= BEGIN
		testNotification NOTIFICATION-TYPE
			STATUS current
			DESCRIPTION "No objects"
			::= { testNotifications 2 }
		END`,
			wantObjects: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module := parseModule(tt.source)
			if len(module.Body) == 0 {
				t.Fatal("expected definitions in module body")
			}
			def, ok := module.Body[0].(*ast.NotificationTypeDef)
			testutil.True(t, ok, "expected NotificationTypeDef, got %T", module.Body[0])
			testutil.Equal(t, "testNotification", def.Name.Name, "notification name")
			testutil.Len(t, def.Objects, tt.wantObjects, "objects count")
			testutil.NotNil(t, def.Status, "status should be set")
		})
	}
}

func TestParseAgentCapabilities(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testAgent AGENT-CAPABILITIES
			PRODUCT-RELEASE "Test Agent 1.0"
			STATUS current
			DESCRIPTION "Test agent capabilities"
			SUPPORTS IF-MIB
				INCLUDES { ifGeneralGroup }
			::= { testCapabilities 1 }
		END`)

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.AgentCapabilitiesDef)
	testutil.True(t, ok, "expected AgentCapabilitiesDef, got %T", module.Body[0])
	testutil.Equal(t, "testAgent", def.Name.Name, "agent capabilities name")
	testutil.Equal(t, "Test Agent 1.0", def.ProductRelease.Value, "product release")
	testutil.Greater(t, len(def.Supports), 0, "should have at least one SUPPORTS clause")
}

func TestParseNotificationGroup(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testNotifGroup NOTIFICATION-GROUP
			NOTIFICATIONS { testNotif1, testNotif2 }
			STATUS current
			DESCRIPTION "Test notification group"
			::= { testConformance 2 }
		END`)

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.NotificationGroupDef)
	testutil.True(t, ok, "expected NotificationGroupDef, got %T", module.Body[0])
	testutil.Equal(t, "testNotifGroup", def.Name.Name, "notification group name")
	testutil.Len(t, def.Notifications, 2, "notifications count")
}

func TestParseObjectIdentity(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testIdentity OBJECT-IDENTITY
			STATUS current
			DESCRIPTION "Test identity"
			::= { testObjects 1 }
		END`)

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ObjectIdentityDef)
	testutil.True(t, ok, "expected ObjectIdentityDef, got %T", module.Body[0])
	testutil.Equal(t, "testIdentity", def.Name.Name, "identity name")
}

// === Boundary conditions ===

func TestParseTruncatedModuleNoEnd(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObject OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test"
			::= { test 1 }`)

	// Should parse something without crashing, even without END
	testutil.Equal(t, "TEST-MIB", module.Name.Name, "module name should be parsed")
}

func TestParseModuleWithMultipleDefinitions(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testRoot OBJECT IDENTIFIER ::= { iso 3 }
		testScalar OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Scalar"
			::= { testRoot 1 }
		TestType ::= TEXTUAL-CONVENTION
			STATUS current
			DESCRIPTION "A type"
			SYNTAX Integer32
		END`)

	testutil.Equal(t, 3, len(module.Body), "should have 3 definitions")
}

func TestParseSyntaxWithRange(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testRange OBJECT-TYPE
			SYNTAX Integer32 (0..100)
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Ranged"
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	constrained, ok := def.Syntax.Syntax.(*ast.TypeSyntaxConstrained)
	testutil.True(t, ok, "expected constrained syntax, got %T", def.Syntax.Syntax)
	rangeConstraint, ok := constrained.Constraint.(*ast.ConstraintRange)
	testutil.True(t, ok, "expected range constraint, got %T", constrained.Constraint)
	testutil.Len(t, rangeConstraint.Ranges, 1, "should have 1 range")
}

func TestParseAugments(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testAugEntry OBJECT-TYPE
			SYNTAX TestAugEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "Augmented entry"
			AUGMENTS { testEntry }
			::= { testAugTable 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	if def.Augments == nil {
		t.Fatal("AUGMENTS clause not parsed")
	}
	testutil.Equal(t, "testEntry", def.Augments.Target.Name, "augments target")
}

func TestParseBitsSyntax(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testBits OBJECT-TYPE
			SYNTAX BITS { monday(0), tuesday(1), wednesday(2) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Bit field"
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	bits, ok := def.Syntax.Syntax.(*ast.TypeSyntaxBits)
	testutil.True(t, ok, "expected TypeSyntaxBits, got %T", def.Syntax.Syntax)
	testutil.Len(t, bits.NamedBits, 3, "named bits count")
	testutil.Equal(t, "monday", bits.NamedBits[0].Name.Name, "first bit name")
	testutil.Equal(t, int64(0), bits.NamedBits[0].Value, "first bit value")
	testutil.Equal(t, "tuesday", bits.NamedBits[1].Name.Name, "second bit name")
	testutil.Equal(t, int64(1), bits.NamedBits[1].Value, "second bit value")
	testutil.Equal(t, "wednesday", bits.NamedBits[2].Name.Name, "third bit name")
	testutil.Equal(t, int64(2), bits.NamedBits[2].Value, "third bit value")
}

func TestParseDefValString(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testStr OBJECT-TYPE
			SYNTAX OCTET STRING
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { "default value" }
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	strVal, ok := def.DefVal.Value.(*ast.DefValContentString)
	testutil.True(t, ok, "expected string DEFVAL, got %T", def.DefVal.Value)
	testutil.Equal(t, "default value", strVal.Value.Value, "DEFVAL string value")
}

func TestParseDefValHex(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testHex OBJECT-TYPE
			SYNTAX OCTET STRING
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { 'FF00'H }
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	hexVal, ok := def.DefVal.Value.(*ast.DefValContentHexString)
	testutil.True(t, ok, "expected DefValContentHexString, got %T", def.DefVal.Value)
	testutil.Equal(t, "FF00", hexVal.Content, "DEFVAL hex content")
}

func TestParseDefValBits(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testBitsDefault OBJECT-TYPE
			SYNTAX BITS { a(0), b(1), c(2) }
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { a, c } }
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	bitsVal, ok := def.DefVal.Value.(*ast.DefValContentBits)
	testutil.True(t, ok, "expected DefValContentBits, got %T", def.DefVal.Value)
	testutil.Len(t, bitsVal.Labels, 2, "BITS DEFVAL labels count")
	testutil.Equal(t, "a", bitsVal.Labels[0].Name, "first BITS DEFVAL label")
	testutil.Equal(t, "c", bitsVal.Labels[1].Name, "second BITS DEFVAL label")
}

func TestParseSequenceOf(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testTable OBJECT-TYPE
			SYNTAX SEQUENCE OF TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "Test table"
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	_, ok = def.Syntax.Syntax.(*ast.TypeSyntaxSequenceOf)
	testutil.True(t, ok, "expected TypeSyntaxSequenceOf, got %T", def.Syntax.Syntax)
}

func TestParseSMIv1ObjectType(t *testing.T) {
	// SMIv1 uses ACCESS instead of MAX-ACCESS
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testSMIv1 OBJECT-TYPE
			SYNTAX INTEGER
			ACCESS read-only
			STATUS mandatory
			DESCRIPTION "SMIv1 object"
			::= { test 1 }
		END`)

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.Equal(t, "testSMIv1", def.Name.Name, "SMIv1 object name")
	testutil.Equal(t, types.AccessReadOnly, def.Access.Value, "SMIv1 access")
	testutil.Equal(t, types.StatusMandatory, def.Status.Value, "SMIv1 mandatory status")
}

// === Strictness Tests ===

func TestIdentifierUnderscoreDiagnostic(t *testing.T) {
	source := `TEST_MIB DEFINITIONS ::= BEGIN
		test_object OBJECT IDENTIFIER ::= { iso 3 }
		END`
	tests := []struct {
		name   string
		config types.DiagnosticConfig
		want   int
	}{
		{"strict", types.StrictConfig(), 2},
		{"permissive", types.PermissiveConfig(), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module := parseModuleWith(source, tt.config)
			got := countDiagnostics(module.Diagnostics, "identifier-underscore")
			testutil.Equal(t, tt.want, got, "identifier-underscore diagnostic count")
		})
	}
}

func TestIdentifierLengthDiagnostic(t *testing.T) {
	longName := "thisIsAReallyLongIdentifierNameThatExceedsSixtyFourCharactersTotal"
	module := parseModuleWith(longName+` DEFINITIONS ::= BEGIN END`, types.StrictConfig())
	testutil.Equal(t, 1, countDiagnostics(module.Diagnostics, "identifier-length-64"), "identifier-length-64 diagnostic count")
}

func TestIdentifierHyphenEndDiagnostic(t *testing.T) {
	module := parseModuleWith(`TEST-MIB- DEFINITIONS ::= BEGIN END`, types.StrictConfig())
	testutil.Equal(t, 1, countDiagnostics(module.Diagnostics, "identifier-hyphen-end"), "identifier-hyphen-end diagnostic count")
}

func TestReservedKeywordDiagnostic(t *testing.T) {
	module := parseModuleWith(`BOOLEAN DEFINITIONS ::= BEGIN END`, types.StrictConfig())
	testutil.Equal(t, 1, countDiagnostics(module.Diagnostics, "keyword-reserved"), "keyword-reserved diagnostic count")
}

func TestParseUnterminatedStringPreservesContent(t *testing.T) {
	// Test parseQuotedString directly with an unterminated string.
	// The lexer produces a TokQuotedString for unterminated strings
	// (span covers from opening quote to EOF). The parser should
	// strip only the opening quote, not the last content char.
	source := []byte(`"hello world`)
	p := New(source, nil, types.PermissiveConfig())

	qs, err := p.parseQuotedString()
	testutil.Nil(t, err, "parseQuotedString should not return error for TokQuotedString")
	testutil.Equal(t, "hello world", qs.Value, "unterminated string should preserve all content after opening quote")
}

func TestParseVariationDescriptionOnly(t *testing.T) {
	// A VARIATION clause with only DESCRIPTION should parse all fields
	// as a unified Variation struct.
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testAgent AGENT-CAPABILITIES
			PRODUCT-RELEASE "1.0"
			STATUS current
			DESCRIPTION "Test"
			SUPPORTS IF-MIB
				INCLUDES { ifGeneralGroup }
				VARIATION ifLinkUpNotification
					DESCRIPTION "Supported"
			::= { test 1 }
		END`)

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.AgentCapabilitiesDef)
	testutil.True(t, ok, "expected AgentCapabilitiesDef, got %T", module.Body[0])
	if len(def.Supports) == 0 {
		t.Fatal("expected SUPPORTS clause")
	}
	if len(def.Supports[0].Variations) == 0 {
		t.Fatal("expected VARIATION clause")
	}

	v := def.Supports[0].Variations[0]
	testutil.Equal(t, "ifLinkUpNotification", v.Name.Name, "variation name")
	testutil.Nil(t, v.Syntax, "no SYNTAX clause")
	testutil.Nil(t, v.Access, "no ACCESS clause")
	testutil.Equal(t, "Supported", v.Description.Value, "description")
}

func TestParseVariationWithSyntax(t *testing.T) {
	// A VARIATION clause with SYNTAX should store it in the unified struct.
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testAgent AGENT-CAPABILITIES
			PRODUCT-RELEASE "1.0"
			STATUS current
			DESCRIPTION "Test"
			SUPPORTS IF-MIB
				INCLUDES { ifGeneralGroup }
				VARIATION ifIndex
					SYNTAX Integer32 (1..100)
					DESCRIPTION "Restricted range"
			::= { test 1 }
		END`)

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.AgentCapabilitiesDef)
	testutil.True(t, ok, "expected AgentCapabilitiesDef, got %T", module.Body[0])
	if len(def.Supports) == 0 || len(def.Supports[0].Variations) == 0 {
		t.Fatal("expected SUPPORTS with VARIATION")
	}

	v := def.Supports[0].Variations[0]
	testutil.Equal(t, "ifIndex", v.Name.Name, "variation name")
	testutil.NotNil(t, v.Syntax, "SYNTAX clause should be present")
	testutil.Equal(t, "Restricted range", v.Description.Value, "description")
}

func TestRecoverToUppercaseObjectType(t *testing.T) {
	// Tests that recoverToDefinition finds definitions starting with
	// uppercase identifiers followed by macro keywords (e.g. vendor MIBs
	// that use uppercase value references like "FooEntry OBJECT-TYPE").
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		badDef GARBAGE NONSENSE
		FooCount OBJECT-TYPE
			SYNTAX Counter32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Found after recovery"
			::= { test 1 }
		END`)

	// Should recover and parse FooCount despite the uppercase identifier
	var found bool
	for _, def := range module.Body {
		if ot, ok := def.(*ast.ObjectTypeDef); ok && ot.Name.Name == "FooCount" {
			found = true
			break
		}
	}
	testutil.True(t, found, "should recover and parse uppercase OBJECT-TYPE definition")
}

func TestParseTextualConventionWithAssignment(t *testing.T) {
	// Verify the ::= form still works after TC dedup refactor
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestDisplay ::= TEXTUAL-CONVENTION
			DISPLAY-HINT "255a"
			STATUS current
			DESCRIPTION "A display string"
			REFERENCE "RFC 1213"
			SYNTAX OCTET STRING (SIZE (0..255))
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.TextualConventionDef)
	testutil.True(t, ok, "expected TextualConventionDef, got %T", module.Body[0])
	testutil.Equal(t, "TestDisplay", def.Name.Name, "TC name")
	testutil.NotNil(t, def.DisplayHint, "display hint should be set")
	testutil.Equal(t, "255a", def.DisplayHint.Value, "display hint value")
	testutil.NotNil(t, def.Reference, "reference should be set")
	testutil.Equal(t, "RFC 1213", def.Reference.Value, "reference value")
}

// === MACRO definition parsing ===

func TestParseMacroDefinition(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		FOOBAR MACRO ::=
		BEGIN
			TYPE NOTATION ::= empty
			VALUE NOTATION ::= value(VALUE OBJECT IDENTIFIER)
		END
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.MacroDefinitionDef)
	testutil.True(t, ok, "expected MacroDefinitionDef, got %T", module.Body[0])
	testutil.Equal(t, "FOOBAR", def.Name.Name, "macro name")
}

func TestParseMacroFollowedByDefinition(t *testing.T) {
	// The critical risk: if parseMacroDefinition consumes the module's END
	// instead of the macro's END, subsequent definitions would be lost.
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		FOOBAR MACRO ::=
		BEGIN
			TYPE NOTATION ::= empty
			VALUE NOTATION ::= value(VALUE OBJECT IDENTIFIER)
		END
		testObject OBJECT IDENTIFIER ::= { iso 3 }
		END`)

	testutil.Len(t, module.Body, 2, "definitions count")
	macro, ok := module.Body[0].(*ast.MacroDefinitionDef)
	testutil.True(t, ok, "expected MacroDefinitionDef, got %T", module.Body[0])
	testutil.Equal(t, "FOOBAR", macro.Name.Name, "macro name")

	valAssign, ok := module.Body[1].(*ast.ValueAssignmentDef)
	testutil.True(t, ok, "expected ValueAssignmentDef, got %T", module.Body[1])
	testutil.Equal(t, "testObject", valAssign.Name.Name, "definition after macro")
}

func TestParseMultipleMacros(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		ALPHA MACRO ::=
		BEGIN
			body content here
		END
		BETA MACRO ::=
		BEGIN
			more body content
		END
		END`)

	testutil.Len(t, module.Body, 2, "definitions count")
	first, ok := module.Body[0].(*ast.MacroDefinitionDef)
	testutil.True(t, ok, "expected first MacroDefinitionDef, got %T", module.Body[0])
	testutil.Equal(t, "ALPHA", first.Name.Name, "first macro name")
	second, ok := module.Body[1].(*ast.MacroDefinitionDef)
	testutil.True(t, ok, "expected second MacroDefinitionDef, got %T", module.Body[1])
	testutil.Equal(t, "BETA", second.Name.Name, "second macro name")
}

func TestParseMacroMissingEnd(t *testing.T) {
	// MACRO without END - should produce diagnostics, not crash
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		FOOBAR MACRO ::=
		BEGIN
			body content`)

	testutil.Greater(t, len(module.Diagnostics), 0, "expected diagnostics for missing END")
}

// === DEFVAL error recovery (skip paths) ===

func TestParseDefValSkipBraced(t *testing.T) {
	// A quoted string inside inner braces is not recognized as BITS/OID,
	// triggering parseDefValSkipBraced to skip to matching brace.
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { "unexpected" } }
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	_, ok = def.DefVal.Value.(*ast.DefValContentUnparsed)
	testutil.True(t, ok, "expected DefValContentUnparsed, got %T", def.DefVal.Value)
}

func TestParseDefValSkipBracedNested(t *testing.T) {
	// Nested braces inside DEFVAL content exercises brace depth counting.
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { { x } } }
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	_, ok = def.DefVal.Value.(*ast.DefValContentUnparsed)
	testutil.True(t, ok, "expected DefValContentUnparsed, got %T", def.DefVal.Value)
}

func TestParseDefValSkipBracedRecovery(t *testing.T) {
	// After skipping unrecognized DEFVAL content, the parser should
	// continue and parse the next definition.
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testBad OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Bad"
			DEFVAL { { "unexpected" } }
			::= { test 1 }
		testGood OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Good"
			::= { test 2 }
		END`)

	testutil.Equal(t, 2, len(module.Body), "both definitions should be parsed")
	second, ok := module.Body[1].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[1])
	testutil.Equal(t, "testGood", second.Name.Name, "second definition name")
}

func TestParseDefValSkipUnknown(t *testing.T) {
	// A semicolon is not a valid DEFVAL token, triggering parseDefValSkipUnknown.
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { ; }
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	_, ok = def.DefVal.Value.(*ast.DefValContentUnparsed)
	testutil.True(t, ok, "expected DefValContentUnparsed, got %T", def.DefVal.Value)
}

func TestParseDefValSkipUnknownWithBraces(t *testing.T) {
	// Unknown token followed by braced content exercises brace depth
	// tracking in parseDefValSkipUnknown.
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { ; { nested } }
			::= { test 1 }
		END`)

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	_, ok = def.DefVal.Value.(*ast.DefValContentUnparsed)
	testutil.True(t, ok, "expected DefValContentUnparsed, got %T", def.DefVal.Value)
}

func TestParseDefValSkipUnknownRecovery(t *testing.T) {
	// After skipping unknown DEFVAL, the parser should continue
	// and parse the next definition.
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testBad OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Bad"
			DEFVAL { ; { nested } }
			::= { test 1 }
		testGood OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Good"
			::= { test 2 }
		END`)

	testutil.Equal(t, 2, len(module.Body), "both definitions should be parsed")
	second, ok := module.Body[1].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[1])
	testutil.Equal(t, "testGood", second.Name.Name, "second definition name")
}

func TestStripQuotedLiteral(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"hex uppercase H", "'FF'H", "FF"},
		{"hex uppercase H with zeros", "'0000'H", "0000"},
		{"hex uppercase H long", "'7FFFFFFF'H", "7FFFFFFF"},

		{"hex lowercase h", "'FF'h", "FF"},
		{"hex lowercase h with zeros", "'0000'h", "0000"},
		{"hex lowercase h long", "'7FFFFFFF'h", "7FFFFFFF"},

		{"binary uppercase B", "'10110'B", "10110"},
		{"binary uppercase B zeros", "'00000000'B", "00000000"},

		{"binary lowercase b", "'10110'b", "10110"},
		{"binary lowercase b zeros", "'00000000'b", "00000000"},

		{"no suffix", "'FF'", "FF'"},
		{"empty string", "", ""},
		{"no quotes", "FF", "FF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripQuotedLiteral(tt.input)
			if got != tt.want {
				t.Errorf("stripQuotedLiteral(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
