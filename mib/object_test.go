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

	tableObj = &Object{name: "fooTable", node: tableNode}
	rowObj = &Object{name: "fooEntry", node: rowNode}
	col1Obj = &Object{name: "fooCol1", node: col1Node}
	col2Obj = &Object{name: "fooCol2", node: col2Node}

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
		if cols[0] != col1Obj || cols[1] != col2Obj {
			t.Errorf("got [%v, %v], want [%v, %v]", cols[0], cols[1], col1Obj, col2Obj)
		}
	})

	t.Run("Columns from row", func(t *testing.T) {
		cols := rowObj.Columns()
		testutil.Len(t, cols, 2, "columns")
		if cols[0] != col1Obj || cols[1] != col2Obj {
			t.Errorf("got [%v, %v], want [%v, %v]", cols[0], cols[1], col1Obj, col2Obj)
		}
	})

	t.Run("Columns from column is nil", func(t *testing.T) {
		testutil.Nil(t, col1Obj.Columns(), "expected nil")
	})
}

func TestEffectiveIndexes(t *testing.T) {
	makeRow := func(name string, idx []IndexEntry) *Object {
		node := &Node{kind: KindRow}
		obj := &Object{name: name, node: node, index: idx}
		node.obj = obj
		return obj
	}

	sentinel := &Object{name: "idxObj"}
	idx := []IndexEntry{{Object: sentinel, Implied: false}}

	t.Run("row with own indexes", func(t *testing.T) {
		row := makeRow("directRow", idx)
		got := row.EffectiveIndexes()
		if len(got) != 1 || got[0].Object != sentinel {
			t.Errorf("got %v, want index pointing to %v", got, sentinel)
		}
	})

	t.Run("row inherits via augments", func(t *testing.T) {
		base := makeRow("baseRow", idx)
		augmenting := makeRow("augRow", nil)
		augmenting.augments = base

		got := augmenting.EffectiveIndexes()
		if len(got) != 1 || got[0].Object != sentinel {
			t.Errorf("got %v, want inherited index pointing to %v", got, sentinel)
		}
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
