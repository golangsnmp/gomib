package mib

import "testing"

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
		if got != tableObj {
			t.Errorf("got %v, want %v", got, tableObj)
		}
	})

	t.Run("Table from column", func(t *testing.T) {
		got := col1Obj.Table()
		if got != tableObj {
			t.Errorf("got %v, want %v", got, tableObj)
		}
	})

	t.Run("Table from table is nil", func(t *testing.T) {
		if got := tableObj.Table(); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("Row from column", func(t *testing.T) {
		got := col1Obj.Row()
		if got != rowObj {
			t.Errorf("got %v, want %v", got, rowObj)
		}
	})

	t.Run("Row from non-column is nil", func(t *testing.T) {
		if got := rowObj.Row(); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("Entry from table", func(t *testing.T) {
		got := tableObj.Entry()
		if got != rowObj {
			t.Errorf("got %v, want %v", got, rowObj)
		}
	})

	t.Run("Entry from non-table is nil", func(t *testing.T) {
		if got := rowObj.Entry(); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("Columns from table", func(t *testing.T) {
		cols := tableObj.Columns()
		if len(cols) != 2 {
			t.Fatalf("got %d columns, want 2", len(cols))
		}
		// sortedChildren orders by arc
		if cols[0] != col1Obj || cols[1] != col2Obj {
			t.Errorf("got [%v, %v], want [%v, %v]", cols[0], cols[1], col1Obj, col2Obj)
		}
	})

	t.Run("Columns from row", func(t *testing.T) {
		cols := rowObj.Columns()
		if len(cols) != 2 {
			t.Fatalf("got %d columns, want 2", len(cols))
		}
		if cols[0] != col1Obj || cols[1] != col2Obj {
			t.Errorf("got [%v, %v], want [%v, %v]", cols[0], cols[1], col1Obj, col2Obj)
		}
	})

	t.Run("Columns from column is nil", func(t *testing.T) {
		if got := col1Obj.Columns(); got != nil {
			t.Errorf("got %v, want nil", got)
		}
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
		if got != nil {
			t.Errorf("got %v, want nil for cycle", got)
		}
	})
}
