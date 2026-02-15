package mib

import "testing"

func TestDiagnosticConfigShouldReport(t *testing.T) {
	tests := []struct {
		name   string
		config DiagnosticConfig
		code   string
		sev    Severity
		want   bool
	}{
		// Strict mode reports everything
		{"strict/fatal", StrictConfig(), "test", SeverityFatal, true},
		{"strict/info", StrictConfig(), "test", SeverityInfo, true},
		{"strict/style", StrictConfig(), "test", SeverityStyle, true},

		// Normal mode (level 3): report sev 0-3
		{"normal/fatal", DefaultConfig(), "test", SeverityFatal, true},
		{"normal/minor", DefaultConfig(), "test", SeverityMinor, true},
		{"normal/style", DefaultConfig(), "test", SeverityStyle, false},
		{"normal/info", DefaultConfig(), "test", SeverityInfo, false},

		// Permissive mode (level 5): report sev 0-5
		{"permissive/fatal", PermissiveConfig(), "test", SeverityFatal, true},
		{"permissive/style", PermissiveConfig(), "test", SeverityStyle, true},
		{"permissive/warning", PermissiveConfig(), "test", SeverityWarning, true},
		{"permissive/info", PermissiveConfig(), "test", SeverityInfo, false},

		// Silent mode suppresses everything
		{"silent/fatal", DiagnosticConfig{Level: StrictnessSilent}, "test", SeverityFatal, false},
		{"silent/info", DiagnosticConfig{Level: StrictnessSilent}, "test", SeverityInfo, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ShouldReport(tt.code, tt.sev)
			if got != tt.want {
				t.Errorf("ShouldReport(%q, %v) = %v, want %v", tt.code, tt.sev, got, tt.want)
			}
		})
	}
}

func TestDiagnosticConfigShouldReportIgnore(t *testing.T) {
	cfg := DiagnosticConfig{
		Level:  StrictnessStrict,
		FailAt: SeveritySevere,
		Ignore: []string{"identifier-underscore", "identifier-*"},
	}

	// Exact match
	if cfg.ShouldReport("identifier-underscore", SeverityStyle) {
		t.Error("ignored code should not be reported")
	}

	// Glob match
	if cfg.ShouldReport("identifier-length-64", SeverityError) {
		t.Error("glob-matched code should not be reported")
	}

	// Non-matching code should be reported
	if !cfg.ShouldReport("missing-import", SeverityError) {
		t.Error("non-matching code should be reported")
	}
}

func TestDiagnosticConfigShouldReportOverrides(t *testing.T) {
	cfg := DiagnosticConfig{
		Level:  StrictnessNormal, // reports sev 0-3
		FailAt: SeveritySevere,
		Overrides: map[string]Severity{
			"my-style-check": SeverityMinor, // upgrade from Style(4) to Minor(3)
		},
	}

	// Without override, style severity would not be reported at normal level
	if cfg.ShouldReport("other-style", SeverityStyle) {
		t.Error("style severity should not be reported at normal level")
	}

	// With override, the code is upgraded to Minor which IS reported
	if !cfg.ShouldReport("my-style-check", SeverityStyle) {
		t.Error("overridden code should be reported (upgraded to Minor)")
	}
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
		{"strict/fatal", StrictConfig(), SeverityFatal, true},
		{"strict/severe", StrictConfig(), SeveritySevere, true},
		{"permissive/fatal", PermissiveConfig(), SeverityFatal, true},
		{"permissive/severe", PermissiveConfig(), SeveritySevere, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ShouldFail(tt.sev)
			if got != tt.want {
				t.Errorf("ShouldFail(%v) = %v, want %v", tt.sev, got, tt.want)
			}
		})
	}
}

func TestDiagnosticConfigIsStrict(t *testing.T) {
	tests := []struct {
		name  string
		level StrictnessLevel
		want  bool
	}{
		{"strict", StrictnessStrict, true},
		{"level 1", 1, true},
		{"level 2", 2, true},
		{"normal", StrictnessNormal, false},
		{"permissive", StrictnessPermissive, false},
		{"silent", StrictnessSilent, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DiagnosticConfig{Level: tt.level}
			got := cfg.IsStrict()
			if got != tt.want {
				t.Errorf("IsStrict() = %v, want %v (level=%d)", got, tt.want, tt.level)
			}
		})
	}
}

func TestDiagnosticConfigAllowSafeFallbacks(t *testing.T) {
	tests := []struct {
		name  string
		level StrictnessLevel
		want  bool
	}{
		{"strict", StrictnessStrict, false},
		{"level 2", 2, false},
		{"normal", StrictnessNormal, true},
		{"permissive", StrictnessPermissive, true},
		{"silent", StrictnessSilent, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DiagnosticConfig{Level: tt.level}
			got := cfg.AllowSafeFallbacks()
			if got != tt.want {
				t.Errorf("AllowSafeFallbacks() = %v, want %v (level=%d)", got, tt.want, tt.level)
			}
		})
	}
}

func TestDiagnosticConfigAllowBestGuessFallbacks(t *testing.T) {
	tests := []struct {
		name  string
		level StrictnessLevel
		want  bool
	}{
		{"strict", StrictnessStrict, false},
		{"normal", StrictnessNormal, false},
		{"level 4", 4, false},
		{"permissive", StrictnessPermissive, true},
		{"silent", StrictnessSilent, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DiagnosticConfig{Level: tt.level}
			got := cfg.AllowBestGuessFallbacks()
			if got != tt.want {
				t.Errorf("AllowBestGuessFallbacks() = %v, want %v (level=%d)", got, tt.want, tt.level)
			}
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
			Diagnostic{
				Severity: SeverityMinor,
				Code:     "identifier-underscore",
				Message:  "underscore in identifier",
				Module:   "IF-MIB",
				Line:     42,
				Column:   5,
			},
			"[minor] IF-MIB:42:5: underscore in identifier",
		},
		{
			"line only",
			Diagnostic{
				Severity: SeverityError,
				Code:     "import-not-found",
				Message:  "module not found",
				Module:   "MY-MIB",
				Line:     10,
			},
			"[error] MY-MIB:10: module not found",
		},
		{
			"no location",
			Diagnostic{
				Severity: SeverityFatal,
				Code:     "parse-error",
				Message:  "unexpected EOF",
				Module:   "BAD-MIB",
			},
			"[fatal] BAD-MIB: unexpected EOF",
		},
		{
			"no module",
			Diagnostic{
				Severity: SeverityWarning,
				Code:     "general",
				Message:  "something happened",
			},
			"[warning] something happened",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.diag.String()
			if got != tt.want {
				t.Errorf("Diagnostic.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
