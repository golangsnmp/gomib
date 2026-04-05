package mib

import (
	"cmp"
	"iter"
	"slices"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// Module represents a loaded and resolved MIB module.
type Module struct {
	name         string
	language     Language
	sourcePath   string
	base         bool
	oid          OID
	organization string
	contactInfo  string
	description  string
	lastUpdated  string
	revisions    []Revision
	rawImports   []module.Import // flat per-symbol imports from parser

	objects       []*Object
	types         []*Type
	notifications []*Notification
	groups        []*Group
	compliances   []*Compliance
	capabilities  []*Capability
	nodes         []*Node

	lineTable []int

	// usedImportNames records which imported symbols were referenced during
	// resolution. Populated by the resolver after all phases complete.
	usedImportNames map[string]struct{}

	// resolvedImports maps imported symbol names to the resolved Module
	// that defines them. Populated by the resolver after import resolution.
	resolvedImports map[string]*Module

	// Name-indexed maps for O(1) lookups, populated by Add*() methods.
	objectsByName       map[string]*Object
	typesByName         map[string]*Type
	notificationsByName map[string]*Notification
	groupsByName        map[string]*Group
	compliancesByName   map[string]*Compliance
	capabilitiesByName  map[string]*Capability
	nodesByName         map[string]*Node
}

// newModule returns a Module initialized with the given name.
func newModule(name string) *Module {
	return &Module{
		name:                name,
		objectsByName:       make(map[string]*Object),
		typesByName:         make(map[string]*Type),
		notificationsByName: make(map[string]*Notification),
		groupsByName:        make(map[string]*Group),
		compliancesByName:   make(map[string]*Compliance),
		capabilitiesByName:  make(map[string]*Capability),
		nodesByName:         make(map[string]*Node),
	}
}

// Name returns the module name (e.g. "IF-MIB").
func (m *Module) Name() string { return m.name }

// Language returns the SMI language version of this module.
func (m *Module) Language() Language { return m.language }

// SourcePath returns the file path this module was loaded from, or "" for synthetic modules.
func (m *Module) SourcePath() string { return m.sourcePath }

// IsBase reports whether this is a synthetic base module (e.g. SNMPv2-SMI).
func (m *Module) IsBase() bool { return m.base }

// LineCol converts a byte offset into 1-based line and column numbers using
// this module's line table. Returns (0, 0) for synthetic modules or invalid offsets.
func (m *Module) LineCol(offset ByteOffset) (line, col int) {
	return types.LineColFromTable(m.lineTable, offset)
}

// OID returns the MODULE-IDENTITY OID, or nil if not declared.
func (m *Module) OID() OID { return slices.Clone(m.oid) }

// Organization returns the ORGANIZATION clause text, or "".
func (m *Module) Organization() string { return m.organization }

// ContactInfo returns the CONTACT-INFO clause text, or "".
func (m *Module) ContactInfo() string { return m.contactInfo }

// Description returns the DESCRIPTION clause text.
func (m *Module) Description() string { return m.description }

// LastUpdated returns the LAST-UPDATED clause value, or "".
func (m *Module) LastUpdated() string { return m.lastUpdated }

// Revisions returns the REVISION clauses in declaration order.
func (m *Module) Revisions() []Revision { return slices.Clone(m.revisions) }

// Imports returns the IMPORTS declarations for this module, grouped by source
// module. Symbols within each group are sorted alphabetically. Module order
// follows first occurrence in the flat import list.
func (m *Module) Imports() []Import {
	if len(m.rawImports) == 0 {
		return nil
	}
	var order []string
	grouped := make(map[string][]ImportSymbol)
	for _, imp := range m.rawImports {
		if _, ok := grouped[imp.Module]; !ok {
			order = append(order, imp.Module)
		}
		grouped[imp.Module] = append(grouped[imp.Module], ImportSymbol{
			Name: imp.Symbol,
			Span: imp.Span,
		})
	}
	result := make([]Import, 0, len(order))
	for _, modName := range order {
		syms := grouped[modName]
		slices.SortFunc(syms, func(a, b ImportSymbol) int {
			return cmp.Compare(a.Name, b.Name)
		})
		result = append(result, Import{Module: modName, Symbols: syms})
	}
	return result
}

// Objects returns all OBJECT-TYPE definitions in this module.
func (m *Module) Objects() []*Object { return slices.Clone(m.objects) }

// Types returns all type definitions in this module.
func (m *Module) Types() []*Type { return slices.Clone(m.types) }

// Notifications returns all NOTIFICATION-TYPE and TRAP-TYPE definitions in this module.
func (m *Module) Notifications() []*Notification { return slices.Clone(m.notifications) }

// Groups returns all OBJECT-GROUP and NOTIFICATION-GROUP definitions in this module.
func (m *Module) Groups() []*Group { return slices.Clone(m.groups) }

// Compliances returns all MODULE-COMPLIANCE definitions in this module.
func (m *Module) Compliances() []*Compliance { return slices.Clone(m.compliances) }

// Capabilities returns all AGENT-CAPABILITIES definitions in this module.
func (m *Module) Capabilities() []*Capability { return slices.Clone(m.capabilities) }

// Nodes returns all OID tree nodes registered by this module.
func (m *Module) Nodes() []*Node { return slices.Clone(m.nodes) }

// Tables returns the OBJECT-TYPE definitions classified as tables.
func (m *Module) Tables() []*Object { return objectsByKind(m.objects, KindTable) }

// Scalars returns the OBJECT-TYPE definitions classified as scalars.
func (m *Module) Scalars() []*Object { return objectsByKind(m.objects, KindScalar) }

// Columns returns the OBJECT-TYPE definitions classified as table columns.
func (m *Module) Columns() []*Object { return objectsByKind(m.objects, KindColumn) }

// Rows returns the OBJECT-TYPE definitions classified as table rows.
func (m *Module) Rows() []*Object { return objectsByKind(m.objects, KindRow) }

// Node returns the node with the given name in this module, or nil if not found.
func (m *Module) Node(name string) *Node {
	return m.nodesByName[name]
}

// Object returns the object with the given name in this module, or nil if not found.
func (m *Module) Object(name string) *Object {
	return m.objectsByName[name]
}

// Type returns the type with the given name in this module, or nil if not found.
func (m *Module) Type(name string) *Type {
	return m.typesByName[name]
}

// Notification returns the notification with the given name in this module, or nil if not found.
func (m *Module) Notification(name string) *Notification {
	return m.notificationsByName[name]
}

// Group returns the group with the given name in this module, or nil if not found.
func (m *Module) Group(name string) *Group {
	return m.groupsByName[name]
}

// Compliance returns the compliance with the given name in this module, or nil if not found.
func (m *Module) Compliance(name string) *Compliance {
	return m.compliancesByName[name]
}

// Capability returns the capability with the given name in this module, or nil if not found.
func (m *Module) Capability(name string) *Capability {
	return m.capabilitiesByName[name]
}

// Definitions returns an iterator over all definitions in this module as Symbol
// values. Yields Objects, Types, Notifications, Groups, Compliances,
// Capabilities, then plain Nodes (nodes not covered by another entity type).
func (m *Module) Definitions() iter.Seq[Symbol] {
	return func(yield func(Symbol) bool) {
		for _, o := range m.objects {
			if !yield(symbolFromObject(o)) {
				return
			}
		}
		for _, t := range m.types {
			if !yield(symbolFromType(t)) {
				return
			}
		}
		for _, n := range m.notifications {
			if !yield(symbolFromNotification(n)) {
				return
			}
		}
		for _, g := range m.groups {
			if !yield(symbolFromGroup(g)) {
				return
			}
		}
		for _, c := range m.compliances {
			if !yield(symbolFromCompliance(c)) {
				return
			}
		}
		for _, c := range m.capabilities {
			if !yield(symbolFromCapability(c)) {
				return
			}
		}
		for _, nd := range m.nodes {
			if nd.obj != nil || nd.notif != nil || nd.group != nil || nd.compliance != nil || nd.capability != nil {
				continue
			}
			if !yield(symbolFromNode(nd)) {
				return
			}
		}
	}
}

// DefinesSymbol reports whether this module defines a symbol with the given name.
func (m *Module) DefinesSymbol(name string) bool {
	return !m.Symbol(name).IsZero()
}

// ImportsSymbol reports whether this module's IMPORTS clause includes the given name.
func (m *Module) ImportsSymbol(name string) bool {
	for _, imp := range m.rawImports {
		if imp.Symbol == name {
			return true
		}
	}
	return false
}

// IsImportUsed reports whether the named imported symbol was referenced
// during resolution. Returns false for non-imported or unknown names.
func (m *Module) IsImportUsed(name string) bool {
	if m.usedImportNames == nil {
		return false
	}
	_, ok := m.usedImportNames[name]
	return ok
}

// Symbol returns the resolved definition for the given name in this module.
// Lookup priority matches Definitions(): objects, types, notifications, groups,
// compliances, capabilities, then plain nodes. Returns a zero Symbol if not found.
func (m *Module) Symbol(name string) Symbol {
	if o := m.objectsByName[name]; o != nil {
		return symbolFromObject(o)
	}
	if t := m.typesByName[name]; t != nil {
		return symbolFromType(t)
	}
	if n := m.notificationsByName[name]; n != nil {
		return symbolFromNotification(n)
	}
	if g := m.groupsByName[name]; g != nil {
		return symbolFromGroup(g)
	}
	if c := m.compliancesByName[name]; c != nil {
		return symbolFromCompliance(c)
	}
	if c := m.capabilitiesByName[name]; c != nil {
		return symbolFromCapability(c)
	}
	if n := m.nodesByName[name]; n != nil {
		return symbolFromNode(n)
	}
	return Symbol{}
}

// ImportSource returns the resolved source Module for an imported symbol,
// or nil if the name is not an import.
func (m *Module) ImportSource(name string) *Module {
	if m.resolvedImports == nil {
		return nil
	}
	return m.resolvedImports[name]
}

// AvailableSymbols returns an iterator over all symbols available in this
// module's scope: own definitions first, then imported symbols looked up
// from their source modules. Imported symbols follow IMPORTS declaration
// order. Names that are also own definitions are yielded only once (as the
// own definition).
func (m *Module) AvailableSymbols() iter.Seq[Symbol] {
	return func(yield func(Symbol) bool) {
		// Own definitions first.
		seen := make(map[string]struct{})
		for sym := range m.Definitions() {
			seen[sym.Name()] = struct{}{}
			if !yield(sym) {
				return
			}
		}
		// Imported symbols in IMPORTS declaration order.
		if m.resolvedImports == nil {
			return
		}
		for _, imp := range m.rawImports {
			if _, already := seen[imp.Symbol]; already {
				continue
			}
			seen[imp.Symbol] = struct{}{}
			sourceMod := m.resolvedImports[imp.Symbol]
			if sourceMod == nil {
				continue
			}
			sym := sourceMod.Symbol(imp.Symbol)
			if sym.IsZero() {
				continue
			}
			if !yield(sym) {
				return
			}
		}
	}
}

// ExportedSymbols returns an iterator over the symbols exported by this module.
// In SMI all definitions are implicitly exported (no EXPORTS clause), so this
// is semantically identical to Definitions() but provides a named API matching
// the intent.
func (m *Module) ExportedSymbols() iter.Seq[Symbol] {
	return m.Definitions()
}

func (m *Module) setUsedImportNames(u map[string]struct{}) { m.usedImportNames = u }
func (m *Module) setResolvedImports(ri map[string]*Module) { m.resolvedImports = ri }
func (m *Module) setLineTable(t []int)                     { m.lineTable = t }
func (m *Module) setSourcePath(path string)                { m.sourcePath = path }
func (m *Module) setBase(b bool)                           { m.base = b }
func (m *Module) setLanguage(l Language)                   { m.language = l }
func (m *Module) setOID(oid OID)                           { m.oid = oid }
func (m *Module) setOrganization(org string)               { m.organization = org }
func (m *Module) setContactInfo(info string)               { m.contactInfo = info }
func (m *Module) setDescription(desc string)               { m.description = desc }
func (m *Module) setLastUpdated(s string)                  { m.lastUpdated = s }
func (m *Module) setRevisions(revs []Revision)             { m.revisions = revs }
func (m *Module) setRawImports(imports []module.Import)    { m.rawImports = imports }

func (m *Module) addObject(obj *Object) {
	m.objects = append(m.objects, obj)
	if m.objectsByName[obj.name] == nil {
		m.objectsByName[obj.name] = obj
	}
}

// addType uses first-write-wins for the name index, like all add methods.
// Multiple modules can define the same name (e.g. DisplayString in both
// RFC1213-MIB and SNMPv2-TC). The slice holds all definitions; the map
// keeps the first.
func (m *Module) addType(t *Type) {
	m.types = append(m.types, t)
	if m.typesByName[t.name] == nil {
		m.typesByName[t.name] = t
	}
}

func (m *Module) addNotification(n *Notification) {
	m.notifications = append(m.notifications, n)
	if m.notificationsByName[n.name] == nil {
		m.notificationsByName[n.name] = n
	}
}

func (m *Module) addGroup(g *Group) {
	m.groups = append(m.groups, g)
	if m.groupsByName[g.name] == nil {
		m.groupsByName[g.name] = g
	}
}

func (m *Module) addCompliance(c *Compliance) {
	m.compliances = append(m.compliances, c)
	if m.compliancesByName[c.name] == nil {
		m.compliancesByName[c.name] = c
	}
}

func (m *Module) addCapability(c *Capability) {
	m.capabilities = append(m.capabilities, c)
	if m.capabilitiesByName[c.name] == nil {
		m.capabilitiesByName[c.name] = c
	}
}

func (m *Module) addNode(n *Node) {
	m.nodes = append(m.nodes, n)
	if m.nodesByName[n.name] == nil {
		m.nodesByName[n.name] = n
	}
}
