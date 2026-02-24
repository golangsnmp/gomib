package parser

import (
	"fmt"
	"testing"

	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestParseModuleComplianceRefinements(t *testing.T) {
	tests := []struct {
		name            string
		objectClause    string
		wantSyntax      bool
		wantWriteSyntax bool
		wantMinAccess   bool
		minAccessValue  types.Access
	}{
		{
			name: "syntax only",
			objectClause: `OBJECT testObj
					SYNTAX Integer32 (0..100)
					DESCRIPTION "Refined syntax"`,
			wantSyntax: true,
		},
		{
			name: "write-syntax only",
			objectClause: `OBJECT testObj
					WRITE-SYNTAX Integer32 (1..50)
					DESCRIPTION "Write-syntax refinement"`,
			wantWriteSyntax: true,
		},
		{
			name: "min-access only",
			objectClause: `OBJECT testObj
					MIN-ACCESS read-only
					DESCRIPTION "Min-access refinement"`,
			wantMinAccess:  true,
			minAccessValue: types.AccessReadOnly,
		},
		{
			name: "all refinements",
			objectClause: `OBJECT testObj
					SYNTAX Integer32 (0..100)
					WRITE-SYNTAX Integer32 (1..50)
					MIN-ACCESS read-only
					DESCRIPTION "Full refinement"`,
			wantSyntax:      true,
			wantWriteSyntax: true,
			wantMinAccess:   true,
			minAccessValue:  types.AccessReadOnly,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := fmt.Sprintf(`TEST-MIB DEFINITIONS ::= BEGIN
		testCompliance MODULE-COMPLIANCE
			STATUS current
			DESCRIPTION "Test compliance"
			MODULE
				MANDATORY-GROUPS { testGroup1 }
				%s
			::= { testConformance 1 }
		END`, tt.objectClause)

			module := parseModule(source)

			if len(module.Body) == 0 {
				t.Fatal("expected definitions in module body")
			}
			def, ok := module.Body[0].(*ast.ModuleComplianceDef)
			testutil.True(t, ok, "expected ModuleComplianceDef, got %T", module.Body[0])
			testutil.Equal(t, 1, len(def.Modules), "module clauses count")
			mod := def.Modules[0]
			testutil.Equal(t, 1, len(mod.Compliances), "compliances count")
			obj, ok := mod.Compliances[0].(*ast.ComplianceObject)
			testutil.True(t, ok, "expected ComplianceObject, got %T", mod.Compliances[0])

			if tt.wantSyntax {
				testutil.NotNil(t, obj.Syntax, "SYNTAX should be set")
			} else {
				testutil.Nil(t, obj.Syntax, "SYNTAX should not be set")
			}
			if tt.wantWriteSyntax {
				testutil.NotNil(t, obj.WriteSyntax, "WRITE-SYNTAX should be set")
			} else {
				testutil.Nil(t, obj.WriteSyntax, "WRITE-SYNTAX should not be set")
			}
			if tt.wantMinAccess {
				testutil.NotNil(t, obj.MinAccess, "MIN-ACCESS should be set")
				testutil.Equal(t, tt.minAccessValue, obj.MinAccess.Value, "MIN-ACCESS value")
			} else {
				testutil.Nil(t, obj.MinAccess, "MIN-ACCESS should not be set")
			}
		})
	}
}

func TestParseModuleComplianceGroupAndObject(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
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

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ModuleComplianceDef)
	testutil.True(t, ok, "expected ModuleComplianceDef, got %T", module.Body[0])

	mod := def.Modules[0]
	testutil.Equal(t, 2, len(mod.Compliances), "compliances count (GROUP + OBJECT)")

	_, ok = mod.Compliances[0].(*ast.ComplianceGroup)
	testutil.True(t, ok, "first compliance should be ComplianceGroup, got %T", mod.Compliances[0])

	_, ok = mod.Compliances[1].(*ast.ComplianceObject)
	testutil.True(t, ok, "second compliance should be ComplianceObject, got %T", mod.Compliances[1])
}

func TestParseModuleComplianceNamedModule(t *testing.T) {
	module := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testCompliance MODULE-COMPLIANCE
			STATUS current
			DESCRIPTION "Test compliance"
			MODULE SNMPv2-MIB
				MANDATORY-GROUPS { systemGroup }
			::= { testConformance 1 }
		END`)

	if len(module.Body) == 0 {
		t.Fatal("expected definitions in module body")
	}

	def, ok := module.Body[0].(*ast.ModuleComplianceDef)
	testutil.True(t, ok, "expected ModuleComplianceDef, got %T", module.Body[0])

	testutil.Equal(t, 1, len(def.Modules), "module clauses count")
	mod := def.Modules[0]
	testutil.NotNil(t, mod.ModuleName, "module name should be set")
	testutil.Equal(t, "SNMPv2-MIB", mod.ModuleName.Name, "module name")
	testutil.Len(t, mod.MandatoryGroups, 1, "mandatory groups count")
	testutil.Equal(t, "systemGroup", mod.MandatoryGroups[0].Name, "mandatory group name")
}
