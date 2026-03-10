package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func TestGroupCount(t *testing.T) {
	m := loadTestMIB(t)

	groups := m.Groups()
	count := len(groups)

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

func TestGroupConformanceNode(t *testing.T) {
	m := loadTestMIB(t)

	checkConformanceNode(t, m, conformanceNodeSpec{
		name:       "snmpGroup",
		module:     "SNMPv2-MIB",
		kind:       mib.KindGroup,
		status:     mib.StatusCurrent,
		wrongOwner: "IF-MIB",
	},
		func(m *mib.Mib, name string) *mib.Group { return m.Group(name) },
		func(mod *mib.Module, name string) *mib.Group { return mod.Group(name) },
		(*mib.Group).Node,
		(*mib.Group).Module,
		(*mib.Group).OID,
		(*mib.Group).Status,
		(*mib.Group).Description,
		(*mib.Node).Group,
		isNilGroup,
	)
}

func TestGroupIsNotNotificationGroup(t *testing.T) {
	m := loadTestMIB(t)

	g := requireGroup(t, m, "snmpGroup")
	testutil.False(t, g.IsNotificationGroup(),
		"snmpGroup is an OBJECT-GROUP, not NOTIFICATION-GROUP")
}

func TestObjectGroupMembers(t *testing.T) {
	m := loadTestMIB(t)

	g := requireGroup(t, m, "snmpGroup")

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

	g := requireGroup(t, m, "snmpBasicNotificationsGroup")

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

func TestNodeGroupNonGroup(t *testing.T) {
	m := loadTestMIB(t)

	ifIndex := m.Node("ifIndex")
	if ifIndex == nil {
		t.Fatal("ifIndex not found")
	}
	testutil.Nil(t, ifIndex.Group(), "ifIndex should not have a Group()")
}

func TestSmallObjectGroup(t *testing.T) {
	m := loadTestMIB(t)

	// snmpCommunityGroup has 2 members:
	// OBJECTS { snmpInBadCommunityNames, snmpInBadCommunityUses }
	g := m.Group("snmpCommunityGroup")
	testutil.NotNil(t, g, "Group(snmpCommunityGroup)")

	testutil.False(t, g.IsNotificationGroup(),
		"snmpCommunityGroup is an OBJECT-GROUP")
	testutil.Equal(t, 2, len(g.Members()),
		"snmpCommunityGroup should have 2 members")
}

func TestWarmStartNotificationGroup(t *testing.T) {
	m := loadTestMIB(t)

	// snmpWarmStartNotificationGroup NOTIFICATION-GROUP
	// NOTIFICATIONS { warmStart }
	g := m.Group("snmpWarmStartNotificationGroup")
	testutil.NotNil(t, g, "Group(snmpWarmStartNotificationGroup)")

	testutil.True(t, g.IsNotificationGroup(),
		"snmpWarmStartNotificationGroup should be a NOTIFICATION-GROUP")
	testutil.Equal(t, 1, len(g.Members()),
		"snmpWarmStartNotificationGroup should have 1 member")

	if len(g.Members()) > 0 {
		testutil.Equal(t, "warmStart", g.Members()[0].Name(),
			"sole member should be warmStart")
	}
}
