// Package mibimpl provides a Builder for constructing a Mib incrementally.
package mibimpl

import "github.com/golangsnmp/gomib/mib"

// Builder constructs a Mib incrementally during resolution.
type Builder struct {
	m *mib.Mib
}

// NewBuilder returns a Builder with an initialized, empty Mib.
func NewBuilder() *Builder {
	return &Builder{m: mib.NewMib()}
}

// Mib returns the constructed Mib. The Builder must not be used after
// this call; subsequent method calls will panic.
func (b *Builder) Mib() *mib.Mib {
	m := b.m
	m.SetNodeCount(b.countNodes())
	b.m = nil
	return m
}

func (b *Builder) countNodes() int {
	count := 0
	for range b.m.Nodes() {
		count++
	}
	return count
}

// Root returns the pseudo-root of the OID tree.
func (b *Builder) Root() *mib.Node {
	return b.m.Root()
}

// RegisterNode indexes a named node for lookup by name.
func (b *Builder) RegisterNode(name string, n *mib.Node) {
	b.m.RegisterNode(name, n)
}

// GetOrCreateNode walks the OID tree, creating intermediate nodes as needed.
func (b *Builder) GetOrCreateNode(oid mib.OID) *mib.Node {
	nd := b.m.Root()
	for _, arc := range oid {
		nd = nd.GetOrCreateChild(arc)
	}
	return nd
}

// GetOrCreateRoot returns a top-level node (arc 0, 1, or 2), creating
// it if needed.
func (b *Builder) GetOrCreateRoot(arc uint32) *mib.Node {
	return b.m.Root().GetOrCreateChild(arc)
}

func (b *Builder) AddModule(mod *mib.Module)           { b.m.AddModule(mod) }
func (b *Builder) AddType(t *mib.Type)                 { b.m.AddType(t) }
func (b *Builder) AddObject(obj *mib.Object)           { b.m.AddObject(obj) }
func (b *Builder) AddNotification(n *mib.Notification) { b.m.AddNotification(n) }
func (b *Builder) AddGroup(g *mib.Group)               { b.m.AddGroup(g) }
func (b *Builder) AddCompliance(c *mib.Compliance)     { b.m.AddCompliance(c) }
func (b *Builder) AddCapability(c *mib.Capability)     { b.m.AddCapability(c) }
func (b *Builder) AddUnresolved(ref mib.UnresolvedRef) { b.m.AddUnresolved(ref) }
func (b *Builder) AddDiagnostic(d mib.Diagnostic)      { b.m.AddDiagnostic(d) }

// Modules returns all registered modules.
func (b *Builder) Modules() []*mib.Module { return b.m.Modules() }

// NodeCount walks the OID tree and returns the total node count.
func (b *Builder) NodeCount() int {
	count := 0
	for range b.m.Nodes() {
		count++
	}
	return count
}

// Types returns all registered types.
func (b *Builder) Types() []*mib.Type {
	return b.m.Types()
}

// Module looks up a module by name.
func (b *Builder) Module(name string) *mib.Module {
	return b.m.Module(name)
}

// EmptyMib returns a Mib with no modules or definitions loaded.
func EmptyMib() *mib.Mib {
	return NewBuilder().Mib()
}
