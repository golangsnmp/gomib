package module

import (
	"strings"
	"testing"

	"github.com/golangsnmp/gomib/internal/parser"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestSpanToLineCol(t *testing.T) {
	source := []byte("line1\nline2\nline3\n")

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
			line, col := spanToLineCol(source, tt.offset)
			if line != tt.wantLine || col != tt.wantCol {
				t.Errorf("spanToLineCol(source, %d) = (%d, %d), want (%d, %d)",
					tt.offset, line, col, tt.wantLine, tt.wantCol)
			}
		})
	}

	// Nil source returns (0, 0)
	line, col := spanToLineCol(nil, 5)
	if line != 0 || col != 0 {
		t.Errorf("spanToLineCol(nil, 5) = (%d, %d), want (0, 0)", line, col)
	}

	// Out of range offset returns (0, 0)
	line, col = spanToLineCol(source, 100)
	if line != 0 || col != 0 {
		t.Errorf("spanToLineCol(source, 100) = (%d, %d), want (0, 0)", line, col)
	}
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

	p := parser.New(source, nil, types.StrictConfig())
	ast := p.ParseModule()
	if ast == nil {
		t.Fatal("parse returned nil")
	}

	mod := Lower(ast, source, nil, types.StrictConfig())
	if mod == nil {
		t.Fatal("lower returned nil")
	}

	var found bool
	for _, d := range mod.Diagnostics {
		if d.Code == types.DiagIdentifierUnderscore {
			found = true
			// test_object is on line 16, column 1 of the source
			if d.Line != 16 {
				t.Errorf("expected line 16 for identifier-underscore diagnostic, got %d", d.Line)
			}
			if d.Column != 1 {
				t.Errorf("expected column 1 for identifier-underscore diagnostic, got %d", d.Column)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected %s diagnostic", types.DiagIdentifierUnderscore)
	}
}

func TestLower_DiagnosticSourceLocation_Synthetic(t *testing.T) {
	// Lowering-generated diagnostics (no span) should keep line/column 0.
	// An SMIv2 module without MODULE-IDENTITY triggers a lowering diagnostic.
	// Use permissive config so the warning-level diagnostic is visible.
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

	p := parser.New(source, nil, types.PermissiveConfig())
	ast := p.ParseModule()
	if ast == nil {
		t.Fatal("parse returned nil")
	}

	mod := Lower(ast, source, nil, types.PermissiveConfig())
	if mod == nil {
		t.Fatal("lower returned nil")
	}

	var found bool
	for _, d := range mod.Diagnostics {
		if d.Code == types.DiagMissingModuleIdentity {
			found = true
			// Lowering diagnostics now include source location from the module span
			if d.Line == 0 || d.Column == 0 {
				t.Errorf("expected non-zero line/column for lowering diagnostic, got line=%d, column=%d", d.Line, d.Column)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected %s diagnostic", types.DiagMissingModuleIdentity)
	}
}

func TestLower_SNMPv2MIBNotTreatedAsBaseModule(t *testing.T) {
	// SNMPv2-MIB is NOT a synthetic base module. A module named "SNMPv2-MIB"
	// without MODULE-IDENTITY should get the missing-module-identity diagnostic.
	// This tests that language detection and base module checks use
	// consistent definitions via BaseModuleFromName/IsBaseModule.
	// Use permissive config so the warning-level diagnostic is visible.
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

	p := parser.New(source, nil, types.PermissiveConfig())
	ast := p.ParseModule()
	if ast == nil {
		t.Fatal("parse returned nil")
	}

	mod := Lower(ast, source, nil, types.PermissiveConfig())
	if mod == nil {
		t.Fatal("lower returned nil")
	}

	// SNMPv2-MIB without MODULE-IDENTITY should get a diagnostic
	var found bool
	for _, d := range mod.Diagnostics {
		if d.Code == types.DiagMissingModuleIdentity {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("SNMPv2-MIB without MODULE-IDENTITY should get %s diagnostic", types.DiagMissingModuleIdentity)
	}
}

func TestLower_BaseModuleSkipsModuleIdentityCheck(t *testing.T) {
	// Actual base modules (SNMPv2-SMI, SNMPv2-TC, etc.) are synthetic and
	// should NOT get the missing-module-identity check.
	source := []byte(`SNMPv2-SMI DEFINITIONS ::= BEGIN

IMPORTS
    ;

END
`)

	p := parser.New(source, nil, types.DefaultConfig())
	ast := p.ParseModule()
	if ast == nil {
		t.Fatal("parse returned nil")
	}

	mod := Lower(ast, source, nil, types.DefaultConfig())
	if mod == nil {
		t.Fatal("lower returned nil")
	}

	for _, d := range mod.Diagnostics {
		if d.Code == types.DiagMissingModuleIdentity {
			t.Errorf("base module SNMPv2-SMI should not get %s diagnostic", types.DiagMissingModuleIdentity)
		}
	}
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
	ast := p.ParseModule()
	if ast == nil {
		t.Fatal("parse returned nil")
	}

	mod := Lower(ast, source, nil, types.PermissiveConfig())
	if mod == nil {
		t.Fatal("lower returned nil")
	}

	// Should have a diagnostic for the negative BITS position
	var found bool
	for _, d := range mod.Diagnostics {
		if strings.Contains(d.Message, "BITS") || strings.Contains(d.Message, "position") || strings.Contains(d.Message, "negative") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected diagnostic for negative BITS position, got none")
	}
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
			p := parser.New(source, nil, tc.config)
			ast := p.ParseModule()
			if ast == nil {
				t.Fatal("parse returned nil")
			}

			mod := Lower(ast, source, nil, tc.config)
			if mod == nil {
				t.Fatal("lower returned nil")
			}

			var found bool
			for _, d := range mod.Diagnostics {
				if d.Code == types.DiagMissingModuleIdentity {
					if d.Severity != types.SeverityWarning {
						t.Errorf("expected SeverityWarning in %s, got %v", tc.name, d.Severity)
					}
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %s diagnostic", types.DiagMissingModuleIdentity)
			}
		})
	}
}
