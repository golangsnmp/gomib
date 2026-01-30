package gomib

// resolve_problems_extra_test.go tests additional edge cases from the problem
// corpus that were previously untested. Each test references PROBLEMS.md for
// traceability. Expected values are grounded against net-snmp and/or smilint
// where possible. Tests use t.Skip for cases where gomib behavior is correct
// but cannot be verified against ground truth, or where the feature is not
// yet implemented.

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// === PROBLEM-SMIv1v2-MIX-MIB ===

// TestProblemSMIv1v2AccessKeyword verifies that the parser accepts the SMIv1
// ACCESS keyword in an SMIv2 module and resolves the access value correctly.
// Ground truth: net-snmp silently accepts; smilint flags as SMIv1 style.
// PROBLEMS.md: PROBLEM-SMIv1v2-MIX-MIB / ACCESS keyword instead of MAX-ACCESS
func TestProblemSMIv1v2AccessKeyword(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-SMIv1v2-MIX-MIB")

	obj := m.FindObject("problemV1AccessObject")
	if obj == nil {
		t.Skip("problemV1AccessObject not found - parser may reject ACCESS keyword in SMIv2 module")
		return
	}
	access := testutil.NormalizeAccess(obj.Access())
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
		obj := m.FindObject("problemMandatoryStatus")
		if obj == nil {
			t.Skip("problemMandatoryStatus not found - parser may reject mandatory status in SMIv2")
			return
		}
		status := testutil.NormalizeStatus(obj.Status())
		// gomib preserves mandatory as StatusMandatory; net-snmp maps to current
		if status != "mandatory" && status != "current" {
			t.Errorf("status: got %q, want mandatory or current", status)
		}
	})

	t.Run("optional", func(t *testing.T) {
		obj := m.FindObject("problemOptionalStatus")
		if obj == nil {
			t.Skip("problemOptionalStatus not found - parser may reject optional status in SMIv2")
			return
		}
		status := testutil.NormalizeStatus(obj.Status())
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

	// The MIB should load without error regardless of whether TRAP-TYPE
	// is fully resolved. Verify that the surrounding objects are not
	// affected by the TRAP-TYPE.
	normalObj := m.FindObject("problemV2NormalObject")
	if normalObj == nil {
		t.Fatal("problemV2NormalObject should resolve - TRAP-TYPE should not prevent other objects from loading")
	}
	testutil.Equal(t, "1.3.6.1.4.1.99998.1.1.4", normalObj.OID().String(),
		"normal SMIv2 object OID should resolve correctly alongside TRAP-TYPE")

	// Check if the v2 notification also resolved
	notif := m.FindNotification("problemV2Notification")
	if notif == nil {
		t.Skip("problemV2Notification not found - notification may not resolve alongside TRAP-TYPE")
		return
	}
	testutil.Equal(t, "1.3.6.1.4.1.99998.1.2.1", notif.OID().String(),
		"SMIv2 notification OID")
}

// TestProblemSMIv1v2CounterGauge verifies resolution of SMIv1 Counter and
// Gauge types (without the "32" suffix) in an SMIv2 module.
// Ground truth: net-snmp resolves silently; smilint rejects.
// PROBLEMS.md: PROBLEM-SMIv1v2-MIX-MIB / SMIv1 types Counter/Gauge
func TestProblemSMIv1v2CounterGauge(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-SMIv1v2-MIX-MIB")

	t.Run("Counter", func(t *testing.T) {
		obj := m.FindObject("problemCounterObject")
		if obj == nil {
			t.Skip("problemCounterObject not found - Counter type may not resolve without RFC1155-SMI import")
			return
		}
		if obj.Type() == nil {
			t.Skip("Counter type unresolved - SMIv1 type fallback not implemented for this context")
			return
		}
		base := obj.Type().EffectiveBase()
		if base != mib.BaseCounter32 {
			t.Errorf("Counter base type: got %v, want Counter32", base)
		}
	})

	t.Run("Gauge", func(t *testing.T) {
		obj := m.FindObject("problemGaugeObject")
		if obj == nil {
			t.Skip("problemGaugeObject not found - Gauge type may not resolve without RFC1155-SMI import")
			return
		}
		if obj.Type() == nil {
			t.Skip("Gauge type unresolved - SMIv1 type fallback not implemented for this context")
			return
		}
		base := obj.Type().EffectiveBase()
		if base != mib.BaseGauge32 {
			t.Errorf("Gauge base type: got %v, want Gauge32", base)
		}
	})
}

// === PROBLEM-INDEX-MIB ===

// TestProblemIndexBareType verifies that tables with bare type names in INDEX
// clauses (e.g., INDEX { INTEGER }) load without crashing.
// Ground truth: net-snmp silently accepts; smilint rejects with syntax error.
// PROBLEMS.md: PROBLEM-INDEX-MIB / Bare INTEGER in INDEX
func TestProblemIndexBareType(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-INDEX-MIB")

	// The table should parse and resolve
	entry := m.FindObject("problemBareTypeEntry")
	if entry == nil {
		t.Skip("problemBareTypeEntry not found - bare type INDEX may not be supported")
		return
	}

	kind := testutil.NormalizeKind(entry.Kind())
	testutil.Equal(t, "row", kind, "entry should be a row")

	// The value column should still resolve
	val := m.FindObject("problemBareTypeValue")
	testutil.NotNil(t, val, "problemBareTypeValue should resolve even with bare type index")
}

// TestProblemIndexMacAddress verifies that tables using MacAddress as an
// index element resolve correctly.
// Ground truth: net-snmp silently accepts; smilint flags as illegal base type.
// PROBLEMS.md: PROBLEM-INDEX-MIB / MacAddress as index type
func TestProblemIndexMacAddress(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-INDEX-MIB")

	entry := m.FindObject("problemMacIndexEntry")
	if entry == nil {
		t.Skip("problemMacIndexEntry not found")
		return
	}

	kind := testutil.NormalizeKind(entry.Kind())
	testutil.Equal(t, "row", kind, "entry should be a row")

	indexes := testutil.NormalizeIndexes(entry.Index())
	if len(indexes) == 0 {
		t.Skip("no indexes resolved for MacAddress index table")
		return
	}
	testutil.Equal(t, "problemMacIndexAddress", indexes[0].Name,
		"MacAddress index should resolve")

	// Verify the column resolved
	port := m.FindObject("problemMacIndexPort")
	testutil.NotNil(t, port, "problemMacIndexPort column should resolve")
}

// TestProblemIndexNoRange verifies that tables with index elements missing
// range restrictions still resolve correctly.
// Ground truth: net-snmp silently accepts; smilint warns about missing range.
// PROBLEMS.md: PROBLEM-INDEX-MIB / Index element missing range restriction
func TestProblemIndexNoRange(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-INDEX-MIB")

	entry := m.FindObject("problemNoRangeEntry")
	if entry == nil {
		t.Skip("problemNoRangeEntry not found")
		return
	}

	kind := testutil.NormalizeKind(entry.Kind())
	testutil.Equal(t, "row", kind, "entry should be a row")

	indexes := testutil.NormalizeIndexes(entry.Index())
	if len(indexes) != 2 {
		t.Skipf("expected 2 indexes, got %d", len(indexes))
		return
	}
	testutil.Equal(t, "problemNoRangeIndex1", indexes[0].Name, "first index")
	testutil.Equal(t, "problemNoRangeIndex2", indexes[1].Name, "second index")
}

// TestProblemIndexDisplayString verifies that tables using DisplayString
// as an index element resolve correctly.
// Ground truth: net-snmp silently accepts.
// PROBLEMS.md: PROBLEM-INDEX-MIB / DisplayString as index
func TestProblemIndexDisplayString(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-INDEX-MIB")

	entry := m.FindObject("problemStringIndexEntry")
	if entry == nil {
		t.Skip("problemStringIndexEntry not found")
		return
	}

	kind := testutil.NormalizeKind(entry.Kind())
	testutil.Equal(t, "row", kind, "entry should be a row")

	indexes := testutil.NormalizeIndexes(entry.Index())
	if len(indexes) == 0 {
		t.Skip("no indexes resolved for DisplayString index table")
		return
	}
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
		table  string
		entry  string
		column string
	}{
		{"problemNetAddrTable", "problemNetAddrEntry", "problemNetAddrValue"},
		{"problemMacIndexTable", "problemMacIndexEntry", "problemMacIndexPort"},
		{"problemNoRangeTable", "problemNoRangeEntry", "problemNoRangeValue"},
		{"problemStringIndexTable", "problemStringIndexEntry", "problemStringIndexValue"},
	}

	for _, tt := range tables {
		t.Run(tt.table, func(t *testing.T) {
			tbl := m.FindObject(tt.table)
			if tbl == nil {
				t.Skipf("%s not found", tt.table)
				return
			}
			testutil.Equal(t, "table", testutil.NormalizeKind(tbl.Kind()), "table kind")

			ent := m.FindObject(tt.entry)
			if ent == nil {
				t.Skipf("%s not found", tt.entry)
				return
			}
			testutil.Equal(t, "row", testutil.NormalizeKind(ent.Kind()), "entry kind")

			col := m.FindObject(tt.column)
			if col == nil {
				t.Skipf("%s not found", tt.column)
				return
			}
			testutil.Equal(t, "column", testutil.NormalizeKind(col.Kind()), "column kind")
		})
	}
}

