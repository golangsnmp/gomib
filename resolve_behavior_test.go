package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func loadTypeChainsMIB(t testing.TB) mib.Mib {
	t.Helper()
	return loadProblemMIB(t, "PROBLEM-TYPECHAINS-MIB")
}

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
			obj := m.Object(tt.name)
			testutil.NotNil(t, obj, "Object(%s)", tt.name)
			typ := obj.Type()
			testutil.NotNil(t, typ, "Type() for %s", tt.name)
			testutil.Equal(t, tt.wantBase, typ.EffectiveBase(),
				"base type for %s", tt.name)
		})
	}
}

func TestTypeChainDisplayHintInheritance(t *testing.T) {
	m := loadTypeChainsMIB(t)

	t.Run("direct hint", func(t *testing.T) {
		// MyString has DISPLAY-HINT "255a"
		obj := m.Object("problemTwoLevelChain")
		testutil.NotNil(t, obj, "Object(problemTwoLevelChain)")
		hint := obj.EffectiveDisplayHint()
		testutil.Equal(t, "255a", hint,
			"two-level chain should inherit display hint from MyString")
	})

	t.Run("inherited through chain", func(t *testing.T) {
		// MySpecialGauge -> MyFormattedGauge (has "d-2") -> Gauge32
		obj := m.Object("problemInheritedHint")
		testutil.NotNil(t, obj, "Object(problemInheritedHint)")
		hint := obj.EffectiveDisplayHint()
		testutil.Equal(t, "d-2", hint,
			"should inherit display hint from MyFormattedGauge")
	})

	t.Run("no hint on primitive", func(t *testing.T) {
		obj := m.Object("problemDirectInteger")
		testutil.NotNil(t, obj, "Object(problemDirectInteger)")
		testutil.Equal(t, "", obj.EffectiveDisplayHint(),
			"direct Integer32 should have no display hint")
	})
}

func TestTypeChainSizeInheritance(t *testing.T) {
	m := loadTypeChainsMIB(t)

	t.Run("direct size", func(t *testing.T) {
		// MyString has SIZE (0..64), object uses MyString
		obj := m.Object("problemTwoLevelChain")
		testutil.NotNil(t, obj, "Object(problemTwoLevelChain)")
		sizes := obj.EffectiveSizes()
		testutil.NotEmpty(t, sizes, "EffectiveSizes()")
		testutil.Equal(t, 1, len(sizes), "should have 1 size range")
		testutil.Equal(t, int64(0), sizes[0].Min, "size min")
		testutil.Equal(t, int64(64), sizes[0].Max, "size max")
	})

	t.Run("inherited through chain", func(t *testing.T) {
		// MySizedLabel -> MySizedString (SIZE 1..100) -> DisplayString
		obj := m.Object("problemInheritedSize")
		testutil.NotNil(t, obj, "Object(problemInheritedSize)")
		sizes := obj.EffectiveSizes()
		testutil.NotEmpty(t, sizes, "EffectiveSizes()")
		testutil.Equal(t, 1, len(sizes), "should have 1 size range")
		testutil.Equal(t, int64(1), sizes[0].Min, "size min from MySizedString")
		testutil.Equal(t, int64(100), sizes[0].Max, "size max from MySizedString")
	})
}

func TestTypeChainEnumInheritance(t *testing.T) {
	m := loadTypeChainsMIB(t)

	t.Run("TC enum chain", func(t *testing.T) {
		// MyFilteredStatus -> MyStatus -> INTEGER { active(1), inactive(2), unknown(3) }
		obj := m.Object("problemEnumChain")
		testutil.NotNil(t, obj, "Object(problemEnumChain)")
		enums := obj.EffectiveEnums()
		testutil.NotEmpty(t, enums, "EffectiveEnums()")
		enumMap := testutil.NormalizeEnums(enums)
		testutil.Equal(t, "active", enumMap[1], "enum value 1")
		testutil.Equal(t, "inactive", enumMap[2], "enum value 2")
		testutil.Equal(t, "unknown", enumMap[3], "enum value 3")
	})

	t.Run("inline enum", func(t *testing.T) {
		obj := m.Object("problemInlineEnum")
		testutil.NotNil(t, obj, "Object(problemInlineEnum)")
		enums := obj.EffectiveEnums()
		testutil.NotEmpty(t, enums, "EffectiveEnums()")
		enumMap := testutil.NormalizeEnums(enums)
		testutil.Equal(t, "up", enumMap[1], "inline enum value 1")
		testutil.Equal(t, "down", enumMap[2], "inline enum value 2")
	})
}

