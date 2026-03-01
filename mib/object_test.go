package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// buildTableTree constructs a table -> row -> columns fixture.
//
//	root
//	  └── tableNode (KindTable, arc 1)
//	        └── rowNode (KindRow, arc 1)
//	              ├── col1Node (KindColumn, arc 1)
//	              └── col2Node (KindColumn, arc 2)
func buildTableTree() (tableObj, rowObj, col1Obj, col2Obj *Object) {
	root := &Node{kind: KindInternal}

	tableNode := &Node{arc: 1, kind: KindTable, parent: root, children: make(map[uint32]*Node)}
	root.children = map[uint32]*Node{1: tableNode}

	rowNode := &Node{arc: 1, kind: KindRow, parent: tableNode, children: make(map[uint32]*Node)}
	tableNode.children[1] = rowNode

	col1Node := &Node{arc: 1, kind: KindColumn, parent: rowNode}
	col2Node := &Node{arc: 2, kind: KindColumn, parent: rowNode}
	rowNode.children[1] = col1Node
	rowNode.children[2] = col2Node

	tableObj = &Object{entity: entity{name: "fooTable", node: tableNode}}
	rowObj = &Object{entity: entity{name: "fooEntry", node: rowNode}}
	col1Obj = &Object{entity: entity{name: "fooCol1", node: col1Node}}
	col2Obj = &Object{entity: entity{name: "fooCol2", node: col2Node}}

	tableNode.obj = tableObj
	rowNode.obj = rowObj
	col1Node.obj = col1Obj
	col2Node.obj = col2Obj

	return tableObj, rowObj, col1Obj, col2Obj
}

func TestTableNavigation(t *testing.T) {
	tableObj, rowObj, col1Obj, col2Obj := buildTableTree()

	t.Run("Table from row", func(t *testing.T) {
		got := rowObj.Table()
		testutil.Equal(t, tableObj, got, "got")
	})

	t.Run("Table from column", func(t *testing.T) {
		got := col1Obj.Table()
		testutil.Equal(t, tableObj, got, "got")
	})

	t.Run("Table from table is nil", func(t *testing.T) {
		testutil.Nil(t, tableObj.Table(), "expected nil")
	})

	t.Run("Row from column", func(t *testing.T) {
		got := col1Obj.Row()
		testutil.Equal(t, rowObj, got, "got")
	})

	t.Run("Row from non-column is nil", func(t *testing.T) {
		testutil.Nil(t, rowObj.Row(), "expected nil")
	})

	t.Run("Entry from table", func(t *testing.T) {
		got := tableObj.Entry()
		testutil.Equal(t, rowObj, got, "got")
	})

	t.Run("Entry from non-table is nil", func(t *testing.T) {
		testutil.Nil(t, rowObj.Entry(), "expected nil")
	})

	t.Run("Columns from table", func(t *testing.T) {
		cols := tableObj.Columns()
		testutil.Len(t, cols, 2, "columns")
		// sortedChildren orders by arc
		testutil.Equal(t, col1Obj, cols[0], "col 0")
		testutil.Equal(t, col2Obj, cols[1], "col 1")
	})

	t.Run("Columns from row", func(t *testing.T) {
		cols := rowObj.Columns()
		testutil.Len(t, cols, 2, "columns")
		testutil.Equal(t, col1Obj, cols[0], "col 0")
		testutil.Equal(t, col2Obj, cols[1], "col 1")
	})

	t.Run("Columns from column is nil", func(t *testing.T) {
		testutil.Nil(t, col1Obj.Columns(), "expected nil")
	})
}

func TestIsIndex(t *testing.T) {
	_, rowObj, col1Obj, col2Obj := buildTableTree()

	// col1 is in the row's INDEX
	rowObj.index = []IndexEntry{{Object: col1Obj}}

	t.Run("index column is an index", func(t *testing.T) {
		testutil.True(t, col1Obj.IsIndex(), "index column should be an index")
	})

	t.Run("data column is not an index", func(t *testing.T) {
		testutil.False(t, col2Obj.IsIndex(), "data column should not be an index")
	})

	t.Run("row is not an index", func(t *testing.T) {
		testutil.False(t, rowObj.IsIndex(), "row should not be an index")
	})
}

func TestAugmentedBy(t *testing.T) {
	makeRow := func(name string) *Object {
		node := &Node{kind: KindRow}
		obj := &Object{entity: entity{name: name, node: node}}
		node.obj = obj
		return obj
	}

	t.Run("base row with augmenters", func(t *testing.T) {
		base := makeRow("baseEntry")
		aug1 := makeRow("aug1Entry")
		aug2 := makeRow("aug2Entry")
		aug1.augments = base
		aug2.augments = base
		base.augmentedBy = []*Object{aug1, aug2}

		got := base.AugmentedBy()
		testutil.Len(t, got, 2, "augmented by")
		testutil.Equal(t, aug1, got[0], "first augmenter")
		testutil.Equal(t, aug2, got[1], "second augmenter")
	})

	t.Run("row without augmenters returns nil", func(t *testing.T) {
		row := makeRow("plainEntry")
		testutil.Nil(t, row.AugmentedBy(), "no augmenters")
	})

	t.Run("returned slice is a copy", func(t *testing.T) {
		base := makeRow("baseEntry")
		aug := makeRow("augEntry")
		base.augmentedBy = []*Object{aug}

		got := base.AugmentedBy()
		got[0] = nil
		testutil.NotNil(t, base.AugmentedBy()[0], "original should be unmodified")
	})
}

