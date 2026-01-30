package gomib

// resolve_strictness_test.go verifies that resolution behavior changes with
// strictness levels. The strictness system gates fallback resolution strategies:
//
//   - Safe fallbacks (level >= 3, Normal): module aliases, import forwarding
//   - Best-guess fallbacks (level >= 5, Permissive): global type lookup, SMI global OID roots
//
// These tests load synthetic MIBs at different strictness levels and verify
// that resolution outcomes differ. Expected values are grounded against
// net-snmp (which always resolves at maximum permissiveness) and libsmi
// (which fails on unknown module names like SNMPv2-SMI-v1).

import (
	"context"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// loadAtStrictness loads a MIB at the given strictness level using both the
// primary corpus (for dependencies) and the problems directory.
func loadAtStrictness(t testing.TB, name string, level mib.StrictnessLevel) mib.Mib {
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
	m, err := LoadModules(ctx, []string{name}, src, WithStrictness(level))
	if err != nil {
		t.Fatalf("LoadModules(%s, %s) failed: %v", name, level, err)
	}
	return m
}

// unresolvedSymbols returns the set of unresolved symbol names for a given
// module and kind (e.g., "type", "oid", "import").
func unresolvedSymbols(m mib.Mib, module, kind string) map[string]bool {
	result := make(map[string]bool)
	for _, u := range m.Unresolved() {
		if u.Module == module && u.Kind == kind {
			result[u.Symbol] = true
		}
	}
	return result
}

// --- Best-guess fallback: global SMI type lookup (level >= 5) ---

// TestTypeFallbackPermissiveOnly verifies that SMI global types (Counter64,
// Gauge32, etc.) resolve only in permissive mode when not explicitly imported.
//
// PROBLEM-IMPORTS-MIB imports enterprises and Integer32 from SNMPv2-SMI but
// deliberately omits Counter64, Gauge32, Unsigned32, and TimeTicks. These are
// SMI base types that net-snmp resolves implicitly at all levels.
//
// Ground truth:
//   - net-snmp: always resolves (global type lookup, no import required)
//   - libsmi: "Counter64 implicitly defined, not imported from SNMPv2-SMI"
//   - gomib strict/normal: types unresolved (best-guess fallback disabled)
//   - gomib permissive: types resolve via isSmiGlobalType() fallback
func TestTypeFallbackPermissiveOnly(t *testing.T) {
	// SMI base types that need AllowBestGuessFallbacks (level >= 5)
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
		m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", mib.StrictnessStrict)
		unresolved := unresolvedSymbols(m, "PROBLEM-IMPORTS-MIB", "type")

		for _, tt := range smiTypes {
			t.Run(tt.object, func(t *testing.T) {
				obj := m.FindObject(tt.object)
				testutil.NotNil(t, obj, "object should exist (OID resolves via imported enterprises)")
				if obj == nil {
					return
				}
				testutil.Nil(t, obj.Type(), "type should be nil in strict mode (no global type fallback)")
				testutil.True(t, unresolved[tt.wantBase.String()],
					"type %s should be in unresolved list", tt.wantBase)
			})
		}
	})

	t.Run("normal", func(t *testing.T) {
		m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", mib.StrictnessNormal)
		unresolved := unresolvedSymbols(m, "PROBLEM-IMPORTS-MIB", "type")

		for _, tt := range smiTypes {
			t.Run(tt.object, func(t *testing.T) {
				obj := m.FindObject(tt.object)
				testutil.NotNil(t, obj, "object should exist (OID resolves via imported enterprises)")
				if obj == nil {
					return
				}
				// Normal mode (level 3) does NOT enable best-guess fallbacks (level >= 5),
				// so global type lookup is disabled - same outcome as strict for types.
				testutil.Nil(t, obj.Type(), "type should be nil in normal mode (no global type fallback)")
				testutil.True(t, unresolved[tt.wantBase.String()],
					"type %s should be in unresolved list", tt.wantBase)
			})
		}
	})

	t.Run("permissive", func(t *testing.T) {
		m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", mib.StrictnessPermissive)
		unresolved := unresolvedSymbols(m, "PROBLEM-IMPORTS-MIB", "type")

		for _, tt := range smiTypes {
			t.Run(tt.object, func(t *testing.T) {
				obj := m.FindObject(tt.object)
				testutil.NotNil(t, obj, "object should exist")
				if obj == nil {
					return
				}
				// Permissive mode enables AllowBestGuessFallbacks, which triggers
				// isSmiGlobalType() in LookupTypeForModule - matches net-snmp behavior.
				testutil.NotNil(t, obj.Type(), "type should resolve in permissive mode via global type fallback")
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

// TestTCFallbackUnresolved verifies that textual convention types (DisplayString,
// TruthValue) from SNMPv2-TC remain unresolved at all levels when not imported.
// These are NOT SMI global types, so even the global type fallback doesn't cover them.
//
// Ground truth:
//   - net-snmp: resolves implicitly (broader global search than gomib)
//   - gomib: unresolved at all levels (global fallback only covers SMI base types)
//
// This documents a known divergence from net-snmp. The global type fallback
// intentionally limits scope to SMI base types to avoid false resolution.
func TestTCFallbackUnresolved(t *testing.T) {
	tcObjects := []string{
		"problemMissingDisplayString",
		"problemMissingTruthValue",
	}

	levels := []struct {
		name  string
		level mib.StrictnessLevel
	}{
		{"strict", mib.StrictnessStrict},
		{"normal", mib.StrictnessNormal},
		{"permissive", mib.StrictnessPermissive},
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", lvl.level)

			for _, objName := range tcObjects {
				t.Run(objName, func(t *testing.T) {
					obj := m.FindObject(objName)
					testutil.NotNil(t, obj, "object should exist (OID resolves)")
					if obj == nil {
						return
					}
					// TC types are not covered by the SMI global type fallback,
					// so they remain unresolved at all strictness levels.
					testutil.Nil(t, obj.Type(),
						"TC type should be nil (not in SMI global type set)")
				})
			}
		})
	}
}

// --- Safe fallback: module aliases (level >= 3) ---

// TestModuleAliasNormalAndAbove verifies that module alias resolution
// (SNMPv2-SMI-v1 -> SNMPv2-SMI, SNMPv2-TC-v1 -> SNMPv2-TC) is gated by
// AllowSafeFallbacks (level >= 3).
//
// PROBLEM-IMPORTS-ALIAS-MIB imports all symbols from SNMPv2-SMI-v1 and
// SNMPv2-TC-v1, which are old names used by real MIBs like RADLAN-MIB.
//
// Ground truth:
//   - net-snmp: resolves aliases silently (has its own internal alias table)
//   - libsmi: "failed to locate module `SNMPv2-SMI-v1'" (no alias support)
//   - gomib strict: matches libsmi (alias table disabled)
//   - gomib normal/permissive: matches net-snmp (alias table enabled)
func TestModuleAliasNormalAndAbove(t *testing.T) {
	t.Run("strict", func(t *testing.T) {
		m := loadAtStrictness(t, "PROBLEM-IMPORTS-ALIAS-MIB", mib.StrictnessStrict)

		// Strict mode: AllowSafeFallbacks = false.
		// Module aliases are disabled, so imports from SNMPv2-SMI-v1 and
		// SNMPv2-TC-v1 fail. This cascades: enterprises is unresolved,
		// so the entire OID chain fails, and objects are not created.
		unresolvedImports := unresolvedSymbols(m, "PROBLEM-IMPORTS-ALIAS-MIB", "import")
		testutil.True(t, unresolvedImports["enterprises"],
			"enterprises should be unresolved (aliased module not found)")
		testutil.True(t, unresolvedImports["Integer32"],
			"Integer32 should be unresolved (aliased module not found)")
		testutil.True(t, unresolvedImports["DisplayString"],
			"DisplayString should be unresolved (aliased module not found)")

		// OID chain fails because enterprises is not in scope
		unresolvedOids := unresolvedSymbols(m, "PROBLEM-IMPORTS-ALIAS-MIB", "oid")
		testutil.True(t, unresolvedOids["enterprises"],
			"enterprises OID should be unresolved")

		// Objects are not resolvable
		testutil.Nil(t, m.FindObject("problemAliasString"),
			"problemAliasString should not resolve in strict mode")
		testutil.Nil(t, m.FindObject("problemAliasInteger"),
			"problemAliasInteger should not resolve in strict mode")
	})

	t.Run("normal", func(t *testing.T) {
		m := loadAtStrictness(t, "PROBLEM-IMPORTS-ALIAS-MIB", mib.StrictnessNormal)

		// Normal mode: AllowSafeFallbacks = true.
		// Module alias table maps SNMPv2-SMI-v1 -> SNMPv2-SMI and
		// SNMPv2-TC-v1 -> SNMPv2-TC. All imports resolve.
		unresolvedImports := unresolvedSymbols(m, "PROBLEM-IMPORTS-ALIAS-MIB", "import")
		testutil.Equal(t, 0, len(unresolvedImports),
			"no imports should be unresolved with alias fallback")

		// Objects resolve with correct types (matches net-snmp)
		str := m.FindObject("problemAliasString")
		testutil.NotNil(t, str, "problemAliasString should resolve in normal mode")
		if str != nil {
			testutil.NotNil(t, str.Type(), "type should resolve")
			if str.Type() != nil {
				testutil.Equal(t, mib.BaseOctetString, str.Type().EffectiveBase(),
					"DisplayString base type should be OCTET STRING")
			}
		}

		intObj := m.FindObject("problemAliasInteger")
		testutil.NotNil(t, intObj, "problemAliasInteger should resolve in normal mode")
		if intObj != nil {
			testutil.NotNil(t, intObj.Type(), "type should resolve")
			if intObj.Type() != nil {
				testutil.Equal(t, mib.BaseInteger32, intObj.Type().EffectiveBase(),
					"Integer32 base type should be Integer32")
			}
		}
	})

	t.Run("permissive", func(t *testing.T) {
		m := loadAtStrictness(t, "PROBLEM-IMPORTS-ALIAS-MIB", mib.StrictnessPermissive)

		// Permissive mode should behave the same as normal for module aliases
		// (safe fallbacks are available at both levels).
		unresolvedImports := unresolvedSymbols(m, "PROBLEM-IMPORTS-ALIAS-MIB", "import")
		testutil.Equal(t, 0, len(unresolvedImports),
			"no imports should be unresolved with alias fallback")

		str := m.FindObject("problemAliasString")
		testutil.NotNil(t, str, "problemAliasString should resolve in permissive mode")
		if str != nil && str.Type() != nil {
			testutil.Equal(t, mib.BaseOctetString, str.Type().EffectiveBase(),
				"DisplayString base type should be OCTET STRING")
		}

		intObj := m.FindObject("problemAliasInteger")
		testutil.NotNil(t, intObj, "problemAliasInteger should resolve in permissive mode")
		if intObj != nil && intObj.Type() != nil {
			testutil.Equal(t, mib.BaseInteger32, intObj.Type().EffectiveBase(),
				"Integer32 base type should be Integer32")
		}
	})
}

// --- Best-guess fallback: SMI global OID roots (level >= 5) ---

// TestOIDGlobalRootPermissiveOnly verifies that OID definitions referencing
// "enterprises" without importing it only resolve in permissive mode.
// This is tested by MISSING-IMPORT-TEST-MIB in the strictness/violations corpus.
//
// Ground truth:
//   - net-snmp: resolves enterprises globally (implicit root knowledge)
//   - libsmi: depends on import; fails without it
//   - gomib strict/normal: OID chain fails (enterprises not in scope)
//   - gomib permissive: resolves via lookupSmiGlobalOidRoot()
//
// Note: load_test.go already tests this scenario; this test adds explicit
// OID value verification and unresolved ref checking.
func TestOIDGlobalRootPermissiveOnly(t *testing.T) {
	corpus, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree corpus failed: %v", err)
	}
	violations, err := DirTree("testdata/strictness/violations")
	if err != nil {
		t.Fatalf("DirTree violations failed: %v", err)
	}
	src := Multi(corpus, violations)

	load := func(t *testing.T, level mib.StrictnessLevel) mib.Mib {
		t.Helper()
		ctx := context.Background()
		m, err := LoadModules(ctx, []string{"MISSING-IMPORT-TEST-MIB"}, src, WithStrictness(level))
		if err != nil {
			t.Fatalf("LoadModules failed: %v", err)
		}
		return m
	}

	t.Run("strict", func(t *testing.T) {
		m := load(t, mib.StrictnessStrict)
		unresolvedOids := unresolvedSymbols(m, "MISSING-IMPORT-TEST-MIB", "oid")

		testutil.True(t, unresolvedOids["enterprises"],
			"enterprises OID should be unresolved in strict mode")
		testutil.Nil(t, m.FindObject("testObject"),
			"testObject should not resolve (OID chain broken)")
	})

	t.Run("normal", func(t *testing.T) {
		m := load(t, mib.StrictnessNormal)
		unresolvedOids := unresolvedSymbols(m, "MISSING-IMPORT-TEST-MIB", "oid")

		// Normal mode has safe fallbacks but NOT best-guess fallbacks.
		// Global OID root lookup requires best-guess (level >= 5).
		testutil.True(t, len(unresolvedOids) > 0,
			"should have unresolved OIDs in normal mode")
		testutil.Nil(t, m.FindObject("testObject"),
			"testObject should not resolve in normal mode")
	})

	t.Run("permissive", func(t *testing.T) {
		m := load(t, mib.StrictnessPermissive)
		unresolvedOids := unresolvedSymbols(m, "MISSING-IMPORT-TEST-MIB", "oid")

		testutil.Equal(t, 0, len(unresolvedOids),
			"no OID should be unresolved in permissive mode")

		obj := m.FindObject("testObject")
		testutil.NotNil(t, obj, "testObject should resolve in permissive mode")
		if obj != nil {
			// enterprises = 1.3.6.1.4.1, MIB = .99999, object = .1
			testutil.Equal(t, "1.3.6.1.4.1.99999.1", obj.OID().String(),
				"testObject OID should match net-snmp")
		}
	})
}

// --- Behavioral boundary verification ---

// TestStrictnessLevelBoundaries verifies the exact boundary conditions of
// the two guard functions: AllowSafeFallbacks (level >= 3) and
// AllowBestGuessFallbacks (level >= 5).
func TestStrictnessLevelBoundaries(t *testing.T) {
	// Test the guard functions directly via DiagnosticConfig
	tests := []struct {
		level     mib.StrictnessLevel
		wantSafe  bool
		wantGuess bool
	}{
		{0, false, false}, // Strict
		{1, false, false},
		{2, false, false},
		{3, true, false}, // Normal
		{4, true, false},
		{5, true, true}, // Permissive
		{6, true, true}, // Silent
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			cfg := mib.DiagnosticConfig{Level: tt.level}
			testutil.Equal(t, tt.wantSafe, cfg.AllowSafeFallbacks(),
				"AllowSafeFallbacks at level %d", tt.level)
			testutil.Equal(t, tt.wantGuess, cfg.AllowBestGuessFallbacks(),
				"AllowBestGuessFallbacks at level %d", tt.level)
		})
	}
}
