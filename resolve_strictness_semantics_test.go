package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

// resolve_strictness_semantics_test.go covers semantic lookups layered on top
// of symbol resolution, such as AUGMENTS, INDEX, notifications, groups, and
// compliance members.

func TestStrictForwardedAugmentsAndIndexResolution(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-FORWARDING-MIB", mib.ResolverStrict)

	augmentRow := requireObject(t, m, "problemForwardedAugmentEntry")
	augments := augmentRow.Augments()
	testutil.NotNil(t, augments, "AUGMENTS target should resolve through forwarding")
	if augments != nil {
		testutil.Equal(t, "forwardedBaseEntry", augments.Name(),
			"AUGMENTS target name")
		testutil.Equal(t, "PROBLEM-FORWARDING-SOURCE-MIB", augments.Module().Name(),
			"AUGMENTS target should be attributed to source module")
	}

	indexRow := requireObject(t, m, "problemForwardedIndexEntry")
	indexes := indexRow.Index()
	testutil.Len(t, indexes, 1, "forwarded INDEX entries")
	if len(indexes) != 1 {
		return
	}

	idxObj := indexes[0].Object
	testutil.NotNil(t, idxObj, "forwarded index object should resolve")
	if idxObj == nil {
		return
	}

	testutil.Equal(t, "forwardedBaseIndex", idxObj.Name(), "forwarded index name")
	testutil.Equal(t, "PROBLEM-FORWARDING-SOURCE-MIB", idxObj.Module().Name(),
		"forwarded index should be attributed to source module")
	testutil.Equal(t, mib.BaseOctetString, idxObj.Type().EffectiveBase(),
		"forwarded MacAddress index base type")
	testutil.Equal(t, "1x:", idxObj.EffectiveDisplayHint(),
		"forwarded MacAddress display hint")

	sizes := idxObj.EffectiveSizes()
	testutil.Len(t, sizes, 1, "forwarded MacAddress size constraints")
	if len(sizes) == 1 {
		testutil.Equal(t, int64(6), sizes[0].Min, "MacAddress size min")
		testutil.Equal(t, int64(6), sizes[0].Max, "MacAddress size max")
	}
}

func TestStrictForwardedNotificationObjectsResolve(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-FORWARDING-MIB", mib.ResolverStrict)

	notif := requireNotification(t, m, "problemForwardedNotification")
	objects := notif.Objects()
	testutil.Len(t, objects, 2, "forwarded notification objects")
	if len(objects) != 2 {
		return
	}

	testutil.Equal(t, "forwardedSourceObject", objects[0].Name(), "first forwarded notification object")
	testutil.Equal(t, "PROBLEM-FORWARDING-SOURCE-MIB", objects[0].Module().Name(),
		"forwarded notification object should be attributed to source module")
	testutil.Equal(t, "forwardedBaseStatus", objects[1].Name(), "second forwarded notification object")
	testutil.Equal(t, "PROBLEM-FORWARDING-SOURCE-MIB", objects[1].Module().Name(),
		"forwarded notification object should be attributed to source module")

	unresolved := unresolvedSymbols(m, "PROBLEM-FORWARDING-MIB", mib.UnresolvedNotificationObject)
	testutil.Equal(t, 0, len(unresolved), "forwarded notification objects should not be unresolved in strict mode")
}

func TestStrictForwardedGroupAndComplianceMemberResolution(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-FORWARDING-MIB", mib.ResolverStrict)

	objGroup := requireGroup(t, m, "problemForwardedObjectGroup")
	members := objGroup.Members()
	testutil.Len(t, members, 2, "forwarded object group members")
	if len(members) == 2 {
		testutil.Equal(t, "forwardedSourceObject", members[0].Name(), "forwarded object group member 0")
		testutil.Equal(t, "forwardedBaseStatus", members[1].Name(), "forwarded object group member 1")
	}

	notifGroup := requireGroup(t, m, "problemForwardedNotificationGroup")
	testutil.True(t, notifGroup.IsNotificationGroup(), "should be a notification group")
	members = notifGroup.Members()
	testutil.Len(t, members, 1, "forwarded notification group members")
	if len(members) == 1 {
		testutil.Equal(t, "forwardedSourceNotification", members[0].Name(), "forwarded notification group member")
	}

	testutil.Equal(t, 3, countModuleDiagnostics(m, "PROBLEM-FORWARDING-MIB", types.DiagComplianceMemberNotLocal),
		"forwarded group members should resolve in strict mode and then trigger locality diagnostics")
	testutil.Equal(t, 1, countModuleDiagnostics(m, "PROBLEM-FORWARDING-MIB", types.DiagComplianceGroupStatus),
		"forwarded compliance group should resolve in strict mode and trigger status diagnostic")
}

