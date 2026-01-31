package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// === Group collection access ===

func TestGroupCount(t *testing.T) {
	m := loadTestMIB(t)

	count := m.GroupCount()
	groups := m.Groups()
	testutil.Equal(t, count, len(groups),
		"GroupCount() should match Groups() length")

	// SNMPv2-MIB alone defines 8 groups, so combined fixture modules should have more
	testutil.Greater(t, count, 0, "should have groups from fixture MIBs")
}

func TestModuleGroups(t *testing.T) {
	m := loadTestMIB(t)

	snmpMIB := m.Module("SNMPv2-MIB")
	if snmpMIB == nil {
		t.Fatal("SNMPv2-MIB not found")
	}

	groups := snmpMIB.Groups()
	testutil.Greater(t, len(groups), 0,
		"SNMPv2-MIB should have groups")

	names := make(map[string]bool)
	for _, g := range groups {
		names[g.Name()] = true
	}

	// SNMPv2-MIB defines: snmpGroup, snmpCommunityGroup, snmpSetGroup,
	// systemGroup, snmpBasicNotificationsGroup, snmpWarmStartNotificationGroup,
	// snmpNotificationGroup, snmpObsoleteGroup
	testutil.True(t, names["snmpGroup"],
		"SNMPv2-MIB should contain snmpGroup")
	testutil.True(t, names["systemGroup"],
		"SNMPv2-MIB should contain systemGroup")
	testutil.True(t, names["snmpBasicNotificationsGroup"],
		"SNMPv2-MIB should contain snmpBasicNotificationsGroup")
}

func TestModuleGroupLookup(t *testing.T) {
	m := loadTestMIB(t)

	snmpMIB := m.Module("SNMPv2-MIB")
	if snmpMIB == nil {
		t.Fatal("SNMPv2-MIB not found")
	}

	g := snmpMIB.Group("snmpGroup")
	testutil.NotNil(t, g, "Module.Group(snmpGroup) should not be nil")
	if g != nil {
		testutil.Equal(t, "snmpGroup", g.Name(), "group name")
	}

	// Non-existent group
	testutil.Nil(t, snmpMIB.Group("noSuchGroup"),
		"non-existent group should return nil")
}

// === FindGroup lookups ===

func TestFindGroupByName(t *testing.T) {
	m := loadTestMIB(t)

	g := m.FindGroup("snmpGroup")
	testutil.NotNil(t, g, "FindGroup(snmpGroup) should not be nil")
	if g != nil {
		testutil.Equal(t, "snmpGroup", g.Name(), "group name")
	}
}

func TestFindGroupByQualifiedName(t *testing.T) {
	m := loadTestMIB(t)

	g := m.FindGroup("SNMPv2-MIB::snmpGroup")
	testutil.NotNil(t, g, "FindGroup(SNMPv2-MIB::snmpGroup) should not be nil")
	if g != nil {
		testutil.Equal(t, "snmpGroup", g.Name(), "group name")
	}

	// Wrong module
	testutil.Nil(t, m.FindGroup("IF-MIB::snmpGroup"),
		"snmpGroup should not be in IF-MIB")
}

func TestFindGroupByOID(t *testing.T) {
	m := loadTestMIB(t)

	g := m.FindGroup("snmpGroup")
	if g == nil {
		t.Skip("snmpGroup not found by name")
		return
	}

	oid := g.OID().String()

	// Numeric OID lookup
	g2 := m.FindGroup(oid)
	testutil.NotNil(t, g2, "FindGroup by numeric OID %s should work", oid)
	if g2 != nil {
		testutil.Equal(t, "snmpGroup", g2.Name(), "group found by OID")
	}

	// Dotted OID lookup
	g3 := m.FindGroup("." + oid)
	testutil.NotNil(t, g3, "FindGroup by dotted OID .%s should work", oid)
}

func TestFindGroupNotFound(t *testing.T) {
	m := loadTestMIB(t)

	testutil.Nil(t, m.FindGroup("noSuchGroup"),
		"non-existent group name should return nil")
	testutil.Nil(t, m.FindGroup("99.99.99"),
		"non-existent OID should return nil")
	testutil.Nil(t, m.FindGroup("FAKE-MIB::snmpGroup"),
		"non-existent module should return nil")
}

// === Group metadata ===

func TestGroupMetadata(t *testing.T) {
	m := loadTestMIB(t)

	g := m.FindGroup("snmpGroup")
	if g == nil {
		t.Fatal("snmpGroup not found")
	}

	// Name
	testutil.Equal(t, "snmpGroup", g.Name(), "group name")

	// Node
	node := g.Node()
	testutil.NotNil(t, node, "Group.Node() should not be nil")
	if node != nil {
		testutil.Equal(t, "snmpGroup", node.Name(), "node name matches group")
		testutil.Equal(t, mib.KindGroup, node.Kind(), "node kind should be KindGroup")
	}

	// Module
	mod := g.Module()
	testutil.NotNil(t, mod, "Group.Module() should not be nil")
	if mod != nil {
		testutil.Equal(t, "SNMPv2-MIB", mod.Name(), "group module")
	}

	// OID
	oid := g.OID()
	testutil.Greater(t, len(oid), 0, "group OID should not be empty")

	// Status
	testutil.Equal(t, mib.StatusCurrent, g.Status(), "snmpGroup should be current")

	// Description
	desc := g.Description()
	testutil.Greater(t, len(desc), 0, "snmpGroup should have a description")

	// Not a notification group
	testutil.False(t, g.IsNotificationGroup(),
		"snmpGroup is an OBJECT-GROUP, not NOTIFICATION-GROUP")
}

