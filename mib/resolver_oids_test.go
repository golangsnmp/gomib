package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/graph"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestWellKnownRootArc(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{"ccitt", 0},
		{"iso", 1},
		{"joint-iso-ccitt", 2},
		{"internet", -1},
		{"enterprises", -1},
		{"", -1},
		{"ISO", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wellKnownRootArc(tt.name)
			testutil.Equal(t, tt.want, got, "wellKnownRootArc()")
		})
	}
}

func TestLanguageRank(t *testing.T) {
	tests := []struct {
		lang types.Language
		want int
	}{
		{types.LanguageSMIv2, 2},
		{types.LanguageSMIv1, 1},
		{types.LanguageUnknown, 0},
	}

	for _, tt := range tests {
		t.Run(tt.lang.String(), func(t *testing.T) {
			got := languageRank(tt.lang)
			testutil.Equal(t, tt.want, got, "languageRank()")
		})
	}
}

func TestCollectOidDefinitions(t *testing.T) {
	mod := &module.Module{Name: "TEST-MIB"}

	oid := module.NewOidAssignment([]module.OidComponent{
		&module.OidComponentName{NameValue: "enterprises"},
		&module.OidComponentNumber{Value: 42},
	}, types.Synthetic)

	trapOid := module.NewOidAssignment([]module.OidComponent{
		&module.OidComponentName{NameValue: "enterprises"},
		&module.OidComponentNumber{Value: 99},
	}, types.Synthetic)

	mod.Definitions = []module.Definition{
		&module.ObjectType{DefBase: module.DefBase{Name: "myObject"}, Oid: oid},
		&module.ModuleIdentity{DefBase: module.DefBase{Name: "myModId"}, Oid: oid},
		&module.ObjectIdentity{DefBase: module.DefBase{Name: "myObjId"}, Oid: oid},
		&module.Notification{DefBase: module.DefBase{Name: "myNotif"}, Oid: &trapOid},
		&module.Notification{DefBase: module.DefBase{Name: "myTrap"}, TrapInfo: &module.TrapInfo{Enterprise: "enterprises", TrapNumber: 1}},
		&module.Notification{DefBase: module.DefBase{Name: "emptyNotif"}}, // no oid, no trap info - skipped
		&module.ValueAssignment{DefBase: module.DefBase{Name: "myVal"}, Oid: oid},
		&module.ObjectGroup{DefBase: module.DefBase{Name: "myGrp"}, Oid: oid},
		&module.NotificationGroup{DefBase: module.DefBase{Name: "myNotifGrp"}, Oid: oid},
		&module.ModuleCompliance{DefBase: module.DefBase{Name: "myComp"}, Oid: oid},
		&module.AgentCapabilities{DefBase: module.DefBase{Name: "myCap"}, Oid: oid},
		&module.TypeDef{DefBase: module.DefBase{Name: "MyType"}}, // skipped
	}

	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
	ctx.modules = []*module.Module{mod}
	defs := collectOidDefinitions(ctx)

	// All OID-bearing definitions except TypeDef and the empty notification
	testutil.Len(t, defs.oidDefs, 9, "oid defs")

	testutil.Len(t, defs.trapDefs, 1, "trap defs")

	testutil.Equal(t, "myTrap", defs.trapDefs[0].defName(), "trap def name")

	// The empty notification should produce a diagnostic.
	diags := ctx.Diagnostics()
	testutil.Len(t, diags, 1, "diagnostics")
	testutil.Equal(t, types.DiagNotifNoOid, diags[0].Code, "diagnostic code")
}

func TestCollectOidDefinitionsEmpty(t *testing.T) {
	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
	defs := collectOidDefinitions(ctx)

	testutil.Len(t, defs.oidDefs, 0, "expected no oid defs, got")
	testutil.Len(t, defs.trapDefs, 0, "expected no trap defs, got")
}

func TestGetOidParentSymbol(t *testing.T) {
	mod := &module.Module{Name: "TEST-MIB"}
	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
	ctx.modules = []*module.Module{mod}

	makeOidDef := func(components []module.OidComponent) oidDefinition {
		oid := module.NewOidAssignment(components, types.Synthetic)
		return oidDefinition{
			mod:  mod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}
	}

	t.Run("nil oid", func(t *testing.T) {
		def := oidDefinition{
			mod:  mod,
			def:  &module.Notification{DefBase: module.DefBase{Name: "n"}},
			kind: defNotification,
		}
		_, ok := getOidParentSymbol(ctx, def)
		testutil.False(t, ok, "expected false for nil oid")
	})

	t.Run("empty components", func(t *testing.T) {
		def := makeOidDef(nil)
		_, ok := getOidParentSymbol(ctx, def)
		testutil.False(t, ok, "expected false for empty components")
	})

	t.Run("OidComponentNumber", func(t *testing.T) {
		def := makeOidDef([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		})
		_, ok := getOidParentSymbol(ctx, def)
		testutil.False(t, ok, "expected false for numeric root")
	})

	t.Run("OidComponentName well-known root", func(t *testing.T) {
		def := makeOidDef([]module.OidComponent{
			&module.OidComponentName{NameValue: "iso"},
		})
		_, ok := getOidParentSymbol(ctx, def)
		testutil.False(t, ok, "expected false for well-known root name")
	})

	t.Run("OidComponentName local def", func(t *testing.T) {
		localMod := &module.Module{
			Name: "LOCAL-MIB",
			Definitions: []module.Definition{
				&module.ValueAssignment{
					DefBase: module.DefBase{Name: "enterprises"},
					Oid: module.NewOidAssignment([]module.OidComponent{
						&module.OidComponentNumber{Value: 1},
					}, types.Synthetic),
				},
			},
		}
		localCtx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		setTestModules(localCtx, []*module.Module{localMod})
		def := oidDefinition{
			mod: localMod,
			def: &module.ValueAssignment{
				DefBase: module.DefBase{Name: "myNode"},
				Oid: module.NewOidAssignment([]module.OidComponent{
					&module.OidComponentName{NameValue: "enterprises"},
					&module.OidComponentNumber{Value: 1},
				}, types.Synthetic),
			},
			kind: defValueAssignment,
		}
		sym, ok := getOidParentSymbol(localCtx, def)
		testutil.True(t, ok, "expected true for local definition")
		testutil.Equal(t, "LOCAL-MIB", sym.Module, "module")
		testutil.Equal(t, "enterprises", sym.Name, "name")
	})

	t.Run("OidComponentNamedNumber with known name", func(t *testing.T) {
		localMod := &module.Module{
			Name: "LOCAL-MIB",
			Definitions: []module.Definition{
				&module.ValueAssignment{
					DefBase: module.DefBase{Name: "org"},
					Oid: module.NewOidAssignment([]module.OidComponent{
						&module.OidComponentNumber{Value: 3},
					}, types.Synthetic),
				},
			},
		}
		localCtx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		setTestModules(localCtx, []*module.Module{localMod})
		def := oidDefinition{
			mod: localMod,
			def: &module.ValueAssignment{
				DefBase: module.DefBase{Name: "test"},
				Oid: module.NewOidAssignment([]module.OidComponent{
					&module.OidComponentNamedNumber{NameValue: "org", NumberValue: 3},
				}, types.Synthetic),
			},
			kind: defValueAssignment,
		}
		sym, ok := getOidParentSymbol(localCtx, def)
		testutil.True(t, ok, "expected true")
		testutil.Equal(t, "org", sym.Name, "got name")
	})

	t.Run("OidComponentNamedNumber with well-known root", func(t *testing.T) {
		def := makeOidDef([]module.OidComponent{
			&module.OidComponentNamedNumber{NameValue: "iso", NumberValue: 1},
		})
		_, ok := getOidParentSymbol(ctx, def)
		testutil.False(t, ok, "expected false for well-known root named number")
	})

	t.Run("OidComponentNamedNumber unknown falls back to no dependency", func(t *testing.T) {
		def := makeOidDef([]module.OidComponent{
			&module.OidComponentNamedNumber{NameValue: "unknown", NumberValue: 99},
		})
		_, ok := getOidParentSymbol(ctx, def)
		testutil.False(t, ok, "expected false for unknown named number (has numeric fallback)")
	})

	t.Run("OidComponentQualifiedName", func(t *testing.T) {
		def := makeOidDef([]module.OidComponent{
			&module.OidComponentQualifiedName{ModuleValue: "SNMPv2-SMI", NameValue: "enterprises"},
		})
		sym, ok := getOidParentSymbol(ctx, def)
		testutil.True(t, ok, "expected true for qualified name")
		testutil.Equal(t, "SNMPv2-SMI", sym.Module, "module")
		testutil.Equal(t, "enterprises", sym.Name, "name")
	})

	t.Run("OidComponentQualifiedNamedNumber", func(t *testing.T) {
		def := makeOidDef([]module.OidComponent{
			&module.OidComponentQualifiedNamedNumber{ModuleValue: "RFC1155-SMI", NameValue: "private", NumberValue: 4},
		})
		sym, ok := getOidParentSymbol(ctx, def)
		testutil.True(t, ok, "expected true for qualified named number")
		testutil.Equal(t, "RFC1155-SMI", sym.Module, "module")
		testutil.Equal(t, "private", sym.Name, "name")
	})

	t.Run("Normal mode resolves SMI global root", func(t *testing.T) {
		vendorMod := &module.Module{Name: "VENDOR-MIB"}
		normalCtx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		normalCtx.modules = []*module.Module{vendorMod}

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentName{NameValue: "enterprises"},
			&module.OidComponentNumber{Value: 42},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  vendorMod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "vendorRoot"}, Oid: oid},
			kind: defValueAssignment,
		}

		sym, ok := getOidParentSymbol(normalCtx, def)
		testutil.True(t, ok, "expected true in Normal mode for SMI global root")
		testutil.Equal(t, "SNMPv2-SMI", sym.Module, "module")
		testutil.Equal(t, "enterprises", sym.Name, "name")
	})

	t.Run("Strict mode rejects unimported SMI global", func(t *testing.T) {
		vendorMod := &module.Module{Name: "VENDOR-MIB"}
		strictCtx := newResolverContext(nil, ResolverStrict, VerboseConfig())
		strictCtx.modules = []*module.Module{vendorMod}

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentName{NameValue: "enterprises"},
			&module.OidComponentNumber{Value: 42},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  vendorMod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "vendorRoot"}, Oid: oid},
			kind: defValueAssignment,
		}

		_, ok := getOidParentSymbol(strictCtx, def)
		testutil.False(t, ok, "expected false in strict mode for unimported SMI global")
	})
}

