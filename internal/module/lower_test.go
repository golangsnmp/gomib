package module

import (
	"strings"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestLineColFromTable(t *testing.T) {
	source := []byte("line1\nline2\nline3\n")
	table := types.BuildLineTable(source)

	tests := []struct {
		name     string
		offset   types.ByteOffset
		wantLine int
		wantCol  int
	}{
		{"start of file", 0, 1, 1},
		{"middle of line 1", 3, 1, 4},
		{"end of line 1 (newline)", 5, 1, 6},
		{"start of line 2", 6, 2, 1},
		{"middle of line 2", 9, 2, 4},
		{"start of line 3", 12, 3, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, col := types.LineColFromTable(table, tt.offset)
			testutil.Equal(t, tt.wantLine, line, "LineColFromTable(table, %d) line", tt.offset)
			testutil.Equal(t, tt.wantCol, col, "LineColFromTable(table, %d) col", tt.offset)
		})
	}

	// Nil table returns (0, 0)
	line, col := types.LineColFromTable(nil, 5)
	testutil.Equal(t, 0, line, "LineColFromTable(nil, 5) line")
	testutil.Equal(t, 0, col, "LineColFromTable(nil, 5) col")
}

func TestLower_DiagnosticSourceLocation(t *testing.T) {
	// Underscore identifier triggers a parser diagnostic with a span.
	// After lowering, the diagnostic should preserve line/column from that span.
	source := []byte(`UNDERSCORE-TEST DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises
        FROM SNMPv2-SMI;

underscoreTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

test_object OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { underscoreTest 1 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.StrictConfig(), types.DiagIdentifierUnderscore)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagIdentifierUnderscore)
	// test_object is on line 16, column 1 of the source
	testutil.Equal(t, 16, d.Line, "identifier-underscore diagnostic line")
	testutil.Equal(t, 1, d.Column, "identifier-underscore diagnostic column")
}

func TestLower_DiagnosticSourceLocation_Synthetic(t *testing.T) {
	// Lowering-generated diagnostics should include source location
	// from the module span. An SMIv2 module without MODULE-IDENTITY
	// triggers a lowering diagnostic.
	source := []byte(`NO-IDENTITY-MIB DEFINITIONS ::= BEGIN

IMPORTS
    OBJECT-TYPE, Integer32
        FROM SNMPv2-SMI;

someObject OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { 1 3 6 1 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.PermissiveConfig(), types.DiagMissingModuleIdentity)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagMissingModuleIdentity)
	testutil.True(t, d.Line != 0, "expected non-zero line for lowering diagnostic, got %d", d.Line)
	testutil.True(t, d.Column != 0, "expected non-zero column for lowering diagnostic, got %d", d.Column)
}

func TestLower_SNMPv2MIBNotTreatedAsBaseModule(t *testing.T) {
	// SNMPv2-MIB is NOT a synthetic base module. A module named "SNMPv2-MIB"
	// without MODULE-IDENTITY should get the missing-module-identity diagnostic.
	// This tests that language detection and base module checks use
	// consistent definitions via BaseModuleFromName/IsBaseModule.
	source := []byte(`SNMPv2-MIB DEFINITIONS ::= BEGIN

IMPORTS
    OBJECT-TYPE, Integer32
        FROM SNMPv2-SMI;

sysDescr OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { 1 3 6 1 2 1 1 1 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.PermissiveConfig(), types.DiagMissingModuleIdentity)
	testutil.NotNil(t, d, "SNMPv2-MIB without MODULE-IDENTITY should get %s diagnostic", types.DiagMissingModuleIdentity)
}

func TestLower_BaseModuleSkipsModuleIdentityCheck(t *testing.T) {
	// Actual base modules (SNMPv2-SMI, SNMPv2-TC, etc.) are synthetic and
	// should NOT get the missing-module-identity check.
	source := []byte(`SNMPv2-SMI DEFINITIONS ::= BEGIN

IMPORTS
    ;

END
`)

	d := lowerAndFindDiagnostic(t, source, types.DefaultConfig(), types.DiagMissingModuleIdentity)
	testutil.Nil(t, d, "base module SNMPv2-SMI should not get %s diagnostic", types.DiagMissingModuleIdentity)
}

func TestLower_BitsNumberNegative(t *testing.T) {
	source := []byte(`BITS-TEST DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE
        FROM SNMPv2-SMI;

bitsTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { 1 3 6 1 4 1 99999 }

badBitsObject OBJECT-TYPE
    SYNTAX      BITS { goodBit(0), badBit(-1) }
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { bitsTest 1 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.PermissiveConfig(), types.DiagBitsNumberNegative)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagBitsNumberNegative)
	testutil.Equal(t, types.SeverityError, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "badBit"), "message should mention badBit: %s", d.Message)
}

