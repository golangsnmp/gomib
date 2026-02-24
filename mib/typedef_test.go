package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestEffectiveBase(t *testing.T) {
	t.Run("direct", func(t *testing.T) {
		ty := newType("MyInt")
		ty.setBase(BaseInteger32)
		testutil.Equal(t, BaseInteger32, ty.EffectiveBase(), "got")
	})

	t.Run("inherited from parent", func(t *testing.T) {
		parent := newType("Parent")
		parent.setBase(BaseOctetString)
		child := newType("Child")
		child.setParent(parent)

		testutil.Equal(t, BaseOctetString, child.EffectiveBase(), "got")
	})

	t.Run("inherited from grandparent", func(t *testing.T) {
		grandparent := newType("GP")
		grandparent.setBase(BaseGauge32)
		parent := newType("Parent")
		parent.setParent(grandparent)
		child := newType("Child")
		child.setParent(parent)

		testutil.Equal(t, BaseGauge32, child.EffectiveBase(), "got")
	})

	t.Run("child shadows parent", func(t *testing.T) {
		parent := newType("Parent")
		parent.setBase(BaseOctetString)
		child := newType("Child")
		child.setBase(BaseInteger32)
		child.setParent(parent)

		testutil.Equal(t, BaseInteger32, child.EffectiveBase(), "got")
	})

	t.Run("no base anywhere", func(t *testing.T) {
		parent := newType("Parent")
		child := newType("Child")
		child.setParent(parent)

		testutil.Equal(t, 0, child.EffectiveBase(), "got")
	})
}

func TestEffectiveEnums(t *testing.T) {
	parentEnums := []NamedValue{{Label: "up", Value: 1}, {Label: "down", Value: 2}}
	childEnums := []NamedValue{{Label: "active", Value: 1}}

	t.Run("inherited from parent", func(t *testing.T) {
		parent := newType("Parent")
		parent.setEnums(parentEnums)
		child := newType("Child")
		child.setParent(parent)

		got := child.EffectiveEnums()
		testutil.SliceEqual(t, parentEnums, got, "got")
	})

	t.Run("child shadows parent", func(t *testing.T) {
		parent := newType("Parent")
		parent.setEnums(parentEnums)
		child := newType("Child")
		child.setEnums(childEnums)
		child.setParent(parent)

		got := child.EffectiveEnums()
		testutil.SliceEqual(t, childEnums, got, "got")
	})

	t.Run("none anywhere", func(t *testing.T) {
		child := newType("Child")
		child.setParent(newType("Parent"))

		testutil.Nil(t, child.EffectiveEnums(), "expected nil")
	})

	t.Run("returns clone", func(t *testing.T) {
		ty := newType("T")
		ty.setEnums(parentEnums)
		got := ty.EffectiveEnums()
		got[0].Label = "mutated"
		if ty.EffectiveEnums()[0].Label == "mutated" {
			t.Error("EffectiveEnums returned a reference to internal slice")
		}
	})
}

func TestIsEnumeration(t *testing.T) {
	t.Run("integer with enums", func(t *testing.T) {
		ty := newType("Status")
		ty.setBase(BaseInteger32)
		ty.setEnums([]NamedValue{{Label: "active", Value: 1}})
		testutil.True(t, ty.IsEnumeration(), "want true")
	})

	t.Run("integer without enums", func(t *testing.T) {
		ty := newType("MyInt")
		ty.setBase(BaseInteger32)
		testutil.False(t, ty.IsEnumeration(), "want false")
	})

	t.Run("non-integer with enums", func(t *testing.T) {
		ty := newType("Weird")
		ty.setBase(BaseOctetString)
		ty.setEnums([]NamedValue{{Label: "x", Value: 1}})
		testutil.False(t, ty.IsEnumeration(), "want false")
	})

	t.Run("enums inherited from parent", func(t *testing.T) {
		parent := newType("Parent")
		parent.setBase(BaseInteger32)
		parent.setEnums([]NamedValue{{Label: "up", Value: 1}})
		child := newType("Child")
		child.setParent(parent)

		testutil.True(t, child.IsEnumeration(), "want true")
	})
}

func TestIsCounter(t *testing.T) {
	t.Run("counter32", func(t *testing.T) {
		ty := newType("C32")
		ty.setBase(BaseCounter32)
		testutil.True(t, ty.IsCounter(), "want true for Counter32")
	})

	t.Run("counter64", func(t *testing.T) {
		ty := newType("C64")
		ty.setBase(BaseCounter64)
		testutil.True(t, ty.IsCounter(), "want true for Counter64")
	})

	t.Run("gauge is not counter", func(t *testing.T) {
		ty := newType("G")
		ty.setBase(BaseGauge32)
		testutil.False(t, ty.IsCounter(), "want false for Gauge32")
	})
}
