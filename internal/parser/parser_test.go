package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestParseEmptyModule(t *testing.T) {
	mod := parseModule("TEST-MIB DEFINITIONS ::= BEGIN END")

	testutil.Equal(t, "TEST-MIB", mod.Name, "module name")
	testutil.Len(t, mod.Definitions, 0, "body should be empty")
}

func TestParseModuleWithImports(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		IMPORTS
			MODULE-IDENTITY, OBJECT-TYPE FROM SNMPv2-SMI
			DisplayString FROM SNMPv2-TC;
		END`)

	testutil.Equal(t, "TEST-MIB", mod.Name, "module name")
	// Imports are now flat: one entry per symbol
	testutil.Len(t, mod.Imports, 3, "imports count (flat, one per symbol)")
	testutil.Equal(t, "SNMPv2-SMI", mod.Imports[0].Module, "first import module")
	testutil.Equal(t, "MODULE-IDENTITY", mod.Imports[0].Symbol, "first import symbol")
	testutil.Equal(t, "SNMPv2-SMI", mod.Imports[1].Module, "second import module")
	testutil.Equal(t, "OBJECT-TYPE", mod.Imports[1].Symbol, "second import symbol")
	testutil.Equal(t, "SNMPv2-TC", mod.Imports[2].Module, "third import module")
	testutil.Equal(t, "DisplayString", mod.Imports[2].Symbol, "third import symbol")
}

func TestParseValueAssignment(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObject OBJECT IDENTIFIER ::= { iso 3 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ValueAssignment)
	testutil.True(t, ok, "expected ValueAssignment, got %T", mod.Definitions[0])
	testutil.Equal(t, "testObject", def.Name, "definition name")
	testutil.Len(t, def.Oid.Components, 2, "OID components count")
}

func TestParseSimpleObjectType(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testIndex OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test description"
			::= { testEntry 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.Equal(t, "testIndex", def.Name, "definition name")
	testutil.Equal(t, types.AccessReadOnly, def.Access, "access value")
	testutil.Equal(t, types.StatusCurrent, def.Status, "status value")
	testutil.Equal(t, "Test description", def.Description, "description value")
}

func TestParseIntegerEnum(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testStatus OBJECT-TYPE
			SYNTAX INTEGER { up(1), down(2), testing(3) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test status"
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	enumSyntax, ok := def.Syntax.(*module.TypeSyntaxIntegerEnum)
	testutil.True(t, ok, "expected IntegerEnum syntax, got %T", def.Syntax)
	testutil.Len(t, enumSyntax.NamedNumbers, 3, "named numbers count")
	testutil.Equal(t, "up", enumSyntax.NamedNumbers[0].Name, "first named number name")
	testutil.Equal(t, int64(1), enumSyntax.NamedNumbers[0].Value, "first named number value")
}

func TestParseIntegerEnumMissingComma(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testStatus OBJECT-TYPE
			SYNTAX INTEGER { up(1) down(2), testing(3) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test status"
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	enumSyntax, ok := def.Syntax.(*module.TypeSyntaxIntegerEnum)
	testutil.True(t, ok, "expected IntegerEnum syntax, got %T", def.Syntax)
	testutil.Len(t, enumSyntax.NamedNumbers, 3, "named numbers count")
	testutil.Equal(t, "up", enumSyntax.NamedNumbers[0].Name, "first named number name")
	testutil.Equal(t, "down", enumSyntax.NamedNumbers[1].Name, "second named number name")
	testutil.Equal(t, "testing", enumSyntax.NamedNumbers[2].Name, "third named number name")
	testutil.Equal(t, 1, countDiagnostics(mod.Diagnostics, types.DiagMissingComma), "missing comma diagnostic")
}

func TestParseEnumDuplicateName(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testStatus OBJECT-TYPE
			SYNTAX INTEGER { up(1), down(2), up(3) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test"
			::= { test 1 }
		END`)

	testutil.Equal(t, 1, countDiagnostics(mod.Diagnostics, types.DiagEnumNameRedefinition), "enum-name-redefinition diagnostic")
}

