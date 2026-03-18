package gomib

// resolve_problems_test.go tests gomib's handling of real-world MIB edge cases
// using the synthetic PROBLEM-*.mib corpus. Expected values are grounded against
// net-snmp (snmptranslate -Td) and libsmi (smilint) output. Covers hex/binary
// strings, keyword DEFVALs, notifications, revisions, access levels, imports,
// SMIv1/v2 mixing, index edge cases, DEFVAL variants, module aliases, and naming.

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// loadProblemMIB loads a problem MIB by name using both the primary corpus
// (for dependencies like SNMPv2-SMI, SNMPv2-TC) and the problems directory.
func loadProblemMIB(t testing.TB, name string) *mib.Mib {
	t.Helper()
	corpus := mustDir(t, testutil.PrimaryCorpusDir(t))
	problems := mustDir(t, testutil.ProblemsCorpusDir(t))
	diag := mib.DiagnosticConfig{
		Reporting: mib.ReportingVerbose,
		FailAt:    mib.SeverityFatal,
		Ignore:    []string{"identifier-underscore", "identifier-length-32", "bad-identifier-case"},
	}
	m, err := Load(context.Background(),
		WithSource(corpus, problems),
		WithModules(name),
		WithResolverStrictness(mib.ResolverPermissive),
		WithDiagnosticConfig(diag),
	)
	if err != nil {
		t.Fatalf("Load(%s) failed: %v", name, err)
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
		name       string
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
			obj := requireObject(t, m, tt.name)

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
			obj := requireObject(t, m, tt.name)

			dv := obj.DefaultValue()
			if dv.IsZero() {
				t.Fatalf("object %s has no DEFVAL", tt.name)
			}

			if dv.Kind() != mib.DefValKindBytes {
				t.Fatalf("object %s DEFVAL kind=%v, expected Bytes", tt.name, dv.Kind())
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
		name       string
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
			obj := requireObject(t, m, tt.name)

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
		notif := requireNotification(t, m, "problemNotifNormal")

		varbinds := normalizeVarbinds(notif.Objects())
		testutil.Len(t, varbinds, 2, "normal notification should have 2 varbinds")
		testutil.SliceEqual(t,
			[]string{"problemNotifStatus", "problemNotifDescription"},
			varbinds, "normal notification varbinds")
	})

	t.Run("not-accessible in OBJECTS", func(t *testing.T) {
		// net-snmp: OBJECTS { problemNotifIndex, problemNotifStatus }
		// Both net-snmp and gomib should include the not-accessible index object.
		// smilint [3]: "object problemNotifIndex of notification must not be not-accessible"
		notif := requireNotification(t, m, "problemNotifWithIndex")

		varbinds := normalizeVarbinds(notif.Objects())
		wantVarbinds := []string{"problemNotifIndex", "problemNotifStatus"}
		if !varbindsEquivalent(varbinds, wantVarbinds) {
			t.Errorf("varbinds: got %v, want %v (net-snmp ground truth)", varbinds, wantVarbinds)
		}
	})

	t.Run("undefined varbind", func(t *testing.T) {
		// net-snmp: OBJECTS { problemNotifStatus, problemUndefinedVarbind }
		// net-snmp preserves unresolved names; gomib excludes them.
		notif := requireNotification(t, m, "problemNotifWithUndefined")

		varbinds := normalizeVarbinds(notif.Objects())
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
		for _, v := range varbinds {
			if v == "problemUndefinedVarbind" {
				t.Errorf("problemUndefinedVarbind should not appear in resolved varbinds")
			}
		}
		t.Logf("divergence: net-snmp includes undefined varbind, gomib excludes it (varbinds=%v)", varbinds)
	})
}

// TestProblemNotifBadGroup verifies that a diagnostic is emitted when an
// OBJECT-GROUP includes a not-accessible object.
// Ground truth: smilint [3] "node is an invalid member of object group"
func TestProblemNotifBadGroup(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-NOTIFICATIONS-MIB")
	requireModuleDiagnostic(t, m, "PROBLEM-NOTIFICATIONS-MIB",
		"group-not-accessible", 1, "problemNotifIndex", "problemNotifBadGroup")
}

// TestProblemNotifObjectAccess verifies that a diagnostic is emitted when a
// NOTIFICATION-TYPE OBJECTS clause references a not-accessible object.
// Ground truth: smilint [3] "object must not be not-accessible"
func TestProblemNotifObjectAccess(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-NOTIFICATIONS-MIB")
	requireModuleDiagnostic(t, m, "PROBLEM-NOTIFICATIONS-MIB",
		"notification-object-access", 1, "problemNotifIndex", "problemNotifWithIndex")
}

// TestProblemNotifNotReversible verifies that a diagnostic is emitted when a
// SMIv2 notification's parent node has a non-zero sub-identifier.
// Ground truth: smilint [5] "notification is not reverse mappable"
func TestProblemNotifNotReversible(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-NOTIFICATIONS-MIB")
	requireModuleDiagnostic(t, m, "PROBLEM-NOTIFICATIONS-MIB",
		"notification-not-reversible", 1, "problemNotifNotReversible")

	// Notifications under .0. prefix should not trigger the diagnostic.
	for _, d := range m.Diagnostics() {
		if d.Code == "notification-not-reversible" && strings.Contains(d.Message, "problemNotifNormal") {
			t.Errorf("problemNotifNormal is under .0. prefix and should not trigger not-reversible")
		}
	}
}

// TestProblemNotifIdTooLarge verifies that a diagnostic is emitted when a
// notification's last sub-identifier exceeds INT32_MAX.
// Ground truth: smilint [5] "last sub-identifier of notification too large"
func TestProblemNotifIdTooLarge(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-NOTIFICATIONS-MIB")
	requireModuleDiagnostic(t, m, "PROBLEM-NOTIFICATIONS-MIB",
		"notification-id-too-large", 1, "problemNotifIdTooLarge")
}