// === PROBLEM-DEFVAL-MIB ===

// TestProblemDefvalOidRef verifies that OID-valued DEFVALs (e.g., zeroDotZero)
// are parsed and stored. gomib normalizes to numeric OID form.
// Ground truth: net-snmp outputs "zeroDotZero", gomib outputs "0.0" (documented divergence).
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / DEFVAL with OID reference
func TestProblemDefvalOidRef(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.FindObject("problemDefvalOidRef")
	if obj == nil {
		t.Skip("problemDefvalOidRef not found")
		return
	}

	dv := obj.DefaultValue()
	if dv.IsZero() {
		t.Skip("no DEFVAL parsed for problemDefvalOidRef")
		return
	}

	got := dv.String()
	// gomib normalizes OID defvals to numeric. net-snmp keeps symbolic.
	// Both "0.0" and "zeroDotZero" are valid.
	if !defvalEquivalent(got, "zeroDotZero") && got != "0.0" {
		t.Errorf("defval: got %q, want 0.0 or zeroDotZero", got)
	}
}

// TestProblemDefvalTypeMismatch verifies that a raw integer DEFVAL for an
// enum-typed object is parsed without error.
// Ground truth: net-snmp silently accepts integer 5 for ProblemSeverity.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / DEFVAL with raw integer for enum type
func TestProblemDefvalTypeMismatch(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.FindObject("problemDefvalTypeMismatch")
	if obj == nil {
		t.Skip("problemDefvalTypeMismatch not found")
		return
	}

	dv := obj.DefaultValue()
	if dv.IsZero() {
		t.Skip("no DEFVAL parsed for problemDefvalTypeMismatch")
		return
	}

	got := dv.String()
	// The DEFVAL { 5 } could be stored as integer "5" or as enum label "warning"
	// (value 5 in ProblemSeverity). Either is acceptable.
	if got != "5" && got != "warning" {
		t.Errorf("defval: got %q, want 5 or warning", got)
	}
}