func TestTypeChainApplicationTypePreservation(t *testing.T) {
	m := loadTypeChainsMIB(t)

	t.Run("Counter32 via TC", func(t *testing.T) {
		// MyCounter -> Counter32
		obj := m.Object("problemAppTypeChain")
		testutil.NotNil(t, obj, "Object(problemAppTypeChain)")
		typ := obj.Type()
		testutil.NotNil(t, typ, "Type()")
		// Counter32 is an application type - base should be Counter32, not Integer32
		testutil.Equal(t, mib.BaseCounter32, typ.EffectiveBase(),
			"Counter32 application type should be preserved through TC chain")
	})

	t.Run("Gauge32 via TC chain", func(t *testing.T) {
		// MySpecialGauge -> MyFormattedGauge -> Gauge32
		obj := m.Object("problemInheritedHint")
		testutil.NotNil(t, obj, "Object(problemInheritedHint)")
		typ := obj.Type()
		testutil.NotNil(t, typ, "Type()")
		testutil.Equal(t, mib.BaseGauge32, typ.EffectiveBase(),
			"Gauge32 application type should be preserved through two-level TC chain")
	})
}

func TestTypeChainTCFlagPropagation(t *testing.T) {
	m := loadTypeChainsMIB(t)

	t.Run("TC type has flag", func(t *testing.T) {
		obj := m.Object("problemTwoLevelChain")
		testutil.NotNil(t, obj, "Object(problemTwoLevelChain)")
		testutil.NotNil(t, obj.Type(), "Type()")
		testutil.True(t, obj.Type().IsTextualConvention(),
			"MyString should be a TC")
	})

	t.Run("inline enum does not have TC flag", func(t *testing.T) {
		obj := m.Object("problemInlineEnum")
		testutil.NotNil(t, obj, "Object(problemInlineEnum)")
		testutil.NotNil(t, obj.Type(), "Type()")
		// Inline INTEGER { ... } resolves to the primitive INTEGER type,
		// which is not a TC
		testutil.False(t, obj.Type().IsTextualConvention(),
			"inline INTEGER enum should not be a TC")
	})
}

func loadSemanticsMIB(t testing.TB) mib.Mib {
	t.Helper()
	return loadProblemMIB(t, "PROBLEM-SEMANTICS-MIB")
}

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
			obj := m.Object(tt.name)
			testutil.NotNil(t, obj, "Object(%s)", tt.name)
			kind := testutil.NormalizeKind(obj.Kind())
			testutil.Equal(t, tt.wantKind, kind,
				"kind for %s", tt.name)
		})
	}
}

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
			obj := m.Object(tt.name)
			testutil.NotNil(t, obj, "Object(%s)", tt.name)
			kind := testutil.NormalizeKind(obj.Kind())
			testutil.Equal(t, tt.wantKind, kind,
				"kind for %s", tt.name)
		})
	}
}

func TestAugmentsResolution(t *testing.T) {
	m := loadSemanticsMIB(t)

	entry := m.Object("problemAugEntry")
	testutil.NotNil(t, entry, "Object(problemAugEntry)")

	aug := entry.Augments()
	testutil.NotNil(t, aug, "Augments() for problemAugEntry")

	testutil.Equal(t, "problemSemEntry", aug.Name(),
		"AUGMENTS should reference problemSemEntry")
}

func TestIndexResolution(t *testing.T) {
	m := loadSemanticsMIB(t)

	entry := m.Object("problemSemEntry")
	testutil.NotNil(t, entry, "Object(problemSemEntry)")

	indexes := testutil.NormalizeIndexes(entry.Index())
	testutil.NotEmpty(t, indexes, "NormalizeIndexes()")

	testutil.Equal(t, 1, len(indexes), "should have 1 index")
	testutil.Equal(t, "problemSemIndex", indexes[0].Name, "index object name")
	testutil.False(t, indexes[0].Implied, "index should not be IMPLIED")
}

func TestNotificationObjectsResolution(t *testing.T) {
	m := loadSemanticsMIB(t)

	t.Run("normal objects", func(t *testing.T) {
		notif := m.Notification("problemSemNotifNormal")
		testutil.NotNil(t, notif, "Notification(problemSemNotifNormal)")
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		testutil.SliceEqual(t,
			[]string{"problemSemName", "problemSemValue"},
			varbinds, "normal notification varbinds")
	})

	t.Run("empty objects", func(t *testing.T) {
		notif := m.Notification("problemSemNotifEmpty")
		testutil.NotNil(t, notif, "Notification(problemSemNotifEmpty)")
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		testutil.Equal(t, 0, len(varbinds),
			"empty notification should have no varbinds")
	})

	t.Run("not-accessible index in objects", func(t *testing.T) {
		notif := m.Notification("problemSemNotifWithIndex")
		testutil.NotNil(t, notif, "Notification(problemSemNotifWithIndex)")
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		// Both objects should resolve, including the not-accessible index
		testutil.Len(t, varbinds, 2, "should include not-accessible index")
		testutil.SliceEqual(t,
			[]string{"problemSemIndex", "problemSemName"},
			varbinds, "notification varbinds with index")
	})

	t.Run("augment column in objects", func(t *testing.T) {
		notif := m.Notification("problemSemNotifAugObj")
		testutil.NotNil(t, notif, "Notification(problemSemNotifAugObj)")
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		testutil.Len(t, varbinds, 2, "should include augment column")
		testutil.SliceEqual(t,
			[]string{"problemAugExtra", "problemSemValue"},
			varbinds, "notification varbinds with augment object")
	})

	t.Run("scalar in objects", func(t *testing.T) {
		notif := m.Notification("problemSemNotifScalar")
		testutil.NotNil(t, notif, "Notification(problemSemNotifScalar)")
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		testutil.Len(t, varbinds, 1, "should have 1 varbind")
		testutil.SliceEqual(t, []string{"problemScalar1"}, varbinds,
			"scalar notification varbinds")
	})
}

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
			notif := m.Notification(tt.name)
			testutil.NotNil(t, notif, "Notification(%s)", tt.name)
			testutil.Equal(t, tt.wantOID, notif.OID().String(),
				"OID for %s", tt.name)
		})
	}
}