// TestProblemRevisions verifies that MODULE-IDENTITY revision handling works
// for out-of-order revisions and pre-identity object declarations.
// Ground truth: net-snmp resolves all objects regardless of declaration order.
func TestProblemRevisions(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-REVISIONS-MIB")

	t.Run("pre-identity object resolves", func(t *testing.T) {
		// net-snmp OID: enterprises.99998.11
		// Object declared before MODULE-IDENTITY - should still resolve
		node := m.Node("problemRevPreIdentity")
		testutil.NotNil(t, node, "problemRevPreIdentity should resolve despite being before MODULE-IDENTITY")
		testutil.Equal(t, "1.3.6.1.4.1.99998.11", node.OID().String(),
			"problemRevPreIdentity OID")
	})

	t.Run("post-identity object resolves", func(t *testing.T) {
		// net-snmp OID: enterprises.99998.11.1.1.1
		obj := m.Object("problemRevTestObject")
		testutil.NotNil(t, obj, "problemRevTestObject should resolve")
		testutil.Equal(t, "1.3.6.1.4.1.99998.11.1.1.1", obj.OID().String(),
			"problemRevTestObject OID")
	})

	t.Run("module identity resolves", func(t *testing.T) {
		// net-snmp OID: enterprises.99998.11.1
		node := m.Node("problemRevisionsMIB")
		testutil.NotNil(t, node, "problemRevisionsMIB MODULE-IDENTITY should resolve")
		testutil.Equal(t, "1.3.6.1.4.1.99998.11.1", node.OID().String(),
			"problemRevisionsMIB OID")
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

	t.Run("missing revision for LAST-UPDATED emits diagnostic", func(t *testing.T) {
		// LAST-UPDATED "202601280000Z" but no REVISION with that date.
		// smilint [3]: "revision for last update is missing"
		found := false
		for _, d := range m.Diagnostics() {
			if d.Code == "revision-last-updated" && d.Module == "PROBLEM-REVISIONS-MIB" {
				found = true
				if !strings.Contains(d.Message, "202601280000Z") {
					t.Errorf("diagnostic message should mention the date, got: %s", d.Message)
				}
			}
		}
		testutil.True(t, found,
			"should emit revision-last-updated diagnostic for missing revision")
	})

	t.Run("out-of-order revisions emit not-descending diagnostic", func(t *testing.T) {
		// Revisions are 2024, 2025, 2023 - 2025 after 2024 violates descending order.
		// smilint [3]: "revision not in reverse chronological order"
		found := false
		for _, d := range m.Diagnostics() {
			if d.Code == "revision-not-descending" && d.Module == "PROBLEM-REVISIONS-MIB" {
				found = true
			}
		}
		testutil.True(t, found,
			"should emit revision-not-descending diagnostic for out-of-order revisions")
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
		obj := m.Object("problemScalarReadCreate")
		testutil.NotNil(t, obj, "problemScalarReadCreate should be found")
		if obj == nil {
			return
		}
		access := normalizeAccess(obj.Access())
		testutil.Equal(t, "read-create", access,
			"scalar read-create should preserve access value (matches net-snmp)")
	})

	t.Run("write-only", func(t *testing.T) {
		// net-snmp: MAX-ACCESS write-only
		// smilint [2]: "access write-only is no longer allowed in SMIv2"
		obj := m.Object("problemWriteOnly")
		testutil.NotNil(t, obj, "problemWriteOnly should be found")
		if obj == nil {
			return
		}
		access := normalizeAccess(obj.Access())
		testutil.Equal(t, "write-only", access,
			"write-only should be preserved (matches net-snmp)")
	})

	t.Run("table column access equivalence", func(t *testing.T) {
		// read-write on a column in a RowStatus table
		// net-snmp: MAX-ACCESS read-write
		rwObj := m.Object("problemAccessTestValue")
		testutil.NotNil(t, rwObj, "problemAccessTestValue should be found")
		if rwObj == nil {
			return
		}
		rwAccess := normalizeAccess(rwObj.Access())
		testutil.Equal(t, "read-write", rwAccess,
			"column read-write should be preserved")

		// read-create on another column
		// net-snmp: MAX-ACCESS read-create
		rcObj := m.Object("problemAccessTestName")
		testutil.NotNil(t, rcObj, "problemAccessTestName should be found")
		if rcObj == nil {
			return
		}
		rcAccess := normalizeAccess(rcObj.Access())
		testutil.Equal(t, "read-create", rcAccess,
			"column read-create should be preserved")
	})
}

// TestProblemImports verifies that missing base type imports are resolved
// via fallback mechanisms in permissive mode.
// Ground truth: net-snmp silently resolves all base types regardless of imports.
func TestProblemImports(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-IMPORTS-MIB")

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
			obj := m.Object(tt.name)
			testutil.NotNil(t, obj, "object %s should resolve in permissive mode", tt.name)
			if obj == nil {
				return
			}

			gotType := normalizeType(obj.Type())
			testutil.Equal(t, tt.wantType, gotType,
				"base type for %s", tt.name)
		})
	}

	// Textual convention types (from SNMPv2-TC) - net-snmp resolves these
	// implicitly. gomib resolves them at permissive level via TC fallback.
	tcTests := []struct {
		name     string
		wantType string // net-snmp ground truth
	}{
		{"problemMissingDisplayString", "OCTET STRING"},
		{"problemMissingTruthValue", "Integer32"},
	}

	for _, tt := range tcTests {
		t.Run(tt.name, func(t *testing.T) {
			obj := m.Object(tt.name)
			testutil.NotNil(t, obj, "object %s should resolve in permissive mode", tt.name)
			if obj == nil {
				return
			}

			gotType := normalizeType(obj.Type())
			testutil.Equal(t, tt.wantType, gotType,
				"TC base type for %s (matches net-snmp)", tt.name)
		})
	}
}

// containsYear checks if a date string contains a 4-digit year.
func containsYear(date, year string) bool {
	return strings.Contains(date, year)
}

// TestProblemSMIv1v2AccessKeyword verifies that the parser accepts the SMIv1
// ACCESS keyword in an SMIv2 module and resolves the access value correctly.
// Ground truth: net-snmp silently accepts; smilint flags as SMIv1 style.
// PROBLEMS.md: PROBLEM-SMIv1v2-MIX-MIB / ACCESS keyword instead of MAX-ACCESS
func TestProblemSMIv1v2AccessKeyword(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-SMIv1v2-MIX-MIB")

	obj := m.Object("problemV1AccessObject")
	testutil.NotNil(t, obj, "Object(problemV1AccessObject)")
	access := normalizeAccess(obj.Access())
	testutil.Equal(t, "read-only", access,
		"ACCESS read-only in SMIv2 module should resolve (matches net-snmp)")
}

// TestProblemSMIv1v2MandatoryStatus verifies that mandatory and optional status
// values from SMIv1 are accepted in SMIv2 modules.
// Ground truth: net-snmp silently accepts; smilint flags as invalid status.
// PROBLEMS.md: PROBLEM-SMIv1v2-MIX-MIB / mandatory/optional status
func TestProblemSMIv1v2MandatoryStatus(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-SMIv1v2-MIX-MIB")

	t.Run("mandatory", func(t *testing.T) {
		obj := m.Object("problemMandatoryStatus")
		testutil.NotNil(t, obj, "Object(problemMandatoryStatus)")
		status := normalizeStatus(obj.Status())
		// gomib preserves mandatory as StatusMandatory; net-snmp maps to current
		if status != "mandatory" && status != "current" {
			t.Errorf("status: got %q, want mandatory or current", status)
		}
	})

	t.Run("optional", func(t *testing.T) {
		obj := m.Object("problemOptionalStatus")
		testutil.NotNil(t, obj, "Object(problemOptionalStatus)")
		status := normalizeStatus(obj.Status())
		if status != "optional" && status != "obsolete" {
			t.Errorf("status: got %q, want optional or obsolete", status)
		}
	})
}

