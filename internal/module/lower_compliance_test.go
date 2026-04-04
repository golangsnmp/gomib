package module

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestLower_ModuleCompliance(t *testing.T) {
	source := `COMPLIANCE-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises
        FROM SNMPv2-SMI
    MODULE-COMPLIANCE, OBJECT-GROUP
        FROM SNMPv2-CONF;

complianceTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testCompliance MODULE-COMPLIANCE
    STATUS current
    DESCRIPTION "Test compliance statement"
    REFERENCE "RFC 9999"
    MODULE
        MANDATORY-GROUPS { testGroup1, testGroup2 }
        GROUP testOptionalGroup
            DESCRIPTION "Conditionally required"
        OBJECT testObj1
            SYNTAX Integer32 (0..100)
            DESCRIPTION "Refined range"
        OBJECT testObj2
            MIN-ACCESS read-only
            DESCRIPTION "Reduced access"
    ::= { complianceTest 1 }

END
`

	mod := lowerModule(t, source)
	mc := requireDef[*ModuleCompliance](t, mod, "testCompliance")
	testutil.NotNil(t, mc, "expected testCompliance definition")

	// Top-level fields
	testutil.Equal(t, "testCompliance", mc.Name, "Name")
	testutil.Equal(t, types.StatusCurrent, mc.Status, "Status")
	testutil.Equal(t, "Test compliance statement", mc.Description, "Description")
	testutil.Equal(t, "RFC 9999", mc.Reference, "Reference")

	// Modules
	testutil.Len(t, mc.Modules, 1, "Modules")
	cm := mc.Modules[0]

	// MODULE without a name means current module
	testutil.Equal(t, "", cm.ModuleName, "ModuleName should be empty for current module")

	// MANDATORY-GROUPS
	testutil.SliceEqual(t, []string{"testGroup1", "testGroup2"}, cm.MandatoryGroups, "MandatoryGroups")

	// GROUP clauses
	testutil.Len(t, cm.Groups, 1, "Groups")
	testutil.Equal(t, "testOptionalGroup", cm.Groups[0].Group, "Groups[0].Group")
	testutil.Equal(t, "Conditionally required", cm.Groups[0].Description, "Groups[0].Description")

	// OBJECT clauses
	testutil.Len(t, cm.Objects, 2, "Objects")

	// First object: testObj1 with SYNTAX refinement
	obj1 := cm.Objects[0]
	testutil.Equal(t, "testObj1", obj1.Object, "Objects[0].Object")
	testutil.NotNil(t, obj1.Syntax, "Objects[0].Syntax should be non-nil")
	_, isSyntaxConstrained := obj1.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, isSyntaxConstrained, "Objects[0].Syntax should be *TypeSyntaxConstrained")
	testutil.Nil(t, obj1.WriteSyntax, "Objects[0].WriteSyntax should be nil")
	testutil.Nil(t, obj1.MinAccess, "Objects[0].MinAccess should be nil")
	testutil.Equal(t, "Refined range", obj1.Description, "Objects[0].Description")

	// Second object: testObj2 with MIN-ACCESS
	obj2 := cm.Objects[1]
	testutil.Equal(t, "testObj2", obj2.Object, "Objects[1].Object")
	testutil.Nil(t, obj2.Syntax, "Objects[1].Syntax should be nil")
	testutil.Nil(t, obj2.WriteSyntax, "Objects[1].WriteSyntax should be nil")
	testutil.NotNil(t, obj2.MinAccess, "Objects[1].MinAccess should be non-nil")
	testutil.Equal(t, types.AccessReadOnly, *obj2.MinAccess, "Objects[1].MinAccess value")
	testutil.Equal(t, "Reduced access", obj2.Description, "Objects[1].Description")
}

