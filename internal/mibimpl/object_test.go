package mibimpl

import (
	"testing"

	"github.com/golangsnmp/gomib/mib"
)

func TestEffectiveIndexes_CircularAugments(t *testing.T) {
	// Create two row objects that augment each other (A augments B, B augments A).
	// EffectiveIndexes() must not infinitely recurse.
	nodeA := &Node{kind: mib.KindRow, parent: &Node{}}
	nodeB := &Node{kind: mib.KindRow, parent: &Node{}}

	objA := NewObject("rowA")
	objA.SetNode(nodeA)

	objB := NewObject("rowB")
	objB.SetNode(nodeB)

	objA.SetAugments(objB)
	objB.SetAugments(objA)

	// Should return nil (no index found) without stack overflow.
	result := objA.EffectiveIndexes()
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestEffectiveIndexes_SelfAugment(t *testing.T) {
	// Object augments itself - degenerate cycle of length 1.
	node := &Node{kind: mib.KindRow, parent: &Node{}}
	obj := NewObject("selfRow")
	obj.SetNode(node)
	obj.SetAugments(obj)

	result := obj.EffectiveIndexes()
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestEffectiveIndexes_NormalAugments(t *testing.T) {
	// Normal case: A augments B, B has indexes. A should inherit B's indexes.
	nodeA := &Node{kind: mib.KindRow, parent: &Node{}}
	nodeB := &Node{kind: mib.KindRow, parent: &Node{}}

	indexObj := NewObject("indexCol")
	indexNode := &Node{kind: mib.KindColumn, parent: &Node{}}
	indexObj.SetNode(indexNode)

	objB := NewObject("rowB")
	objB.SetNode(nodeB)
	objB.SetIndex([]mib.IndexEntry{{Object: indexObj, Implied: false}})

	objA := NewObject("rowA")
	objA.SetNode(nodeA)
	objA.SetAugments(objB)

	result := objA.EffectiveIndexes()
	if len(result) != 1 {
		t.Fatalf("expected 1 index entry, got %d", len(result))
	}
	if result[0].Object.Name() != "indexCol" {
		t.Errorf("expected index object 'indexCol', got %q", result[0].Object.Name())
	}
}

func TestEffectiveIndexes_OwnIndexPreferred(t *testing.T) {
	// When the object has its own index, augments should not be followed.
	nodeA := &Node{kind: mib.KindRow, parent: &Node{}}
	nodeB := &Node{kind: mib.KindRow, parent: &Node{}}

	ownIdx := NewObject("ownIdx")
	ownIdx.SetNode(&Node{kind: mib.KindColumn, parent: &Node{}})

	augIdx := NewObject("augIdx")
	augIdx.SetNode(&Node{kind: mib.KindColumn, parent: &Node{}})

	objB := NewObject("rowB")
	objB.SetNode(nodeB)
	objB.SetIndex([]mib.IndexEntry{{Object: augIdx}})

	objA := NewObject("rowA")
	objA.SetNode(nodeA)
	objA.SetIndex([]mib.IndexEntry{{Object: ownIdx}})
	objA.SetAugments(objB)

	result := objA.EffectiveIndexes()
	if len(result) != 1 {
		t.Fatalf("expected 1 index entry, got %d", len(result))
	}
	if result[0].Object.Name() != "ownIdx" {
		t.Errorf("expected own index 'ownIdx', got %q", result[0].Object.Name())
	}
}

func TestEffectiveIndexes_NotRow(t *testing.T) {
	// EffectiveIndexes returns nil for non-row objects.
	node := &Node{kind: mib.KindScalar, parent: &Node{}}
	obj := NewObject("scalar")
	obj.SetNode(node)
	obj.SetIndex([]mib.IndexEntry{{Object: obj}})

	result := obj.EffectiveIndexes()
	if result != nil {
		t.Errorf("expected nil for scalar, got %v", result)
	}
}