// TestProblemSMIv1v2TrapType verifies that TRAP-TYPE macros in SMIv2 modules
// are at least parsed without crashing. TRAP-TYPE is an SMIv1 construct that
// appears in many vendor MIBs (e.g., RADLAN-MIB with 931 occurrences).
// Ground truth: net-snmp silently accepts; smilint rejects.
// PROBLEMS.md: PROBLEM-SMIv1v2-MIX-MIB / TRAP-TYPE macro
func TestProblemSMIv1v2TrapType(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-SMIv1v2-MIX-MIB")

	// Verify surrounding SMIv2 objects are not affected by TRAP-TYPE.
	normalObj := m.Object("problemV2NormalObject")
	testutil.NotNil(t, normalObj, "Object(problemV2NormalObject)")
	testutil.Equal(t, "1.3.6.1.4.1.99998.1.1.4", normalObj.OID().String(),
		"normal SMIv2 object OID")

	notif := m.Notification("problemV2Notification")
	testutil.NotNil(t, notif, "Notification(problemV2Notification)")
	testutil.Equal(t, "1.3.6.1.4.1.99998.1.2.1", notif.OID().String(),
		"SMIv2 notification OID")

	// Verify the TRAP-TYPE itself resolves.
	// OID = enterprise.0.trapNumber = 1.3.6.1.4.1.99998.1.0.1
	trap := m.Notification("problemV1Trap")
	testutil.NotNil(t, trap, "Notification(problemV1Trap)")
	testutil.Equal(t, "1.3.6.1.4.1.99998.1.0.1", trap.OID().String(),
		"TRAP-TYPE OID")

	ti := trap.TrapInfo()
	testutil.NotNil(t, ti, "TrapInfo for problemV1Trap")
	testutil.Equal(t, "problemSmiv1v2MixMIB", ti.Enterprise, "trap enterprise")
	testutil.Equal(t, uint32(1), ti.TrapNumber, "trap number")

	objs := trap.Objects()
	testutil.Len(t, objs, 1, "TRAP-TYPE VARIABLES count")
	testutil.Equal(t, "problemV1AccessObject", objs[0].Name(), "TRAP-TYPE VARIABLES[0]")
}

// TestProblemSMIv1v2CounterGauge verifies resolution of SMIv1 Counter and
// Gauge types (without the "32" suffix) in an SMIv2 module.
// Ground truth: net-snmp resolves silently; smilint rejects.
// PROBLEMS.md: PROBLEM-SMIv1v2-MIX-MIB / SMIv1 types Counter/Gauge
func TestProblemSMIv1v2CounterGauge(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-SMIv1v2-MIX-MIB")

	t.Run("Counter", func(t *testing.T) {
		obj := m.Object("problemCounterObject")
		testutil.NotNil(t, obj, "problemCounterObject should resolve via SMIv1 type fallback")
		if obj == nil {
			return
		}
		testutil.NotNil(t, obj.Type(), "Counter type should resolve via RFC1155-SMI fallback")
		if obj.Type() == nil {
			return
		}
		testutil.Equal(t, mib.BaseCounter32, obj.Type().EffectiveBase(), "Counter base type")
	})

	t.Run("Gauge", func(t *testing.T) {
		obj := m.Object("problemGaugeObject")
		testutil.NotNil(t, obj, "problemGaugeObject should resolve via SMIv1 type fallback")
		if obj == nil {
			return
		}
		testutil.NotNil(t, obj.Type(), "Gauge type should resolve via RFC1155-SMI fallback")
		if obj.Type() == nil {
			return
		}
		testutil.Equal(t, mib.BaseGauge32, obj.Type().EffectiveBase(), "Gauge base type")
	})
}

// TestProblemIndexBareType verifies that tables with bare type names in INDEX
// clauses (e.g., INDEX { INTEGER }) load without crashing.
// Ground truth: net-snmp silently accepts; smilint rejects with syntax error.
// PROBLEMS.md: PROBLEM-INDEX-MIB / Bare INTEGER in INDEX
func TestProblemIndexBareType(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-INDEX-MIB")

	entry := m.Object("problemBareTypeEntry")
	testutil.NotNil(t, entry, "Object(problemBareTypeEntry)")

	kind := normalizeKind(entry.Kind())
	testutil.Equal(t, "row", kind, "entry should be a row")

	idx := entry.Index()
	testutil.Len(t, idx, 1, "bare type entry should have one index")
	testutil.Nil(t, idx[0].Object, "bare type index Object should be nil")
	testutil.Equal(t, "INTEGER", idx[0].TypeName, "bare type index TypeName")

	val := m.Object("problemBareTypeValue")
	testutil.NotNil(t, val, "problemBareTypeValue should resolve even with bare type index")
}

// TestProblemIndexMacAddress verifies that tables using MacAddress as an
// index element resolve correctly.
// Ground truth: net-snmp silently accepts; smilint flags as illegal base type.
// PROBLEMS.md: PROBLEM-INDEX-MIB / MacAddress as index type
func TestProblemIndexMacAddress(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-INDEX-MIB")

	entry := m.Object("problemMacIndexEntry")
	testutil.NotNil(t, entry, "Object(problemMacIndexEntry)")

	kind := normalizeKind(entry.Kind())
	testutil.Equal(t, "row", kind, "entry should be a row")

	indexes := normalizeIndexes(entry.Index())
	testutil.NotEmpty(t, indexes, "indexes for MacAddress index table")
	testutil.Equal(t, "problemMacIndexAddress", indexes[0].Name,
		"MacAddress index should resolve")

	port := m.Object("problemMacIndexPort")
	testutil.NotNil(t, port, "problemMacIndexPort column should resolve")
}

