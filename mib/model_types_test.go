package mib

import "testing"

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

	if n.Name() != "linkDown" {
		t.Errorf("Name() = %q, want %q", n.Name(), "linkDown")
	}
	if n.Node() != nd {
		t.Error("Node() returned wrong node")
	}
	if n.Module() != mod {
		t.Error("Module() returned wrong module")
	}
	if n.Status() != StatusCurrent {
		t.Errorf("Status() = %v, want %v", n.Status(), StatusCurrent)
	}
	if n.Description() != "Link is down" {
		t.Errorf("Description() = %q, want %q", n.Description(), "Link is down")
	}
	if n.Reference() != "RFC 2863" {
		t.Errorf("Reference() = %q, want %q", n.Reference(), "RFC 2863")
	}
	objs := n.Objects()
	if len(objs) != 2 {
		t.Fatalf("Objects() len = %d, want 2", len(objs))
	}
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
	if oid.String() != "3" {
		t.Errorf("OID() = %v, want 3", oid)
	}
}

func TestNotificationOIDNilNode(t *testing.T) {
	n := newNotification("noNode")
	oid := n.OID()
	if oid != nil {
		t.Errorf("OID() = %v, want nil", oid)
	}
}

func TestNotificationString(t *testing.T) {
	root := &Node{kind: KindInternal}
	nd := &Node{arc: 5, parent: root}
	root.children = map[uint32]*Node{5: nd}

	n := newNotification("testNotif")
	n.setNode(nd)

	got := n.String()
	if got != "testNotif (5)" {
		t.Errorf("String() = %q, want %q", got, "testNotif (5)")
	}
}

func TestNotificationStringNil(t *testing.T) {
	var n *Notification
	if n.String() != "<nil>" {
		t.Errorf("nil String() = %q, want %q", n.String(), "<nil>")
	}
}

func TestNotificationTrapInfo(t *testing.T) {
	n := newNotification("coldStart")

	if n.TrapInfo() != nil {
		t.Error("TrapInfo() should be nil initially")
	}

	ti := &TrapInfo{Enterprise: "enterprises", TrapNumber: 0}
	n.setTrapInfo(ti)

	if n.TrapInfo() != ti {
		t.Error("TrapInfo() should return the set value")
	}
}

func TestNotificationObjectsClone(t *testing.T) {
	obj := &Object{name: "testObj"}
	n := newNotification("testNotif")
	n.addObject(obj)

	objs := n.Objects()
	objs[0] = nil

	if n.objects[0] == nil {
		t.Error("mutating Objects() return should not affect internal state")
	}
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

	if g.Name() != "testGroup" {
		t.Errorf("Name() = %q, want %q", g.Name(), "testGroup")
	}
	if g.Node() != nd {
		t.Error("Node() returned wrong node")
	}
	if g.Module() != mod {
		t.Error("Module() returned wrong module")
	}
	if g.Status() != StatusCurrent {
		t.Errorf("Status() = %v, want %v", g.Status(), StatusCurrent)
	}
	if g.Description() != "Test group" {
		t.Errorf("Description() = %q, want %q", g.Description(), "Test group")
	}
	if g.Reference() != "RFC 1234" {
		t.Errorf("Reference() = %q, want %q", g.Reference(), "RFC 1234")
	}
	if g.IsNotificationGroup() {
		t.Error("IsNotificationGroup() should be false")
	}
	members := g.Members()
	if len(members) != 2 {
		t.Fatalf("Members() len = %d, want 2", len(members))
	}
}

func TestGroupNotificationGroup(t *testing.T) {
	g := newGroup("notifGroup")
	g.setIsNotificationGroup(true)

	if !g.IsNotificationGroup() {
		t.Error("IsNotificationGroup() should be true")
	}
}

func TestGroupOIDNilNode(t *testing.T) {
	g := newGroup("noNode")
	if g.OID() != nil {
		t.Errorf("OID() = %v, want nil", g.OID())
	}
}

func TestGroupStringNil(t *testing.T) {
	var g *Group
	if g.String() != "<nil>" {
		t.Errorf("nil String() = %q, want %q", g.String(), "<nil>")
	}
}

func TestGroupMembersClone(t *testing.T) {
	nd := &Node{name: "obj1"}
	g := newGroup("testGroup")
	g.addMember(nd)

	members := g.Members()
	members[0] = nil

	if g.members[0] == nil {
		t.Error("mutating Members() return should not affect internal state")
	}
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

	if c.Name() != "testCompliance" {
		t.Errorf("Name() = %q, want %q", c.Name(), "testCompliance")
	}
	if c.Node() != nd {
		t.Error("Node() returned wrong node")
	}
	if c.Module() != mod {
		t.Error("Module() returned wrong module")
	}
	if c.Status() != StatusCurrent {
		t.Errorf("Status() = %v, want %v", c.Status(), StatusCurrent)
	}
	if c.Description() != "Test compliance" {
		t.Errorf("Description() = %q, want %q", c.Description(), "Test compliance")
	}
	if c.Reference() != "RFC 5678" {
		t.Errorf("Reference() = %q, want %q", c.Reference(), "RFC 5678")
	}
	mods := c.Modules()
	if len(mods) != 1 {
		t.Fatalf("Modules() len = %d, want 1", len(mods))
	}
	if mods[0].ModuleName != "IF-MIB" {
		t.Errorf("Modules()[0].ModuleName = %q, want %q", mods[0].ModuleName, "IF-MIB")
	}
}