func TestLower_BitsNumberTooLarge(t *testing.T) {
	source := []byte(`BITS-TEST DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE
        FROM SNMPv2-SMI;

bitsTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { 1 3 6 1 4 1 99999 }

hugeBitsObject OBJECT-TYPE
    SYNTAX      BITS { goodBit(0), hugeBit(524280) }
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { bitsTest 1 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.PermissiveConfig(), types.DiagBitsNumberTooLarge)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagBitsNumberTooLarge)
	testutil.Equal(t, types.SeverityError, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "hugeBit"), "message should mention hugeBit: %s", d.Message)
}

func TestLower_BitsNumberLarge(t *testing.T) {
	source := []byte(`BITS-TEST DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE
        FROM SNMPv2-SMI;

bitsTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { 1 3 6 1 4 1 99999 }

largeBitsObject OBJECT-TYPE
    SYNTAX      BITS { goodBit(0), largeBit(128) }
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { bitsTest 1 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.PermissiveConfig(), types.DiagBitsNumberLarge)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagBitsNumberLarge)
	testutil.Equal(t, types.SeverityStyle, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "largeBit"), "message should mention largeBit: %s", d.Message)
}

func TestLower_BitsNumberLarge_NotForSmall(t *testing.T) {
	// Positions under 128 should not trigger the large warning.
	source := []byte(`BITS-TEST DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE
        FROM SNMPv2-SMI;

bitsTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { 1 3 6 1 4 1 99999 }

okBitsObject OBJECT-TYPE
    SYNTAX      BITS { bit0(0), bit127(127) }
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { bitsTest 1 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.PermissiveConfig(), types.DiagBitsNumberLarge)
	testutil.Nil(t, d, "position 127 should not trigger %s", types.DiagBitsNumberLarge)
}

func TestLower_EnumZero_SMIv1(t *testing.T) {
	// Zero enum value in SMIv1 should produce a diagnostic.
	source := []byte(`ENUM-TEST DEFINITIONS ::= BEGIN

IMPORTS
    enterprises
        FROM RFC1155-SMI;

testObject OBJECT-TYPE
    SYNTAX      INTEGER { off(0), on(1) }
    ACCESS      read-only
    STATUS      mandatory
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.PermissiveConfig(), types.DiagEnumZero)
	testutil.NotNil(t, d, "expected %s diagnostic for SMIv1 enum with zero", types.DiagEnumZero)
	testutil.Equal(t, types.SeverityError, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "off"), "message should mention off: %s", d.Message)
}