// TestProblemIndexNoRange verifies that tables with index elements missing
// range restrictions still resolve correctly.
// Ground truth: net-snmp silently accepts; smilint warns about missing range.
// PROBLEMS.md: PROBLEM-INDEX-MIB / Index element missing range restriction
func TestProblemIndexNoRange(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-INDEX-MIB")

	entry := m.Object("problemNoRangeEntry")
	testutil.NotNil(t, entry, "Object(problemNoRangeEntry)")

	kind := normalizeKind(entry.Kind())
	testutil.Equal(t, "row", kind, "entry should be a row")

	indexes := normalizeIndexes(entry.Index())
	testutil.Len(t, indexes, 2, "indexes count")
	testutil.Equal(t, "problemNoRangeIndex1", indexes[0].Name, "first index")
	testutil.Equal(t, "problemNoRangeIndex2", indexes[1].Name, "second index")
}

// TestProblemIndexDisplayString verifies that tables using DisplayString
// as an index element resolve correctly.
// Ground truth: net-snmp silently accepts.
// PROBLEMS.md: PROBLEM-INDEX-MIB / DisplayString as index
func TestProblemIndexDisplayString(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-INDEX-MIB")

	entry := m.Object("problemStringIndexEntry")
	testutil.NotNil(t, entry, "Object(problemStringIndexEntry)")

	kind := normalizeKind(entry.Kind())
	testutil.Equal(t, "row", kind, "entry should be a row")

	indexes := normalizeIndexes(entry.Index())
	testutil.NotEmpty(t, indexes, "indexes for DisplayString index table")
	testutil.Equal(t, "problemStringIndexName", indexes[0].Name,
		"DisplayString index should resolve")
}

// TestProblemIndexTableKinds verifies that all tables in PROBLEM-INDEX-MIB
// get the correct kind inference (table/row/column).
// This is a structural test that verifies the semantics phase handles
// various index patterns without misclassifying nodes.
func TestProblemIndexTableKinds(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-INDEX-MIB")

	tables := []struct {
		table      string
		entry      string
		column     string
		indexNames []string
	}{
		{
			"problemNetAddrTable", "problemNetAddrEntry", "problemNetAddrValue",
			[]string{"problemNetAddrIfIndex", "problemNetAddrAddress"},
		},
		{
			"problemMacIndexTable", "problemMacIndexEntry", "problemMacIndexPort",
			[]string{"problemMacIndexAddress"},
		},
		{
			"problemNoRangeTable", "problemNoRangeEntry", "problemNoRangeValue",
			[]string{"problemNoRangeIndex1", "problemNoRangeIndex2"},
		},
		{
			"problemStringIndexTable", "problemStringIndexEntry", "problemStringIndexValue",
			[]string{"problemStringIndexName"},
		},
	}

	for _, tt := range tables {
		t.Run(tt.table, func(t *testing.T) {
			tbl := m.Object(tt.table)
			testutil.NotNil(t, tbl, "Object(%s)", tt.table)
			testutil.Equal(t, "table", normalizeKind(tbl.Kind()), "table kind")

			ent := m.Object(tt.entry)
			testutil.NotNil(t, ent, "Object(%s)", tt.entry)
			testutil.Equal(t, "row", normalizeKind(ent.Kind()), "entry kind")

			indexes := normalizeIndexes(ent.Index())
			testutil.Len(t, indexes, len(tt.indexNames), "index count")
			for i, name := range tt.indexNames {
				testutil.Equal(t, name, indexes[i].Name, "index[%d]", i)
			}

			col := m.Object(tt.column)
			testutil.NotNil(t, col, "Object(%s)", tt.column)
			testutil.Equal(t, "column", normalizeKind(col.Kind()), "column kind")
		})
	}
}

// TestProblemDefvalOidRef verifies that OID-valued DEFVALs (e.g., zeroDotZero)
// are parsed and stored with the symbolic name from MIB source.
// Ground truth: net-snmp outputs "zeroDotZero", gomib also outputs "zeroDotZero".
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / DEFVAL with OID reference
func TestProblemDefvalOidRef(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalOidRef")
	testutil.NotNil(t, obj, "Object(problemDefvalOidRef)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DefaultValue() for %s", "problemDefvalOidRef")

	got := dv.String()
	testutil.Equal(t, "zeroDotZero", got, "OID defval should return symbolic name")
}

// TestProblemDefvalTypeMismatch verifies that a raw integer DEFVAL for an
// enum-typed object is parsed without error.
// Ground truth: net-snmp silently accepts integer 5 for ProblemSeverity.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / DEFVAL with raw integer for enum type
func TestProblemDefvalTypeMismatch(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalTypeMismatch")
	testutil.NotNil(t, obj, "Object(problemDefvalTypeMismatch)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DefaultValue() for %s", "problemDefvalTypeMismatch")

	got := dv.String()
	// The DEFVAL { 5 } could be stored as integer "5" or as enum label "warning"
	// (value 5 in ProblemSeverity). Either is acceptable.
	if got != "5" && got != "warning" {
		t.Errorf("defval: got %q, want 5 or warning", got)
	}
}

// TestProblemDefvalBadEnum verifies that a DEFVAL with an undefined enum label
// is handled without crashing and emits a defval-enum diagnostic.
// Ground truth: smilint flags mismatch; net-snmp silently accepts.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / DEFVAL with undefined enum label
func TestProblemDefvalBadEnum(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalBadEnum")
	testutil.NotNil(t, obj, "Object(problemDefvalBadEnum)")

	dv := obj.DefaultValue()
	// "unknown" is not in the enum {up(1), down(2), testing(3)}.
	// Parser stores the raw label as DefValKindEnum.
	testutil.True(t, !dv.IsZero(), "DEFVAL should be parsed even with undefined enum label")
	testutil.Equal(t, mib.DefValKindEnum, dv.Kind(), "bad enum DEFVAL kind")
	testutil.Equal(t, "unknown", dv.String(), "bad enum DEFVAL preserved as raw label")

	n := countDiagnostics(m, "defval-enum")
	testutil.True(t, n >= 1, "expected at least 1 defval-enum diagnostic for bad enum label, got %d", n)
}

// TestProblemDefvalLargeHex verifies that a 16-byte hex DEFVAL is parsed.
// Ground truth: net-snmp outputs "0" (truncates zeros). gomib preserves full hex.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / Large hex DEFVAL
func TestProblemDefvalLargeHex(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalLargeHex")
	testutil.NotNil(t, obj, "Object(problemDefvalLargeHex)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DefaultValue() for problemDefvalLargeHex")
	testutil.Equal(t, mib.DefValKindBytes, dv.Kind(), "large hex defval kind")

	got := dv.String()
	// 16 bytes > 8, so DefVal.String() uses hex format instead of decimal
	testutil.Equal(t, "0x00000000000000000000000000000000", got, "large hex defval string")

	// Verify equivalence with net-snmp's representation ("0" for all-zero hex)
	testutil.True(t, defvalEquivalent(got, "0"), "large hex defval should be equivalent to net-snmp's \"0\"")
}

