package gomib

// resolve_behavior_test.go tests resolver behavior: type chains, semantic analysis, and import shadowing.

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// === Type chain resolution ===

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
			testutil.NotNil(t, obj, "FindObject(%s)", tt.name)
			typ := obj.Type()
			testutil.NotNil(t, typ, "Type() for %s", tt.name)
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
		testutil.NotNil(t, obj, "FindObject(problemTwoLevelChain)")
		hint := obj.EffectiveDisplayHint()
		testutil.Equal(t, "255a", hint,
			"two-level chain should inherit display hint from MyString")
	})

	t.Run("inherited through chain", func(t *testing.T) {
		// MySpecialGauge -> MyFormattedGauge (has "d-2") -> Gauge32
		obj := m.FindObject("problemInheritedHint")
		testutil.NotNil(t, obj, "FindObject(problemInheritedHint)")
		hint := obj.EffectiveDisplayHint()
		testutil.Equal(t, "d-2", hint,
			"should inherit display hint from MyFormattedGauge")
	})

	t.Run("no hint on primitive", func(t *testing.T) {
		obj := m.FindObject("problemDirectInteger")
		testutil.NotNil(t, obj, "FindObject(problemDirectInteger)")
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
		testutil.NotNil(t, obj, "FindObject(problemTwoLevelChain)")
		sizes := obj.EffectiveSizes()
		testutil.NotEmpty(t, sizes, "EffectiveSizes()")
		testutil.Equal(t, 1, len(sizes), "should have 1 size range")
		testutil.Equal(t, int64(0), sizes[0].Min, "size min")
		testutil.Equal(t, int64(64), sizes[0].Max, "size max")
	})

	t.Run("inherited through chain", func(t *testing.T) {
		// MySizedLabel -> MySizedString (SIZE 1..100) -> DisplayString
		obj := m.FindObject("problemInheritedSize")
		testutil.NotNil(t, obj, "FindObject(problemInheritedSize)")
		sizes := obj.EffectiveSizes()
		testutil.NotEmpty(t, sizes, "EffectiveSizes()")
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
		testutil.NotNil(t, obj, "FindObject(problemEnumChain)")
		enums := obj.EffectiveEnums()
		testutil.NotEmpty(t, enums, "EffectiveEnums()")
		enumMap := testutil.NormalizeEnums(enums)
		testutil.Equal(t, "active", enumMap[1], "enum value 1")
		testutil.Equal(t, "inactive", enumMap[2], "enum value 2")
		testutil.Equal(t, "unknown", enumMap[3], "enum value 3")
	})

	t.Run("inline enum", func(t *testing.T) {
		obj := m.FindObject("problemInlineEnum")
		testutil.NotNil(t, obj, "FindObject(problemInlineEnum)")
		enums := obj.EffectiveEnums()
		testutil.NotEmpty(t, enums, "EffectiveEnums()")
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
		testutil.NotNil(t, obj, "FindObject(problemAppTypeChain)")
		typ := obj.Type()
		testutil.NotNil(t, typ, "Type()")
		// Counter32 is an application type - base should be Counter32, not Integer32
		testutil.Equal(t, mib.BaseCounter32, typ.EffectiveBase(),
			"Counter32 application type should be preserved through TC chain")
	})

	t.Run("Gauge32 via TC chain", func(t *testing.T) {
		// MySpecialGauge -> MyFormattedGauge -> Gauge32
		obj := m.FindObject("problemInheritedHint")
		testutil.NotNil(t, obj, "FindObject(problemInheritedHint)")
		typ := obj.Type()
		testutil.NotNil(t, typ, "Type()")
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
		testutil.NotNil(t, obj, "FindObject(problemTwoLevelChain)")
		testutil.NotNil(t, obj.Type(), "Type()")
		testutil.True(t, obj.Type().IsTextualConvention(),
			"MyString should be a TC")
	})

	t.Run("inline enum does not have TC flag", func(t *testing.T) {
		obj := m.FindObject("problemInlineEnum")
		testutil.NotNil(t, obj, "FindObject(problemInlineEnum)")
		testutil.NotNil(t, obj.Type(), "Type()")
		// Inline INTEGER { ... } resolves to the primitive INTEGER type,
		// which is not a TC
		testutil.False(t, obj.Type().IsTextualConvention(),
			"inline INTEGER enum should not be a TC")
	})
}

// === Semantic analysis ===

func loadSemanticsMIB(t testing.TB) mib.Mib {
	t.Helper()
	return loadProblemMIB(t, "PROBLEM-SEMANTICS-MIB")
}

