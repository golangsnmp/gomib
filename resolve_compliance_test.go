package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// === Compliance collection access ===

func TestComplianceCount(t *testing.T) {
	m := loadTestMIB(t)

	count := m.ComplianceCount()
	compliances := m.Compliances()
	testutil.Equal(t, count, len(compliances),
		"ComplianceCount() should match Compliances() length")

	// SNMPv2-MIB defines 2, IF-MIB defines 3, etc.
	testutil.Greater(t, count, 0, "should have compliances from fixture MIBs")
}

func TestModuleCompliances(t *testing.T) {
	m := loadTestMIB(t)

	snmpMIB := m.Module("SNMPv2-MIB")
	if snmpMIB == nil {
		t.Fatal("SNMPv2-MIB not found")
	}

	compliances := snmpMIB.Compliances()
	testutil.Equal(t, 2, len(compliances),
		"SNMPv2-MIB should have 2 compliances")

	names := make(map[string]bool)
	for _, c := range compliances {
		names[c.Name()] = true
	}

	testutil.True(t, names["snmpBasicCompliance"],
		"SNMPv2-MIB should contain snmpBasicCompliance")
	testutil.True(t, names["snmpBasicComplianceRev2"],
		"SNMPv2-MIB should contain snmpBasicComplianceRev2")
}

func TestModuleComplianceLookup(t *testing.T) {
	m := loadTestMIB(t)

	snmpMIB := m.Module("SNMPv2-MIB")
	if snmpMIB == nil {
		t.Fatal("SNMPv2-MIB not found")
	}

	c := snmpMIB.ComplianceByName("snmpBasicComplianceRev2")
	testutil.NotNil(t, c, "Module.ComplianceByName(snmpBasicComplianceRev2) should not be nil")
	if c != nil {
		testutil.Equal(t, "snmpBasicComplianceRev2", c.Name(), "compliance name")
	}

	testutil.Nil(t, snmpMIB.ComplianceByName("noSuchCompliance"),
		"non-existent compliance should return nil")
}

// === FindCompliance lookups ===

func TestFindComplianceByName(t *testing.T) {
	m := loadTestMIB(t)

	c := m.FindCompliance("snmpBasicComplianceRev2")
	testutil.NotNil(t, c, "FindCompliance(snmpBasicComplianceRev2) should not be nil")
	if c != nil {
		testutil.Equal(t, "snmpBasicComplianceRev2", c.Name(), "compliance name")
	}
}

func TestFindComplianceByQualifiedName(t *testing.T) {
	m := loadTestMIB(t)

	c := m.FindCompliance("SNMPv2-MIB::snmpBasicComplianceRev2")
	testutil.NotNil(t, c, "FindCompliance(SNMPv2-MIB::snmpBasicComplianceRev2) should not be nil")
	if c != nil {
		testutil.Equal(t, "snmpBasicComplianceRev2", c.Name(), "compliance name")
	}

	// Wrong module
	testutil.Nil(t, m.FindCompliance("IF-MIB::snmpBasicComplianceRev2"),
		"snmpBasicComplianceRev2 should not be in IF-MIB")
}

func TestFindComplianceByOID(t *testing.T) {
	m := loadTestMIB(t)

	c := m.FindCompliance("snmpBasicComplianceRev2")
	if c == nil {
		t.Skip("snmpBasicComplianceRev2 not found by name")
		return
	}

	oid := c.OID().String()

	// Numeric OID lookup
	c2 := m.FindCompliance(oid)
	testutil.NotNil(t, c2, "FindCompliance by numeric OID %s should work", oid)
	if c2 != nil {
		testutil.Equal(t, "snmpBasicComplianceRev2", c2.Name(), "compliance found by OID")
	}

	// Dotted OID lookup
	c3 := m.FindCompliance("." + oid)
	testutil.NotNil(t, c3, "FindCompliance by dotted OID .%s should work", oid)
}

func TestFindComplianceNotFound(t *testing.T) {
	m := loadTestMIB(t)

	testutil.Nil(t, m.FindCompliance("noSuchCompliance"),
		"non-existent compliance name should return nil")
	testutil.Nil(t, m.FindCompliance("99.99.99"),
		"non-existent OID should return nil")
	testutil.Nil(t, m.FindCompliance("FAKE-MIB::snmpBasicCompliance"),
		"non-existent module should return nil")
}

// === Compliance metadata ===

