package model

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func signedRange(min, max int64) Range {
	return Range{Min: NewSignedRangeBound(min), Max: NewSignedRangeBound(max)}
}

func rawMinRange(raw string, max int64) Range {
	return Range{Min: NewRawRangeBound(raw), Max: NewSignedRangeBound(max)}
}

func rawMaxRange(min int64, raw string) Range {
	return Range{Min: NewSignedRangeBound(min), Max: NewRawRangeBound(raw)}
}

func symbolicRange(min, max RangeBoundKind) Range {
	return Range{Min: RangeBound{Kind: min}, Max: RangeBound{Kind: max}}
}

func TestIntersectRanges(t *testing.T) {
	rawMin := rawMinRange("vendor-min", 10)
	rawMax := rawMaxRange(10, "vendor-max")
	knownUnion := []Range{signedRange(0, 10), signedRange(20, 30)}
	tests := []struct {
		name   string
		parent []Range
		child  []Range
		want   []Range
	}{
		{
			name:   "raw child disjoint from concrete parent",
			parent: []Range{signedRange(10, 20)},
			child:  []Range{{Min: NewRawRangeBound("child-min"), Max: NewSignedRangeBound(5)}},
		},
		{
			name:   "raw child only retains uncertain alternative",
			parent: []Range{signedRange(0, 3), signedRange(10, 20)},
			child:  []Range{{Min: NewRawRangeBound("child-min"), Max: NewSignedRangeBound(5)}},
			want:   []Range{{Min: NewRawRangeBound("child-min"), Max: NewSignedRangeBound(3)}},
		},
		{
			name:   "raw child maximum cannot hide disjoint lower bound",
			parent: []Range{signedRange(0, 10)},
			child:  []Range{{Min: NewSignedRangeBound(20), Max: NewRawRangeBound("child-max")}},
		},
		{
			name:   "raw parent minimum cannot hide disjoint child",
			parent: []Range{rawMin},
			child:  []Range{signedRange(20, 30)},
		},
		{
			name:   "raw parent maximum cannot hide disjoint child",
			parent: []Range{rawMaxRange(20, "vendor-max")},
			child:  []Range{signedRange(0, 10)},
		},
		{
			name:   "MAX MAX removes non-maximum raw branch",
			parent: []Range{rawMin, signedRange(20, 30)},
			child:  []Range{symbolicRange(RangeBoundMax, RangeBoundMax)},
			want:   []Range{signedRange(30, 30)},
		},
		{
			name:   "MAX concrete is empty below global maximum",
			parent: []Range{rawMin, signedRange(20, 30)},
			child:  []Range{{Min: RangeBound{Kind: RangeBoundMax}, Max: NewSignedRangeBound(25)}},
		},
		{
			name:   "MIN concrete narrows each possible branch",
			parent: []Range{rawMin, signedRange(20, 30)},
			child:  []Range{{Min: RangeBound{Kind: RangeBoundMin}, Max: NewSignedRangeBound(25)}},
			want:   []Range{rawMin, signedRange(20, 25)},
		},
		{
			name:   "concrete MAX removes raw-min branch",
			parent: []Range{rawMin, signedRange(20, 30)},
			child:  []Range{{Min: NewSignedRangeBound(25), Max: RangeBound{Kind: RangeBoundMax}}},
			want:   []Range{signedRange(25, 30)},
		},
		{
			name:   "concrete MAX narrows opposite raw endpoint",
			parent: []Range{rawMax, signedRange(20, 30)},
			child:  []Range{{Min: NewSignedRangeBound(25), Max: RangeBound{Kind: RangeBoundMax}}},
			want:   []Range{rawMaxRange(25, "vendor-max"), signedRange(25, 30)},
		},
		{
			name:   "MIN MIN selects global minimum",
			parent: knownUnion,
			child:  []Range{symbolicRange(RangeBoundMin, RangeBoundMin)},
			want:   []Range{signedRange(0, 0)},
		},
		{
			name:   "MAX MAX selects global maximum",
			parent: knownUnion,
			child:  []Range{symbolicRange(RangeBoundMax, RangeBoundMax)},
			want:   []Range{signedRange(30, 30)},
		},
		{
			name:   "MIN concrete",
			parent: knownUnion,
			child:  []Range{{Min: RangeBound{Kind: RangeBoundMin}, Max: NewSignedRangeBound(25)}},
			want:   []Range{signedRange(0, 10), signedRange(20, 25)},
		},
		{
			name:   "concrete MAX",
			parent: knownUnion,
			child:  []Range{{Min: NewSignedRangeBound(25), Max: RangeBound{Kind: RangeBoundMax}}},
			want:   []Range{signedRange(25, 30)},
		},
		{
			name:   "MAX concrete empty",
			parent: knownUnion,
			child:  []Range{{Min: RangeBound{Kind: RangeBoundMax}, Max: NewSignedRangeBound(25)}},
		},
		{
			name:   "concrete MIN empty",
			parent: knownUnion,
			child:  []Range{{Min: NewSignedRangeBound(25), Max: RangeBound{Kind: RangeBoundMin}}},
		},
		{
			name:   "MAX MIN empty",
			parent: knownUnion,
			child:  []Range{symbolicRange(RangeBoundMax, RangeBoundMin)},
		},
		{
			name:   "touching alternatives are retained",
			parent: []Range{signedRange(0, 10), signedRange(20, 30)},
			child:  []Range{signedRange(10, 20)},
			want:   []Range{signedRange(10, 10), signedRange(20, 20)},
		},
		{
			name:   "multiple children do not duplicate impossible raw parent",
			parent: []Range{rawMin},
			child:  []Range{signedRange(20, 30), signedRange(5, 8)},
			want:   []Range{{Min: NewRawRangeBound("vendor-min"), Max: NewSignedRangeBound(8)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntersectRanges(tt.child, tt.parent)
			testutil.SliceEqual(t, tt.want, got, "intersection")
		})
	}
}

func TestIntersectRangesUsesUnionWideExtremeWitnesses(t *testing.T) {
	tests := []struct {
		name     string
		parent   []Range
		disjoint Range
		adjacent Range
	}{
		{
			name:     "lower MAX with raw candidate maximum",
			parent:   []Range{rawMaxRange(10, "vendor-max"), signedRange(20, 30)},
			disjoint: Range{Min: RangeBound{Kind: RangeBoundMax}, Max: NewSignedRangeBound(25)},
			adjacent: Range{Min: RangeBound{Kind: RangeBoundMax}, Max: NewSignedRangeBound(30)},
		},
		{
			name:     "upper MIN with raw candidate minimum",
			parent:   []Range{rawMinRange("vendor-min", 30), signedRange(20, 40)},
			disjoint: Range{Min: NewSignedRangeBound(25), Max: RangeBound{Kind: RangeBoundMin}},
			adjacent: Range{Min: NewSignedRangeBound(20), Max: RangeBound{Kind: RangeBoundMin}},
		},
	}

	for _, tt := range tests {
		for _, reversed := range []bool{false, true} {
			name := "original order"
			parent := slices.Clone(tt.parent)
			if reversed {
				name = "swapped order"
				slices.Reverse(parent)
			}
			t.Run(tt.name+"/"+name, func(t *testing.T) {
				testutil.Len(t, IntersectRanges([]Range{tt.disjoint}, parent), 0, "provably disjoint")
				testutil.True(t, len(IntersectRanges([]Range{tt.adjacent}, parent)) > 0, "adjacent boundary retained")
			})
		}
	}
}

func TestIntersectRangesUsesFullWidthComparisons(t *testing.T) {
	tests := []struct {
		name   string
		parent Range
		child  Range
		want   Range
	}{
		{
			name:   "mixed signed unsigned",
			parent: Range{Min: NewSignedRangeBound(-10), Max: NewUnsignedRangeBound(20)},
			child:  Range{Min: NewUnsignedRangeBound(0), Max: NewUnsignedRangeBound(math.MaxUint64)},
			want:   Range{Min: NewUnsignedRangeBound(0), Max: NewUnsignedRangeBound(20)},
		},
		{
			name:   "above MaxInt64",
			parent: Range{Min: NewUnsignedRangeBound(math.MaxInt64), Max: NewUnsignedRangeBound(math.MaxUint64)},
			child:  Range{Min: NewUnsignedRangeBound(math.MaxInt64 + 1), Max: NewUnsignedRangeBound(math.MaxUint64)},
			want:   Range{Min: NewUnsignedRangeBound(math.MaxInt64 + 1), Max: NewUnsignedRangeBound(math.MaxUint64)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntersectRanges([]Range{tt.child}, []Range{tt.parent})
			testutil.SliceEqual(t, []Range{tt.want}, got, "intersection")
		})
	}
}

func TestIntersectRangesUnionOrderInvariant(t *testing.T) {
	parent := []Range{rawMinRange("vendor-min", 10), signedRange(20, 30), signedRange(40, 50)}
	child := []Range{
		{Min: RangeBound{Kind: RangeBoundMin}, Max: NewSignedRangeBound(25)},
		{Min: NewSignedRangeBound(45), Max: RangeBound{Kind: RangeBoundMax}},
		symbolicRange(RangeBoundMin, RangeBoundMin),
		symbolicRange(RangeBoundMax, RangeBoundMax),
		{Min: RangeBound{Kind: RangeBoundMax}, Max: NewSignedRangeBound(45)},
	}
	want := normalizedRanges(IntersectRanges(child, parent))

	reversedParent := slices.Clone(parent)
	slices.Reverse(reversedParent)
	reversedChild := slices.Clone(child)
	slices.Reverse(reversedChild)

	for _, inputs := range []struct {
		name   string
		parent []Range
		child  []Range
	}{
		{name: "parent reversed", parent: reversedParent, child: child},
		{name: "child reversed", parent: parent, child: reversedChild},
		{name: "both reversed", parent: reversedParent, child: reversedChild},
	} {
		t.Run(inputs.name, func(t *testing.T) {
			got := normalizedRanges(IntersectRanges(inputs.child, inputs.parent))
			testutil.SliceEqual(t, want, got, "semantic range multiset")
		})
	}
}

func normalizedRanges(ranges []Range) []string {
	result := make([]string, len(ranges))
	for i, r := range ranges {
		result[i] = fmt.Sprintf("%d:%s..%d:%s", r.Min.Kind, r.Min.String(), r.Max.Kind, r.Max.String())
	}
	slices.Sort(result)
	return result
}
