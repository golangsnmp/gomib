package module

import (
	"slices"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestBaseModuleNames_DefensiveCopy(t *testing.T) {
	original := BaseModuleNames()
	saved := slices.Clone(original)

	// Mutate the returned slice.
	original[0] = "CORRUPTED"

	// A second call must still return the original names.
	got := BaseModuleNames()
	testutil.SliceEqual(t, saved, got, "BaseModuleNames() was mutated by caller")
}
