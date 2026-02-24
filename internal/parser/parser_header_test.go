package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestParseMalformedHeaders(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"missing DEFINITIONS", `TEST-MIB ::= BEGIN END`},
		{"missing ::=", `TEST-MIB DEFINITIONS BEGIN END`},
		{"missing BEGIN", `TEST-MIB DEFINITIONS ::= END`},
		{"empty input", ``},
		{"garbage only", `{ } ; 42`},
		{"bad DEFINITIONS keyword", `TEST-MIB DECALRATIONS ::= BEGIN END`},
		{"module name only", `TEST-MIB`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			module := parseModule(tt.source)
			testutil.Equal(t, "UNKNOWN", module.Name.Name, "module name should be UNKNOWN on header failure")
			testutil.Greater(t, len(module.Diagnostics), 0, "should have diagnostics")
			testutil.Equal(t, types.DiagParseError, module.Diagnostics[0].Code, "diagnostic code")
		})
	}
}
