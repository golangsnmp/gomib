package mibimpl

import "github.com/golangsnmp/gomib/mib"

// Builder constructs a Mib incrementally during resolution.
type Builder struct {
	data *Data
}

// NewBuilder returns a Builder with an initialized, empty Mib.
func NewBuilder() *Builder {
	return &Builder{
		data: &Data{
			root:         &Node{kind: mib.KindInternal},
			moduleByName: make(map[string]*Module),
			nameToNodes:  make(map[string][]*Node),
			typeByName:   make(map[string]*Type),
		},
	}
}

// Mib returns the constructed Mib. The Builder must not be used after
// this call; subsequent method calls will panic.
func (b *Builder) Mib() mib.Mib {
	d := b.data
	d.nodeCount = b.countNodes()
	b.data = nil
	return d
}

func (b *Builder) countNodes() int {
	count := 0
	for range b.data.Nodes() {
		count++
	}
	return count
}

// Root returns the pseudo-root of the OID tree.
func (b *Builder) Root() *Node {
	return b.data.root
}

// RegisterNode indexes a named node for lookup by name.
func (b *Builder) RegisterNode(name string, n *Node) {
	if name != "" {
		b.data.nameToNodes[name] = append(b.data.nameToNodes[name], n)
	}
}

// GetOrCreateNode walks the OID tree, creating intermediate nodes as needed.
func (b *Builder) GetOrCreateNode(oid mib.OID) *Node {
	nd := b.data.root
	for _, arc := range oid {
		nd = nd.GetOrCreateChild(arc)
	}
	return nd
}

// GetOrCreateRoot returns a top-level node (arc 0, 1, or 2), creating
// it if needed.
func (b *Builder) GetOrCreateRoot(arc uint32) *Node {
	return b.data.root.GetOrCreateChild(arc)
}

// AddModule registers a resolved module.
func (b *Builder) AddModule(mod *Module) {
	b.data.modules = append(b.data.modules, mod)
	if mod.name != "" {
		b.data.moduleByName[mod.name] = mod
	}
}

// AddType registers a resolved type, indexing the first occurrence by name.
func (b *Builder) AddType(t *Type) {
	b.data.types = append(b.data.types, t)
	if t.name != "" && b.data.typeByName[t.name] == nil {
		b.data.typeByName[t.name] = t
	}
}

// AddObject registers a resolved object.
func (b *Builder) AddObject(obj *Object) {
	b.data.objects = append(b.data.objects, obj)
}

// AddNotification registers a resolved notification.
func (b *Builder) AddNotification(n *Notification) {
	b.data.notifications = append(b.data.notifications, n)
}

// AddGroup registers a resolved group.
func (b *Builder) AddGroup(g *Group) {
	b.data.groups = append(b.data.groups, g)
}

// AddCompliance registers a resolved compliance statement.
func (b *Builder) AddCompliance(c *Compliance) {
	b.data.compliances = append(b.data.compliances, c)
}

// AddCapability registers a resolved agent capability statement.
func (b *Builder) AddCapability(c *Capability) {
	b.data.capabilities = append(b.data.capabilities, c)
}

// AddUnresolved records a reference that could not be resolved.
func (b *Builder) AddUnresolved(ref mib.UnresolvedRef) {
	b.data.unresolved = append(b.data.unresolved, ref)
}

// AddDiagnostic records a warning or error encountered during resolution.
func (b *Builder) AddDiagnostic(d mib.Diagnostic) {
	b.data.diagnostics = append(b.data.diagnostics, d)
}

// ModuleCount reports the number of registered modules.
func (b *Builder) ModuleCount() int {
	return len(b.data.modules)
}

// TypeCount reports the number of registered types.
func (b *Builder) TypeCount() int {
	return len(b.data.types)
}

// ObjectCount reports the number of registered objects.
func (b *Builder) ObjectCount() int {
	return len(b.data.objects)
}

// NotificationCount reports the number of registered notifications.
func (b *Builder) NotificationCount() int {
	return len(b.data.notifications)
}

// GroupCount reports the number of registered groups.
func (b *Builder) GroupCount() int {
	return len(b.data.groups)
}

// ComplianceCount reports the number of registered compliance statements.
func (b *Builder) ComplianceCount() int {
	return len(b.data.compliances)
}

// CapabilityCount reports the number of registered capabilities.
func (b *Builder) CapabilityCount() int {
	return len(b.data.capabilities)
}

// NodeCount walks the OID tree and returns the total node count.
func (b *Builder) NodeCount() int {
	count := 0
	for range b.data.Nodes() {
		count++
	}
	return count
}

// Types returns all registered types.
func (b *Builder) Types() []*Type {
	return b.data.types
}

// Module looks up a module by name.
func (b *Builder) Module(name string) *Module {
	return b.data.moduleByName[name]
}

// NewModule returns a Module initialized with the given name.
func NewModule(name string) *Module {
	return &Module{name: name}
}

// NewObject returns an Object initialized with the given name.
func NewObject(name string) *Object {
	return &Object{name: name}
}

// NewType returns a Type initialized with the given name.
func NewType(name string) *Type {
	return &Type{name: name}
}

// NewNotification returns a Notification initialized with the given name.
func NewNotification(name string) *Notification {
	return &Notification{name: name}
}

// NewGroup returns a Group initialized with the given name.
func NewGroup(name string) *Group {
	return &Group{name: name}
}

// NewCompliance returns a Compliance initialized with the given name.
func NewCompliance(name string) *Compliance {
	return &Compliance{name: name}
}

// NewCapability returns a Capability initialized with the given name.
func NewCapability(name string) *Capability {
	return &Capability{name: name}
}

// EmptyMib returns a Mib with no modules or definitions loaded.
func EmptyMib() mib.Mib {
	return NewBuilder().Mib()
}
