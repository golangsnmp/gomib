package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// --- Notification ---

func TestNotificationAccessors(t *testing.T) {
	root := &Node{kind: KindInternal}
	nd := &Node{arc: 1, parent: root, children: make(map[uint32]*Node)}
	root.children = map[uint32]*Node{1: nd}

	mod := newModule("TEST-MIB")
	obj1 := &Object{name: "testObj1"}
	obj2 := &Object{name: "testObj2"}

	n := newNotification("linkDown")
	n.setNode(nd)
	n.setModule(mod)
	n.setStatus(StatusCurrent)
	n.setDescription("Link is down")
	n.setReference("RFC 2863")
	n.addObject(obj1)
	n.addObject(obj2)

	testutil.Equal(t, "linkDown", n.Name(), "Name()")
	testutil.Equal(t, nd, n.Node(), "Node() returned wrong node")
	testutil.Equal(t, mod, n.Module(), "Module() returned wrong module")
	testutil.Equal(t, StatusCurrent, n.Status(), "Status()")
	testutil.Equal(t, "Link is down", n.Description(), "Description()")
	testutil.Equal(t, "RFC 2863", n.Reference(), "Reference()")
	objs := n.Objects()
	testutil.Len(t, objs, 2, "Objects() len")
	if objs[0] != obj1 || objs[1] != obj2 {
		t.Error("Objects() returned wrong objects")
	}
}

func TestNotificationOID(t *testing.T) {
	root := &Node{kind: KindInternal}
	nd := &Node{arc: 3, parent: root}
	root.children = map[uint32]*Node{3: nd}

	n := newNotification("testNotif")
	n.setNode(nd)

	oid := n.OID()
	testutil.Equal(t, "3", oid.String(), "OID()")
}

func TestNotificationOIDNilNode(t *testing.T) {
	n := newNotification("noNode")
	oid := n.OID()
	testutil.Nil(t, oid, "OID()")
}

func TestNotificationString(t *testing.T) {
	root := &Node{kind: KindInternal}
	nd := &Node{arc: 5, parent: root}
	root.children = map[uint32]*Node{5: nd}

	n := newNotification("testNotif")
	n.setNode(nd)

	got := n.String()
	testutil.Equal(t, "testNotif (5)", got, "String()")
}

func TestNotificationStringNil(t *testing.T) {
	var n *Notification
	testutil.Equal(t, "<nil>", n.String(), "nil String()")
}

func TestNotificationTrapInfo(t *testing.T) {
	n := newNotification("coldStart")

	testutil.Nil(t, n.TrapInfo(), "TrapInfo() should be nil initially")

	ti := &TrapInfo{Enterprise: "enterprises", TrapNumber: 0}
	n.setTrapInfo(ti)

	testutil.Equal(t, ti, n.TrapInfo(), "TrapInfo() should return the set value")
}

func TestNotificationObjectsClone(t *testing.T) {
	obj := &Object{name: "testObj"}
	n := newNotification("testNotif")
	n.addObject(obj)

	objs := n.Objects()
	objs[0] = nil

	testutil.NotNil(t, n.objects[0], "mutating Objects() return should not affect internal state")
}

// --- Group ---

func TestGroupAccessors(t *testing.T) {
	root := &Node{kind: KindInternal}
	nd := &Node{arc: 1, parent: root}
	root.children = map[uint32]*Node{1: nd}

	mod := newModule("TEST-MIB")
	member1 := &Node{name: "obj1"}
	member2 := &Node{name: "obj2"}

	g := newGroup("testGroup")
	g.setNode(nd)
	g.setModule(mod)
	g.setStatus(StatusCurrent)
	g.setDescription("Test group")
	g.setReference("RFC 1234")
	g.setIsNotificationGroup(false)
	g.addMember(member1)
	g.addMember(member2)

	testutil.Equal(t, "testGroup", g.Name(), "Name()")
	testutil.Equal(t, nd, g.Node(), "Node() returned wrong node")
	testutil.Equal(t, mod, g.Module(), "Module() returned wrong module")
	testutil.Equal(t, StatusCurrent, g.Status(), "Status()")
	testutil.Equal(t, "Test group", g.Description(), "Description()")
	testutil.Equal(t, "RFC 1234", g.Reference(), "Reference()")
	testutil.False(t, g.IsNotificationGroup(), "IsNotificationGroup() should be false")
	members := g.Members()
	testutil.Len(t, members, 2, "Members() len")
}

