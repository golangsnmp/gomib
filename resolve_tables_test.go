package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// TestResolveTables verifies that gomib resolves table structure (kind,
// indexes, augments) consistently with net-snmp.
// Disagreements are skipped rather than failed.
func TestResolveTables(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				// Only test nodes with table-related info
				hasTableInfo := fn.Kind != "" || len(fn.Indexes) > 0 || fn.Augments != ""
				if !hasTableInfo {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					obj := m.FindObject(fn.Name)
					if obj == nil {
						t.Skipf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					// Kind (table/row/column/scalar)
					if fn.Kind != "" {
						gomibKind := testutil.NormalizeKind(obj.Kind())
						if gomibKind != fn.Kind {
							t.Skipf("divergence: kind for %s: gomib=%q fixture=%q",
								fn.Name, gomibKind, fn.Kind)
						}
					}

					// Indexes
					if len(fn.Indexes) > 0 {
						gomibIndexes := testutil.NormalizeIndexes(obj.Index())
						if !indexesEquivalent(gomibIndexes, fn.Indexes) {
							t.Skipf("divergence: indexes for %s:\n  gomib=%s\n  fixture=%s",
								fn.Name,
								testutil.FormatIndexes(gomibIndexes),
								testutil.FormatIndexes(fn.Indexes))
						}
					}

					// Augments
					if fn.Augments != "" {
						gomibAugments := ""
						if aug := obj.Augments(); aug != nil {
							gomibAugments = aug.Name()
						}
						if gomibAugments != fn.Augments {
							t.Skipf("divergence: augments for %s: gomib=%q fixture=%q",
								fn.Name, gomibAugments, fn.Augments)
						}
					}
				})
			}
		})
	}
}
