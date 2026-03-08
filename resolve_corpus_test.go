package gomib

// resolve_corpus_test.go exercises resolver features that are absent from the
// net-snmp fixture MIBs. The fixture set (IF-MIB, SNMPv2-MIB, IP-MIB,
// ENTITY-MIB, BRIDGE-MIB) has zero BITS-typed nodes, zero IMPLIED indexes,
// no bare type indexes, no TRAP-TYPE definitions, and only validates Kind for
// rows. These tests verify those features against real IETF MIBs where the
// expected values are grounded in the MIB source text.

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// TestCorpusBitsResolution verifies that BITS-typed OBJECT-TYPE nodes have
// their bit labels resolved. DISMAN-EVENT-MIB has four inline BITS objects.
// Covers: codeaudit.md "Zero BITS-typed nodes"
func TestCorpusBitsResolution(t *testing.T) {
	m := loadCorpusMIB(t, "DISMAN-EVENT-MIB")

	tests := []struct {
		name string
		bits map[int]string
	}{
		{"mteTriggerTest", map[int]string{0: "existence", 1: "boolean", 2: "threshold"}},
		{"mteTriggerExistenceTest", map[int]string{0: "present", 1: "absent", 2: "changed"}},
		{"mteTriggerExistenceStartup", map[int]string{0: "present", 1: "absent"}},
		{"mteEventActions", map[int]string{0: "notification", 1: "set"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := m.Object(tt.name)
			testutil.NotNil(t, obj, "Object(%s)", tt.name)

			typ := obj.Type()
			testutil.NotNil(t, typ, "Type() for %s", tt.name)
			testutil.Equal(t, mib.BaseBits, typ.EffectiveBase(), "base type for %s", tt.name)

			gomibBits := normalizeEnums(obj.EffectiveBits())
			testutil.Equal(t, len(tt.bits), len(gomibBits), "bit count for %s", tt.name)
			for val, label := range tt.bits {
				got, ok := gomibBits[val]
				testutil.True(t, ok, "bit %d should exist for %s", val, tt.name)
				testutil.Equal(t, label, got, "bit %d label for %s", val, tt.name)
			}
		})
	}
}

