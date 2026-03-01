package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func TestResolveNodeType(t *testing.T) {
	forEachFixtureNode(t, nil, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		node := m.Node(fn.Name)
		if node == nil {
			t.Fatalf("divergence: gomib cannot find node %q (fixture OID %s)", fn.Name, fn.OID)
		}
		gomibNodeType := normalizeNodeType(node)
		if !typesEquivalent(gomibNodeType, fn.NodeType) {
			t.Errorf("divergence: node type for %s: gomib=%q fixture=%q",
				fn.Name, gomibNodeType, fn.NodeType)
		}
	})
}

func TestResolveOIDs(t *testing.T) {
	forEachFixtureNode(t, nil, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		node := m.Node(fn.Name)
		if node == nil {
			t.Fatalf("divergence: gomib cannot find node %q (fixture OID %s)", fn.Name, fn.OID)
		}
		gotOID := node.OID().String()
		if gotOID != fn.OID {
			t.Errorf("divergence: OID for %s: gomib=%s fixture=%s", fn.Name, gotOID, fn.OID)
		}
	})
}

func TestResolveTypes(t *testing.T) {
	forEachFixtureNode(t, isObjectTypeNode, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		obj := requireFixtureObject(t, m, fn)

		gomibType := normalizeType(obj.Type())
		if !typesEquivalent(gomibType, fn.Type) {
			t.Errorf("divergence: type for %s: gomib=%q fixture=%q",
				fn.Name, gomibType, fn.Type)
		}

		gomibTC := ""
		if t := obj.Type(); t != nil && t.IsTextualConvention() {
			gomibTC = t.Name()
		}
		if fn.TCName != "" || gomibTC != "" {
			if gomibTC != fn.TCName {
				t.Errorf("divergence: TC name for %s: gomib=%q fixture=%q",
					fn.Name, gomibTC, fn.TCName)
			}
		}

		gomibHint := obj.EffectiveDisplayHint()
		if fn.Hint != "" || gomibHint != "" {
			if !hintsEquivalent(gomibHint, fn.Hint) {
				t.Errorf("divergence: display hint for %s: gomib=%q fixture=%q",
					fn.Name, gomibHint, fn.Hint)
			}
		}
	})
}

func TestResolveEnums(t *testing.T) {
	forEachFixtureNode(t, func(fn *testutil.FixtureNode) bool {
		return isObjectTypeNode(fn) && len(fn.EnumValues) > 0
	}, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		obj := requireFixtureObject(t, m, fn)

		gomibEnums := normalizeEnums(obj.EffectiveEnums())
		if !enumsEquivalent(gomibEnums, fn.EnumValues) {
			t.Errorf("divergence: enums for %s:\n  gomib=%s\n  fixture=%s",
				fn.Name,
				formatEnums(gomibEnums),
				formatEnums(fn.EnumValues))
		}
	})
}

func TestResolveBits(t *testing.T) {
	forEachFixtureNode(t, func(fn *testutil.FixtureNode) bool {
		return isObjectTypeNode(fn) && len(fn.BitValues) > 0
	}, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		obj := requireFixtureObject(t, m, fn)

		gomibBits := normalizeEnums(obj.EffectiveBits())
		if !enumsEquivalent(gomibBits, fn.BitValues) {
			t.Errorf("divergence: bits for %s:\n  gomib=%s\n  fixture=%s",
				fn.Name,
				formatEnums(gomibBits),
				formatEnums(fn.BitValues))
		}
	})
}

func TestResolveTables(t *testing.T) {
	forEachFixtureNode(t, func(fn *testutil.FixtureNode) bool {
		return fn.Kind != "" || len(fn.Indexes) > 0 || fn.Augments != ""
	}, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		obj := requireFixtureObject(t, m, fn)

		if fn.Kind != "" {
			gomibKind := normalizeKind(obj.Kind())
			if gomibKind != fn.Kind {
				t.Errorf("divergence: kind for %s: gomib=%q fixture=%q",
					fn.Name, gomibKind, fn.Kind)
			}
		}

		if len(fn.Indexes) > 0 {
			gomibIndexes := normalizeIndexes(obj.Index())
			if !indexesEquivalent(gomibIndexes, fn.Indexes) {
				t.Errorf("divergence: indexes for %s:\n  gomib=%s\n  fixture=%s",
					fn.Name,
					formatIndexes(gomibIndexes),
					formatIndexes(fn.Indexes))
			}
		}

		if fn.Augments != "" {
			gomibAugments := ""
			if aug := obj.Augments(); aug != nil {
				gomibAugments = aug.Name()
			}
			if gomibAugments != fn.Augments {
				t.Errorf("divergence: augments for %s: gomib=%q fixture=%q",
					fn.Name, gomibAugments, fn.Augments)
			}
		}
	})
}

func TestResolveAccess(t *testing.T) {
	forEachFixtureNode(t, func(fn *testutil.FixtureNode) bool {
		return fn.Access != ""
	}, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		obj := requireFixtureObject(t, m, fn)

		gomibAccess := normalizeAccess(obj.Access())
		isSMIv1 := obj.Module().Language() == mib.LanguageSMIv1
		if !accessEquivalent(gomibAccess, fn.Access, isSMIv1) {
			t.Errorf("divergence: access for %s: gomib=%q fixture=%q",
				fn.Name, gomibAccess, fn.Access)
		}
	})
}

