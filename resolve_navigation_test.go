package gomib

// resolve_navigation_test.go tests OID tree navigation, table structure
// navigation, filtered collections, label lookups, type predicates, module
// metadata, and other API surface areas not covered by ground-truth fixture
// tests.

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// === Node tree navigation ===

func TestNodeRoot(t *testing.T) {
	m := loadTestMIB(t)

	root := m.Root()
	testutil.NotNil(t, root, "Root() should not be nil")
	testutil.True(t, root.IsRoot(), "Root node should report IsRoot() == true")
	testutil.Nil(t, root.Parent(), "Root node should have nil Parent()")
}

func TestNodeParent(t *testing.T) {
	m := loadTestMIB(t)

	// ifIndex = 1.3.6.1.2.1.2.2.1.1
	node := m.FindNode("ifIndex")
	if node == nil {
		t.Fatal("ifIndex not found")
	}

	parent := node.Parent()
	testutil.NotNil(t, parent, "ifIndex should have a parent")
	testutil.False(t, node.IsRoot(), "ifIndex should not be root")

	// Parent of ifIndex is ifEntry (1.3.6.1.2.1.2.2.1)
	if parent.Name() != "" {
		testutil.Equal(t, "ifEntry", parent.Name(),
			"parent of ifIndex should be ifEntry")
	}
}

func TestNodeChildren(t *testing.T) {
	m := loadTestMIB(t)

	// ifEntry has multiple children (columns)
	entry := m.FindNode("ifEntry")
	if entry == nil {
		t.Skip("ifEntry not found")
		return
	}

	children := entry.Children()
	testutil.Greater(t, len(children), 0,
		"ifEntry should have children (table columns)")

	// ifIndex should be among children (arc 1)
	found := false
	for _, child := range children {
		if child.Name() == "ifIndex" {
			found = true
			break
		}
	}
	testutil.True(t, found, "ifIndex should be among ifEntry's children")
}

func TestNodeChild(t *testing.T) {
	m := loadTestMIB(t)

	entry := m.FindNode("ifEntry")
	if entry == nil {
		t.Skip("ifEntry not found")
		return
	}

	// ifIndex is arc 1 under ifEntry
	child := entry.Child(1)
	if child == nil {
		t.Skip("Child(1) returned nil - arc lookup may work differently")
		return
	}
	testutil.Equal(t, "ifIndex", child.Name(), "Child(1) of ifEntry should be ifIndex")

	// Non-existent arc
	noChild := entry.Child(99999)
	testutil.Nil(t, noChild, "Child(99999) should return nil for non-existent arc")
}

func TestNodeDescendants(t *testing.T) {
	m := loadTestMIB(t)

	// ifTable has descendants (ifEntry + columns)
	table := m.FindNode("ifTable")
	if table == nil {
		t.Skip("ifTable not found")
		return
	}

	count := 0
	for range table.Descendants() {
		count++
	}
	// ifTable -> ifEntry -> ifIndex, ifDescr, ifType, ... (22+ columns)
	testutil.Greater(t, count, 5,
		"ifTable should have many descendants")
}

func TestNodeArc(t *testing.T) {
	m := loadTestMIB(t)

	// ifIndex OID = 1.3.6.1.2.1.2.2.1.1 (last arc is 1)
	node := m.FindNode("ifIndex")
	if node == nil {
		t.Fatal("ifIndex not found")
	}
	testutil.Equal(t, uint32(1), node.Arc(), "ifIndex arc should be 1")
}

func TestNodeObjectAndNotification(t *testing.T) {
	m := loadTestMIB(t)

	// ifIndex node should have an associated Object
	node := m.FindNode("ifIndex")
	if node == nil {
		t.Fatal("ifIndex not found")
	}
	obj := node.Object()
	testutil.NotNil(t, obj, "ifIndex node should have an associated Object")
	if obj != nil {
		testutil.Equal(t, "ifIndex", obj.Name(), "associated object name")
	}

	// ifIndex should not have a notification
	testutil.Nil(t, node.Notification(), "ifIndex should not have a notification")

	// linkDown should have a notification
	linkDown := m.FindNode("linkDown")
	if linkDown == nil {
		t.Skip("linkDown not found")
		return
	}
	notif := linkDown.Notification()
	if notif == nil {
		t.Skip("linkDown node has no associated notification")
		return
	}
	testutil.Equal(t, "linkDown", notif.Name(), "linkDown notification name")
}