func TestComplianceOIDNilNode(t *testing.T) {
	c := newCompliance("noNode")
	if c.OID() != nil {
		t.Errorf("OID() = %v, want nil", c.OID())
	}
}

func TestComplianceStringNil(t *testing.T) {
	var c *Compliance
	if c.String() != "<nil>" {
		t.Errorf("nil String() = %q, want %q", c.String(), "<nil>")
	}
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

	if cap.Name() != "testCap" {
		t.Errorf("Name() = %q, want %q", cap.Name(), "testCap")
	}
	if cap.Node() != nd {
		t.Error("Node() returned wrong node")
	}
	if cap.Module() != mod {
		t.Error("Module() returned wrong module")
	}
	if cap.Status() != StatusCurrent {
		t.Errorf("Status() = %v, want %v", cap.Status(), StatusCurrent)
	}
	if cap.Description() != "Test capabilities" {
		t.Errorf("Description() = %q, want %q", cap.Description(), "Test capabilities")
	}
	if cap.Reference() != "RFC 9999" {
		t.Errorf("Reference() = %q, want %q", cap.Reference(), "RFC 9999")
	}
	if cap.ProductRelease() != "Agent 1.0" {
		t.Errorf("ProductRelease() = %q, want %q", cap.ProductRelease(), "Agent 1.0")
	}
	supports := cap.Supports()
	if len(supports) != 1 {
		t.Fatalf("Supports() len = %d, want 1", len(supports))
	}
	if supports[0].ModuleName != "IF-MIB" {
		t.Errorf("Supports()[0].ModuleName = %q, want %q", supports[0].ModuleName, "IF-MIB")
	}
}

func TestCapabilityOIDNilNode(t *testing.T) {
	cap := newCapability("noNode")
	if cap.OID() != nil {
		t.Errorf("OID() = %v, want nil", cap.OID())
	}
}

func TestCapabilityStringNil(t *testing.T) {
	var cap *Capability
	if cap.String() != "<nil>" {
		t.Errorf("nil String() = %q, want %q", cap.String(), "<nil>")
	}
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

	if mod.Name() != "IF-MIB" {
		t.Errorf("Name() = %q, want %q", mod.Name(), "IF-MIB")
	}
	if mod.Language() != LanguageSMIv2 {
		t.Errorf("Language() = %v, want %v", mod.Language(), LanguageSMIv2)
	}
	if mod.SourcePath() != "/usr/share/snmp/mibs/IF-MIB.mib" {
		t.Errorf("SourcePath() = %q", mod.SourcePath())
	}
	if mod.OID().String() != "1.3.6.1.2.1.31" {
		t.Errorf("OID() = %v", mod.OID())
	}
	if mod.Organization() != "IETF" {
		t.Errorf("Organization() = %q", mod.Organization())
	}
	if mod.ContactInfo() != "info@ietf.org" {
		t.Errorf("ContactInfo() = %q", mod.ContactInfo())
	}
	if mod.Description() != "MIB for network interfaces" {
		t.Errorf("Description() = %q", mod.Description())
	}
	if mod.LastUpdated() != "200006140000Z" {
		t.Errorf("LastUpdated() = %q", mod.LastUpdated())
	}
	if len(mod.Revisions()) != 1 {
		t.Fatalf("Revisions() len = %d, want 1", len(mod.Revisions()))
	}
	if len(mod.Imports()) != 1 {
		t.Fatalf("Imports() len = %d, want 1", len(mod.Imports()))
	}
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

	if mod.Object("testObj") != obj {
		t.Error("Object() lookup failed")
	}
	if mod.Type("TestType") != typ {
		t.Error("Type() lookup failed")
	}
	if mod.Notification("testNotif") != notif {
		t.Error("Notification() lookup failed")
	}
	if mod.Group("testGroup") != grp {
		t.Error("Group() lookup failed")
	}
	if mod.Compliance("testComp") != comp {
		t.Error("Compliance() lookup failed")
	}
	if mod.Capability("testCap") != cap {
		t.Error("Capability() lookup failed")
	}
	if mod.Node("testNode") != nd {
		t.Error("Node() lookup failed")
	}

	// Not found
	if mod.Object("missing") != nil {
		t.Error("Object() should return nil for missing name")
	}
	if mod.Type("missing") != nil {
		t.Error("Type() should return nil for missing name")
	}
	if mod.Notification("missing") != nil {
		t.Error("Notification() should return nil for missing name")
	}
	if mod.Group("missing") != nil {
		t.Error("Group() should return nil for missing name")
	}
	if mod.Compliance("missing") != nil {
		t.Error("Compliance() should return nil for missing name")
	}
	if mod.Capability("missing") != nil {
		t.Error("Capability() should return nil for missing name")
	}
	if mod.Node("missing") != nil {
		t.Error("Node() should return nil for missing name")
	}
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

	if len(mod.Objects()) != 4 {
		t.Errorf("Objects() len = %d, want 4", len(mod.Objects()))
	}
	if len(mod.Tables()) != 1 {
		t.Errorf("Tables() len = %d, want 1", len(mod.Tables()))
	}
	if len(mod.Rows()) != 1 {
		t.Errorf("Rows() len = %d, want 1", len(mod.Rows()))
	}
	if len(mod.Columns()) != 1 {
		t.Errorf("Columns() len = %d, want 1", len(mod.Columns()))
	}
	if len(mod.Scalars()) != 1 {
		t.Errorf("Scalars() len = %d, want 1", len(mod.Scalars()))
	}
}
