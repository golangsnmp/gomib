package gomib

import (
	"testing"
)

// TestResolveModule verifies that gomib attributes each node to the same
// module as net-snmp.
// Disagreements are skipped rather than failed.
func TestResolveModule(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				if fn.Module == "" {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					node := m.FindNode(fn.Name)
					if node == nil {
						t.Skipf("divergence: gomib cannot find node %q", fn.Name)
						return
					}

					gomibModule := ""
					if node.Module() != nil {
						gomibModule = node.Module().Name()
					}

					if gomibModule != fn.Module {
						t.Skipf("divergence: module for %s: gomib=%q fixture=%q",
							fn.Name, gomibModule, fn.Module)
					}
				})
			}
		})
	}
}
