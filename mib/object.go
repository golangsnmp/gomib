package mib

import "slices"

// Object is an OBJECT-TYPE definition.
type Object struct {
	name     string
	node     *Node
	module   *Module
	typ      *Type
	access   Access
	status   Status
	desc     string
	ref      string
	units    string
	defVal   *DefVal
	augments *Object
	index    []IndexEntry

	hint   string
	sizes  []Range
	ranges []Range
	enums  []NamedValue
	bits   []NamedValue
}

// newObject returns an Object initialized with the given name.
func newObject(name string) *Object {
	return &Object{name: name}
}

func (o *Object) Name() string        { return o.name }
func (o *Object) Node() *Node         { return o.node }
func (o *Object) Module() *Module     { return o.module }
func (o *Object) Type() *Type         { return o.typ }
func (o *Object) Access() Access      { return o.access }
func (o *Object) Status() Status      { return o.status }
func (o *Object) Description() string { return o.desc }
func (o *Object) Reference() string   { return o.ref }
func (o *Object) Units() string       { return o.units }
func (o *Object) Augments() *Object   { return o.augments }

func (o *Object) OID() OID {
	if o == nil || o.node == nil {
		return nil
	}
	return o.node.OID()
}

func (o *Object) Kind() Kind {
	if o.node == nil {
		return KindUnknown
	}
	return o.node.kind
}

func (o *Object) DefaultValue() DefVal {
	if o.defVal == nil {
		return DefVal{}
	}
	return *o.defVal
}

func (o *Object) EffectiveDisplayHint() string { return o.hint }
func (o *Object) EffectiveSizes() []Range      { return slices.Clone(o.sizes) }
func (o *Object) EffectiveRanges() []Range     { return slices.Clone(o.ranges) }
func (o *Object) EffectiveEnums() []NamedValue { return slices.Clone(o.enums) }
func (o *Object) EffectiveBits() []NamedValue  { return slices.Clone(o.bits) }
func (o *Object) Index() []IndexEntry          { return slices.Clone(o.index) }

func (o *Object) Enum(label string) (NamedValue, bool) { return findNamedValue(o.enums, label) }
func (o *Object) Bit(label string) (NamedValue, bool)  { return findNamedValue(o.bits, label) }

// Table returns the table object that contains this row or column, or nil.
func (o *Object) Table() *Object {
	if o.node == nil {
		return nil
	}
	switch o.node.kind {
	case KindRow:
		if o.node.parent != nil && o.node.parent.obj != nil {
			return o.node.parent.obj
		}
	case KindColumn:
		if o.node.parent != nil && o.node.parent.parent != nil {
			if tbl := o.node.parent.parent.obj; tbl != nil {
				return tbl
			}
		}
	}
	return nil
}

func (o *Object) Row() *Object {
	if o.node == nil {
		return nil
	}
	if o.node.kind == KindColumn {
		if o.node.parent != nil && o.node.parent.obj != nil {
			return o.node.parent.obj
		}
	}
	return nil
}

func (o *Object) Entry() *Object {
	if o.node == nil || o.node.kind != KindTable {
		return nil
	}
	for _, child := range o.node.sortedChildren() {
		if child.kind == KindRow && child.obj != nil {
			return child.obj
		}
	}
	return nil
}

func (o *Object) Columns() []*Object {
	if o.node == nil {
		return nil
	}
	var rowNode *Node
	switch o.node.kind {
	case KindTable:
		for _, child := range o.node.sortedChildren() {
			if child.kind == KindRow {
				rowNode = child
				break
			}
		}
	case KindRow:
		rowNode = o.node
	default:
		return nil
	}
	if rowNode == nil {
		return nil
	}
	var cols []*Object
	for _, child := range rowNode.sortedChildren() {
		if child.kind == KindColumn && child.obj != nil {
			cols = append(cols, child.obj)
		}
	}
	return cols
}

func (o *Object) EffectiveIndexes() []IndexEntry {
	return o.effectiveIndexes(make(map[*Object]struct{}))
}

func (o *Object) effectiveIndexes(visited map[*Object]struct{}) []IndexEntry {
	if o.node == nil || o.node.kind != KindRow {
		return nil
	}
	if len(o.index) > 0 {
		return slices.Clone(o.index)
	}
	if o.augments != nil {
		if _, seen := visited[o]; seen {
			return nil
		}
		visited[o] = struct{}{}
		return o.augments.effectiveIndexes(visited)
	}
	return nil
}

func (o *Object) IsTable() bool  { return o.node != nil && o.node.kind == KindTable }
func (o *Object) IsRow() bool    { return o.node != nil && o.node.kind == KindRow }
func (o *Object) IsColumn() bool { return o.node != nil && o.node.kind == KindColumn }
func (o *Object) IsScalar() bool { return o.node != nil && o.node.kind == KindScalar }

// String returns a brief summary: "name (oid)".
func (o *Object) String() string {
	if o == nil {
		return "<nil>"
	}
	return o.name + " (" + o.OID().String() + ")"
}

func (o *Object) setNode(n *Node)                  { o.node = n }
func (o *Object) setModule(m *Module)              { o.module = m }
func (o *Object) setType(t *Type)                  { o.typ = t }
func (o *Object) setAccess(a Access)               { o.access = a }
func (o *Object) setStatus(s Status)               { o.status = s }
func (o *Object) setDescription(d string)          { o.desc = d }
func (o *Object) setReference(r string)            { o.ref = r }
func (o *Object) setUnits(u string)                { o.units = u }
func (o *Object) setDefaultValue(d *DefVal)        { o.defVal = d }
func (o *Object) setAugments(a *Object)            { o.augments = a }
func (o *Object) setIndex(idx []IndexEntry)        { o.index = idx }
func (o *Object) setEffectiveHint(h string)        { o.hint = h }
func (o *Object) setEffectiveSizes(s []Range)      { o.sizes = s }
func (o *Object) setEffectiveRanges(r []Range)     { o.ranges = r }
func (o *Object) setEffectiveEnums(e []NamedValue) { o.enums = e }
func (o *Object) setEffectiveBits(b []NamedValue)  { o.bits = b }

// objectsByKind returns objects whose node matches the given kind.
func objectsByKind(objs []*Object, kind Kind) []*Object {
	var result []*Object
	for _, obj := range objs {
		if obj.node != nil && obj.node.kind == kind {
			result = append(result, obj)
		}
	}
	return result
}