// TestCorpusImpliedIndexes verifies that IMPLIED indexes have their Implied
// flag set to true. SNMP-TARGET-MIB has two tables with IMPLIED indexes.
// Covers: codeaudit.md "Zero IMPLIED indexes"
func TestCorpusImpliedIndexes(t *testing.T) {
	m := loadCorpusMIB(t, "SNMP-TARGET-MIB")

	tests := []struct {
		entry   string
		indexes []testutil.IndexInfo
	}{
		{
			"snmpTargetAddrEntry",
			[]testutil.IndexInfo{{Name: "snmpTargetAddrName", Implied: true}},
		},
		{
			"snmpTargetParamsEntry",
			[]testutil.IndexInfo{{Name: "snmpTargetParamsName", Implied: true}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.entry, func(t *testing.T) {
			obj := m.Object(tt.entry)
			testutil.NotNil(t, obj, "Object(%s)", tt.entry)

			indexes := normalizeIndexes(obj.Index())
			testutil.Len(t, indexes, len(tt.indexes), "index count for %s", tt.entry)

			for i, want := range tt.indexes {
				testutil.Equal(t, want.Name, indexes[i].Name,
					"index[%d] name for %s", i, tt.entry)
				testutil.Equal(t, want.Implied, indexes[i].Implied,
					"index[%d] implied for %s", i, tt.entry)
			}
		})
	}
}

// TestCorpusImpliedMixedIndexes verifies IMPLIED handling in tables with
// multiple indexes where only the last is IMPLIED. DISMAN-EVENT-MIB has
// entries like INDEX { mteOwner, IMPLIED mteTriggerName }.
func TestCorpusImpliedMixedIndexes(t *testing.T) {
	m := loadCorpusMIB(t, "DISMAN-EVENT-MIB")

	tests := []struct {
		entry   string
		indexes []testutil.IndexInfo
	}{
		{
			"mteTriggerEntry",
			[]testutil.IndexInfo{
				{Name: "mteOwner", Implied: false},
				{Name: "mteTriggerName", Implied: true},
			},
		},
		{
			"mteEventEntry",
			[]testutil.IndexInfo{
				{Name: "mteOwner", Implied: false},
				{Name: "mteEventName", Implied: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.entry, func(t *testing.T) {
			obj := m.Object(tt.entry)
			testutil.NotNil(t, obj, "Object(%s)", tt.entry)

			indexes := normalizeIndexes(obj.Index())
			testutil.Len(t, indexes, len(tt.indexes), "index count for %s", tt.entry)

			for i, want := range tt.indexes {
				testutil.Equal(t, want.Name, indexes[i].Name,
					"index[%d] name for %s", i, tt.entry)
				testutil.Equal(t, want.Implied, indexes[i].Implied,
					"index[%d] implied for %s", i, tt.entry)
			}
		})
	}
}

// TestCorpusTrapType verifies TRAP-TYPE resolution for SMIv1 notifications.
// SNMP-REPEATER-MIB defines three TRAP-TYPE macros with ENTERPRISE and
// VARIABLES clauses.
// Covers: codeaudit.md "No TRAP-TYPE definitions"
func TestCorpusTrapType(t *testing.T) {
	m := loadCorpusMIB(t, "SNMP-REPEATER-MIB")

	tests := []struct {
		name       string
		enterprise string
		trapNumber uint32
		variables  []string
	}{
		{"rptrHealth", "snmpDot3RptrMgt", 1, []string{"rptrOperStatus"}},
		{"rptrGroupChange", "snmpDot3RptrMgt", 2, []string{"rptrGroupIndex"}},
		{"rptrResetEvent", "snmpDot3RptrMgt", 3, []string{"rptrOperStatus"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notif := m.Notification(tt.name)
			testutil.NotNil(t, notif, "Notification(%s)", tt.name)

			// Verify OID is assigned (TRAP-TYPE uses enterprise.0.trapNumber).
			oid := notif.OID()
			testutil.True(t, len(oid) > 0, "OID for %s should be non-empty", tt.name)

			// Verify TrapInfo is populated.
			ti := notif.TrapInfo()
			testutil.NotNil(t, ti, "TrapInfo for %s", tt.name)
			testutil.Equal(t, tt.enterprise, ti.Enterprise, "enterprise for %s", tt.name)
			testutil.Equal(t, tt.trapNumber, ti.TrapNumber, "trap number for %s", tt.name)

			// Verify VARIABLES resolved to objects.
			objs := notif.Objects()
			testutil.Len(t, objs, len(tt.variables), "VARIABLES count for %s", tt.name)
			for i, wantName := range tt.variables {
				testutil.Equal(t, wantName, objs[i].Name(),
					"VARIABLES[%d] for %s", i, tt.name)
			}
		})
	}
}

// TestCorpusKindInference verifies that Kind is correctly inferred for all
// four OBJECT-TYPE classifications: table, row, column, and scalar.
// Uses IF-MIB which has clear examples of each.
// Covers: codeaudit.md "Kind only populated for rows"
func TestCorpusKindInference(t *testing.T) {
	m := loadCorpusMIB(t, "IF-MIB")

	tests := []struct {
		name string
		kind mib.Kind
	}{
		// Table: SYNTAX SEQUENCE OF
		{"ifTable", mib.KindTable},
		{"ifXTable", mib.KindTable},
		{"ifStackTable", mib.KindTable},
		// Row: has INDEX or AUGMENTS
		{"ifEntry", mib.KindRow},
		{"ifXEntry", mib.KindRow}, // AUGMENTS ifEntry
		{"ifStackEntry", mib.KindRow},
		// Column: child of row
		{"ifIndex", mib.KindColumn},
		{"ifDescr", mib.KindColumn},
		{"ifType", mib.KindColumn},
		{"ifMtu", mib.KindColumn},
		{"ifSpeed", mib.KindColumn},
		{"ifName", mib.KindColumn},      // from ifXEntry
		{"ifHighSpeed", mib.KindColumn}, // from ifXEntry
		// Scalar: not in any table
		{"ifNumber", mib.KindScalar},
		{"ifTableLastChange", mib.KindScalar},
		{"ifStackLastChange", mib.KindScalar},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := m.Object(tt.name)
			testutil.NotNil(t, obj, "Object(%s)", tt.name)
			testutil.Equal(t, tt.kind.String(), obj.Kind().String(),
				"kind for %s", tt.name)
		})
	}
}

// TestCorpusBaseModuleOwnership verifies that well-known OIDs (iso, org, dod,
// internet, etc.) are owned by their base modules even when vendor MIBs
// redeclare them as intermediate path components.
func TestCorpusBaseModuleOwnership(t *testing.T) {
	// Load vendor MIBs that redeclare well-known OIDs as path prefixes:
	//   IEEE8023-LAG-MIB: { iso(1) member-body(2) us(840) ... }
	//   RAPID-CITY: { iso org(3) dod(6) ... }
	m := loadCorpusMIB(t, "IEEE8023-LAG-MIB", WithModules("RAPID-CITY"))

	// These OIDs are defined by multiple base modules (SNMPv2-SMI and
	// RFC1155-SMI both define org, dod, internet, etc.). Either base module
	// is acceptable; the key invariant is that no vendor module owns them.
	wellKnown := []string{"iso", "org", "dod", "internet", "mgmt", "enterprises"}

	for _, name := range wellKnown {
		t.Run(name, func(t *testing.T) {
			node := m.Node(name)
			testutil.NotNil(t, node, "Node(%s)", name)
			mod := node.Module()
			testutil.NotNil(t, mod, "Module() for %s", name)
			testutil.True(t, mod.IsBase(),
				"module for %s should be a base module, got %s", name, mod.Name())
		})
	}
}

// TestCorpusBareTypeIndexes verifies that bare type indexes (INDEX { INTEGER })
// produce IndexEntry values with Object=nil and TypeName set.
// Already tested via TestProblemIndexBareType but included here for
// completeness of the fixture blind spot documentation.
// Covers: codeaudit.md "No bare type indexes"
func TestCorpusBareTypeIndexes(t *testing.T) {
	m := loadProblemMIB(t, "PROBLEM-INDEX-MIB")

	entry := m.Object("problemBareTypeEntry")
	testutil.NotNil(t, entry, "Object(problemBareTypeEntry)")

	idx := entry.Index()
	testutil.Len(t, idx, 1, "bare type entry should have one index")
	testutil.Nil(t, idx[0].Object, "bare type index Object should be nil")
	testutil.Equal(t, "INTEGER", idx[0].TypeName, "bare type index TypeName")
	testutil.Equal(t, false, idx[0].Implied, "bare type index should not be implied")
}