func TestParseEnumDuplicateValue(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testStatus OBJECT-TYPE
			SYNTAX INTEGER { up(1), down(2), testing(1) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test"
			::= { test 1 }
		END`)

	testutil.Equal(t, 1, countDiagnostics(mod.Diagnostics, types.DiagEnumValueRedefinition), "enum-value-redefinition diagnostic")
}

func TestParseEnumZeroSMIv1(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testStatus OBJECT-TYPE
			SYNTAX INTEGER { none(0), up(1) }
			ACCESS read-only
			STATUS mandatory
			DESCRIPTION "Test"
			::= { test 1 }
		END`)

	testutil.Equal(t, 1, countDiagnostics(mod.Diagnostics, types.DiagEnumZero), "enum-zero diagnostic for SMIv1")
}

func TestParseEnumZeroSMIv2(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		IMPORTS MODULE-IDENTITY FROM SNMPv2-SMI;
		testStatus OBJECT-TYPE
			SYNTAX INTEGER { none(0), up(1) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test"
			::= { test 1 }
		END`)

	testutil.Equal(t, 0, countDiagnostics(mod.Diagnostics, types.DiagEnumZero), "enum-zero should not fire for SMIv2")
}

func TestParseModuleIdentity(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testMIB MODULE-IDENTITY
			LAST-UPDATED "200001010000Z"
			ORGANIZATION "Test Org"
			CONTACT-INFO "test@test.com"
			DESCRIPTION "Test MIB"
			::= { enterprises 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ModuleIdentity)
	testutil.True(t, ok, "expected ModuleIdentity, got %T", mod.Definitions[0])
	testutil.Equal(t, "testMIB", def.Name, "definition name")
	testutil.Equal(t, "200001010000Z", def.LastUpdated, "last-updated value")
	testutil.Equal(t, "Test Org", def.Organization, "organization value")
}

func TestParseTextualConvention(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestString TEXTUAL-CONVENTION
			DISPLAY-HINT "255a"
			STATUS current
			DESCRIPTION "Test string type"
			SYNTAX OCTET STRING (SIZE (0..255))
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.TypeDef)
	testutil.True(t, ok, "expected TypeDef, got %T", mod.Definitions[0])
	testutil.Equal(t, "TestString", def.Name, "definition name")
	testutil.True(t, def.IsTextualConvention, "should be a textual convention")
	testutil.Equal(t, "255a", def.DisplayHint, "display-hint value")
	constrained, ok := def.Syntax.(*module.TypeSyntaxConstrained)
	testutil.True(t, ok, "expected constrained syntax, got %T", def.Syntax)
	sizeConstraint, ok := constrained.Constraint.(*module.ConstraintSize)
	testutil.True(t, ok, "expected SIZE constraint, got %T", constrained.Constraint)
	testutil.Len(t, sizeConstraint.Ranges, 1, "size constraint ranges count")
}

func TestParseObjectGroup(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testGroup OBJECT-GROUP
			OBJECTS { testObject1, testObject2 }
			STATUS current
			DESCRIPTION "Test group"
			::= { testConformance 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectGroup)
	testutil.True(t, ok, "expected ObjectGroup, got %T", mod.Definitions[0])
	testutil.Equal(t, "testGroup", def.Name, "definition name")
	testutil.Len(t, def.Objects, 2, "objects count")
}

func TestParseTypeAssignment(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestEntry ::= SEQUENCE {
			testIndex Integer32,
			testName DisplayString
		}
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.TypeDef)
	testutil.True(t, ok, "expected TypeDef, got %T", mod.Definitions[0])
	testutil.Equal(t, "TestEntry", def.Name, "definition name")
	testutil.False(t, def.IsTextualConvention, "should not be a textual convention")
	seq, ok := def.Syntax.(*module.TypeSyntaxSequence)
	testutil.True(t, ok, "expected SEQUENCE syntax, got %T", def.Syntax)
	testutil.Len(t, seq.Fields, 2, "sequence fields count")
}

func TestParseTaggedType(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		MyCounter ::= [APPLICATION 1] IMPLICIT INTEGER (0..4294967295)
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.TypeDef)
	testutil.True(t, ok, "expected TypeDef, got %T", mod.Definitions[0])
	testutil.Equal(t, "MyCounter", def.Name, "definition name")
	// Tagged types are normalized to the underlying type during parsing.
	// The underlying type should be a constrained INTEGER.
	_, ok = def.Syntax.(*module.TypeSyntaxConstrained)
	testutil.True(t, ok, "expected constrained underlying type, got %T", def.Syntax)
}

func TestParseDefVal(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testDefault OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { 42 }
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	intVal, ok := def.DefVal.(*module.DefValInteger)
	testutil.True(t, ok, "expected integer DEFVAL, got %T", def.DefVal)
	testutil.Equal(t, int64(42), intVal.Value, "DEFVAL value")
}

func TestParseIndex(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testEntry OBJECT-TYPE
			SYNTAX TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "Test entry"
			INDEX { testIndex, IMPLIED testName }
			::= { testTable 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.Len(t, def.Index, 2, "index items count")
	testutil.False(t, def.Index[0].Implied, "first index should not be IMPLIED")
	testutil.True(t, def.Index[1].Implied, "second index should be IMPLIED")
}

func TestParseIndexBareOctetString(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testEntry OBJECT-TYPE
			SYNTAX TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "Test entry"
			INDEX { OCTET STRING, testOther }
			::= { testTable 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.Len(t, def.Index, 2, "OCTET STRING should be one index item, not two")
	testutil.Equal(t, "OCTET STRING", def.Index[0].Object, "first index should be OCTET STRING")
	testutil.Equal(t, "testOther", def.Index[1].Object, "second index should be testOther")
}

func TestParseErrorRecovery(t *testing.T) {
	// This source has an error in the first definition but should parse the second
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
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
	testutil.Greater(t, len(mod.Diagnostics), 0, "expected diagnostics from parse error")

	// Should have recovered and parsed the second definition
	testutil.Greater(t, len(mod.Definitions), 0, "expected at least one definition after recovery")

	// Verify the recovered definition is goodObject
	var found bool
	for _, def := range mod.Definitions {
		if objDef, ok := def.(*module.ObjectType); ok && objDef.Name == "goodObject" {
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
			mod := parseModule(tt.source)
			if len(mod.Definitions) == 0 {
				t.Fatal("expected definitions in module")
			}
			def, ok := mod.Definitions[0].(*module.Notification)
			testutil.True(t, ok, "expected Notification, got %T", mod.Definitions[0])
			testutil.Equal(t, tt.wantName, def.Name, "trap name")
			testutil.NotNil(t, def.TrapInfo, "TrapInfo should be set for TRAP-TYPE")
			testutil.Equal(t, "testEnterprise", def.TrapInfo.Enterprise, "enterprise")
			testutil.Equal(t, tt.wantNumber, def.TrapInfo.TrapNumber, "trap number")
			testutil.Len(t, def.Objects, tt.wantVars, "variables count")
			if tt.wantDesc {
				testutil.True(t, def.Description != "", "description should be set")
			}
			if tt.wantRef {
				testutil.True(t, def.Reference != "", "reference should be set")
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
			mod := parseModule(tt.source)
			if len(mod.Definitions) == 0 {
				t.Fatal("expected definitions in module")
			}
			def, ok := mod.Definitions[0].(*module.Notification)
			testutil.True(t, ok, "expected Notification, got %T", mod.Definitions[0])
			testutil.Equal(t, "testNotification", def.Name, "notification name")
			testutil.Len(t, def.Objects, tt.wantObjects, "objects count")
			testutil.Equal(t, types.StatusCurrent, def.Status, "status should be current")
		})
	}
}

func TestParseAgentCapabilities(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testAgent AGENT-CAPABILITIES
			PRODUCT-RELEASE "Test Agent 1.0"
			STATUS current
			DESCRIPTION "Test agent capabilities"
			SUPPORTS IF-MIB
				INCLUDES { ifGeneralGroup }
			::= { testCapabilities 1 }
		END`)

	if len(mod.Definitions) == 0 {
		t.Fatal("expected definitions in module")
	}

	def, ok := mod.Definitions[0].(*module.AgentCapabilities)
	testutil.True(t, ok, "expected AgentCapabilities, got %T", mod.Definitions[0])
	testutil.Equal(t, "testAgent", def.Name, "agent capabilities name")
	testutil.Equal(t, "Test Agent 1.0", def.ProductRelease, "product release")
	testutil.Greater(t, len(def.Supports), 0, "should have at least one SUPPORTS clause")
}

