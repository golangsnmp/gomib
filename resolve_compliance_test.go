package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func TestComplianceCount(t *testing.T) {
	m := loadTestMIB(t)

	compliances := m.Compliances()
	count := len(compliances)

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

func TestComplianceConformanceNode(t *testing.T) {
	m := loadTestMIB(t)

	checkConformanceNode(t, m, conformanceNodeSpec{
		name:       "snmpBasicComplianceRev2",
		module:     "SNMPv2-MIB",
		kind:       mib.KindCompliance,
		status:     mib.StatusCurrent,
		wrongOwner: "IF-MIB",
	},
		func(m *mib.Mib, name string) *mib.Compliance { return m.Compliance(name) },
		func(mod *mib.Module, name string) *mib.Compliance { return mod.Compliance(name) },
		(*mib.Compliance).Node,
		(*mib.Compliance).Module,
		(*mib.Compliance).OID,
		(*mib.Compliance).Status,
		(*mib.Compliance).Description,
		(*mib.Node).Compliance,
		isNilCompliance,
	)
}

func TestComplianceNodeNonCompliance(t *testing.T) {
	m := loadTestMIB(t)

	ifIndex := m.Node("ifIndex")
	if ifIndex == nil {
		t.Fatal("ifIndex not found")
	}
	testutil.Nil(t, ifIndex.Compliance(), "ifIndex should not have a Compliance()")
}

func TestDeprecatedCompliance(t *testing.T) {
	m := loadTestMIB(t)

	c := m.Compliance("snmpBasicCompliance")
	testutil.NotNil(t, c, "Compliance(snmpBasicCompliance)")

	testutil.Equal(t, mib.StatusDeprecated, c.Status(),
		"snmpBasicCompliance should be deprecated")
}

func TestComplianceModules(t *testing.T) {
	m := loadTestMIB(t)

	c := m.Compliance("snmpBasicComplianceRev2")
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