func TestCheckSmiv2IdentifierHyphens(t *testing.T) {
	t.Run("SMIv2 with hyphen emits diagnostic", func(t *testing.T) {
		mod := &module.Module{Name: "MY-MIB", Language: types.LanguageSMIv2}
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		}, types.Synthetic)
		defs := []oidDefinition{
			{mod: mod, def: &module.ValueAssignment{DefBase: module.DefBase{Name: "my-object"}, Oid: oid}, kind: defValueAssignment},
		}

		// Use verbose reporting so SeverityWarning diagnostics are reported.
		ctx := newResolverContext(nil, ResolverNormal, VerboseConfig())
		checkSmiv2IdentifierHyphens(ctx, defs)

		d := hasDiag(t, ctx.Diagnostics(), "identifier-hyphen-smiv2")
		testutil.Equal(t, SeverityWarning, d.Severity, "severity")
		testutil.Equal(t, "MY-MIB", d.Module, "module")
	})

	t.Run("SMIv2 without hyphen emits nothing", func(t *testing.T) {
		mod := &module.Module{Name: "MY-MIB", Language: types.LanguageSMIv2}
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		}, types.Synthetic)
		defs := []oidDefinition{
			{mod: mod, def: &module.ValueAssignment{DefBase: module.DefBase{Name: "myObject"}, Oid: oid}, kind: defValueAssignment},
		}

		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		checkSmiv2IdentifierHyphens(ctx, defs)

		testutil.Len(t, ctx.Diagnostics(), 0, "expected no diagnostics, got")
	})

	t.Run("SMIv1 with hyphen emits nothing", func(t *testing.T) {
		mod := &module.Module{Name: "MY-MIB", Language: types.LanguageSMIv1}
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		}, types.Synthetic)
		defs := []oidDefinition{
			{mod: mod, def: &module.ValueAssignment{DefBase: module.DefBase{Name: "my-object"}, Oid: oid}, kind: defValueAssignment},
		}

		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		checkSmiv2IdentifierHyphens(ctx, defs)

		testutil.Len(t, ctx.Diagnostics(), 0, "expected no diagnostics for SMIv1, got")
	})

	t.Run("base module skipped", func(t *testing.T) {
		mod := &module.Module{Name: "SNMPv2-SMI", Language: types.LanguageSMIv2}
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		}, types.Synthetic)
		defs := []oidDefinition{
			{mod: mod, def: &module.ValueAssignment{DefBase: module.DefBase{Name: "mib-2"}, Oid: oid}, kind: defValueAssignment},
		}

		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		checkSmiv2IdentifierHyphens(ctx, defs)

		testutil.Len(t, ctx.Diagnostics(), 0, "expected no diagnostics for base module, got")
	})
}

func TestResolveNumericComponent(t *testing.T) {
	t.Run("nil parent creates root-level node", func(t *testing.T) {
		ctx := newTestContext()
		node := resolveNumericComponent(ctx, nil, 1)
		testutil.NotNil(t, node, "expected non-nil node")
		testutil.Equal(t, 1, node.Arc(), "arc")
		// The node is a child of the pseudo-root, so its parent is
		// the pseudo-root (not nil). Verify it's the same node that
		// Builder.GetOrCreateRoot returns.
		testutil.Equal(t, ctx.mib.Root().getOrCreateChild(1), node, "expected same node as Builder.GetOrCreateRoot(1)")
	})

	t.Run("creates child of existing parent", func(t *testing.T) {
		ctx := newTestContext()
		parent := ctx.mib.Root().getOrCreateChild(1) // iso
		child := resolveNumericComponent(ctx, parent, 3)
		testutil.NotNil(t, child, "expected non-nil child")
		testutil.Equal(t, 3, child.Arc(), "arc")
		testutil.False(t, child.IsRoot(), "expected non-root (has parent)")
	})

	t.Run("returns same node on repeat", func(t *testing.T) {
		ctx := newTestContext()
		parent := ctx.mib.Root().getOrCreateChild(1)
		child1 := resolveNumericComponent(ctx, parent, 3)
		child2 := resolveNumericComponent(ctx, parent, 3)
		testutil.Equal(t, child2, child1, "expected same node on repeated getOrCreateChild")
	})
}

func TestLookupOrCreateWellKnownRoot(t *testing.T) {
	tests := []struct {
		name    string
		wantArc uint32
		wantOk  bool
	}{
		{"iso", 1, true},
		{"ccitt", 0, true},
		{"joint-iso-ccitt", 2, true},
		{"internet", 0, false},
		{"unknown", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestContext()
			node, ok := lookupOrCreateWellKnownRoot(ctx, tt.name)
			testutil.Equal(t, tt.wantOk, ok, "ok")
			if ok {
				testutil.Equal(t, tt.wantArc, node.Arc(), "arc")
			}
		})
	}
}

func TestLookupSmiGlobalOidRoot(t *testing.T) {
	t.Run("returns node when registered in SNMPv2-SMI", func(t *testing.T) {
		smiMod := &module.Module{Name: "SNMPv2-SMI"}
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{smiMod}
		ctx.moduleIndex["SNMPv2-SMI"] = []*module.Module{smiMod}

		node := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1)
		ctx.registerModuleNodeSymbol(smiMod, "internet", node)

		got, ok := lookupSmiGlobalOidRoot(ctx, "internet")
		testutil.True(t, ok, "expected true")
		testutil.Equal(t, node, got, "expected same node")
	})

	t.Run("returns node when registered in RFC1155-SMI", func(t *testing.T) {
		rfc1155Mod := &module.Module{Name: "RFC1155-SMI"}
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{rfc1155Mod}
		ctx.moduleIndex["RFC1155-SMI"] = []*module.Module{rfc1155Mod}

		node := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1)
		ctx.registerModuleNodeSymbol(rfc1155Mod, "internet", node)

		got, ok := lookupSmiGlobalOidRoot(ctx, "internet")
		testutil.True(t, ok, "expected true")
		testutil.Equal(t, node, got, "expected same node")
	})

	t.Run("returns false for non-global name", func(t *testing.T) {
		ctx := newTestContext()
		_, ok := lookupSmiGlobalOidRoot(ctx, "myCustomOid")
		testutil.False(t, ok, "expected false for non-global name")
	})

	t.Run("returns false when module not loaded", func(t *testing.T) {
		ctx := newTestContext()
		_, ok := lookupSmiGlobalOidRoot(ctx, "internet")
		testutil.False(t, ok, "expected false when module not loaded")
	})
}

