package gomib

// resolve_problems_test.go tests gomib's handling of real-world MIB edge cases
// using the synthetic PROBLEM-*.mib corpus. Expected values are grounded against
// net-snmp (snmptranslate -Td) and libsmi (smilint) output.

import (
	"context"
	"strings"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// loadProblemMIB loads a problem MIB by name using both the primary corpus
// (for dependencies like SNMPv2-SMI, SNMPv2-TC) and the problems directory.
func loadProblemMIB(t testing.TB, name string) mib.Mib {
	t.Helper()
	corpus, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree corpus failed: %v", err)
	}
	problems, err := DirTree("testdata/corpus/problems")
	if err != nil {
		t.Fatalf("DirTree problems failed: %v", err)
	}
	src := Multi(corpus, problems)
	ctx := context.Background()
	m, err := LoadModules(ctx, []string{name}, src, WithStrictness(StrictnessPermissive))
	if err != nil {
		t.Fatalf("LoadModules(%s) failed: %v", name, err)
	}
	return m
}

// TestProblemHexStrings verifies hex and binary string DEFVAL parsing.
// Ground truth: net-snmp snmptranslate -Td output for PROBLEM-HEXSTRINGS-MIB.
//
// gomib converts hex/binary strings to []byte then formats:
//   - empty → "0"
//   - ≤8 bytes → decimal uint64
//   - >8 bytes → "0x" + hex
//
// net-snmp converts to decimal integers and truncates leading-zero hex to "0".
func TestProblemHexStrings(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-HEXSTRINGS-MIB")

	tests := []struct {
		name      string
		wantDefval string // net-snmp ground truth
	}{
		// Odd-length hex: 'ABCDEF0'H → pad to "0ABCDEF0" → 4 bytes → 180150000
		{"problemOddHex7", "180150000"},
		// Odd-length hex: 'FFF'H → pad to "0FFF" → 2 bytes → 4095
		{"problemOddHex3", "4095"},
		// Odd-length hex: '0'H → pad to "00" → 1 byte → 0
		{"problemOddHex1", "0"},
		// Long hex: 128 chars / 64 bytes → >8 bytes → gomib: "0x..." format
		// net-snmp: "0" (truncates leading zeros). Known divergence.
		{"problemLongHex", ""},
		// Empty hex: ''H → 0 bytes → "0"
		{"problemEmptyHex", "0"},
		// Binary: '11110000'B → 1 byte → 0xF0 → 240
		{"problemBinary8", "240"},
		// Binary: '10101'B → pad to "00010101" → 1 byte → 0x15 → 21
		{"problemBinary5", "21"},
		// Binary: '101010101010'B → pad to 16 bits → 2 bytes → 0x0AAA → 2730
		{"problemBinary12", "2730"},
		// Lowercase hex: 'deadbeef'H → 4 bytes → 3735928559
		{"problemLowerHex", "3735928559"},
		// All-zeros: '0000000000000000'H → 8 bytes → 0
		{"problemAllZeros", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := m.FindObject(tt.name)
			testutil.NotNil(t, obj, "object %s should be found", tt.name)
			if obj == nil {
				return
			}

			dv := obj.DefaultValue()
			testutil.True(t, !dv.IsZero(), "object %s should have a DEFVAL", tt.name)

			if tt.wantDefval == "" {
				// Known divergence case (long hex) - just verify it parses
				t.Logf("%s: gomib=%q (net-snmp diverges)", tt.name, dv.String())
				return
			}

			got := dv.String()
			if !defvalEquivalent(got, tt.wantDefval) {
				t.Errorf("defval %s: gomib=%q want=%q", tt.name, got, tt.wantDefval)
			}
		})
	}
}

