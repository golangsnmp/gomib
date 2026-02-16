package mib

import (
	"slices"
	"testing"
)

func TestEffectiveBase(t *testing.T) {
	t.Run("direct", func(t *testing.T) {
		ty := newType("MyInt")
		ty.setBase(BaseInteger32)
		if got := ty.EffectiveBase(); got != BaseInteger32 {
			t.Errorf("got %v, want %v", got, BaseInteger32)
		}
	})

	t.Run("inherited from parent", func(t *testing.T) {
		parent := newType("Parent")
		parent.setBase(BaseOctetString)
		child := newType("Child")
		child.setParent(parent)

		if got := child.EffectiveBase(); got != BaseOctetString {
			t.Errorf("got %v, want %v", got, BaseOctetString)
		}
	})

	t.Run("inherited from grandparent", func(t *testing.T) {
		grandparent := newType("GP")
		grandparent.setBase(BaseGauge32)
		parent := newType("Parent")
		parent.setParent(grandparent)
		child := newType("Child")
		child.setParent(parent)

		if got := child.EffectiveBase(); got != BaseGauge32 {
			t.Errorf("got %v, want %v", got, BaseGauge32)
		}
	})

	t.Run("child shadows parent", func(t *testing.T) {
		parent := newType("Parent")
		parent.setBase(BaseOctetString)
		child := newType("Child")
		child.setBase(BaseInteger32)
		child.setParent(parent)

		if got := child.EffectiveBase(); got != BaseInteger32 {
			t.Errorf("got %v, want %v", got, BaseInteger32)
		}
	})

	t.Run("no base anywhere", func(t *testing.T) {
		parent := newType("Parent")
		child := newType("Child")
		child.setParent(parent)

		if got := child.EffectiveBase(); got != 0 {
			t.Errorf("got %v, want 0", got)
		}
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
		if !slices.Equal(got, parentEnums) {
			t.Errorf("got %v, want %v", got, parentEnums)
		}
	})

	t.Run("child shadows parent", func(t *testing.T) {
		parent := newType("Parent")
		parent.setEnums(parentEnums)
		child := newType("Child")
		child.setEnums(childEnums)
		child.setParent(parent)

		got := child.EffectiveEnums()
		if !slices.Equal(got, childEnums) {
			t.Errorf("got %v, want %v", got, childEnums)
		}
	})

	t.Run("none anywhere", func(t *testing.T) {
		child := newType("Child")
		child.setParent(newType("Parent"))

		if got := child.EffectiveEnums(); got != nil {
			t.Errorf("got %v, want nil", got)
		}
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
		if !ty.IsEnumeration() {
			t.Error("want true")
		}
	})

	t.Run("integer without enums", func(t *testing.T) {
		ty := newType("MyInt")
		ty.setBase(BaseInteger32)
		if ty.IsEnumeration() {
			t.Error("want false")
		}
	})

	t.Run("non-integer with enums", func(t *testing.T) {
		ty := newType("Weird")
		ty.setBase(BaseOctetString)
		ty.setEnums([]NamedValue{{Label: "x", Value: 1}})
		if ty.IsEnumeration() {
			t.Error("want false")
		}
	})

	t.Run("enums inherited from parent", func(t *testing.T) {
		parent := newType("Parent")
		parent.setBase(BaseInteger32)
		parent.setEnums([]NamedValue{{Label: "up", Value: 1}})
		child := newType("Child")
		child.setParent(parent)

		if !child.IsEnumeration() {
			t.Error("want true")
		}
	})
}

func TestIsCounter(t *testing.T) {
	t.Run("counter32", func(t *testing.T) {
		ty := newType("C32")
		ty.setBase(BaseCounter32)
		if !ty.IsCounter() {
			t.Error("want true for Counter32")
		}
	})

	t.Run("counter64", func(t *testing.T) {
		ty := newType("C64")
		ty.setBase(BaseCounter64)
		if !ty.IsCounter() {
			t.Error("want true for Counter64")
		}
	})

	t.Run("gauge is not counter", func(t *testing.T) {
		ty := newType("G")
		ty.setBase(BaseGauge32)
		if ty.IsCounter() {
			t.Error("want false for Gauge32")
		}
	})
}