func TestParseNotificationGroup(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testNotifGroup NOTIFICATION-GROUP
			NOTIFICATIONS { testNotif1, testNotif2 }
			STATUS current
			DESCRIPTION "Test notification group"
			::= { testConformance 2 }
		END`)

	if len(mod.Definitions) == 0 {
		t.Fatal("expected definitions in module")
	}

	def, ok := mod.Definitions[0].(*module.NotificationGroup)
	testutil.True(t, ok, "expected NotificationGroup, got %T", mod.Definitions[0])
	testutil.Equal(t, "testNotifGroup", def.Name, "notification group name")
	testutil.Len(t, def.Notifications, 2, "notifications count")
}

func TestParseObjectIdentity(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testIdentity OBJECT-IDENTITY
			STATUS current
			DESCRIPTION "Test identity"
			::= { testObjects 1 }
		END`)

	if len(mod.Definitions) == 0 {
		t.Fatal("expected definitions in module")
	}

	def, ok := mod.Definitions[0].(*module.ObjectIdentity)
	testutil.True(t, ok, "expected ObjectIdentity, got %T", mod.Definitions[0])
	testutil.Equal(t, "testIdentity", def.Name, "identity name")
}

// === Boundary conditions ===

func TestParseTruncatedModuleNoEnd(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObject OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test"
			::= { test 1 }`)

	// Should parse something without crashing, even without END
	testutil.Equal(t, "TEST-MIB", mod.Name, "module name should be parsed")
}

func TestParseModuleWithMultipleDefinitions(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
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

	testutil.Equal(t, 3, len(mod.Definitions), "should have 3 definitions")
}

func TestParseSyntaxWithRange(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testRange OBJECT-TYPE
			SYNTAX Integer32 (0..100)
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Ranged"
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	constrained, ok := def.Syntax.(*module.TypeSyntaxConstrained)
	testutil.True(t, ok, "expected constrained syntax, got %T", def.Syntax)
	rangeConstraint, ok := constrained.Constraint.(*module.ConstraintRange)
	testutil.True(t, ok, "expected range constraint, got %T", constrained.Constraint)
	testutil.Len(t, rangeConstraint.Ranges, 1, "should have 1 range")
}

func TestParseAugments(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testAugEntry OBJECT-TYPE
			SYNTAX TestAugEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "Augmented entry"
			AUGMENTS { testEntry }
			::= { testAugTable 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.True(t, def.Augments != "", "AUGMENTS clause not parsed")
	testutil.Equal(t, "testEntry", def.Augments, "augments target")
}

func TestParseBitsSyntax(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testBits OBJECT-TYPE
			SYNTAX BITS { monday(0), tuesday(1), wednesday(2) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Bit field"
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	bits, ok := def.Syntax.(*module.TypeSyntaxBits)
	testutil.True(t, ok, "expected TypeSyntaxBits, got %T", def.Syntax)
	testutil.Len(t, bits.NamedBits, 3, "named bits count")
	testutil.Equal(t, "monday", bits.NamedBits[0].Name, "first bit name")
	testutil.Equal(t, uint32(0), bits.NamedBits[0].Position, "first bit position")
	testutil.Equal(t, "tuesday", bits.NamedBits[1].Name, "second bit name")
	testutil.Equal(t, uint32(1), bits.NamedBits[1].Position, "second bit position")
	testutil.Equal(t, "wednesday", bits.NamedBits[2].Name, "third bit name")
	testutil.Equal(t, uint32(2), bits.NamedBits[2].Position, "third bit position")
}

func TestParseBitsDuplicateName(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testBits OBJECT-TYPE
			SYNTAX BITS { monday(0), tuesday(1), monday(2) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test"
			::= { test 1 }
		END`)

	testutil.Equal(t, 1, countDiagnostics(mod.Diagnostics, types.DiagBitsNameRedefinition), "bits-name-redefinition diagnostic")
}

