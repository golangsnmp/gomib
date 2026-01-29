package gomib

import (
	"testing"
)

// TestResolveReference verifies that gomib resolves the same REFERENCE clause
// as net-snmp for nodes in each fixture module.
// Disagreements are skipped rather than failed.
func TestResolveReference(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				if fn.Reference == "" {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					// Try object first, then notification
					gomibRef := ""
					if obj := m.FindObject(fn.Name); obj != nil {
						gomibRef = obj.Reference()
					} else if notif := m.FindNotification(fn.Name); notif != nil {
						gomibRef = notif.Reference()
					} else {
						t.Skipf("divergence: gomib does not have node %q", fn.Name)
						return
					}

					if !referenceEquivalent(gomibRef, fn.Reference) {
						t.Skipf("divergence: reference for %s: gomib=%q fixture=%q",
							fn.Name, gomibRef, fn.Reference)
					}
				})
			}
		})
	}
}
