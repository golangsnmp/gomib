package types

import (
	"testing"
)

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		s       string
		want    bool
	}{
		// Wildcard only
		{"*", "anything", true},
		{"*", "", true},

		// Trailing wildcard
		{"identifier-*", "identifier-underscore", true},
		{"identifier-*", "identifier-", true},
		{"identifier-*", "other-thing", false},
		{"identifier-*", "identifier", false},

		// Leading wildcard
		{"*-mib", "IF-MIB", false},
		{"*-MIB", "IF-MIB", true},
		{"*-underscore", "identifier-underscore", true},

		// Exact match
		{"exact", "exact", true},
		{"exact", "other", false},

		// Middle wildcard
		{"foo*bar", "fooxbar", true},
		{"foo*bar", "foobar", true},
		{"foo*bar", "foobaz", false},
		{"foo*bar", "foo-and-bar", true},
		{"foo*bar", "xfoobar", false},
		{"a*b*c", "axbxc", true},
		{"a*b*c", "abc", true},
		{"a*b*c", "aXXbYYc", true},
		{"a*b*c", "aXXc", false},
		{"a*a*a", "aaa", true},
		{"a*a*a", "aa", false},

		// Edge cases
		{"", "", true},
		{"", "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"/"+tt.s, func(t *testing.T) {
			got := MatchGlob(tt.pattern, tt.s)
			if got != tt.want {
				t.Errorf("MatchGlob(%q, %q) = %v, want %v", tt.pattern, tt.s, got, tt.want)
			}
		})
	}
}

func TestEffectiveSeverity(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Overrides = map[string]Severity{DiagMacroNotImported: SeveritySevere}

	if got := cfg.EffectiveSeverity(DiagMacroNotImported); got != SeveritySevere {
		t.Errorf("EffectiveSeverity() = %v, want %v", got, SeveritySevere)
	}
	if got := cfg.EffectiveSeverity(DiagParseError); got != SeverityError {
		t.Errorf("EffectiveSeverity() without override = %v, want %v", got, SeverityError)
	}
}

func TestShouldReport(t *testing.T) {
	tests := []struct {
		name string
		cfg  DiagnosticConfig
		code string
		sev  Severity
		want bool
	}{
		{
			name: "default reports minor",
			cfg:  DefaultConfig(),
			code: "some-minor",
			sev:  SeverityMinor,
			want: true,
		},
		{
			name: "default suppresses style",
			cfg:  DefaultConfig(),
			code: "some-style",
			sev:  SeverityStyle,
			want: false,
		},
		{
			name: "strict reports info",
			cfg:  VerboseConfig(),
			code: "some-info",
			sev:  SeverityInfo,
			want: true,
		},
		{
			name: "silent still reports fatal",
			cfg: DiagnosticConfig{
				Reporting: ReportingSilent,
			},
			code: "fatal-thing",
			sev:  SeverityFatal,
			want: true,
		},
		{
			name: "silent suppresses non-fatal",
			cfg: DiagnosticConfig{
				Reporting: ReportingSilent,
			},
			code: "some-warning",
			sev:  SeverityWarning,
			want: false,
		},
		{
			name: "ignore suppresses matching code",
			cfg: DiagnosticConfig{
				Reporting: ReportingVerbose,
				Ignore:    []string{"identifier-underscore"},
			},
			code: "identifier-underscore",
			sev:  SeverityWarning,
			want: false,
		},
		{
			name: "ignore supports glob",
			cfg: DiagnosticConfig{
				Reporting: ReportingVerbose,
				Ignore:    []string{"identifier-*"},
			},
			code: "identifier-underscore",
			sev:  SeverityWarning,
			want: false,
		},
		{
			name: "override upgrades severity",
			cfg: DiagnosticConfig{
				Reporting: ReportingDefault,
				Overrides: map[string]Severity{"some-info": SeverityMinor},
			},
			code: "some-info",
			sev:  SeverityInfo,
			want: true,
		},
		{
			name: "override to fatal always reports even with ignore",
			cfg: DiagnosticConfig{
				Reporting: ReportingVerbose,
				Ignore:    []string{"some-code"},
				Overrides: map[string]Severity{"some-code": SeverityFatal},
			},
			code: "some-code",
			sev:  SeverityInfo,
			want: true,
		},
		{
			name: "ignore suppresses non-fatal with override",
			cfg: DiagnosticConfig{
				Reporting: ReportingVerbose,
				Ignore:    []string{"some-code"},
				Overrides: map[string]Severity{"some-code": SeverityWarning},
			},
			code: "some-code",
			sev:  SeverityInfo,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.ShouldReport(tt.code, tt.sev)
			if got != tt.want {
				t.Errorf("ShouldReport(%q, %v) = %v, want %v", tt.code, tt.sev, got, tt.want)
			}
		})
	}
}

func TestShouldRetainDemotion(t *testing.T) {
	cfg := DiagnosticConfig{
		Reporting: ReportingQuiet,
		Overrides: map[string]Severity{DiagParseError: SeverityInfo},
	}
	if !cfg.ShouldRetain(DiagParseError) {
		t.Error("demotion should retain a diagnostic selected by its registered severity")
	}
}

// TestResolverStrictnessOrdering verifies resolver strictness enum ordering.
func TestResolverStrictnessOrdering(t *testing.T) {
	if ResolverStrict >= ResolverNormal || ResolverNormal >= ResolverPermissive {
		t.Fatalf("unexpected resolver ordering: strict=%d normal=%d permissive=%d",
			ResolverStrict, ResolverNormal, ResolverPermissive)
	}
}

func TestReportingLevelOrdering(t *testing.T) {
	if ReportingSilent >= ReportingQuiet || ReportingQuiet >= ReportingDefault || ReportingDefault >= ReportingVerbose {
		t.Fatalf("unexpected reporting ordering: silent=%d quiet=%d default=%d verbose=%d",
			ReportingSilent, ReportingQuiet, ReportingDefault, ReportingVerbose)
	}
}
