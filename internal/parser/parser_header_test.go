package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestParseMalformedHeaderMissingDefinitions(t *testing.T) {
	// Module name present but DEFINITIONS keyword missing
	source := []byte(`TEST-MIB ::= BEGIN END`)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	testutil.Equal(t, "UNKNOWN", module.Name.Name, "module name should be UNKNOWN on header failure")
	testutil.Greater(t, len(module.Diagnostics), 0, "should have diagnostics for missing DEFINITIONS")
	testutil.Equal(t, types.DiagParseError, module.Diagnostics[0].Code, "diagnostic code")
}

func TestParseMalformedHeaderMissingColonColonEqual(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS BEGIN END`)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	testutil.Equal(t, "UNKNOWN", module.Name.Name, "module name should be UNKNOWN on header failure")
	testutil.Greater(t, len(module.Diagnostics), 0, "should have diagnostics for missing ::=")
	testutil.Equal(t, types.DiagParseError, module.Diagnostics[0].Code, "diagnostic code")
}

func TestParseMalformedHeaderMissingBegin(t *testing.T) {
	source := []byte(`TEST-MIB DEFINITIONS ::= END`)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	testutil.Equal(t, "UNKNOWN", module.Name.Name, "module name should be UNKNOWN on header failure")
	testutil.Greater(t, len(module.Diagnostics), 0, "should have diagnostics for missing BEGIN")
	testutil.Equal(t, types.DiagParseError, module.Diagnostics[0].Code, "diagnostic code")
}

func TestParseMalformedHeaderEmpty(t *testing.T) {
	source := []byte(``)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	testutil.Equal(t, "UNKNOWN", module.Name.Name, "empty input should produce UNKNOWN module")
	testutil.Greater(t, len(module.Diagnostics), 0, "should have diagnostics for empty input")
	testutil.Equal(t, types.DiagParseError, module.Diagnostics[0].Code, "diagnostic code")
}

func TestParseMalformedHeaderGarbageOnly(t *testing.T) {
	source := []byte(`{ } ; 42`)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	testutil.Equal(t, "UNKNOWN", module.Name.Name, "garbage input should produce UNKNOWN module")
	testutil.Greater(t, len(module.Diagnostics), 0, "should have diagnostics for garbage input")
	testutil.Equal(t, types.DiagParseError, module.Diagnostics[0].Code, "diagnostic code")
}

func TestParseMalformedHeaderBadDefinitionsKeyword(t *testing.T) {
	source := []byte(`TEST-MIB DECALRATIONS ::= BEGIN END`)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	// "DECALRATIONS" is an uppercase ident but not DEFINITIONS
	testutil.Equal(t, "UNKNOWN", module.Name.Name, "bad keyword should produce UNKNOWN module")
	testutil.Greater(t, len(module.Diagnostics), 0, "should have diagnostics for bad keyword")
	testutil.Equal(t, types.DiagParseError, module.Diagnostics[0].Code, "diagnostic code")
}

func TestParseMalformedHeaderModuleNameOnly(t *testing.T) {
	source := []byte(`TEST-MIB`)
	p := New(source, nil, types.PermissiveConfig())
	module := p.ParseModule()

	testutil.Equal(t, "UNKNOWN", module.Name.Name, "module name only should produce UNKNOWN module")
	testutil.Greater(t, len(module.Diagnostics), 0, "should have diagnostics")
	testutil.Equal(t, types.DiagParseError, module.Diagnostics[0].Code, "diagnostic code")
}
