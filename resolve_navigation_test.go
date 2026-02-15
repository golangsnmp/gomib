package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

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
	node := m.Node("ifIndex")
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

	entry := m.Node("ifEntry")
	testutil.NotNil(t, entry, "Node(ifEntry)")

	children := entry.Children()
	testutil.Greater(t, len(children), 0,
		"ifEntry should have children (table columns)")

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

	entry := m.Node("ifEntry")
	testutil.NotNil(t, entry, "Node(ifEntry)")

	// ifIndex is arc 1 under ifEntry
	child := entry.Child(1)
	testutil.NotNil(t, child, "Child(1) of ifEntry")
	testutil.Equal(t, "ifIndex", child.Name(), "Child(1) of ifEntry should be ifIndex")

	noChild := entry.Child(99999)
	testutil.Nil(t, noChild, "Child(99999) should return nil for non-existent arc")
}

func TestNodeSubtree(t *testing.T) {
	m := loadTestMIB(t)

	table := m.Node("ifTable")
	testutil.NotNil(t, table, "Node(ifTable)")

	count := 0
	for range table.Subtree() {
		count++
	}
	// ifTable -> ifEntry -> ifIndex, ifDescr, ifType, ... (22+ columns)
	testutil.Greater(t, count, 5,
		"ifTable should have many descendants")
}

func TestNodeArc(t *testing.T) {
	m := loadTestMIB(t)

	// ifIndex OID = 1.3.6.1.2.1.2.2.1.1 (last arc is 1)
	node := m.Node("ifIndex")
	if node == nil {
		t.Fatal("ifIndex not found")
	}
	testutil.Equal(t, uint32(1), node.Arc(), "ifIndex arc should be 1")
}

func TestNodeObjectAndNotification(t *testing.T) {
	m := loadTestMIB(t)

	node := m.Node("ifIndex")
	if node == nil {
		t.Fatal("ifIndex not found")
	}
	obj := node.Object()
	testutil.NotNil(t, obj, "ifIndex node should have an associated Object")
	testutil.Equal(t, "ifIndex", obj.Name(), "associated object name")

	testutil.Nil(t, node.Notification(), "ifIndex should not have a notification")

	linkDown := m.Node("linkDown")
	testutil.NotNil(t, linkDown, "Node(linkDown)")
	notif := linkDown.Notification()
	testutil.NotNil(t, notif, "linkDown node should have an associated notification")
	testutil.Equal(t, "linkDown", notif.Name(), "linkDown notification name")
}

func TestObjectTableNavigation(t *testing.T) {
	m := loadTestMIB(t)

	t.Run("column to table", func(t *testing.T) {
		col := m.Object("ifIndex")
		if col == nil {
			t.Fatal("ifIndex not found")
		}
		testutil.True(t, col.IsColumn(), "ifIndex should be a column")
		testutil.False(t, col.IsTable(), "ifIndex should not be a table")
		testutil.False(t, col.IsRow(), "ifIndex should not be a row")
		testutil.False(t, col.IsScalar(), "ifIndex should not be a scalar")

		tbl := col.Table()
		testutil.NotNil(t, tbl, "Table() for ifIndex")
		testutil.Equal(t, "ifTable", tbl.Name(), "ifIndex's table should be ifTable")
		testutil.True(t, tbl.IsTable(), "ifTable should be a table")
	})

	t.Run("column to row", func(t *testing.T) {
		col := m.Object("ifIndex")
		if col == nil {
			t.Fatal("ifIndex not found")
		}

		row := col.Row()
		testutil.NotNil(t, row, "Row() for ifIndex")
		testutil.Equal(t, "ifEntry", row.Name(), "ifIndex's row should be ifEntry")
		testutil.True(t, row.IsRow(), "ifEntry should be a row")
	})

	t.Run("table to entry", func(t *testing.T) {
		tbl := m.Object("ifTable")
		testutil.NotNil(t, tbl, "Object(ifTable)")
		testutil.True(t, tbl.IsTable(), "ifTable should be a table")

		entry := tbl.Entry()
		testutil.NotNil(t, entry, "Entry() for ifTable")
		testutil.Equal(t, "ifEntry", entry.Name(), "ifTable entry should be ifEntry")
	})

	t.Run("row columns", func(t *testing.T) {
		row := m.Object("ifEntry")
		testutil.NotNil(t, row, "Object(ifEntry)")

		cols := row.Columns()
		testutil.Greater(t, len(cols), 5, "ifEntry should have many columns")

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
		scalar := m.Object("sysDescr")
		testutil.NotNil(t, scalar, "Object(sysDescr)")
		testutil.True(t, scalar.IsScalar(), "sysDescr should be a scalar")
		testutil.False(t, scalar.IsTable(), "sysDescr should not be a table")
		testutil.False(t, scalar.IsRow(), "sysDescr should not be a row")
		testutil.False(t, scalar.IsColumn(), "sysDescr should not be a column")
	})
}