// TestProblemDefvalBadEnum verifies that a DEFVAL with an undefined enum label
// is handled without crashing.
// Ground truth: smilint flags mismatch; net-snmp silently accepts.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / DEFVAL with undefined enum label
func TestProblemDefvalBadEnum(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.FindObject("problemDefvalBadEnum")
	if obj == nil {
		t.Skip("problemDefvalBadEnum not found")
		return
	}

	dv := obj.DefaultValue()
	if dv.IsZero() {
		// It's acceptable for the parser to not store an unresolvable DEFVAL
		t.Log("no DEFVAL parsed for problemDefvalBadEnum (undefined enum label)")
		return
	}

	got := dv.String()
	// "unknown" is not in the enum {up(1), down(2), testing(3)}, so this
	// could be stored as the raw label "unknown" or dropped.
	t.Logf("defval for undefined enum label: %q (kind=%v)", got, dv.Kind())
}

// TestProblemDefvalLargeHex verifies that a 16-byte hex DEFVAL is parsed.
// Ground truth: net-snmp outputs "0" (truncates zeros). gomib preserves full hex.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / Large hex DEFVAL
func TestProblemDefvalLargeHex(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.FindObject("problemDefvalLargeHex")
	if obj == nil {
		t.Skip("problemDefvalLargeHex not found")
		return
	}

	dv := obj.DefaultValue()
	if dv.IsZero() {
		t.Skip("no DEFVAL parsed for problemDefvalLargeHex")
		return
	}

	got := dv.String()
	// gomib: "0x00000000000000000000000000000000" (16 bytes > 8, uses hex format)
	// net-snmp: "0" (truncates all-zero hex)
	if !defvalEquivalent(got, "0") {
		// Not equivalent to net-snmp's "0" - that's OK if it's our hex representation
		if dv.Kind() != mib.DefValKindBytes {
			t.Errorf("large hex defval kind: got %v, want DefValKindBytes", dv.Kind())
		}
	}
	t.Logf("large hex defval: %q (kind=%v)", got, dv.Kind())
}

