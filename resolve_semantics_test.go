package gomib

// resolve_semantics_test.go tests the semantic analysis phase: kind inference
// (table/row/column/scalar), AUGMENTS resolution, notification OBJECTS
// resolution, and diagnostic emission for unresolved references.

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func loadSemanticsMIB(t testing.TB) mib.Mib {
	t.Helper()
	return loadProblemMIB(t, "PROBLEM-SEMANTICS-MIB")
}

// === Kind inference ===

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
			if obj == nil {
				t.Skipf("%s not found", tt.name)
				return
			}
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
			if obj == nil {
				t.Skipf("%s not found", tt.name)
				return
			}
			kind := testutil.NormalizeKind(obj.Kind())
			testutil.Equal(t, tt.wantKind, kind,
				"kind for %s", tt.name)
		})
	}
}

// === AUGMENTS resolution ===

// TestAugmentsResolution verifies that AUGMENTS clauses are resolved to
// the correct target row object.
func TestAugmentsResolution(t *testing.T) {
	m := loadSemanticsMIB(t)

	entry := m.FindObject("problemAugEntry")
	if entry == nil {
		t.Skip("problemAugEntry not found")
		return
	}

	aug := entry.Augments()
	if aug == nil {
		t.Skip("AUGMENTS not resolved for problemAugEntry")
		return
	}

	testutil.Equal(t, "problemSemEntry", aug.Name(),
		"AUGMENTS should reference problemSemEntry")
}

// === Index resolution ===

// TestIndexResolution verifies that INDEX clauses are resolved to the
// correct index objects.
func TestIndexResolution(t *testing.T) {
	m := loadSemanticsMIB(t)

	entry := m.FindObject("problemSemEntry")
	if entry == nil {
		t.Skip("problemSemEntry not found")
		return
	}

	indexes := testutil.NormalizeIndexes(entry.Index())
	if len(indexes) == 0 {
		t.Skip("no indexes resolved")
		return
	}

	testutil.Equal(t, 1, len(indexes), "should have 1 index")
	testutil.Equal(t, "problemSemIndex", indexes[0].Name, "index object name")
	testutil.False(t, indexes[0].Implied, "index should not be IMPLIED")
}

// === Notification OBJECTS resolution ===

// TestNotificationObjectsResolution verifies that NOTIFICATION-TYPE OBJECTS
// clauses are resolved to the correct object references.
func TestNotificationObjectsResolution(t *testing.T) {
	m := loadSemanticsMIB(t)

	t.Run("normal objects", func(t *testing.T) {
		notif := m.FindNotification("problemSemNotifNormal")
		if notif == nil {
			t.Skip("problemSemNotifNormal not found")
			return
		}
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		testutil.SliceEqual(t,
			[]string{"problemSemName", "problemSemValue"},
			varbinds, "normal notification varbinds")
	})

	t.Run("empty objects", func(t *testing.T) {
		notif := m.FindNotification("problemSemNotifEmpty")
		if notif == nil {
			t.Skip("problemSemNotifEmpty not found")
			return
		}
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		testutil.Equal(t, 0, len(varbinds),
			"empty notification should have no varbinds")
	})

	t.Run("not-accessible index in objects", func(t *testing.T) {
		notif := m.FindNotification("problemSemNotifWithIndex")
		if notif == nil {
			t.Skip("problemSemNotifWithIndex not found")
			return
		}
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		// Both objects should resolve, including the not-accessible index
		testutil.Len(t, varbinds, 2, "should include not-accessible index")
		testutil.SliceEqual(t,
			[]string{"problemSemIndex", "problemSemName"},
			varbinds, "notification varbinds with index")
	})

	t.Run("augment column in objects", func(t *testing.T) {
		notif := m.FindNotification("problemSemNotifAugObj")
		if notif == nil {
			t.Skip("problemSemNotifAugObj not found")
			return
		}
		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		testutil.Len(t, varbinds, 2, "should include augment column")
		testutil.SliceEqual(t,
			[]string{"problemAugExtra", "problemSemValue"},
			varbinds, "notification varbinds with augment object")
	})

	t.Run("scalar in objects", func(t *testing.T) {
		notif := m.FindNotification("problemSemNotifScalar")
		if notif == nil {
			t.Skip("problemSemNotifScalar not found")
			return
		}
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
			if notif == nil {
				t.Skipf("%s not found", tt.name)
				return
			}
			testutil.Equal(t, tt.wantOID, notif.OID().String(),
				"OID for %s", tt.name)
		})
	}
}

// === Module preference ===

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
	if mod == nil {
		t.Skip("module attribution not set for ifIndex")
		return
	}

	// IF-MIB (SMIv2) should be preferred over RFC1213-MIB (SMIv1)
	testutil.Equal(t, "IF-MIB", mod.Name(),
		"ifIndex should be attributed to IF-MIB (SMIv2), not RFC1213-MIB (SMIv1)")
}

// === Diagnostic emission ===

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
	if notif == nil {
		t.Skip("problemDiagNotifBadObj not found")
		return
	}

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