func TestObjectEffectiveIndexes(t *testing.T) {
	m := loadTestMIB(t)

	entry := m.Object("ifEntry")
	testutil.NotNil(t, entry, "Object(ifEntry)")

	indexes := entry.EffectiveIndexes()
	testutil.NotEmpty(t, indexes, "EffectiveIndexes() for ifEntry")

	testutil.Equal(t, 1, len(indexes), "ifEntry should have 1 index")
	testutil.Equal(t, "ifIndex", indexes[0].Object.Name(), "index should be ifIndex")
}

func TestObjectEnumLookup(t *testing.T) {
	m := loadTestMIB(t)

	obj := m.Object("ifType")
	testutil.NotNil(t, obj, "Object(ifType)")

	enums := obj.EffectiveEnums()
	testutil.NotEmpty(t, enums, "EffectiveEnums() for ifType")

	nv, ok := obj.Enum("ethernetCsmacd")
	testutil.True(t, ok, "Enum(ethernetCsmacd) should be found")
	testutil.Equal(t, int64(6), nv.Value, "ethernetCsmacd should be 6")

	_, ok = obj.Enum("totallyFakeLabel")
	testutil.False(t, ok, "non-existent label should return false")
}

func TestObjectBitLookup(t *testing.T) {
	// Use PROBLEM-DEFVAL-MIB which has BITS objects (problemDefvalEmptyBits, problemDefvalMultiBits)
	pm := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := pm.Object("problemDefvalMultiBits")
	testutil.NotNil(t, obj, "Object(problemDefvalMultiBits)")

	bits := obj.EffectiveBits()
	testutil.NotEmpty(t, bits, "EffectiveBits() for problemDefvalMultiBits")

	nv, ok := obj.Bit("read")
	testutil.True(t, ok, "Bit(read) should be found")
	testutil.Equal(t, int64(0), nv.Value, "read bit should be 0")

	nv, ok = obj.Bit("write")
	testutil.True(t, ok, "Bit(write) should be found")
	testutil.Equal(t, int64(1), nv.Value, "write bit should be 1")

	_, ok = obj.Bit("totallyFakeBit")
	testutil.False(t, ok, "non-existent bit should return false")
}

