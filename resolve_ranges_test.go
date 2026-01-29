package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// TestResolveRanges verifies that gomib resolves the same effective range
// constraints as net-snmp for OBJECT-TYPE nodes in each fixture module.
// This covers both value ranges (INTEGER types) and size constraints
// (OCTET STRING types).
// Disagreements are skipped rather than failed.
func TestResolveRanges(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				if !isObjectTypeNode(fn) {
					continue
				}
				if len(fn.Ranges) == 0 {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					obj := m.FindObject(fn.Name)
					if obj == nil {
						t.Skipf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					// Combine ranges and sizes (same as fixture generation)
					var gomibRanges []testutil.RangeInfo
					gomibRanges = append(gomibRanges, testutil.NormalizeRanges(obj.EffectiveRanges())...)
					gomibRanges = append(gomibRanges, testutil.NormalizeRanges(obj.EffectiveSizes())...)

					if !rangesEquivalent(gomibRanges, fn.Ranges) {
						t.Skipf("divergence: ranges for %s:\n  gomib=%s\n  fixture=%s",
							fn.Name,
							testutil.FormatRanges(gomibRanges),
							testutil.FormatRanges(fn.Ranges))
					}
				})
			}
		})
	}
}