func TestGroupNotificationGroup(t *testing.T) {
	g := newGroup("notifGroup")
	g.setIsNotificationGroup(true)

	testutil.True(t, g.IsNotificationGroup(), "IsNotificationGroup() should be true")
}

func TestGroupOIDNilNode(t *testing.T) {
	g := newGroup("noNode")
	testutil.Nil(t, g.OID(), "OID()")
}

func TestGroupStringNil(t *testing.T) {
	var g *Group
	testutil.Equal(t, "<nil>", g.String(), "nil String()")
}

func TestGroupMembersClone(t *testing.T) {
	nd := &Node{name: "obj1"}
	g := newGroup("testGroup")
	g.addMember(nd)

	members := g.Members()
	members[0] = nil

	testutil.NotNil(t, g.members[0], "mutating Members() return should not affect internal state")
}

// --- Compliance ---

func TestComplianceAccessors(t *testing.T) {
	root := &Node{kind: KindInternal}
	nd := &Node{arc: 1, parent: root}
	root.children = map[uint32]*Node{1: nd}

	mod := newModule("TEST-MIB")

	c := newCompliance("testCompliance")
	c.setNode(nd)
	c.setModule(mod)
	c.setStatus(StatusCurrent)
	c.setDescription("Test compliance")
	c.setReference("RFC 5678")
	c.setModules([]ComplianceModule{
		{ModuleName: "IF-MIB", MandatoryGroups: []string{"ifGroup"}},
	})

	testutil.Equal(t, "testCompliance", c.Name(), "Name()")
	testutil.Equal(t, nd, c.Node(), "Node() returned wrong node")
	testutil.Equal(t, mod, c.Module(), "Module() returned wrong module")
	testutil.Equal(t, StatusCurrent, c.Status(), "Status()")
	testutil.Equal(t, "Test compliance", c.Description(), "Description()")
	testutil.Equal(t, "RFC 5678", c.Reference(), "Reference()")
	mods := c.Modules()
	testutil.Len(t, mods, 1, "Modules() len")
	testutil.Equal(t, "IF-MIB", mods[0].ModuleName, "Modules()[0].ModuleName")
}

func TestComplianceOIDNilNode(t *testing.T) {
	c := newCompliance("noNode")
	testutil.Nil(t, c.OID(), "OID()")
}

func TestComplianceStringNil(t *testing.T) {
	var c *Compliance
	testutil.Equal(t, "<nil>", c.String(), "nil String()")
}

// --- Capability ---

func TestCapabilityAccessors(t *testing.T) {
	root := &Node{kind: KindInternal}
	nd := &Node{arc: 1, parent: root}
	root.children = map[uint32]*Node{1: nd}

	mod := newModule("TEST-MIB")

	cap := newCapability("testCap")
	cap.setNode(nd)
	cap.setModule(mod)
	cap.setStatus(StatusCurrent)
	cap.setDescription("Test capabilities")
	cap.setReference("RFC 9999")
	cap.setProductRelease("Agent 1.0")
	cap.setSupports([]CapabilitiesModule{
		{ModuleName: "IF-MIB", Includes: []string{"ifGroup"}},
	})

	testutil.Equal(t, "testCap", cap.Name(), "Name()")
	testutil.Equal(t, nd, cap.Node(), "Node() returned wrong node")
	testutil.Equal(t, mod, cap.Module(), "Module() returned wrong module")
	testutil.Equal(t, StatusCurrent, cap.Status(), "Status()")
	testutil.Equal(t, "Test capabilities", cap.Description(), "Description()")
	testutil.Equal(t, "RFC 9999", cap.Reference(), "Reference()")
	testutil.Equal(t, "Agent 1.0", cap.ProductRelease(), "ProductRelease()")
	supports := cap.Supports()
	testutil.Len(t, supports, 1, "Supports() len")
	testutil.Equal(t, "IF-MIB", supports[0].ModuleName, "Supports()[0].ModuleName")
}

func TestCapabilityOIDNilNode(t *testing.T) {
	cap := newCapability("noNode")
	testutil.Nil(t, cap.OID(), "OID()")
}

func TestCapabilityStringNil(t *testing.T) {
	var cap *Capability
	testutil.Equal(t, "<nil>", cap.String(), "nil String()")
}

// --- Module ---

