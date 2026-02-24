package gomib

import (
	"context"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func loadCapabilityMIB(t testing.TB) *mib.Mib {
	t.Helper()
	corpus, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree corpus failed: %v", err)
	}
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

func TestModuleCapabilityLookup(t *testing.T) {
	m := loadCapabilityMIB(t)

	mod := m.Module("JNX-SNMPv2-CAPABILITY")
	if mod == nil {
		t.Fatal("JNX-SNMPv2-CAPABILITY not found")
	}

	c := mod.Capability("jnxSnmpV2CapJunos")
	testutil.NotNil(t, c, "Module.Capability(jnxSnmpV2CapJunos) should not be nil")
	testutil.Equal(t, "jnxSnmpV2CapJunos", c.Name(), "capability name")

	testutil.Nil(t, mod.Capability("noSuchCapability"),
		"non-existent capability should return nil")
}

func TestCapabilityMibLookup(t *testing.T) {
	m := loadCapabilityMIB(t)

	c := m.Capability("jnxSnmpV2CapJunos")
	testutil.NotNil(t, c, "Mib.Capability(jnxSnmpV2CapJunos) should not be nil")
	testutil.Equal(t, "jnxSnmpV2CapJunos", c.Name(), "capability name")

	testutil.Nil(t, m.Capability("noSuchCapability"),
		"non-existent capability should return nil")
}

func TestCapabilityMetadata(t *testing.T) {
	m := loadCapabilityMIB(t)

	c := m.Capability("jnxSnmpV2CapJunos")
	if c == nil {
		t.Fatal("jnxSnmpV2CapJunos not found")
	}

	testutil.Equal(t, "jnxSnmpV2CapJunos", c.Name(), "capability name")
	testutil.Equal(t, mib.StatusCurrent, c.Status(), "status should be current")

	testutil.Contains(t, c.Description(), "JUNOS SNMPv2 MIB capabilities",
		"description should mention JUNOS")
	testutil.Equal(t, "All JUNOS Version", c.ProductRelease(),
		"PRODUCT-RELEASE value")

	node := c.Node()
	testutil.NotNil(t, node, "Capability.Node() should not be nil")
	testutil.Equal(t, "jnxSnmpV2CapJunos", node.Name(), "node name matches capability")
	testutil.Equal(t, mib.KindCapability, node.Kind(), "node kind should be KindCapability")

	mod := c.Module()
	testutil.NotNil(t, mod, "Capability.Module() should not be nil")
	testutil.Equal(t, "JNX-SNMPv2-CAPABILITY", mod.Name(), "capability module")

	oid := c.OID()
	testutil.Greater(t, len(oid), 0, "capability OID should not be empty")
}

func TestNodeCapability(t *testing.T) {
	m := loadCapabilityMIB(t)

	node := m.Node("jnxSnmpV2CapJunos")
	if node == nil {
		t.Fatal("jnxSnmpV2CapJunos node not found")
	}

	c := node.Capability()
	testutil.NotNil(t, c, "capability node should have Capability()")
	testutil.Equal(t, "jnxSnmpV2CapJunos", c.Name(), "node Capability() name")

	// MODULE-IDENTITY node should not have a capability
	identity := m.Node("jnxSnmpV2Capability")
	if identity != nil {
		testutil.Nil(t, identity.Capability(),
			"MODULE-IDENTITY node should not have Capability()")
	}
}

func TestCapabilityByOID(t *testing.T) {
	m := loadCapabilityMIB(t)

	c := m.Capability("jnxSnmpV2CapJunos")
	if c == nil {
		t.Fatal("jnxSnmpV2CapJunos not found")
	}

	nd := m.NodeByOID(c.OID())
	testutil.NotNil(t, nd, "NodeByOID should find capability node")
	testutil.NotNil(t, nd.Capability(), "node should have Capability()")
	testutil.Equal(t, "jnxSnmpV2CapJunos", nd.Capability().Name(),
		"capability found by OID")
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
