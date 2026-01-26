package mibimpl

import "github.com/golangsnmp/gomib/mib"

// Object is the concrete implementation of mib.Object.
type Object struct {
	name     string
	node     *Node
	module   *Module
	typ      *Type
	access   mib.Access
	status   mib.Status
	desc     string
	ref      string
	units    string
	defVal   mib.DefVal
	augments *Object
	index    []mib.IndexEntry

	// Pre-computed effective values (from inline constraints + type chain)
	hint   string
	sizes  []mib.Range
	ranges []mib.Range
	enums  []mib.NamedValue
	bits   []mib.NamedValue
}

// Interface methods (mib.Object)

func (o *Object) Name() string {
	return o.name
}

func (o *Object) Module() mib.Module {
	if o.module == nil {
		return nil
	}
	return o.module
}

func (o *Object) OID() mib.Oid {
	if o.node == nil {
		return nil
	}
	return o.node.OID()
}

func (o *Object) Kind() mib.Kind {
	if o.node == nil {
		return mib.KindUnknown
	}
	return o.node.kind
}

func (o *Object) Access() mib.Access {
	return o.access
}

func (o *Object) Status() mib.Status {
	return o.status
}

func (o *Object) Description() string {
	return o.desc
}

func (o *Object) Reference() string {
	return o.ref
}

func (o *Object) Units() string {
	return o.units
}

func (o *Object) DefaultValue() mib.DefVal {
	return o.defVal
}

func (o *Object) EffectiveDisplayHint() string {
	return o.hint
}

func (o *Object) EffectiveSizes() []mib.Range {
	return o.sizes
}

func (o *Object) EffectiveRanges() []mib.Range {
	return o.ranges
}

func (o *Object) EffectiveEnums() []mib.NamedValue {
	return o.enums
}

func (o *Object) EffectiveBits() []mib.NamedValue {
	return o.bits
}

func (o *Object) Enum(label string) (mib.NamedValue, bool) {
	for _, nv := range o.enums {
		if nv.Label == label {
			return nv, true
		}
	}
	return mib.NamedValue{}, false
}

func (o *Object) Bit(label string) (mib.NamedValue, bool) {
	for _, nv := range o.bits {
		if nv.Label == label {
			return nv, true
		}
	}
	return mib.NamedValue{}, false
}

func (o *Object) Node() mib.Node {
	if o.node == nil {
		return nil
	}
	return o.node
}

func (o *Object) Type() mib.Type {
	if o.typ == nil {
		return nil
	}
	return o.typ
}

func (o *Object) Augments() mib.Object {
	if o.augments == nil {
		return nil
	}
	return o.augments
}

func (o *Object) Index() []mib.IndexEntry {
	return o.index
}

// Table navigation
func (o *Object) Table() mib.Object {
	if o.node == nil {
		return nil
	}
	switch o.node.kind {
	case mib.KindRow:
		if o.node.parent != nil && o.node.parent.obj != nil {
			return o.node.parent.obj
		}
	case mib.KindColumn:
		if o.node.parent != nil && o.node.parent.parent != nil {
			if tbl := o.node.parent.parent.obj; tbl != nil {
				return tbl
			}
		}
	}
	return nil
}

func (o *Object) Row() mib.Object {
	if o.node == nil {
		return nil
	}
	if o.node.kind == mib.KindColumn {
		if o.node.parent != nil && o.node.parent.obj != nil {
			return o.node.parent.obj
		}
	}
	return nil
}

func (o *Object) Entry() mib.Object {
	if o.node == nil || o.node.kind != mib.KindTable {
		return nil
	}
	// The row entry is the first child with KindRow
	for _, child := range o.node.sortedChildren() {
		if child.kind == mib.KindRow && child.obj != nil {
			return child.obj
		}
	}
	return nil
}

func (o *Object) Columns() []mib.Object {
	if o.node == nil {
		return nil
	}
	var rowNode *Node
	switch o.node.kind {
	case mib.KindTable:
		// Find the row entry first
		for _, child := range o.node.sortedChildren() {
			if child.kind == mib.KindRow {
				rowNode = child
				break
			}
		}
	case mib.KindRow:
		rowNode = o.node
	default:
		return nil
	}
	if rowNode == nil {
		return nil
	}
	var cols []mib.Object
	for _, child := range rowNode.sortedChildren() {
		if child.kind == mib.KindColumn && child.obj != nil {
			cols = append(cols, child.obj)
		}
	}
	return cols
}

func (o *Object) EffectiveIndexes() []mib.IndexEntry {
	if o.node == nil || o.node.kind != mib.KindRow {
		return nil
	}
	if len(o.index) > 0 {
		return o.index
	}
	if o.augments != nil {
		return o.augments.EffectiveIndexes()
	}
	return nil
}

// Predicates
func (o *Object) IsTable() bool {
	return o.node != nil && o.node.kind == mib.KindTable
}

func (o *Object) IsRow() bool {
	return o.node != nil && o.node.kind == mib.KindRow
}

func (o *Object) IsColumn() bool {
	return o.node != nil && o.node.kind == mib.KindColumn
}

func (o *Object) IsScalar() bool {
	return o.node != nil && o.node.kind == mib.KindScalar
}

// String returns a brief summary: "name (oid)".
func (o *Object) String() string {
	if o == nil {
		return "<nil>"
	}
	return o.name + " (" + o.OID().String() + ")"
}

// Mutation methods (for resolver use)

func (o *Object) SetNode(n *Node) {
	o.node = n
}

func (o *Object) SetModule(m *Module) {
	o.module = m
}

func (o *Object) SetType(t *Type) {
	o.typ = t
}

func (o *Object) SetAccess(a mib.Access) {
	o.access = a
}

func (o *Object) SetStatus(s mib.Status) {
	o.status = s
}

func (o *Object) SetDescription(d string) {
	o.desc = d
}

func (o *Object) SetReference(r string) {
	o.ref = r
}

func (o *Object) SetUnits(u string) {
	o.units = u
}

func (o *Object) SetDefaultValue(d mib.DefVal) {
	o.defVal = d
}

func (o *Object) SetAugments(a *Object) {
	o.augments = a
}

func (o *Object) SetIndex(idx []mib.IndexEntry) {
	o.index = idx
}

func (o *Object) SetEffectiveHint(h string) {
	o.hint = h
}

func (o *Object) SetEffectiveSizes(s []mib.Range) {
	o.sizes = s
}

func (o *Object) SetEffectiveRanges(r []mib.Range) {
	o.ranges = r
}

func (o *Object) SetEffectiveEnums(e []mib.NamedValue) {
	o.enums = e
}

func (o *Object) SetEffectiveBits(b []mib.NamedValue) {
	o.bits = b
}

// InternalNode returns the concrete node for resolver use.
func (o *Object) InternalNode() *Node {
	return o.node
}

// InternalType returns the concrete type for resolver use.
func (o *Object) InternalType() *Type {
	return o.typ
}

// InternalAugments returns the concrete augments object for resolver use.
func (o *Object) InternalAugments() *Object {
	return o.augments
}
