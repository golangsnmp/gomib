package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// resolve_strictness_types_test.go covers strictness behavior for type lookup,
// type forwarding, alias resolution, and imported metadata.

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

	levels := []struct {
		strictnessCase
		wantResolved bool
	}{
		{strictnessCase: strictnessCase{name: "strict", level: mib.ResolverStrict}},
		{strictnessCase: strictnessCase{name: "normal", level: mib.ResolverNormal}, wantResolved: true},
		{strictnessCase: strictnessCase{name: "permissive", level: mib.ResolverPermissive}, wantResolved: true},
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", lvl.level)
			unresolved := unresolvedSymbols(m, "PROBLEM-IMPORTS-MIB", mib.UnresolvedType)

			for _, tt := range smiTypes {
				t.Run(tt.object, func(t *testing.T) {
					obj := requireObject(t, m, tt.object)
					if !lvl.wantResolved {
						testutil.Nil(t, obj.Type(), "type should remain unresolved at %s", lvl.name)
						testutil.True(t, unresolved[tt.wantBase.String()],
							"type %s should be unresolved at %s", tt.wantBase, lvl.name)
						return
					}

					testutil.NotNil(t, obj.Type(), "type should resolve at %s", lvl.name)
					if obj.Type() != nil {
						testutil.Equal(t, tt.wantBase, obj.Type().EffectiveBase(),
							"base type for %s at %s", tt.object, lvl.name)
					}
					testutil.False(t, unresolved[tt.wantBase.String()],
						"type %s should not be unresolved at %s", tt.wantBase, lvl.name)
				})
			}
		})
	}
}

func TestTCFallbackStrictness(t *testing.T) {
	tcObjects := []struct {
		name     string
		wantType string
	}{
		{"problemMissingDisplayString", "OCTET STRING"},
		{"problemMissingTruthValue", "Integer32"},
	}

	levels := []struct {
		strictnessCase
		wantResolved bool
	}{
		{strictnessCase: strictnessCase{name: "strict", level: mib.ResolverStrict}},
		{strictnessCase: strictnessCase{name: "normal", level: mib.ResolverNormal}, wantResolved: true},
		{strictnessCase: strictnessCase{name: "permissive", level: mib.ResolverPermissive}, wantResolved: true},
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", lvl.level)

			for _, tc := range tcObjects {
				t.Run(tc.name, func(t *testing.T) {
					obj := requireObject(t, m, tc.name)
					if !lvl.wantResolved {
						testutil.Nil(t, obj.Type(), "TC type should be nil at %s", lvl.name)
						return
					}

					testutil.NotNil(t, obj.Type(), "TC type should resolve at %s", lvl.name)
					if obj.Type() == nil {
						return
					}
					gotType := normalizeType(obj.Type())
					testutil.Equal(t, tc.wantType, gotType,
						"TC base type at %s", lvl.name)
				})
			}
		})
	}
}

func TestRealCorpusTypeFallbackStrictness(t *testing.T) {
	levels := []struct {
		strictnessCase
		wantResolved bool
	}{
		{strictnessCase: strictnessCase{name: "strict", level: mib.ResolverStrict}},
		{strictnessCase: strictnessCase{name: "normal", level: mib.ResolverNormal}, wantResolved: true},
		{strictnessCase: strictnessCase{name: "permissive", level: mib.ResolverPermissive}, wantResolved: true},
	}

	tests := []struct {
		object   string
		symbol   string
		wantBase mib.BaseType
	}{
		{object: "hwpingCtlTargetPort", symbol: "Integer32", wantBase: mib.BaseInteger32},
		{object: "hwpingResultsRttNumDisconnects", symbol: "Gauge32", wantBase: mib.BaseGauge32},
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadAtStrictness(t, "HUAWEI-DISMAN-PING-MIB", lvl.level)
			unresolved := unresolvedSymbols(m, "HUAWEI-DISMAN-PING-MIB", mib.UnresolvedType)

			for _, tt := range tests {
				t.Run(tt.object, func(t *testing.T) {
					obj := requireObject(t, m, tt.object)
					if !lvl.wantResolved {
						testutil.Nil(t, obj.Type(), "type should remain unresolved at %s", lvl.name)
						testutil.True(t, unresolved[tt.symbol],
							"type %s should be unresolved at %s", tt.symbol, lvl.name)
						return
					}

					testutil.NotNil(t, obj.Type(), "type should resolve at %s", lvl.name)
					if obj.Type() != nil {
						testutil.Equal(t, tt.wantBase, obj.Type().EffectiveBase(),
							"base type for %s at %s", tt.object, lvl.name)
					}
					testutil.False(t, unresolved[tt.symbol],
						"type %s should not be unresolved at %s", tt.symbol, lvl.name)
				})
			}
		})
	}
}