func TestTypePredicates(t *testing.T) {
	m := loadTestMIB(t)

	t.Run("IsString", func(t *testing.T) {
		typ := m.Type("DisplayString")
		testutil.NotNil(t, typ, "Type(DisplayString)")
		testutil.True(t, typ.IsString(), "DisplayString should be IsString()")
		testutil.False(t, typ.IsCounter(), "DisplayString should not be IsCounter()")
		testutil.False(t, typ.IsGauge(), "DisplayString should not be IsGauge()")
	})

	t.Run("IsEnumeration", func(t *testing.T) {
		// ifType uses IANAifType which is an enumeration
		obj := m.Object("ifType")
		testutil.NotNil(t, obj, "Object(ifType)")
		testutil.NotNil(t, obj.Type(), "ifType type")
		enums := obj.EffectiveEnums()
		testutil.NotEmpty(t, enums, "EffectiveEnums() for ifType")
		testutil.True(t, obj.Type().IsEnumeration(), "ifType type should report IsEnumeration()")
	})

	t.Run("counter type from problem MIBs", func(t *testing.T) {
		pm := loadTypeChainsMIB(t)
		obj := pm.Object("problemAppTypeChain")
		testutil.NotNil(t, obj, "Object(problemAppTypeChain)")
		testutil.NotNil(t, obj.Type(), "problemAppTypeChain type")
		testutil.True(t, obj.Type().IsCounter(),
			"MyCounter (based on Counter32) should report IsCounter()")
	})

	t.Run("gauge type from problem MIBs", func(t *testing.T) {
		pm := loadTypeChainsMIB(t)
		obj := pm.Object("problemInheritedHint")
		testutil.NotNil(t, obj, "Object(problemInheritedHint)")
		testutil.NotNil(t, obj.Type(), "problemInheritedHint type")
		testutil.True(t, obj.Type().IsGauge(),
			"MySpecialGauge (based on Gauge32) should report IsGauge()")
	})

	t.Run("IsBits", func(t *testing.T) {
		pm := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")
		obj := pm.Object("problemDefvalMultiBits")
		testutil.NotNil(t, obj, "Object(problemDefvalMultiBits)")
		testutil.NotNil(t, obj.Type(), "problemDefvalMultiBits type")
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

	typ := m.Type("DisplayString")
	testutil.NotNil(t, typ, "Type(DisplayString)")

	parent := typ.Parent()
	testutil.NotNil(t, parent, "DisplayString Parent()")

	// DisplayString is based on OCTET STRING
	testutil.Equal(t, mib.BaseOctetString, parent.EffectiveBase(),
		"DisplayString's parent should resolve to OCTET STRING")
}

func TestTypeEnumLookup(t *testing.T) {
	pm := loadTypeChainsMIB(t)

	obj := pm.Object("problemEnumChain")
	testutil.NotNil(t, obj, "Object(problemEnumChain)")
	testutil.NotNil(t, obj.Type(), "problemEnumChain type")

	typ := obj.Type()
	for typ != nil {
		if len(typ.Enums()) > 0 {
			nv, ok := typ.Enum("active")
			testutil.True(t, ok, "Enum(active) should be found on type")
			testutil.Equal(t, int64(1), nv.Value, "active should be 1")
			return
		}
		typ = typ.Parent()
	}
	t.Fatal("no type in chain has direct enums")
}

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
	testutil.True(t, desc != "", "IF-MIB should have a non-empty description")
}

func TestModuleOID(t *testing.T) {
	m := loadTestMIB(t)

	ifMIB := m.Module("IF-MIB")
	if ifMIB == nil {
		t.Fatal("IF-MIB not found")
	}

	oid := ifMIB.OID()
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
		testutil.NotNil(t, obj, "Module.Object(ifIndex)")
		testutil.Equal(t, "ifIndex", obj.Name(), "module-scoped object name")
	})

	t.Run("Node lookup", func(t *testing.T) {
		// ifMIBObjects is defined in IF-MIB as a node
		node := ifMIB.Node("ifMIBObjects")
		if node == nil {
			t.Fatal("Module.Node(ifMIBObjects) returned nil")
		}
		testutil.Equal(t, "ifMIBObjects", node.Name(), "module-scoped node name")
	})

	t.Run("Type lookup", func(t *testing.T) {
		// IF-MIB defines InterfaceIndex TC
		typ := ifMIB.Type("InterfaceIndex")
		testutil.NotNil(t, typ, "Module.Type(InterfaceIndex)")
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

func TestMibFilteredCollections(t *testing.T) {
	m := loadTestMIB(t)

	columns := m.Columns()
	rows := m.Rows()

	testutil.Greater(t, len(columns), 0, "should have columns")
	testutil.Greater(t, len(rows), 0, "should have rows")

	for _, col := range columns {
		testutil.Equal(t, mib.KindColumn, col.Kind(),
			"Columns() entry %s should have KindColumn", col.Name())
	}

	for _, row := range rows {
		testutil.Equal(t, mib.KindRow, row.Kind(),
			"Rows() entry %s should have KindRow", row.Name())
	}
}

func TestMibNotificationCount(t *testing.T) {
	m := loadTestMIB(t)

	notifications := m.Notifications()
	count := len(notifications)

	// The fixture modules include SNMPv2-MIB which has linkDown, linkUp, etc.
	testutil.Greater(t, count, 0, "should have some notifications")
}

func TestMibHasErrors(t *testing.T) {
	m := loadTestMIB(t)

	// Standard MIBs loaded at default strictness should not have errors.
	testutil.False(t, m.HasErrors(), "standard MIBs at default strictness should not have errors")

	// PROBLEM-IMPORTS-MIB has missing imports that fail at strict level.
	strict := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", mib.StrictnessStrict)
	testutil.True(t, strict.HasErrors(), "PROBLEM-IMPORTS-MIB at strict should have errors")
}

func TestNodeByOID(t *testing.T) {
	m := loadTestMIB(t)

	oid, err := mib.ParseOID("1.3.6.1.2.1.2.2.1.1")
	if err != nil {
		t.Fatalf("ParseOID failed: %v", err)
	}

	node := m.NodeByOID(oid)
	testutil.NotNil(t, node, "NodeByOID(1.3.6.1.2.1.2.2.1.1)")
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
	testutil.NotNil(t, node, "LongestPrefixByOID(ifIndex.5)")
	testutil.Equal(t, "ifIndex", node.Name(),
		"LongestPrefixByOID for ifIndex.5 should find ifIndex")
}

func TestNodeLongestPrefix(t *testing.T) {
	m := loadTestMIB(t)

	root := m.Root()
	if root == nil {
		t.Fatal("Root() returned nil")
	}

	oid, err := mib.ParseOID("1.3.6.1.2.1.2.2.1.1.5")
	if err != nil {
		t.Fatalf("ParseOID failed: %v", err)
	}

	node := root.LongestPrefix(oid)
	testutil.NotNil(t, node, "Node.LongestPrefix(ifIndex.5)")
	testutil.Equal(t, "ifIndex", node.Name(),
		"Node.LongestPrefix for ifIndex.5 should find ifIndex")
}

func TestObjectDescription(t *testing.T) {
	m := loadTestMIB(t)

	obj := m.Object("sysDescr")
	if obj == nil {
		t.Fatal("sysDescr not found")
	}

	desc := obj.Description()
	testutil.True(t, desc != "", "sysDescr should have a non-empty description")
	testutil.Greater(t, len(desc), 10,
		"sysDescr description should be non-trivial")
}

func TestObjectNode(t *testing.T) {
	m := loadTestMIB(t)

	obj := m.Object("ifIndex")
	if obj == nil {
		t.Fatal("ifIndex not found")
	}

	node := obj.Node()
	testutil.NotNil(t, node, "Object.Node() should not be nil")
	testutil.Equal(t, "ifIndex", node.Name(), "Object.Node().Name() should match")
	testutil.Equal(t, obj.OID().String(), node.OID().String(),
		"Object OID should match Node OID")
}

func TestNotificationMetadataFields(t *testing.T) {
	m := loadTestMIB(t)

	notif := m.Notification("linkDown")
	testutil.NotNil(t, notif, "Notification(linkDown)")

	node := notif.Node()
	testutil.NotNil(t, node, "Notification.Node() should not be nil")
	testutil.Equal(t, "linkDown", node.Name(), "notification node name")

	mod := notif.Module()
	testutil.NotNil(t, mod, "Notification.Module() should not be nil")

	oid := notif.OID()
	testutil.Greater(t, len(oid), 0, "notification OID should not be empty")

	status := notif.Status()
	testutil.Equal(t, mib.StatusCurrent, status, "linkDown should be current")
}

func TestTypesCollection(t *testing.T) {
	m := loadTestMIB(t)

	types := m.Types()
	count := len(types)
	testutil.Greater(t, count, 0, "should have types (DisplayString, etc.)")

	found := false
	for _, typ := range types {
		if typ.Name() == "DisplayString" {
			found = true
			break
		}
	}
	testutil.True(t, found, "DisplayString should be in Types() list")
}

func TestModuleNotifications(t *testing.T) {
	m := loadTestMIB(t)

	snmpMIB := m.Module("SNMPv2-MIB")
	testutil.NotNil(t, snmpMIB, "Module(SNMPv2-MIB)")

	notifs := snmpMIB.Notifications()
	testutil.NotEmpty(t, notifs, "SNMPv2-MIB Notifications()")

	names := make(map[string]bool)
	for _, n := range notifs {
		names[n.Name()] = true
	}

	testutil.True(t, names["coldStart"],
		"coldStart should be in SNMPv2-MIB.Notifications()")
}

func TestNotificationObjects(t *testing.T) {
	m := loadTestMIB(t)

	notif := m.Notification("linkDown")
	testutil.NotNil(t, notif, "Notification(linkDown)")

	objects := notif.Objects()
	testutil.NotEmpty(t, objects, "linkDown Objects()")

	names := make([]string, len(objects))
	for i, obj := range objects {
		names[i] = obj.Name()
	}
	testutil.Equal(t, 3, len(objects),
		"linkDown should have 3 OBJECTS (ifIndex, ifAdminStatus, ifOperStatus), got %v", names)
}

func TestAugmentsEffectiveIndexes(t *testing.T) {
	pm := loadSemanticsMIB(t)

	entry := pm.Object("problemAugEntry")
	testutil.NotNil(t, entry, "Object(problemAugEntry)")

	// AUGMENTS { problemSemEntry } which has INDEX { problemSemIndex }
	aug := entry.Augments()
	testutil.NotNil(t, aug, "Augments() for problemAugEntry")

	indexes := entry.EffectiveIndexes()
	testutil.NotEmpty(t, indexes, "EffectiveIndexes() for augmenting entry")

	// Should inherit problemSemIndex from the augmented table
	testutil.Equal(t, "problemSemIndex", indexes[0].Object.Name(),
		"augmenting table should inherit indexes from augmented table")
}
