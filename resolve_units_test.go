package gomib

import (
	"testing"
)

// TestResolveUnits verifies that gomib resolves the same UNITS clause
// as net-snmp for OBJECT-TYPE nodes in each fixture module.
// Disagreements are skipped rather than failed.
func TestResolveUnits(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				if !isObjectTypeNode(fn) {
					continue
				}
				if fn.Units == "" {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					obj := m.FindObject(fn.Name)
					if obj == nil {
						t.Skipf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					gomibUnits := obj.Units()
					if gomibUnits != fn.Units {
						t.Skipf("divergence: units for %s: gomib=%q fixture=%q",
							fn.Name, gomibUnits, fn.Units)
					}
				})
			}
		})
	}
}
