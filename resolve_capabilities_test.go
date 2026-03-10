package gomib

import (
	"context"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func loadCapabilityMIB(t testing.TB) *mib.Mib {
	t.Helper()
	corpus := mustDir(t, testutil.PrimaryCorpusDir())
	// Load both JNX-SNMPv2-CAPABILITY and SNMPv2-MIB so the resolver can
	// look up variation targets in the SUPPORTS module.
	m, err := Load(context.Background(),
		WithSource(corpus),
		WithModules("JNX-SNMPv2-CAPABILITY", "SNMPv2-MIB"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	return m
}

func TestModuleCapabilities(t *testing.T) {
	m := loadCapabilityMIB(t)

	mod := m.Module("JNX-SNMPv2-CAPABILITY")
	if mod == nil {
		t.Fatal("JNX-SNMPv2-CAPABILITY not found")
	}

	caps := mod.Capabilities()
	testutil.Equal(t, 1, len(caps),
		"JNX-SNMPv2-CAPABILITY should have 1 capability")

	testutil.Equal(t, "jnxSnmpV2CapJunos", caps[0].Name(),
		"capability name")
}

func TestCapabilityConformanceNode(t *testing.T) {
	m := loadCapabilityMIB(t)

	checkConformanceNode(t, m, conformanceNodeSpec{
		name:   "jnxSnmpV2CapJunos",
		module: "JNX-SNMPv2-CAPABILITY",
		kind:   mib.KindCapability,
		status: mib.StatusCurrent,
	},
		func(m *mib.Mib, name string) *mib.Capability { return m.Capability(name) },
		func(mod *mib.Module, name string) *mib.Capability { return mod.Capability(name) },
		(*mib.Capability).Node,
		(*mib.Capability).Module,
		(*mib.Capability).OID,
		(*mib.Capability).Status,
		(*mib.Capability).Description,
		(*mib.Node).Capability,
		isNilCapability,
	)
}

func TestCapabilityExtraMetadata(t *testing.T) {
	m := loadCapabilityMIB(t)

	c := m.Capability("jnxSnmpV2CapJunos")
	if c == nil {
		t.Fatal("jnxSnmpV2CapJunos not found")
	}

	testutil.Contains(t, c.Description(), "JUNOS SNMPv2 MIB capabilities",
		"description should mention JUNOS")
	testutil.Equal(t, "All JUNOS Version", c.ProductRelease(),
		"PRODUCT-RELEASE value")
}

func TestNodeCapabilityNonCapability(t *testing.T) {
	m := loadCapabilityMIB(t)

	// MODULE-IDENTITY node should not have a capability
	identity := m.Node("jnxSnmpV2Capability")
	if identity != nil {
		testutil.Nil(t, identity.Capability(),
			"MODULE-IDENTITY node should not have Capability()")
	}
}

func TestCapabilitySupports(t *testing.T) {
	m := loadCapabilityMIB(t)

	c := m.Capability("jnxSnmpV2CapJunos")
	if c == nil {
		t.Fatal("jnxSnmpV2CapJunos not found")
	}

	supports := c.Supports()
	testutil.Equal(t, 1, len(supports), "should have 1 SUPPORTS clause")

	if len(supports) == 0 {
		return
	}

	sup := supports[0]
	testutil.Equal(t, "SNMPv2-MIB", sup.ModuleName, "supported module name")

	// INCLUDES { systemGroup, snmpGroup, snmpCommunityGroup,
	//            snmpSetGroup, snmpBasicNotificationsGroup }
	testutil.Equal(t, 5, len(sup.Includes), "should have 5 INCLUDES groups")

	includeSet := make(map[string]bool)
	for _, g := range sup.Includes {
		includeSet[g] = true
	}
	testutil.True(t, includeSet["systemGroup"], "should include systemGroup")
	testutil.True(t, includeSet["snmpGroup"], "should include snmpGroup")
	testutil.True(t, includeSet["snmpCommunityGroup"], "should include snmpCommunityGroup")
	testutil.True(t, includeSet["snmpSetGroup"], "should include snmpSetGroup")
	testutil.True(t, includeSet["snmpBasicNotificationsGroup"],
		"should include snmpBasicNotificationsGroup")

	// All 7 VARIATIONs reference OBJECT-TYPE names from SNMPv2-MIB.
	// The resolver classifies them as ObjectVariations by looking up
	// each name's node kind in the SUPPORTS module.
	testutil.Equal(t, 7, len(sup.ObjectVariations),
		"all 7 variations reference objects, not notifications")
	testutil.Equal(t, 0, len(sup.NotificationVariations),
		"should have no notification variations")

	variationMap := make(map[string]mib.ObjectVariation)
	for _, v := range sup.ObjectVariations {
		variationMap[v.Object] = v
	}

	// sysDescr: ACCESS read-only
	sysDescr, ok := variationMap["sysDescr"]
	testutil.True(t, ok, "should have sysDescr variation")
	testutil.NotNil(t, sysDescr.Access, "sysDescr should have ACCESS")
	testutil.Equal(t, mib.AccessReadOnly, *sysDescr.Access,
		"sysDescr ACCESS should be read-only")

	// sysORLastChange: ACCESS not-implemented
	sysORLastChange, ok := variationMap["sysORLastChange"]
	testutil.True(t, ok, "should have sysORLastChange variation")
	testutil.NotNil(t, sysORLastChange.Access, "sysORLastChange should have ACCESS")
	testutil.Equal(t, mib.AccessNotImplemented, *sysORLastChange.Access,
		"sysORLastChange ACCESS should be not-implemented")
}

func TestCapabilityString(t *testing.T) {
	m := loadCapabilityMIB(t)

	c := m.Capability("jnxSnmpV2CapJunos")
	if c == nil {
		t.Fatal("jnxSnmpV2CapJunos not found")
	}

	s := c.String()
	testutil.Contains(t, s, "jnxSnmpV2CapJunos", "String() should contain name")
	testutil.Contains(t, s, "(", "String() should contain OID in parens")
}