func TestSemanticGlobalLookupStrictnessBoundaries(t *testing.T) {
	levels := []struct {
		name                      string
		level                     mib.ResolverStrictness
		wantNotificationObjects   int
		wantObjectsUnresolved     int
		wantGroupMemberUnresolved int
		wantMemberNotLocal        int
		wantComplianceGroupStatus int
	}{
		{
			name:                      "strict",
			level:                     mib.ResolverStrict,
			wantNotificationObjects:   0,
			wantObjectsUnresolved:     1,
			wantGroupMemberUnresolved: 1,
			wantMemberNotLocal:        1,
			wantComplianceGroupStatus: 0,
		},
		{
			name:                      "normal",
			level:                     mib.ResolverNormal,
			wantNotificationObjects:   0,
			wantObjectsUnresolved:     1,
			wantGroupMemberUnresolved: 1,
			wantMemberNotLocal:        1,
			wantComplianceGroupStatus: 0,
		},
		{
			name:                      "permissive",
			level:                     mib.ResolverPermissive,
			wantNotificationObjects:   1,
			wantObjectsUnresolved:     0,
			wantGroupMemberUnresolved: 0,
			wantMemberNotLocal:        1,
			wantComplianceGroupStatus: 1,
		},
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadAtStrictness(t, "PROBLEM-SEMANTIC-GLOBAL-MIB", lvl.level)

			notif := m.Notification("problemGlobalNotification")
			testutil.NotNil(t, notif, "semantic global notification should resolve")
			if notif != nil {
				testutil.Len(t, notif.Objects(), lvl.wantNotificationObjects,
					"notification object count at %s", lvl.name)
			}

			testutil.Equal(t, lvl.wantObjectsUnresolved,
				countDiagnostics(m, types.DiagObjectsUnresolved),
				"notification object unresolved count at %s", lvl.name)
			testutil.Equal(t, lvl.wantGroupMemberUnresolved,
				countDiagnostics(m, types.DiagGroupMemberUnresolved),
				"group member unresolved count at %s", lvl.name)
			testutil.Equal(t, lvl.wantMemberNotLocal,
				countDiagnostics(m, types.DiagComplianceMemberNotLocal),
				"group member locality count at %s", lvl.name)
			testutil.Equal(t, lvl.wantComplianceGroupStatus,
				countDiagnostics(m, types.DiagComplianceGroupStatus),
				"compliance group status count at %s", lvl.name)
		})
	}
}

func TestStrictModeIndexResolution(t *testing.T) {
	entries := []string{
		"rlPortGvrpTimersEntry",
		"rlStormCtrlEntry",
	}

	for _, lvl := range allStrictnessCases {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadAtStrictness(t, "RADLAN-MIB", lvl.level)

			for _, entryName := range entries {
				t.Run(entryName, func(t *testing.T) {
					obj := m.Object(entryName)
					testutil.NotNil(t, obj, "object %s should exist", entryName)
					if obj == nil {
						return
					}
					idx := obj.Index()
					testutil.True(t, len(idx) > 0,
						"INDEX should not be empty for %s at %s", entryName, lvl.name)
					if len(idx) > 0 {
						testutil.Equal(t, "dot1dBasePort", idx[0].Object.Name(),
							"first INDEX object should be dot1dBasePort")
					}
				})
			}
		})
	}
}

func TestRealCorpusStrictSemanticResolution(t *testing.T) {
	t.Run("HUAWEI imported augments stays deterministic", func(t *testing.T) {
		m := loadAtStrictness(t, "HUAWEI-DISMAN-PING-MIB", mib.ResolverStrict)

		obj := requireObject(t, m, "hwPingCtlEntry")
		augments := obj.Augments()
		testutil.NotNil(t, augments, "AUGMENTS target should resolve in strict mode")
		if augments != nil {
			testutil.Equal(t, "pingCtlEntry", augments.Name(), "AUGMENTS target name")
			testutil.Equal(t, "DISMAN-PING-MIB", augments.Module().Name(),
				"AUGMENTS target should stay attributed to imported module")
		}
	})

	t.Run("TIMETRA compliance members stay local in strict mode", func(t *testing.T) {
		m := loadAtStrictness(t, "TIMETRA-MPLS-MIB", mib.ResolverStrict)

		comp := m.Compliance("tmnxMplsV22v0Compliance")
		testutil.NotNil(t, comp, "Compliance(tmnxMplsV22v0Compliance)")
		if comp == nil {
			return
		}

		modules := comp.Modules()
		testutil.Len(t, modules, 1, "compliance modules")
		if len(modules) != 1 {
			return
		}

		testutil.Equal(t, 4, len(modules[0].MandatoryGroups),
			"mandatory group count")
		testutil.Equal(t, 0, countModuleDiagnostics(m, "TIMETRA-MPLS-MIB", types.DiagGroupMemberUnresolved),
			"group members should resolve in strict mode")
		testutil.Equal(t, 0, countModuleDiagnostics(m, "TIMETRA-MPLS-MIB", types.DiagComplianceGroupStatus),
			"resolved local groups should not need permissive recovery")

		group := requireGroup(t, m, "tmnxMplsLspRateCountersGroup")
		members := group.Members()
		testutil.Len(t, members, 4, "group should retain all strict-mode members")
		if len(members) == 4 {
			testutil.Equal(t, "vRtrMplsLspStatsAggregateOnly", members[0].Name(),
				"first strict-mode group member")
			testutil.Equal(t, "vRtrMplsLspStatsRateMbps", members[3].Name(),
				"last strict-mode group member")
		}
	})
}
