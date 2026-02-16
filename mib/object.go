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

// OID returns the object's position in the OID tree, or nil if unresolved.
func (o *Object) OID() OID {
	if o == nil || o.node == nil {
		return nil
	}
	return o.node.OID()
}

// Kind reports the structural classification of this object's tree node.
func (o *Object) Kind() Kind {
	if o == nil || o.node == nil {
		return KindUnknown
	}
	return o.node.kind
}

// DefaultValue returns the DEFVAL clause, or a zero DefVal if none was declared.
func (o *Object) DefaultValue() DefVal {
	if o == nil || o.defVal == nil {
		return DefVal{}
	}
	return *o.defVal
}

// EffectiveDisplayHint returns the display hint resolved through the type chain.
func (o *Object) EffectiveDisplayHint() string { return o.hint }

// EffectiveSizes returns size constraints resolved through the type chain.
func (o *Object) EffectiveSizes() []Range { return slices.Clone(o.sizes) }

// EffectiveRanges returns range constraints resolved through the type chain.
func (o *Object) EffectiveRanges() []Range { return slices.Clone(o.ranges) }

// EffectiveEnums returns enumeration values resolved through the type chain.
func (o *Object) EffectiveEnums() []NamedValue { return slices.Clone(o.enums) }

// EffectiveBits returns bit definitions resolved through the type chain.
func (o *Object) EffectiveBits() []NamedValue { return slices.Clone(o.bits) }

// Index returns the declared INDEX entries for this object.
func (o *Object) Index() []IndexEntry { return slices.Clone(o.index) }

// Enum looks up an enumeration value by label.
func (o *Object) Enum(label string) (NamedValue, bool) { return findNamedValue(o.enums, label) }

// Bit looks up a BITS value by label.
func (o *Object) Bit(label string) (NamedValue, bool) { return findNamedValue(o.bits, label) }

// Table returns the table object that contains this row or column, or nil.
func (o *Object) Table() *Object {
	if o == nil || o.node == nil {
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

// Row returns the parent row object for a column, or nil.
func (o *Object) Row() *Object {
	if o == nil || o.node == nil {
		return nil
	}
	if o.node.kind == KindColumn {
		if o.node.parent != nil && o.node.parent.obj != nil {
			return o.node.parent.obj
		}
	}
	return nil
}

// Entry returns the row entry for a table, or nil.
func (o *Object) Entry() *Object {
	if o == nil || o.node == nil || o.node.kind != KindTable {
		return nil
	}
	for _, child := range o.node.sortedChildren() {
		if child.kind == KindRow && child.obj != nil {
			return child.obj
		}
	}
	return nil
}

// Columns returns the column objects for a table or row, or nil.
func (o *Object) Columns() []*Object {
	if o == nil || o.node == nil {
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

// EffectiveIndexes returns INDEX entries for a row, following the AUGMENTS
// chain if the row has no indexes of its own.
func (o *Object) EffectiveIndexes() []IndexEntry {
	if o == nil {
		return nil
	}
	return o.effectiveIndexes(make(map[*Object]struct{}))
}

func (o *Object) effectiveIndexes(visited map[*Object]struct{}) []IndexEntry {
	if o == nil || o.node == nil || o.node.kind != KindRow {
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

// IsTable reports whether this object is a table node.
func (o *Object) IsTable() bool { return o != nil && o.node != nil && o.node.kind == KindTable }

// IsRow reports whether this object is a table row (entry) node.
func (o *Object) IsRow() bool { return o != nil && o.node != nil && o.node.kind == KindRow }

// IsColumn reports whether this object is a table column node.
func (o *Object) IsColumn() bool { return o != nil && o.node != nil && o.node.kind == KindColumn }

// IsScalar reports whether this object is a scalar node.
func (o *Object) IsScalar() bool { return o != nil && o.node != nil && o.node.kind == KindScalar }

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

func objectsByKind(objs []*Object, kind Kind) []*Object {
	var result []*Object
	for _, obj := range objs {
		if obj.node != nil && obj.node.kind == kind {
			result = append(result, obj)
		}
	}
	return result
}