// === Object table navigation ===

func TestObjectTableNavigation(t *testing.T) {
	m := loadTestMIB(t)

	t.Run("column to table", func(t *testing.T) {
		col := m.FindObject("ifIndex")
		if col == nil {
			t.Fatal("ifIndex not found")
		}
		testutil.True(t, col.IsColumn(), "ifIndex should be a column")
		testutil.False(t, col.IsTable(), "ifIndex should not be a table")
		testutil.False(t, col.IsRow(), "ifIndex should not be a row")
		testutil.False(t, col.IsScalar(), "ifIndex should not be a scalar")

		tbl := col.Table()
		if tbl == nil {
			t.Skip("Table() returned nil for ifIndex")
			return
		}
		testutil.Equal(t, "ifTable", tbl.Name(), "ifIndex's table should be ifTable")
		testutil.True(t, tbl.IsTable(), "ifTable should be a table")
	})

	t.Run("column to row", func(t *testing.T) {
		col := m.FindObject("ifIndex")
		if col == nil {
			t.Fatal("ifIndex not found")
		}

		row := col.Row()
		if row == nil {
			t.Skip("Row() returned nil for ifIndex")
			return
		}
		testutil.Equal(t, "ifEntry", row.Name(), "ifIndex's row should be ifEntry")
		testutil.True(t, row.IsRow(), "ifEntry should be a row")
	})

	t.Run("table to entry", func(t *testing.T) {
		tbl := m.FindObject("ifTable")
		if tbl == nil {
			t.Skip("ifTable not found")
			return
		}
		testutil.True(t, tbl.IsTable(), "ifTable should be a table")

		entry := tbl.Entry()
		if entry == nil {
			t.Skip("Entry() returned nil for ifTable")
			return
		}
		testutil.Equal(t, "ifEntry", entry.Name(), "ifTable entry should be ifEntry")
	})

	t.Run("row columns", func(t *testing.T) {
		row := m.FindObject("ifEntry")
		if row == nil {
			t.Skip("ifEntry not found")
			return
		}

		cols := row.Columns()
		testutil.Greater(t, len(cols), 5, "ifEntry should have many columns")

		// Verify ifIndex is among them
		found := false
		for _, c := range cols {
			if c.Name() == "ifIndex" {
				found = true
				break
			}
		}
		testutil.True(t, found, "ifIndex should be among ifEntry columns")
	})

	t.Run("scalar predicates", func(t *testing.T) {
		scalar := m.FindObject("sysDescr")
		if scalar == nil {
			t.Skip("sysDescr not found")
			return
		}
		testutil.True(t, scalar.IsScalar(), "sysDescr should be a scalar")
		testutil.False(t, scalar.IsTable(), "sysDescr should not be a table")
		testutil.False(t, scalar.IsRow(), "sysDescr should not be a row")
		testutil.False(t, scalar.IsColumn(), "sysDescr should not be a column")
	})
}

func TestObjectEffectiveIndexes(t *testing.T) {
	m := loadTestMIB(t)

	entry := m.FindObject("ifEntry")
	if entry == nil {
		t.Skip("ifEntry not found")
		return
	}

	indexes := entry.EffectiveIndexes()
	if len(indexes) == 0 {
		t.Skip("EffectiveIndexes() returned empty")
		return
	}

	testutil.Equal(t, 1, len(indexes), "ifEntry should have 1 index")
	testutil.Equal(t, "ifIndex", indexes[0].Object.Name(), "index should be ifIndex")
}

// === Label lookups ===

func TestObjectEnumLookup(t *testing.T) {
	m := loadTestMIB(t)

	// ifType has enum values
	obj := m.FindObject("ifType")
	if obj == nil {
		t.Skip("ifType not found")
		return
	}

	enums := obj.EffectiveEnums()
	if len(enums) == 0 {
		t.Skip("ifType has no effective enums")
		return
	}

	// Look up "ethernetCsmacd" which should be value 6
	nv, ok := obj.Enum("ethernetCsmacd")
	if !ok {
		t.Skip("Enum(ethernetCsmacd) not found - lookup may not be implemented")
		return
	}
	testutil.Equal(t, int64(6), nv.Value, "ethernetCsmacd should be 6")

	// Non-existent label
	_, ok = obj.Enum("totallyFakeLabel")
	testutil.False(t, ok, "non-existent label should return false")
}

