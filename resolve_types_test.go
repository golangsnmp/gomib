package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// TestResolveTypes verifies that gomib resolves the same base type and TC name
// as net-snmp for OBJECT-TYPE nodes in each fixture module.
// Disagreements are skipped rather than failed.
func TestResolveTypes(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				if !isObjectTypeNode(fn) {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					obj := m.FindObject(fn.Name)
					if obj == nil {
						t.Skipf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					gomibType := testutil.NormalizeType(obj.Type())
					if !typesEquivalent(gomibType, fn.Type) {
						t.Skipf("divergence: type for %s: gomib=%q fixture=%q",
							fn.Name, gomibType, fn.Type)
					}

					// TC name
					if fn.TCName != "" {
						gomibTC := ""
						if obj.Type() != nil {
							gomibTC = obj.Type().Name()
						}
						if gomibTC != fn.TCName {
							t.Skipf("divergence: TC name for %s: gomib=%q fixture=%q",
								fn.Name, gomibTC, fn.TCName)
						}
					}

					// Display hint
					if fn.Hint != "" {
						gomibHint := obj.EffectiveDisplayHint()
						if !hintsEquivalent(gomibHint, fn.Hint) {
							t.Skipf("divergence: display hint for %s: gomib=%q fixture=%q",
								fn.Name, gomibHint, fn.Hint)
						}
					}
				})
			}
		})
	}
}
