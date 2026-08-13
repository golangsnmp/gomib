package resolver

import (
	"math"
	"testing"

	"github.com/golangsnmp/gomib/internal/cst/lower"
	cstparser "github.com/golangsnmp/gomib/internal/cst/parser"
	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestResolvedRangeSemanticsMixedRawParentAlternatives(t *testing.T) {
	source := []byte(`
MIXED-RANGE-MIB DEFINITIONS ::= BEGIN
IMPORTS Integer32 FROM SNMPv2-SMI;

RawFirstParent ::= Integer32 (VENDOR-MIN..10 | 20..30)
RawFirstChild ::= RawFirstParent (MIN..25)
RawLastParent ::= Integer32 (20..30 | VENDOR-MIN..10)
RawLastChild ::= RawLastParent (MIN..25)
MaxRawFirstParent ::= Integer32 (10..VENDOR-MAX | 20..30)
MaxRawFirstChild ::= MaxRawFirstParent (25..MAX)
MaxRawLastParent ::= Integer32 (20..30 | 10..VENDOR-MAX)
MaxRawLastChild ::= MaxRawLastParent (25..MAX)
GlobalMaxOnlyChild ::= RawFirstParent (MAX..MAX)
GlobalMaxTo25Child ::= RawFirstParent (MAX..25)
MaxWitnessFirstParent ::= Integer32 (10..VENDOR-MAX | 20..30)
MaxWitnessFirstEmpty ::= MaxWitnessFirstParent (MAX..25)
MaxWitnessFirstAdjacent ::= MaxWitnessFirstParent (MAX..30)
MaxWitnessLastParent ::= Integer32 (20..30 | 10..VENDOR-MAX)
MaxWitnessLastEmpty ::= MaxWitnessLastParent (MAX..25)
MaxWitnessLastAdjacent ::= MaxWitnessLastParent (MAX..30)
MinWitnessFirstParent ::= Integer32 (VENDOR-MIN..30 | 20..40)
MinWitnessFirstEmpty ::= MinWitnessFirstParent (25..MIN)
MinWitnessFirstAdjacent ::= MinWitnessFirstParent (20..MIN)
MinWitnessLastParent ::= Integer32 (20..40 | VENDOR-MIN..30)
MinWitnessLastEmpty ::= MinWitnessLastParent (25..MIN)
MinWitnessLastAdjacent ::= MinWitnessLastParent (20..MIN)
RawChildParent ::= Integer32 (10..20)
RawChildDisjoint ::= RawChildParent (VENDOR-CHILD-MIN..5)
END
`)
	cfg := types.VerboseConfig()
	parser := cstparser.New(source, nil, cfg)
	mods := lower.Lower(parser.ParseModule(), source, parser.Diagnostics(), cfg)
	m := resolveWithBase(mods, nil, &cfg)
	mod := m.Module("MIXED-RANGE-MIB")

	for _, tt := range []struct {
		name          string
		concreteIndex int
		rawIndex      int
		rawMin        model.RangeBound
		rawMax        model.RangeBound
		concreteMin   model.RangeBound
		concreteMax   model.RangeBound
	}{
		{
			name: "RawFirstChild", concreteIndex: 1, rawIndex: 0,
			rawMin: model.NewRawRangeBound("VENDOR-MIN"), rawMax: model.NewUnsignedRangeBound(10),
			concreteMin: model.NewUnsignedRangeBound(20), concreteMax: model.NewUnsignedRangeBound(25),
		},
		{
			name: "RawLastChild", concreteIndex: 0, rawIndex: 1,
			rawMin: model.NewRawRangeBound("VENDOR-MIN"), rawMax: model.NewUnsignedRangeBound(10),
			concreteMin: model.NewUnsignedRangeBound(20), concreteMax: model.NewUnsignedRangeBound(25),
		},
		{
			name: "MaxRawFirstChild", concreteIndex: 1, rawIndex: 0,
			rawMin: model.NewUnsignedRangeBound(25), rawMax: model.NewRawRangeBound("VENDOR-MAX"),
			concreteMin: model.NewUnsignedRangeBound(25), concreteMax: model.NewUnsignedRangeBound(30),
		},
		{
			name: "MaxRawLastChild", concreteIndex: 0, rawIndex: 1,
			rawMin: model.NewUnsignedRangeBound(25), rawMax: model.NewRawRangeBound("VENDOR-MAX"),
			concreteMin: model.NewUnsignedRangeBound(25), concreteMax: model.NewUnsignedRangeBound(30),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ranges := mod.Type(tt.name).EffectiveRanges()
			testutil.Len(t, ranges, 2, "all parent alternatives")
			testutil.Equal(t, tt.rawMin, ranges[tt.rawIndex].Min, "raw branch minimum")
			testutil.Equal(t, tt.rawMax, ranges[tt.rawIndex].Max, "raw branch maximum")
			testutil.Equal(t, tt.concreteMin, ranges[tt.concreteIndex].Min, "concrete branch minimum")
			testutil.Equal(t, tt.concreteMax, ranges[tt.concreteIndex].Max, "concrete branch maximum")
		})
	}

	globalMax := mod.Type("GlobalMaxOnlyChild").EffectiveRanges()
	testutil.Len(t, globalMax, 1, "only the possible global-maximum branch remains")
	testutil.Equal(t, model.NewUnsignedRangeBound(30), globalMax[0].Min, "global maximum minimum")
	testutil.Equal(t, model.NewUnsignedRangeBound(30), globalMax[0].Max, "global maximum maximum")

	testutil.Len(t, mod.Type("GlobalMaxTo25Child").EffectiveRanges(), 0, "MAX..25 is empty")
	testutil.True(t, mod.Type("GlobalMaxTo25Child").EffectiveRangesConstrained(), "MAX..25 remains constrained")
	for _, name := range []string{
		"MaxWitnessFirstEmpty", "MaxWitnessLastEmpty",
		"MinWitnessFirstEmpty", "MinWitnessLastEmpty",
	} {
		testutil.Len(t, mod.Type(name).EffectiveRanges(), 0, name+" is empty")
		testutil.True(t, mod.Type(name).EffectiveRangesConstrained(), name+" remains constrained")
	}
	for _, name := range []string{
		"MaxWitnessFirstAdjacent", "MaxWitnessLastAdjacent",
		"MinWitnessFirstAdjacent", "MinWitnessLastAdjacent",
	} {
		testutil.True(t, len(mod.Type(name).EffectiveRanges()) > 0, name+" retains adjacent boundary")
	}
	testutil.Len(t, mod.Type("RawChildDisjoint").EffectiveRanges(), 0, "raw child endpoint cannot hide disjoint known endpoints")
	testutil.True(t, mod.Type("RawChildDisjoint").EffectiveRangesConstrained(), "raw child remains constrained")
	testutil.Equal(t, 6, countDiagnostics(m.Diagnostics(), types.DiagConstraintEmptyIntersection), "empty-intersection diagnostics")
}

