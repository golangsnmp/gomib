package model

import "github.com/golangsnmp/gomib/internal/module"

// --- Mib setters ---

func SetMibNodeCount(m *Mib, n int)                { m.setNodeCount(n) }
func AddMibModule(m *Mib, mod *Module)             { m.addModule(mod) }
func AddMibObject(m *Mib, obj *Object)             { m.addObject(obj) }
func AddMibType(m *Mib, t *Type)                   { m.addType(t) }
func AddMibNotification(m *Mib, n *Notification)   { m.addNotification(n) }
func AddMibGroup(m *Mib, g *Group)                 { m.addGroup(g) }
func AddMibCompliance(m *Mib, c *Compliance)       { m.addCompliance(c) }
func AddMibCapability(m *Mib, c *Capability)       { m.addCapability(c) }
func RegisterMibNode(m *Mib, name string, n *Node) { m.registerNode(name, n) }
func AddMibDiagnostic(m *Mib, d *Diagnostic)       { m.addDiagnostic(d) }
func AddMibUnresolved(m *Mib, ref UnresolvedRef)   { m.addUnresolved(ref) }

// --- Node setters ---

func GetOrCreateChild(n *Node, arc uint32) *Node       { return n.getOrCreateChild(arc) }
func SetNodeSpan(n *Node, s Span)                      { n.setSpan(s) }
func SetNodeName(n *Node, name string)                 { n.setName(name) }
func SetNodeDescription(n *Node, d string)             { n.setDescription(d) }
func SetNodeReference(n *Node, r string)               { n.setReference(r) }
func SetNodeKind(n *Node, k Kind)                      { n.setKind(k) }
func SetNodeModule(n *Node, m *Module)                 { n.setModule(m) }
func SetNodeObject(n *Node, obj *Object)               { n.setObject(obj) }
func SetNodeNotification(n *Node, notif *Notification) { n.setNotification(notif) }
func SetNodeGroup(n *Node, g *Group)                   { n.setGroup(g) }
func SetNodeCompliance(n *Node, c *Compliance)         { n.setCompliance(c) }
func SetNodeCapability(n *Node, c *Capability)         { n.setCapability(c) }
func SetNodeObjectIdentity(n *Node, status Status)     { n.setObjectIdentity(status) }

// --- Object entity setters ---

func SetObjectSpan(o *Object, s Span)            { o.setSpan(s) }
func SetObjectNode(o *Object, nd *Node)          { o.setNode(nd) }
func SetObjectModule(o *Object, m *Module)       { o.setModule(m) }
func SetObjectStatus(o *Object, s Status)        { o.setStatus(s) }
func SetObjectDescription(o *Object, d string)   { o.setDescription(d) }
func SetObjectReference(o *Object, r string)     { o.setReference(r) }
func SetObjectStatusSpan(o *Object, s Span)      { o.setStatusSpan(s) }
func SetObjectDescriptionSpan(o *Object, s Span) { o.setDescriptionSpan(s) }
func SetObjectReferenceSpan(o *Object, s Span)   { o.setReferenceSpan(s) }
func SetObjectOidRefs(o *Object, refs []OidRef)  { o.setOidRefs(refs) }

// Object-specific setters

func SetObjectType(o *Object, t *Type)              { o.setType(t) }
func SetObjectAccess(o *Object, a Access)           { o.setAccess(a) }
func SetObjectUnits(o *Object, u string)            { o.setUnits(u) }
func SetObjectDefaultValue(o *Object, d *DefVal)    { o.setDefaultValue(d) }
func SetObjectAugments(o, a *Object)                { o.setAugments(a) }
func AddObjectAugmentedBy(o, a *Object)             { o.addAugmentedBy(a) }
func SetObjectIndex(o *Object, idx []IndexEntry)    { o.setIndex(idx) }
func SetObjectEffectiveHint(o *Object, h string)    { o.setEffectiveHint(h) }
func SetObjectDeclaredSizes(o *Object, s []Range)   { o.setDeclaredSizes(s) }
func SetObjectDeclaredRanges(o *Object, r []Range)  { o.setDeclaredRanges(r) }
func SetObjectEffectiveSizes(o *Object, s []Range)  { o.setEffectiveSizes(s) }
func SetObjectEffectiveRanges(o *Object, r []Range) { o.setEffectiveRanges(r) }
func SetObjectEffectiveSizesConstrained(o *Object, v bool) {
	o.setEffectiveSizesConstrained(v)
}