func TestShouldPreferModule(t *testing.T) {
	t.Run("nil currentMod always prefers new", func(t *testing.T) {
		srcMod := &module.Module{Name: "NEW-MIB", Language: types.LanguageSMIv1}
		newMod := newModule("NEW-MIB")
		ctx := newTestContext()
		ctx.moduleToResolved = map[*module.Module]*Module{srcMod: newMod}

		testutil.True(t, shouldPreferModule(ctx, nil, srcMod), "expected true when currentMod is nil")
	})

	t.Run("nil currentSrcMod prefers new", func(t *testing.T) {
		srcMod := &module.Module{Name: "NEW-MIB", Language: types.LanguageSMIv1}
		newMod := newModule("NEW-MIB")
		currentMod := newModule("OLD-MIB")
		ctx := newTestContext()
		ctx.moduleToResolved = map[*module.Module]*Module{srcMod: newMod}
		ctx.resolvedToModule = map[*Module]*module.Module{} // currentMod not mapped

		testutil.True(t, shouldPreferModule(ctx, currentMod, srcMod), "expected true when currentSrcMod lookup returns nil")
	})

	t.Run("SMIv2 preferred over SMIv1", func(t *testing.T) {
		newSrc := &module.Module{Name: "NEW-MIB", Language: types.LanguageSMIv2}
		oldSrc := &module.Module{Name: "OLD-MIB", Language: types.LanguageSMIv1}
		newMod := newModule("NEW-MIB")
		oldMod := newModule("OLD-MIB")

		ctx := newTestContext()
		ctx.moduleToResolved = map[*module.Module]*Module{newSrc: newMod, oldSrc: oldMod}
		ctx.resolvedToModule = map[*Module]*module.Module{oldMod: oldSrc, newMod: newSrc}

		testutil.True(t, shouldPreferModule(ctx, oldMod, newSrc), "expected SMIv2 to be preferred over SMIv1")
	})

	t.Run("SMIv1 not preferred over SMIv2", func(t *testing.T) {
		newSrc := &module.Module{Name: "NEW-MIB", Language: types.LanguageSMIv1}
		oldSrc := &module.Module{Name: "OLD-MIB", Language: types.LanguageSMIv2}
		newMod := newModule("NEW-MIB")
		oldMod := newModule("OLD-MIB")

		ctx := newTestContext()
		ctx.moduleToResolved = map[*module.Module]*Module{newSrc: newMod, oldSrc: oldMod}
		ctx.resolvedToModule = map[*Module]*module.Module{oldMod: oldSrc, newMod: newSrc}

		testutil.False(t, shouldPreferModule(ctx, oldMod, newSrc), "expected SMIv1 NOT to be preferred over SMIv2")
	})

	t.Run("same language uses LAST-UPDATED tiebreaker", func(t *testing.T) {
		newSrc := &module.Module{
			Name:     "NEW-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ModuleIdentity{DefBase: module.DefBase{Name: "newMIB"}, LastUpdated: "200501010000Z"},
			},
		}
		oldSrc := &module.Module{
			Name:     "OLD-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ModuleIdentity{DefBase: module.DefBase{Name: "oldMIB"}, LastUpdated: "200001010000Z"},
			},
		}
		newMod := newModule("NEW-MIB")
		oldMod := newModule("OLD-MIB")

		ctx := newTestContext()
		ctx.moduleToResolved = map[*module.Module]*Module{newSrc: newMod, oldSrc: oldMod}
		ctx.resolvedToModule = map[*Module]*module.Module{oldMod: oldSrc, newMod: newSrc}

		testutil.True(t, shouldPreferModule(ctx, oldMod, newSrc), "expected newer LAST-UPDATED to win")
	})

	t.Run("same language older LAST-UPDATED loses", func(t *testing.T) {
		newSrc := &module.Module{
			Name:     "OLD-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ModuleIdentity{DefBase: module.DefBase{Name: "oldMIB"}, LastUpdated: "199901010000Z"},
			},
		}
		oldSrc := &module.Module{
			Name:     "NEW-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ModuleIdentity{DefBase: module.DefBase{Name: "newMIB"}, LastUpdated: "200501010000Z"},
			},
		}
		newMod := newModule("OLD-MIB")
		oldMod := newModule("NEW-MIB")

		ctx := newTestContext()
		ctx.moduleToResolved = map[*module.Module]*Module{newSrc: newMod, oldSrc: oldMod}
		ctx.resolvedToModule = map[*Module]*module.Module{oldMod: oldSrc, newMod: newSrc}

		testutil.False(t, shouldPreferModule(ctx, oldMod, newSrc), "expected older LAST-UPDATED to lose")
	})

	t.Run("base module beats vendor module", func(t *testing.T) {
		baseSrc := &module.Module{Name: "SNMPv2-SMI", Language: types.LanguageSMIv2}
		vendorSrc := &module.Module{Name: "VENDOR-MIB", Language: types.LanguageSMIv2}
		baseMod := newModule("SNMPv2-SMI")
		vendorMod := newModule("VENDOR-MIB")

		ctx := newTestContext()
		ctx.moduleToResolved = map[*module.Module]*Module{baseSrc: baseMod, vendorSrc: vendorMod}
		ctx.resolvedToModule = map[*Module]*module.Module{vendorMod: vendorSrc, baseMod: baseSrc}

		// Base module should replace vendor module
		testutil.True(t, shouldPreferModule(ctx, vendorMod, baseSrc), "expected base module to replace vendor")
		// Vendor module should not replace base module
		testutil.False(t, shouldPreferModule(ctx, baseMod, vendorSrc), "expected vendor NOT to replace base module")
	})

	t.Run("base module beats vendor even with newer LAST-UPDATED", func(t *testing.T) {
		baseSrc := &module.Module{Name: "RFC1155-SMI", Language: types.LanguageSMIv1}
		vendorSrc := &module.Module{
			Name:     "VENDOR-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ModuleIdentity{DefBase: module.DefBase{Name: "vendorMIB"}, LastUpdated: "202501010000Z"},
			},
		}
		baseMod := newModule("RFC1155-SMI")
		vendorMod := newModule("VENDOR-MIB")

		ctx := newTestContext()
		ctx.moduleToResolved = map[*module.Module]*Module{baseSrc: baseMod, vendorSrc: vendorMod}
		ctx.resolvedToModule = map[*Module]*module.Module{vendorMod: vendorSrc, baseMod: baseSrc}

		// Base module wins even though vendor is SMIv2 with newer timestamp
		testutil.True(t, shouldPreferModule(ctx, vendorMod, baseSrc), "expected base module to win over newer vendor")
		testutil.False(t, shouldPreferModule(ctx, baseMod, vendorSrc), "expected vendor NOT to replace base module")
	})

	t.Run("equal rank and timestamp uses lexicographic module name", func(t *testing.T) {
		// Both modules have the same language and LAST-UPDATED.
		// The lexicographically smaller module name should win.
		aSrc := &module.Module{
			Name:     "ALPHA-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ModuleIdentity{DefBase: module.DefBase{Name: "alphaMIB"}, LastUpdated: "200501010000Z"},
			},
		}
		bSrc := &module.Module{
			Name:     "BRAVO-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ModuleIdentity{DefBase: module.DefBase{Name: "bravoMIB"}, LastUpdated: "200501010000Z"},
			},
		}
		aMod := newModule("ALPHA-MIB")
		bMod := newModule("BRAVO-MIB")

		ctx := newTestContext()
		ctx.moduleToResolved = map[*module.Module]*Module{aSrc: aMod, bSrc: bMod}
		ctx.resolvedToModule = map[*Module]*module.Module{aMod: aSrc, bMod: bSrc}

		// ALPHA < BRAVO lexicographically, so ALPHA should win
		testutil.True(t, shouldPreferModule(ctx, bMod, aSrc), "expected ALPHA-MIB to replace BRAVO-MIB")
		testutil.False(t, shouldPreferModule(ctx, aMod, bSrc), "expected BRAVO-MIB NOT to replace ALPHA-MIB")
	})
}

func TestFinalizeOidDefinition(t *testing.T) {
	tests := []struct {
		name     string
		kind     definitionKind
		wantKind Kind
	}{
		{"ObjectType", defObjectType, KindScalar},
		{"ModuleIdentity", defModuleIdentity, KindNode},
		{"ObjectIdentity", defObjectIdentity, KindNode},
		{"ValueAssignment", defValueAssignment, KindNode},
		{"Notification", defNotification, KindNotification},
		{"ObjectGroup", defObjectGroup, KindGroup},
		{"NotificationGroup", defNotificationGroup, KindGroup},
		{"ModuleCompliance", defModuleCompliance, KindCompliance},
		{"AgentCapabilities", defAgentCapabilities, KindCapability},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcMod := &module.Module{Name: "TEST-MIB", Language: types.LanguageSMIv2}
			ctx := newTestContextForModules(DefaultConfig(), srcMod)
			resolvedMod := ctx.moduleToResolved[srcMod]

			node := buildOIDPath(ctx.mib.Root(), 1, 3, 6)

			oid := module.NewOidAssignment([]module.OidComponent{
				&module.OidComponentNumber{Value: 1},
			}, types.Synthetic)
			def := oidDefinition{
				mod:  srcMod,
				def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "testNode"}, Oid: oid},
				kind: tt.kind,
			}

			finalizeOidDefinition(ctx, def, node, "testNode")

			testutil.Equal(t, tt.wantKind, node.Kind(), "kind")
			testutil.Equal(t, "testNode", node.Name(), "name")
			testutil.Equal(t, resolvedMod, node.Module(), "expected module to be set to resolvedMod")
		})
	}
}