func TestResolveStatus(t *testing.T) {
	forEachFixtureNode(t, func(fn *testutil.FixtureNode) bool {
		return fn.Status != ""
	}, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		gomibStatus := ""
		if obj := m.Object(fn.Name); obj != nil {
			gomibStatus = normalizeStatus(obj.Status())
		} else if notif := m.Notification(fn.Name); notif != nil {
			gomibStatus = normalizeStatus(notif.Status())
		} else {
			t.Fatalf("divergence: gomib does not have node %q", fn.Name)
		}

		if !statusEquivalent(gomibStatus, fn.Status) {
			t.Errorf("divergence: status for %s: gomib=%q fixture=%q",
				fn.Name, gomibStatus, fn.Status)
		}
	})
}

func TestResolveRanges(t *testing.T) {
	forEachFixtureNode(t, func(fn *testutil.FixtureNode) bool {
		return isObjectTypeNode(fn) && len(fn.Ranges) > 0
	}, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		obj := requireFixtureObject(t, m, fn)

		var gomibRanges []testutil.RangeInfo
		gomibRanges = append(gomibRanges, normalizeRanges(obj.EffectiveRanges())...)
		gomibRanges = append(gomibRanges, normalizeRanges(obj.EffectiveSizes())...)

		if !rangesEquivalent(gomibRanges, fn.Ranges) {
			t.Errorf("divergence: ranges for %s:\n  gomib=%s\n  fixture=%s",
				fn.Name,
				formatRanges(gomibRanges),
				formatRanges(fn.Ranges))
		}
	})
}

func TestResolveNotifications(t *testing.T) {
	forEachFixtureNode(t, isNotificationNode, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		notif := m.Notification(fn.Name)
		if notif == nil {
			t.Fatalf("divergence: gomib does not have notification %q", fn.Name)
		}

		gotOID := notif.OID().String()
		if gotOID != fn.OID {
			t.Errorf("divergence: OID for notification %s: gomib=%s fixture=%s",
				fn.Name, gotOID, fn.OID)
		}

		if len(fn.Varbinds) > 0 {
			gomibVarbinds := normalizeVarbinds(notif.Objects())
			if !varbindsEquivalent(gomibVarbinds, fn.Varbinds) {
				t.Errorf("divergence: varbinds for %s:\n  gomib=%v\n  fixture=%v",
					fn.Name, gomibVarbinds, fn.Varbinds)
			}
		}

		if fn.Status != "" {
			gomibStatus := normalizeStatus(notif.Status())
			if !statusEquivalent(gomibStatus, fn.Status) {
				t.Errorf("divergence: status for notification %s: gomib=%q fixture=%q",
					fn.Name, gomibStatus, fn.Status)
			}
		}
	})
}

func TestResolveUnits(t *testing.T) {
	forEachFixtureNode(t, func(fn *testutil.FixtureNode) bool {
		return isObjectTypeNode(fn) && fn.Units != ""
	}, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		obj := requireFixtureObject(t, m, fn)

		gomibUnits := obj.Units()
		if gomibUnits != fn.Units {
			t.Errorf("divergence: units for %s: gomib=%q fixture=%q",
				fn.Name, gomibUnits, fn.Units)
		}
	})
}

func TestResolveDefval(t *testing.T) {
	forEachFixtureNode(t, isObjectTypeNode, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		obj := requireFixtureObject(t, m, fn)

		dv := obj.DefaultValue()
		gomibDefval := ""
		if !dv.IsZero() {
			gomibDefval = dv.String()
		}

		if fn.DefaultValue == "" && gomibDefval == "" {
			return
		}
		if fn.DefaultValue != "" && gomibDefval == "" {
			t.Errorf("divergence: defval for %s: gomib has no defval, fixture=%q",
				fn.Name, fn.DefaultValue)
			return
		}
		if fn.DefaultValue == "" && gomibDefval != "" {
			t.Errorf("divergence: defval for %s: gomib=%q, fixture has no defval",
				fn.Name, gomibDefval)
			return
		}
		if !defvalEquivalent(gomibDefval, fn.DefaultValue) {
			t.Errorf("divergence: defval for %s: gomib=%q fixture=%q",
				fn.Name, gomibDefval, fn.DefaultValue)
		}
	})
}

func TestResolveReference(t *testing.T) {
	forEachFixtureNode(t, func(fn *testutil.FixtureNode) bool {
		return fn.Reference != ""
	}, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		gomibRef := ""
		if obj := m.Object(fn.Name); obj != nil {
			gomibRef = obj.Reference()
		} else if notif := m.Notification(fn.Name); notif != nil {
			gomibRef = notif.Reference()
		} else {
			t.Fatalf("divergence: gomib does not have node %q", fn.Name)
		}

		if !referenceEquivalent(gomibRef, fn.Reference) {
			t.Errorf("divergence: reference for %s: gomib=%q fixture=%q",
				fn.Name, gomibRef, fn.Reference)
		}
	})
}

func TestResolveModule(t *testing.T) {
	forEachFixtureNode(t, func(fn *testutil.FixtureNode) bool {
		return fn.Module != ""
	}, func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) {
		node := m.Node(fn.Name)
		if node == nil {
			t.Fatalf("divergence: gomib cannot find node %q", fn.Name)
		}

		gomibModule := ""
		if node.Module() != nil {
			gomibModule = node.Module().Name()
		}

		if gomibModule != fn.Module {
			t.Errorf("divergence: module for %s: gomib=%q fixture=%q",
				fn.Name, gomibModule, fn.Module)
		}
	})
}