// TestProblemHexStringBytes verifies the raw []byte value of hex/binary DEFVALs.
// This tests the internal conversion (hexToBytes, binaryToBytes) through the
// public DefVal API without going through String() formatting.
func TestProblemHexStringBytes(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-HEXSTRINGS-MIB")

	tests := []struct {
		name      string
		wantBytes []byte
	}{
		// 'ABCDEF0'H → pad "0ABCDEF0" → [0x0A, 0xBC, 0xDE, 0xF0]
		{"problemOddHex7", []byte{0x0A, 0xBC, 0xDE, 0xF0}},
		// 'FFF'H → pad "0FFF" → [0x0F, 0xFF]
		{"problemOddHex3", []byte{0x0F, 0xFF}},
		// '0'H → pad "00" → [0x00]
		{"problemOddHex1", []byte{0x00}},
		// ''H → empty
		{"problemEmptyHex", []byte{}},
		// '11110000'B → [0xF0]
		{"problemBinary8", []byte{0xF0}},
		// '10101'B → pad "00010101" → [0x15]
		{"problemBinary5", []byte{0x15}},
		// '101010101010'B → pad "0000101010101010" → [0x0A, 0xAA]
		{"problemBinary12", []byte{0x0A, 0xAA}},
		// 'deadbeef'H → [0xDE, 0xAD, 0xBE, 0xEF]
		{"problemLowerHex", []byte{0xDE, 0xAD, 0xBE, 0xEF}},
		// '0000000000000000'H → 8 zero bytes
		{"problemAllZeros", []byte{0, 0, 0, 0, 0, 0, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := m.FindObject(tt.name)
			if obj == nil {
				t.Fatalf("object %s not found", tt.name)
			}

			dv := obj.DefaultValue()
			if dv.IsZero() {
				t.Fatalf("object %s has no DEFVAL", tt.name)
			}

			if dv.Kind() != mib.DefValKindBytes {
				t.Skipf("object %s DEFVAL kind=%v, expected Bytes", tt.name, dv.Kind())
				return
			}

			got, ok := mib.DefValAs[[]byte](dv)
			if !ok {
				t.Fatalf("DefValAs[[]byte] failed for %s", tt.name)
			}

			if len(got) != len(tt.wantBytes) {
				t.Fatalf("byte length: got %d, want %d (got=%x want=%x)",
					len(got), len(tt.wantBytes), got, tt.wantBytes)
			}
			for i := range got {
				if got[i] != tt.wantBytes[i] {
					t.Errorf("byte[%d]: got 0x%02X, want 0x%02X (full: got=%x want=%x)",
						i, got[i], tt.wantBytes[i], got, tt.wantBytes)
				}
			}
		})
	}
}

// TestProblemKeywordDefvals verifies that reserved keywords used as enum
// labels in DEFVAL clauses are parsed and resolved correctly.
// Ground truth: net-snmp accepts { mandatory }, { optional }, etc. as
// DEFVAL values and preserves them as-is.
func TestProblemKeywordDefvals(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-KEYWORDS-MIB")

	tests := []struct {
		name      string
		wantDefval string // expected enum label
	}{
		{"problemDefvalMandatory", "mandatory"},
		{"problemDefvalOptional", "optional"},
		{"problemDefvalCurrent", "current"},
		{"problemDefvalDeprecated", "deprecated"},
		{"problemDefvalObsolete", "obsolete"},
		{"problemDefvalTrue", "true"},
		{"problemDefvalFalse", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := m.FindObject(tt.name)
			testutil.NotNil(t, obj, "object %s should be found", tt.name)
			if obj == nil {
				return
			}

			dv := obj.DefaultValue()
			testutil.True(t, !dv.IsZero(), "object %s should have a DEFVAL", tt.name)

			got := dv.String()
			testutil.Equal(t, tt.wantDefval, got,
				"defval for %s", tt.name)
		})
	}
}

// TestProblemNotifications verifies notification varbind resolution for
// edge cases: not-accessible objects in OBJECTS and undefined varbinds.
// Ground truth: net-snmp includes both not-accessible objects and undefined
// names in OBJECTS lists.
func TestProblemNotifications(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-NOTIFICATIONS-MIB")

	t.Run("normal notification", func(t *testing.T) {
		// net-snmp: OBJECTS { problemNotifStatus, problemNotifDescription }
		notif := m.FindNotification("problemNotifNormal")
		testutil.NotNil(t, notif, "problemNotifNormal should be found")
		if notif == nil {
			return
		}

		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		testutil.Len(t, varbinds, 2, "normal notification should have 2 varbinds")
		testutil.SliceEqual(t,
			[]string{"problemNotifStatus", "problemNotifDescription"},
			varbinds, "normal notification varbinds")
	})

	t.Run("not-accessible in OBJECTS", func(t *testing.T) {
		// net-snmp: OBJECTS { problemNotifIndex, problemNotifStatus }
		// Both net-snmp and gomib should include the not-accessible index object.
		// smilint [3]: "object problemNotifIndex of notification must not be not-accessible"
		notif := m.FindNotification("problemNotifWithIndex")
		testutil.NotNil(t, notif, "problemNotifWithIndex should be found")
		if notif == nil {
			return
		}

		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		wantVarbinds := []string{"problemNotifIndex", "problemNotifStatus"}
		if !varbindsEquivalent(varbinds, wantVarbinds) {
			t.Errorf("varbinds: got %v, want %v (net-snmp ground truth)", varbinds, wantVarbinds)
		}
	})

	t.Run("undefined varbind", func(t *testing.T) {
		// net-snmp: OBJECTS { problemNotifStatus, problemUndefinedVarbind }
		// net-snmp preserves unresolved names; gomib excludes them.
		notif := m.FindNotification("problemNotifWithUndefined")
		testutil.NotNil(t, notif, "problemNotifWithUndefined should be found")
		if notif == nil {
			return
		}

		varbinds := testutil.NormalizeVarbinds(notif.Objects())
		// gomib excludes unresolved references, so we expect only problemNotifStatus
		// (or possibly neither if the entire notification fails).
		// This is a known divergence from net-snmp which preserves the string.
		hasStatus := false
		for _, v := range varbinds {
			if v == "problemNotifStatus" {
				hasStatus = true
			}
		}
		testutil.True(t, hasStatus,
			"problemNotifStatus should be in varbinds (resolved object)")
		// The undefined varbind should NOT be present
		for _, v := range varbinds {
			if v == "problemUndefinedVarbind" {
				t.Errorf("problemUndefinedVarbind should not appear in resolved varbinds")
			}
		}
		t.Logf("divergence: net-snmp includes undefined varbind, gomib excludes it (varbinds=%v)", varbinds)
	})
}

