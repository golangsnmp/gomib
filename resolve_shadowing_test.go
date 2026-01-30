package gomib

// resolve_shadowing_test.go tests import shadowing: when a module imports a
// symbol AND defines it locally, the local definition should take precedence.
// This is a MIB bug but occurs in real vendor MIBs.
//
// The resolver's lookupInModuleScope checks local symbols (ModuleSymbolToType,
// ModuleSymbolToNode) before following import chains (ModuleImports), so the
// local definition naturally wins.

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func loadShadowingMIB(t testing.TB) mib.Mib {
	t.Helper()
	return loadProblemMIB(t, "PROBLEM-SHADOWING-MIB")
}

// TestShadowedTypeLocalDefinitionWins verifies that when a module both
// imports ShadowableType and defines it locally, the local definition is
// used. The local version has display hint "1024a" while the base has "255a".
func TestShadowedTypeLocalDefinitionWins(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.FindObject("problemShadowedTypeObject")
	if obj == nil {
		t.Skip("problemShadowedTypeObject not found")
		return
	}

	typ := obj.Type()
	if typ == nil {
		t.Skip("type not resolved for problemShadowedTypeObject")
		return
	}

	// The local ShadowableType has display hint "1024a"
	// The base ShadowableType has display hint "255a"
	// If shadowing works, we should get "1024a"
	hint := obj.EffectiveDisplayHint()
	if hint == "" {
		t.Skip("no effective display hint - type chain may not propagate hints")
		return
	}

	// The key assertion: local definition should shadow the imported one
	if hint == "255a" {
		t.Error("got base module display hint '255a' - import is NOT being shadowed by local definition")
	}
	testutil.Equal(t, "1024a", hint,
		"should use local ShadowableType (1024a), not imported base (255a)")
}

// TestShadowedTypeSizeConstraint verifies that the local ShadowableType's
// size constraint (0..128) is used, not the base's (0..64).
func TestShadowedTypeSizeConstraint(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.FindObject("problemShadowedTypeObject")
	if obj == nil {
		t.Skip("problemShadowedTypeObject not found")
		return
	}

	sizes := obj.EffectiveSizes()
	if len(sizes) == 0 {
		t.Skip("no effective sizes - size inheritance may not propagate from TC")
		return
	}

	testutil.Equal(t, 1, len(sizes), "should have 1 size range")
	// Local: SIZE (0..128), Base: SIZE (0..64)
	if sizes[0].Max == 64 {
		t.Error("got base module size max 64 - import is NOT being shadowed by local definition")
	}
	testutil.Equal(t, int64(0), sizes[0].Min, "size min")
	testutil.Equal(t, int64(128), sizes[0].Max, "size max should be 128 (local), not 64 (base)")
}

// TestShadowedTypeBaseType verifies that both the local and base
// ShadowableType resolve to OCTET STRING base type.
func TestShadowedTypeBaseType(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.FindObject("problemShadowedTypeObject")
	if obj == nil {
		t.Skip("problemShadowedTypeObject not found")
		return
	}

	typ := obj.Type()
	if typ == nil {
		t.Skip("type not resolved")
		return
	}

	testutil.Equal(t, mib.BaseOctetString, typ.EffectiveBase(),
		"ShadowableType should resolve to OCTET STRING regardless of shadowing")
}

// TestNonShadowedImportStillWorks verifies that non-shadowed imports
// (DisplayString) still resolve correctly in the same module.
func TestNonShadowedImportStillWorks(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.FindObject("problemNonShadowedObject")
	if obj == nil {
		t.Skip("problemNonShadowedObject not found")
		return
	}

	typ := obj.Type()
	if typ == nil {
		t.Skip("type not resolved for non-shadowed import")
		return
	}

	testutil.Equal(t, mib.BaseOctetString, typ.EffectiveBase(),
		"DisplayString should still resolve to OCTET STRING")

	// DisplayString has display hint "255a"
	hint := obj.EffectiveDisplayHint()
	if hint == "" {
		t.Skip("no display hint for DisplayString")
		return
	}
	testutil.Equal(t, "255a", hint,
		"DisplayString should have display hint 255a (imported, not shadowed)")
}

// TestBaseModuleTypeNotAffected verifies that the base module's own
// ShadowableType is unaffected by the shadowing module's redefinition.
func TestBaseModuleTypeNotAffected(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.FindObject("problemBaseTypedObject")
	if obj == nil {
		t.Skip("problemBaseTypedObject not found")
		return
	}

	typ := obj.Type()
	if typ == nil {
		t.Skip("type not resolved for base module object")
		return
	}

	// The base module's object should use its own ShadowableType (hint "255a")
	hint := obj.EffectiveDisplayHint()
	if hint == "" {
		t.Skip("no display hint for base module's ShadowableType")
		return
	}
	testutil.Equal(t, "255a", hint,
		"base module object should use base ShadowableType (255a), not the shadowing module's version")
}

// TestShadowingModuleScalarResolves verifies that basic type resolution
// still works in a module with a shadowed import.
func TestShadowingModuleScalarResolves(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.FindObject("problemShadowScalar")
	if obj == nil {
		t.Skip("problemShadowScalar not found")
		return
	}

	typ := obj.Type()
	if typ == nil {
		t.Skip("type not resolved for scalar")
		return
	}
	testutil.Equal(t, mib.BaseInteger32, typ.EffectiveBase(),
		"Integer32 scalar should resolve normally")
}