func TestLower_EnumZero_SMIv2_NoDiag(t *testing.T) {
	// Zero enum value in SMIv2 is allowed.
	source := []byte(`ENUM-TEST DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises
        FROM SNMPv2-SMI;

enumTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testObject OBJECT-TYPE
    SYNTAX      INTEGER { off(0), on(1) }
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { enumTest 1 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.PermissiveConfig(), types.DiagEnumZero)
	testutil.Nil(t, d, "SMIv2 enum with zero should not trigger %s", types.DiagEnumZero)
}

func TestLower_MissingModuleIdentity_AlwaysWarning(t *testing.T) {
	source := []byte(`NO-IDENTITY-MIB DEFINITIONS ::= BEGIN

IMPORTS
    OBJECT-TYPE, Integer32
        FROM SNMPv2-SMI;

someObject OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { 1 3 6 1 }

END
`)

	// Warning-level diagnostics are filtered at default strictness (level 3),
	// so test with permissive (shows warnings) and strict (shows everything).
	for _, tc := range []struct {
		name   string
		config types.DiagnosticConfig
	}{
		{"permissive_config", types.PermissiveConfig()},
		{"strict_config", types.StrictConfig()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := lowerAndFindDiagnostic(t, source, tc.config, types.DiagMissingModuleIdentity)
			testutil.NotNil(t, d, "expected %s diagnostic", types.DiagMissingModuleIdentity)
			testutil.Equal(t, types.SeverityWarning, d.Severity, "expected SeverityWarning in %s", tc.name)
		})
	}
}

func TestLower_ModuleIdentityNotFirst(t *testing.T) {
	source := []byte(`NOT-FIRST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises
        FROM SNMPv2-SMI;

someObject OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { notFirstTest 1 }

notFirstTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.StrictConfig(), types.DiagModuleIdentityNotFirst)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagModuleIdentityNotFirst)
	testutil.Equal(t, types.SeverityWarning, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "NOT-FIRST-MIB"), "message should mention module name: %s", d.Message)
}

func TestLower_ModuleIdentityNotFirst_WhenFirst(t *testing.T) {
	// MODULE-IDENTITY is the first definition; no diagnostic expected.
	source := []byte(`FIRST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises
        FROM SNMPv2-SMI;

firstTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

someObject OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { firstTest 1 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.StrictConfig(), types.DiagModuleIdentityNotFirst)
	testutil.Nil(t, d, "should not get %s when MODULE-IDENTITY is first", types.DiagModuleIdentityNotFirst)
}

func TestLower_ModuleIdentityMultiple(t *testing.T) {
	source := []byte(`MULTI-MI-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, enterprises
        FROM SNMPv2-SMI;

first MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "First"
    REVISION "200001010000Z"
    DESCRIPTION "First"
    ::= { enterprises 99998 }

second MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Second"
    REVISION "200001010000Z"
    DESCRIPTION "Second"
    ::= { enterprises 99999 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.StrictConfig(), types.DiagModuleIdentityMultiple)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagModuleIdentityMultiple)
	testutil.Equal(t, types.SeverityError, d.Severity, "severity")
}

func TestLower_MacroNotImported(t *testing.T) {
	// Uses OBJECT-TYPE but doesn't import it.
	source := []byte(`MACRO-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, Integer32, enterprises
        FROM SNMPv2-SMI;

macroTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

someObj OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { macroTest 1 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.StrictConfig(), types.DiagMacroNotImported)
	testutil.NotNil(t, d, "expected %s diagnostic", types.DiagMacroNotImported)
	testutil.Equal(t, types.SeverityMinor, d.Severity, "severity")
	testutil.True(t, strings.Contains(d.Message, "OBJECT-TYPE"), "message should mention OBJECT-TYPE: %s", d.Message)
}

func TestLower_MacroNotImported_WhenImported(t *testing.T) {
	// OBJECT-TYPE is properly imported; no diagnostic expected.
	source := []byte(`MACRO-OK-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises
        FROM SNMPv2-SMI;

macroOk MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

someObj OBJECT-TYPE
    SYNTAX      Integer32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "Test"
    ::= { macroOk 1 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.StrictConfig(), types.DiagMacroNotImported)
	testutil.Nil(t, d, "should not get %s when macro is imported", types.DiagMacroNotImported)
}

func TestLower_MacroNotImported_SMIv1(t *testing.T) {
	// SMIv1 modules should NOT get macro-not-imported diagnostics.
	source := []byte(`SMIV1-MACRO-MIB DEFINITIONS ::= BEGIN

IMPORTS
    Counter
        FROM RFC1155-SMI;

someObj OBJECT-TYPE
    SYNTAX      Counter
    ACCESS      read-only
    STATUS      mandatory
    DESCRIPTION "Test"
    ::= { 1 3 6 1 }

END
`)

	d := lowerAndFindDiagnostic(t, source, types.StrictConfig(), types.DiagMacroNotImported)
	testutil.Nil(t, d, "SMIv1 module should not get %s", types.DiagMacroNotImported)
}
