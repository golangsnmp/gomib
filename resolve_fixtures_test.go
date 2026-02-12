package gomib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

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

func TestResolveTables(t *testing.T) {
	m := loadTestMIB(t)

	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)

			for _, fn := range fixture {
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

					if fn.Kind != "" {
						gomibKind := testutil.NormalizeKind(obj.Kind())
						if gomibKind != fn.Kind {
							t.Errorf("divergence: kind for %s: gomib=%q fixture=%q",
								fn.Name, gomibKind, fn.Kind)
						}
					}

					if len(fn.Indexes) > 0 {
						gomibIndexes := testutil.NormalizeIndexes(obj.Index())
						if !indexesEquivalent(gomibIndexes, fn.Indexes) {
							t.Errorf("divergence: indexes for %s:\n  gomib=%s\n  fixture=%s",
								fn.Name,
								testutil.FormatIndexes(gomibIndexes),
								testutil.FormatIndexes(fn.Indexes))
						}
					}

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
					isSMIv1 := obj.Module().Language() == mib.LanguageSMIv1
					if !accessEquivalent(gomibAccess, fn.Access, isSMIv1) {
						t.Errorf("divergence: access for %s: gomib=%q fixture=%q",
							fn.Name, gomibAccess, fn.Access)
					}
				})
			}
		})
	}
}

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

					gotOID := notif.OID().String()
					if gotOID != fn.OID {
						t.Errorf("divergence: OID for notification %s: gomib=%s fixture=%s",
							fn.Name, gotOID, fn.OID)
					}

					if len(fn.Varbinds) > 0 {
						gomibVarbinds := testutil.NormalizeVarbinds(notif.Objects())
						if !varbindsEquivalent(gomibVarbinds, fn.Varbinds) {
							t.Errorf("divergence: varbinds for %s:\n  gomib=%v\n  fixture=%v",
								fn.Name, gomibVarbinds, fn.Varbinds)
						}
					}

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
