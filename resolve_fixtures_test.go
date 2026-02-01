// resolve_fixtures_test.go verifies gomib output against net-snmp JSON fixtures.
package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
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
						t.Errorf("divergence: gomib cannot find node %q (fixture OID %s)", fn.Name, oid)
						return
					}
					gotOID := node.OID().String()
					if gotOID != oid {
						t.Errorf("divergence: OID for %s: gomib=%s fixture=%s", fn.Name, gotOID, oid)
					}
				})
			}
		})
	}
}

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
						t.Errorf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					gomibType := testutil.NormalizeType(obj.Type())
					if !typesEquivalent(gomibType, fn.Type) {
						t.Errorf("divergence: type for %s: gomib=%q fixture=%q",
							fn.Name, gomibType, fn.Type)
					}

					// TC name
					if fn.TCName != "" {
						gomibTC := ""
						if obj.Type() != nil {
							gomibTC = obj.Type().Name()
						}
						if gomibTC != fn.TCName {
							t.Errorf("divergence: TC name for %s: gomib=%q fixture=%q",
								fn.Name, gomibTC, fn.TCName)
						}
					}

					// Display hint
					if fn.Hint != "" {
						gomibHint := obj.EffectiveDisplayHint()
						if !hintsEquivalent(gomibHint, fn.Hint) {
							t.Errorf("divergence: display hint for %s: gomib=%q fixture=%q",
								fn.Name, gomibHint, fn.Hint)
						}
					}
				})
			}
		})
	}
}

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
						t.Errorf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					gomibEnums := testutil.NormalizeEnums(obj.EffectiveEnums())
					if !enumsEquivalent(gomibEnums, fn.EnumValues) {
						t.Errorf("divergence: enums for %s:\n  gomib=%s\n  fixture=%s",
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
						t.Errorf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					gomibBits := testutil.NormalizeEnums(obj.EffectiveBits())
					if !enumsEquivalent(gomibBits, fn.BitValues) {
						t.Errorf("divergence: bits for %s:\n  gomib=%s\n  fixture=%s",
							fn.Name,
							testutil.FormatEnums(gomibBits),
							testutil.FormatEnums(fn.BitValues))
					}
				})
			}
		})
	}
}

