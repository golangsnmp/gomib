package model

import "slices"

// Object is an OBJECT-TYPE definition from an SMIv1 or SMIv2 module.
// Each object is classified by its [Kind]: scalar, table, row, or column.
// Objects are attached to the OID tree via their [Node].
type Object struct {
	entity
	typ          *Type
	access       Access
	units        string
	defVal       *DefVal
	augments     *Object
	augmentedBy  []*Object
	syntaxSpan   Span
	accessSpan   Span
	unitsSpan    Span
	augmentsSpan Span
	defValSpan   Span
	index        []IndexEntry

	hint             string
	sizes            []Range
	ranges           []Range
	enums            []NamedValue
	bits             []NamedValue
	sequenceTypeName string
}

// Type returns the resolved type of this object, or nil if unresolved.
func (o *Object) Type() *Type { return o.typ }

// Access returns the MAX-ACCESS or ACCESS clause value.
func (o *Object) Access() Access { return o.access }

// SyntaxSpan returns the source byte range of this object's SYNTAX clause.
func (o *Object) SyntaxSpan() Span { return o.syntaxSpan }

// AccessSpan returns the source byte range of this object's ACCESS clause.
func (o *Object) AccessSpan() Span { return o.accessSpan }

// UnitsSpan returns the source byte range of this object's UNITS clause.
func (o *Object) UnitsSpan() Span { return o.unitsSpan }

// AugmentsSpan returns the source byte range of this object's AUGMENTS clause.
func (o *Object) AugmentsSpan() Span { return o.augmentsSpan }

// DefaultValueSpan returns the source byte range of this object's DEFVAL clause.
func (o *Object) DefaultValueSpan() Span { return o.defValSpan }

// Units returns the UNITS clause text, or "".
func (o *Object) Units() string { return o.units }

// Augments returns the row object this one augments, or nil.
func (o *Object) Augments() *Object { return o.augments }

// AugmentedBy returns the row objects that augment this one, or nil.
// Only meaningful for KindRow objects with their own INDEX clause.
func (o *Object) AugmentedBy() []*Object { return slices.Clone(o.augmentedBy) }

// Kind reports the structural classification of this object's tree node.
func (o *Object) Kind() Kind {
	if o.node == nil {
		return KindUnknown
	}
	return o.node.kind
}

// DefaultValue returns the DEFVAL clause, or a zero DefVal if none was declared.
func (o *Object) DefaultValue() DefVal {
	if o.defVal == nil {
		return DefVal{}
	}
	return *o.defVal
}

// EffectiveDisplayHint returns the display hint resolved through the type chain.
func (o *Object) EffectiveDisplayHint() string { return o.hint }

// ParsedDisplayHint parses and validates this object's effective DISPLAY-HINT,
// returning a structured [DisplayHint]. Returns nil if the object has no display
// hint or the hint is malformed.
func (o *Object) ParsedDisplayHint() *DisplayHint {
	if o.hint == "" {
		return nil
	}
	return ParseDisplayHint(o.hint)
}

// FormatInteger formats an integer value using this object's effective
// DISPLAY-HINT. Returns "" and false if the object has no display hint or the
// hint is not a valid integer hint.
func (o *Object) FormatInteger(value int64, hexCase HexCase) (string, bool) {
	if o.hint == "" {
		return "", false
	}
	return FormatInteger(o.hint, value, hexCase)
}

// ScaleInteger applies this object's DISPLAY-HINT as numeric scaling. Only
// "d" and "d-N" hints produce a result. Returns 0 and false if the hint is
// absent, non-decimal, or malformed.
func (o *Object) ScaleInteger(value int64) (float64, bool) {
	if o.hint == "" {
		return 0, false
	}
	return ScaleInteger(o.hint, value)
}

// FormatOctets formats an octet string using this object's effective
// DISPLAY-HINT. Returns "" and false if the object has no display hint, the
// hint is malformed, or data is empty.
func (o *Object) FormatOctets(data []byte, hexCase HexCase) (string, bool) {
	if o.hint == "" {
		return "", false
	}
	return FormatOctets(o.hint, data, hexCase)
}

// EffectiveSizes returns size constraints resolved through the type chain.
func (o *Object) EffectiveSizes() []Range { return slices.Clone(o.sizes) }

// EffectiveRanges returns range constraints resolved through the type chain.
func (o *Object) EffectiveRanges() []Range { return slices.Clone(o.ranges) }