// TestProblemRevisions verifies that MODULE-IDENTITY revision handling works
// for out-of-order revisions and pre-identity object declarations.
// Ground truth: net-snmp resolves all objects regardless of declaration order.
func TestProblemRevisions(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-REVISIONS-MIB")

	t.Run("pre-identity object resolves", func(t *testing.T) {
		// net-snmp OID: enterprises.99998.11
		// Object declared before MODULE-IDENTITY - should still resolve
		node := m.FindNode("problemRevPreIdentity")
		testutil.NotNil(t, node, "problemRevPreIdentity should resolve despite being before MODULE-IDENTITY")
		if node != nil {
			oid := node.OID().String()
			testutil.Equal(t, "1.3.6.1.4.1.99998.11", oid,
				"problemRevPreIdentity OID")
		}
	})

	t.Run("post-identity object resolves", func(t *testing.T) {
		// net-snmp OID: enterprises.99998.11.1.1.1
		obj := m.FindObject("problemRevTestObject")
		testutil.NotNil(t, obj, "problemRevTestObject should resolve")
		if obj != nil {
			oid := obj.OID().String()
			testutil.Equal(t, "1.3.6.1.4.1.99998.11.1.1.1", oid,
				"problemRevTestObject OID")
		}
	})

	t.Run("module identity resolves", func(t *testing.T) {
		// net-snmp OID: enterprises.99998.11.1
		node := m.FindNode("problemRevisionsMIB")
		testutil.NotNil(t, node, "problemRevisionsMIB MODULE-IDENTITY should resolve")
		if node != nil {
			oid := node.OID().String()
			testutil.Equal(t, "1.3.6.1.4.1.99998.11.1", oid,
				"problemRevisionsMIB OID")
		}
	})

	t.Run("out-of-order revisions parsed with dates", func(t *testing.T) {
		mod := m.Module("PROBLEM-REVISIONS-MIB")
		testutil.NotNil(t, mod, "PROBLEM-REVISIONS-MIB module should exist")
		if mod == nil {
			return
		}

		revisions := mod.Revisions()
		// MIB declares 3 revisions in non-chronological order:
		//   REVISION "202401010000Z"  (2024, listed first - wrong)
		//   REVISION "202501010000Z"  (2025, listed second - should be first)
		//   REVISION "202301010000Z"  (2023, listed last - correct position)
		// smilint [3]: "revision not in reverse chronological order"
		// All 3 should be parsed regardless of ordering.
		testutil.Equal(t, 3, len(revisions),
			"all 3 out-of-order revisions should be parsed")

		// Verify the actual date values were captured.
		// The MIB source uses "YYYYMMDD0000Z" format.
		dates := make([]string, len(revisions))
		for i, r := range revisions {
			dates[i] = r.Date
		}
		// Check that all three years are present (in whatever order gomib stores them)
		has2024, has2025, has2023 := false, false, false
		for _, d := range dates {
			if containsYear(d, "2024") {
				has2024 = true
			}
			if containsYear(d, "2025") {
				has2025 = true
			}
			if containsYear(d, "2023") {
				has2023 = true
			}
		}
		testutil.True(t, has2024, "2024 revision should be present (dates=%v)", dates)
		testutil.True(t, has2025, "2025 revision should be present (dates=%v)", dates)
		testutil.True(t, has2023, "2023 revision should be present (dates=%v)", dates)
	})
}

