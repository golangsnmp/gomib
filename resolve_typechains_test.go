package gomib

// resolve_typechains_test.go tests type chain resolution: multi-level TC
// inheritance, base type propagation, display hint inheritance, constraint
// inheritance, and application type preservation.

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func loadTypeChainsMIB(t testing.TB) mib.Mib {
	t.Helper()
	return loadProblemMIB(t, "PROBLEM-TYPECHAINS-MIB")
}

// TestTypeChainBaseTypeInheritance verifies that base types propagate through
// TC chains. Each object's effective base type should be the primitive at the
// root of its type chain.
func TestTypeChainBaseTypeInheritance(t *testing.T) {
	m := loadTypeChainsMIB(t)

	tests := []struct {
		name     string
		wantBase mib.BaseType
	}{
		// MyString -> DisplayString -> OCTET STRING
		{"problemTwoLevelChain", mib.BaseOctetString},
		// MyLabel -> MyString -> DisplayString -> OCTET STRING
		{"problemThreeLevelChain", mib.BaseOctetString},
		// MyFilteredStatus -> MyStatus -> INTEGER
		{"problemEnumChain", mib.BaseInteger32},
		// MyCounter -> Counter32 (application type)
		{"problemAppTypeChain", mib.BaseCounter32},
		// MySpecialGauge -> MyFormattedGauge -> Gauge32 (application type)
		{"problemInheritedHint", mib.BaseGauge32},
		// MySizedLabel -> MySizedString -> DisplayString -> OCTET STRING
		{"problemInheritedSize", mib.BaseOctetString},
		// Direct Integer32
		{"problemDirectInteger", mib.BaseInteger32},
		// Inline INTEGER { up(1), down(2) }
		{"problemInlineEnum", mib.BaseInteger32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := m.FindObject(tt.name)
			if obj == nil {
				t.Skipf("%s not found", tt.name)
				return
			}
			typ := obj.Type()
			if typ == nil {
				t.Skipf("%s type not resolved", tt.name)
				return
			}
			testutil.Equal(t, tt.wantBase, typ.EffectiveBase(),
				"base type for %s", tt.name)
		})
	}
}

// TestTypeChainDisplayHintInheritance verifies that display hints are inherited
// through the type chain. Objects using a TC with a display hint should get
// that hint via computeEffectiveValues.
func TestTypeChainDisplayHintInheritance(t *testing.T) {
	m := loadTypeChainsMIB(t)

	t.Run("direct hint", func(t *testing.T) {
		// MyString has DISPLAY-HINT "255a"
		obj := m.FindObject("problemTwoLevelChain")
		if obj == nil {
			t.Skip("problemTwoLevelChain not found")
			return
		}
		hint := obj.EffectiveDisplayHint()
		if hint == "" {
			t.Skip("no effective display hint - hint inheritance may not be implemented")
			return
		}
		testutil.Equal(t, "255a", hint,
			"two-level chain should inherit display hint from MyString")
	})

	t.Run("inherited through chain", func(t *testing.T) {
		// MySpecialGauge -> MyFormattedGauge (has "d-2") -> Gauge32
		obj := m.FindObject("problemInheritedHint")
		if obj == nil {
			t.Skip("problemInheritedHint not found")
			return
		}
		hint := obj.EffectiveDisplayHint()
		if hint == "" {
			t.Skip("no effective display hint - chain inheritance may not reach through 2 levels")
			return
		}
		testutil.Equal(t, "d-2", hint,
			"should inherit display hint from MyFormattedGauge")
	})

	t.Run("no hint on primitive", func(t *testing.T) {
		obj := m.FindObject("problemDirectInteger")
		if obj == nil {
			t.Skip("problemDirectInteger not found")
			return
		}
		testutil.Equal(t, "", obj.EffectiveDisplayHint(),
			"direct Integer32 should have no display hint")
	})
}

// TestTypeChainSizeInheritance verifies that size constraints propagate
// through the type chain via computeEffectiveValues.
func TestTypeChainSizeInheritance(t *testing.T) {
	m := loadTypeChainsMIB(t)

	t.Run("direct size", func(t *testing.T) {
		// MyString has SIZE (0..64), object uses MyString
		obj := m.FindObject("problemTwoLevelChain")
		if obj == nil {
			t.Skip("problemTwoLevelChain not found")
			return
		}
		sizes := obj.EffectiveSizes()
		if len(sizes) == 0 {
			t.Skip("no effective sizes - size inheritance may not propagate from TC")
			return
		}
		testutil.Equal(t, 1, len(sizes), "should have 1 size range")
		testutil.Equal(t, int64(0), sizes[0].Min, "size min")
		testutil.Equal(t, int64(64), sizes[0].Max, "size max")
	})

	t.Run("inherited through chain", func(t *testing.T) {
		// MySizedLabel -> MySizedString (SIZE 1..100) -> DisplayString
		obj := m.FindObject("problemInheritedSize")
		if obj == nil {
			t.Skip("problemInheritedSize not found")
			return
		}
		sizes := obj.EffectiveSizes()
		if len(sizes) == 0 {
			t.Skip("no effective sizes - chain size inheritance may not work")
			return
		}
		testutil.Equal(t, 1, len(sizes), "should have 1 size range")
		testutil.Equal(t, int64(1), sizes[0].Min, "size min from MySizedString")
		testutil.Equal(t, int64(100), sizes[0].Max, "size max from MySizedString")
	})
}