// TestKindInferenceTableStructure verifies that the semantic phase correctly
// classifies table, row, column, and scalar objects.
func TestKindInferenceTableStructure(t *testing.T) {
	m := loadSemanticsMIB(t)

	tests := []struct {
		name     string
		wantKind string
	}{
		{"problemSemTable", "table"},
		{"problemSemEntry", "row"},
		{"problemSemIndex", "column"},
		{"problemSemName", "column"},
		{"problemSemValue", "column"},
		{"problemScalar1", "scalar"},
		{"problemScalar2", "scalar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := m.FindObject(tt.name)
			testutil.NotNil(t, obj, "FindObject(%s)", tt.name)
			kind := testutil.NormalizeKind(obj.Kind())
			testutil.Equal(t, tt.wantKind, kind,
				"kind for %s", tt.name)
		})
	}
}

// TestKindInferenceAugmentsTable verifies kind inference for AUGMENTS tables.
func TestKindInferenceAugmentsTable(t *testing.T) {
	m := loadSemanticsMIB(t)

	tests := []struct {
		name     string
		wantKind string
	}{
		{"problemAugTable", "table"},
		{"problemAugEntry", "row"},
		{"problemAugExtra", "column"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := m.FindObject(tt.name)
			testutil.NotNil(t, obj, "FindObject(%s)", tt.name)
			kind := testutil.NormalizeKind(obj.Kind())
			testutil.Equal(t, tt.wantKind, kind,
				"kind for %s", tt.name)
		})
	}
}

// TestAugmentsResolution verifies that AUGMENTS clauses are resolved to
// the correct target row object.
func TestAugmentsResolution(t *testing.T) {
	m := loadSemanticsMIB(t)

	entry := m.FindObject("problemAugEntry")
	testutil.NotNil(t, entry, "FindObject(problemAugEntry)")

	aug := entry.Augments()
	testutil.NotNil(t, aug, "Augments() for problemAugEntry")

	testutil.Equal(t, "problemSemEntry", aug.Name(),
		"AUGMENTS should reference problemSemEntry")
}

// TestIndexResolution verifies that INDEX clauses are resolved to the
// correct index objects.
func TestIndexResolution(t *testing.T) {
	m := loadSemanticsMIB(t)

	entry := m.FindObject("problemSemEntry")
	testutil.NotNil(t, entry, "FindObject(problemSemEntry)")

	indexes := testutil.NormalizeIndexes(entry.Index())
	testutil.NotEmpty(t, indexes, "NormalizeIndexes()")

	testutil.Equal(t, 1, len(indexes), "should have 1 index")
	testutil.Equal(t, "problemSemIndex", indexes[0].Name, "index object name")
	testutil.False(t, indexes[0].Implied, "index should not be IMPLIED")
}

// TestNotificationObjectsResolution verifies that NOTIFICATION-TYPE OBJECTS
// clauses are resolved to the correct object references.
func TestNotificationObjectsResolution(t *testing.T) {
	m := loadSemanticsMIB(t)

	t.Run("normal objects", func(t *testing.T) {
		notif := m.FindNotification("problemSemNotifNormal")
		testutil.NotNil(t, notif, "FindNotification(problemSemNotifNormal)")
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		testutil.SliceEqual(t,
			[]string{"problemSemName", "problemSemValue"},
			varbinds, "normal notification varbinds")
	})

	t.Run("empty objects", func(t *testing.T) {
		notif := m.FindNotification("problemSemNotifEmpty")
		testutil.NotNil(t, notif, "FindNotification(problemSemNotifEmpty)")
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		testutil.Equal(t, 0, len(varbinds),
			"empty notification should have no varbinds")
	})

	t.Run("not-accessible index in objects", func(t *testing.T) {
		notif := m.FindNotification("problemSemNotifWithIndex")
		testutil.NotNil(t, notif, "FindNotification(problemSemNotifWithIndex)")
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		// Both objects should resolve, including the not-accessible index
		testutil.Len(t, varbinds, 2, "should include not-accessible index")
		testutil.SliceEqual(t,
			[]string{"problemSemIndex", "problemSemName"},
			varbinds, "notification varbinds with index")
	})

	t.Run("augment column in objects", func(t *testing.T) {
		notif := m.FindNotification("problemSemNotifAugObj")
		testutil.NotNil(t, notif, "FindNotification(problemSemNotifAugObj)")
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		testutil.Len(t, varbinds, 2, "should include augment column")
		testutil.SliceEqual(t,
			[]string{"problemAugExtra", "problemSemValue"},
			varbinds, "notification varbinds with augment object")
	})

	t.Run("scalar in objects", func(t *testing.T) {
		notif := m.FindNotification("problemSemNotifScalar")
		testutil.NotNil(t, notif, "FindNotification(problemSemNotifScalar)")
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		testutil.Len(t, varbinds, 1, "should have 1 varbind")
		testutil.SliceEqual(t, []string{"problemScalar1"}, varbinds,
			"scalar notification varbinds")
	})
}

