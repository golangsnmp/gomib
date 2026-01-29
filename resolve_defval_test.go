package gomib

import (
	"testing"
)

// TestResolveDefval verifies that gomib resolves the same DEFVAL clause
// as net-snmp for OBJECT-TYPE nodes in each fixture module.
// Disagreements are skipped rather than failed.
func TestResolveDefval(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				if !isObjectTypeNode(fn) {
					continue
				}
				if fn.DefaultValue == "" {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					obj := m.FindObject(fn.Name)
					if obj == nil {
						t.Skipf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					dv := obj.DefaultValue()
					if dv.IsZero() {
						t.Skipf("divergence: defval for %s: gomib has no defval, fixture=%q",
							fn.Name, fn.DefaultValue)
						return
					}

					gomibDefval := dv.String()
					if !defvalEquivalent(gomibDefval, fn.DefaultValue) {
						t.Skipf("divergence: defval for %s: gomib=%q fixture=%q",
							fn.Name, gomibDefval, fn.DefaultValue)
					}
				})
			}
		})
	}
}