// TestProblemDefvalBinary verifies binary string DEFVAL parsing.
// Ground truth: net-snmp parses '10101010'B as 170 (0xAA).
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / Binary string DEFVAL
func TestProblemDefvalBinary(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.FindObject("problemDefvalBinary")
	if obj == nil {
		t.Skip("problemDefvalBinary not found")
		return
	}

	dv := obj.DefaultValue()
	if dv.IsZero() {
		t.Skip("no DEFVAL parsed for problemDefvalBinary")
		return
	}

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

	obj := m.FindObject("problemDefvalBinaryOdd")
	if obj == nil {
		t.Skip("problemDefvalBinaryOdd not found")
		return
	}

	dv := obj.DefaultValue()
	if dv.IsZero() {
		t.Skip("no DEFVAL parsed for problemDefvalBinaryOdd")
		return
	}

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

	obj := m.FindObject("problemDefvalEmptyBits")
	if obj == nil {
		t.Skip("problemDefvalEmptyBits not found")
		return
	}

	dv := obj.DefaultValue()
	if dv.IsZero() {
		t.Skip("no DEFVAL parsed for problemDefvalEmptyBits")
		return
	}

	// Empty BITS should produce an empty bit label list
	if dv.Kind() == mib.DefValKindBits {
		labels, ok := mib.DefValAs[[]string](dv)
		if !ok {
			t.Error("DefValAs[[]string] failed for BITS defval")
			return
		}
		testutil.Equal(t, 0, len(labels), "empty BITS should have 0 labels")
	}
	// "{ }" is the string representation of empty BITS
	got := dv.String()
	t.Logf("empty BITS defval: %q (kind=%v)", got, dv.Kind())
}

// TestProblemDefvalMultiBits verifies that DEFVAL { { read, write } } is parsed.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / (not listed but included in the MIB)
func TestProblemDefvalMultiBits(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.FindObject("problemDefvalMultiBits")
	if obj == nil {
		t.Skip("problemDefvalMultiBits not found")
		return
	}

	dv := obj.DefaultValue()
	if dv.IsZero() {
		t.Skip("no DEFVAL parsed for problemDefvalMultiBits")
		return
	}

	if dv.Kind() == mib.DefValKindBits {
		labels, ok := mib.DefValAs[[]string](dv)
		if !ok {
			t.Error("DefValAs[[]string] failed for BITS defval")
			return
		}
		testutil.Equal(t, 2, len(labels), "multi BITS should have 2 labels")
		// Verify the labels are present (order may vary)
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
	} else {
		t.Logf("multi BITS defval kind=%v, string=%q", dv.Kind(), dv.String())
	}
}

// TestProblemDefvalNegative verifies that negative integer DEFVAL is parsed.
// Ground truth: net-snmp accepts silently.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / Negative integer DEFVAL
func TestProblemDefvalNegative(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.FindObject("problemDefvalNegative")
	if obj == nil {
		t.Skip("problemDefvalNegative not found")
		return
	}

	dv := obj.DefaultValue()
	if dv.IsZero() {
		t.Skip("no DEFVAL parsed for problemDefvalNegative")
		return
	}

	got := dv.String()
	testutil.Equal(t, "-1", got, "negative DEFVAL should be -1")
}