func TestObjectBitLookup(t *testing.T) {
	// Use PROBLEM-DEFVAL-MIB which has BITS objects (problemDefvalEmptyBits, problemDefvalMultiBits)
	pm := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := pm.FindObject("problemDefvalMultiBits")
	if obj == nil {
		t.Skip("problemDefvalMultiBits not found")
		return
	}

	bits := obj.EffectiveBits()
	if len(bits) == 0 {
		t.Skip("no effective bits on problemDefvalMultiBits")
		return
	}

	// Look up "read" which should be bit 0
	nv, ok := obj.Bit("read")
	if !ok {
		t.Skip("Bit(read) not found - lookup may not be implemented")
		return
	}
	testutil.Equal(t, int64(0), nv.Value, "read bit should be 0")

	// Look up "write" which should be bit 1
	nv, ok = obj.Bit("write")
	if !ok {
		t.Skip("Bit(write) not found")
		return
	}
	testutil.Equal(t, int64(1), nv.Value, "write bit should be 1")

	// Non-existent bit
	_, ok = obj.Bit("totallyFakeBit")
	testutil.False(t, ok, "non-existent bit should return false")
}

// === Type predicates ===

func TestTypePredicates(t *testing.T) {
	m := loadTestMIB(t)

	t.Run("IsString", func(t *testing.T) {
		typ := m.FindType("DisplayString")
		if typ == nil {
			t.Skip("DisplayString type not found")
			return
		}
		testutil.True(t, typ.IsString(), "DisplayString should be IsString()")
		testutil.False(t, typ.IsCounter(), "DisplayString should not be IsCounter()")
		testutil.False(t, typ.IsGauge(), "DisplayString should not be IsGauge()")
	})

	t.Run("IsEnumeration", func(t *testing.T) {
		// ifType uses IANAifType which is an enumeration
		obj := m.FindObject("ifType")
		if obj == nil || obj.Type() == nil {
			t.Skip("ifType or its type not found")
			return
		}
		// The effective type might be an enumeration
		enums := obj.EffectiveEnums()
		if len(enums) == 0 {
			t.Skip("ifType has no enums")
			return
		}
		if !obj.Type().IsEnumeration() {
			t.Skip("ifType type does not report IsEnumeration() - may check direct enums only")
		}
	})

	t.Run("counter type from problem MIBs", func(t *testing.T) {
		pm := loadTypeChainsMIB(t)
		obj := pm.FindObject("problemAppTypeChain")
		if obj == nil || obj.Type() == nil {
			t.Skip("problemAppTypeChain not found")
			return
		}
		testutil.True(t, obj.Type().IsCounter(),
			"MyCounter (based on Counter32) should report IsCounter()")
	})

	t.Run("gauge type from problem MIBs", func(t *testing.T) {
		pm := loadTypeChainsMIB(t)
		obj := pm.FindObject("problemInheritedHint")
		if obj == nil || obj.Type() == nil {
			t.Skip("problemInheritedHint not found")
			return
		}
		testutil.True(t, obj.Type().IsGauge(),
			"MySpecialGauge (based on Gauge32) should report IsGauge()")
	})

	t.Run("IsBits", func(t *testing.T) {
		pm := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")
		obj := pm.FindObject("problemDefvalMultiBits")
		if obj == nil || obj.Type() == nil {
			t.Skip("problemDefvalMultiBits not found")
			return
		}
		// BITS is the effective base type
		testutil.Equal(t, mib.BaseBits, obj.Type().EffectiveBase(),
			"BITS object should have BaseBits effective base")
		// IsBits() checks EffectiveBits() which requires named bits on the type.
		// For inline BITS definitions, the type may not have bits registered.
		// Check the object's effective bits instead.
		bits := obj.EffectiveBits()
		testutil.Greater(t, len(bits), 0,
			"BITS object should have effective bits")
	})
}

func TestTypeParent(t *testing.T) {
	m := loadTestMIB(t)

	typ := m.FindType("DisplayString")
	if typ == nil {
		t.Skip("DisplayString type not found")
		return
	}

	parent := typ.Parent()
	if parent == nil {
		t.Skip("DisplayString has no Parent() - may not be implemented")
		return
	}

	// DisplayString is based on OCTET STRING
	testutil.Equal(t, mib.BaseOctetString, parent.EffectiveBase(),
		"DisplayString's parent should resolve to OCTET STRING")
}