func TestResolvedRangeSemanticsEndToEnd(t *testing.T) {
	source := []byte(`
RANGE-TEST-MIB DEFINITIONS ::= BEGIN
IMPORTS
    OBJECT-TYPE, Integer32, Counter64
        FROM SNMPv2-SMI;

ParentRange ::= Integer32 (-10..20)
ChildRange ::= ParentRange (MIN..MAX)
DisjointRange ::= ParentRange (30..40)
DisjointDescendant ::= DisjointRange (35..36)
HugeRange ::= Counter64 (0..18446744073709551615)

childObject OBJECT-TYPE
    SYNTAX ChildRange (MIN..10)
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "child"
    ::= { 1 3 6 1 4 1 99997 1 }

disjointObject OBJECT-TYPE
    SYNTAX ParentRange (30..40)
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "disjoint"
    DEFVAL { 5 }
    ::= { 1 3 6 1 4 1 99997 2 }
END
`)
	cfg := types.VerboseConfig()
	parser := cstparser.New(source, nil, cfg)
	mods := lower.Lower(parser.ParseModule(), source, parser.Diagnostics(), cfg)
	m := resolveWithBase(mods, nil, &cfg)
	mod := m.Module("RANGE-TEST-MIB")

	child := mod.Type("ChildRange")
	testutil.Equal(t, model.RangeBoundMin, child.Ranges()[0].Min.Kind, "MIN preserved")
	testutil.Equal(t, model.RangeBoundMax, child.Ranges()[0].Max.Kind, "MAX preserved")
	testutil.Equal(t, model.NewSignedRangeBound(-10), child.EffectiveRanges()[0].Min, "inherited min")
	testutil.Equal(t, model.NewUnsignedRangeBound(20), child.EffectiveRanges()[0].Max, "inherited max")

	hugeType := mod.Type("HugeRange")
	huge := hugeType.Ranges()[0]
	testutil.Equal(t, model.NewUnsignedRangeBound(math.MaxUint64), huge.Max, "Counter64 maximum preserved")
	testutil.Equal(t, model.NewUnsignedRangeBound(math.MaxUint64), hugeType.EffectiveRanges()[0].Max, "Counter64 effective maximum preserved")

	disjoint := mod.Type("DisjointRange")
	testutil.Len(t, disjoint.EffectiveRanges(), 0, "disjoint type range")
	testutil.True(t, disjoint.EffectiveRangesConstrained(), "empty type intersection is constrained")
	descendant := mod.Type("DisjointDescendant")
	testutil.Len(t, descendant.EffectiveRanges(), 0, "disjoint descendant range")
	testutil.True(t, descendant.EffectiveRangesConstrained(), "empty descendant intersection is constrained")

	obj := mod.Object("childObject")
	testutil.Equal(t, model.NewSignedRangeBound(-10), obj.EffectiveRanges()[0].Min, "object inherited min")
	testutil.Equal(t, model.NewUnsignedRangeBound(10), obj.EffectiveRanges()[0].Max, "object narrowed max")

	disjointObj := mod.Object("disjointObject")
	testutil.Len(t, disjointObj.EffectiveRanges(), 0, "disjoint object range")
	testutil.True(t, disjointObj.EffectiveRangesConstrained(), "empty object intersection is constrained")
	hasDiag(t, m.Diagnostics(), types.DiagDefvalRange)
	hasDiag(t, m.Diagnostics(), types.DiagMinMaxRange)
	testutil.Equal(t, 3, countDiagnostics(m.Diagnostics(), types.DiagConstraintEmptyIntersection), "empty-intersection diagnostics")

	silent := types.SilentConfig()
	parser = cstparser.New(source, nil, silent)
	mods = lower.Lower(parser.ParseModule(), source, parser.Diagnostics(), silent)
	silentMib := resolveWithBase(mods, nil, &silent)
	noDiag(t, silentMib.Diagnostics(), types.DiagMinMaxRange, types.DiagConstraintEmptyIntersection)
}