func TestModuleAccessors(t *testing.T) {
	mod := newModule("IF-MIB")
	mod.setLanguage(LanguageSMIv2)
	mod.setSourcePath("/usr/share/snmp/mibs/IF-MIB.mib")
	mod.setOID(OID{1, 3, 6, 1, 2, 1, 31})
	mod.setOrganization("IETF")
	mod.setContactInfo("info@ietf.org")
	mod.setDescription("MIB for network interfaces")
	mod.setLastUpdated("200006140000Z")
	mod.setRevisions([]Revision{
		{Date: "2000-06-14", Description: "Initial version"},
	})
	mod.setImports([]Import{
		{Module: "SNMPv2-SMI", Symbols: []string{"MODULE-IDENTITY"}},
	})

	testutil.Equal(t, "IF-MIB", mod.Name(), "Name()")
	testutil.Equal(t, LanguageSMIv2, mod.Language(), "Language()")
	testutil.Equal(t, "/usr/share/snmp/mibs/IF-MIB.mib", mod.SourcePath(), "SourcePath()")
	testutil.Equal(t, "1.3.6.1.2.1.31", mod.OID().String(), "OID()")
	testutil.Equal(t, "IETF", mod.Organization(), "Organization()")
	testutil.Equal(t, "info@ietf.org", mod.ContactInfo(), "ContactInfo()")
	testutil.Equal(t, "MIB for network interfaces", mod.Description(), "Description()")
	testutil.Equal(t, "200006140000Z", mod.LastUpdated(), "LastUpdated()")
	testutil.Len(t, mod.Revisions(), 1, "Revisions() len")
	testutil.Len(t, mod.Imports(), 1, "Imports() len")
}

func TestModuleEntityLookups(t *testing.T) {
	mod := newModule("TEST-MIB")

	obj := &Object{name: "testObj"}
	mod.addObject(obj)

	typ := &Type{name: "TestType"}
	mod.addType(typ)

	notif := &Notification{name: "testNotif"}
	mod.addNotification(notif)

	grp := &Group{name: "testGroup"}
	mod.addGroup(grp)

	comp := &Compliance{name: "testComp"}
	mod.addCompliance(comp)

	cap := &Capability{name: "testCap"}
	mod.addCapability(cap)

	nd := &Node{name: "testNode"}
	mod.addNode(nd)

	testutil.Equal(t, obj, mod.Object("testObj"), "Object() lookup failed")
	testutil.Equal(t, typ, mod.Type("TestType"), "Type() lookup failed")
	testutil.Equal(t, notif, mod.Notification("testNotif"), "Notification() lookup failed")
	testutil.Equal(t, grp, mod.Group("testGroup"), "Group() lookup failed")
	testutil.Equal(t, comp, mod.Compliance("testComp"), "Compliance() lookup failed")
	testutil.Equal(t, cap, mod.Capability("testCap"), "Capability() lookup failed")
	testutil.Equal(t, nd, mod.Node("testNode"), "Node() lookup failed")

	// Not found
	testutil.Nil(t, mod.Object("missing"), "Object() should return nil for missing name")
	testutil.Nil(t, mod.Type("missing"), "Type() should return nil for missing name")
	testutil.Nil(t, mod.Notification("missing"), "Notification() should return nil for missing name")
	testutil.Nil(t, mod.Group("missing"), "Group() should return nil for missing name")
	testutil.Nil(t, mod.Compliance("missing"), "Compliance() should return nil for missing name")
	testutil.Nil(t, mod.Capability("missing"), "Capability() should return nil for missing name")
	testutil.Nil(t, mod.Node("missing"), "Node() should return nil for missing name")
}

func TestModuleCollections(t *testing.T) {
	mod := newModule("TEST-MIB")

	root := &Node{kind: KindInternal}
	tableNode := &Node{arc: 1, kind: KindTable, parent: root}
	rowNode := &Node{arc: 1, kind: KindRow, parent: tableNode}
	colNode := &Node{arc: 1, kind: KindColumn, parent: rowNode}
	scalarNode := &Node{arc: 2, kind: KindScalar, parent: root}

	tableObj := &Object{name: "fooTable", node: tableNode}
	rowObj := &Object{name: "fooEntry", node: rowNode}
	colObj := &Object{name: "fooCol", node: colNode}
	scalarObj := &Object{name: "fooScalar", node: scalarNode}

	mod.addObject(tableObj)
	mod.addObject(rowObj)
	mod.addObject(colObj)
	mod.addObject(scalarObj)

	testutil.Len(t, mod.Objects(), 4, "Objects() len")
	testutil.Len(t, mod.Tables(), 1, "Tables() len")
	testutil.Len(t, mod.Rows(), 1, "Rows() len")
	testutil.Len(t, mod.Columns(), 1, "Columns() len")
	testutil.Len(t, mod.Scalars(), 1, "Scalars() len")
}
