package gomib

import (
	"slices"
	"testing"
)

func TestDedup(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"no dups", []string{"/a", "/b", "/c"}, []string{"/a", "/b", "/c"}},
		{"with dups", []string{"/a", "/b", "/a", "/c", "/b"}, []string{"/a", "/b", "/c"}},
		{"all same", []string{"/a", "/a", "/a"}, []string{"/a"}},
		{"empty", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedup(tt.input)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
