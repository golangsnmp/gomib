package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/ast"
)

func TestParseEmptyModule(t *testing.T) {
	source := []byte("TEST-MIB DEFINITIONS ::= BEGIN END")
	p := New(source, nil)
	module := p.ParseModule()

	if module.Name.Name != "TEST-MIB" {
		t.Errorf("expected module name TEST-MIB, got %s", module.Name.Name)
	}
	if module.DefinitionsKind != ast.DefinitionsKindDefinitions {
		t.Errorf("expected DefinitionsKindDefinitions")
	}
	if len(module.Body) != 0 {
		t.Errorf("expected empty body, got %d definitions", len(module.Body))
	}
}

func TestParseModuleWithImports(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		IMPORTS
			MODULE-IDENTITY, OBJECT-TYPE FROM SNMPv2-SMI
			DisplayString FROM SNMPv2-TC;
		END`)
	p := New(source, nil)
	module := p.ParseModule()

	if module.Name.Name != "TEST-MIB" {
		t.Errorf("expected module name TEST-MIB, got %s", module.Name.Name)
	}
	if len(module.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(module.Imports))
	}
	if module.Imports[0].FromModule.Name != "SNMPv2-SMI" {
		t.Errorf("expected first import from SNMPv2-SMI, got %s", module.Imports[0].FromModule.Name)
	}
	if len(module.Imports[0].Symbols) != 2 {
		t.Errorf("expected 2 symbols in first import, got %d", len(module.Imports[0].Symbols))
	}
	if module.Imports[1].FromModule.Name != "SNMPv2-TC" {
		t.Errorf("expected second import from SNMPv2-TC, got %s", module.Imports[1].FromModule.Name)
	}
}

func TestParseValueAssignment(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testObject OBJECT IDENTIFIER ::= { iso 3 }
		END`)
	p := New(source, nil)
	module := p.ParseModule()

	if len(module.Body) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(module.Body))
	}
	def, ok := module.Body[0].(*ast.ValueAssignmentDef)
	if !ok {
		t.Fatalf("expected ValueAssignmentDef, got %T", module.Body[0])
	}
	if def.Name.Name != "testObject" {
		t.Errorf("expected name testObject, got %s", def.Name.Name)
	}
	if len(def.OidAssignment.Components) != 2 {
		t.Errorf("expected 2 OID components, got %d", len(def.OidAssignment.Components))
	}
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
	p := New(source, nil)
	module := p.ParseModule()

	if len(module.Body) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(module.Body))
	}
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	if !ok {
		t.Fatalf("expected ObjectTypeDef, got %T", module.Body[0])
	}
	if def.Name.Name != "testIndex" {
		t.Errorf("expected name testIndex, got %s", def.Name.Name)
	}
	if def.Access.Value != ast.AccessValueReadOnly {
		t.Errorf("expected read-only access, got %v", def.Access.Value)
	}
	if def.Status == nil {
		t.Error("expected status to be set")
	} else if def.Status.Value != ast.StatusValueCurrent {
		t.Errorf("expected current status, got %v", def.Status.Value)
	}
	if def.Description == nil {
		t.Error("expected description to be set")
	} else if def.Description.Value != "Test description" {
		t.Errorf("expected description 'Test description', got '%s'", def.Description.Value)
	}
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
	p := New(source, nil)
	module := p.ParseModule()

	if len(module.Body) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(module.Body))
	}
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	if !ok {
		t.Fatalf("expected ObjectTypeDef, got %T", module.Body[0])
	}
	enumSyntax, ok := def.Syntax.Syntax.(*ast.TypeSyntaxIntegerEnum)
	if !ok {
		t.Fatalf("expected IntegerEnum syntax, got %T", def.Syntax.Syntax)
	}
	if len(enumSyntax.NamedNumbers) != 3 {
		t.Errorf("expected 3 named numbers, got %d", len(enumSyntax.NamedNumbers))
	}
	if enumSyntax.NamedNumbers[0].Name.Name != "up" || enumSyntax.NamedNumbers[0].Value != 1 {
		t.Errorf("expected up(1), got %s(%d)", enumSyntax.NamedNumbers[0].Name.Name, enumSyntax.NamedNumbers[0].Value)
	}
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
	p := New(source, nil)
	module := p.ParseModule()

	if len(module.Body) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(module.Body))
	}
	def, ok := module.Body[0].(*ast.ModuleIdentityDef)
	if !ok {
		t.Fatalf("expected ModuleIdentityDef, got %T", module.Body[0])
	}
	if def.Name.Name != "testMIB" {
		t.Errorf("expected name testMIB, got %s", def.Name.Name)
	}
	if def.LastUpdated.Value != "200001010000Z" {
		t.Errorf("expected last-updated '200001010000Z', got '%s'", def.LastUpdated.Value)
	}
	if def.Organization.Value != "Test Org" {
		t.Errorf("expected organization 'Test Org', got '%s'", def.Organization.Value)
	}
}

