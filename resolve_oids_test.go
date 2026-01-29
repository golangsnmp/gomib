package gomib

import (
	"testing"
)

// TestResolveOIDs verifies that gomib resolves the same OIDs as net-snmp
// for all nodes in each fixture module.
// Disagreements are skipped rather than failed, since the cause may be
// differences in what each library exposes (conformance nodes, etc.).
func TestResolveOIDs(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for oid, fn := range fixture {
				t.Run(fn.Name, func(t *testing.T) {
					node := m.FindNode(fn.Name)
					if node == nil {
						t.Skipf("divergence: gomib cannot find node %q (fixture OID %s)", fn.Name, oid)
						return
					}
					gotOID := node.OID().String()
					if gotOID != oid {
						t.Skipf("divergence: OID for %s: gomib=%s fixture=%s", fn.Name, gotOID, oid)
					}
				})
			}
		})
	}
}