func SetObjectEffectiveRangesConstrained(o *Object, v bool) {
	o.setEffectiveRangesConstrained(v)
}
func SetObjectEffectiveEnums(o *Object, e []NamedValue) { o.setEffectiveEnums(e) }
func SetObjectEffectiveBits(o *Object, b []NamedValue)  { o.setEffectiveBits(b) }
func SetObjectSequenceTypeName(o *Object, s string)     { o.setSequenceTypeName(s) }
func SetObjectSyntaxSpan(o *Object, s Span)             { o.setSyntaxSpan(s) }
func SetObjectAccessSpan(o *Object, s Span)             { o.setAccessSpan(s) }
func SetObjectUnitsSpan(o *Object, s Span)              { o.setUnitsSpan(s) }
func SetObjectAugmentsSpan(o *Object, s Span)           { o.setAugmentsSpan(s) }
func SetObjectDefaultValueSpan(o *Object, s Span)       { o.setDefaultValueSpan(s) }

// --- Notification entity setters ---

func SetNotificationSpan(n *Notification, s Span)            { n.setSpan(s) }
func SetNotificationNode(n *Notification, nd *Node)          { n.setNode(nd) }
func SetNotificationModule(n *Notification, m *Module)       { n.setModule(m) }
func SetNotificationStatus(n *Notification, s Status)        { n.setStatus(s) }
func SetNotificationDescription(n *Notification, d string)   { n.setDescription(d) }
func SetNotificationReference(n *Notification, r string)     { n.setReference(r) }
func SetNotificationStatusSpan(n *Notification, s Span)      { n.setStatusSpan(s) }
func SetNotificationDescriptionSpan(n *Notification, s Span) { n.setDescriptionSpan(s) }
func SetNotificationReferenceSpan(n *Notification, s Span)   { n.setReferenceSpan(s) }
func SetNotificationOidRefs(n *Notification, refs []OidRef)  { n.setOidRefs(refs) }
func AddNotificationObject(n *Notification, obj *Object)     { n.addObject(obj) }
func SetNotificationTrapInfo(n *Notification, t *TrapInfo)   { n.setTrapInfo(t) }

// --- Group entity setters ---

func SetGroupSpan(g *Group, s Span)                { g.setSpan(s) }
func SetGroupNode(g *Group, nd *Node)              { g.setNode(nd) }
func SetGroupModule(g *Group, m *Module)           { g.setModule(m) }
func SetGroupStatus(g *Group, s Status)            { g.setStatus(s) }
func SetGroupDescription(g *Group, d string)       { g.setDescription(d) }
func SetGroupReference(g *Group, r string)         { g.setReference(r) }
func SetGroupStatusSpan(g *Group, s Span)          { g.setStatusSpan(s) }
func SetGroupDescriptionSpan(g *Group, s Span)     { g.setDescriptionSpan(s) }
func SetGroupReferenceSpan(g *Group, s Span)       { g.setReferenceSpan(s) }
func SetGroupOidRefs(g *Group, refs []OidRef)      { g.setOidRefs(refs) }
func AddGroupMember(g *Group, nd *Node)            { g.addMember(nd) }
func SetGroupIsNotificationGroup(g *Group, v bool) { g.setIsNotificationGroup(v) }

// --- Compliance entity setters ---

func SetComplianceSpan(c *Compliance, s Span)                        { c.setSpan(s) }
func SetComplianceNode(c *Compliance, nd *Node)                      { c.setNode(nd) }
func SetComplianceModule(c *Compliance, m *Module)                   { c.setModule(m) }
func SetComplianceStatus(c *Compliance, s Status)                    { c.setStatus(s) }
func SetComplianceDescription(c *Compliance, d string)               { c.setDescription(d) }
func SetComplianceReference(c *Compliance, r string)                 { c.setReference(r) }
func SetComplianceStatusSpan(c *Compliance, s Span)                  { c.setStatusSpan(s) }
func SetComplianceDescriptionSpan(c *Compliance, s Span)             { c.setDescriptionSpan(s) }
func SetComplianceReferenceSpan(c *Compliance, s Span)               { c.setReferenceSpan(s) }
func SetComplianceOidRefs(c *Compliance, refs []OidRef)              { c.setOidRefs(refs) }
func SetComplianceModules(c *Compliance, modules []ComplianceModule) { c.setModules(modules) }

