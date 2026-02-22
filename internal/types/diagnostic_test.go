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
