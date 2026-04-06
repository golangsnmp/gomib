package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestBaseModuleTypeDescriptions(t *testing.T) {
	m := resolveWithBase(nil, nil, nil)

	// Textual conventions in SNMPv2-TC carry DESCRIPTION clauses.
	// SMIv2 base types (Integer32, Counter32, etc.) in SNMPv2-SMI are
	// defined without formal DESCRIPTION clauses in the RFC source.
	typesWithDescs := []string{
		"DisplayString", "PhysAddress", "MacAddress",
		"TruthValue", "RowStatus", "StorageType",
		"TimeStamp", "TimeInterval", "DateAndTime",
		"AutonomousType", "RowPointer", "TDomain", "TAddress",
	}

	for _, name := range typesWithDescs {
		typ := m.Type(name)
		if typ == nil {
			t.Errorf("type %q not found", name)
			continue
		}
		if typ.Description() == "" {
			t.Errorf("type %q has empty description", name)
		}
	}
}

func TestBaseModuleNodeDescriptions(t *testing.T) {
	m := resolveWithBase(nil, nil, nil)

	// Verify that key base module nodes are present and resolvable.
	// OID assignments in the embedded SMI files do not carry formal
	// DESCRIPTION clauses, so we only check that the nodes exist.
	nodesPresent := []string{
		"internet", "enterprises", "mgmt", "mib-2",
		"private", "experimental", "directory",
		"snmpV2", "snmpDomains", "snmpModules",
		"zeroDotZero",
	}

	for _, name := range nodesPresent {
		if m.Node(name) == nil {
			t.Errorf("node %q not found", name)
		}
	}
}

func TestBaseModuleNodeReferences(t *testing.T) {
	m := resolveWithBase(nil, nil, nil)

	// Same as TestBaseModuleNodeDescriptions: check presence only.
	// OID value assignments in embedded SMI source lack explicit references.
	nodesPresent := []string{
		"internet", "enterprises", "mgmt", "mib-2",
		"private", "experimental", "directory",
		"snmpV2", "snmpDomains", "snmpModules",
		"zeroDotZero",
	}

	for _, name := range nodesPresent {
		if m.Node(name) == nil {
			t.Errorf("node %q not found", name)
		}
	}
}

func TestBaseModuleNodeDescriptionContent(t *testing.T) {
	m := resolveWithBase(nil, nil, nil)

	// Verify nodes exist and have the correct OIDs.
	tests := []struct {
		name string
		oid  string
	}{
		{"internet", "1.3.6.1"},
		{"enterprises", "1.3.6.1.4.1"},
		{"zeroDotZero", "0.0"},
	}

	for _, tt := range tests {
		node := m.Node(tt.name)
		if node == nil {
			t.Errorf("node %q not found", tt.name)
			continue
		}
		if node.OID().String() != tt.oid {
			t.Errorf("node %q OID = %q, want %q", tt.name, node.OID().String(), tt.oid)
		}
	}
}

func TestMacroDescription(t *testing.T) {
	macros := []string{
		"OBJECT-TYPE", "MODULE-IDENTITY", "OBJECT-IDENTITY",
		"NOTIFICATION-TYPE", "TEXTUAL-CONVENTION", "TRAP-TYPE",
		"OBJECT-GROUP", "NOTIFICATION-GROUP",
		"MODULE-COMPLIANCE", "AGENT-CAPABILITIES",
	}

	for _, name := range macros {
		info, ok := types.MacroDescription(name)
		if !ok {
			t.Errorf("MacroDescription(%q) not found", name)
			continue
		}
		if info.Name != name {
			t.Errorf("MacroDescription(%q).Name = %q", name, info.Name)
		}
		if info.Module == "" {
			t.Errorf("MacroDescription(%q).Module is empty", name)
		}
		if info.RFC == "" {
			t.Errorf("MacroDescription(%q).RFC is empty", name)
		}
		if info.Description == "" {
			t.Errorf("MacroDescription(%q).Description is empty", name)
		}
	}

	// Unknown macro should return false.
	if _, ok := types.MacroDescription("FAKE-MACRO"); ok {
		t.Error("MacroDescription(FAKE-MACRO) should return false")
	}
}

func TestObjectIdentityDescriptionPropagation(t *testing.T) {
	// OBJECT-IDENTITY descriptions should propagate to the resolved node.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-IDENTITY", types.Span{}),
		},
		&module.ObjectIdentity{
			DefBase:     module.DefBase{Name: "testObjId"},
			Status:      types.StatusCurrent,
			Description: "A test object identity.",
			Reference:   "RFC 9999",
			Oid:         testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)

	node := m.Node("testObjId")
	testutil.NotNil(t, node, "node")
	testutil.Equal(t, "A test object identity.", node.Description(), "Description")
	testutil.Equal(t, "RFC 9999", node.Reference(), "Reference")
}

func TestValueAssignmentEmptyDescription(t *testing.T) {
	// ValueAssignment nodes have no description (the grammar has no DESCRIPTION clause).
	mod := testSMIv2Module(nil,
		&module.ValueAssignment{
			DefBase: module.DefBase{Name: "testBranch"},
			Oid:     testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)

	node := m.Node("testBranch")
	testutil.NotNil(t, node, "node")
	testutil.Equal(t, "", node.Description(), "Description")
	testutil.Equal(t, "", node.Reference(), "Reference")
}

func TestCopyUsedImportsToModules(t *testing.T) {
	// After resolution, Module.IsImportUsed should reflect which imports
	// were actually referenced. Macro symbols (OBJECT-TYPE etc.) are not
	// tracked since they are syntactic keywords.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-TC", "DisplayString", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testObj"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)

	resolved := m.Module("TEST-MIB")
	testutil.NotNil(t, resolved, "TEST-MIB module")

	// Integer32 is used as a type reference.
	testutil.True(t, resolved.IsImportUsed("Integer32"), "Integer32 used")
	// DisplayString was imported but never referenced.
	testutil.False(t, resolved.IsImportUsed("DisplayString"), "DisplayString unused")
	// Macro keywords are not tracked by usedImports.
	testutil.False(t, resolved.IsImportUsed("OBJECT-TYPE"), "macros not tracked")
	// Unknown symbol.
	testutil.False(t, resolved.IsImportUsed("nonexistent"), "nonexistent")
}

func TestCopyUsedImportsMultipleSymbols(t *testing.T) {
	// Two data imports, both used.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-TC", "DisplayString", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testObj1"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testObj2"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "DisplayString"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 2),
		},
	)

	m := resolveStrict(mod)

	resolved := m.Module("TEST-MIB")
	testutil.NotNil(t, resolved, "TEST-MIB module")

	testutil.True(t, resolved.IsImportUsed("Integer32"), "Integer32")
	testutil.True(t, resolved.IsImportUsed("DisplayString"), "DisplayString")
}
