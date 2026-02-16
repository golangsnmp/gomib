package mib

import (
	"slices"
	"testing"
)

// buildTree constructs:
//
//	root
//	  ├── a (arc 1)
//	  │     ├── c (arc 3)
//	  │     └── d (arc 1)
//	  └── b (arc 5)
//	        └── e (arc 2)
func buildTree() *Node {
	root := &Node{kind: KindInternal}

	a := &Node{arc: 1, name: "a", parent: root, children: make(map[uint32]*Node)}
	b := &Node{arc: 5, name: "b", parent: root, children: make(map[uint32]*Node)}
	root.children = map[uint32]*Node{1: a, 5: b}

	c := &Node{arc: 3, name: "c", parent: a}
	d := &Node{arc: 1, name: "d", parent: a}
	a.children = map[uint32]*Node{3: c, 1: d}

	e := &Node{arc: 2, name: "e", parent: b}
	b.children = map[uint32]*Node{2: e}

	return root
}

func TestChildrenSortOrder(t *testing.T) {
	root := buildTree()

	// Root children should be sorted by arc: a(1), b(5)
	children := root.Children()
	if len(children) != 2 {
		t.Fatalf("got %d children, want 2", len(children))
	}
	if children[0].name != "a" || children[1].name != "b" {
		t.Errorf("got [%s, %s], want [a, b]", children[0].name, children[1].name)
	}

	// Node a's children: d(arc 1) before c(arc 3)
	aChildren := children[0].Children()
	if len(aChildren) != 2 {
		t.Fatalf("got %d children for a, want 2", len(aChildren))
	}
	if aChildren[0].name != "d" || aChildren[1].name != "c" {
		t.Errorf("got [%s, %s], want [d, c]", aChildren[0].name, aChildren[1].name)
	}
}

func TestChildrenReturnsCopy(t *testing.T) {
	root := buildTree()

	c1 := root.Children()
	c2 := root.Children()
	if &c1[0] == &c2[0] {
		t.Error("Children() should return a new slice each call")
	}
}

func TestSubtreeOrder(t *testing.T) {
	root := buildTree()

	var names []string
	for nd := range root.Subtree() {
		names = append(names, nd.name)
	}

	// Pre-order DFS, children sorted by arc:
	// root("") -> a(1) -> d(1) -> c(3) -> b(5) -> e(2)
	want := []string{"", "a", "d", "c", "b", "e"}
	if !slices.Equal(names, want) {
		t.Errorf("got %v, want %v", names, want)
	}
}

func TestSubtreeEarlyStop(t *testing.T) {
	root := buildTree()

	var names []string
	for nd := range root.Subtree() {
		names = append(names, nd.name)
		if nd.name == "d" {
			break
		}
	}

	want := []string{"", "a", "d"}
	if !slices.Equal(names, want) {
		t.Errorf("got %v, want %v", names, want)
	}
}

func TestWalkOID(t *testing.T) {
	root := buildTree()

	tests := []struct {
		name      string
		oid       OID
		wantName  string
		wantExact bool
	}{
		{"exact leaf", OID{1, 3}, "c", true},
		{"exact mid", OID{1}, "a", true},
		{"partial - extra arc beyond leaf", OID{1, 3, 99}, "c", false},
		{"partial - unknown arc at first level", OID{99}, "", false},
		{"partial - unknown arc at second level", OID{1, 7}, "a", false},
		{"empty OID", OID{}, "", true},
		{"nil OID", nil, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, exact := root.walkOID(tt.oid)
			if node.name != tt.wantName {
				t.Errorf("node = %q, want %q", node.name, tt.wantName)
			}
			if exact != tt.wantExact {
				t.Errorf("exact = %v, want %v", exact, tt.wantExact)
			}
		})
	}
}