func TestFinalizeOidDefinitionNameModuleSync(t *testing.T) {
	// When two modules define the same OID with different names,
	// the node's name and module must stay in sync. The preferred
	// module's name should win; a non-preferred module must not
	// overwrite the name.

	t.Run("non-preferred module does not overwrite name", func(t *testing.T) {
		// SMIv2 module defines "preferredName" first
		preferredSrc := &module.Module{Name: "PREFERRED-MIB", Language: types.LanguageSMIv2}
		// SMIv1 module defines "loserName" second (lower rank, should not win)
		loserSrc := &module.Module{Name: "LOSER-MIB", Language: types.LanguageSMIv1}
		ctx := newTestContextForModules(DefaultConfig(), preferredSrc, loserSrc)
		preferredMod := ctx.moduleToResolved[preferredSrc]

		node := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 4, 1, 42)

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		}, types.Synthetic)

		// First: preferred module claims the node
		def1 := oidDefinition{
			mod:  preferredSrc,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "preferredName"}, Oid: oid},
			kind: defValueAssignment,
		}
		finalizeOidDefinition(ctx, def1, node, "preferredName")

		testutil.Equal(t, "preferredName", node.Name(), "name after preferred")
		testutil.Equal(t, preferredMod, node.Module(), "module after preferred")

		// Second: non-preferred module tries to claim the same node
		def2 := oidDefinition{
			mod:  loserSrc,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "loserName"}, Oid: oid},
			kind: defValueAssignment,
		}
		finalizeOidDefinition(ctx, def2, node, "loserName")

		// Name and module must both still reflect the preferred module
		testutil.Equal(t, "preferredName", node.Name(), "name should not be overwritten by non-preferred module")
		testutil.Equal(t, preferredMod, node.Module(), "module should not be overwritten by non-preferred module")
	})

	t.Run("non-preferred module does not overwrite kind", func(t *testing.T) {
		// SMIv2 module defines as OBJECT-IDENTITY (KindNode) first
		preferredSrc := &module.Module{Name: "PREFERRED-MIB", Language: types.LanguageSMIv2}
		// SMIv1 module defines as OBJECT-TYPE (KindScalar) second
		loserSrc := &module.Module{Name: "LOSER-MIB", Language: types.LanguageSMIv1}
		ctx := newTestContextForModules(DefaultConfig(), preferredSrc, loserSrc)
		preferredMod := ctx.moduleToResolved[preferredSrc]

		node := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 4, 1, 44)

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		}, types.Synthetic)

		def1 := oidDefinition{
			mod:  preferredSrc,
			def:  &module.ObjectIdentity{DefBase: module.DefBase{Name: "preferredNode"}, Oid: oid},
			kind: defObjectIdentity,
		}
		finalizeOidDefinition(ctx, def1, node, "preferredNode")
		testutil.Equal(t, KindNode, node.Kind(), "kind after preferred")

		def2 := oidDefinition{
			mod:  loserSrc,
			def:  &module.ObjectType{DefBase: module.DefBase{Name: "loserObj"}, Oid: oid},
			kind: defObjectType,
		}
		finalizeOidDefinition(ctx, def2, node, "loserObj")

		testutil.Equal(t, KindNode, node.Kind(), "kind should not be overwritten by non-preferred module")
		testutil.Equal(t, "preferredNode", node.Name(), "name should not be overwritten")
		testutil.Equal(t, preferredMod, node.Module(), "module should not be overwritten")
	})

	t.Run("preferred module does overwrite name", func(t *testing.T) {
		// SMIv1 module defines "olderName" first
		olderSrc := &module.Module{Name: "OLD-MIB", Language: types.LanguageSMIv1}
		// SMIv2 module defines "newerName" second (higher rank, should win)
		newerSrc := &module.Module{Name: "NEW-MIB", Language: types.LanguageSMIv2}
		ctx := newTestContextForModules(DefaultConfig(), olderSrc, newerSrc)
		newerMod := ctx.moduleToResolved[newerSrc]

		node := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 4, 1, 43)

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		}, types.Synthetic)

		// First: older module claims the node
		def1 := oidDefinition{
			mod:  olderSrc,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "olderName"}, Oid: oid},
			kind: defValueAssignment,
		}
		finalizeOidDefinition(ctx, def1, node, "olderName")

		// Second: preferred module overwrites
		def2 := oidDefinition{
			mod:  newerSrc,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "newerName"}, Oid: oid},
			kind: defValueAssignment,
		}
		finalizeOidDefinition(ctx, def2, node, "newerName")

		// Both name and module should reflect the preferred module
		testutil.Equal(t, "newerName", node.Name(), "name should be overwritten by preferred module")
		testutil.Equal(t, newerMod, node.Module(), "module should be overwritten by preferred module")
	})
}

func TestOidDefinitionDefName(t *testing.T) {
	oid := module.NewOidAssignment([]module.OidComponent{
		&module.OidComponentNumber{Value: 1},
	}, types.Synthetic)

	def := oidDefinition{
		mod:  &module.Module{Name: "TEST-MIB"},
		def:  &module.ObjectType{DefBase: module.DefBase{Name: "sysDescr"}, Oid: oid},
		kind: defObjectType,
	}

	testutil.Equal(t, "sysDescr", def.defName(), "defName()")
}

func TestOidDefinitionOid(t *testing.T) {
	t.Run("returns oid when present", func(t *testing.T) {
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
			&module.OidComponentNumber{Value: 3},
		}, types.Synthetic)

		def := oidDefinition{
			mod:  &module.Module{Name: "TEST-MIB"},
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}
		got := def.oid()
		testutil.NotNil(t, got, "expected non-nil oid")
		testutil.Len(t, got.Components, 2, "components")
	})

	t.Run("returns nil for typedef", func(t *testing.T) {
		def := oidDefinition{
			mod:  &module.Module{Name: "TEST-MIB"},
			def:  &module.TypeDef{DefBase: module.DefBase{Name: "MyType"}},
			kind: defValueAssignment,
		}
		testutil.Nil(t, def.oid(), "expected nil oid for TypeDef")
	})
}

func TestSmiGlobalOidRoots(t *testing.T) {
	expected := []string{
		"internet", "directory", "mgmt", "mib-2", "transmission",
		"experimental", "private", "enterprises", "security",
		"snmpV2", "snmpDomains", "snmpProxys", "snmpModules",
		"zeroDotZero", "snmp",
	}

	for _, name := range expected {
		_, ok := smiGlobalOidRoots[name]
		testutil.True(t, ok, "expected %q in smiGlobalOidRoots", name)
	}

	_, ok := smiGlobalOidRoots["iso"]
	testutil.False(t, ok, "iso should not be in smiGlobalOidRoots (it is a well-known root)")
}

func TestCollectOidDefinitionsKindMapping(t *testing.T) {
	// Verify that each definition type maps to the correct definitionKind.
	mod := &module.Module{Name: "TEST-MIB"}

	oid := module.NewOidAssignment([]module.OidComponent{
		&module.OidComponentNumber{Value: 1},
	}, types.Synthetic)

	mod.Definitions = []module.Definition{
		&module.ObjectType{DefBase: module.DefBase{Name: "obj"}, Oid: oid},
		&module.ModuleIdentity{DefBase: module.DefBase{Name: "modId"}, Oid: oid},
		&module.ObjectIdentity{DefBase: module.DefBase{Name: "objId"}, Oid: oid},
		&module.Notification{DefBase: module.DefBase{Name: "notif"}, Oid: &oid},
		&module.ValueAssignment{DefBase: module.DefBase{Name: "val"}, Oid: oid},
		&module.ObjectGroup{DefBase: module.DefBase{Name: "grp"}, Oid: oid},
		&module.NotificationGroup{DefBase: module.DefBase{Name: "notifGrp"}, Oid: oid},
		&module.ModuleCompliance{DefBase: module.DefBase{Name: "comp"}, Oid: oid},
		&module.AgentCapabilities{DefBase: module.DefBase{Name: "cap"}, Oid: oid},
	}

	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
	ctx.modules = []*module.Module{mod}
	defs := collectOidDefinitions(ctx)

	expected := []struct {
		name string
		kind definitionKind
	}{
		{"obj", defObjectType},
		{"modId", defModuleIdentity},
		{"objId", defObjectIdentity},
		{"notif", defNotification},
		{"val", defValueAssignment},
		{"grp", defObjectGroup},
		{"notifGrp", defNotificationGroup},
		{"comp", defModuleCompliance},
		{"cap", defAgentCapabilities},
	}

	testutil.Len(t, defs.oidDefs, len(expected), "oid defs")

	for i, exp := range expected {
		d := defs.oidDefs[i]
		testutil.Equal(t, exp.name, d.defName(), "[] name")
		testutil.Equal(t, exp.kind, d.kind, "[] kind")
	}
}