// TestResolveTables verifies that gomib resolves table structure (kind,
// indexes, augments) consistently with net-snmp.
// Disagreements are skipped rather than failed.
func TestResolveTables(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
				// Only test nodes with table-related info
				hasTableInfo := fn.Kind != "" || len(fn.Indexes) > 0 || fn.Augments != ""
				if !hasTableInfo {
					continue
				}

				t.Run(fn.Name, func(t *testing.T) {
					obj := m.FindObject(fn.Name)
					if obj == nil {
						t.Errorf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					// Kind (table/row/column/scalar)
					if fn.Kind != "" {
						gomibKind := testutil.NormalizeKind(obj.Kind())
						if gomibKind != fn.Kind {
							t.Errorf("divergence: kind for %s: gomib=%q fixture=%q",
								fn.Name, gomibKind, fn.Kind)
						}
					}

					// Indexes
					if len(fn.Indexes) > 0 {
						gomibIndexes := testutil.NormalizeIndexes(obj.Index())
						if !indexesEquivalent(gomibIndexes, fn.Indexes) {
							t.Errorf("divergence: indexes for %s:\n  gomib=%s\n  fixture=%s",
								fn.Name,
								testutil.FormatIndexes(gomibIndexes),
								testutil.FormatIndexes(fn.Indexes))
						}
					}

					// Augments
					if fn.Augments != "" {
						gomibAugments := ""
						if aug := obj.Augments(); aug != nil {
							gomibAugments = aug.Name()
						}
						if gomibAugments != fn.Augments {
							t.Errorf("divergence: augments for %s: gomib=%q fixture=%q",
								fn.Name, gomibAugments, fn.Augments)
						}
					}
				})
			}
		})
	}
}

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
						t.Errorf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					gomibAccess := testutil.NormalizeAccess(obj.Access())
					if !accessEquivalent(gomibAccess, fn.Access) {
						t.Errorf("divergence: access for %s: gomib=%q fixture=%q",
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
						t.Errorf("divergence: gomib does not have node %q", fn.Name)
						return
					}

					if !statusEquivalent(gomibStatus, fn.Status) {
						t.Errorf("divergence: status for %s: gomib=%q fixture=%q",
							fn.Name, gomibStatus, fn.Status)
					}
				})
			}
		})
	}
}

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
						t.Errorf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					// Combine ranges and sizes (same as fixture generation)
					var gomibRanges []testutil.RangeInfo
					gomibRanges = append(gomibRanges, testutil.NormalizeRanges(obj.EffectiveRanges())...)
					gomibRanges = append(gomibRanges, testutil.NormalizeRanges(obj.EffectiveSizes())...)

					if !rangesEquivalent(gomibRanges, fn.Ranges) {
						t.Errorf("divergence: ranges for %s:\n  gomib=%s\n  fixture=%s",
							fn.Name,
							testutil.FormatRanges(gomibRanges),
							testutil.FormatRanges(fn.Ranges))
					}
				})
			}
		})
	}
}

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
						t.Errorf("divergence: gomib does not have notification %q", fn.Name)
						return
					}

					// Verify OID
					gotOID := notif.OID().String()
					if gotOID != fn.OID {
						t.Errorf("divergence: OID for notification %s: gomib=%s fixture=%s",
							fn.Name, gotOID, fn.OID)
					}

					// Verify varbinds (OBJECTS clause)
					if len(fn.Varbinds) > 0 {
						gomibVarbinds := testutil.NormalizeVarbinds(notif.Objects())
						if !varbindsEquivalent(gomibVarbinds, fn.Varbinds) {
							t.Errorf("divergence: varbinds for %s:\n  gomib=%v\n  fixture=%v",
								fn.Name, gomibVarbinds, fn.Varbinds)
						}
					}

					// Verify status if present
					if fn.Status != "" {
						gomibStatus := testutil.NormalizeStatus(notif.Status())
						if !statusEquivalent(gomibStatus, fn.Status) {
							t.Errorf("divergence: status for notification %s: gomib=%q fixture=%q",
								fn.Name, gomibStatus, fn.Status)
						}
					}
				})
			}
		})
	}
}

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
						t.Errorf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					gomibUnits := obj.Units()
					if gomibUnits != fn.Units {
						t.Errorf("divergence: units for %s: gomib=%q fixture=%q",
							fn.Name, gomibUnits, fn.Units)
					}
				})
			}
		})
	}
}

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
						t.Errorf("divergence: gomib does not have object %q", fn.Name)
						return
					}

					dv := obj.DefaultValue()
					if dv.IsZero() {
						t.Errorf("divergence: defval for %s: gomib has no defval, fixture=%q",
							fn.Name, fn.DefaultValue)
						return
					}

					gomibDefval := dv.String()
					if !defvalEquivalent(gomibDefval, fn.DefaultValue) {
						t.Errorf("divergence: defval for %s: gomib=%q fixture=%q",
							fn.Name, gomibDefval, fn.DefaultValue)
					}
				})
			}
		})
	}
}

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
						t.Errorf("divergence: gomib does not have node %q", fn.Name)
						return
					}

					if !referenceEquivalent(gomibRef, fn.Reference) {
						t.Errorf("divergence: reference for %s: gomib=%q fixture=%q",
							fn.Name, gomibRef, fn.Reference)
					}
				})
			}
		})
	}
}

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
						t.Errorf("divergence: gomib cannot find node %q", fn.Name)
						return
					}

					gomibModule := ""
					if node.Module() != nil {
						gomibModule = node.Module().Name()
					}

					if gomibModule != fn.Module {
						t.Errorf("divergence: module for %s: gomib=%q fixture=%q",
							fn.Name, gomibModule, fn.Module)
					}
				})
			}
		})
	}
}