// TestProblemDefvalBinary verifies binary string DEFVAL parsing.
// Ground truth: net-snmp parses '10101010'B as 170 (0xAA).
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / Binary string DEFVAL
func TestProblemDefvalBinary(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalBinary")
	testutil.NotNil(t, obj, "Object(problemDefvalBinary)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DefaultValue() for %s", "problemDefvalBinary")

	got := dv.String()
	// '10101010'B = 0xAA = 170
	if got != "170" {
		t.Errorf("binary defval: got %q, want 170", got)
	}
}

// TestProblemDefvalBinaryOdd verifies binary string DEFVAL with non-multiple-of-8 bits.
// Ground truth: net-snmp parses '101'B as 5 (pad to 00000101).
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / Binary string not multiple of 8 bits
func TestProblemDefvalBinaryOdd(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalBinaryOdd")
	testutil.NotNil(t, obj, "Object(problemDefvalBinaryOdd)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DefaultValue() for %s", "problemDefvalBinaryOdd")

	got := dv.String()
	// '101'B padded to '00000101'B = 0x05 = 5
	if got != "5" {
		t.Errorf("odd binary defval: got %q, want 5", got)
	}
}

// TestProblemDefvalEmptyBits verifies that DEFVAL { { } } (empty BITS) is parsed.
// Ground truth: net-snmp accepts silently.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / Empty BITS DEFVAL
func TestProblemDefvalEmptyBits(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalEmptyBits")
	testutil.NotNil(t, obj, "Object(problemDefvalEmptyBits)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DefaultValue() should be set")
	testutil.Equal(t, mib.DefValKindBits, dv.Kind(), "kind")

	labels, ok := mib.DefValAs[[]string](dv)
	testutil.True(t, ok, "DefValAs[[]string]")
	testutil.Equal(t, 0, len(labels), "empty BITS should have 0 labels")
}

// TestProblemDefvalMultiBits verifies that DEFVAL { { read, write } } is parsed.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / (not listed but included in the MIB)
func TestProblemDefvalMultiBits(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalMultiBits")
	testutil.NotNil(t, obj, "Object(problemDefvalMultiBits)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DefaultValue() should be set")
	testutil.Equal(t, mib.DefValKindBits, dv.Kind(), "kind")

	labels, ok := mib.DefValAs[[]string](dv)
	testutil.True(t, ok, "DefValAs[[]string]")
	testutil.Equal(t, 2, len(labels), "multi BITS should have 2 labels")

	hasRead, hasWrite := false, false
	for _, l := range labels {
		if l == "read" {
			hasRead = true
		}
		if l == "write" {
			hasWrite = true
		}
	}
	testutil.True(t, hasRead, "should have 'read' bit")
	testutil.True(t, hasWrite, "should have 'write' bit")
}

// TestProblemDefvalNegative verifies that negative integer DEFVAL is parsed.
// Ground truth: net-snmp accepts silently.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / Negative integer DEFVAL
func TestProblemDefvalNegative(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalNegative")
	testutil.NotNil(t, obj, "Object(problemDefvalNegative)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DefaultValue() for %s", "problemDefvalNegative")

	got := dv.String()
	testutil.Equal(t, "-1", got, "negative DEFVAL should be -1")
}

// TestProblemDefvalSpecialString verifies quoted string DEFVAL parsing.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / (not listed but included)
func TestProblemDefvalSpecialString(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalSpecialString")
	testutil.NotNil(t, obj, "Object(problemDefvalSpecialString)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DefaultValue() for %s", "problemDefvalSpecialString")

	// The raw MIB has DEFVAL { "default-value" }
	got := dv.String()
	if got != `"default-value"` {
		t.Errorf("string defval: got %q, want %q", got, `"default-value"`)
	}
}

// TestProblemDefvalNegativeUnsigned verifies that a negative DEFVAL for an
// unsigned type emits a defval-basetype diagnostic.
func TestProblemDefvalNegativeUnsigned(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalNegativeUnsigned")
	testutil.NotNil(t, obj, "Object(problemDefvalNegativeUnsigned)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DEFVAL should be set")

	n := countDiagnostics(m, "defval-basetype")
	testutil.True(t, n >= 1, "expected at least 1 defval-basetype diagnostic, got %d", n)
}

// TestProblemDefvalOutOfRange verifies that a DEFVAL outside the RANGE
// constraint emits a defval-range diagnostic.
func TestProblemDefvalOutOfRange(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalOutOfRange")
	testutil.NotNil(t, obj, "Object(problemDefvalOutOfRange)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DEFVAL should be set")

	n := countDiagnostics(m, "defval-range")
	testutil.True(t, n >= 1, "expected at least 1 defval-range diagnostic, got %d", n)
}

// TestProblemDefvalBadBitsLabel verifies that a DEFVAL with a BITS label
// not defined in the type emits a defval-enum diagnostic.
func TestProblemDefvalBadBitsLabel(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalBadBitsLabel")
	testutil.NotNil(t, obj, "Object(problemDefvalBadBitsLabel)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DEFVAL should be set")
	testutil.Equal(t, mib.DefValKindBits, dv.Kind(), "kind")

	n := countDiagnostics(m, "defval-enum")
	// At least 1 for the bad label "c" (may have more from other test objects)
	testutil.True(t, n >= 1, "expected at least 1 defval-enum diagnostic for bad BITS label, got %d", n)
}

// TestProblemDefvalIntForEnum verifies that an integer DEFVAL not matching
// any enumeration value emits a defval-enum diagnostic.
func TestProblemDefvalIntForEnum(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.Object("problemDefvalIntForEnum")
	testutil.NotNil(t, obj, "Object(problemDefvalIntForEnum)")

	dv := obj.DefaultValue()
	testutil.True(t, !dv.IsZero(), "DEFVAL should be set")
	testutil.Equal(t, mib.DefValKindInt, dv.Kind(), "kind")

	n := countDiagnostics(m, "defval-enum")
	testutil.True(t, n >= 1, "expected at least 1 defval-enum diagnostic for int not matching enum, got %d", n)
}

// TestProblemRangeExchanged verifies that exchanged range limits (min > max)
// emit a range-exchanged diagnostic.
func TestProblemRangeExchanged(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-RANGES-MIB")

	n := countDiagnostics(m, "range-exchanged")
	testutil.True(t, n >= 1, "expected at least 1 range-exchanged diagnostic, got %d", n)
}

// TestProblemRangeBounds verifies that range limits exceeding basetype bounds
// emit a range-bounds diagnostic.
func TestProblemRangeBounds(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-RANGES-MIB")

	n := countDiagnostics(m, "range-bounds")
	testutil.True(t, n >= 1, "expected at least 1 range-bounds diagnostic, got %d", n)
}

