package mib

import (
	"cmp"
	"iter"
	"slices"
	"strconv"
	"strings"
)

// Node is a point in the OID tree.
// The tree has a single pseudo-root node (where Parent == nil and OID is empty).
// All other nodes have a parent and their OID is computed from the parent chain.
type Node struct {
	arc      uint32 // arc value from parent; 0 for root
	Name     string // primary label; empty for root and unnamed nodes
	Kind     Kind   // inferred node kind
	Parent   *Node  // nil only for root
	children map[uint32]*Node
	Module   *Module       // defining module, nil for root/internal
	Object   *Object       // nil if not an object node
	Notif    *Notification // nil if not a notification node
}

// Arc returns the arc value from the parent to this node.
// Returns 0 for the root node.
func (n *Node) Arc() uint32 {
	return n.arc
}

// OID returns the full OID for this node by walking up the parent chain.
// Returns nil for the root node.
func (n *Node) OID() Oid {
	if n.Parent == nil {
		return nil
	}
	// Count depth
	depth := 0
	for node := n; node.Parent != nil; node = node.Parent {
		depth++
	}
	// Build OID from root down
	oid := make(Oid, depth)
	i := depth - 1
	for node := n; node.Parent != nil; node = node.Parent {
		oid[i] = node.arc
		i--
	}
	return oid
}

// Child returns the child node with the given arc, or nil if not found. O(1).
func (n *Node) Child(arc uint32) *Node {
	if n.children == nil {
		return nil
	}
	return n.children[arc]
}

// Children returns all child nodes sorted by arc value.
func (n *Node) Children() []*Node {
	if len(n.children) == 0 {
		return nil
	}
	result := make([]*Node, 0, len(n.children))
	for _, child := range n.children {
		result = append(result, child)
	}
	slices.SortFunc(result, func(a, b *Node) int {
		return cmp.Compare(a.arc, b.arc)
	})
	return result
}

// Walk traverses the subtree rooted at this node in pre-order.
// The callback receives each node. Return false to stop walking.
func (n *Node) Walk(fn func(*Node) bool) {
	n.walk(fn)
}

func (n *Node) walk(fn func(*Node) bool) bool {
	if !fn(n) {
		return false
	}
	for _, child := range n.Children() {
		if !child.walk(fn) {
			return false
		}
	}
	return true
}

// yieldAll yields this node and all descendants to the iterator.
func (n *Node) yieldAll(yield func(*Node) bool) bool {
	if !yield(n) {
		return false
	}
	for _, child := range n.Children() {
		if !child.yieldAll(yield) {
			return false
		}
	}
	return true
}

// Descendants returns an iterator over this node and all its descendants in pre-order.
//
//	for node := range root.Descendants() {
//	    fmt.Println(node.Name, node.OID())
//	}
func (n *Node) Descendants() iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		n.yieldAll(yield)
	}
}

// IsRoot returns true if this is the pseudo-root node.
func (n *Node) IsRoot() bool {
	return n.Parent == nil
}

// String returns a brief summary: "name (oid)" or just "(oid)" for unnamed nodes.
func (n *Node) String() string {
	if n == nil {
		return "<nil>"
	}
	if n.Parent == nil {
		return "(root)"
	}
	if n.Name == "" {
		return "(" + n.OID().String() + ")"
	}
	return n.Name + " (" + n.OID().String() + ")"
}

// GetOrCreateChild returns the child with the given arc, creating it if needed.
func (n *Node) GetOrCreateChild(arc uint32) *Node {
	if n.children == nil {
		n.children = make(map[uint32]*Node)
	}
	if child, ok := n.children[arc]; ok {
		return child
	}
	child := &Node{
		arc:    arc,
		Parent: n,
		Kind:   KindInternal,
	}
	n.children[arc] = child
	return child
}

// Object is an OBJECT-TYPE definition.
type Object struct {
	Name        string
	Node        *Node   // always non-nil; OID available via Node.OID()
	Module      *Module // defining module
	Type        *Type   // resolved type, nil only if unresolved
	Access      Access
	Status      Status
	Description string
	Units       string
	Reference   string
	Index       []IndexEntry // for rows; nil otherwise
	Augments    *Object      // for rows with AUGMENTS; nil otherwise
	DefVal      DefVal       // nil if no DEFVAL

	// Pre-computed effective values (from inline constraints + type chain)
	Hint        string       // effective DISPLAY-HINT; empty if none
	Size        []Range      // effective SIZE constraint; nil if none
	ValueRange  []Range      // effective value range; nil if none
	NamedValues []NamedValue // effective enum/BITS values; nil if none
}

// OID returns the object's OID.
func (o *Object) OID() Oid {
	if o.Node == nil {
		return nil
	}
	return o.Node.OID()
}

// Kind returns the inferred kind (Scalar, Table, Row, Column).
func (o *Object) Kind() Kind {
	if o.Node == nil {
		return KindUnknown
	}
	return o.Node.Kind
}

// String returns a brief summary: "name (oid)".
func (o *Object) String() string {
	if o == nil {
		return "<nil>"
	}
	return o.Name + " (" + o.OID().String() + ")"
}

// IndexEntry describes an index component for a table row.
type IndexEntry struct {
	Object  *Object // always non-nil in resolved model
	Implied bool    // IMPLIED keyword present
}

// Range for size/value constraints.
type Range struct {
	Min, Max int64
}

// String returns the range as "min..max" or just "value" if min equals max.
func (r Range) String() string {
	if r.Min == r.Max {
		return strconv.FormatInt(r.Min, 10)
	}
	return strconv.FormatInt(r.Min, 10) + ".." + strconv.FormatInt(r.Max, 10)
}

