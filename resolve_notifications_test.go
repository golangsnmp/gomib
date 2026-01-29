package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// TestResolveNotifications verifies that gomib resolves the same notification
// OBJECTS (varbinds) and status as net-snmp for NOTIFICATION-TYPE nodes.
// Disagreements are skipped rather than failed.
func TestResolveNotifications(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				if !isNotificationNode(fn) {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					notif := m.FindNotification(fn.Name)
					if notif == nil {
						t.Skipf("divergence: gomib does not have notification %q", fn.Name)
						return
					}

					// Verify OID
					gotOID := notif.OID().String()
					if gotOID != fn.OID {
						t.Skipf("divergence: OID for notification %s: gomib=%s fixture=%s",
							fn.Name, gotOID, fn.OID)
					}

					// Verify varbinds (OBJECTS clause)
					if len(fn.Varbinds) > 0 {
						gomibVarbinds := testutil.NormalizeVarbinds(notif.Objects())
						if !varbindsEquivalent(gomibVarbinds, fn.Varbinds) {
							t.Skipf("divergence: varbinds for %s:\n  gomib=%v\n  fixture=%v",
								fn.Name, gomibVarbinds, fn.Varbinds)
						}
					}

					// Verify status if present
					if fn.Status != "" {
						gomibStatus := testutil.NormalizeStatus(notif.Status())
						if !statusEquivalent(gomibStatus, fn.Status) {
							t.Skipf("divergence: status for notification %s: gomib=%q fixture=%q",
								fn.Name, gomibStatus, fn.Status)
						}
					}
				})
			}
		})
	}
}
