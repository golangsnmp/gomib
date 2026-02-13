package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func TestParseEmptyModule(t *testing.T) {
	source := []byte("TEST-MIB DEFINITIONS ::= BEGIN END")
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Equal(t, "TEST-MIB", module.Name.Name, "module name")
	testutil.Equal(t, ast.DefinitionsKindDefinitions, module.DefinitionsKind, "definitions kind")
	testutil.Len(t, module.Body, 0, "body should be empty")
}

func TestParseModuleWithImports(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		IMPORTS
			MODULE-IDENTITY, OBJECT-TYPE FROM SNMPv2-SMI
			DisplayString FROM SNMPv2-TC;
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Equal(t, "TEST-MIB", module.Name.Name, "module name")
	testutil.Len(t, module.Imports, 2, "imports count")
	testutil.Equal(t, "SNMPv2-SMI", module.Imports[0].FromModule.Name, "first import module")
	testutil.Len(t, module.Imports[0].Symbols, 2, "first import symbols count")
	testutil.Equal(t, "SNMPv2-TC", module.Imports[1].FromModule.Name, "second import module")
}

func TestParseValueAssignment(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testObject OBJECT IDENTIFIER ::= { iso 3 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ValueAssignmentDef)
	testutil.True(t, ok, "expected ValueAssignmentDef, got %T", module.Body[0])
	testutil.Equal(t, "testObject", def.Name.Name, "definition name")
	testutil.Len(t, def.OidAssignment.Components, 2, "OID components count")
}