func TestEffectiveIndexes(t *testing.T) {
	makeRow := func(name string, idx []IndexEntry) *Object {
		node := &Node{kind: KindRow}
		obj := &Object{entity: entity{name: name, node: node}, index: idx}
		node.obj = obj
		return obj
	}

	sentinel := &Object{entity: entity{name: "idxObj"}}
	idx := []IndexEntry{{Object: sentinel, Implied: false}}

	t.Run("row with own indexes", func(t *testing.T) {
		row := makeRow("directRow", idx)
		got := row.EffectiveIndexes()
		testutil.Len(t, got, 1, "effective indexes")
		testutil.Equal(t, sentinel, got[0].Object, "index object")
	})

	t.Run("row inherits via augments", func(t *testing.T) {
		base := makeRow("baseRow", idx)
		augmenting := makeRow("augRow", nil)
		augmenting.augments = base

		got := augmenting.EffectiveIndexes()
		testutil.Len(t, got, 1, "effective indexes via augments")
		testutil.Equal(t, sentinel, got[0].Object, "inherited index object")
	})

	t.Run("bare type index entry", func(t *testing.T) {
		bareIdx := []IndexEntry{{TypeName: "INTEGER", Implied: false}}
		row := makeRow("bareTypeRow", bareIdx)
		got := row.EffectiveIndexes()
		testutil.Len(t, got, 1, "effective indexes")
		testutil.Nil(t, got[0].Object, "bare type Object should be nil")
		testutil.Equal(t, "INTEGER", got[0].TypeName, "bare type TypeName")
	})

	t.Run("augments cycle terminates", func(t *testing.T) {
		a := makeRow("rowA", nil)
		b := makeRow("rowB", nil)
		a.augments = b
		b.augments = a

		got := a.EffectiveIndexes()
		testutil.Nil(t, got, "got , want nil for cycle")
	})
}

func TestClassifyIndexEncoding(t *testing.T) {
	makeObj := func(base BaseType, sizes []Range) *Object {
		typ := newType("t")
		typ.setBase(base)
		obj := &Object{entity: entity{name: "idx"}, typ: typ, sizes: sizes}
		return obj
	}

	tests := []struct {
		name    string
		obj     *Object
		implied bool
		want    IndexEncoding
	}{
		{"Integer32", makeObj(BaseInteger32, nil), false, IndexEncodingInteger},
		{"Unsigned32", makeObj(BaseUnsigned32, nil), false, IndexEncodingInteger},
		{"Gauge32", makeObj(BaseGauge32, nil), false, IndexEncodingInteger},
		{"TimeTicks", makeObj(BaseTimeTicks, nil), false, IndexEncodingInteger},
		{"Counter32", makeObj(BaseCounter32, nil), false, IndexEncodingInteger},
		{"IpAddress", makeObj(BaseIpAddress, nil), false, IndexEncodingIpAddress},
		{"fixed size OCTET STRING", makeObj(BaseOctetString, []Range{{Min: 6, Max: 6}}), false, IndexEncodingFixedString},
		{"variable OCTET STRING", makeObj(BaseOctetString, nil), false, IndexEncodingLengthPrefixed},
		{"variable OCTET STRING with size range", makeObj(BaseOctetString, []Range{{Min: 0, Max: 255}}), false, IndexEncodingLengthPrefixed},
		{"OCTET STRING with IMPLIED", makeObj(BaseOctetString, nil), true, IndexEncodingImplied},
		{"OBJECT IDENTIFIER", makeObj(BaseObjectIdentifier, nil), false, IndexEncodingLengthPrefixed},
		{"OBJECT IDENTIFIER with IMPLIED", makeObj(BaseObjectIdentifier, nil), true, IndexEncodingImplied},
		{"BITS", makeObj(BaseBits, nil), false, IndexEncodingLengthPrefixed},
		{"Opaque", makeObj(BaseOpaque, nil), false, IndexEncodingLengthPrefixed},
		{"nil object", nil, false, IndexEncodingUnknown},
		{"nil type", &Object{entity: entity{name: "x"}}, false, IndexEncodingUnknown},
		{"unknown base type", makeObj(BaseUnknown, nil), false, IndexEncodingUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Equal(t, tt.want, classifyIndexEncoding(tt.obj, tt.implied), "classifyIndexEncoding()")
		})
	}
}

func TestIsFixedSize(t *testing.T) {
	tests := []struct {
		name  string
		sizes []Range
		want  bool
	}{
		{"nil", nil, false},
		{"empty", []Range{}, false},
		{"fixed", []Range{{Min: 6, Max: 6}}, true},
		{"range", []Range{{Min: 0, Max: 255}}, false},
		{"zero fixed", []Range{{Min: 0, Max: 0}}, false},
		{"multiple", []Range{{Min: 4, Max: 4}, {Min: 6, Max: 6}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Equal(t, tt.want, isFixedSize(tt.sizes), "isFixedSize()")
		})
	}
}