// TestNotificationMetadata verifies notification status and OID resolution.
func TestNotificationMetadata(t *testing.T) {
	m := loadSemanticsMIB(t)

	tests := []struct {
		name    string
		wantOID string
	}{
		{"problemSemNotifNormal", "1.3.6.1.4.1.99998.24.2.1"},
		{"problemSemNotifEmpty", "1.3.6.1.4.1.99998.24.2.2"},
		{"problemSemNotifWithIndex", "1.3.6.1.4.1.99998.24.2.3"},
		{"problemSemNotifAugObj", "1.3.6.1.4.1.99998.24.2.4"},
		{"problemSemNotifScalar", "1.3.6.1.4.1.99998.24.2.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notif := m.FindNotification(tt.name)
			testutil.NotNil(t, notif, "FindNotification(%s)", tt.name)
			testutil.Equal(t, tt.wantOID, notif.OID().String(),
				"OID for %s", tt.name)
		})
	}
}

// TestModulePreferenceSMIv2OverSMIv1 verifies that when both SMIv1 and SMIv2
// modules define the same OID, the SMIv2 module is preferred.
// This is tested via IF-MIB (SMIv2) and RFC1213-MIB (SMIv1) which both
// define objects like ifIndex.
func TestModulePreferenceSMIv2OverSMIv1(t *testing.T) {
	m := loadTestMIB(t)

	obj := m.FindObject("ifIndex")
	if obj == nil {
		t.Fatal("ifIndex not found")
	}

	mod := obj.Module()
	testutil.NotNil(t, mod, "Module() for ifIndex")

	// IF-MIB (SMIv2) should be preferred over RFC1213-MIB (SMIv1)
	testutil.Equal(t, "IF-MIB", mod.Name(),
		"ifIndex should be attributed to IF-MIB (SMIv2), not RFC1213-MIB (SMIv1)")
}

// TestDiagnosticEmissionUnresolvedType verifies that referencing a non-existent
// type emits a "type-unknown" diagnostic.
func TestDiagnosticEmissionUnresolvedType(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DIAGNOSTICS-MIB")

	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == "type-unknown" && d.Module == "PROBLEM-DIAGNOSTICS-MIB" {
			found = true
			testutil.Contains(t, d.Message, "NonExistentType",
				"diagnostic message should mention the unresolved type name")
			break
		}
	}
	testutil.True(t, found, "should emit type-unknown diagnostic for NonExistentType")

	// Verify it appears in Unresolved() too
	unresolved := unresolvedSymbols(m, "PROBLEM-DIAGNOSTICS-MIB", "type")
	testutil.True(t, unresolved["NonExistentType"],
		"NonExistentType should be in unresolved list")
}

// TestDiagnosticEmissionUnresolvedIndex verifies that referencing a
// non-existent index object emits an "index-unresolved" diagnostic.
func TestDiagnosticEmissionUnresolvedIndex(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DIAGNOSTICS-MIB")

	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == "index-unresolved" && d.Module == "PROBLEM-DIAGNOSTICS-MIB" {
			found = true
			testutil.Contains(t, d.Message, "nonExistentIndex",
				"diagnostic message should mention the unresolved index name")
			break
		}
	}
	testutil.True(t, found, "should emit index-unresolved diagnostic for nonExistentIndex")
}

// TestDiagnosticEmissionUnresolvedNotificationObject verifies that referencing
// a non-existent object in OBJECTS emits an "objects-unresolved" diagnostic.
func TestDiagnosticEmissionUnresolvedNotificationObject(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DIAGNOSTICS-MIB")

	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == "objects-unresolved" && d.Module == "PROBLEM-DIAGNOSTICS-MIB" {
			found = true
			testutil.Contains(t, d.Message, "totallyBogusObject",
				"diagnostic message should mention the unresolved object name")
			break
		}
	}
	testutil.True(t, found, "should emit objects-unresolved diagnostic for totallyBogusObject")
}

