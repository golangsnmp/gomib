package module

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestLower_TypeSyntax(t *testing.T) {
	t.Run("TypeRef", func(t *testing.T) {
		src := `TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, enterprises
        FROM SNMPv2-SMI
    DisplayString
        FROM SNMPv2-TC;

testMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testObj OBJECT-TYPE
    SYNTAX      DisplayString
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { testMIB 1 }

END
`
		mod := lowerModule(t, src)
		obj := findDef[*ObjectType](t, mod, "testObj")
		ref, ok := obj.Syntax.(*TypeSyntaxTypeRef)
		testutil.True(t, ok, "expected *TypeSyntaxTypeRef, got %T", obj.Syntax)
		testutil.Equal(t, "DisplayString", ref.Name, "type ref name")
	})

	t.Run("IntegerEnum", func(t *testing.T) {
		src := `TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises
        FROM SNMPv2-SMI;

testMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testObj OBJECT-TYPE
    SYNTAX      INTEGER { up(1), down(2), testing(3) }
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { testMIB 1 }

END
`
		mod := lowerModule(t, src)
		obj := findDef[*ObjectType](t, mod, "testObj")
		ie, ok := obj.Syntax.(*TypeSyntaxIntegerEnum)
		testutil.True(t, ok, "expected *TypeSyntaxIntegerEnum, got %T", obj.Syntax)
		testutil.Equal(t, "", ie.Base, "bare INTEGER should have empty Base")
		testutil.Len(t, ie.NamedNumbers, 3, "named number count")

		testutil.Equal(t, "up", ie.NamedNumbers[0].Name, "first named number name")
		testutil.Equal(t, int64(1), ie.NamedNumbers[0].Value, "first named number value")

		testutil.Equal(t, "down", ie.NamedNumbers[1].Name, "second named number name")
		testutil.Equal(t, int64(2), ie.NamedNumbers[1].Value, "second named number value")

		testutil.Equal(t, "testing", ie.NamedNumbers[2].Name, "third named number name")
		testutil.Equal(t, int64(3), ie.NamedNumbers[2].Value, "third named number value")
	})

	t.Run("IntegerEnum_BareINTEGER_EmptyBase", func(t *testing.T) {
		// Type assignment with bare INTEGER enum should have empty Base.
		src := `TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, enterprises
        FROM SNMPv2-SMI;

testMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

MyEnumType ::= INTEGER { val1(1), val2(2) }

END
`
		mod := lowerModule(t, src)
		td := findDef[*TypeDef](t, mod, "MyEnumType")
		ie, ok := td.Syntax.(*TypeSyntaxIntegerEnum)
		testutil.True(t, ok, "expected *TypeSyntaxIntegerEnum, got %T", td.Syntax)
		testutil.Equal(t, "", ie.Base, "bare INTEGER type assignment should have empty Base")
		testutil.Len(t, ie.NamedNumbers, 2, "named number count")
		testutil.Equal(t, "val1", ie.NamedNumbers[0].Name, "first named number name")
		testutil.Equal(t, int64(1), ie.NamedNumbers[0].Value, "first named number value")
		testutil.Equal(t, "val2", ie.NamedNumbers[1].Name, "second named number name")
		testutil.Equal(t, int64(2), ie.NamedNumbers[1].Value, "second named number value")
	})

	t.Run("Bits", func(t *testing.T) {
		src := `TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, enterprises
        FROM SNMPv2-SMI;

testMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testObj OBJECT-TYPE
    SYNTAX      BITS { flag1(0), flag2(1), flag3(31) }
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { testMIB 1 }

END
`
		mod := lowerModule(t, src)
		obj := findDef[*ObjectType](t, mod, "testObj")
		bits, ok := obj.Syntax.(*TypeSyntaxBits)
		testutil.True(t, ok, "expected *TypeSyntaxBits, got %T", obj.Syntax)
		testutil.Len(t, bits.NamedBits, 3, "named bit count")

		testutil.Equal(t, "flag1", bits.NamedBits[0].Name, "first named bit name")
		testutil.Equal(t, uint32(0), bits.NamedBits[0].Position, "first named bit position")

		testutil.Equal(t, "flag2", bits.NamedBits[1].Name, "second named bit name")
		testutil.Equal(t, uint32(1), bits.NamedBits[1].Position, "second named bit position")

		testutil.Equal(t, "flag3", bits.NamedBits[2].Name, "third named bit name")
		testutil.Equal(t, uint32(31), bits.NamedBits[2].Position, "third named bit position (uint32)")
	})

	t.Run("Constrained", func(t *testing.T) {
		src := `TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, enterprises
        FROM SNMPv2-SMI;

testMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testObj OBJECT-TYPE
    SYNTAX      OCTET STRING (SIZE (0..255))
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { testMIB 1 }

END
`
		mod := lowerModule(t, src)
		obj := findDef[*ObjectType](t, mod, "testObj")
		constrained, ok := obj.Syntax.(*TypeSyntaxConstrained)
		testutil.True(t, ok, "expected *TypeSyntaxConstrained, got %T", obj.Syntax)

		// Base should be OCTET STRING
		_, baseOk := constrained.Base.(*TypeSyntaxOctetString)
		testutil.True(t, baseOk, "expected base *TypeSyntaxOctetString, got %T", constrained.Base)

		// Constraint should be SIZE
		sizeConstraint, sizeOk := constrained.Constraint.(*ConstraintSize)
		testutil.True(t, sizeOk, "expected *ConstraintSize, got %T", constrained.Constraint)
		testutil.Len(t, sizeConstraint.Ranges, 1, "range count")
	})

	t.Run("SequenceOf", func(t *testing.T) {
		src := `TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, enterprises
        FROM SNMPv2-SMI;

testMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testTable OBJECT-TYPE
    SYNTAX      SEQUENCE OF TestEntry
    MAX-ACCESS  not-accessible
    STATUS      current
    DESCRIPTION "Test"
    ::= { testMIB 1 }

END
`
		mod := lowerModule(t, src)
		obj := findDef[*ObjectType](t, mod, "testTable")
		seqOf, ok := obj.Syntax.(*TypeSyntaxSequenceOf)
		testutil.True(t, ok, "expected *TypeSyntaxSequenceOf, got %T", obj.Syntax)
		testutil.Equal(t, "TestEntry", seqOf.EntryType, "entry type name")
	})

	t.Run("Sequence", func(t *testing.T) {
		src := `TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, Integer32, enterprises
        FROM SNMPv2-SMI
    DisplayString
        FROM SNMPv2-TC;

testMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

TestEntry ::= SEQUENCE {
    col1 Integer32,
    col2 DisplayString
}

END
`
		mod := lowerModule(t, src)
		td := findDef[*TypeDef](t, mod, "TestEntry")
		seq, ok := td.Syntax.(*TypeSyntaxSequence)
		testutil.True(t, ok, "expected *TypeSyntaxSequence, got %T", td.Syntax)
		testutil.Len(t, seq.Fields, 2, "field count")

		// First field
		testutil.Equal(t, "col1", seq.Fields[0].Name, "first field name")
		ref0, ok0 := seq.Fields[0].Syntax.(*TypeSyntaxTypeRef)
		testutil.True(t, ok0, "expected field 0 syntax *TypeSyntaxTypeRef, got %T", seq.Fields[0].Syntax)
		testutil.Equal(t, "INTEGER", ref0.Name, "first field type name")

		// Second field
		testutil.Equal(t, "col2", seq.Fields[1].Name, "second field name")
		ref1, ok1 := seq.Fields[1].Syntax.(*TypeSyntaxTypeRef)
		testutil.True(t, ok1, "expected field 1 syntax *TypeSyntaxTypeRef, got %T", seq.Fields[1].Syntax)
		testutil.Equal(t, "DisplayString", ref1.Name, "second field type name")
	})

	t.Run("OctetString", func(t *testing.T) {
		src := `TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, enterprises
        FROM SNMPv2-SMI;

testMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testObj OBJECT-TYPE
    SYNTAX      OCTET STRING
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { testMIB 1 }

END
`
		mod := lowerModule(t, src)
		obj := findDef[*ObjectType](t, mod, "testObj")
		_, ok := obj.Syntax.(*TypeSyntaxOctetString)
		testutil.True(t, ok, "expected *TypeSyntaxOctetString, got %T", obj.Syntax)
	})

	t.Run("ObjectIdentifier", func(t *testing.T) {
		src := `TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, enterprises
        FROM SNMPv2-SMI;

testMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testObj OBJECT-TYPE
    SYNTAX      OBJECT IDENTIFIER
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { testMIB 1 }

END
`
		mod := lowerModule(t, src)
		obj := findDef[*ObjectType](t, mod, "testObj")
		_, ok := obj.Syntax.(*TypeSyntaxObjectIdentifier)
		testutil.True(t, ok, "expected *TypeSyntaxObjectIdentifier, got %T", obj.Syntax)
	})
}
