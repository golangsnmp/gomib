package module

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

const constraintTestSource = `
CONSTRAINT-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Integer32, Unsigned32, enterprises
        FROM SNMPv2-SMI;

constraintTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

sizeRangeObj OBJECT-TYPE
    SYNTAX      OCTET STRING (SIZE (0..255))
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { constraintTest 1 }

sizeSingleObj OBJECT-TYPE
    SYNTAX      OCTET STRING (SIZE (6))
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { constraintTest 2 }

valueRangeObj OBJECT-TYPE
    SYNTAX      Integer32 (0..65535)
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { constraintTest 3 }

minMaxObj OBJECT-TYPE
    SYNTAX      Integer32 (MIN..MAX)
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { constraintTest 4 }

multiRangeObj OBJECT-TYPE
    SYNTAX      Integer32 (1..10 | 20..30)
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { constraintTest 5 }

negativeRangeObj OBJECT-TYPE
    SYNTAX      Integer32 (-100..100)
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { constraintTest 6 }

END
`

func TestLower_Constraints(t *testing.T) {
	mod := lowerModule(t, constraintTestSource)

	t.Run("size-range", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "sizeRangeObj")
		constrained, ok := obj.Syntax.(*TypeSyntaxConstrained)
		testutil.True(t, ok, "expected *TypeSyntaxConstrained, got %T", obj.Syntax)

		cs, ok := constrained.Constraint.(*ConstraintSize)
		testutil.True(t, ok, "expected *ConstraintSize, got %T", constrained.Constraint)
		testutil.Len(t, cs.Ranges, 1, "expected 1 range")

		r := cs.Ranges[0]
		minVal, ok := r.Min.(*RangeValueUnsigned)
		testutil.True(t, ok, "expected *RangeValueUnsigned for Min, got %T", r.Min)
		testutil.Equal(t, uint64(0), minVal.Value, "Min value")

		testutil.NotNil(t, r.Max, "Max should not be nil for a range")
		maxVal, ok := r.Max.(*RangeValueUnsigned)
		testutil.True(t, ok, "expected *RangeValueUnsigned for Max, got %T", r.Max)
		testutil.Equal(t, uint64(255), maxVal.Value, "Max value")
	})

	t.Run("size-single-value", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "sizeSingleObj")
		constrained, ok := obj.Syntax.(*TypeSyntaxConstrained)
		testutil.True(t, ok, "expected *TypeSyntaxConstrained, got %T", obj.Syntax)

		cs, ok := constrained.Constraint.(*ConstraintSize)
		testutil.True(t, ok, "expected *ConstraintSize, got %T", constrained.Constraint)
		testutil.Len(t, cs.Ranges, 1, "expected 1 range")

		r := cs.Ranges[0]
		minVal, ok := r.Min.(*RangeValueUnsigned)
		testutil.True(t, ok, "expected *RangeValueUnsigned for Min, got %T", r.Min)
		testutil.Equal(t, uint64(6), minVal.Value, "Min value")

		// Single-value constraint: Max should be nil.
		testutil.Nil(t, r.Max, "Max should be nil for single-value constraint")
	})

	t.Run("value-range", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "valueRangeObj")
		constrained, ok := obj.Syntax.(*TypeSyntaxConstrained)
		testutil.True(t, ok, "expected *TypeSyntaxConstrained, got %T", obj.Syntax)

		cr, ok := constrained.Constraint.(*ConstraintRange)
		testutil.True(t, ok, "expected *ConstraintRange, got %T", constrained.Constraint)
		testutil.Len(t, cr.Ranges, 1, "expected 1 range")

		r := cr.Ranges[0]
		// Non-negative integers may be parsed as signed or unsigned.
		// Verify the numeric values regardless of signedness.
		assertRangeValueEquals(t, r.Min, 0, "Min")
		testutil.NotNil(t, r.Max, "Max should not be nil")
		assertRangeValueEquals(t, r.Max, 65535, "Max")
	})

	t.Run("min-max", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "minMaxObj")
		constrained, ok := obj.Syntax.(*TypeSyntaxConstrained)
		testutil.True(t, ok, "expected *TypeSyntaxConstrained, got %T", obj.Syntax)

		cr, ok := constrained.Constraint.(*ConstraintRange)
		testutil.True(t, ok, "expected *ConstraintRange, got %T", constrained.Constraint)
		testutil.Len(t, cr.Ranges, 1, "expected 1 range")

		r := cr.Ranges[0]
		_, ok = r.Min.(*RangeValueMin)
		testutil.True(t, ok, "expected *RangeValueMin for Min, got %T", r.Min)

		testutil.NotNil(t, r.Max, "Max should not be nil")
		_, ok = r.Max.(*RangeValueMax)
		testutil.True(t, ok, "expected *RangeValueMax for Max, got %T", r.Max)
	})

	t.Run("multiple-ranges", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "multiRangeObj")
		constrained, ok := obj.Syntax.(*TypeSyntaxConstrained)
		testutil.True(t, ok, "expected *TypeSyntaxConstrained, got %T", obj.Syntax)

		cr, ok := constrained.Constraint.(*ConstraintRange)
		testutil.True(t, ok, "expected *ConstraintRange, got %T", constrained.Constraint)
		testutil.Len(t, cr.Ranges, 2, "expected 2 ranges")

		// First range: 1..10
		r0 := cr.Ranges[0]
		assertRangeValueEquals(t, r0.Min, 1, "range[0].Min")
		testutil.NotNil(t, r0.Max, "range[0].Max should not be nil")
		assertRangeValueEquals(t, r0.Max, 10, "range[0].Max")

		// Second range: 20..30
		r1 := cr.Ranges[1]
		assertRangeValueEquals(t, r1.Min, 20, "range[1].Min")
		testutil.NotNil(t, r1.Max, "range[1].Max should not be nil")
		assertRangeValueEquals(t, r1.Max, 30, "range[1].Max")
	})

	t.Run("negative-range", func(t *testing.T) {
		obj := findDef[*ObjectType](t, mod, "negativeRangeObj")
		constrained, ok := obj.Syntax.(*TypeSyntaxConstrained)
		testutil.True(t, ok, "expected *TypeSyntaxConstrained, got %T", obj.Syntax)

		cr, ok := constrained.Constraint.(*ConstraintRange)
		testutil.True(t, ok, "expected *ConstraintRange, got %T", constrained.Constraint)
		testutil.Len(t, cr.Ranges, 1, "expected 1 range")

		r := cr.Ranges[0]
		// Negative value must be preserved as a signed integer.
		minVal, ok := r.Min.(*RangeValueSigned)
		testutil.True(t, ok, "expected *RangeValueSigned for negative Min, got %T", r.Min)
		testutil.Equal(t, int64(-100), minVal.Value, "Min value")

		testutil.NotNil(t, r.Max, "Max should not be nil")
		assertRangeValueEquals(t, r.Max, 100, "Max")
	})
}

// assertRangeValueEquals checks that the given RangeValue holds the expected
// numeric value. It accepts both *RangeValueSigned and *RangeValueUnsigned
// since the parser may produce either for non-negative integers.
func assertRangeValueEquals(t *testing.T, rv RangeValue, want int64, label string) {
	t.Helper()
	switch v := rv.(type) {
	case *RangeValueSigned:
		testutil.Equal(t, want, v.Value, "%s (signed)", label)
	case *RangeValueUnsigned:
		testutil.Equal(t, uint64(want), v.Value, "%s (unsigned)", label)
	default:
		t.Fatalf("%s: expected *RangeValueSigned or *RangeValueUnsigned, got %T", label, rv)
	}
}
