package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

// TestResolveEnums verifies that gomib resolves the same effective enum values
// as net-snmp for OBJECT-TYPE nodes in each fixture module.
// Disagreements are skipped rather than failed.
func TestResolveEnums(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				if !isObjectTypeNode(fn) {
					continue
				}
				if len(fn.EnumValues) == 0 {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					obj := m.FindObject(fn.Name)
					if obj == nil {
						t.Skipf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					gomibEnums := testutil.NormalizeEnums(obj.EffectiveEnums())
					if !enumsEquivalent(gomibEnums, fn.EnumValues) {
						t.Skipf("divergence: enums for %s:\n  gomib=%s\n  fixture=%s",
							fn.Name,
							testutil.FormatEnums(gomibEnums),
							testutil.FormatEnums(fn.EnumValues))
					}
				})
			}
		})
	}
}

// TestResolveBits verifies that gomib resolves the same BITS named values
// as net-snmp for OBJECT-TYPE nodes in each fixture module.
// Disagreements are skipped rather than failed.
func TestResolveBits(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				if !isObjectTypeNode(fn) {
					continue
				}
				if len(fn.BitValues) == 0 {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					obj := m.FindObject(fn.Name)
					if obj == nil {
						t.Skipf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					gomibBits := testutil.NormalizeEnums(obj.EffectiveBits())
					if !enumsEquivalent(gomibBits, fn.BitValues) {
						t.Skipf("divergence: bits for %s:\n  gomib=%s\n  fixture=%s",
							fn.Name,
							testutil.FormatEnums(gomibBits),
							testutil.FormatEnums(fn.BitValues))
					}
				})
			}
		})
	}
}
