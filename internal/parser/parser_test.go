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
