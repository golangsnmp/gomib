package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestParseModuleComplianceObjectSyntax(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testCompliance MODULE-COMPLIANCE
			STATUS current
			DESCRIPTION "Test compliance"
			MODULE
				MANDATORY-GROUPS { testGroup1 }
				OBJECT testObj
					SYNTAX Integer32 (0..100)
					DESCRIPTION "Refined syntax"
			::= { testConformance 1 }
		END`)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ModuleComplianceDef)
	if !ok {
		t.Fatalf("expected ModuleComplianceDef, got %T", module.Body[0])
	}
	testutil.Equal(t, "testCompliance", def.Name.Name, "compliance name")
	testutil.Equal(t, 1, len(def.Modules), "module clauses count")

	mod := def.Modules[0]
	testutil.Equal(t, 1, len(mod.Compliances), "compliances count")

	obj, ok := mod.Compliances[0].(*ast.ComplianceObject)
	if !ok {
		t.Fatalf("expected ComplianceObject, got %T", mod.Compliances[0])
	}
	testutil.Equal(t, "testObj", obj.Object.Name, "object name")
	testutil.NotNil(t, obj.Syntax, "SYNTAX refinement should be set")
	testutil.Nil(t, obj.WriteSyntax, "WRITE-SYNTAX should not be set")
	testutil.Nil(t, obj.MinAccess, "MIN-ACCESS should not be set")
}

func TestParseModuleComplianceObjectWriteSyntax(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testCompliance MODULE-COMPLIANCE
			STATUS current
			DESCRIPTION "Test compliance"
			MODULE
				MANDATORY-GROUPS { testGroup1 }
				OBJECT testObj
					WRITE-SYNTAX Integer32 (1..50)
					DESCRIPTION "Write-syntax refinement"
			::= { testConformance 1 }
		END`)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ModuleComplianceDef)
	if !ok {
		t.Fatalf("expected ModuleComplianceDef, got %T", module.Body[0])
	}

	mod := def.Modules[0]
	testutil.Equal(t, 1, len(mod.Compliances), "compliances count")

	obj, ok := mod.Compliances[0].(*ast.ComplianceObject)
	if !ok {
		t.Fatalf("expected ComplianceObject, got %T", mod.Compliances[0])
	}
	testutil.Nil(t, obj.Syntax, "SYNTAX should not be set")
	testutil.NotNil(t, obj.WriteSyntax, "WRITE-SYNTAX refinement should be set")
}

func TestParseModuleComplianceObjectMinAccess(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testCompliance MODULE-COMPLIANCE
			STATUS current
			DESCRIPTION "Test compliance"
			MODULE
				MANDATORY-GROUPS { testGroup1 }
				OBJECT testObj
					MIN-ACCESS read-only
					DESCRIPTION "Min-access refinement"
			::= { testConformance 1 }
		END`)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ModuleComplianceDef)
	if !ok {
		t.Fatalf("expected ModuleComplianceDef, got %T", module.Body[0])
	}

	mod := def.Modules[0]
	testutil.Equal(t, 1, len(mod.Compliances), "compliances count")

	obj, ok := mod.Compliances[0].(*ast.ComplianceObject)
	if !ok {
		t.Fatalf("expected ComplianceObject, got %T", mod.Compliances[0])
	}
	testutil.NotNil(t, obj.MinAccess, "MIN-ACCESS should be set")
	testutil.Equal(t, types.AccessReadOnly, obj.MinAccess.Value, "MIN-ACCESS value")
}

func TestParseModuleComplianceAllRefinements(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testCompliance MODULE-COMPLIANCE
			STATUS current
			DESCRIPTION "Test compliance"
			MODULE
				MANDATORY-GROUPS { testGroup1 }
				OBJECT testObj
					SYNTAX Integer32 (0..100)
					WRITE-SYNTAX Integer32 (1..50)
					MIN-ACCESS read-only
					DESCRIPTION "Full refinement"
			::= { testConformance 1 }
		END`)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ModuleComplianceDef)
	if !ok {
		t.Fatalf("expected ModuleComplianceDef, got %T", module.Body[0])
	}

	mod := def.Modules[0]
	testutil.Equal(t, 1, len(mod.Compliances), "compliances count")

	obj, ok := mod.Compliances[0].(*ast.ComplianceObject)
	if !ok {
		t.Fatalf("expected ComplianceObject, got %T", mod.Compliances[0])
	}
	testutil.NotNil(t, obj.Syntax, "SYNTAX should be set")
	testutil.NotNil(t, obj.WriteSyntax, "WRITE-SYNTAX should be set")
	testutil.NotNil(t, obj.MinAccess, "MIN-ACCESS should be set")
}

func TestParseModuleComplianceGroupAndObject(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testCompliance MODULE-COMPLIANCE
			STATUS current
			DESCRIPTION "Test compliance"
			MODULE
				MANDATORY-GROUPS { testGroup1 }
				GROUP testOptionalGroup
					DESCRIPTION "Optional group"
				OBJECT testObj
					SYNTAX Integer32 (0..100)
					DESCRIPTION "Refined object"
			::= { testConformance 1 }
		END`)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ModuleComplianceDef)
	if !ok {
		t.Fatalf("expected ModuleComplianceDef, got %T", module.Body[0])
	}

	mod := def.Modules[0]
	testutil.Equal(t, 2, len(mod.Compliances), "compliances count (GROUP + OBJECT)")

	_, ok = mod.Compliances[0].(*ast.ComplianceGroup)
	testutil.True(t, ok, "first compliance should be ComplianceGroup, got %T", mod.Compliances[0])

	_, ok = mod.Compliances[1].(*ast.ComplianceObject)
	testutil.True(t, ok, "second compliance should be ComplianceObject, got %T", mod.Compliances[1])
}

func TestParseModuleComplianceNamedModule(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testCompliance MODULE-COMPLIANCE
			STATUS current
			DESCRIPTION "Test compliance"
			MODULE SNMPv2-MIB
				MANDATORY-GROUPS { systemGroup }
			::= { testConformance 1 }
		END`)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ModuleComplianceDef)
	if !ok {
		t.Fatalf("expected ModuleComplianceDef, got %T", module.Body[0])
	}

	testutil.Equal(t, 1, len(def.Modules), "module clauses count")
	mod := def.Modules[0]
	testutil.NotNil(t, mod.ModuleName, "module name should be set")
	testutil.Equal(t, "SNMPv2-MIB", mod.ModuleName.Name, "module name")
}
