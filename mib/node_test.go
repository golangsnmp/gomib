package mib

import (
	"sync"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
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
	testutil.Len(t, children, 2, "children")
	testutil.Equal(t, "a", children[0].name, "children[0].name")
	testutil.Equal(t, "b", children[1].name, "children[1].name")

	// Node a's children: d(arc 1) before c(arc 3)
	aChildren := children[0].Children()
	testutil.Len(t, aChildren, 2, "children for a")
	testutil.Equal(t, "d", aChildren[0].name, "aChildren[0].name")
	testutil.Equal(t, "c", aChildren[1].name, "aChildren[1].name")
}

func TestChildrenReturnsCopy(t *testing.T) {
	root := buildTree()

	c1 := root.Children()
	c2 := root.Children()
	testutil.True(t, &c1[0] != &c2[0], "Children() should return a new slice each call")
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
	testutil.SliceEqual(t, want, names, "got")
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
	testutil.SliceEqual(t, want, names, "got")
}

func TestDescendantsOrder(t *testing.T) {
	root := buildTree()

	var names []string
	for nd := range root.Descendants() {
		names = append(names, nd.name)
	}

	// Same as Subtree but without root itself.
	want := []string{"a", "d", "c", "b", "e"}
	testutil.SliceEqual(t, want, names, "got")
}

func TestDescendantsEarlyStop(t *testing.T) {
	root := buildTree()

	var names []string
	for nd := range root.Descendants() {
		names = append(names, nd.name)
		if nd.name == "d" {
			break
		}
	}

	want := []string{"a", "d"}
	testutil.SliceEqual(t, want, names, "got")
}

func TestDescendantsLeaf(t *testing.T) {
	root := buildTree()

	// Leaf node should have no descendants.
	leaf := root.children[1].children[3] // node "c"
	var count int
	for range leaf.Descendants() {
		count++
	}
	testutil.Equal(t, 0, count, "leaf descendants")
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
			testutil.Equal(t, tt.wantName, node.name, "node")
			testutil.Equal(t, tt.wantExact, exact, "exact")
		})
	}
}

func TestSortedChildrenConcurrent(t *testing.T) {
	root := buildTree()

	// Access Children() concurrently from multiple goroutines.
	// Run with -race to verify no data race on sortedCache.
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			children := root.Children()
			testutil.Len(t, children, 2, "children")
		}()
	}
	wg.Wait()
}