func TestParseBitsDuplicatePosition(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testBits OBJECT-TYPE
			SYNTAX BITS { monday(0), tuesday(1), wednesday(0) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test"
			::= { test 1 }
		END`)

	testutil.Equal(t, 1, countDiagnostics(mod.Diagnostics, types.DiagBitsValueRedefinition), "bits-value-redefinition diagnostic")
}

func TestParseBitsNegativePosition(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testBits OBJECT-TYPE
			SYNTAX BITS { bad(-1) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test"
			::= { test 1 }
		END`)

	testutil.Equal(t, 1, countDiagnostics(mod.Diagnostics, types.DiagBitsNumberNegative), "bits-number-negative diagnostic")
	def := mod.Definitions[0].(*module.ObjectType)
	bits := def.Syntax.(*module.TypeSyntaxBits)
	testutil.Equal(t, uint32(0), bits.NamedBits[0].Position, "negative position should be clamped to 0")
}

func TestParseBitsLargePosition(t *testing.T) {
	mod := parseModuleWith(`TEST-MIB DEFINITIONS ::= BEGIN
		testBits OBJECT-TYPE
			SYNTAX BITS { big(128) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test"
			::= { test 1 }
		END`, types.VerboseConfig())

	testutil.Equal(t, 1, countDiagnostics(mod.Diagnostics, types.DiagBitsNumberLarge), "bits-number-large diagnostic")
	def := mod.Definitions[0].(*module.ObjectType)
	bits := def.Syntax.(*module.TypeSyntaxBits)
	testutil.Equal(t, uint32(128), bits.NamedBits[0].Position, "large position should be preserved")
}

