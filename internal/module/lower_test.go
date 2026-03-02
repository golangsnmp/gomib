package module

import (
	"strings"
	"testing"

	"github.com/golangsnmp/gomib/internal/parser"
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

func TestLower_NegativeBitsPosition(t *testing.T) {
	// A BITS type with a negative position value should produce a diagnostic.
	// Negative positions are invalid per RFC 2578.
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
    DESCRIPTION "Test object with invalid negative BITS position"
    ::= { bitsTest 1 }

END
`)

	p := parser.New(source, nil, types.PermissiveConfig())
	mods := p.ParseModule()
	testutil.Greater(t, len(mods), 0, "parse returned no modules")

	mod := Lower(mods[0], source, nil, types.PermissiveConfig())
	testutil.NotNil(t, mod, "lower returned nil")

	// Should have a diagnostic for the negative BITS position
	var found bool
	for _, d := range mod.Diagnostics {
		if strings.Contains(d.Message, "BITS") || strings.Contains(d.Message, "position") || strings.Contains(d.Message, "negative") {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected diagnostic for negative BITS position, got none")
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
