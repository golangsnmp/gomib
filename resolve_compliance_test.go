package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

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

	c := snmpMIB.Compliance("snmpBasicComplianceRev2")
	testutil.NotNil(t, c, "Module.Compliance(snmpBasicComplianceRev2) should not be nil")
	testutil.Equal(t, "snmpBasicComplianceRev2", c.Name(), "compliance name")

	testutil.Nil(t, snmpMIB.Compliance("noSuchCompliance"),
		"non-existent compliance should return nil")
}

func TestFindCompliance(t *testing.T) {
	m := loadTestMIB(t)

	c := m.FindCompliance("snmpBasicComplianceRev2")
	testutil.NotNil(t, c, "FindCompliance(snmpBasicComplianceRev2) should not be nil")
	testutil.Equal(t, "snmpBasicComplianceRev2", c.Name(), "compliance name")
}

func TestFindComplianceByQualifiedName(t *testing.T) {
	m := loadTestMIB(t)

	c := m.FindCompliance("SNMPv2-MIB::snmpBasicComplianceRev2")
	testutil.NotNil(t, c, "FindCompliance(SNMPv2-MIB::snmpBasicComplianceRev2) should not be nil")
	testutil.Equal(t, "snmpBasicComplianceRev2", c.Name(), "compliance name")

	testutil.Nil(t, m.FindCompliance("IF-MIB::snmpBasicComplianceRev2"),
		"snmpBasicComplianceRev2 should not be in IF-MIB")
}

func TestFindComplianceByOID(t *testing.T) {
	m := loadTestMIB(t)

	c := m.FindCompliance("snmpBasicComplianceRev2")
	testutil.NotNil(t, c, "FindCompliance(snmpBasicComplianceRev2)")

	oid := c.OID().String()

	c2 := m.FindCompliance(oid)
	testutil.NotNil(t, c2, "FindCompliance by numeric OID %s should work", oid)
	testutil.Equal(t, "snmpBasicComplianceRev2", c2.Name(), "compliance found by OID")

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

func TestComplianceMetadata(t *testing.T) {
	m := loadTestMIB(t)

	c := m.FindCompliance("snmpBasicComplianceRev2")
	if c == nil {
		t.Fatal("snmpBasicComplianceRev2 not found")
	}

	testutil.Equal(t, "snmpBasicComplianceRev2", c.Name(), "compliance name")

	node := c.Node()
	testutil.NotNil(t, node, "Compliance.Node() should not be nil")
	testutil.Equal(t, "snmpBasicComplianceRev2", node.Name(), "node name matches compliance")
	testutil.Equal(t, mib.KindCompliance, node.Kind(), "node kind should be KindCompliance")

	mod := c.Module()
	testutil.NotNil(t, mod, "Compliance.Module() should not be nil")
	testutil.Equal(t, "SNMPv2-MIB", mod.Name(), "compliance module")

	oid := c.OID()
	testutil.Greater(t, len(oid), 0, "compliance OID should not be empty")

	testutil.Equal(t, mib.StatusCurrent, c.Status(), "snmpBasicComplianceRev2 should be current")

	desc := c.Description()
	testutil.Greater(t, len(desc), 0, "snmpBasicComplianceRev2 should have a description")
}

func TestDeprecatedCompliance(t *testing.T) {
	m := loadTestMIB(t)

	c := m.FindCompliance("snmpBasicCompliance")
	testutil.NotNil(t, c, "FindCompliance(snmpBasicCompliance)")

	testutil.Equal(t, mib.StatusDeprecated, c.Status(),
		"snmpBasicCompliance should be deprecated")
}

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

func TestNodeCompliance(t *testing.T) {
	m := loadTestMIB(t)

	node := m.FindNode("snmpBasicComplianceRev2")
	if node == nil {
		t.Fatal("snmpBasicComplianceRev2 node not found")
	}

	c := node.Compliance()
	testutil.NotNil(t, c, "compliance node should have Compliance()")
	testutil.Equal(t, "snmpBasicComplianceRev2", c.Name(), "node Compliance() name")

	ifIndex := m.FindNode("ifIndex")
	if ifIndex == nil {
		t.Fatal("ifIndex not found")
	}
	testutil.Nil(t, ifIndex.Compliance(), "ifIndex should not have a Compliance()")
}

func TestComplianceNodeModule(t *testing.T) {
	m := loadTestMIB(t)

	node := m.FindNode("snmpBasicComplianceRev2")
	if node == nil {
		t.Fatal("snmpBasicComplianceRev2 node not found")
	}

	mod := node.Module()
	testutil.NotNil(t, mod, "compliance node should have a Module()")
	testutil.Equal(t, "SNMPv2-MIB", mod.Name(),
		"snmpBasicComplianceRev2 node module should be SNMPv2-MIB")
}