func TestParseBitsTooLargePosition(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testBits OBJECT-TYPE
			SYNTAX BITS { huge(524280) }
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "Test"
			::= { test 1 }
		END`)

	testutil.Equal(t, 1, countDiagnostics(mod.Diagnostics, types.DiagBitsNumberTooLarge), "bits-number-too-large diagnostic")
	def := mod.Definitions[0].(*module.ObjectType)
	bits := def.Syntax.(*module.TypeSyntaxBits)
	testutil.Equal(t, uint32(0), bits.NamedBits[0].Position, "too-large position should be clamped to 0")
}

func TestParseDefValString(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testStr OBJECT-TYPE
			SYNTAX OCTET STRING
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { "default value" }
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	strVal, ok := def.DefVal.(*module.DefValString)
	testutil.True(t, ok, "expected string DEFVAL, got %T", def.DefVal)
	testutil.Equal(t, "default value", strVal.Value, "DEFVAL string value")
}

func TestParseDefValHex(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testHex OBJECT-TYPE
			SYNTAX OCTET STRING
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { 'FF00'H }
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	hexVal, ok := def.DefVal.(*module.DefValHexString)
	testutil.True(t, ok, "expected DefValHexString, got %T", def.DefVal)
	testutil.Equal(t, "FF00", hexVal.Value, "DEFVAL hex content")
}

func TestParseDefValBits(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testBitsDefault OBJECT-TYPE
			SYNTAX BITS { a(0), b(1), c(2) }
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { a, c } }
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	bitsVal, ok := def.DefVal.(*module.DefValBits)
	testutil.True(t, ok, "expected DefValBits, got %T", def.DefVal)
	testutil.Len(t, bitsVal.Labels, 2, "BITS DEFVAL labels count")
	testutil.Equal(t, "a", bitsVal.Labels[0], "first BITS DEFVAL label")
	testutil.Equal(t, "c", bitsVal.Labels[1], "second BITS DEFVAL label")
}

func TestParseSequenceOf(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testTable OBJECT-TYPE
			SYNTAX SEQUENCE OF TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "Test table"
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	_, ok = def.Syntax.(*module.TypeSyntaxSequenceOf)
	testutil.True(t, ok, "expected TypeSyntaxSequenceOf, got %T", def.Syntax)
}

func TestParseSMIv1ObjectType(t *testing.T) {
	// SMIv1 uses ACCESS instead of MAX-ACCESS
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testSMIv1 OBJECT-TYPE
			SYNTAX INTEGER
			ACCESS read-only
			STATUS mandatory
			DESCRIPTION "SMIv1 object"
			::= { test 1 }
		END`)

	if len(mod.Definitions) == 0 {
		t.Fatal("expected definitions in module")
	}

	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.Equal(t, "testSMIv1", def.Name, "SMIv1 object name")
	testutil.Equal(t, types.AccessReadOnly, def.Access, "SMIv1 access")
	testutil.Equal(t, types.StatusMandatory, def.Status, "SMIv1 mandatory status")
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
		{"strict", types.VerboseConfig(), 2},
		{"permissive", types.DefaultConfig(), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mod := parseModuleWith(source, tt.config)
			got := countDiagnostics(mod.Diagnostics, "identifier-underscore")
			testutil.Equal(t, tt.want, got, "identifier-underscore diagnostic count")
		})
	}
}