// TestProblemRangeOverlap verifies that overlapping ranges emit a
// range-overlap diagnostic.
func TestProblemRangeOverlap(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-RANGES-MIB")

	n := countDiagnostics(m, "range-overlap")
	testutil.True(t, n >= 1, "expected at least 1 range-overlap diagnostic, got %d", n)
}

// TestProblemRangeAscending verifies that non-ascending ranges emit a
// range-ascending diagnostic.
func TestProblemRangeAscending(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-RANGES-MIB")

	n := countDiagnostics(m, "range-ascending")
	testutil.True(t, n >= 1, "expected at least 1 range-ascending diagnostic, got %d", n)
}

// TestProblemSizeIllegal verifies that SIZE constraints on non-OCTET-STRING
// types emit a size-illegal diagnostic.
func TestProblemSizeIllegal(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-RANGES-MIB")

	n := countDiagnostics(m, "size-illegal")
	testutil.True(t, n >= 2, "expected at least 2 size-illegal diagnostics, got %d", n)
}

// TestProblemRangeIllegal verifies that range constraints on non-numeric
// types emit a range-illegal diagnostic.
func TestProblemRangeIllegal(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-RANGES-MIB")

	n := countDiagnostics(m, "range-illegal")
	testutil.True(t, n >= 2, "expected at least 2 range-illegal diagnostics, got %d", n)
}

// TestProblemCounterRangeIllegal verifies that range constraints on Counter32
// or Counter64 types emit a counter-range-illegal diagnostic.
func TestProblemCounterRangeIllegal(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-RANGES-MIB")

	n := countDiagnostics(m, "counter-range-illegal")
	testutil.True(t, n >= 2, "expected at least 2 counter-range-illegal diagnostics, got %d", n)
}

// TestProblemEnumNameRedefinition verifies that duplicate enum names
// emit an enum-name-redefinition diagnostic.
func TestProblemEnumNameRedefinition(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-ENUM-BITS-DUPES-MIB")

	n := countDiagnostics(m, "enum-name-redefinition")
	testutil.True(t, n >= 2, "expected at least 2 enum-name-redefinition diagnostics, got %d", n)
}

// TestProblemEnumValueRedefinition verifies that duplicate enum values
// emit an enum-value-redefinition diagnostic.
func TestProblemEnumValueRedefinition(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-ENUM-BITS-DUPES-MIB")

	n := countDiagnostics(m, "enum-value-redefinition")
	testutil.True(t, n >= 2, "expected at least 2 enum-value-redefinition diagnostics, got %d", n)
}

// TestProblemBitsNameRedefinition verifies that duplicate BITS names
// emit a bits-name-redefinition diagnostic.
func TestProblemBitsNameRedefinition(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-ENUM-BITS-DUPES-MIB")

	n := countDiagnostics(m, "bits-name-redefinition")
	testutil.True(t, n >= 1, "expected at least 1 bits-name-redefinition diagnostic, got %d", n)
}

// TestProblemBitsValueRedefinition verifies that duplicate BITS positions
// emit a bits-value-redefinition diagnostic.
func TestProblemBitsValueRedefinition(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-ENUM-BITS-DUPES-MIB")

	n := countDiagnostics(m, "bits-value-redefinition")
	testutil.True(t, n >= 1, "expected at least 1 bits-value-redefinition diagnostic, got %d", n)
}

// TestProblemSubtypeEnumIllegal verifies that derived enum types with values
// not present in the parent emit a subtype-enumeration-illegal diagnostic.
func TestProblemSubtypeEnumIllegal(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-ENUM-SUBTYPE-MIB")

	n := countDiagnostics(m, "subtype-enumeration-illegal")
	testutil.True(t, n >= 3, "expected at least 3 subtype-enumeration-illegal diagnostics, got %d", n)
}

// TestProblemImportsAliasNormal verifies that module alias resolution works
// at normal strictness (safe fallback).
// Ground truth: net-snmp fails on SNMPv2-SMI-v1 (unlike gomib which has alias table).
// PROBLEMS.md: PROBLEM-IMPORTS-ALIAS-MIB / SNMPv2-SMI-v1 / SNMPv2-TC-v1
func TestProblemImportsAliasNormal(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-IMPORTS-ALIAS-MIB", mib.ResolverNormal)

	str := m.Object("problemAliasString")
	testutil.NotNil(t, str, "Object(problemAliasString)")

	testutil.NotNil(t, str.Type(), "type for problemAliasString")
	testutil.Equal(t, mib.BaseOctetString, str.Type().EffectiveBase(),
		"DisplayString from SNMPv2-TC-v1 should resolve to OCTET STRING")

	testutil.Equal(t, "1.3.6.1.4.1.99998.3.1.1", str.OID().String(),
		"OID should resolve through aliased module imports")
}

// TestProblemImportsAliasStrict verifies that module aliases are disabled in
// strict mode.
// PROBLEMS.md: PROBLEM-IMPORTS-ALIAS-MIB / SNMPv2-SMI-v1
func TestProblemImportsAliasStrict(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-IMPORTS-ALIAS-MIB", mib.ResolverStrict)

	unresolved := unresolvedSymbols(m, "PROBLEM-IMPORTS-ALIAS-MIB", mib.UnresolvedImport)
	testutil.True(t, len(unresolved) > 0,
		"aliases should not resolve at strict level")

	str := m.Object("problemAliasString")
	testutil.Nil(t, str, "objects should not resolve via aliased imports at strict level")
}

// TestProblemNamingUppercase verifies that uppercase-starting identifiers
// (common in Huawei MIBs) are handled by the parser in permissive mode.
// Ground truth: net-snmp silently accepts; smilint rejects.
// PROBLEMS.md: PROBLEM-NAMING-MIB / Uppercase initial letter
func TestProblemNamingUppercase(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-NAMING-MIB")

	normalObj := m.Object("problemNormalObject")
	testutil.NotNil(t, normalObj, "normal lowercase object should resolve")

	uppercaseNames := []string{
		"NetEngine8000SysOid",
		"S6730-H28Y4C",
		"S5735-L8T4X-A1",
		"NetEngine-A800",
	}

	for _, name := range uppercaseNames {
		t.Run(name, func(t *testing.T) {
			node := m.Node(name)
			testutil.NotNil(t, node, "%s should resolve in permissive mode", name)
		})
	}
}