func TestResolveOids(t *testing.T) {
	t.Run("resolves qualified dependencies across modules", func(t *testing.T) {
		parentMod := &module.Module{
			Name:     "PARENT-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ValueAssignment{
					DefBase: module.DefBase{Name: "parentRoot"},
					Oid: module.NewOidAssignment([]module.OidComponent{
						&module.OidComponentName{NameValue: "iso"},
						&module.OidComponentNumber{Value: 3},
						&module.OidComponentNumber{Value: 6},
						&module.OidComponentNumber{Value: 1},
						&module.OidComponentNumber{Value: 4},
						&module.OidComponentNumber{Value: 1},
						&module.OidComponentNumber{Value: 55555},
					}, types.Span{}),
				},
			},
		}

		childMod := &module.Module{
			Name:     "CHILD-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ValueAssignment{
					DefBase: module.DefBase{Name: "childNode"},
					Oid: module.NewOidAssignment([]module.OidComponent{
						&module.OidComponentQualifiedName{ModuleValue: "PARENT-MIB", NameValue: "parentRoot"},
						&module.OidComponentNumber{Value: 7},
					}, types.Span{}),
				},
			},
		}

		ctx := newTestContextForModules(VerboseConfig(), parentMod, childMod)

		resolveOids(ctx)

		parentNode, ok := ctx.LookupNodeForModule(parentMod, "parentRoot")
		testutil.True(t, ok, "parentRoot should resolve")
		testutil.True(t, parentNode.OID().Equal(OID{1, 3, 6, 1, 4, 1, 55555}), "parentRoot OID")

		childNode, ok := ctx.LookupNodeForModule(childMod, "childNode")
		testutil.True(t, ok, "childNode should resolve")
		testutil.True(t, childNode.OID().Equal(OID{1, 3, 6, 1, 4, 1, 55555, 7}), "childNode OID")
		noDiag(t, ctx.Diagnostics(), types.DiagOidOrphan, types.DiagOidRecursive)
	})

	t.Run("resolves trap types after OID definitions", func(t *testing.T) {
		mod := &module.Module{
			Name:     "TRAP-MIB",
			Language: types.LanguageSMIv1,
			Definitions: []module.Definition{
				&module.ValueAssignment{
					DefBase: module.DefBase{Name: "vendorEnterprise"},
					Oid: module.NewOidAssignment([]module.OidComponent{
						&module.OidComponentName{NameValue: "iso"},
						&module.OidComponentNumber{Value: 3},
						&module.OidComponentNumber{Value: 6},
						&module.OidComponentNumber{Value: 1},
						&module.OidComponentNumber{Value: 4},
						&module.OidComponentNumber{Value: 1},
						&module.OidComponentNumber{Value: 424242},
					}, types.Span{}),
				},
				&module.Notification{
					DefBase:  module.DefBase{Name: "vendorTrap"},
					TrapInfo: &module.TrapInfo{Enterprise: "vendorEnterprise", TrapNumber: 11},
				},
			},
		}

		ctx := newTestContextForModules(VerboseConfig(), mod)

		resolveOids(ctx)

		enterpriseNode, ok := ctx.LookupNodeForModule(mod, "vendorEnterprise")
		testutil.True(t, ok, "vendorEnterprise should resolve")
		testutil.True(t, enterpriseNode.OID().Equal(OID{1, 3, 6, 1, 4, 1, 424242}), "vendorEnterprise OID")

		trapNode, ok := ctx.LookupNodeForModule(mod, "vendorTrap")
		testutil.True(t, ok, "vendorTrap should resolve")
		testutil.True(t, trapNode.OID().Equal(OID{1, 3, 6, 1, 4, 1, 424242, 0, 11}), "vendorTrap OID")
		testutil.Equal(t, KindNotification, trapNode.Kind(), "vendorTrap kind")
		noDiag(t, ctx.Diagnostics(), types.DiagOidOrphan, types.DiagOidRecursive)
	})

	t.Run("records recursive diagnostics for graph cycles", func(t *testing.T) {
		mod := &module.Module{
			Name:     "CYCLE-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ValueAssignment{
					DefBase: module.DefBase{Name: "nodeA"},
					Oid: module.NewOidAssignment([]module.OidComponent{
						&module.OidComponentName{NameValue: "nodeB"},
						&module.OidComponentNumber{Value: 1},
					}, types.Span{}),
				},
				&module.ValueAssignment{
					DefBase: module.DefBase{Name: "nodeB"},
					Oid: module.NewOidAssignment([]module.OidComponent{
						&module.OidComponentName{NameValue: "nodeA"},
						&module.OidComponentNumber{Value: 2},
					}, types.Span{}),
				},
			},
		}

		ctx := newTestContextForModules(VerboseConfig(), mod)

		resolveOids(ctx)

		requireDiagCount(t, ctx.Diagnostics(), types.DiagOidRecursive, 2)
		_, ok := ctx.LookupNodeForModule(mod, "nodeA")
		testutil.False(t, ok, "nodeA should remain unresolved")
		_, ok = ctx.LookupNodeForModule(mod, "nodeB")
		testutil.False(t, ok, "nodeB should remain unresolved")
	})
}

func TestTrapTypeRef(t *testing.T) {
	notif := &module.Notification{
		DefBase:  module.DefBase{Name: "myTrap", Span: types.Span{Start: 10, End: 20}},
		TrapInfo: &module.TrapInfo{Enterprise: "enterprises", TrapNumber: 5},
	}
	ref := trapTypeRef{mod: &module.Module{Name: "TEST-MIB"}, notif: notif}

	testutil.Equal(t, "myTrap", ref.defName(), "defName()")

	enterprise, trapNum, span, ok := ref.trapInfo()
	testutil.True(t, ok, "expected ok = true")
	testutil.Equal(t, "enterprises", enterprise, "enterprise")
	testutil.Equal(t, 5, trapNum, "trapNumber")
	testutil.Equal(t, types.ByteOffset(10), span.Start, "span.Start")
	testutil.Equal(t, types.ByteOffset(20), span.End, "span.End")
}

func TestTrapTypeRefNilTrapInfo(t *testing.T) {
	notif := &module.Notification{DefBase: module.DefBase{Name: "noTrap"}}
	ref := trapTypeRef{mod: &module.Module{Name: "TEST-MIB"}, notif: notif}

	_, _, _, ok := ref.trapInfo()
	testutil.False(t, ok, "expected ok = false for nil TrapInfo")
}

func TestFinalizeModuleIdentityOIDOnlySetForPreferred(t *testing.T) {
	// When two modules define MODULE-IDENTITY at the same OID node,
	// only the preferred module should have SetOID called.

	v2Src := &module.Module{Name: "NEW-MIB", Language: types.LanguageSMIv2}
	v1Src := &module.Module{Name: "OLD-MIB", Language: types.LanguageSMIv1}
	ctx := newTestContextForModules(DefaultConfig(), v2Src, v1Src)
	v2Mod := ctx.moduleToResolved[v2Src]
	v1Mod := ctx.moduleToResolved[v1Src]

	node := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 2)

	oid := module.NewOidAssignment([]module.OidComponent{
		&module.OidComponentNumber{Value: 1},
	}, types.Synthetic)

	// First: finalize the preferred module (SMIv2) - should get OID
	v2Def := oidDefinition{
		mod:  v2Src,
		def:  &module.ModuleIdentity{DefBase: module.DefBase{Name: "newMIB"}, Oid: oid},
		kind: defModuleIdentity,
	}
	finalizeOidDefinition(ctx, v2Def, node, "newMIB")

	testutil.NotNil(t, v2Mod.OID(), "preferred module should have OID set")

	// Second: finalize the non-preferred module (SMIv1) at the same node
	v1Def := oidDefinition{
		mod:  v1Src,
		def:  &module.ModuleIdentity{DefBase: module.DefBase{Name: "oldMIB"}, Oid: oid},
		kind: defModuleIdentity,
	}
	finalizeOidDefinition(ctx, v1Def, node, "oldMIB")

	// The non-preferred module should NOT have its OID set
	testutil.Nil(t, v1Mod.OID(), "non-preferred module should not have OID set, got")

	// The preferred module should still have OID set
	testutil.NotNil(t, v2Mod.OID(), "preferred module OID should still be set")
}

func TestResolveTrapTypeDefinitions_GenericTraps(t *testing.T) {
	// RFC 3584 section 3: generic traps 0-5 with ENTERPRISE snmpTraps
	// resolve to snmpTraps.(trapNumber+1). This matches net-snmp's
	// netsnmp_build_trap_oid() in agent_trap.c.
	//
	// SMIv1-translated MIBs define these as:
	//   coldStart TRAP-TYPE
	//       ENTERPRISE snmpTraps
	//       ::= 0

	srcMod := &module.Module{Name: "TEST-V1-MIB", Language: types.LanguageSMIv1}
	ctx := newTestContextForModules(DefaultConfig(), srcMod)

	// Build snmpTraps node at 1.3.6.1.6.3.1.1.5
	snmpTrapsNode := ctx.mib.Root().
		getOrCreateChild(1). // iso
		getOrCreateChild(3). // org
		getOrCreateChild(6). // dod
		getOrCreateChild(1). // internet
		getOrCreateChild(6). // snmpV2
		getOrCreateChild(3). // snmpModules
		getOrCreateChild(1). // snmpMIB
		getOrCreateChild(1). // snmpMIBObjects
		getOrCreateChild(5)  // snmpTraps
	snmpTrapsNode.setName("snmpTraps")
	ctx.registerModuleNodeSymbol(srcMod, "snmpTraps", snmpTrapsNode)

	// The six generic traps per RFC 3584
	genericTraps := []struct {
		name       string
		trapNumber uint32
		wantOID    OID
	}{
		{"coldStart", 0, OID{1, 3, 6, 1, 6, 3, 1, 1, 5, 1}},
		{"warmStart", 1, OID{1, 3, 6, 1, 6, 3, 1, 1, 5, 2}},
		{"linkDown", 2, OID{1, 3, 6, 1, 6, 3, 1, 1, 5, 3}},
		{"linkUp", 3, OID{1, 3, 6, 1, 6, 3, 1, 1, 5, 4}},
		{"authenticationFailure", 4, OID{1, 3, 6, 1, 6, 3, 1, 1, 5, 5}},
		{"egpNeighborLoss", 5, OID{1, 3, 6, 1, 6, 3, 1, 1, 5, 6}},
	}

	var defs []trapTypeRef
	for _, tt := range genericTraps {
		defs = append(defs, trapTypeRef{
			mod: srcMod,
			notif: &module.Notification{
				DefBase:  module.DefBase{Name: tt.name},
				TrapInfo: &module.TrapInfo{Enterprise: "snmpTraps", TrapNumber: tt.trapNumber},
			},
		})
	}

	resolveTrapTypeDefinitions(ctx, defs)

	for _, tt := range genericTraps {
		node, ok := ctx.LookupNodeForModule(srcMod, tt.name)
		testutil.True(t, ok, "%s: not resolved", tt.name)
		got := node.OID()
		testutil.True(t, got.Equal(tt.wantOID), ": OID")
		testutil.Equal(t, KindNotification, node.Kind(), ": kind")
	}
}