func TestLower_ModuleCompliance_NamedModule(t *testing.T) {
	source := `NAMED-COMPLIANCE-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, enterprises
        FROM SNMPv2-SMI
    MODULE-COMPLIANCE
        FROM SNMPv2-CONF;

namedCompTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99998 }

extCompliance MODULE-COMPLIANCE
    STATUS current
    DESCRIPTION "External module compliance"
    MODULE SNMPv2-MIB
        MANDATORY-GROUPS { systemGroup }
    ::= { namedCompTest 1 }

END
`

	mod := lowerModule(t, source)
	mc := requireDef[*ModuleCompliance](t, mod, "extCompliance")
	testutil.NotNil(t, mc, "expected extCompliance definition")

	testutil.Len(t, mc.Modules, 1, "Modules")
	cm := mc.Modules[0]

	testutil.Equal(t, "SNMPv2-MIB", cm.ModuleName, "ModuleName should be SNMPv2-MIB")
	testutil.SliceEqual(t, []string{"systemGroup"}, cm.MandatoryGroups, "MandatoryGroups")
	testutil.Len(t, cm.Groups, 0, "Groups should be empty")
	testutil.Len(t, cm.Objects, 0, "Objects should be empty")
}

func TestLower_AgentCapabilities(t *testing.T) {
	source := `AGENT-CAP-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, Integer32, enterprises
        FROM SNMPv2-SMI
    AGENT-CAPABILITIES
        FROM SNMPv2-CONF;

agentCapTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99997 }

testAgent AGENT-CAPABILITIES
    PRODUCT-RELEASE "Test Agent v1.0"
    STATUS current
    DESCRIPTION "Test agent capabilities"
    REFERENCE "Internal"
    SUPPORTS IF-MIB
        INCLUDES { ifGeneralGroup, ifStackGroup }
        VARIATION ifAdminStatus
            SYNTAX Integer32 (1..2)
            ACCESS read-only
            DESCRIPTION "Only up and down supported"
        VARIATION ifAlias
            DESCRIPTION "Not configurable"
    ::= { agentCapTest 1 }

END
`

	mod := lowerModule(t, source)
	ac := requireDef[*AgentCapabilities](t, mod, "testAgent")
	testutil.NotNil(t, ac, "expected testAgent definition")

	// Top-level fields
	testutil.Equal(t, "testAgent", ac.Name, "Name")
	testutil.Equal(t, "Test Agent v1.0", ac.ProductRelease, "ProductRelease")
	testutil.Equal(t, types.StatusCurrent, ac.Status, "Status")
	testutil.Equal(t, "Test agent capabilities", ac.Description, "Description")
	testutil.Equal(t, "Internal", ac.Reference, "Reference")

	// Supports
	testutil.Len(t, ac.Supports, 1, "Supports")
	sm := ac.Supports[0]

	testutil.Equal(t, "IF-MIB", sm.ModuleName, "ModuleName")
	testutil.SliceEqual(t, []string{"ifGeneralGroup", "ifStackGroup"}, sm.Includes, "Includes")
	testutil.Len(t, sm.Variations, 2, "Variations")

	// First variation: ifAdminStatus with SYNTAX and ACCESS
	v1 := sm.Variations[0]
	testutil.Equal(t, "ifAdminStatus", v1.Name, "Variations[0].Name")
	testutil.NotNil(t, v1.Syntax, "Variations[0].Syntax should be non-nil")
	_, isSyntaxConstrained := v1.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, isSyntaxConstrained, "Variations[0].Syntax should be *TypeSyntaxConstrained")
	testutil.NotNil(t, v1.Access, "Variations[0].Access should be non-nil")
	testutil.Equal(t, types.AccessReadOnly, *v1.Access, "Variations[0].Access value")
	testutil.Nil(t, v1.WriteSyntax, "Variations[0].WriteSyntax should be nil")
	testutil.Nil(t, v1.CreationRequires, "Variations[0].CreationRequires should be nil")
	testutil.Nil(t, v1.DefVal, "Variations[0].DefVal should be nil")
	testutil.Equal(t, "Only up and down supported", v1.Description, "Variations[0].Description")

	// Second variation: ifAlias with only DESCRIPTION
	v2 := sm.Variations[1]
	testutil.Equal(t, "ifAlias", v2.Name, "Variations[1].Name")
	testutil.Nil(t, v2.Syntax, "Variations[1].Syntax should be nil")
	testutil.Nil(t, v2.Access, "Variations[1].Access should be nil")
	testutil.Nil(t, v2.WriteSyntax, "Variations[1].WriteSyntax should be nil")
	testutil.Nil(t, v2.CreationRequires, "Variations[1].CreationRequires should be nil")
	testutil.Nil(t, v2.DefVal, "Variations[1].DefVal should be nil")
	testutil.Equal(t, "Not configurable", v2.Description, "Variations[1].Description")
}
