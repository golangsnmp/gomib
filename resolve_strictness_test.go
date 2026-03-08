package gomib

// resolve_strictness_test.go verifies that resolution behavior changes with
// strictness levels, and tests import forwarding chains and partial resolution.
// The strictness system gates fallback resolution strategies:
//
//   - Constrained fallbacks (normal+): module aliases, import forwarding,
//     well-known type/OID roots
//   - Global-search fallbacks (permissive): global type lookup and semantic global lookup
//
// These tests load synthetic MIBs at different strictness levels and verify
// that resolution outcomes differ. Expected values are grounded against
// net-snmp (which always resolves at maximum permissiveness) and libsmi
// (which fails on unknown module names like SNMPv2-SMI-v1).

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func unresolvedSymbols(m *mib.Mib, module string, kind mib.UnresolvedKind) map[string]bool {
	result := make(map[string]bool)
	for _, u := range m.Unresolved() {
		if u.Module == module && u.Kind == kind {
			result[u.Symbol] = true
		}
	}
	return result
}

// TestTypeFallbackStrictness verifies that well-known SMI global types
// (Counter64, Gauge32, etc.) resolve in normal/permissive mode when not
// explicitly imported, but not in strict mode.
//
// PROBLEM-IMPORTS-MIB imports enterprises and Integer32 from SNMPv2-SMI but
// deliberately omits Counter64, Gauge32, Unsigned32, and TimeTicks. These are
// SMI base types that net-snmp resolves implicitly at all levels.
//
// Ground truth:
//   - net-snmp: always resolves (global type lookup, no import required)
//   - libsmi: "Counter64 implicitly defined, not imported from SNMPv2-SMI"
//   - gomib strict: types unresolved (constrained fallbacks disabled)
//   - gomib normal/permissive: types resolve via well-known type fallback
func TestTypeFallbackStrictness(t *testing.T) {
	smiTypes := []struct {
		object   string
		wantBase mib.BaseType
	}{
		{"problemMissingCounter64", mib.BaseCounter64},
		{"problemMissingGauge32", mib.BaseGauge32},
		{"problemMissingUnsigned32", mib.BaseUnsigned32},
		{"problemMissingTimeTicks", mib.BaseTimeTicks},
	}

	t.Run("strict", func(t *testing.T) {
		m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", mib.ResolverStrict)
		unresolved := unresolvedSymbols(m, "PROBLEM-IMPORTS-MIB", mib.UnresolvedType)

		for _, tt := range smiTypes {
			t.Run(tt.object, func(t *testing.T) {
				obj := m.Object(tt.object)
				testutil.NotNil(t, obj, "object should exist (OID resolves via imported enterprises)")
				testutil.Nil(t, obj.Type(), "type should be nil in strict mode (no global type fallback)")
				testutil.True(t, unresolved[tt.wantBase.String()],
					"type %s should be in unresolved list", tt.wantBase)
			})
		}
	})

	t.Run("normal", func(t *testing.T) {
		m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", mib.ResolverNormal)
		unresolved := unresolvedSymbols(m, "PROBLEM-IMPORTS-MIB", mib.UnresolvedType)

		for _, tt := range smiTypes {
			t.Run(tt.object, func(t *testing.T) {
				obj := m.Object(tt.object)
				testutil.NotNil(t, obj, "object should exist (OID resolves via imported enterprises)")
				testutil.NotNil(t, obj.Type(), "type should resolve in normal mode via constrained fallback")
				if obj.Type() != nil {
					testutil.Equal(t, tt.wantBase, obj.Type().EffectiveBase(),
						"base type for %s should match expected type", tt.object)
				}
				testutil.False(t, unresolved[tt.wantBase.String()],
					"type %s should NOT be in unresolved list", tt.wantBase)
			})
		}
	})

	t.Run("permissive", func(t *testing.T) {
		m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", mib.ResolverPermissive)
		unresolved := unresolvedSymbols(m, "PROBLEM-IMPORTS-MIB", mib.UnresolvedType)

		for _, tt := range smiTypes {
			t.Run(tt.object, func(t *testing.T) {
				obj := m.Object(tt.object)
				testutil.NotNil(t, obj, "object should exist")
				if obj == nil {
					return
				}
				testutil.NotNil(t, obj.Type(), "type should resolve in permissive mode")
				if obj.Type() != nil {
					testutil.Equal(t, tt.wantBase, obj.Type().EffectiveBase(),
						"base type for %s should match net-snmp", tt.object)
				}
				testutil.False(t, unresolved[tt.wantBase.String()],
					"type %s should NOT be in unresolved list", tt.wantBase)
			})
		}
	})
}

