package model

import "math"

// BaseConstraint returns the inherent bounds used to interpret a declared
// constraint for the given SMI base type.
func BaseConstraint(base BaseType, size bool, span Span) (Range, bool) {
	if size {
		return Range{Min: NewUnsignedRangeBound(0), Max: NewUnsignedRangeBound(65535), Span: span}, true
	}
	var min, max RangeBound
	switch base {
	case BaseInteger32:
		min = NewSignedRangeBound(math.MinInt32)
		max = NewSignedRangeBound(math.MaxInt32)
	case BaseUnsigned32, BaseGauge32, BaseTimeTicks, BaseCounter32:
		min = NewUnsignedRangeBound(0)
		max = NewUnsignedRangeBound(math.MaxUint32)
	case BaseCounter64:
		min = NewUnsignedRangeBound(0)
		max = NewUnsignedRangeBound(math.MaxUint64)
	default:
		return Range{}, false
	}
	return Range{Min: min, Max: max, Span: span}, true
}

// IntersectRanges intersects child constraints with parent constraints. MIN
// and MAX in the child are relative to the entire parent union. The result is
// empty when the two constraints are disjoint.
func IntersectRanges(child, parent []Range) []Range {
	if len(child) == 0 {
		return nil
	}
	if len(parent) == 0 {
		return append([]Range(nil), child...)
	}

	globalMaxLowerBounds, globalMinUpperBounds := unionExtremeWitnesses(parent)
	result := make([]Range, 0, len(child)*len(parent))
	for _, c := range child {
		for i, p := range parent {
			childMin, possible := resolveChildRangeBound(c.Min, p, parent, i, true)
			if !possible {
				continue
			}
			childMax, possible := resolveChildRangeBound(c.Max, p, parent, i, false)
			if !possible {
				continue
			}

			// Check every lower candidate against every upper candidate. This can
			// prove the intersection empty even when one of the other endpoints is
			// unresolved.
			lowers := []RangeBound{p.Min, childMin}
			if c.Min.Kind == RangeBoundMax {
				lowers = append(lowers, globalMaxLowerBounds...)
			}
			uppers := []RangeBound{p.Max, childMax}
			if c.Max.Kind == RangeBoundMin {
				uppers = append(uppers, globalMinUpperBounds...)
			}
			if anyRangeBoundGreater(lowers, uppers) {
				continue
			}

			lower := maxRangeBound(childMin, p.Min)
			upper := minRangeBound(childMax, p.Max)
			result = append(result, Range{Min: lower, Max: upper, Span: c.Span})
		}
	}
	return result
}

// unionExtremeWitnesses returns known lower bounds for the union's global
// maximum and known upper bounds for its global minimum. Either concrete
// endpoint is a valid witness because each parent alternative must be valid.
func unionExtremeWitnesses(ranges []Range) (maxLower, minUpper []RangeBound) {
	for _, r := range ranges {
		if r.Min.IsConcrete() {
			maxLower = append(maxLower, r.Min)
			minUpper = append(minUpper, r.Min)
		}
		if r.Max.IsConcrete() {
			maxLower = append(maxLower, r.Max)
			minUpper = append(minUpper, r.Max)
		}
	}
	return maxLower, minUpper
}

// resolveChildRangeBound resolves a symbolic child endpoint for one parent
// alternative. A lower MIN and upper MAX do not constrain an alternative. A
// lower MAX or upper MIN only applies to alternatives which could contain the
// corresponding global extreme of the complete parent union.
func resolveChildRangeBound(bound RangeBound, alternative Range, parent []Range, index int, lower bool) (RangeBound, bool) {
	if lower {
		switch bound.Kind {
		case RangeBoundMin:
			return alternative.Min, true
		case RangeBoundMax:
			if !couldContainGlobalMax(parent, index) {
				return RangeBound{}, false
			}
			return alternative.Max, true
		}
	} else {
		switch bound.Kind {
		case RangeBoundMax:
			return alternative.Max, true
		case RangeBoundMin:
			if !couldContainGlobalMin(parent, index) {
				return RangeBound{}, false
			}
			return alternative.Min, true
		}
	}
	return bound, true
}

func couldContainGlobalMax(ranges []Range, index int) bool {
	candidate := ranges[index].Max
	for i, other := range ranges {
		if i == index {
			continue
		}
		if comparison, ok := other.Max.Compare(candidate); ok && comparison > 0 {
			return false
		}
		// Even if the other maximum is raw, a greater known minimum proves
		// that its maximum must also be greater for a valid range.
		if comparison, ok := other.Min.Compare(candidate); ok && comparison > 0 {
			return false
		}
	}
	return true
}

func couldContainGlobalMin(ranges []Range, index int) bool {
	candidate := ranges[index].Min
	for i, other := range ranges {
		if i == index {
			continue
		}
		if comparison, ok := other.Min.Compare(candidate); ok && comparison < 0 {
			return false
		}
		// Even if the other minimum is raw, a lesser known maximum proves
		// that its minimum must also be lesser for a valid range.
		if comparison, ok := other.Max.Compare(candidate); ok && comparison < 0 {
			return false
		}
	}
	return true
}

func anyRangeBoundGreater(lowers, uppers []RangeBound) bool {
	for _, lower := range lowers {
		for _, upper := range uppers {
			if comparison, ok := lower.Compare(upper); ok && comparison > 0 {
				return true
			}
		}
	}
	return false
}

func maxRangeBound(a, b RangeBound) RangeBound {
	if comparison, ok := a.Compare(b); ok {
		if comparison < 0 {
			return b
		}
		return a
	}
	return unresolvedRangeBound(a, b)
}

func minRangeBound(a, b RangeBound) RangeBound {
	if comparison, ok := a.Compare(b); ok {
		if comparison > 0 {
			return b
		}
		return a
	}
	return unresolvedRangeBound(a, b)
}

// unresolvedRangeBound preserves uncertainty when an exact min/max cannot be
// represented. If both operands are raw, the first (the child-side operand at
// call sites above) is selected consistently as a conservative bound.
func unresolvedRangeBound(a, b RangeBound) RangeBound {
	if a.Kind == RangeBoundRaw {
		return a
	}
	return b
}

// RangeContainsInt64 reports whether a signed value is within a range.
func RangeContainsInt64(r Range, value int64) bool {
	v := NewSignedRangeBound(value)
	lower, lok := r.Min.Compare(v)
	upper, uok := r.Max.Compare(v)
	return lok && uok && lower <= 0 && upper >= 0
}

// RangeContainsUint64 reports whether an unsigned value is within a range.
func RangeContainsUint64(r Range, value uint64) bool {
	v := NewUnsignedRangeBound(value)
	lower, lok := r.Min.Compare(v)
	upper, uok := r.Max.Compare(v)
	return lok && uok && lower <= 0 && upper >= 0
}
