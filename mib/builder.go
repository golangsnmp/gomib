package mib

// Builder constructs a Mib incrementally.
// Use NewBuilder() to create a builder, add modules/objects/types,
// then call Mib() to get the final immutable Mib.
//
// This type is intended for internal use by the resolver.
// Most users should use the Load functions from the gomib package instead.
type Builder struct {
	mib *Mib
}

// NewBuilder creates a new Builder with an empty Mib.
func NewBuilder() *Builder {
	return &Builder{mib: NewMib()}
}

// Mib returns the constructed Mib.
// After calling this, the Builder should not be used further.
func (b *Builder) Mib() *Mib {
	return b.mib
}

// Root returns the pseudo-root of the OID tree.
func (b *Builder) Root() *Node {
	return b.mib.root
}

// RegisterNode adds a node to the name index.
func (b *Builder) RegisterNode(name string, node *Node) {
	if name != "" {
		b.mib.nameToNodes[name] = append(b.mib.nameToNodes[name], node)
	}
}

// GetOrCreateNode returns the node at the given OID, creating nodes along the path as needed.
func (b *Builder) GetOrCreateNode(oid Oid) *Node {
	node := b.mib.root
	for _, arc := range oid {
		node = node.GetOrCreateChild(arc)
	}
	return node
}

// GetOrCreateRoot returns the root node with the given arc (0, 1, or 2), creating if needed.
func (b *Builder) GetOrCreateRoot(arc uint32) *Node {
	return b.mib.root.GetOrCreateChild(arc)
}

// AddModule adds a module to the Mib.
func (b *Builder) AddModule(mod *Module) {
	b.mib.modules = append(b.mib.modules, mod)
	if mod.Name != "" {
		b.mib.moduleByName[mod.Name] = mod
	}
}

// AddType adds a type to the Mib.
func (b *Builder) AddType(t *Type) {
	b.mib.types = append(b.mib.types, t)
	if t.Name != "" && b.mib.typeByName[t.Name] == nil {
		b.mib.typeByName[t.Name] = t
	}
}

// AddObject adds an object to the Mib.
func (b *Builder) AddObject(obj *Object) {
	b.mib.objects = append(b.mib.objects, obj)
}

// AddNotification adds a notification to the Mib.
func (b *Builder) AddNotification(n *Notification) {
	b.mib.notifications = append(b.mib.notifications, n)
}

// AddUnresolved adds an unresolved reference.
func (b *Builder) AddUnresolved(ref UnresolvedRef) {
	b.mib.unresolved = append(b.mib.unresolved, ref)
}

// AddDiagnostic adds a diagnostic message.
func (b *Builder) AddDiagnostic(d Diagnostic) {
	b.mib.diagnostics = append(b.mib.diagnostics, d)
}

// ModuleCount returns the number of modules.
func (b *Builder) ModuleCount() int {
	return len(b.mib.modules)
}

// TypeCount returns the number of types.
func (b *Builder) TypeCount() int {
	return len(b.mib.types)
}

// ObjectCount returns the number of objects.
func (b *Builder) ObjectCount() int {
	return len(b.mib.objects)
}

// NotificationCount returns the number of notifications.
func (b *Builder) NotificationCount() int {
	return len(b.mib.notifications)
}

// NodeCount returns the total number of nodes in the tree.
func (b *Builder) NodeCount() int {
	count := 0
	b.mib.root.Walk(func(*Node) bool {
		count++
		return true
	})
	return count
}

// Types returns all types for iteration during resolution.
func (b *Builder) Types() []*Type {
	return b.mib.types
}
