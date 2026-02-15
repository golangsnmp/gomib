package types

import "testing"

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

		// Edge cases
		{"", "", true},
		{"", "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"/"+tt.s, func(t *testing.T) {
			got := matchGlob(tt.pattern, tt.s)
			if got != tt.want {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.s, got, tt.want)
			}
		})
	}
}