func TestModulePreferenceSMIv2OverSMIv1(t *testing.T) {
	m := loadTestMIB(t)

	obj := m.Object("ifIndex")
	if obj == nil {
		t.Fatal("ifIndex not found")
	}

	mod := obj.Module()
	testutil.NotNil(t, mod, "Module() for ifIndex")

	testutil.Equal(t, "IF-MIB", mod.Name(),
		"ifIndex should be attributed to IF-MIB (SMIv2), not RFC1213-MIB (SMIv1)")
}

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

	unresolved := unresolvedSymbols(m, "PROBLEM-DIAGNOSTICS-MIB", mib.UnresolvedType)
	testutil.True(t, unresolved["NonExistentType"],
		"NonExistentType should be in unresolved list")
}

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

func TestDiagnosticValidObjectNoFalsePositives(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DIAGNOSTICS-MIB")

	obj := m.Object("problemValidType")
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

func TestDiagnosticNotifPartialResolution(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DIAGNOSTICS-MIB")

	notif := m.Notification("problemDiagNotifBadObj")
	testutil.NotNil(t, notif, "Notification(problemDiagNotifBadObj)")

	varbinds := testutil.NormalizeVarbinds(notif.Objects())

	hasDiagCol := false
	for _, v := range varbinds {
		if v == "problemDiagCol" {
			hasDiagCol = true
		}
		if v == "totallyBogusObject" {
			t.Error("totallyBogusObject should not appear in resolved varbinds")
		}
	}
	testutil.True(t, hasDiagCol,
		"problemDiagCol should be in resolved varbinds")
}

func loadShadowingMIB(t testing.TB) mib.Mib {
	t.Helper()
	return loadProblemMIB(t, "PROBLEM-SHADOWING-MIB")
}

func TestShadowedTypeLocalDefinitionWins(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.Object("problemShadowedTypeObject")
	testutil.NotNil(t, obj, "Object(problemShadowedTypeObject)")

	typ := obj.Type()
	testutil.NotNil(t, typ, "Type() for problemShadowedTypeObject")

	// The local ShadowableType has display hint "1024a"
	// The base ShadowableType has display hint "255a"
	// If shadowing works, we should get "1024a"
	hint := obj.EffectiveDisplayHint()
	testutil.Equal(t, "1024a", hint,
		"should use local ShadowableType (1024a), not imported base (255a)")
}

func TestShadowedTypeSizeConstraint(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.Object("problemShadowedTypeObject")
	testutil.NotNil(t, obj, "Object(problemShadowedTypeObject)")

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

func TestShadowedTypeBaseType(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.Object("problemShadowedTypeObject")
	testutil.NotNil(t, obj, "Object(problemShadowedTypeObject)")

	typ := obj.Type()
	testutil.NotNil(t, typ, "Type()")

	testutil.Equal(t, mib.BaseOctetString, typ.EffectiveBase(),
		"ShadowableType should resolve to OCTET STRING regardless of shadowing")
}

func TestNonShadowedImportStillWorks(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.Object("problemNonShadowedObject")
	testutil.NotNil(t, obj, "Object(problemNonShadowedObject)")

	typ := obj.Type()
	testutil.NotNil(t, typ, "Type() for problemNonShadowedObject")

	testutil.Equal(t, mib.BaseOctetString, typ.EffectiveBase(),
		"DisplayString should still resolve to OCTET STRING")

	// DisplayString has display hint "255a"
	hint := obj.EffectiveDisplayHint()
	testutil.Equal(t, "255a", hint,
		"DisplayString should have display hint 255a (imported, not shadowed)")
}

func TestBaseModuleTypeNotAffected(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.Object("problemBaseTypedObject")
	testutil.NotNil(t, obj, "Object(problemBaseTypedObject)")

	typ := obj.Type()
	testutil.NotNil(t, typ, "Type() for problemBaseTypedObject")

	// The base module's object should use its own ShadowableType (hint "255a")
	hint := obj.EffectiveDisplayHint()
	testutil.Equal(t, "255a", hint,
		"base module object should use base ShadowableType (255a), not the shadowing module's version")
}

func TestShadowingModuleScalarResolves(t *testing.T) {
	m := loadShadowingMIB(t)

	obj := m.Object("problemShadowScalar")
	testutil.NotNil(t, obj, "Object(problemShadowScalar)")

	typ := obj.Type()
	testutil.NotNil(t, typ, "Type()")
	testutil.Equal(t, mib.BaseInteger32, typ.EffectiveBase(),
		"Integer32 scalar should resolve normally")
}
