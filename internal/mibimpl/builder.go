package mibimpl

import "github.com/golangsnmp/gomib/mib"

// Builder constructs a Mib incrementally.
// Use NewBuilder() to create a builder, add modules/objects/types,
// then call Mib() to get the final immutable Mib.
//
// This type is intended for internal use by the resolver.
type Builder struct {
	data *MibData
}

// NewBuilder creates a new Builder with an empty Mib.
func NewBuilder() *Builder {
	return &Builder{
		data: &MibData{
			root:         &Node{kind: mib.KindInternal},
			moduleByName: make(map[string]*Module),
			nameToNodes:  make(map[string][]*Node),
			typeByName:   make(map[string]*Type),
		},
	}
}

// Mib returns the constructed Mib as an interface.
// After calling this, the Builder should not be used further.
func (b *Builder) Mib() mib.Mib {
	return b.data
}

// Root returns the concrete pseudo-root of the OID tree for mutations.
func (b *Builder) Root() *Node {
	return b.data.root
}

// RegisterNode adds a node to the name index.
func (b *Builder) RegisterNode(name string, n *Node) {
	if name != "" {
		b.data.nameToNodes[name] = append(b.data.nameToNodes[name], n)
	}
}

// GetOrCreateNode returns the node at the given OID, creating nodes along the path as needed.
func (b *Builder) GetOrCreateNode(oid mib.Oid) *Node {
	nd := b.data.root
	for _, arc := range oid {
		nd = nd.GetOrCreateChild(arc)
	}
	return nd
}

// GetOrCreateRoot returns the root node with the given arc (0, 1, or 2), creating if needed.
func (b *Builder) GetOrCreateRoot(arc uint32) *Node {
	return b.data.root.GetOrCreateChild(arc)
}

// AddModule adds a module to the Mib.
func (b *Builder) AddModule(mod *Module) {
	b.data.modules = append(b.data.modules, mod)
	if mod.name != "" {
		b.data.moduleByName[mod.name] = mod
	}
}

// AddType adds a type to the Mib.
func (b *Builder) AddType(t *Type) {
	b.data.types = append(b.data.types, t)
	if t.name != "" && b.data.typeByName[t.name] == nil {
		b.data.typeByName[t.name] = t
	}
}

// AddObject adds an object to the Mib.
func (b *Builder) AddObject(obj *Object) {
	b.data.objects = append(b.data.objects, obj)
}

// AddNotification adds a notification to the Mib.
func (b *Builder) AddNotification(n *Notification) {
	b.data.notifications = append(b.data.notifications, n)
}

// AddUnresolved adds an unresolved reference.
func (b *Builder) AddUnresolved(ref mib.UnresolvedRef) {
	b.data.unresolved = append(b.data.unresolved, ref)
}

// AddDiagnostic adds a diagnostic message.
func (b *Builder) AddDiagnostic(d mib.Diagnostic) {
	b.data.diagnostics = append(b.data.diagnostics, d)
}

// ModuleCount returns the number of modules.
func (b *Builder) ModuleCount() int {
	return len(b.data.modules)
}

// TypeCount returns the number of types.
func (b *Builder) TypeCount() int {
	return len(b.data.types)
}

// ObjectCount returns the number of objects.
func (b *Builder) ObjectCount() int {
	return len(b.data.objects)
}

// NotificationCount returns the number of notifications.
func (b *Builder) NotificationCount() int {
	return len(b.data.notifications)
}

// NodeCount returns the total number of nodes in the tree.
func (b *Builder) NodeCount() int {
	count := 0
	for range b.data.Nodes() {
		count++
	}
	return count
}

// Types returns all types for iteration during resolution.
func (b *Builder) Types() []*Type {
	return b.data.types
}

// Module returns the concrete module by name for resolver use.
func (b *Builder) Module(name string) *Module {
	return b.data.moduleByName[name]
}

// NewModule creates a new module with the given name.
func NewModule(name string) *Module {
	return &Module{name: name}
}

// NewObject creates a new object with the given name.
func NewObject(name string) *Object {
	return &Object{name: name}
}

// NewType creates a new type with the given name.
func NewType(name string) *Type {
	return &Type{name: name}
}

// NewNotification creates a new notification with the given name.
func NewNotification(name string) *Notification {
	return &Notification{name: name}
}

// EmptyMib returns a new empty Mib.
func EmptyMib() mib.Mib {
	return NewBuilder().Mib()
}