// TestDiagnosticValidObjectNoFalsePositives verifies that valid objects in the
// same module do not trigger unresolved diagnostics.
func TestDiagnosticValidObjectNoFalsePositives(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DIAGNOSTICS-MIB")

	// problemValidType uses Integer32 which is always available
	obj := m.FindObject("problemValidType")
	testutil.NotNil(t, obj, "valid object should resolve")
	if obj == nil {
		return
	}
	testutil.NotNil(t, obj.Type(), "valid object should have a type")
	if obj.Type() != nil {
		testutil.Equal(t, mib.BaseInteger32, obj.Type().EffectiveBase(),
			"valid object should have Integer32 type")
	}
}

// TestDiagnosticNotifPartialResolution verifies that a notification with
// mixed valid/invalid OBJECTS resolves the valid ones and emits diagnostics
// for the invalid ones.
func TestDiagnosticNotifPartialResolution(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DIAGNOSTICS-MIB")

	notif := m.FindNotification("problemDiagNotifBadObj")
	testutil.NotNil(t, notif, "FindNotification(problemDiagNotifBadObj)")

	varbinds := testutil.NormalizeVarbinds(notif.Objects())

	// problemDiagCol should resolve
	hasDiagCol := false
	for _, v := range varbinds {
		if v == "problemDiagCol" {
			hasDiagCol = true
		}
		// totallyBogusObject should NOT be present
		if v == "totallyBogusObject" {
			t.Error("totallyBogusObject should not appear in resolved varbinds")
		}
	}
	testutil.True(t, hasDiagCol,
		"problemDiagCol should be in resolved varbinds")
}

// === Import shadowing ===

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
	testutil.NotNil(t, obj, "FindObject(problemShadowedTypeObject)")

	typ := obj.Type()
	testutil.NotNil(t, typ, "Type() for problemShadowedTypeObject")

	// The local ShadowableType has display hint "1024a"
	// The base ShadowableType has display hint "255a"
	// If shadowing works, we should get "1024a"
	hint := obj.EffectiveDisplayHint()
	testutil.Equal(t, "1024a", hint,
		"should use local ShadowableType (1024a), not imported base (255a)")
}

// TestShadowedTypeSizeConstraint verifies that the local ShadowableType's
// size constraint (0..128) is used, not the base's (0..64).
func TestShadowedTypeSizeConstraint(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.FindObject("problemShadowedTypeObject")
	testutil.NotNil(t, obj, "FindObject(problemShadowedTypeObject)")

	sizes := obj.EffectiveSizes()
	testutil.NotEmpty(t, sizes, "EffectiveSizes()")

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
	testutil.NotNil(t, obj, "FindObject(problemShadowedTypeObject)")

	typ := obj.Type()
	testutil.NotNil(t, typ, "Type()")

	testutil.Equal(t, mib.BaseOctetString, typ.EffectiveBase(),
		"ShadowableType should resolve to OCTET STRING regardless of shadowing")
}

// TestNonShadowedImportStillWorks verifies that non-shadowed imports
// (DisplayString) still resolve correctly in the same module.
func TestNonShadowedImportStillWorks(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.FindObject("problemNonShadowedObject")
	testutil.NotNil(t, obj, "FindObject(problemNonShadowedObject)")

	typ := obj.Type()
	testutil.NotNil(t, typ, "Type() for problemNonShadowedObject")

	testutil.Equal(t, mib.BaseOctetString, typ.EffectiveBase(),
		"DisplayString should still resolve to OCTET STRING")

	// DisplayString has display hint "255a"
	hint := obj.EffectiveDisplayHint()
	testutil.Equal(t, "255a", hint,
		"DisplayString should have display hint 255a (imported, not shadowed)")
}

// TestBaseModuleTypeNotAffected verifies that the base module's own
// ShadowableType is unaffected by the shadowing module's redefinition.
func TestBaseModuleTypeNotAffected(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.FindObject("problemBaseTypedObject")
	testutil.NotNil(t, obj, "FindObject(problemBaseTypedObject)")

	typ := obj.Type()
	testutil.NotNil(t, typ, "Type() for problemBaseTypedObject")

	// The base module's object should use its own ShadowableType (hint "255a")
	hint := obj.EffectiveDisplayHint()
	testutil.Equal(t, "255a", hint,
		"base module object should use base ShadowableType (255a), not the shadowing module's version")
}

// TestShadowingModuleScalarResolves verifies that basic type resolution
// still works in a module with a shadowed import.
func TestShadowingModuleScalarResolves(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.FindObject("problemShadowScalar")
	testutil.NotNil(t, obj, "FindObject(problemShadowScalar)")

	typ := obj.Type()
	testutil.NotNil(t, typ, "Type()")
	testutil.Equal(t, mib.BaseInteger32, typ.EffectiveBase(),
		"Integer32 scalar should resolve normally")
}