// === Group members ===

func TestObjectGroupMembers(t *testing.T) {
	m := loadTestMIB(t)

	g := m.FindGroup("snmpGroup")
	if g == nil {
		t.Fatal("snmpGroup not found")
	}

	members := g.Members()
	// snmpGroup OBJECTS { snmpInPkts, snmpInBadVersions, snmpInASNParseErrs,
	//                     snmpSilentDrops, snmpProxyDrops, snmpEnableAuthenTraps }
	testutil.Equal(t, 6, len(members),
		"snmpGroup should have 6 members")

	names := make(map[string]bool)
	for _, nd := range members {
		names[nd.Name()] = true
	}
	testutil.True(t, names["snmpInPkts"], "snmpInPkts should be a member")
	testutil.True(t, names["snmpSilentDrops"], "snmpSilentDrops should be a member")
	testutil.True(t, names["snmpEnableAuthenTraps"], "snmpEnableAuthenTraps should be a member")
}

func TestNotificationGroupMembers(t *testing.T) {
	m := loadTestMIB(t)

	g := m.FindGroup("snmpBasicNotificationsGroup")
	if g == nil {
		t.Fatal("snmpBasicNotificationsGroup not found")
	}

	testutil.True(t, g.IsNotificationGroup(),
		"snmpBasicNotificationsGroup should be a NOTIFICATION-GROUP")

	members := g.Members()
	// NOTIFICATIONS { coldStart, authenticationFailure }
	testutil.Equal(t, 2, len(members),
		"snmpBasicNotificationsGroup should have 2 members")

	names := make(map[string]bool)
	for _, nd := range members {
		names[nd.Name()] = true
	}
	testutil.True(t, names["coldStart"], "coldStart should be a member")
	testutil.True(t, names["authenticationFailure"],
		"authenticationFailure should be a member")
}

// === Node.Group() ===

func TestNodeGroup(t *testing.T) {
	m := loadTestMIB(t)

	// Group node should have associated group
	node := m.FindNode("snmpGroup")
	if node == nil {
		t.Fatal("snmpGroup node not found")
	}

	g := node.Group()
	testutil.NotNil(t, g, "group node should have Group()")
	if g != nil {
		testutil.Equal(t, "snmpGroup", g.Name(), "node Group() name")
	}

	// Non-group node should return nil
	ifIndex := m.FindNode("ifIndex")
	if ifIndex == nil {
		t.Fatal("ifIndex not found")
	}
	testutil.Nil(t, ifIndex.Group(), "ifIndex should not have a Group()")
}

// === Node.Module() for group nodes ===

func TestGroupNodeModule(t *testing.T) {
	m := loadTestMIB(t)

	node := m.FindNode("snmpGroup")
	if node == nil {
		t.Fatal("snmpGroup node not found")
	}

	mod := node.Module()
	testutil.NotNil(t, mod, "group node should have a Module()")
	if mod != nil {
		testutil.Equal(t, "SNMPv2-MIB", mod.Name(),
			"snmpGroup node module should be SNMPv2-MIB")
	}
}

// === Smaller groups ===

func TestSmallObjectGroup(t *testing.T) {
	m := loadTestMIB(t)

	// snmpCommunityGroup has 2 members:
	// OBJECTS { snmpInBadCommunityNames, snmpInBadCommunityUses }
	g := m.FindGroup("snmpCommunityGroup")
	if g == nil {
		t.Skip("snmpCommunityGroup not found")
		return
	}

	testutil.False(t, g.IsNotificationGroup(),
		"snmpCommunityGroup is an OBJECT-GROUP")
	testutil.Equal(t, 2, len(g.Members()),
		"snmpCommunityGroup should have 2 members")
}

func TestWarmStartNotificationGroup(t *testing.T) {
	m := loadTestMIB(t)

	// snmpWarmStartNotificationGroup NOTIFICATION-GROUP
	// NOTIFICATIONS { warmStart }
	g := m.FindGroup("snmpWarmStartNotificationGroup")
	if g == nil {
		t.Skip("snmpWarmStartNotificationGroup not found")
		return
	}

	testutil.True(t, g.IsNotificationGroup(),
		"snmpWarmStartNotificationGroup should be a NOTIFICATION-GROUP")
	testutil.Equal(t, 1, len(g.Members()),
		"snmpWarmStartNotificationGroup should have 1 member")

	if len(g.Members()) > 0 {
		testutil.Equal(t, "warmStart", g.Members()[0].Name(),
			"sole member should be warmStart")
	}
}