// TestTCFallbackStrictness verifies that textual convention types (DisplayString,
// TruthValue) from SNMPv2-TC resolve at normal/permissive levels but remain
// unresolved at strict level when not imported.
//
// Ground truth:
//   - net-snmp: resolves implicitly at all levels
//   - gomib: resolves at normal/permissive (constrained fallback), unresolved at strict
func TestTCFallbackStrictness(t *testing.T) {
	tcObjects := []struct {
		name     string
		wantType string
	}{
		{"problemMissingDisplayString", "OCTET STRING"},
		{"problemMissingTruthValue", "Integer32"},
	}

	// Strict: TC types are unresolved (no import, constrained fallbacks disabled)
	for _, lvlName := range []string{"strict"} {
		lvl := mib.ResolverStrict
		if lvlName == "normal" {
			lvl = mib.ResolverNormal
		}
		t.Run(lvlName, func(t *testing.T) {
			m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", lvl)

			for _, tc := range tcObjects {
				t.Run(tc.name, func(t *testing.T) {
					obj := m.Object(tc.name)
					testutil.NotNil(t, obj, "object should exist (OID resolves)")
					if obj == nil {
						return
					}
					testutil.Nil(t, obj.Type(),
						"TC type should be nil at %s (not imported)", lvlName)
				})
			}
		})
	}

	// Normal and permissive: TC types resolve via SNMPv2-TC fallback
	for _, lvl := range []struct {
		name  string
		level mib.ResolverStrictness
	}{
		{"normal", mib.ResolverNormal},
		{"permissive", mib.ResolverPermissive},
	} {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", lvl.level)

			for _, tc := range tcObjects {
				t.Run(tc.name, func(t *testing.T) {
					obj := m.Object(tc.name)
					testutil.NotNil(t, obj, "object should resolve")
					if obj == nil {
						return
					}
					testutil.NotNil(t, obj.Type(),
						"TC type should resolve at %s level", lvl.name)
					if obj.Type() == nil {
						return
					}
					gotType := normalizeType(obj.Type())
					testutil.Equal(t, tc.wantType, gotType,
						"TC base type (matches net-snmp)")
				})
			}
		})
	}
}

// TestModuleAliasStrictness verifies that module alias resolution
// (SNMPv2-SMI-v1 -> SNMPv2-SMI, SNMPv2-TC-v1 -> SNMPv2-TC) works at all
// non-strict levels.
//
// PROBLEM-IMPORTS-ALIAS-MIB imports all symbols from SNMPv2-SMI-v1 and
// SNMPv2-TC-v1, which are old names used by real MIBs like RADLAN-MIB.
//
// Ground truth:
//   - net-snmp: resolves aliases silently (has its own internal alias table)
//   - libsmi: "failed to locate module `SNMPv2-SMI-v1'" (no alias support)
//   - gomib: normal/permissive resolve aliases; strict does not
func TestModuleAliasStrictness(t *testing.T) {
	levels := []struct {
		name  string
		level mib.ResolverStrictness
		ok    bool
	}{
		{"strict", mib.ResolverStrict, false},
		{"normal", mib.ResolverNormal, true},
		{"permissive", mib.ResolverPermissive, true},
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadAtStrictness(t, "PROBLEM-IMPORTS-ALIAS-MIB", lvl.level)

			unresolvedImports := unresolvedSymbols(m, "PROBLEM-IMPORTS-ALIAS-MIB", mib.UnresolvedImport)
			str := m.Object("problemAliasString")
			intObj := m.Object("problemAliasInteger")

			if lvl.ok {
				testutil.Equal(t, 0, len(unresolvedImports),
					"no imports should be unresolved with alias resolution")
				testutil.NotNil(t, str, "problemAliasString should resolve")
				if str != nil && str.Type() != nil {
					testutil.Equal(t, mib.BaseOctetString, str.Type().EffectiveBase(),
						"DisplayString base type should be OCTET STRING")
				}
				testutil.NotNil(t, intObj, "problemAliasInteger should resolve")
				if intObj != nil && intObj.Type() != nil {
					testutil.Equal(t, mib.BaseInteger32, intObj.Type().EffectiveBase(),
						"Integer32 base type should be Integer32")
				}
				return
			}

			testutil.True(t, len(unresolvedImports) > 0,
				"strict mode should keep aliased imports unresolved")
			testutil.Nil(t, str, "problemAliasString should not resolve in strict mode")
			testutil.Nil(t, intObj, "problemAliasInteger should not resolve in strict mode")
		})
	}
}