// --- Capability entity setters ---

func SetCapabilitySpan(c *Capability, s Span)                     { c.setSpan(s) }
func SetCapabilityNode(c *Capability, nd *Node)                   { c.setNode(nd) }
func SetCapabilityModule(c *Capability, m *Module)                { c.setModule(m) }
func SetCapabilityStatus(c *Capability, s Status)                 { c.setStatus(s) }
func SetCapabilityDescription(c *Capability, d string)            { c.setDescription(d) }
func SetCapabilityReference(c *Capability, r string)              { c.setReference(r) }
func SetCapabilityStatusSpan(c *Capability, s Span)               { c.setStatusSpan(s) }
func SetCapabilityDescriptionSpan(c *Capability, s Span)          { c.setDescriptionSpan(s) }
func SetCapabilityReferenceSpan(c *Capability, s Span)            { c.setReferenceSpan(s) }
func SetCapabilityOidRefs(c *Capability, refs []OidRef)           { c.setOidRefs(refs) }
func SetCapabilityProductRelease(c *Capability, r string)         { c.setProductRelease(r) }
func SetCapabilitySupports(c *Capability, s []CapabilitiesModule) { c.setSupports(s) }

// --- Module setters ---

func SetModuleUsedImportNames(m *Module, u map[string]struct{}) { m.setUsedImportNames(u) }
func SetModuleResolvedImports(m *Module, ri map[string]*Module) { m.setResolvedImports(ri) }
func SetModuleLineTable(m *Module, t []int)                     { m.setLineTable(t) }
func SetModuleSourcePath(m *Module, path string)                { m.setSourcePath(path) }
func SetModuleBase(m *Module, b bool)                           { m.setBase(b) }
func SetModuleLanguage(m *Module, l Language)                   { m.setLanguage(l) }
func SetModuleOID(m *Module, oid OID)                           { m.setOID(oid) }
func SetModuleOrganization(m *Module, org string)               { m.setOrganization(org) }
func SetModuleContactInfo(m *Module, info string)               { m.setContactInfo(info) }
func SetModuleDescription(m *Module, desc string)               { m.setDescription(desc) }
func SetModuleLastUpdated(m *Module, s string)                  { m.setLastUpdated(s) }
func SetModuleRevisions(m *Module, revs []Revision)             { m.setRevisions(revs) }
func SetModuleRawImports(m *Module, imports []module.Import)    { m.setRawImports(imports) }
func AddModuleObject(m *Module, obj *Object)                    { m.addObject(obj) }
func AddModuleType(m *Module, t *Type)                          { m.addType(t) }
func AddModuleNotification(m *Module, n *Notification)          { m.addNotification(n) }
func AddModuleGroup(m *Module, g *Group)                        { m.addGroup(g) }
func AddModuleCompliance(m *Module, c *Compliance)              { m.addCompliance(c) }
func AddModuleCapability(m *Module, c *Capability)              { m.addCapability(c) }
func AddModuleNode(m *Module, n *Node)                          { m.addNode(n) }
func AddModuleNodeNamed(m *Module, name string, n *Node)        { m.addNodeNamed(name, n) }

// --- Type setters ---

func SetTypeSpan(t *Type, s Span)          { t.setSpan(s) }
func SetTypeSyntaxSpan(t *Type, s Span)    { t.setSyntaxSpan(s) }
func SetTypeModule(t *Type, m *Module)     { t.setModule(m) }
func SetTypeBase(t *Type, b BaseType)      { t.setBase(b) }
func SetTypeParent(t, p *Type)             { t.setParent(p) }
func SetTypeStatus(t *Type, s Status)      { t.setStatus(s) }
func SetTypeDisplayHint(t *Type, h string) { t.setDisplayHint(h) }
func SetTypeDescription(t *Type, d string) { t.setDescription(d) }
func SetTypeReference(t *Type, r string)   { t.setReference(r) }
func SetTypeIsTC(t *Type, b bool)          { t.setIsTC(b) }
func SetTypeSizes(t *Type, s []Range)      { t.setSizes(s) }
func SetTypeRanges(t *Type, r []Range)     { t.setRanges(r) }
func SetTypeEnums(t *Type, e []NamedValue) { t.setEnums(e) }
func SetTypeBits(t *Type, b []NamedValue)  { t.setBits(b) }