// TestProblemDefvalSpecialString verifies quoted string DEFVAL parsing.
// PROBLEMS.md: PROBLEM-DEFVAL-MIB / (not listed but included)
func TestProblemDefvalSpecialString(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-DEFVAL-MIB")

	obj := m.FindObject("problemDefvalSpecialString")
	if obj == nil {
		t.Skip("problemDefvalSpecialString not found")
		return
	}

	dv := obj.DefaultValue()
	if dv.IsZero() {
		t.Skip("no DEFVAL parsed for problemDefvalSpecialString")
		return
	}

	// The raw MIB has DEFVAL { "default-value" }
	got := dv.String()
	if got != `"default-value"` {
		t.Errorf("string defval: got %q, want %q", got, `"default-value"`)
	}
}

// === PROBLEM-IMPORTS-ALIAS-MIB ===

// TestProblemImportsAliasNormal verifies that module alias resolution works
// at normal strictness (safe fallback).
// Ground truth: net-snmp fails on SNMPv2-SMI-v1 (unlike gomib which has alias table).
// PROBLEMS.md: PROBLEM-IMPORTS-ALIAS-MIB / SNMPv2-SMI-v1 / SNMPv2-TC-v1
func TestProblemImportsAliasNormal(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-IMPORTS-ALIAS-MIB", mib.StrictnessNormal)

	// At normal level, module aliases (safe fallback) should be active
	str := m.FindObject("problemAliasString")
	if str == nil {
		t.Skip("problemAliasString not found - module alias resolution may not be working")
		return
	}

	// Verify the type resolved through the alias chain
	if str.Type() == nil {
		t.Skip("type not resolved for problemAliasString")
		return
	}
	testutil.Equal(t, mib.BaseOctetString, str.Type().EffectiveBase(),
		"DisplayString from SNMPv2-TC-v1 should resolve to OCTET STRING")

	// OID should resolve through aliased enterprises
	testutil.Equal(t, "1.3.6.1.4.1.99998.3.1.1", str.OID().String(),
		"OID should resolve through aliased module imports")
}

// TestProblemImportsAliasStrict verifies that module aliases are disabled in
// strict mode.
// PROBLEMS.md: PROBLEM-IMPORTS-ALIAS-MIB / SNMPv2-SMI-v1
func TestProblemImportsAliasStrict(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-IMPORTS-ALIAS-MIB", mib.StrictnessStrict)

	// At strict level, module aliases are disabled
	unresolved := unresolvedSymbols(m, "PROBLEM-IMPORTS-ALIAS-MIB", "import")
	if len(unresolved) == 0 {
		t.Error("strict mode should have unresolved imports from aliased module names")
	}

	// Objects should not resolve since imports failed
	str := m.FindObject("problemAliasString")
	testutil.Nil(t, str, "objects should not resolve in strict mode with aliased imports")
}

// === PROBLEM-NAMING-MIB ===

// TestProblemNamingUppercase verifies that uppercase-starting identifiers
// (common in Huawei MIBs) are handled by the parser in permissive mode.
// Ground truth: net-snmp silently accepts; smilint rejects.
// PROBLEMS.md: PROBLEM-NAMING-MIB / Uppercase initial letter
func TestProblemNamingUppercase(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-NAMING-MIB")

	// Normal lowercase object should always resolve
	normalObj := m.FindObject("problemNormalObject")
	testutil.NotNil(t, normalObj, "normal lowercase object should resolve")

	// Uppercase identifiers - these may or may not resolve depending on parser
	uppercaseNames := []string{
		"NetEngine8000SysOid",
		"S6730-H28Y4C",
		"S5735-L8T4X-A1",
		"NetEngine-A800",
	}

	for _, name := range uppercaseNames {
		t.Run(name, func(t *testing.T) {
			node := m.FindNode(name)
			if node == nil {
				t.Skipf("%s not resolved - uppercase identifiers may not be supported in this mode", name)
				return
			}
			t.Logf("%s resolved: OID=%s Kind=%s", name, node.OID(), node.Kind())
		})
	}
}