// TestOIDGlobalRootStrictness verifies that OID definitions referencing
// "enterprises" without importing it resolve in normal/permissive mode.
// This is tested by MISSING-IMPORT-TEST-MIB in the strictness/violations corpus.
//
// Ground truth:
//   - net-snmp: resolves enterprises globally (implicit root knowledge)
//   - libsmi: depends on import; fails without it
//   - gomib strict: OID chain fails (enterprises not in scope)
//   - gomib normal/permissive: resolves via lookupSmiGlobalOidRoot()
//
// Note: load_test.go already tests this scenario; this test adds explicit
// OID value verification and unresolved ref checking.
func TestOIDGlobalRootStrictness(t *testing.T) {
	t.Run("strict", func(t *testing.T) {
		m := loadViolationMIB(t, "MISSING-IMPORT-TEST-MIB", mib.ResolverStrict)
		unresolvedOids := unresolvedSymbols(m, "MISSING-IMPORT-TEST-MIB", mib.UnresolvedOID)

		testutil.True(t, unresolvedOids["enterprises"],
			"enterprises OID should be unresolved in strict mode")
		testutil.Nil(t, m.Object("testObject"),
			"testObject should not resolve (OID chain broken)")
	})

	t.Run("normal", func(t *testing.T) {
		m := loadViolationMIB(t, "MISSING-IMPORT-TEST-MIB", mib.ResolverNormal)
		unresolvedOids := unresolvedSymbols(m, "MISSING-IMPORT-TEST-MIB", mib.UnresolvedOID)

		testutil.Equal(t, 0, len(unresolvedOids),
			"no OID should be unresolved in normal mode")
		obj := m.Object("testObject")
		testutil.NotNil(t, obj, "testObject should resolve in normal mode")
		if obj != nil {
			testutil.Equal(t, "1.3.6.1.4.1.99999.1", obj.OID().String(),
				"testObject OID should match expected value")
		}
	})

	t.Run("permissive", func(t *testing.T) {
		m := loadViolationMIB(t, "MISSING-IMPORT-TEST-MIB", mib.ResolverPermissive)
		unresolvedOids := unresolvedSymbols(m, "MISSING-IMPORT-TEST-MIB", mib.UnresolvedOID)

		testutil.Equal(t, 0, len(unresolvedOids),
			"no OID should be unresolved in permissive mode")

		obj := m.Object("testObject")
		testutil.NotNil(t, obj, "testObject should resolve in permissive mode")
		// enterprises = 1.3.6.1.4.1, MIB = .99999, object = .1
		testutil.Equal(t, "1.3.6.1.4.1.99999.1", obj.OID().String(),
			"testObject OID should match net-snmp")
	})
}

// TestResolverStrictnessBoundaries verifies resolver fallback tier gating.
func TestResolverStrictnessBoundaries(t *testing.T) {
	tests := []struct {
		level           mib.ResolverStrictness
		wantConstrained bool
		wantGlobal      bool
	}{
		{mib.ResolverStrict, false, false},
		{mib.ResolverNormal, true, false},
		{mib.ResolverPermissive, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			testutil.Equal(t, tt.wantConstrained, tt.level.AllowConstrainedFallbacks(),
				"AllowConstrainedFallbacks at level %d", tt.level)
			testutil.Equal(t, tt.wantGlobal, tt.level.AllowGlobalFallbacks(),
				"AllowGlobalFallbacks at level %d", tt.level)
		})
	}
}

// TestImportForwardingTypeResolution verifies that a type imported via a
// forwarding chain resolves to the same base type as a direct import.
//
// Chain: PROBLEM-FORWARDING-MIB imports ForwardedType FROM
//
//	PROBLEM-FORWARDING-RELAY-MIB, which imports it FROM
//	PROBLEM-FORWARDING-SOURCE-MIB.
//
// tryImportForwarding (imports.go:167-208) checks the relay module's own
// import list and follows it to the source module.
func TestImportForwardingTypeResolution(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-FORWARDING-MIB", mib.ResolverNormal)

	obj := m.Object("problemForwardedTypeObject")
	testutil.NotNil(t, obj, "Object(problemForwardedTypeObject)")

	typ := obj.Type()
	testutil.NotNil(t, typ, "Type()")

	// ForwardedType is a TC based on DisplayString -> OCTET STRING
	testutil.Equal(t, mib.BaseOctetString, typ.EffectiveBase(),
		"ForwardedType should resolve to OCTET STRING via forwarding chain")
}

// TestImportForwardingOidResolution verifies that an OID parent imported via
// a forwarding chain produces the correct numeric OID.
//
// PROBLEM-FORWARDING-MIB defines problemForwardedOidObject under
// forwardedSourceRoot, which it imports from the relay module.
func TestImportForwardingOidResolution(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-FORWARDING-MIB", mib.ResolverNormal)

	obj := m.Object("problemForwardedOidObject")
	testutil.NotNil(t, obj, "Object(problemForwardedOidObject)")

	// forwardedSourceRoot = enterprises.99998.20.1
	// problemForwardedOidObject = forwardedSourceRoot.10
	testutil.Equal(t, "1.3.6.1.4.1.99998.20.1.10", obj.OID().String(),
		"OID should resolve through forwarded parent")
}

