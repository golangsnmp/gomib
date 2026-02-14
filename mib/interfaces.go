package mib

import "iter"

// Mib is the top-level container for loaded MIB data.
// It is immutable after construction and safe for concurrent reads.
//
// Collection methods return slices, except Nodes() and Node.Descendants()
// which return iter.Seq[Node] because the full node set can be large and
// callers typically filter or stop early.
type Mib interface {
	// Tree access
	Root() Node
	Nodes() iter.Seq[Node]

	// Lookups by name, OID string, or qualified name (e.g. "IF-MIB::ifIndex").
	// All Find* methods return nil if no match is found.
	FindNode(query string) Node
	FindObject(query string) Object
	FindType(query string) Type
	FindNotification(query string) Notification
	FindGroup(query string) Group
	FindCompliance(query string) Compliance
	FindCapabilities(query string) Capabilities

	// By OID value. Returns nil if no matching node exists.
	NodeByOID(oid Oid) Node
	LongestPrefixByOID(oid Oid) Node

	// Module access. Returns nil if no module with the given name is loaded.
	Module(name string) Module
	Modules() []Module

	// Collection access
	Objects() []Object
	Types() []Type
	Notifications() []Notification
	Groups() []Group
	Compliances() []Compliance
	Capabilities() []Capabilities

	// Filtered collections
	Tables() []Object
	Scalars() []Object
	Columns() []Object
	Rows() []Object

	// Counts
	ModuleCount() int
	ObjectCount() int
	TypeCount() int
	NotificationCount() int
	GroupCount() int
	ComplianceCount() int
	CapabilitiesCount() int
	NodeCount() int

	// Diagnostics
	Unresolved() []UnresolvedRef
	Diagnostics() []Diagnostic
	HasErrors() bool // any diagnostic at Error or above
}

// Node is a point in the OID tree.
type Node interface {
	Arc() uint32
	Name() string
	OID() Oid
	Kind() Kind
	Module() Module

	// Associated definitions, nil if none.
	Object() Object
	Notification() Notification
	Group() Group
	Compliance() Compliance
	Capabilities() Capabilities

	// Tree navigation. Parent returns nil for the root node.
	Parent() Node
	// Child returns nil if no child with the given arc exists.
	Child(arc uint32) Node
	Children() []Node
	// Descendants returns all nodes in the subtree via lazy iteration.
	Descendants() iter.Seq[Node]

	// Prefix matching. Returns nil if no prefix matches.
	LongestPrefix(oid Oid) Node

	// Predicates
	IsRoot() bool
}

// Module represents a loaded and resolved MIB module.
type Module interface {
	Name() string
	Language() Language
	OID() Oid
	Organization() string
	ContactInfo() string
	Description() string
	Revisions() []Revision

	// Collections
	Objects() []Object
	Types() []Type
	Notifications() []Notification
	Groups() []Group
	Compliances() []Compliance
	Capabilities() []Capabilities

	// Filtered collections
	Tables() []Object
	Scalars() []Object
	Columns() []Object
	Rows() []Object

	// Lookups. All return nil if no definition with the given name exists.
	Node(name string) Node
	Object(name string) Object
	Type(name string) Type
	Notification(name string) Notification
	Group(name string) Group
	Compliance(name string) Compliance
	CapabilitiesByName(name string) Capabilities
}

// Object is an OBJECT-TYPE definition.
type Object interface {
	Name() string
	Node() Node
	Module() Module
	OID() Oid

	// Type info. Type returns nil if the type could not be resolved.
	Type() Type
	Kind() Kind
	Access() Access
	Status() Status

	// Metadata
	Description() string
	Reference() string
	Units() string
	DefaultValue() DefVal

	// Table structure - navigation.
	// Table returns the containing table for a row or column, nil otherwise.
	Table() Object
	// Row returns the containing row for a column, nil otherwise.
	Row() Object
	// Entry returns the row object for a table, nil for non-tables.
	Entry() Object
	Columns() []Object

	// Table structure - index.
	// Augments returns the augmented row, nil if not an augmenting row.
	Augments() Object
	Index() []IndexEntry
	EffectiveIndexes() []IndexEntry

	// Effective constraints (from inline + type chain)
	EffectiveDisplayHint() string
	EffectiveSizes() []Range
	EffectiveRanges() []Range
	EffectiveEnums() []NamedValue
	EffectiveBits() []NamedValue

	// Lookups into effective constraints
	Enum(label string) (NamedValue, bool)
	Bit(label string) (NamedValue, bool)

	// Predicates
	IsTable() bool
	IsRow() bool
	IsColumn() bool
	IsScalar() bool
}

// Type is a type definition (textual convention or type reference).
type Type interface {
	Name() string
	Module() Module
	Base() BaseType
	// Parent returns the parent type in the type chain, nil for base types.
	Parent() Type
	Status() Status

	// Metadata
	DisplayHint() string
	Description() string
	Reference() string

	// Direct constraints
	Sizes() []Range
	Ranges() []Range
	Enums() []NamedValue
	Bits() []NamedValue

	// Lookups
	Enum(label string) (NamedValue, bool)
	Bit(label string) (NamedValue, bool)

	// Classification predicates
	IsTextualConvention() bool
	IsCounter() bool
	IsGauge() bool
	IsString() bool
	IsEnumeration() bool
	IsBits() bool

	// Effective (walks type chain)
	EffectiveBase() BaseType
	EffectiveDisplayHint() string
	EffectiveSizes() []Range
	EffectiveRanges() []Range
	EffectiveEnums() []NamedValue
	EffectiveBits() []NamedValue
}

// Notification is a NOTIFICATION-TYPE or TRAP-TYPE.
type Notification interface {
	Name() string
	Node() Node
	Module() Module
	OID() Oid
	Status() Status
	Description() string
	Reference() string
	Objects() []Object
}

// Group is an OBJECT-GROUP or NOTIFICATION-GROUP definition.
type Group interface {
	Name() string
	Node() Node
	Module() Module
	OID() Oid
	Status() Status
	Description() string
	Reference() string

	// Members returns the resolved OID tree nodes for each member
	// of this group (objects for OBJECT-GROUP, notifications for
	// NOTIFICATION-GROUP).
	Members() []Node

	// IsNotificationGroup reports whether this is a NOTIFICATION-GROUP.
	IsNotificationGroup() bool
}

// Compliance is a MODULE-COMPLIANCE definition.
type Compliance interface {
	Name() string
	Node() Node
	Module() Module
	OID() Oid
	Status() Status
	Description() string
	Reference() string

	// Modules returns the MODULE clauses in this compliance statement.
	Modules() []ComplianceModule
}

// Capabilities is an AGENT-CAPABILITIES definition.
type Capabilities interface {
	Name() string
	Node() Node
	Module() Module
	OID() Oid
	Status() Status
	Description() string
	Reference() string
	ProductRelease() string

	// Supports returns the SUPPORTS clauses in this capabilities statement.
	Supports() []CapabilitiesModule
}