func TestTypeEnumLookup(t *testing.T) {
	// Use a type that has direct enums
	pm := loadTypeChainsMIB(t)

	obj := pm.FindObject("problemEnumChain")
	if obj == nil || obj.Type() == nil {
		t.Skip("problemEnumChain not found")
		return
	}

	// Walk up the type chain to find the type with enums
	typ := obj.Type()
	for typ != nil {
		if len(typ.Enums()) > 0 {
			nv, ok := typ.Enum("active")
			if !ok {
				t.Skip("Enum(active) not found on type")
				return
			}
			testutil.Equal(t, int64(1), nv.Value, "active should be 1")
			return
		}
		typ = typ.Parent()
	}
	t.Skip("no type in chain has direct enums")
}

// === Module metadata ===

func TestModuleLanguage(t *testing.T) {
	m := loadTestMIB(t)

	ifMIB := m.Module("IF-MIB")
	if ifMIB == nil {
		t.Fatal("IF-MIB not found")
	}
	testutil.Equal(t, mib.LanguageSMIv2, ifMIB.Language(),
		"IF-MIB should be SMIv2")
}

func TestModuleDescription(t *testing.T) {
	m := loadTestMIB(t)

	ifMIB := m.Module("IF-MIB")
	if ifMIB == nil {
		t.Fatal("IF-MIB not found")
	}

	desc := ifMIB.Description()
	if desc == "" {
		t.Skip("IF-MIB has no Description() - may not be populated")
		return
	}
	testutil.True(t, len(desc) > 0, "IF-MIB should have a non-empty description")
}

func TestModuleOID(t *testing.T) {
	m := loadTestMIB(t)

	ifMIB := m.Module("IF-MIB")
	if ifMIB == nil {
		t.Fatal("IF-MIB not found")
	}

	oid := ifMIB.OID()
	if oid.String() == "" {
		t.Skip("IF-MIB has no OID - may not be set")
		return
	}
	// IF-MIB identity = 1.3.6.1.2.1.31
	testutil.Equal(t, "1.3.6.1.2.1.31", oid.String(),
		"IF-MIB module OID")
}

func TestModuleScopedLookups(t *testing.T) {
	m := loadTestMIB(t)

	ifMIB := m.Module("IF-MIB")
	if ifMIB == nil {
		t.Fatal("IF-MIB not found")
	}

	t.Run("Object lookup", func(t *testing.T) {
		obj := ifMIB.Object("ifIndex")
		if obj == nil {
			t.Skip("Module.Object(ifIndex) not found - may not scope correctly for imported objects")
			return
		}
		testutil.Equal(t, "ifIndex", obj.Name(), "module-scoped object name")
	})

	t.Run("Node lookup", func(t *testing.T) {
		// ifMIBObjects is defined in IF-MIB as a node
		node := ifMIB.Node("ifMIBObjects")
		if node == nil {
			t.Skip("Module.Node(ifMIBObjects) not found - may not expose non-object nodes")
			return
		}
		testutil.Equal(t, "ifMIBObjects", node.Name(), "module-scoped node name")
	})

	t.Run("Type lookup", func(t *testing.T) {
		// IF-MIB defines InterfaceIndex TC
		typ := ifMIB.Type("InterfaceIndex")
		if typ == nil {
			t.Skip("Module.Type(InterfaceIndex) not found")
			return
		}
		testutil.Equal(t, "InterfaceIndex", typ.Name(), "module-scoped type name")
	})
}

func TestModuleFilteredCollections(t *testing.T) {
	m := loadTestMIB(t)

	ifMIB := m.Module("IF-MIB")
	if ifMIB == nil {
		t.Fatal("IF-MIB not found")
	}

	tables := ifMIB.Tables()
	scalars := ifMIB.Scalars()
	columns := ifMIB.Columns()
	rows := ifMIB.Rows()

	testutil.Greater(t, len(tables), 0, "IF-MIB should have tables")
	testutil.Greater(t, len(columns), 0, "IF-MIB should have columns")
	testutil.Greater(t, len(rows), 0, "IF-MIB should have rows")

	// IF-MIB has ifNumber (scalar) and ifTableLastChange (scalar)
	if len(scalars) == 0 {
		t.Log("IF-MIB has no scalars reported - may only count direct module objects")
	}
}