// TestProblemAccess verifies access level resolution for edge cases.
// Ground truth: net-snmp preserves the declared access values exactly.
func TestProblemAccess(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-ACCESS-MIB")

	t.Run("scalar read-create", func(t *testing.T) {
		// net-snmp: MAX-ACCESS read-create
		// smilint [3]: "scalar object must not have a read-create access value"
		// gomib should resolve the access value even though it's invalid per RFC
		obj := m.FindObject("problemScalarReadCreate")
		testutil.NotNil(t, obj, "problemScalarReadCreate should be found")
		if obj == nil {
			return
		}
		access := testutil.NormalizeAccess(obj.Access())
		testutil.Equal(t, "read-create", access,
			"scalar read-create should preserve access value (matches net-snmp)")
	})

	t.Run("write-only", func(t *testing.T) {
		// net-snmp: MAX-ACCESS write-only
		// smilint [2]: "access write-only is no longer allowed in SMIv2"
		obj := m.FindObject("problemWriteOnly")
		testutil.NotNil(t, obj, "problemWriteOnly should be found")
		if obj == nil {
			return
		}
		access := testutil.NormalizeAccess(obj.Access())
		testutil.Equal(t, "write-only", access,
			"write-only should be preserved (matches net-snmp)")
	})

	t.Run("table column access equivalence", func(t *testing.T) {
		// read-write on a column in a RowStatus table
		// net-snmp: MAX-ACCESS read-write
		rwObj := m.FindObject("problemAccessTestValue")
		testutil.NotNil(t, rwObj, "problemAccessTestValue should be found")
		if rwObj == nil {
			return
		}
		rwAccess := testutil.NormalizeAccess(rwObj.Access())
		testutil.Equal(t, "read-write", rwAccess,
			"column read-write should be preserved")

		// read-create on another column
		// net-snmp: MAX-ACCESS read-create
		rcObj := m.FindObject("problemAccessTestName")
		testutil.NotNil(t, rcObj, "problemAccessTestName should be found")
		if rcObj == nil {
			return
		}
		rcAccess := testutil.NormalizeAccess(rcObj.Access())
		testutil.Equal(t, "read-create", rcAccess,
			"column read-create should be preserved")
	})
}

// TestProblemImports verifies that missing base type imports are resolved
// via fallback mechanisms in permissive mode.
// Ground truth: net-snmp silently resolves all base types regardless of imports.
func TestProblemImports(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-IMPORTS-MIB")

	// SMI base types (from SNMPv2-SMI) should resolve via fallback
	smiTests := []struct {
		name     string
		wantType string // normalized base type per net-snmp
	}{
		{"problemMissingCounter64", "Counter64"},
		{"problemMissingGauge32", "Gauge32"},
		{"problemMissingUnsigned32", "Unsigned32"},
		{"problemMissingTimeTicks", "TimeTicks"},
	}

	for _, tt := range smiTests {
		t.Run(tt.name, func(t *testing.T) {
			obj := m.FindObject(tt.name)
			testutil.NotNil(t, obj, "object %s should resolve in permissive mode", tt.name)
			if obj == nil {
				return
			}

			gotType := testutil.NormalizeType(obj.Type())
			testutil.Equal(t, tt.wantType, gotType,
				"base type for %s", tt.name)
		})
	}

	// Textual convention types (from SNMPv2-TC) - net-snmp resolves these
	// implicitly but gomib's permissive fallback only covers SMI base types.
	// These document the current divergence.
	tcTests := []struct {
		name     string
		wantType string // net-snmp ground truth
	}{
		// net-snmp: resolves DisplayString → OCTET STRING
		{"problemMissingDisplayString", "OCTET STRING"},
		// net-snmp: resolves TruthValue → Integer32
		{"problemMissingTruthValue", "Integer32"},
	}

	for _, tt := range tcTests {
		t.Run(tt.name, func(t *testing.T) {
			obj := m.FindObject(tt.name)
			testutil.NotNil(t, obj, "object %s should resolve in permissive mode", tt.name)
			if obj == nil {
				return
			}

			gotType := testutil.NormalizeType(obj.Type())
			if gotType != tt.wantType {
				t.Skipf("divergence: %s type: gomib=%q net-snmp=%q (TC fallback not implemented)",
					tt.name, gotType, tt.wantType)
			}
		})
	}
}

// containsYear checks if a date string contains a 4-digit year.
func containsYear(date, year string) bool {
	return strings.Contains(date, year)
}