func TestResolveTrapTypeDefinitions_EnterpriseSpecific(t *testing.T) {
	// Enterprise-specific traps use the enterprise.0.trapNumber convention
	// per RFC 3584 section 3.

	srcMod := &module.Module{Name: "VENDOR-MIB", Language: types.LanguageSMIv1}
	ctx := newTestContextForModules(DefaultConfig(), srcMod)

	// Build vendor enterprise node at 1.3.6.1.4.1.9 (e.g. Cisco)
	enterpriseNode := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 4, 1, 9)
	enterpriseNode.setName("cisco")
	ctx.registerModuleNodeSymbol(srcMod, "cisco", enterpriseNode)

	defs := []trapTypeRef{
		{
			mod: srcMod,
			notif: &module.Notification{
				DefBase:  module.DefBase{Name: "vendorTrap"},
				TrapInfo: &module.TrapInfo{Enterprise: "cisco", TrapNumber: 42},
			},
		},
	}

	resolveTrapTypeDefinitions(ctx, defs)

	node, ok := ctx.LookupNodeForModule(srcMod, "vendorTrap")
	testutil.True(t, ok, "vendorTrap: not resolved")
	// enterprise.0.trapNumber = cisco.0.42
	wantOID := OID{1, 3, 6, 1, 4, 1, 9, 0, 42}
	got := node.OID()
	testutil.True(t, got.Equal(wantOID), "vendorTrap: OID")
	testutil.Equal(t, KindNotification, node.Kind(), "vendorTrap: kind")
}

func TestResolveTrapTypeDefinitions_PreferenceGating(t *testing.T) {
	// Two modules define the same TRAP-TYPE OID with different names.
	// The preferred module (SMIv2 > SMIv1) should own name, kind, and module.
	v2Mod := &module.Module{Name: "V2-MIB", Language: types.LanguageSMIv2}
	v1Mod := &module.Module{Name: "V1-MIB", Language: types.LanguageSMIv1}
	ctx := newTestContextForModules(DefaultConfig(), v2Mod, v1Mod)
	v2Resolved := ctx.moduleToResolved[v2Mod]

	// Build enterprise node
	enterpriseNode := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 4, 1, 100)
	enterpriseNode.setName("testEnterprise")
	ctx.registerModuleNodeSymbol(v2Mod, "testEnterprise", enterpriseNode)
	ctx.registerModuleNodeSymbol(v1Mod, "testEnterprise", enterpriseNode)

	// V2 module defines the trap first (preferred)
	defs := []trapTypeRef{
		{
			mod: v2Mod,
			notif: &module.Notification{
				DefBase:  module.DefBase{Name: "preferredName"},
				TrapInfo: &module.TrapInfo{Enterprise: "testEnterprise", TrapNumber: 1},
			},
		},
		{
			mod: v1Mod,
			notif: &module.Notification{
				DefBase:  module.DefBase{Name: "lessPreferredName"},
				TrapInfo: &module.TrapInfo{Enterprise: "testEnterprise", TrapNumber: 1},
			},
		},
	}

	resolveTrapTypeDefinitions(ctx, defs)

	// The trap node at enterprise.0.1 should have the V2 module's name
	trapNode := enterpriseNode.getOrCreateChild(0).getOrCreateChild(1)
	testutil.Equal(t, "preferredName", trapNode.Name(), "name should reflect preferred module")
	testutil.Equal(t, KindNotification, trapNode.Kind(), "kind")
	testutil.Equal(t, v2Resolved, trapNode.Module(), "module should reflect preferred module")
}

func TestIsSnmpTrapsOID(t *testing.T) {
	tests := []struct {
		name string
		oid  OID
		want bool
	}{
		{"snmpTraps", OID{1, 3, 6, 1, 6, 3, 1, 1, 5}, true},
		{"enterprises", OID{1, 3, 6, 1, 4, 1}, false},
		{"snmpTraps child", OID{1, 3, 6, 1, 6, 3, 1, 1, 5, 1}, false},
		{"snmpTraps parent", OID{1, 3, 6, 1, 6, 3, 1, 1}, false},
		{"nil", nil, false},
		{"empty", OID{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Equal(t, tt.want, isSnmpTrapsOID(tt.oid), "isSnmpTrapsOID()")
		})
	}
}

func TestResolveNamedNumberComponent(t *testing.T) {
	t.Run("returns existing node when name is registered", func(t *testing.T) {
		mod := &module.Module{Name: "TEST-MIB"}
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{mod}

		// Register a node under the name "org".
		orgNode := buildOIDPath(ctx.mib.Root(), 1, 3)
		ctx.registerModuleNodeSymbol(mod, "org", orgNode)

		parent := ctx.mib.Root().getOrCreateChild(1)
		def := oidDefinition{
			mod:  mod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: module.NewOidAssignment(nil, types.Synthetic)},
			kind: defValueAssignment,
		}

		got, ok := resolveNamedNumberComponent(ctx, def, parent, "org", 3, false)
		testutil.True(t, ok, "expected ok")
		testutil.Equal(t, orgNode, got, "expected existing node")
	})

	// Named child creation (non-last/isLast behavior) is covered by
	// TestCreateNamedChild which tests createNamedChild directly with
	// more thorough assertions (module, kind, symbol registration).
}

func TestResolveQualifiedNameComponent(t *testing.T) {
	t.Run("returns node from named module", func(t *testing.T) {
		smiMod := &module.Module{Name: "SNMPv2-SMI"}
		defMod := &module.Module{Name: "TEST-MIB"}
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{smiMod, defMod}
		ctx.moduleIndex["SNMPv2-SMI"] = []*module.Module{smiMod}

		entNode := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 4, 1)
		ctx.registerModuleNodeSymbol(smiMod, "enterprises", entNode)

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentQualifiedName{ModuleValue: "SNMPv2-SMI", NameValue: "enterprises"},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  defMod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		got, ok := resolveQualifiedNameComponent(ctx, def, "SNMPv2-SMI", "enterprises", types.Synthetic)
		testutil.True(t, ok, "expected ok")
		testutil.Equal(t, entNode, got, "expected node from named module")
	})

	t.Run("records unresolved when not found", func(t *testing.T) {
		defMod := &module.Module{Name: "TEST-MIB"}
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{defMod}

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentQualifiedName{ModuleValue: "UNKNOWN-MIB", NameValue: "foo"},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  defMod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		got, ok := resolveQualifiedNameComponent(ctx, def, "UNKNOWN-MIB", "foo", types.Synthetic)
		testutil.False(t, ok, "expected false for missing module")
		testutil.Nil(t, got, "expected nil node")

		diags := ctx.Diagnostics()
		testutil.Len(t, diags, 1, "expected 1 diagnostic")
		testutil.Equal(t, types.DiagOidOrphan, diags[0].Code, "diagnostic code")
		testutil.Contains(t, diags[0].Message, "UNKNOWN-MIB.foo", "diagnostic should reference qualified name")
	})
}

func TestResolveQualifiedNamedNumberComponent(t *testing.T) {
	t.Run("returns node from named module", func(t *testing.T) {
		smiMod := &module.Module{Name: "SNMPv2-SMI"}
		defMod := &module.Module{Name: "TEST-MIB"}
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{smiMod, defMod}
		ctx.moduleIndex["SNMPv2-SMI"] = []*module.Module{smiMod}

		entNode := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 4, 1)
		ctx.registerModuleNodeSymbol(smiMod, "enterprises", entNode)

		parent := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 4)
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentQualifiedNamedNumber{ModuleValue: "SNMPv2-SMI", NameValue: "enterprises", NumberValue: 1},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  defMod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		got, ok := resolveQualifiedNamedNumberComponent(ctx, def, parent, "SNMPv2-SMI", "enterprises", 1, false)
		testutil.True(t, ok, "expected ok")
		testutil.Equal(t, entNode, got, "expected node from named module")

		// Verify symbol was registered in defMod.
		regNode, regOk := ctx.LookupNodeForModule(defMod, "enterprises")
		testutil.True(t, regOk, "expected symbol registered in defMod")
		testutil.Equal(t, entNode, regNode, "registered node")
	})

	t.Run("falls back to createNamedChild when not found", func(t *testing.T) {
		defMod := &module.Module{Name: "TEST-MIB"}
		resolvedMod := newModule("TEST-MIB")
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{defMod}
		ctx.moduleToResolved[defMod] = resolvedMod

		parent := ctx.mib.Root().getOrCreateChild(1)
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentQualifiedNamedNumber{ModuleValue: "UNKNOWN-MIB", NameValue: "foo", NumberValue: 7},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  defMod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		got, ok := resolveQualifiedNamedNumberComponent(ctx, def, parent, "UNKNOWN-MIB", "foo", 7, false)
		testutil.True(t, ok, "expected ok via numeric fallback")
		testutil.Equal(t, uint32(7), got.Arc(), "arc")
		testutil.Equal(t, "foo", got.Name(), "name should be set for non-last component")
	})
}

