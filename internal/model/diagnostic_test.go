package model

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestDiagnosticConfigShouldReport(t *testing.T) {
	tests := []struct {
		name   string
		config DiagnosticConfig
		code   string
		sev    Severity
		want   bool
	}{
		{"verbose/info", VerboseConfig(), "test", SeverityInfo, true},
		{"default/minor", DefaultConfig(), "test", SeverityMinor, true},
		{"default/style", DefaultConfig(), "test", SeverityStyle, false},
		{"quiet/error", QuietConfig(), "test", SeverityError, true},
		{"quiet/warning", QuietConfig(), "test", SeverityWarning, false},
		{"silent/fatal", DiagnosticConfig{Reporting: ReportingSilent}, "test", SeverityFatal, true},
		{"silent/info", DiagnosticConfig{Reporting: ReportingSilent}, "test", SeverityInfo, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ShouldReport(tt.code, tt.sev)
			testutil.Equal(t, tt.want, got, "ShouldReport(, )")
		})
	}
}

func TestDiagnosticConfigShouldReportIgnore(t *testing.T) {
	cfg := DiagnosticConfig{
		Reporting: ReportingVerbose,
		FailAt:    SeveritySevere,
		Ignore:    []string{"identifier-underscore", "identifier-*"},
	}

	testutil.False(t, cfg.ShouldReport("identifier-underscore", SeverityStyle), "ignored code should not be reported")
	testutil.False(t, cfg.ShouldReport("identifier-length-64", SeverityError), "glob-matched code should not be reported")
	testutil.True(t, cfg.ShouldReport("missing-import", SeverityError), "non-matching code should be reported")
}

func TestDiagnosticConfigShouldFail(t *testing.T) {
	tests := []struct {
		name   string
		config DiagnosticConfig
		sev    Severity
		want   bool
	}{
		{"default/fatal", DefaultConfig(), SeverityFatal, true},
		{"default/severe", DefaultConfig(), SeveritySevere, true},
		{"default/error", DefaultConfig(), SeverityError, false},
		{"silent/fatal", SilentConfig(), SeverityFatal, true},
		{"silent/severe", SilentConfig(), SeveritySevere, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ShouldFail(tt.sev)
			testutil.Equal(t, tt.want, got, "ShouldFail()")
		})
	}
}

func TestResolverStrictnessFallbackGates(t *testing.T) {
	tests := []struct {
		name               string
		strictness         ResolverStrictness
		wantConstrained    bool
		wantGlobalFallback bool
	}{
		{"strict", ResolverStrict, false, false},
		{"normal", ResolverNormal, true, false},
		{"permissive", ResolverPermissive, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Equal(t, tt.wantConstrained, tt.strictness.AllowConstrainedFallbacks(), "AllowConstrainedFallbacks()")
			testutil.Equal(t, tt.wantGlobalFallback, tt.strictness.AllowGlobalFallbacks(), "AllowGlobalFallbacks()")
		})
	}
}

func TestDiagnosticString(t *testing.T) {
	tests := []struct {
		name string
		diag Diagnostic
		want string
	}{
		{
			"full",
			Diagnostic{Severity: SeverityMinor, Code: "identifier-underscore", Message: "underscore in identifier", Module: "IF-MIB", Line: 42, Column: 5},
			"[minor] IF-MIB:42:5: underscore in identifier",
		},
		{
			"line only",
			Diagnostic{Severity: SeverityError, Code: "import-not-found", Message: "module not found", Module: "MY-MIB", Line: 10},
			"[error] MY-MIB:10: module not found",
		},
		{
			"no location",
			Diagnostic{Severity: SeverityFatal, Code: "parse-error", Message: "unexpected EOF", Module: "BAD-MIB"},
			"[fatal] BAD-MIB: unexpected EOF",
		},
		{
			"no module",
			Diagnostic{Severity: SeverityWarning, Code: "general", Message: "something happened"},
			"[warning] something happened",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.diag.String()
			testutil.Equal(t, tt.want, got, "Diagnostic.String()")
		})
	}
}
