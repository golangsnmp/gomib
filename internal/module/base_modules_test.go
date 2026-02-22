package module

import (
	"slices"
	"testing"
)

func TestBaseModuleNames_DefensiveCopy(t *testing.T) {
	original := BaseModuleNames()
	saved := slices.Clone(original)

	// Mutate the returned slice.
	original[0] = "CORRUPTED"

	// A second call must still return the original names.
	got := BaseModuleNames()
	if !slices.Equal(got, saved) {
		t.Errorf("BaseModuleNames() was mutated by caller:\n got %v\nwant %v", got, saved)
	}
}