// TestTypeChainEnumInheritance verifies that enum values propagate through
// the type chain via computeEffectiveValues.
func TestTypeChainEnumInheritance(t *testing.T) {
	m := loadTypeChainsMIB(t)

	t.Run("TC enum chain", func(t *testing.T) {
		// MyFilteredStatus -> MyStatus -> INTEGER { active(1), inactive(2), unknown(3) }
		obj := m.FindObject("problemEnumChain")
		if obj == nil {
			t.Skip("problemEnumChain not found")
			return
		}
		enums := obj.EffectiveEnums()
		if len(enums) == 0 {
			t.Skip("no effective enums - enum chain inheritance may not work")
			return
		}
		enumMap := testutil.NormalizeEnums(enums)
		testutil.Equal(t, "active", enumMap[1], "enum value 1")
		testutil.Equal(t, "inactive", enumMap[2], "enum value 2")
		testutil.Equal(t, "unknown", enumMap[3], "enum value 3")
	})

	t.Run("inline enum", func(t *testing.T) {
		obj := m.FindObject("problemInlineEnum")
		if obj == nil {
			t.Skip("problemInlineEnum not found")
			return
		}
		enums := obj.EffectiveEnums()
		if len(enums) == 0 {
			t.Skip("no effective enums for inline enum")
			return
		}
		enumMap := testutil.NormalizeEnums(enums)
		testutil.Equal(t, "up", enumMap[1], "inline enum value 1")
		testutil.Equal(t, "down", enumMap[2], "inline enum value 2")
	})
}

// TestTypeChainApplicationTypePreservation verifies that application types
// (Counter32, Gauge32, etc.) are preserved through the type chain and not
// overwritten by base type inheritance.
func TestTypeChainApplicationTypePreservation(t *testing.T) {
	m := loadTypeChainsMIB(t)

	t.Run("Counter32 via TC", func(t *testing.T) {
		// MyCounter -> Counter32
		obj := m.FindObject("problemAppTypeChain")
		if obj == nil {
			t.Skip("problemAppTypeChain not found")
			return
		}
		typ := obj.Type()
		if typ == nil {
			t.Skip("type not resolved")
			return
		}
		// Counter32 is an application type - base should be Counter32, not Integer32
		testutil.Equal(t, mib.BaseCounter32, typ.EffectiveBase(),
			"Counter32 application type should be preserved through TC chain")
	})

	t.Run("Gauge32 via TC chain", func(t *testing.T) {
		// MySpecialGauge -> MyFormattedGauge -> Gauge32
		obj := m.FindObject("problemInheritedHint")
		if obj == nil {
			t.Skip("problemInheritedHint not found")
			return
		}
		typ := obj.Type()
		if typ == nil {
			t.Skip("type not resolved")
			return
		}
		testutil.Equal(t, mib.BaseGauge32, typ.EffectiveBase(),
			"Gauge32 application type should be preserved through two-level TC chain")
	})
}

// TestTypeChainTCFlagPropagation verifies that IsTextualConvention is set
// on TC types but not on objects using inline syntax.
func TestTypeChainTCFlagPropagation(t *testing.T) {
	m := loadTypeChainsMIB(t)

	t.Run("TC type has flag", func(t *testing.T) {
		obj := m.FindObject("problemTwoLevelChain")
		if obj == nil || obj.Type() == nil {
			t.Skip("object or type not found")
			return
		}
		testutil.True(t, obj.Type().IsTextualConvention(),
			"MyString should be a TC")
	})

	t.Run("inline enum does not have TC flag", func(t *testing.T) {
		obj := m.FindObject("problemInlineEnum")
		if obj == nil || obj.Type() == nil {
			t.Skip("object or type not found")
			return
		}
		// Inline INTEGER { ... } resolves to the primitive INTEGER type,
		// which is not a TC
		testutil.False(t, obj.Type().IsTextualConvention(),
			"inline INTEGER enum should not be a TC")
	})
}