// NamedValue represents a labeled integer from an enum or BITS definition.
// For INTEGER enums, Value is the enum constant.
// For BITS, Value is the bit position (0-based).
type NamedValue struct {
	Label string
	Value int64
}

// DefVal is the interface for default values.
// All DefVal types implement String() for display.
type DefVal interface {
	String() string
	defVal()
}

// DefValInt is a signed integer default value.
type DefValInt int64

func (DefValInt) defVal() {}

// String returns the integer as a decimal string.
func (d DefValInt) String() string { return strconv.FormatInt(int64(d), 10) }

// DefValUnsigned is an unsigned integer default value.
type DefValUnsigned uint64

func (DefValUnsigned) defVal() {}

// String returns the integer as a decimal string.
func (d DefValUnsigned) String() string { return strconv.FormatUint(uint64(d), 10) }

// DefValString is a quoted string default value.
type DefValString string

func (DefValString) defVal() {}

// String returns the string value with quotes.
func (d DefValString) String() string { return `"` + string(d) + `"` }

// DefValHexString is a hex string default value (e.g., '1F2E'H).
type DefValHexString string

func (DefValHexString) defVal() {}

// String returns the hex string in MIB format (e.g., '1F2E'H).
func (d DefValHexString) String() string { return "'" + string(d) + "'H" }

// DefValBinaryString is a binary string default value (e.g., '1010'B).
type DefValBinaryString string

func (DefValBinaryString) defVal() {}

// String returns the binary string in MIB format (e.g., '1010'B).
func (d DefValBinaryString) String() string { return "'" + string(d) + "'B" }

// DefValEnum is an enumeration label default value.
type DefValEnum string

func (DefValEnum) defVal() {}

// String returns the enum label.
func (d DefValEnum) String() string { return string(d) }

// DefValBits is a BITS default value (list of bit labels).
type DefValBits []string

func (DefValBits) defVal() {}

// String returns the bit labels in braces (e.g., { bit1, bit2 }).
func (d DefValBits) String() string {
	if len(d) == 0 {
		return "{ }"
	}
	return "{ " + strings.Join(d, ", ") + " }"
}

// DefValOID is an OID default value.
type DefValOID Oid

func (DefValOID) defVal() {}

// String returns the OID as a dotted string.
func (d DefValOID) String() string { return Oid(d).String() }

// Type is a type definition (textual convention or type reference).
type Type struct {
	Name        string
	Module      *Module
	Base        BaseType
	Parent      *Type // parent TC, nil for base types
	Hint        string
	Size        []Range
	ValueRange  []Range
	NamedValues []NamedValue
	IsTC        bool // true if TEXTUAL-CONVENTION
	Status      Status
	Description string
	Reference   string
}

// String returns a brief summary: "Name (BaseType)" or just "BaseType" for anonymous types.
func (t *Type) String() string {
	if t == nil {
		return "<nil>"
	}
	if t.Name == "" {
		return t.Base.String()
	}
	return t.Name + " (" + t.Base.String() + ")"
}

// Notification is a NOTIFICATION-TYPE or TRAP-TYPE.
type Notification struct {
	Name        string
	Node        *Node     // always non-nil; OID available via Node.OID()
	Module      *Module   // defining module
	Objects     []*Object // notification objects (OBJECTS clause)
	Status      Status
	Description string
	Reference   string
}

// OID returns the notification's OID.
func (n *Notification) OID() Oid {
	if n.Node == nil {
		return nil
	}
	return n.Node.OID()
}

// String returns a brief summary: "name (oid)".
func (n *Notification) String() string {
	if n == nil {
		return "<nil>"
	}
	return n.Name + " (" + n.OID().String() + ")"
}

// Module is a MIB module.
type Module struct {
	Name         string
	Language     Language
	OID          Oid // MODULE-IDENTITY OID, nil for SMIv1
	Organization string
	ContactInfo  string
	Description  string
	Revisions    []Revision

	// Internal indices for per-module queries
	objects       []*Object
	types         []*Type
	notifications []*Notification
}

// Objects returns all objects defined in this module.
func (m *Module) Objects() []*Object {
	return m.objects
}

// Object returns the object with the given name, or nil if not found.
func (m *Module) Object(name string) *Object {
	for _, obj := range m.objects {
		if obj.Name == name {
			return obj
		}
	}
	return nil
}

// Types returns all types defined in this module.
func (m *Module) Types() []*Type {
	return m.types
}

// Notifications returns all notifications defined in this module.
func (m *Module) Notifications() []*Notification {
	return m.notifications
}

// AddType adds a type to this module's type list.
func (m *Module) AddType(t *Type) {
	m.types = append(m.types, t)
}

// AddObject adds an object to this module's object list.
func (m *Module) AddObject(obj *Object) {
	m.objects = append(m.objects, obj)
}

// AddNotification adds a notification to this module's notification list.
func (m *Module) AddNotification(n *Notification) {
	m.notifications = append(m.notifications, n)
}

// Revision describes a module revision.
type Revision struct {
	Date        string // "YYYY-MM-DD" or original format
	Description string
}

// Diagnostic represents a parse or resolution issue.
type Diagnostic struct {
	Severity Severity
	Module   string // source module name
	Message  string
	Line     int // 0 if not applicable
}

// UnresolvedRef describes a symbol that could not be resolved.
type UnresolvedRef struct {
	Kind   string // "type", "object", "import"
	Symbol string // the unresolved symbol
	Module string // where it was referenced
}