func TestIdentifierLengthDiagnostic(t *testing.T) {
	longName := "thisIsAReallyLongIdentifierNameThatExceedsSixtyFourCharactersTotal"
	mod := parseModuleWith(longName+` DEFINITIONS ::= BEGIN END`, types.VerboseConfig())
	testutil.Equal(t, 1, countDiagnostics(mod.Diagnostics, "identifier-length-64"), "identifier-length-64 diagnostic count")
}

func TestIdentifierHyphenEndDiagnostic(t *testing.T) {
	mod := parseModuleWith(`TEST-MIB- DEFINITIONS ::= BEGIN END`, types.VerboseConfig())
	testutil.Equal(t, 1, countDiagnostics(mod.Diagnostics, "identifier-hyphen-end"), "identifier-hyphen-end diagnostic count")
}

func TestReservedKeywordDiagnostic(t *testing.T) {
	mod := parseModuleWith(`BOOLEAN DEFINITIONS ::= BEGIN END`, types.VerboseConfig())
	testutil.Equal(t, 1, countDiagnostics(mod.Diagnostics, "keyword-reserved"), "keyword-reserved diagnostic count")
}

func TestParseUnterminatedStringPreservesContent(t *testing.T) {
	// Test parseQuotedString directly with an unterminated string.
	// The lexer produces a TokQuotedString for unterminated strings
	// (span covers from opening quote to EOF). The parser should
	// strip only the opening quote, not the last content char.
	source := []byte(`"hello world`)
	p := New(source, nil, types.DefaultConfig())

	qs, _, err := p.parseQuotedString()
	testutil.Nil(t, err, "parseQuotedString should not return error for TokQuotedString")
	testutil.Equal(t, "hello world", qs, "unterminated string should preserve all content after opening quote")
}

