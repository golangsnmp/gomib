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
			cfg:  StrictConfig(),
			code: "some-info",
			sev:  SeverityInfo,
			want: true,
		},
		{
			name: "silent still reports fatal",
			cfg: DiagnosticConfig{
				Level: StrictnessSilent,
			},
			code: "fatal-thing",
			sev:  SeverityFatal,
			want: true,
		},
		{
			name: "silent suppresses non-fatal",
			cfg: DiagnosticConfig{
				Level: StrictnessSilent,
			},
			code: "some-warning",
			sev:  SeverityWarning,
			want: false,
		},
		{
			name: "ignore suppresses matching code",
			cfg: DiagnosticConfig{
				Level:  StrictnessStrict,
				Ignore: []string{"identifier-underscore"},
			},
			code: "identifier-underscore",
			sev:  SeverityWarning,
			want: false,
		},
		{
			name: "ignore supports glob",
			cfg: DiagnosticConfig{
				Level:  StrictnessStrict,
				Ignore: []string{"identifier-*"},
			},
			code: "identifier-underscore",
			sev:  SeverityWarning,
			want: false,
		},
		{
			name: "override upgrades severity",
			cfg: DiagnosticConfig{
				Level:     StrictnessPermissive,
				Overrides: map[string]Severity{"some-info": SeverityWarning},
			},
			code: "some-info",
			sev:  SeverityInfo,
			want: true,
		},
		{
			name: "override to fatal always reports even with ignore",
			cfg: DiagnosticConfig{
				Level:     StrictnessStrict,
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
				Level:     StrictnessStrict,
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

// TestStrictnessSeverityAlignment verifies that the numeric values of
// StrictnessLevel and Severity enums remain aligned. ShouldReport relies
// on comparing int(sev) <= int(level) for threshold filtering.
func TestStrictnessSeverityAlignment(t *testing.T) {
	if int(StrictnessNormal) != int(SeverityMinor) {
		t.Errorf("StrictnessNormal(%d) != SeverityMinor(%d)", StrictnessNormal, SeverityMinor)
	}
	if int(StrictnessPermissive) != int(SeverityWarning) {
		t.Errorf("StrictnessPermissive(%d) != SeverityWarning(%d)", StrictnessPermissive, SeverityWarning)
	}
	if int(StrictnessSilent) != int(SeverityInfo) {
		t.Errorf("StrictnessSilent(%d) != SeverityInfo(%d)", StrictnessSilent, SeverityInfo)
	}
	if int(StrictnessStrict) != int(SeverityFatal) {
		t.Errorf("StrictnessStrict(%d) != SeverityFatal(%d)", StrictnessStrict, SeverityFatal)
	}
}
