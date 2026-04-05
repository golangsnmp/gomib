package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestParseMalformedHeaders(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantMsgSub string // substring expected in the first diagnostic message
	}{
		{"missing DEFINITIONS", `TEST-MIB ::= BEGIN END`, "DEFINITIONS"},
		{"missing ::=", `TEST-MIB DEFINITIONS BEGIN END`, "COLON_COLON_EQUAL"},
		{"missing BEGIN", `TEST-MIB DEFINITIONS ::= END`, "BEGIN"},
		{"empty input", ``, "identifier"},
		{"garbage only", `{ } ; 42`, "identifier"},
		{"bad DEFINITIONS keyword", `TEST-MIB DECALRATIONS ::= BEGIN END`, "DEFINITIONS"},
		{"module name only", `TEST-MIB`, "DEFINITIONS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mod := parseModule(tt.source)
			testutil.Equal(t, "UNKNOWN", mod.Name, "module name should be UNKNOWN on header failure")
			testutil.Greater(t, len(mod.Diagnostics), 0, "should have diagnostics")
			testutil.Equal(t, types.DiagParseError, mod.Diagnostics[0].Code, "diagnostic code")
			testutil.Contains(t, mod.Diagnostics[0].Message, tt.wantMsgSub, "diagnostic message should indicate the parse failure point")
		})
	}
}