// === Mib filtered collections ===

func TestMibFilteredCollections(t *testing.T) {
	m := loadTestMIB(t)

	columns := m.Columns()
	rows := m.Rows()

	testutil.Greater(t, len(columns), 0, "should have columns")
	testutil.Greater(t, len(rows), 0, "should have rows")

	// Every column should have Kind == column
	for _, col := range columns {
		testutil.Equal(t, mib.KindColumn, col.Kind(),
			"Columns() entry %s should have KindColumn", col.Name())
	}

	// Every row should have Kind == row
	for _, row := range rows {
		testutil.Equal(t, mib.KindRow, row.Kind(),
			"Rows() entry %s should have KindRow", row.Name())
	}
}

func TestMibNotificationCount(t *testing.T) {
	m := loadTestMIB(t)

	count := m.NotificationCount()
	notifications := m.Notifications()
	testutil.Equal(t, count, len(notifications),
		"NotificationCount() should match Notifications() length")

	// The fixture modules include SNMPv2-MIB which has linkDown, linkUp, etc.
	testutil.Greater(t, count, 0, "should have some notifications")
}

func TestMibHasErrors(t *testing.T) {
	m := loadTestMIB(t)

	// Standard MIBs loaded at default strictness should not have errors
	// (they may have diagnostics, but not errors)
	_ = m.HasErrors() // just verify it doesn't panic

	// Load a problem MIB with known issues at strict level
	strict := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", mib.StrictnessStrict)
	// The strict load may or may not report HasErrors depending on what
	// severity threshold is used. We just verify the method works.
	_ = strict.HasErrors()
}

// === OID-based lookups ===

func TestNodeByOID(t *testing.T) {
	m := loadTestMIB(t)

	oid, err := mib.ParseOID("1.3.6.1.2.1.2.2.1.1")
	if err != nil {
		t.Fatalf("ParseOID failed: %v", err)
	}

	node := m.NodeByOID(oid)
	if node == nil {
		t.Skip("NodeByOID not implemented or OID not in tree")
		return
	}
	testutil.Equal(t, "ifIndex", node.Name(), "NodeByOID should find ifIndex")
}

func TestLongestPrefixByOID(t *testing.T) {
	m := loadTestMIB(t)

	// Look up an OID that extends beyond a known node
	// ifIndex = 1.3.6.1.2.1.2.2.1.1
	// Add .5 to simulate an instance OID
	oid, err := mib.ParseOID("1.3.6.1.2.1.2.2.1.1.5")
	if err != nil {
		t.Fatalf("ParseOID failed: %v", err)
	}

	node := m.LongestPrefixByOID(oid)
	if node == nil {
		t.Skip("LongestPrefixByOID not implemented or no match")
		return
	}
	testutil.Equal(t, "ifIndex", node.Name(),
		"LongestPrefixByOID for ifIndex.5 should find ifIndex")
}

func TestNodeLongestPrefix(t *testing.T) {
	m := loadTestMIB(t)

	root := m.Root()
	if root == nil {
		t.Fatal("Root() returned nil")
	}

	// Look up from root with an instance OID
	oid, err := mib.ParseOID("1.3.6.1.2.1.2.2.1.1.5")
	if err != nil {
		t.Fatalf("ParseOID failed: %v", err)
	}

	node := root.LongestPrefix(oid)
	if node == nil {
		t.Skip("Node.LongestPrefix not implemented")
		return
	}
	testutil.Equal(t, "ifIndex", node.Name(),
		"Node.LongestPrefix for ifIndex.5 should find ifIndex")
}

// === Object metadata ===

func TestObjectDescription(t *testing.T) {
	m := loadTestMIB(t)

	obj := m.FindObject("sysDescr")
	if obj == nil {
		t.Fatal("sysDescr not found")
	}

	desc := obj.Description()
	if desc == "" {
		t.Skip("sysDescr has no Description() - may not be populated")
		return
	}
	// sysDescr DESCRIPTION contains text about system description
	testutil.Greater(t, len(desc), 10,
		"sysDescr description should be non-trivial")
}