func TestParseVariationDescriptionOnly(t *testing.T) {
	// A VARIATION clause with only DESCRIPTION should parse all fields
	// as a unified Variation struct.
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
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

	if len(mod.Definitions) == 0 {
		t.Fatal("expected definitions in module")
	}

	def, ok := mod.Definitions[0].(*module.AgentCapabilities)
	testutil.True(t, ok, "expected AgentCapabilities, got %T", mod.Definitions[0])
	if len(def.Supports) == 0 {
		t.Fatal("expected SUPPORTS clause")
	}
	if len(def.Supports[0].Variations) == 0 {
		t.Fatal("expected VARIATION clause")
	}

	v := def.Supports[0].Variations[0]
	testutil.Equal(t, "ifLinkUpNotification", v.Name, "variation name")
	testutil.Nil(t, v.Syntax, "no SYNTAX clause")
	testutil.Nil(t, v.Access, "no ACCESS clause")
	testutil.Equal(t, "Supported", v.Description, "description")
}

func TestParseVariationWithSyntax(t *testing.T) {
	// A VARIATION clause with SYNTAX should store it in the unified struct.
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
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

	if len(mod.Definitions) == 0 {
		t.Fatal("expected definitions in module")
	}

	def, ok := mod.Definitions[0].(*module.AgentCapabilities)
	testutil.True(t, ok, "expected AgentCapabilities, got %T", mod.Definitions[0])
	if len(def.Supports) == 0 || len(def.Supports[0].Variations) == 0 {
		t.Fatal("expected SUPPORTS with VARIATION")
	}

	v := def.Supports[0].Variations[0]
	testutil.Equal(t, "ifIndex", v.Name, "variation name")
	testutil.NotNil(t, v.Syntax, "SYNTAX clause should be present")
	testutil.Equal(t, "Restricted range", v.Description, "description")
}

func TestRecoverToUppercaseObjectType(t *testing.T) {
	// Tests that recoverToDefinition finds definitions starting with
	// uppercase identifiers followed by macro keywords (e.g. vendor MIBs
	// that use uppercase value references like "FooEntry OBJECT-TYPE").
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
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
	for _, def := range mod.Definitions {
		if ot, ok := def.(*module.ObjectType); ok && ot.Name == "FooCount" {
			found = true
			break
		}
	}
	testutil.True(t, found, "should recover and parse uppercase OBJECT-TYPE definition")
}

func TestParseTextualConventionWithAssignment(t *testing.T) {
	// Verify the ::= form still works after TC dedup refactor
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		TestDisplay ::= TEXTUAL-CONVENTION
			DISPLAY-HINT "255a"
			STATUS current
			DESCRIPTION "A display string"
			REFERENCE "RFC 1213"
			SYNTAX OCTET STRING (SIZE (0..255))
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.TypeDef)
	testutil.True(t, ok, "expected TypeDef, got %T", mod.Definitions[0])
	testutil.Equal(t, "TestDisplay", def.Name, "TC name")
	testutil.True(t, def.IsTextualConvention, "should be a textual convention")
	testutil.Equal(t, "255a", def.DisplayHint, "display hint value")
	testutil.Equal(t, "RFC 1213", def.Reference, "reference value")
}

// === MACRO definition parsing ===