func TestParseTextualConvention(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		TestString TEXTUAL-CONVENTION
			DISPLAY-HINT "255a"
			STATUS current
			DESCRIPTION "Test string type"
			SYNTAX OCTET STRING (SIZE (0..255))
		END`)
	p := New(source, nil)
	module := p.ParseModule()

	if len(module.Body) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(module.Body))
	}
	def, ok := module.Body[0].(*ast.TextualConventionDef)
	if !ok {
		t.Fatalf("expected TextualConventionDef, got %T", module.Body[0])
	}
	if def.Name.Name != "TestString" {
		t.Errorf("expected name TestString, got %s", def.Name.Name)
	}
	if def.DisplayHint == nil || def.DisplayHint.Value != "255a" {
		t.Errorf("expected display-hint '255a'")
	}
	constrained, ok := def.Syntax.Syntax.(*ast.TypeSyntaxConstrained)
	if !ok {
		t.Fatalf("expected constrained syntax, got %T", def.Syntax.Syntax)
	}
	sizeConstraint, ok := constrained.Constraint.(*ast.ConstraintSize)
	if !ok {
		t.Fatalf("expected SIZE constraint, got %T", constrained.Constraint)
	}
	if len(sizeConstraint.Ranges) != 1 {
		t.Errorf("expected 1 range, got %d", len(sizeConstraint.Ranges))
	}
}

func TestParseObjectGroup(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testGroup OBJECT-GROUP
			OBJECTS { testObject1, testObject2 }
			STATUS current
			DESCRIPTION "Test group"
			::= { testConformance 1 }
		END`)
	p := New(source, nil)
	module := p.ParseModule()

	if len(module.Body) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(module.Body))
	}
	def, ok := module.Body[0].(*ast.ObjectGroupDef)
	if !ok {
		t.Fatalf("expected ObjectGroupDef, got %T", module.Body[0])
	}
	if def.Name.Name != "testGroup" {
		t.Errorf("expected name testGroup, got %s", def.Name.Name)
	}
	if len(def.Objects) != 2 {
		t.Errorf("expected 2 objects, got %d", len(def.Objects))
	}
}

func TestParseTypeAssignment(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		TestEntry ::= SEQUENCE {
			testIndex Integer32,
			testName DisplayString
		}
		END`)
	p := New(source, nil)
	module := p.ParseModule()

	if len(module.Body) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(module.Body))
	}
	def, ok := module.Body[0].(*ast.TypeAssignmentDef)
	if !ok {
		t.Fatalf("expected TypeAssignmentDef, got %T", module.Body[0])
	}
	if def.Name.Name != "TestEntry" {
		t.Errorf("expected name TestEntry, got %s", def.Name.Name)
	}
	seq, ok := def.Syntax.(*ast.TypeSyntaxSequence)
	if !ok {
		t.Fatalf("expected SEQUENCE syntax, got %T", def.Syntax)
	}
	if len(seq.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(seq.Fields))
	}
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
	p := New(source, nil)
	module := p.ParseModule()

	if len(module.Body) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(module.Body))
	}
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	if !ok {
		t.Fatalf("expected ObjectTypeDef, got %T", module.Body[0])
	}
	if def.DefVal == nil {
		t.Fatal("expected DEFVAL to be set")
	}
	intVal, ok := def.DefVal.Value.(*ast.DefValContentInteger)
	if !ok {
		t.Fatalf("expected integer DEFVAL, got %T", def.DefVal.Value)
	}
	if intVal.Value != 42 {
		t.Errorf("expected DEFVAL 42, got %d", intVal.Value)
	}
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
	p := New(source, nil)
	module := p.ParseModule()

	if len(module.Body) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(module.Body))
	}
	def, ok := module.Body[0].(*ast.ObjectTypeDef)
	if !ok {
		t.Fatalf("expected ObjectTypeDef, got %T", module.Body[0])
	}
	if def.Index == nil {
		t.Fatal("expected INDEX to be set")
	}
	indexClause, ok := def.Index.(*ast.IndexClauseIndex)
	if !ok {
		t.Fatalf("expected IndexClauseIndex, got %T", def.Index)
	}
	if len(indexClause.Items) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(indexClause.Items))
	}
	if indexClause.Items[0].Implied {
		t.Error("first index should not be IMPLIED")
	}
	if !indexClause.Items[1].Implied {
		t.Error("second index should be IMPLIED")
	}
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
	p := New(source, nil)
	module := p.ParseModule()

	// Should have some diagnostics from the error
	if len(module.Diagnostics) == 0 {
		t.Error("expected diagnostics from parse error")
	}

	// Should have recovered and parsed the second definition
	if len(module.Body) < 1 {
		t.Error("expected at least one definition after recovery")
	}
}
