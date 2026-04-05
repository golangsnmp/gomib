package parser

import (
	"fmt"
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestParseModuleComplianceBaseline(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testCompliance MODULE-COMPLIANCE
			STATUS current
			DESCRIPTION "Test compliance"
			MODULE
				MANDATORY-GROUPS { testGroup1 }
			::= { testConformance 1 }
		END`)

	if len(mod.Definitions) == 0 {
		t.Fatal("expected definitions in module")
	}

	def, ok := mod.Definitions[0].(*module.ModuleCompliance)
	testutil.True(t, ok, "expected ModuleCompliance, got %T", mod.Definitions[0])
	testutil.Equal(t, "testCompliance", def.Name, "compliance name")
	testutil.Greater(t, len(def.Modules), 0, "should have at least one MODULE clause")
	if len(def.Modules) > 0 {
		testutil.Greater(t, len(def.Modules[0].MandatoryGroups), 0, "should have mandatory groups")
	}
}

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

			mod := parseModule(source)

			if len(mod.Definitions) == 0 {
				t.Fatal("expected definitions in module")
			}
			def, ok := mod.Definitions[0].(*module.ModuleCompliance)
			testutil.True(t, ok, "expected ModuleCompliance, got %T", mod.Definitions[0])
			testutil.Equal(t, 1, len(def.Modules), "module clauses count")
			cm := def.Modules[0]
			testutil.Equal(t, 1, len(cm.Objects), "objects count")
			obj := cm.Objects[0]

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
				testutil.Equal(t, tt.minAccessValue, *obj.MinAccess, "MIN-ACCESS value")
			} else {
				testutil.Nil(t, obj.MinAccess, "MIN-ACCESS should not be set")
			}
		})
	}
}

func TestParseModuleComplianceGroupAndObject(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
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

	if len(mod.Definitions) == 0 {
		t.Fatal("expected definitions in module")
	}

	def, ok := mod.Definitions[0].(*module.ModuleCompliance)
	testutil.True(t, ok, "expected ModuleCompliance, got %T", mod.Definitions[0])

	cm := def.Modules[0]
	testutil.Equal(t, 1, len(cm.Groups), "groups count")
	testutil.Equal(t, 1, len(cm.Objects), "objects count")

	testutil.Equal(t, "testOptionalGroup", cm.Groups[0].Group, "group name")
	testutil.Equal(t, "testObj", cm.Objects[0].Object, "object name")
}

func TestParseModuleComplianceNamedModule(t *testing.T) {
	mod := parseModule(`TEST-MIB DEFINITIONS ::= BEGIN
		testCompliance MODULE-COMPLIANCE
			STATUS current
			DESCRIPTION "Test compliance"
			MODULE SNMPv2-MIB
				MANDATORY-GROUPS { systemGroup }
			::= { testConformance 1 }
		END`)

	if len(mod.Definitions) == 0 {
		t.Fatal("expected definitions in module")
	}

	def, ok := mod.Definitions[0].(*module.ModuleCompliance)
	testutil.True(t, ok, "expected ModuleCompliance, got %T", mod.Definitions[0])

	testutil.Equal(t, 1, len(def.Modules), "module clauses count")
	cm := def.Modules[0]
	testutil.Equal(t, "SNMPv2-MIB", cm.ModuleName, "module name")
	testutil.Len(t, cm.MandatoryGroups, 1, "mandatory groups count")
	testutil.Equal(t, "systemGroup", cm.MandatoryGroups[0], "mandatory group name")
}