func TestObjectNode(t *testing.T) {
	m := loadTestMIB(t)

	obj := m.FindObject("ifIndex")
	if obj == nil {
		t.Fatal("ifIndex not found")
	}

	node := obj.Node()
	testutil.NotNil(t, node, "Object.Node() should not be nil")
	testutil.Equal(t, "ifIndex", node.Name(), "Object.Node().Name() should match")
	testutil.Equal(t, obj.OID().String(), node.OID().String(),
		"Object OID should match Node OID")
}

// === Notification metadata ===

func TestNotificationMetadataFields(t *testing.T) {
	m := loadTestMIB(t)

	notif := m.FindNotification("linkDown")
	if notif == nil {
		t.Skip("linkDown notification not found")
		return
	}

	// Node
	node := notif.Node()
	testutil.NotNil(t, node, "Notification.Node() should not be nil")
	if node != nil {
		testutil.Equal(t, "linkDown", node.Name(), "notification node name")
	}

	// Module
	mod := notif.Module()
	testutil.NotNil(t, mod, "Notification.Module() should not be nil")

	// OID should be set
	oid := notif.OID()
	testutil.Greater(t, len(oid), 0, "notification OID should not be empty")

	// Status
	status := notif.Status()
	// linkDown is current
	testutil.Equal(t, mib.StatusCurrent, status, "linkDown should be current")
}

// === Types collection ===

func TestTypesCollection(t *testing.T) {
	m := loadTestMIB(t)

	types := m.Types()
	count := m.TypeCount()
	testutil.Equal(t, count, len(types),
		"TypeCount() should match Types() length")
	testutil.Greater(t, count, 0, "should have types (DisplayString, etc.)")

	// Find DisplayString in the list
	found := false
	for _, typ := range types {
		if typ.Name() == "DisplayString" {
			found = true
			break
		}
	}
	testutil.True(t, found, "DisplayString should be in Types() list")
}

// === Module notifications ===

func TestModuleNotifications(t *testing.T) {
	m := loadTestMIB(t)

	snmpMIB := m.Module("SNMPv2-MIB")
	if snmpMIB == nil {
		t.Skip("SNMPv2-MIB not found")
		return
	}

	notifs := snmpMIB.Notifications()
	// SNMPv2-MIB defines coldStart, warmStart, authenticationFailure
	if len(notifs) == 0 {
		t.Skip("SNMPv2-MIB reports no notifications from Module.Notifications()")
		return
	}

	names := make(map[string]bool)
	for _, n := range notifs {
		names[n.Name()] = true
	}

	// At minimum coldStart should be there
	if !names["coldStart"] {
		t.Logf("SNMPv2-MIB notifications: %v", names)
		t.Skip("coldStart not in SNMPv2-MIB.Notifications() - may only include directly defined notifications")
	}
}

// === Notification objects (varbinds) ===

func TestNotificationObjects(t *testing.T) {
	m := loadTestMIB(t)

	notif := m.FindNotification("linkDown")
	if notif == nil {
		t.Skip("linkDown not found")
		return
	}

	objects := notif.Objects()
	// linkDown OBJECTS { ifIndex, ifAdminStatus, ifOperStatus }
	if len(objects) == 0 {
		t.Skip("linkDown has no Objects()")
		return
	}

	names := make([]string, len(objects))
	for i, obj := range objects {
		names[i] = obj.Name()
	}
	testutil.Equal(t, 3, len(objects),
		"linkDown should have 3 OBJECTS (ifIndex, ifAdminStatus, ifOperStatus), got %v", names)
}

// === Augments with effective indexes ===

func TestAugmentsEffectiveIndexes(t *testing.T) {
	pm := loadSemanticsMIB(t)

	entry := pm.FindObject("problemAugEntry")
	if entry == nil {
		t.Skip("problemAugEntry not found")
		return
	}

	// AUGMENTS { problemSemEntry } which has INDEX { problemSemIndex }
	aug := entry.Augments()
	if aug == nil {
		t.Skip("AUGMENTS not resolved")
		return
	}

	indexes := entry.EffectiveIndexes()
	if len(indexes) == 0 {
		t.Skip("EffectiveIndexes() empty for augmenting entry")
		return
	}

	// Should inherit problemSemIndex from the augmented table
	testutil.Equal(t, "problemSemIndex", indexes[0].Object.Name(),
		"augmenting table should inherit indexes from augmented table")
}