func TestModuleAliasStrictness(t *testing.T) {
	levels := []struct {
		strictnessCase
		ok bool
	}{
		{strictnessCase: strictnessCase{name: "strict", level: mib.ResolverStrict}},
		{strictnessCase: strictnessCase{name: "normal", level: mib.ResolverNormal}, ok: true},
		{strictnessCase: strictnessCase{name: "permissive", level: mib.ResolverPermissive}, ok: true},
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

func TestImportForwardingTypeResolution(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-FORWARDING-MIB", mib.ResolverNormal)

	obj := requireObject(t, m, "problemForwardedTypeObject")
	typ := obj.Type()
	testutil.NotNil(t, typ, "Type()")

	testutil.Equal(t, mib.BaseOctetString, typ.EffectiveBase(),
		"ForwardedType should resolve to OCTET STRING via forwarding chain")
}

func TestImportForwardingAllLevels(t *testing.T) {
	for _, lvl := range allStrictnessCases {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadAtStrictness(t, "PROBLEM-FORWARDING-MIB", lvl.level)

			unresolvedImports := unresolvedSymbols(m, "PROBLEM-FORWARDING-MIB", mib.UnresolvedImport)
			testutil.Equal(t, 0, len(unresolvedImports),
				"all imports should resolve via forwarding at %s", lvl.name)
		})
	}
}

func TestImportForwardingSourceModuleCorrectness(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-FORWARDING-MIB", mib.ResolverNormal)

	srcObj := requireObject(t, m, "forwardedSourceObject")
	testutil.NotNil(t, srcObj.Module(), "Module()")

	testutil.Equal(t, "PROBLEM-FORWARDING-SOURCE-MIB", srcObj.Module().Name(),
		"source object should be attributed to the source module")
}

func TestImportForwardingRelayOwnObjects(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-FORWARDING-RELAY-MIB", mib.ResolverNormal)

	obj := requireObject(t, m, "relayOwnObject")
	testutil.Equal(t, mib.BaseInteger32, obj.Type().EffectiveBase(),
		"relay own object should have Integer32 type")
}

func TestPartialResolution(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-IMPORTS-MIB", mib.ResolverStrict)

	unresolvedImports := unresolvedSymbols(m, "PROBLEM-IMPORTS-MIB", mib.UnresolvedImport)
	testutil.False(t, unresolvedImports["Integer32"],
		"Integer32 should resolve (directly imported from SNMPv2-SMI)")
	testutil.False(t, unresolvedImports["enterprises"],
		"enterprises should resolve (directly imported from SNMPv2-SMI)")
}