// TestProblemNamingHyphens verifies that a diagnostic is emitted for
// identifiers containing hyphens in SMIv2 modules.
// Ground truth: smilint [5] "object identifier name should not include hyphens in SMIv2 MIB"
// PROBLEMS.md: PROBLEM-NAMING-MIB / Hyphens in SMIv2 object identifier names
func TestProblemNamingHyphens(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-NAMING-MIB")

	hyphenDiags := make(map[string]bool)
	for _, d := range m.Diagnostics() {
		if d.Code == "identifier-hyphen-smiv2" && d.Module == "PROBLEM-NAMING-MIB" {
			hyphenDiags[d.Message] = true
		}
	}

	wantFlagged := []string{"S6730-H28Y4C", "S5735-L8T4X-A1", "NetEngine-A800"}
	for _, name := range wantFlagged {
		found := false
		for msg := range hyphenDiags {
			if strings.Contains(msg, name) {
				found = true
				break
			}
		}
		testutil.True(t, found,
			"should emit identifier-hyphen-smiv2 diagnostic for %s", name)
	}

	for msg := range hyphenDiags {
		if strings.Contains(msg, "NetEngine8000SysOid") {
			t.Errorf("NetEngine8000SysOid should not be flagged (no hyphens), got: %s", msg)
		}
	}
}

func TestHexLiteralRanges(t *testing.T) {
	m := loadCorpusMIB(t, "INTEGRATED-SERVICES-MIB")

	for _, d := range m.Diagnostics() {
		if d.Module == "INTEGRATED-SERVICES-MIB" && d.Severity <= mib.SeverityError {
			t.Errorf("unexpected error-level diagnostic: %s", d.Message)
		}
	}

	typ := m.Type("MessageSize")
	testutil.NotNil(t, typ, "MessageSize type should exist")

	ranges := typ.Ranges()
	testutil.Equal(t, len(ranges), 1, "MessageSize should have 1 range")
	if len(ranges) == 1 {
		testutil.Equal(t, ranges[0].Min, int64(0), "MessageSize range min")
		testutil.Equal(t, ranges[0].Max, int64(2147483647), "MessageSize range max")
	}

	for _, name := range []string{"BitRate", "BurstSize"} {
		typ := m.Type(name)
		testutil.NotNil(t, typ, "%s type should exist", name)

		ranges := typ.Ranges()
		testutil.Equal(t, len(ranges), 1, "%s should have 1 range", name)
		if len(ranges) == 1 {
			testutil.Equal(t, ranges[0].Min, int64(0), "%s range min", name)
			testutil.Equal(t, ranges[0].Max, int64(2147483647), "%s range max", name)
		}
	}
}

func TestHexLiteralDefval(t *testing.T) {
	m := loadCorpusMIB(t, "RIPv2-MIB")

	for _, d := range m.Diagnostics() {
		if d.Module == "RIPv2-MIB" && d.Severity <= mib.SeverityError {
			t.Errorf("unexpected error-level diagnostic: %s", d.Message)
		}
	}

	obj := m.Object("rip2IfConfDomain")
	testutil.NotNil(t, obj, "rip2IfConfDomain should exist")

	defval := obj.DefaultValue()
	testutil.True(t, !defval.IsZero(), "rip2IfConfDomain should have DEFVAL")
}

func TestModuleIdentityPermissive(t *testing.T) {
	t.Run("IPV6-TC", func(t *testing.T) {
		m := loadCorpusMIB(t, "IPV6-TC", WithResolverStrictness(mib.ResolverPermissive))

		for _, d := range m.Diagnostics() {
			if d.Module == "IPV6-TC" && d.Severity <= mib.SeverityError {
				t.Errorf("unexpected error-level diagnostic in permissive mode: code=%s, severity=%v, msg=%s",
					d.Code, d.Severity, d.Message)
			}
		}

		testutil.NotNil(t, m.Type("Ipv6Address"), "Ipv6Address type should exist")
		testutil.NotNil(t, m.Type("Ipv6AddressPrefix"), "Ipv6AddressPrefix type should exist")
	})

	t.Run("IPV6-MIB", func(t *testing.T) {
		m := loadCorpusMIB(t, "IPV6-MIB", WithResolverStrictness(mib.ResolverPermissive))

		for _, d := range m.Diagnostics() {
			if d.Module == "IPV6-MIB" && d.Severity <= mib.SeverityError {
				t.Errorf("unexpected error-level diagnostic in permissive mode: code=%s, severity=%v, msg=%s",
					d.Code, d.Severity, d.Message)
			}
		}

		testutil.NotNil(t, m.Object("ipv6IfDescr"), "ipv6IfDescr should exist")
		testutil.NotNil(t, m.Object("ipv6IfIndex"), "ipv6IfIndex should exist")
	})
}

// TestProblemSharedOIDs tests that when two SMIv2 modules with identical
// LAST-UPDATED dates define the same OBJECT IDENTIFIER value assignments,
// both modules' Nodes() lists contain those nodes.
//
// Reproduces: HOST-RESOURCES-MIB and HOST-RESOURCES-TYPES both define
// hrStorageTypes, hrDeviceTypes, hrFSTypes. With identical LAST-UPDATED
// dates, shouldPreferModule returns false for the second module, so its
// nodes never get added via addNode().
func TestProblemSharedOIDs(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-SHARED-OIDS-MIB")

	sharedMod := m.Module("PROBLEM-SHARED-OIDS-MIB")
	if sharedMod == nil {
		t.Fatal("PROBLEM-SHARED-OIDS-MIB not found")
	}

	baseMod := m.Module("PROBLEM-SHARED-OIDS-BASE-MIB")
	if baseMod == nil {
		t.Fatal("PROBLEM-SHARED-OIDS-BASE-MIB not found")
	}

	sharedOIDs := []string{"sharedBranch1", "sharedBranch2", "sharedBranch3"}

	// The shared nodes should be globally findable
	for _, name := range sharedOIDs {
		node := m.Node(name)
		if node == nil {
			t.Errorf("global Node(%s) returned nil", name)
		}
	}

	// Both modules define these OIDs. Each module's Nodes() list
	// should contain them.
	t.Run("base module nodes", func(t *testing.T) {
		baseNodes := make(map[string]bool)
		for _, n := range baseMod.Nodes() {
			baseNodes[n.Name()] = true
		}
		for _, name := range sharedOIDs {
			if !baseNodes[name] {
				t.Errorf("PROBLEM-SHARED-OIDS-BASE-MIB.Nodes() missing %s", name)
			}
		}
	})

	t.Run("shared module nodes", func(t *testing.T) {
		sharedNodes := make(map[string]bool)
		for _, n := range sharedMod.Nodes() {
			sharedNodes[n.Name()] = true
		}
		for _, name := range sharedOIDs {
			if !sharedNodes[name] {
				t.Errorf("PROBLEM-SHARED-OIDS-MIB.Nodes() missing %s", name)
			}
		}
	})

	// OBJECT-IDENTITY nodes unique to the shared module should always be present
	uniqueNodes := []string{"sharedItem1", "sharedItem2", "sharedItem3"}
	t.Run("unique nodes in shared module", func(t *testing.T) {
		sharedNodes := make(map[string]bool)
		for _, n := range sharedMod.Nodes() {
			sharedNodes[n.Name()] = true
		}
		for _, name := range uniqueNodes {
			if !sharedNodes[name] {
				t.Errorf("PROBLEM-SHARED-OIDS-MIB.Nodes() missing unique node %s", name)
			}
		}
	})
}