// EffectiveEnums returns enumeration values resolved through the type chain.
func (o *Object) EffectiveEnums() []NamedValue { return slices.Clone(o.enums) }

// EffectiveBits returns bit definitions resolved through the type chain.
func (o *Object) EffectiveBits() []NamedValue { return slices.Clone(o.bits) }

// SequenceTypeName returns the original SEQUENCE type name from the row's
// SYNTAX clause, or "" if not available. This preserves the original casing
// (e.g., "DSLConnectionTableEntry") that would otherwise be lost.
func (o *Object) SequenceTypeName() string { return o.sequenceTypeName }

// Index returns the declared INDEX entries for this object.
func (o *Object) Index() []IndexEntry { return slices.Clone(o.index) }

// Enum looks up an enumeration value by label.
func (o *Object) Enum(label string) (NamedValue, bool) { return findNamedValue(o.enums, label) }

// Bit looks up a BITS value by label.
func (o *Object) Bit(label string) (NamedValue, bool) { return findNamedValue(o.bits, label) }

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

// Row returns the parent row object for a column, or nil.
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

// Entry returns the row entry for a table, or nil.
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

// Columns returns the column objects for a table or row, or nil.
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

// EffectiveIndexes returns INDEX entries for a row, following the AUGMENTS
// chain if the row has no indexes of its own.
func (o *Object) EffectiveIndexes() []IndexEntry {
	return o.effectiveIndexes(make(map[*Object]struct{}))
}

func (o *Object) effectiveIndexes(visited map[*Object]struct{}) []IndexEntry {
	if o.node == nil {
		return nil
	}
	// For columns, delegate to the parent row.
	if o.node.kind == KindColumn {
		if row := o.Row(); row != nil {
			return row.effectiveIndexes(visited)
		}
		return nil
	}
	if o.node.kind != KindRow {
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
func (o *Object) IsTable() bool { return o.node != nil && o.node.kind == KindTable }

// IsRow reports whether this object is a table row (entry) node.
func (o *Object) IsRow() bool { return o.node != nil && o.node.kind == KindRow }

// IsColumn reports whether this object is a table column node.
func (o *Object) IsColumn() bool { return o.node != nil && o.node.kind == KindColumn }

// IsScalar reports whether this object is a scalar node.
func (o *Object) IsScalar() bool { return o.node != nil && o.node.kind == KindScalar }

// IsIndex reports whether this column object appears in its parent row's
// effective INDEX clause. RFC 2578 s7.7 calls these "auxiliary objects"
// and requires them to be not-accessible, with limited exceptions.
func (o *Object) IsIndex() bool {
	if o.node == nil || o.node.kind != KindColumn {
		return false
	}
	for _, idx := range o.EffectiveIndexes() {
		if idx.Object == o {
			return true
		}
	}
	return false
}

func (o *Object) setType(t *Type)                  { o.typ = t }
func (o *Object) setAccess(a Access)               { o.access = a }
func (o *Object) setUnits(u string)                { o.units = u }
func (o *Object) setDefaultValue(d *DefVal)        { o.defVal = d }
func (o *Object) setAugments(a *Object)            { o.augments = a }
func (o *Object) addAugmentedBy(a *Object)         { o.augmentedBy = append(o.augmentedBy, a) }
func (o *Object) setIndex(idx []IndexEntry)        { o.index = idx }
func (o *Object) setEffectiveHint(h string)        { o.hint = h }
func (o *Object) setEffectiveSizes(s []Range)      { o.sizes = s }
func (o *Object) setEffectiveRanges(r []Range)     { o.ranges = r }
func (o *Object) setEffectiveEnums(e []NamedValue) { o.enums = e }
func (o *Object) setEffectiveBits(b []NamedValue)  { o.bits = b }
func (o *Object) setSequenceTypeName(s string)     { o.sequenceTypeName = s }
func (o *Object) setSyntaxSpan(s Span)             { o.syntaxSpan = s }
func (o *Object) setAccessSpan(s Span)             { o.accessSpan = s }
func (o *Object) setUnitsSpan(s Span)              { o.unitsSpan = s }
func (o *Object) setAugmentsSpan(s Span)           { o.augmentsSpan = s }
func (o *Object) setDefaultValueSpan(s Span)       { o.defValSpan = s }

func objectsByKind(objs []*Object, kind Kind) []*Object {
	var result []*Object
	for _, obj := range objs {
		if obj.node != nil && obj.node.kind == kind {
			result = append(result, obj)
		}
	}
	return result
}