func TestCreateNamedChild(t *testing.T) {
	t.Run("non-last sets name and module and kind", func(t *testing.T) {
		mod := &module.Module{Name: "TEST-MIB"}
		resolvedMod := newModule("TEST-MIB")
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{mod}
		ctx.moduleToResolved[mod] = resolvedMod

		parent := ctx.mib.Root().getOrCreateChild(1)
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNamedNumber{NameValue: "org", NumberValue: 3},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  mod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		child, ok := createNamedChild(ctx, def, parent, "org", 3, false)
		testutil.True(t, ok, "expected ok")
		testutil.Equal(t, uint32(3), child.Arc(), "arc")
		testutil.Equal(t, "org", child.Name(), "name")
		testutil.Equal(t, resolvedMod, child.Module(), "module")
		testutil.Equal(t, KindNode, child.Kind(), "kind")

		// Verify symbol registered in module scope.
		regNode, regOk := ctx.LookupNodeForModule(mod, "org")
		testutil.True(t, regOk, "expected symbol registered")
		testutil.Equal(t, child, regNode, "registered node")
	})

	t.Run("last component does not set name or kind", func(t *testing.T) {
		mod := &module.Module{Name: "TEST-MIB"}
		resolvedMod := newModule("TEST-MIB")
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{mod}
		ctx.moduleToResolved[mod] = resolvedMod

		parent := ctx.mib.Root().getOrCreateChild(1)
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNamedNumber{NameValue: "leaf", NumberValue: 5},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  mod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		child, ok := createNamedChild(ctx, def, parent, "leaf", 5, true)
		testutil.True(t, ok, "expected ok")
		testutil.Equal(t, uint32(5), child.Arc(), "arc")
		testutil.Equal(t, "", child.Name(), "name should not be set for last component")
		testutil.Equal(t, KindInternal, child.Kind(), "kind should remain Internal for last component")
	})

	t.Run("does not override non-Internal kind", func(t *testing.T) {
		mod := &module.Module{Name: "TEST-MIB"}
		resolvedMod := newModule("TEST-MIB")
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{mod}
		ctx.moduleToResolved[mod] = resolvedMod

		parent := ctx.mib.Root().getOrCreateChild(1)
		// Pre-create child and set its kind to something other than Internal.
		existing := parent.getOrCreateChild(3)
		existing.setKind(KindScalar)

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNamedNumber{NameValue: "org", NumberValue: 3},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  mod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		child, ok := createNamedChild(ctx, def, parent, "org", 3, false)
		testutil.True(t, ok, "expected ok")
		testutil.Equal(t, KindScalar, child.Kind(), "kind should not be overridden when not Internal")
	})

	t.Run("nil parent uses root", func(t *testing.T) {
		mod := &module.Module{Name: "TEST-MIB"}
		resolvedMod := newModule("TEST-MIB")
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{mod}
		ctx.moduleToResolved[mod] = resolvedMod

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNamedNumber{NameValue: "iso", NumberValue: 1},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  mod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		child, ok := createNamedChild(ctx, def, nil, "iso", 1, false)
		testutil.True(t, ok, "expected ok")
		testutil.Equal(t, uint32(1), child.Arc(), "arc")
		// Should be the same node as root's child.
		testutil.Equal(t, ctx.mib.Root().getOrCreateChild(1), child, "expected root child")
	})

	t.Run("non-last does not overwrite preferred owner", func(t *testing.T) {
		// A base module owns a node. A vendor module traversing through it
		// as an intermediate path component should not take ownership.
		baseMod := &module.Module{Name: "SNMPv2-SMI", Language: types.LanguageSMIv2}
		vendorMod := &module.Module{Name: "VENDOR-MIB", Language: types.LanguageSMIv2}
		baseResolved := newModule("SNMPv2-SMI")
		vendorResolved := newModule("VENDOR-MIB")

		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{baseMod, vendorMod}
		ctx.moduleToResolved[baseMod] = baseResolved
		ctx.moduleToResolved[vendorMod] = vendorResolved
		ctx.resolvedToModule[baseResolved] = baseMod
		ctx.resolvedToModule[vendorResolved] = vendorMod

		parent := ctx.mib.Root().getOrCreateChild(1)
		// Base module owns "org" at arc 3
		existing := parent.getOrCreateChild(3)
		existing.setName("org")
		existing.setModule(baseResolved)
		existing.setKind(KindNode)

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNamedNumber{NameValue: "org", NumberValue: 3},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  vendorMod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "vendorRoot"}, Oid: oid},
			kind: defValueAssignment,
		}

		child, ok := createNamedChild(ctx, def, parent, "org", 3, false)
		testutil.True(t, ok, "expected ok")
		// Base module should retain ownership
		testutil.Equal(t, "org", child.Name(), "name should not be overwritten")
		testutil.Equal(t, baseResolved, child.Module(), "base module should retain ownership")
	})
}

func TestResolveOidComponentDispatch(t *testing.T) {
	t.Run("NamedNumber dispatches correctly", func(t *testing.T) {
		mod := &module.Module{Name: "TEST-MIB"}
		resolvedMod := newModule("TEST-MIB")
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{mod}
		ctx.moduleToResolved[mod] = resolvedMod

		parent := ctx.mib.Root().getOrCreateChild(1)
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNamedNumber{NameValue: "org", NumberValue: 3},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  mod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		comp := &module.OidComponentNamedNumber{NameValue: "org", NumberValue: 3}
		got, ok := resolveOidComponent(ctx, def, parent, comp, false)
		testutil.True(t, ok, "expected ok")
		testutil.Equal(t, uint32(3), got.Arc(), "arc")
	})

	t.Run("QualifiedName dispatches correctly", func(t *testing.T) {
		smiMod := &module.Module{Name: "SNMPv2-SMI"}
		defMod := &module.Module{Name: "TEST-MIB"}
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{smiMod, defMod}
		ctx.moduleIndex["SNMPv2-SMI"] = []*module.Module{smiMod}

		entNode := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 4, 1)
		ctx.registerModuleNodeSymbol(smiMod, "enterprises", entNode)

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentQualifiedName{ModuleValue: "SNMPv2-SMI", NameValue: "enterprises"},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  defMod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		comp := &module.OidComponentQualifiedName{ModuleValue: "SNMPv2-SMI", NameValue: "enterprises"}
		got, ok := resolveOidComponent(ctx, def, nil, comp, false)
		testutil.True(t, ok, "expected ok")
		testutil.Equal(t, entNode, got, "expected enterprises node")
	})

	t.Run("QualifiedNamedNumber dispatches correctly", func(t *testing.T) {
		smiMod := &module.Module{Name: "SNMPv2-SMI"}
		defMod := &module.Module{Name: "TEST-MIB"}
		resolvedDefMod := newModule("TEST-MIB")
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{smiMod, defMod}
		ctx.moduleIndex["SNMPv2-SMI"] = []*module.Module{smiMod}
		ctx.moduleToResolved[defMod] = resolvedDefMod

		entNode := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 4, 1)
		ctx.registerModuleNodeSymbol(smiMod, "enterprises", entNode)

		parent := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 4)
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentQualifiedNamedNumber{ModuleValue: "SNMPv2-SMI", NameValue: "enterprises", NumberValue: 1},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  defMod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		comp := &module.OidComponentQualifiedNamedNumber{ModuleValue: "SNMPv2-SMI", NameValue: "enterprises", NumberValue: 1}
		got, ok := resolveOidComponent(ctx, def, parent, comp, false)
		testutil.True(t, ok, "expected ok")
		testutil.Equal(t, entNode, got, "expected enterprises node")
	})
}

