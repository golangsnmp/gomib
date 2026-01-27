package mib

import "iter"

// Mib is the top-level container for loaded MIB data.
// It is immutable after construction and safe for concurrent reads.
type Mib interface {
	// Tree access
	Root() Node
	Nodes() iter.Seq[Node]

	// Lookups - Find* methods handle name, OID string, qualified name
	FindNode(query string) Node
	FindObject(query string) Object
	FindType(query string) Type
	FindNotification(query string) Notification

	// By OID (for when you already have an Oid value)
	NodeByOID(oid Oid) Node
	LongestPrefixByOID(oid Oid) Node

	// Module access
	Module(name string) Module
	Modules() []Module

	// Collection access
	Objects() []Object
	Types() []Type
	Notifications() []Notification

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

	// Associated definitions (nil if none)
	Object() Object
	Notification() Notification

	// Tree navigation
	Parent() Node
	Child(arc uint32) Node
	Children() []Node
	Descendants() iter.Seq[Node]

	// Prefix matching
	LongestPrefix(oid Oid) Node

	// Predicates
	IsRoot() bool
}

// Module is a MIB module.
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

	// Filtered collections
	Tables() []Object
	Scalars() []Object
	Columns() []Object
	Rows() []Object

	// Lookups
	Node(name string) Node
	Object(name string) Object
	Type(name string) Type
	Notification(name string) Notification
}

// Object is an OBJECT-TYPE definition.
type Object interface {
	Name() string
	Node() Node
	Module() Module
	OID() Oid

	// Type info
	Type() Type
	Kind() Kind
	Access() Access
	Status() Status

	// Metadata
	Description() string
	Reference() string
	Units() string
	DefaultValue() DefVal

	// Table structure - navigation
	Table() Object
	Row() Object
	Entry() Object
	Columns() []Object

	// Table structure - index
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