func TestComplianceMetadata(t *testing.T) {
	m := loadTestMIB(t)

	c := m.FindCompliance("snmpBasicComplianceRev2")
	if c == nil {
		t.Fatal("snmpBasicComplianceRev2 not found")
	}

	// Name
	testutil.Equal(t, "snmpBasicComplianceRev2", c.Name(), "compliance name")

	// Node
	node := c.Node()
	testutil.NotNil(t, node, "Compliance.Node() should not be nil")
	if node != nil {
		testutil.Equal(t, "snmpBasicComplianceRev2", node.Name(), "node name matches compliance")
		testutil.Equal(t, mib.KindCompliance, node.Kind(), "node kind should be KindCompliance")
	}

	// Module
	mod := c.Module()
	testutil.NotNil(t, mod, "Compliance.Module() should not be nil")
	if mod != nil {
		testutil.Equal(t, "SNMPv2-MIB", mod.Name(), "compliance module")
	}

	// OID
	oid := c.OID()
	testutil.Greater(t, len(oid), 0, "compliance OID should not be empty")

	// Status
	testutil.Equal(t, mib.StatusCurrent, c.Status(), "snmpBasicComplianceRev2 should be current")

	// Description
	desc := c.Description()
	testutil.Greater(t, len(desc), 0, "snmpBasicComplianceRev2 should have a description")
}

func TestDeprecatedCompliance(t *testing.T) {
	m := loadTestMIB(t)

	c := m.FindCompliance("snmpBasicCompliance")
	if c == nil {
		t.Skip("snmpBasicCompliance not found")
		return
	}

	testutil.Equal(t, mib.StatusDeprecated, c.Status(),
		"snmpBasicCompliance should be deprecated")
}

// === Compliance modules (MODULE clauses) ===

func TestComplianceModules(t *testing.T) {
	m := loadTestMIB(t)

	c := m.FindCompliance("snmpBasicComplianceRev2")
	if c == nil {
		t.Fatal("snmpBasicComplianceRev2 not found")
	}

	modules := c.Modules()
	testutil.Equal(t, 1, len(modules),
		"snmpBasicComplianceRev2 should have 1 MODULE clause")

	if len(modules) == 0 {
		return
	}

	mod := modules[0]

	// MANDATORY-GROUPS { snmpGroup, snmpSetGroup, systemGroup, snmpBasicNotificationsGroup }
	testutil.Equal(t, 4, len(mod.MandatoryGroups),
		"should have 4 mandatory groups")

	mandatorySet := make(map[string]bool)
	for _, g := range mod.MandatoryGroups {
		mandatorySet[g] = true
	}
	testutil.True(t, mandatorySet["snmpGroup"], "snmpGroup should be mandatory")
	testutil.True(t, mandatorySet["systemGroup"], "systemGroup should be mandatory")

	// GROUP clauses: snmpCommunityGroup, snmpWarmStartNotificationGroup
	testutil.Equal(t, 2, len(mod.Groups),
		"should have 2 GROUP refinements")
}

// === Node.Compliance() ===

func TestNodeCompliance(t *testing.T) {
	m := loadTestMIB(t)

	// Compliance node should have associated compliance
	node := m.FindNode("snmpBasicComplianceRev2")
	if node == nil {
		t.Fatal("snmpBasicComplianceRev2 node not found")
	}

	c := node.Compliance()
	testutil.NotNil(t, c, "compliance node should have Compliance()")
	if c != nil {
		testutil.Equal(t, "snmpBasicComplianceRev2", c.Name(), "node Compliance() name")
	}

	// Non-compliance node should return nil
	ifIndex := m.FindNode("ifIndex")
	if ifIndex == nil {
		t.Fatal("ifIndex not found")
	}
	testutil.Nil(t, ifIndex.Compliance(), "ifIndex should not have a Compliance()")
}

// === Node.Module() for compliance nodes ===

func TestComplianceNodeModule(t *testing.T) {
	m := loadTestMIB(t)

	node := m.FindNode("snmpBasicComplianceRev2")
	if node == nil {
		t.Fatal("snmpBasicComplianceRev2 node not found")
	}

	mod := node.Module()
	testutil.NotNil(t, mod, "compliance node should have a Module()")
	if mod != nil {
		testutil.Equal(t, "SNMPv2-MIB", mod.Name(),
			"snmpBasicComplianceRev2 node module should be SNMPv2-MIB")
	}
}
