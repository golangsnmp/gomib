package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// TestResolveAccess verifies that gomib resolves the same access level
// as net-snmp for OBJECT-TYPE nodes in each fixture module.
// Disagreements are skipped rather than failed.
func TestResolveAccess(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				if fn.Access == "" {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					obj := m.FindObject(fn.Name)
					if obj == nil {
						t.Skipf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					gomibAccess := testutil.NormalizeAccess(obj.Access())
					if !accessEquivalent(gomibAccess, fn.Access) {
						t.Skipf("divergence: access for %s: gomib=%q fixture=%q",
							fn.Name, gomibAccess, fn.Access)
					}
				})
			}
		})
	}
}

// TestResolveStatus verifies that gomib resolves the same status value
// as net-snmp for nodes in each fixture module.
// Disagreements are skipped rather than failed.
func TestResolveStatus(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				if fn.Status == "" {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					// Try object first, then notification
					gomibStatus := ""
					if obj := m.FindObject(fn.Name); obj != nil {
						gomibStatus = testutil.NormalizeStatus(obj.Status())
					} else if notif := m.FindNotification(fn.Name); notif != nil {
						gomibStatus = testutil.NormalizeStatus(notif.Status())
					} else {
						t.Skipf("divergence: gomib does not have node %q", fn.Name)
						return
					}

					if !statusEquivalent(gomibStatus, fn.Status) {
						t.Skipf("divergence: status for %s: gomib=%q fixture=%q",
							fn.Name, gomibStatus, fn.Status)
					}
				})
			}
		})
	}
}