// TestProblemCasefoldIntegerType verifies that "Integer" (non-standard casing
// of INTEGER) is recognized as a base integer type.
// Real: CT-SERIES-1000-MIB wirelessSimActive uses SYNTAX Integer.
// net-snmp: accepts silently.
func TestProblemCasefoldIntegerType(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-CASEFOLDING-MIB")

	obj := m.Object("problemCasefoldInteger")
	testutil.NotNil(t, obj, "Object(problemCasefoldInteger)")

	typ := obj.Type()
	testutil.NotNil(t, typ, "Type() for problemCasefoldInteger")
	testutil.Equal(t, mib.BaseInteger32, typ.EffectiveBase(),
		"Integer should resolve to BaseInteger32")
}

// TestProblemCasefoldLowercaseTable verifies that a table using a lowercase
// SEQUENCE type name parses and resolves correctly in permissive mode.
// Real: CT-SERIES-1000-MIB wirelessSimTableEntry uses lowercase for both
// the SYNTAX type reference and the ::= SEQUENCE type assignment.
// net-snmp: accepts silently.
func TestProblemCasefoldLowercaseTable(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-CASEFOLDING-MIB")

	// No error-level diagnostics for this module
	for _, d := range m.Diagnostics() {
		if d.Module == "PROBLEM-CASEFOLDING-MIB" && d.Severity <= mib.SeverityError {
			t.Errorf("unexpected error diagnostic: code=%s msg=%s", d.Code, d.Message)
		}
	}

	// Table object
	tbl := m.Object("problemCasefoldTable")
	testutil.NotNil(t, tbl, "Object(problemCasefoldTable)")
	testutil.Equal(t, "table", normalizeKind(tbl.Kind()), "table kind")

	// Row entry
	entry := m.Object("problemCasefoldEntry")
	testutil.NotNil(t, entry, "Object(problemCasefoldEntry)")
	testutil.Equal(t, "row", normalizeKind(entry.Kind()), "entry kind")

	// Index
	indexes := normalizeIndexes(entry.Index())
	testutil.Len(t, indexes, 1, "entry should have one index")
	testutil.Equal(t, "problemCasefoldIndex", indexes[0].Name, "index name")

	// Columns
	columns := []struct {
		name     string
		wantBase mib.BaseType
	}{
		{"problemCasefoldIndex", mib.BaseInteger32},
		{"problemCasefoldName", mib.BaseOctetString},
		{"problemCasefoldValue", mib.BaseInteger32},
	}
	for _, col := range columns {
		t.Run(col.name, func(t *testing.T) {
			obj := m.Object(col.name)
			testutil.NotNil(t, obj, "Object(%s)", col.name)
			testutil.Equal(t, "column", normalizeKind(obj.Kind()), "column kind for %s", col.name)

			typ := obj.Type()
			testutil.NotNil(t, typ, "Type() for %s", col.name)
			testutil.Equal(t, col.wantBase, typ.EffectiveBase(),
				"base type for %s", col.name)
		})
	}
}

// TestProblemDuplicateImportDeterminism verifies that a MIB importing the same
// symbol from two different modules produces deterministic results across
// loads. The first import in the MIB file should win.
func TestProblemDuplicateImportDeterminism(t *testing.T) {
	const iterations = 10

	type result struct {
		isTC       bool
		moduleName string
	}

	var results []result

	for range iterations {
		m := loadProblemMIB(t, "PROBLEM-DUPLICATE-IMPORT-MIB")

		obj := m.Object("problemDupObject")
		if obj == nil {
			t.Fatal("problemDupObject not found")
		}
		typ := obj.Type()
		if typ == nil {
			t.Fatalf("object %s has no type", obj.Name())
		}

		r := result{isTC: typ.IsTextualConvention()}
		if typ.Module() != nil {
			r.moduleName = typ.Module().Name()
		}
		results = append(results, r)
	}

	// All iterations should produce the same result.
	first := results[0]
	for i, r := range results[1:] {
		if r.isTC != first.isTC || r.moduleName != first.moduleName {
			t.Fatalf("nondeterminism at iteration %d: got module=%s isTC=%v, want module=%s isTC=%v",
				i+1, r.moduleName, r.isTC, first.moduleName, first.isTC)
		}
	}

	// The MIB lists DisplayString FROM RFC1213-MIB first,
	// so RFC1213-MIB's version should be used.
	testutil.Equal(t, "RFC1213-MIB", first.moduleName, "expected first-listed import to win")
	testutil.Equal(t, false, first.isTC, "RFC1213-MIB:DisplayString should not be a TC")
}

// TestProblemGroupMembership verifies that group-membership diagnostics are
// emitted for accessible objects and notifications not in any conformance group.
func TestProblemGroupMembership(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-GROUP-MEMBERSHIP-MIB")

	// Collect all group-membership diagnostics.
	var diags []string
	for _, d := range m.Diagnostics() {
		if d.Code == "group-membership" && d.Module == "PROBLEM-GROUP-MEMBERSHIP-MIB" {
			diags = append(diags, d.Message)
		}
	}

	// Expect diagnostics for uncovered objects.
	wantMessages := []string{
		`"problemGMUncoveredScalar" is not in any OBJECT-GROUP`,
		`"problemGMUncoveredCol" is not in any OBJECT-GROUP`,
		`"problemGMUncoveredNotif" is not in any NOTIFICATION-GROUP`,
	}
	for _, want := range wantMessages {
		testutil.True(t, slices.Contains(diags, want), "expected diagnostic: %s", want)
	}

	// Should not report covered objects, not-accessible, tables, or rows.
	noReport := []string{
		"problemGMCoveredScalar",
		"problemGMCoveredCol",
		"problemGMCoveredNotif",
		"problemGMIndex",
		"problemGMTable",
		"problemGMEntry",
	}
	for _, name := range noReport {
		for _, msg := range diags {
			if strings.Contains(msg, name) {
				t.Errorf("should not report %s, got: %s", name, msg)
			}
		}
	}
}
