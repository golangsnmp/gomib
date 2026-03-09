package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func TestStrictForwardedAugmentsAndIndexResolution(t *testing.T) {
	m := loadAtStrictness(t, "PROBLEM-FORWARDING-MIB", mib.ResolverStrict)

	augmentRow := m.Object("problemForwardedAugmentEntry")
	testutil.NotNil(t, augmentRow, "augmented row should resolve in strict mode")
	if augmentRow == nil {
		return
	}

	augments := augmentRow.Augments()
	testutil.NotNil(t, augments, "AUGMENTS target should resolve through forwarding")
	if augments != nil {
		testutil.Equal(t, "forwardedBaseEntry", augments.Name(),
			"AUGMENTS target name")
		testutil.Equal(t, "PROBLEM-FORWARDING-SOURCE-MIB", augments.Module().Name(),
			"AUGMENTS target should be attributed to source module")
	}

	indexRow := m.Object("problemForwardedIndexEntry")
	testutil.NotNil(t, indexRow, "indexed row should resolve in strict mode")
	if indexRow == nil {
		return
	}

	indexes := indexRow.Index()
	testutil.Len(t, indexes, 1, "forwarded INDEX entries")
	if len(indexes) != 1 {
		return
	}

	idxObj := indexes[0].Object
	testutil.NotNil(t, idxObj, "forwarded index object should resolve")
	if idxObj == nil {
		return
	}

	testutil.Equal(t, "forwardedBaseIndex", idxObj.Name(), "forwarded index name")
	testutil.Equal(t, "PROBLEM-FORWARDING-SOURCE-MIB", idxObj.Module().Name(),
		"forwarded index should be attributed to source module")
	testutil.Equal(t, mib.BaseOctetString, idxObj.Type().EffectiveBase(),
		"forwarded MacAddress index base type")
	testutil.Equal(t, "1x:", idxObj.EffectiveDisplayHint(),
		"forwarded MacAddress display hint")

	sizes := idxObj.EffectiveSizes()
	testutil.Len(t, sizes, 1, "forwarded MacAddress size constraints")
	if len(sizes) == 1 {
		testutil.Equal(t, int64(6), sizes[0].Min, "MacAddress size min")
		testutil.Equal(t, int64(6), sizes[0].Max, "MacAddress size max")
	}
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