func TestStrictPartialImportResolutionPerSymbol(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-PARTIAL-IMPORTS-MIB", mib.ResolverStrict)

	unresolved := unresolvedSymbols(m, "PROBLEM-PARTIAL-IMPORTS-MIB", mib.UnresolvedImport)
	testutil.True(t, unresolved["MissingType"], "missing type import should remain unresolved")
	testutil.True(t, unresolved["missingParent"], "missing object import should remain unresolved")
	testutil.False(t, unresolved["DisplayString"], "valid type import should resolve")
	testutil.False(t, unresolved["enterprises"], "valid object import should resolve")
	testutil.False(t, unresolved["Integer32"], "valid type import should resolve")

	str := m.Object("problemPartialString")
	testutil.NotNil(t, str, "object using valid imported type should resolve")
	if str != nil {
		testutil.Equal(t, mib.BaseOctetString, str.Type().EffectiveBase(),
			"DisplayString should resolve in strict mode")
	}

	num := m.Object("problemPartialInteger")
	testutil.NotNil(t, num, "object under valid imported parent should resolve")
	if num != nil {
		testutil.Equal(t, "1.3.6.1.4.1.99998.38.2", num.OID().String(),
			"OID should resolve through valid imported parent")
	}
}

func TestStrictImportedMetadataPreserved(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-STRICT-METADATA-MIB", mib.ResolverStrict)

	display := m.Object("problemStrictDisplayString")
	testutil.NotNil(t, display, "DisplayString object")
	if display != nil {
		testutil.Equal(t, mib.BaseOctetString, display.Type().EffectiveBase(),
			"DisplayString base type")
		testutil.Equal(t, "255a", display.EffectiveDisplayHint(),
			"DisplayString display hint")
		sizes := display.EffectiveSizes()
		testutil.Len(t, sizes, 1, "DisplayString sizes")
		if len(sizes) == 1 {
			testutil.Equal(t, int64(0), sizes[0].Min, "DisplayString size min")
			testutil.Equal(t, int64(255), sizes[0].Max, "DisplayString size max")
		}
	}

	rowStatus := m.Object("problemStrictRowStatus")
	testutil.NotNil(t, rowStatus, "RowStatus object")
	if rowStatus != nil {
		enums := rowStatus.EffectiveEnums()
		testutil.NotEmpty(t, enums, "RowStatus enums")
		active, ok := rowStatus.Enum("active")
		testutil.True(t, ok, "RowStatus active enum")
		if ok {
			testutil.Equal(t, int64(1), active.Value, "RowStatus active value")
		}
		createAndWait, ok := rowStatus.Enum("createAndWait")
		testutil.True(t, ok, "RowStatus createAndWait enum")
		if ok {
			testutil.Equal(t, int64(5), createAndWait.Value, "RowStatus createAndWait value")
		}
	}

	mac := m.Object("problemStrictMacAddress")
	testutil.NotNil(t, mac, "MacAddress object")
	if mac != nil {
		testutil.Equal(t, "1x:", mac.EffectiveDisplayHint(), "MacAddress display hint")
		sizes := mac.EffectiveSizes()
		testutil.Len(t, sizes, 1, "MacAddress sizes")
		if len(sizes) == 1 {
			testutil.Equal(t, int64(6), sizes[0].Min, "MacAddress size min")
			testutil.Equal(t, int64(6), sizes[0].Max, "MacAddress size max")
		}
	}

	bridgeID := m.Object("problemStrictBridgeId")
	testutil.NotNil(t, bridgeID, "BridgeId object")
	if bridgeID != nil {
		sizes := bridgeID.EffectiveSizes()
		testutil.Len(t, sizes, 1, "BridgeId sizes")
		if len(sizes) == 1 {
			testutil.Equal(t, int64(8), sizes[0].Min, "BridgeId size min")
			testutil.Equal(t, int64(8), sizes[0].Max, "BridgeId size max")
		}
	}

	timeout := m.Object("problemStrictTimeout")
	testutil.NotNil(t, timeout, "Timeout object")
	if timeout != nil {
		testutil.Equal(t, "d", timeout.EffectiveDisplayHint(), "Timeout display hint")
		ranges := timeout.EffectiveRanges()
		testutil.Len(t, ranges, 1, "Timeout ranges")
		if len(ranges) == 1 {
			testutil.Equal(t, int64(100), ranges[0].Min, "Timeout range min")
			testutil.Equal(t, int64(1000), ranges[0].Max, "Timeout range max")
		}
	}
}