func TestParseMacroDefinition(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		FOOBAR MACRO ::=
		BEGIN
			TYPE NOTATION ::= empty
			VALUE NOTATION ::= value(VALUE OBJECT IDENTIFIER)
		END
		END`)

	// MACRO definitions are now skipped (not semantic definitions),
	// so the module should have no definitions.
	testutil.Len(t, mod.Definitions, 0, "definitions count (macros are skipped)")
}

func TestParseMacroFollowedByDefinition(t *testing.T) {
	// The critical risk: if parseMacroDefinition consumes the module's END
	// instead of the macro's END, subsequent definitions would be lost.
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		FOOBAR MACRO ::=
		BEGIN
			TYPE NOTATION ::= empty
			VALUE NOTATION ::= value(VALUE OBJECT IDENTIFIER)
		END
		testObject OBJECT IDENTIFIER ::= { iso 3 }
		END`)

	// MACRO is skipped, only the value assignment remains.
	testutil.Len(t, mod.Definitions, 1, "definitions count (macro skipped)")

	valAssign, ok := mod.Definitions[0].(*module.ValueAssignment)
	testutil.True(t, ok, "expected ValueAssignment, got %T", mod.Definitions[0])
	testutil.Equal(t, "testObject", valAssign.Name, "definition after macro")
}

func TestParseMultipleMacros(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		ALPHA MACRO ::=
		BEGIN
			body content here
		END
		BETA MACRO ::=
		BEGIN
			more body content
		END
		END`)

	// Both macros are skipped.
	testutil.Len(t, mod.Definitions, 0, "definitions count (macros are skipped)")
}

func TestParseMacroMissingEnd(t *testing.T) {
	// MACRO without END - should produce diagnostics, not crash
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		FOOBAR MACRO ::=
		BEGIN
			body content`)

	testutil.Greater(t, len(mod.Diagnostics), 0, "expected diagnostics for missing END")
}

// === DEFVAL error recovery (skip paths) ===

func TestParseDefValSkipBraced(t *testing.T) {
	// A quoted string inside inner braces is not recognized as BITS/OID,
	// triggering parseDefValSkipBraced to skip to matching brace.
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { "unexpected" } }
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	_, ok = def.DefVal.(*module.DefValUnparsed)
	testutil.True(t, ok, "expected DefValUnparsed, got %T", def.DefVal)
}

func TestParseDefValSkipBracedNested(t *testing.T) {
	// Nested braces inside DEFVAL content exercises brace depth counting.
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { { { x } } }
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	_, ok = def.DefVal.(*module.DefValUnparsed)
	testutil.True(t, ok, "expected DefValUnparsed, got %T", def.DefVal)
}

func TestParseDefValSkipBracedRecovery(t *testing.T) {
	// After skipping unrecognized DEFVAL content, the parser should
	// continue and parse the next definition.
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
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

	testutil.Equal(t, 2, len(mod.Definitions), "both definitions should be parsed")
	second, ok := mod.Definitions[1].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[1])
	testutil.Equal(t, "testGood", second.Name, "second definition name")
}

func TestParseDefValSkipUnknown(t *testing.T) {
	// A semicolon is not a valid DEFVAL token, triggering parseDefValSkipUnknown.
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { ; }
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	_, ok = def.DefVal.(*module.DefValUnparsed)
	testutil.True(t, ok, "expected DefValUnparsed, got %T", def.DefVal)
}

func TestParseDefValSkipUnknownWithBraces(t *testing.T) {
	// Unknown token followed by braced content exercises brace depth
	// tracking in parseDefValSkipUnknown.
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "Test"
			DEFVAL { ; { nested } }
			::= { test 1 }
		END`)

	testutil.Len(t, mod.Definitions, 1, "definitions count")
	def, ok := mod.Definitions[0].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[0])
	testutil.NotNil(t, def.DefVal, "DEFVAL should be set")
	_, ok = def.DefVal.(*module.DefValUnparsed)
	testutil.True(t, ok, "expected DefValUnparsed, got %T", def.DefVal)
}

func TestParseDefValSkipUnknownRecovery(t *testing.T) {
	// After skipping unknown DEFVAL, the parser should continue
	// and parse the next definition.
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
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

	testutil.Equal(t, 2, len(mod.Definitions), "both definitions should be parsed")
	second, ok := mod.Definitions[1].(*module.ObjectType)
	testutil.True(t, ok, "expected ObjectType, got %T", mod.Definitions[1])
	testutil.Equal(t, "testGood", second.Name, "second definition name")
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