func TestParseSimpleObjectType(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testIndex OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test description"
			::= { testEntry 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.Equal(t, "testIndex", def.Name.Name, "definition name")
	testutil.Equal(t, ast.AccessValueReadOnly, def.Access.Value, "access value")
	testutil.NotNil(t, def.Status, "status should be set")
	testutil.Equal(t, ast.StatusValueCurrent, def.Status.Value, "status value")
	testutil.NotNil(t, def.Description, "description should be set")
	testutil.Equal(t, "Test description", def.Description.Value, "description value")
}

func TestParseIntegerEnum(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testStatus OBJECT-TYPE
			SYNTAX INTEGER { up(1), down(2), testing(3) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test status"
			::= { test 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

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
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testMIB MODULE-IDENTITY
			LAST-UPDATED "200001010000Z"
			ORGANIZATION "Test Org"
			CONTACT-INFO "test@test.com"
			DESCRIPTION "Test MIB"
			::= { enterprises 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ModuleIdentityDef)
	testutil.True(t, ok, "expected ModuleIdentityDef, got %T", module.Body[0])
	testutil.Equal(t, "testMIB", def.Name.Name, "definition name")
	testutil.Equal(t, "200001010000Z", def.LastUpdated.Value, "last-updated value")
	testutil.Equal(t, "Test Org", def.Organization.Value, "organization value")
}

func TestParseTextualConvention(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		TestString TEXTUAL-CONVENTION
			DISPLAY-HINT "255a"
			STATUS current
			DESCRIPTION "Test string type"
			SYNTAX OCTET STRING (SIZE (0..255))
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

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
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testGroup OBJECT-GROUP
			OBJECTS { testObject1, testObject2 }
			STATUS current
			DESCRIPTION "Test group"
			::= { testConformance 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectGroupDef)
	testutil.True(t, ok, "expected ObjectGroupDef, got %T", module.Body[0])
	testutil.Equal(t, "testGroup", def.Name.Name, "definition name")
	testutil.Len(t, def.Objects, 2, "objects count")
}

func TestParseTypeAssignment(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		TestEntry ::= SEQUENCE {
			testIndex Integer32,
			testName DisplayString
		}
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.TypeAssignmentDef)
	testutil.True(t, ok, "expected TypeAssignmentDef, got %T", module.Body[0])
	testutil.Equal(t, "TestEntry", def.Name.Name, "definition name")
	seq, ok := def.Syntax.(*ast.TypeSyntaxSequence)
	testutil.True(t, ok, "expected SEQUENCE syntax, got %T", def.Syntax)
	testutil.Len(t, seq.Fields, 2, "sequence fields count")
}

func TestParseDefVal(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testDefault OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { 42 }
			::= { test 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	testutil.True(t, ok, "expected ObjectTypeDef, got %T", module.Body[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	intVal, ok := def.DefVal.Value.(*ast.DefValContentInteger)
	testutil.True(t, ok, "expected integer DEFVAL, got %T", def.DefVal.Value)
	testutil.Equal(t, int64(42), intVal.Value, "DEFVAL value")
}

func TestParseIndex(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testEntry OBJECT-TYPE
			SYNTAX TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "Test entry"
			INDEX { testIndex, IMPLIED testName }
			::= { testTable 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

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
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testEntry OBJECT-TYPE
			SYNTAX TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "Test entry"
			INDEX { OCTET STRING, testOther }
			::= { testTable 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

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
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
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
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	// Should have some diagnostics from the error
	testutil.Greater(t, len(module.Diagnostics), 0, "expected diagnostics from parse error")

	// Should have recovered and parsed the second definition
	testutil.Greater(t, len(module.Body), 0, "expected at least one definition after recovery")
}

// === SMIv1-specific constructs ===

func TestParseTrapType(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testTrap TRAP-TYPE
			ENTERPRISE testEnterprise
			VARIABLES { testObject1, testObject2 }
			DESCRIPTION "Test trap"
			REFERENCE "RFC 1215"
			::= 5
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.TrapTypeDef)
	if !ok {
		t.Fatalf("expected TrapTypeDef, got %T", module.Body[0])
	}
	testutil.Equal(t, "testTrap", def.Name.Name, "trap name")
	testutil.Equal(t, "testEnterprise", def.Enterprise.Name, "enterprise")
	testutil.Len(t, def.Variables, 2, "variables count")
	testutil.NotNil(t, def.Description, "description should be set")
	testutil.NotNil(t, def.Reference, "reference should be set")
	testutil.Equal(t, uint32(5), def.TrapNumber, "trap number")
}

func TestParseTrapTypeMinimal(t *testing.T) {
	// TRAP-TYPE with only required clauses (ENTERPRISE and trap number)
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		minTrap TRAP-TYPE
			ENTERPRISE testEnterprise
			::= 1
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.TrapTypeDef)
	if !ok {
		t.Fatalf("expected TrapTypeDef, got %T", module.Body[0])
	}
	testutil.Equal(t, "minTrap", def.Name.Name, "trap name")
	testutil.Equal(t, uint32(1), def.TrapNumber, "trap number")
	testutil.Len(t, def.Variables, 0, "no variables")
}

// === Conformance constructs ===

func TestParseNotificationType(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testNotification NOTIFICATION-TYPE
			OBJECTS { testObject1, testObject2 }
			STATUS current
			DESCRIPTION "Test notification"
			::= { testNotifications 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.NotificationTypeDef)
	if !ok {
		t.Fatalf("expected NotificationTypeDef, got %T", module.Body[0])
	}
	testutil.Equal(t, "testNotification", def.Name.Name, "notification name")
	testutil.Len(t, def.Objects, 2, "objects count")
	testutil.NotNil(t, def.Status, "status should be set")
}

func TestParseNotificationTypeNoObjects(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testNotification NOTIFICATION-TYPE
			STATUS current
			DESCRIPTION "No objects"
			::= { testNotifications 2 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.NotificationTypeDef)
	if !ok {
		t.Fatalf("expected NotificationTypeDef, got %T", module.Body[0])
	}
	testutil.Equal(t, "testNotification", def.Name.Name, "notification name")
	testutil.Len(t, def.Objects, 0, "no objects")
}

func TestParseModuleCompliance(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testCompliance MODULE-COMPLIANCE
			STATUS current
			DESCRIPTION "Test compliance"
			MODULE
				MANDATORY-GROUPS { testGroup1 }
			::= { testConformance 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ModuleComplianceDef)
	if !ok {
		t.Fatalf("expected ModuleComplianceDef, got %T", module.Body[0])
	}
	testutil.Equal(t, "testCompliance", def.Name.Name, "compliance name")
	testutil.Greater(t, len(def.Modules), 0, "should have at least one MODULE clause")
	if len(def.Modules) > 0 {
		testutil.Greater(t, len(def.Modules[0].MandatoryGroups), 0, "should have mandatory groups")
	}
}

func TestParseAgentCapabilities(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testAgent AGENT-CAPABILITIES
			PRODUCT-RELEASE "Test Agent 1.0"
			STATUS current
			DESCRIPTION "Test agent capabilities"
			SUPPORTS IF-MIB
				INCLUDES { ifGeneralGroup }
			::= { testCapabilities 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.AgentCapabilitiesDef)
	if !ok {
		t.Fatalf("expected AgentCapabilitiesDef, got %T", module.Body[0])
	}
	testutil.Equal(t, "testAgent", def.Name.Name, "agent capabilities name")
	testutil.Equal(t, "Test Agent 1.0", def.ProductRelease.Value, "product release")
	testutil.Greater(t, len(def.Supports), 0, "should have at least one SUPPORTS clause")
}

func TestParseNotificationGroup(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testNotifGroup NOTIFICATION-GROUP
			NOTIFICATIONS { testNotif1, testNotif2 }
			STATUS current
			DESCRIPTION "Test notification group"
			::= { testConformance 2 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.NotificationGroupDef)
	if !ok {
		t.Fatalf("expected NotificationGroupDef, got %T", module.Body[0])
	}
	testutil.Equal(t, "testNotifGroup", def.Name.Name, "notification group name")
	testutil.Len(t, def.Notifications, 2, "notifications count")
}

func TestParseObjectIdentity(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testIdentity OBJECT-IDENTITY
			STATUS current
			DESCRIPTION "Test identity"
			::= { testObjects 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ObjectIdentityDef)
	if !ok {
		t.Fatalf("expected ObjectIdentityDef, got %T", module.Body[0])
	}
	testutil.Equal(t, "testIdentity", def.Name.Name, "identity name")
}

// === Boundary conditions ===

func TestParseTruncatedModuleNoEnd(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testObject OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test"
			::= { test 1 }`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	// Should parse something without crashing, even without END
	testutil.Equal(t, "TEST-MIB", module.Name.Name, "module name should be parsed")
}

func TestParseModuleWithMultipleDefinitions(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
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
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Equal(t, 3, len(module.Body), "should have 3 definitions")
}

func TestParseSyntaxWithRange(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testRange OBJECT-TYPE
			SYNTAX Integer32 (0..100)
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Ranged"
			::= { test 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	if !ok {
		t.Fatalf("expected ObjectTypeDef, got %T", module.Body[0])
	}
	constrained, ok := def.Syntax.Syntax.(*ast.TypeSyntaxConstrained)
	if !ok {
		t.Fatalf("expected constrained syntax, got %T", def.Syntax.Syntax)
	}
	rangeConstraint, ok := constrained.Constraint.(*ast.ConstraintRange)
	if !ok {
		t.Fatalf("expected range constraint, got %T", constrained.Constraint)
	}
	testutil.Len(t, rangeConstraint.Ranges, 1, "should have 1 range")
}

func TestParseAugments(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testAugEntry OBJECT-TYPE
			SYNTAX TestAugEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "Augmented entry"
			AUGMENTS { testEntry }
			::= { testAugTable 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	if !ok {
		t.Fatalf("expected ObjectTypeDef, got %T", module.Body[0])
	}
	if def.Augments == nil {
		t.Fatal("AUGMENTS clause not parsed")
	}
	testutil.Equal(t, "testEntry", def.Augments.Target.Name, "augments target")
}

func TestParseBitsSyntax(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testBits OBJECT-TYPE
			SYNTAX BITS { monday(0), tuesday(1), wednesday(2) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Bit field"
			::= { test 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	if !ok {
		t.Fatalf("expected ObjectTypeDef, got %T", module.Body[0])
	}
	bits, ok := def.Syntax.Syntax.(*ast.TypeSyntaxBits)
	if !ok {
		t.Fatalf("expected TypeSyntaxBits, got %T", def.Syntax.Syntax)
	}
	testutil.Len(t, bits.NamedBits, 3, "named bits count")
}

func TestParseDefValString(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testStr OBJECT-TYPE
			SYNTAX OCTET STRING
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { "default value" }
			::= { test 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	if !ok {
		t.Fatalf("expected ObjectTypeDef, got %T", module.Body[0])
	}
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	strVal, ok := def.DefVal.Value.(*ast.DefValContentString)
	if !ok {
		t.Fatalf("expected string DEFVAL, got %T", def.DefVal.Value)
	}
	testutil.Equal(t, "default value", strVal.Value.Value, "DEFVAL string value")
}

func TestParseDefValHex(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testHex OBJECT-TYPE
			SYNTAX OCTET STRING
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { 'FF00'H }
			::= { test 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	if !ok {
		t.Fatalf("expected ObjectTypeDef, got %T", module.Body[0])
	}
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
}

func TestParseDefValBits(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testBitsDefault OBJECT-TYPE
			SYNTAX BITS { a(0), b(1), c(2) }
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { a, c } }
			::= { test 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	if !ok {
		t.Fatalf("expected ObjectTypeDef, got %T", module.Body[0])
	}
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
}

func TestParseSequenceOf(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testTable OBJECT-TYPE
			SYNTAX SEQUENCE OF TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "Test table"
			::= { test 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	if !ok {
		t.Fatalf("expected ObjectTypeDef, got %T", module.Body[0])
	}
	_, ok = def.Syntax.Syntax.(*ast.TypeSyntaxSequenceOf)
	if !ok {
		t.Fatalf("expected TypeSyntaxSequenceOf, got %T", def.Syntax.Syntax)
	}
}

func TestParseSMIv1ObjectType(t *testing.T) {
	// SMIv1 uses ACCESS instead of MAX-ACCESS
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testSMIv1 OBJECT-TYPE
			SYNTAX INTEGER
			ACCESS read-only
			STATUS mandatory
			DESCRIPTION "SMIv1 object"
			::= { test 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	if !ok {
		t.Fatalf("expected ObjectTypeDef, got %T", module.Body[0])
	}
	testutil.Equal(t, "testSMIv1", def.Name.Name, "SMIv1 object name")
	testutil.Equal(t, ast.AccessValueReadOnly, def.Access.Value, "SMIv1 access")
	testutil.Equal(t, ast.StatusValueMandatory, def.Status.Value, "SMIv1 mandatory status")
}

// === Strictness Tests ===

func TestIdentifierUnderscoreDiagnostic(t *testing.T) {
	source := []byte(`TEST_MIB DEFINITIONS ::= BEGIN
		test_object OBJECT IDENTIFIER ::= { iso 3 }
		END`)
	p := New(source, nil, mib.StrictConfig())
	module := p.ParseModule()

	// Should have diagnostics for underscores in both module name and object name
	var underscoreDiags int
	for _, d := range module.Diagnostics {
		if d.Code == "identifier-underscore" {
			underscoreDiags++
		}
	}
	testutil.Equal(t, 2, underscoreDiags, "expected 2 identifier-underscore diagnostics")
}

func TestIdentifierUnderscorePermissive(t *testing.T) {
	source := []byte(`TEST_MIB DEFINITIONS ::= BEGIN
		test_object OBJECT IDENTIFIER ::= { iso 3 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	// In permissive mode, underscore diagnostics should be suppressed
	var underscoreDiags int
	for _, d := range module.Diagnostics {
		if d.Code == "identifier-underscore" {
			underscoreDiags++
		}
	}
	testutil.Equal(t, 0, underscoreDiags, "expected no identifier-underscore diagnostics in permissive mode")
}

func TestIdentifierLengthDiagnostic(t *testing.T) {
	// Create an identifier that exceeds 64 characters
	longName := "thisIsAReallyLongIdentifierNameThatExceedsSixtyFourCharactersTotal"
	source := []byte(longName + ` DEFINITIONS ::= BEGIN END`)
	p := New(source, nil, mib.StrictConfig())
	module := p.ParseModule()

	// Should have diagnostic for identifier exceeding 64 chars
	var lengthDiags int
	for _, d := range module.Diagnostics {
		if d.Code == "identifier-length-64" {
			lengthDiags++
		}
	}
	testutil.Equal(t, 1, lengthDiags, "expected identifier-length-64 diagnostic")
}

func TestIdentifierHyphenEndDiagnostic(t *testing.T) {
	source := []byte(`TEST-MIB- DEFINITIONS ::= BEGIN END`)
	p := New(source, nil, mib.StrictConfig())
	module := p.ParseModule()

	// Should have diagnostic for identifier ending with hyphen
	var hyphenDiags int
	for _, d := range module.Diagnostics {
		if d.Code == "identifier-hyphen-end" {
			hyphenDiags++
		}
	}
	testutil.Equal(t, 1, hyphenDiags, "expected identifier-hyphen-end diagnostic")
}

func TestReservedKeywordDiagnostic(t *testing.T) {
	// "BOOLEAN" is a reserved ASN.1 keyword
	source := []byte(`BOOLEAN DEFINITIONS ::= BEGIN END`)
	p := New(source, nil, mib.StrictConfig())
	module := p.ParseModule()

	// Should have diagnostic for reserved keyword
	var keywordDiags int
	for _, d := range module.Diagnostics {
		if d.Code == "keyword-reserved" {
			keywordDiags++
		}
	}
	testutil.Equal(t, 1, keywordDiags, "expected keyword-reserved diagnostic")
}

func TestParseUnterminatedStringPreservesContent(t *testing.T) {
	// Test parseQuotedString directly with an unterminated string.
	// The lexer produces a TokQuotedString for unterminated strings
	// (span covers from opening quote to EOF). The parser should
	// strip only the opening quote, not the last content char.
	source := []byte(`"hello world`)
	p := New(source, nil, mib.PermissiveConfig())

	qs, err := p.parseQuotedString()
	testutil.Nil(t, err, "parseQuotedString should not return error for TokQuotedString")
	testutil.Equal(t, "hello world", qs.Value, "unterminated string should preserve all content after opening quote")
}

func TestParseVariationNotification(t *testing.T) {
	// A VARIATION clause with only ACCESS and DESCRIPTION (no SYNTAX,
	// WRITE-SYNTAX, CREATION-REQUIRES, or DEFVAL) should produce a
	// NotificationVariation, not an ObjectVariation.
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
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
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.AgentCapabilitiesDef)
	if !ok {
		t.Fatalf("expected AgentCapabilitiesDef, got %T", module.Body[0])
	}
	if len(def.Supports) == 0 {
		t.Fatal("expected SUPPORTS clause")
	}
	if len(def.Supports[0].Variations) == 0 {
		t.Fatal("expected VARIATION clause")
	}

	_, ok = def.Supports[0].Variations[0].(*ast.NotificationVariation)
	testutil.True(t, ok, "variation with only DESCRIPTION should be NotificationVariation, got %T",
		def.Supports[0].Variations[0])
}

func TestParseVariationObject(t *testing.T) {
	// A VARIATION clause with SYNTAX should produce an ObjectVariation.
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
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
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.AgentCapabilitiesDef)
	if !ok {
		t.Fatalf("expected AgentCapabilitiesDef, got %T", module.Body[0])
	}
	if len(def.Supports) == 0 || len(def.Supports[0].Variations) == 0 {
		t.Fatal("expected SUPPORTS with VARIATION")
	}

	_, ok = def.Supports[0].Variations[0].(*ast.ObjectVariation)
	testutil.True(t, ok, "variation with SYNTAX should be ObjectVariation, got %T",
		def.Supports[0].Variations[0])
}

func TestRecoverToUppercaseObjectType(t *testing.T) {
	// Tests that recoverToDefinition finds definitions starting with
	// uppercase identifiers followed by macro keywords (e.g. vendor MIBs
	// that use uppercase value references like "FooEntry OBJECT-TYPE").
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		badDef GARBAGE NONSENSE
		FooCount OBJECT-TYPE
			SYNTAX Counter32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Found after recovery"
			::= { test 1 }
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

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
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		TestDisplay ::= TEXTUAL-CONVENTION
			DISPLAY-HINT "255a"
			STATUS current
			DESCRIPTION "A display string"
			REFERENCE "RFC 1213"
			SYNTAX OCTET STRING (SIZE (0..255))
		END`)
	p := New(source, nil, mib.PermissiveConfig())
	module := p.ParseModule()

	testutil.Len(t, module.Body, 1, "definitions count")
	def, ok := module.Body[0].(*ast.TextualConventionDef)
	if !ok {
		t.Fatalf("expected TextualConventionDef, got %T", module.Body[0])
	}
	testutil.Equal(t, "TestDisplay", def.Name.Name, "TC name")
	testutil.NotNil(t, def.DisplayHint, "display hint should be set")
	testutil.Equal(t, "255a", def.DisplayHint.Value, "display hint value")
	testutil.NotNil(t, def.Reference, "reference should be set")
	testutil.Equal(t, "RFC 1213", def.Reference.Value, "reference value")
}