func TestResolveNameComponentBranches(t *testing.T) {
	t.Run("well-known root iso", func(t *testing.T) {
		mod := &module.Module{Name: "TEST-MIB"}
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{mod}

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentName{NameValue: "iso"},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  mod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		got, ok := resolveNameComponent(ctx, def, "iso", types.Synthetic)
		testutil.True(t, ok, "expected ok for well-known root")
		testutil.Equal(t, uint32(1), got.Arc(), "iso should be arc 1")
	})

	t.Run("constrained SMI global fallback", func(t *testing.T) {
		smiMod := &module.Module{Name: "SNMPv2-SMI"}
		defMod := &module.Module{Name: "VENDOR-MIB"}
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{smiMod, defMod}
		ctx.moduleIndex["SNMPv2-SMI"] = []*module.Module{smiMod}

		entNode := buildOIDPath(ctx.mib.Root(), 1, 3, 6, 1, 4, 1)
		ctx.registerModuleNodeSymbol(smiMod, "enterprises", entNode)

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentName{NameValue: "enterprises"},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  defMod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		got, ok := resolveNameComponent(ctx, def, "enterprises", types.Synthetic)
		testutil.True(t, ok, "expected ok in Normal+")
		testutil.Equal(t, entNode, got, "expected enterprises node")
	})

	t.Run("strict mode rejects unimported SMI global", func(t *testing.T) {
		defMod := &module.Module{Name: "VENDOR-MIB"}
		ctx := newResolverContext(nil, ResolverStrict, VerboseConfig())
		ctx.modules = []*module.Module{defMod}

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentName{NameValue: "enterprises"},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  defMod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		_, ok := resolveNameComponent(ctx, def, "enterprises", types.Synthetic)
		testutil.False(t, ok, "expected false in strict mode")

		diags := ctx.Diagnostics()
		testutil.Len(t, diags, 1, "expected 1 diagnostic")
		testutil.Equal(t, types.DiagOidOrphan, diags[0].Code, "diagnostic code")
	})

	t.Run("unknown name records unresolved", func(t *testing.T) {
		defMod := &module.Module{Name: "TEST-MIB"}
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		ctx.modules = []*module.Module{defMod}

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentName{NameValue: "totallyUnknown"},
		}, types.Synthetic)
		def := oidDefinition{
			mod:  defMod,
			def:  &module.ValueAssignment{DefBase: module.DefBase{Name: "test"}, Oid: oid},
			kind: defValueAssignment,
		}

		_, ok := resolveNameComponent(ctx, def, "totallyUnknown", types.Synthetic)
		testutil.False(t, ok, "expected false for unknown name")

		diags := ctx.Diagnostics()
		testutil.Len(t, diags, 1, "expected 1 diagnostic")
		testutil.Contains(t, diags[0].Message, "totallyUnknown", "diagnostic should reference name")
	})
}

func TestOidReuse_DifferentValueAssignments(t *testing.T) {
	// Two VALUE-ASSIGNMENTs at the same OID with different names.
	// Should emit oid-reuse (warning) since VALUE-ASSIGNMENT is not a registered kind.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "firstNode"},
				Oid: module.NewOidAssignment([]module.OidComponent{
					&module.OidComponentName{NameValue: "enterprises"},
					&module.OidComponentNumber{Value: 99999},
				}, types.Span{}),
			},
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "secondNode"},
				Oid: module.NewOidAssignment([]module.OidComponent{
					&module.OidComponentName{NameValue: "enterprises"},
					&module.OidComponentNumber{Value: 99999},
				}, types.Span{}),
			},
		},
	}

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagOidReuse)
}

func TestOidRegistered_DifferentObjectTypes(t *testing.T) {
	// Two OBJECT-TYPEs at the same OID with different names.
	// Should emit oid-registered (severe) since the first is a registered kind.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid: module.NewOidAssignment([]module.OidComponent{
					&module.OidComponentName{NameValue: "enterprises"},
					&module.OidComponentNumber{Value: 99999},
				}, types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "firstObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:  types.StatusCurrent,
				Oid: module.NewOidAssignment([]module.OidComponent{
					&module.OidComponentName{NameValue: "testRoot"},
					&module.OidComponentNumber{Value: 1},
				}, types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "secondObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:  types.StatusCurrent,
				Oid: module.NewOidAssignment([]module.OidComponent{
					&module.OidComponentName{NameValue: "testRoot"},
					&module.OidComponentNumber{Value: 1},
				}, types.Span{}),
			},
		},
	}

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagOidRegistered)
}

func TestOidReuse_SameNameNoDiagnostic(t *testing.T) {
	// Two modules defining the same OID with the same name (cross-module overlap).
	// Should NOT emit any oid-reuse/oid-registered diagnostic.
	mod1 := &module.Module{
		Name:     "TEST-MIB-V1",
		Language: types.LanguageSMIv1,
		Imports: []module.Import{
			module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testNode"},
				Oid: module.NewOidAssignment([]module.OidComponent{
					&module.OidComponentName{NameValue: "enterprises"},
					&module.OidComponentNumber{Value: 99999},
				}, types.Span{}),
			},
		},
	}
	mod2 := &module.Module{
		Name:     "TEST-MIB-V2",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testNode"},
				Oid: module.NewOidAssignment([]module.OidComponent{
					&module.OidComponentName{NameValue: "enterprises"},
					&module.OidComponentNumber{Value: 99999},
				}, types.Span{}),
			},
		},
	}

	m := resolveStrict(mod1, mod2)
	noDiag(t, m.Diagnostics(), types.DiagOidReuse, types.DiagOidRegistered)
}

func TestResolveOidsCycleDetection(t *testing.T) {
	t.Run("mutual cycle", func(t *testing.T) {
		mod := &module.Module{
			Name:     "CYCLE-MIB",
			Language: types.LanguageSMIv2,
			Imports: []module.Import{
				module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			},
		}

		oidA := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentName{NameValue: "nodeB"},
			&module.OidComponentNumber{Value: 1},
		}, types.Span{})
		oidB := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentName{NameValue: "nodeA"},
			&module.OidComponentNumber{Value: 2},
		}, types.Span{})

		mod.Definitions = []module.Definition{
			&module.ValueAssignment{DefBase: module.DefBase{Name: "nodeA"}, Oid: oidA},
			&module.ValueAssignment{DefBase: module.DefBase{Name: "nodeB"}, Oid: oidB},
		}

		m := resolveStrict(mod)

		recursiveDiags := diagsByCode(m.Diagnostics(), types.DiagOidRecursive)
		requireDiagCount(t, m.Diagnostics(), types.DiagOidRecursive, 2)
		for _, d := range recursiveDiags {
			testutil.Contains(t, d.Message, "recursive OID", "diagnostic message")
			testutil.Contains(t, d.Message, "cycle", "diagnostic message should mention cycle")
		}
	})

	t.Run("self reference", func(t *testing.T) {
		mod := &module.Module{
			Name:     "SELF-REF-MIB",
			Language: types.LanguageSMIv2,
		}

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentName{NameValue: "selfNode"},
			&module.OidComponentNumber{Value: 1},
		}, types.Span{})

		mod.Definitions = []module.Definition{
			&module.ValueAssignment{DefBase: module.DefBase{Name: "selfNode"}, Oid: oid},
		}

		m := resolveStrict(mod)

		recursiveDiags := diagsByCode(m.Diagnostics(), types.DiagOidRecursive)
		requireDiagCount(t, m.Diagnostics(), types.DiagOidRecursive, 1)
		testutil.Contains(t, recursiveDiags[0].Message, "references itself", "self-reference message")
	})
}

func TestLastSubidZero(t *testing.T) {
	t.Run("object-type with last subid zero emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name:     "TEST-MIB",
			Language: types.LanguageSMIv2,
			Imports: []module.Import{
				module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			},
		}

		// enterprises.1.0 - last subid is zero
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentName{NameValue: "enterprises"},
			&module.OidComponentNumber{Value: 1},
			&module.OidComponentNumber{Value: 0},
		}, types.Span{})

		mod.Definitions = []module.Definition{
			&module.ObjectType{DefBase: module.DefBase{Name: "badObj"}, Oid: oid},
		}

		m := resolveStrict(mod)

		d := hasDiag(t, m.Diagnostics(), types.DiagLastSubidZero)
		testutil.Contains(t, d.Message, "badObj", "diagnostic message")
	})

	t.Run("value-assignment with last subid zero is exempt", func(t *testing.T) {
		mod := &module.Module{
			Name:     "TEST-MIB",
			Language: types.LanguageSMIv2,
			Imports: []module.Import{
				module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			},
		}

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentName{NameValue: "enterprises"},
			&module.OidComponentNumber{Value: 1},
			&module.OidComponentNumber{Value: 0},
		}, types.Span{})

		mod.Definitions = []module.Definition{
			&module.ValueAssignment{DefBase: module.DefBase{Name: "okNode"}, Oid: oid},
		}

		m := resolveStrict(mod)

		noDiag(t, m.Diagnostics(), types.DiagLastSubidZero)
	})

	t.Run("non-zero last subid emits no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name:     "TEST-MIB",
			Language: types.LanguageSMIv2,
			Imports: []module.Import{
				module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			},
		}

		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentName{NameValue: "enterprises"},
			&module.OidComponentNumber{Value: 1},
		}, types.Span{})

		mod.Definitions = []module.Definition{
			&module.ObjectType{DefBase: module.DefBase{Name: "goodObj"}, Oid: oid},
		}

		m := resolveStrict(mod)

		noDiag(t, m.Diagnostics(), types.DiagLastSubidZero)
	})
}

// Ensure we use the graph.Symbol type correctly in tests.
var _ = graph.Symbol{Module: "test", Name: "test"}