// TestImportForwardingAllLevels verifies that import forwarding works at
// all strictness levels. Forwarding follows re-export chains through
// intermediate modules, which is deterministic and not a guess.
func TestImportForwardingAllLevels(t *testing.T) {
	levels := []struct {
		name  string
		level mib.ResolverStrictness
	}{
		{"strict", mib.ResolverStrict},
		{"normal", mib.ResolverNormal},
		{"permissive", mib.ResolverPermissive},
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadAtStrictness(t, "PROBLEM-FORWARDING-MIB", lvl.level)

			unresolvedImports := unresolvedSymbols(m, "PROBLEM-FORWARDING-MIB", mib.UnresolvedImport)
			testutil.Equal(t, 0, len(unresolvedImports),
				"all imports should resolve via forwarding at %s", lvl.name)
		})
	}
}

// TestImportForwardingSourceModuleCorrectness verifies that forwarded
// symbols are attributed to the correct source module (the one that
// actually defines them, not the relay).
func TestImportForwardingSourceModuleCorrectness(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-FORWARDING-MIB", mib.ResolverNormal)

	srcObj := m.Object("forwardedSourceObject")
	testutil.NotNil(t, srcObj, "Object(forwardedSourceObject)")

	testutil.NotNil(t, srcObj.Module(), "Module()")

	testutil.Equal(t, "PROBLEM-FORWARDING-SOURCE-MIB", srcObj.Module().Name(),
		"source object should be attributed to the source module")
}

// TestImportForwardingRelayOwnObjects verifies that the relay module's own
// objects still resolve correctly alongside forwarded imports.
func TestImportForwardingRelayOwnObjects(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-FORWARDING-RELAY-MIB", mib.ResolverNormal)

	obj := m.Object("relayOwnObject")
	testutil.NotNil(t, obj, "relay module's own object should resolve")
	if obj == nil {
		return
	}
	testutil.Equal(t, mib.BaseInteger32, obj.Type().EffectiveBase(),
		"relay own object should have Integer32 type")
}

// TestPartialResolution verifies that partial import resolution works when
// a module exports some but not all requested symbols.
// This is tested implicitly via PROBLEM-IMPORTS-MIB which imports some
// valid symbols alongside missing ones. Here we verify the valid imports
// resolve while invalid ones are tracked as unresolved.
func TestPartialResolution(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", mib.ResolverStrict)

	unresolvedImports := unresolvedSymbols(m, "PROBLEM-IMPORTS-MIB", mib.UnresolvedImport)
	testutil.False(t, unresolvedImports["Integer32"],
		"Integer32 should resolve (directly imported from SNMPv2-SMI)")
	testutil.False(t, unresolvedImports["enterprises"],
		"enterprises should resolve (directly imported from SNMPv2-SMI)")
}

// TestStrictModeIndexResolution verifies that INDEX entries resolve at strict
// level for valid imports that go through re-exporting modules.
//
// RADLAN-MIB imports dot1dBasePort from BRIDGE-MIB. BRIDGE-MIB defines
// dot1dBasePort but also re-exports MacAddress from SNMPv2-TC. When all
// symbols from BRIDGE-MIB are checked together, the combined set fails
// findCandidateWithAllSymbols because MacAddress is not a direct definition
// in BRIDGE-MIB. Import forwarding and partial resolution handle this
// correctly and should work at all strictness levels.
func TestStrictModeIndexResolution(t *testing.T) {
	levels := []struct {
		name  string
		level mib.ResolverStrictness
	}{
		{"strict", mib.ResolverStrict},
		{"normal", mib.ResolverNormal},
		{"permissive", mib.ResolverPermissive},
	}

	entries := []string{
		"rlPortGvrpTimersEntry",
		"rlStormCtrlEntry",
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadAtStrictness(t, "RADLAN-MIB", lvl.level)

			for _, entryName := range entries {
				t.Run(entryName, func(t *testing.T) {
					obj := m.Object(entryName)
					testutil.NotNil(t, obj, "object %s should exist", entryName)
					if obj == nil {
						return
					}
					idx := obj.Index()
					testutil.True(t, len(idx) > 0,
						"INDEX should not be empty for %s at %s", entryName, lvl.name)
					if len(idx) > 0 {
						testutil.Equal(t, "dot1dBasePort", idx[0].Object.Name(),
							"first INDEX object should be dot1dBasePort")
					}
				})
			}
		})
	}
}
