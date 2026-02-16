package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/graph"
	"github.com/golangsnmp/gomib/internal/module"
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
			if got != tt.want {
				t.Errorf("wellKnownRootArc(%q) = %d, want %d", tt.name, got, tt.want)
			}
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
		{types.LanguageSPPI, 0},
	}

	for _, tt := range tests {
		t.Run(tt.lang.String(), func(t *testing.T) {
			got := languageRank(tt.lang)
			if got != tt.want {
				t.Errorf("languageRank(%v) = %d, want %d", tt.lang, got, tt.want)
			}
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
		&module.ObjectType{Name: "myObject", Oid: oid},
		&module.ModuleIdentity{Name: "myModId", Oid: oid},
		&module.ObjectIdentity{Name: "myObjId", Oid: oid},
		&module.Notification{Name: "myNotif", Oid: &trapOid},
		&module.Notification{Name: "myTrap", TrapInfo: &module.TrapInfo{Enterprise: "enterprises", TrapNumber: 1}},
		&module.Notification{Name: "emptyNotif"}, // no oid, no trap info - skipped
		&module.ValueAssignment{Name: "myVal", Oid: oid},
		&module.ObjectGroup{Name: "myGrp", Oid: oid},
		&module.NotificationGroup{Name: "myNotifGrp", Oid: oid},
		&module.ModuleCompliance{Name: "myComp", Oid: oid},
		&module.AgentCapabilities{Name: "myCap", Oid: oid},
		&module.TypeDef{Name: "MyType"}, // skipped
	}

	ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
	defs := collectOidDefinitions(ctx)

	// All OID-bearing definitions except TypeDef and the empty notification
	if got := len(defs.oidDefs); got != 9 {
		t.Errorf("got %d oid defs, want 9", got)
	}

	if got := len(defs.trapDefs); got != 1 {
		t.Errorf("got %d trap defs, want 1", got)
	}

	if defs.trapDefs[0].defName() != "myTrap" {
		t.Errorf("trap def name = %q, want %q", defs.trapDefs[0].defName(), "myTrap")
	}
}

func TestCollectOidDefinitionsEmpty(t *testing.T) {
	ctx := newResolverContext(nil, nil, DefaultConfig())
	defs := collectOidDefinitions(ctx)

	if len(defs.oidDefs) != 0 {
		t.Errorf("expected no oid defs, got %d", len(defs.oidDefs))
	}
	if len(defs.trapDefs) != 0 {
		t.Errorf("expected no trap defs, got %d", len(defs.trapDefs))
	}
}

func TestGetOidParentSymbol(t *testing.T) {
	mod := &module.Module{Name: "TEST-MIB"}
	ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())

	makeOidDef := func(components []module.OidComponent) oidDefinition {
		oid := module.NewOidAssignment(components, types.Synthetic)
		return oidDefinition{
			mod:  mod,
			def:  &module.ValueAssignment{Name: "test", Oid: oid},
			kind: defValueAssignment,
		}
	}

	t.Run("nil oid", func(t *testing.T) {
		def := oidDefinition{
			mod:  mod,
			def:  &module.Notification{Name: "n"},
			kind: defNotification,
		}
		_, ok := getOidParentSymbol(ctx, def)
		if ok {
			t.Error("expected false for nil oid")
		}
	})

	t.Run("empty components", func(t *testing.T) {
		def := makeOidDef(nil)
		_, ok := getOidParentSymbol(ctx, def)
		if ok {
			t.Error("expected false for empty components")
		}
	})

	t.Run("OidComponentNumber", func(t *testing.T) {
		def := makeOidDef([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		})
		_, ok := getOidParentSymbol(ctx, def)
		if ok {
			t.Error("expected false for numeric root")
		}
	})

	t.Run("OidComponentName well-known root", func(t *testing.T) {
		def := makeOidDef([]module.OidComponent{
			&module.OidComponentName{NameValue: "iso"},
		})
		_, ok := getOidParentSymbol(ctx, def)
		if ok {
			t.Error("expected false for well-known root name")
		}
	})

	t.Run("OidComponentName local def", func(t *testing.T) {
		localMod := &module.Module{
			Name: "LOCAL-MIB",
			Definitions: []module.Definition{
				&module.ValueAssignment{
					Name: "enterprises",
					Oid: module.NewOidAssignment([]module.OidComponent{
						&module.OidComponentNumber{Value: 1},
					}, types.Synthetic),
				},
			},
		}
		localCtx := newResolverContext([]*module.Module{localMod}, nil, DefaultConfig())
		def := oidDefinition{
			mod: localMod,
			def: &module.ValueAssignment{
				Name: "myNode",
				Oid: module.NewOidAssignment([]module.OidComponent{
					&module.OidComponentName{NameValue: "enterprises"},
					&module.OidComponentNumber{Value: 1},
				}, types.Synthetic),
			},
			kind: defValueAssignment,
		}
		sym, ok := getOidParentSymbol(localCtx, def)
		if !ok {
			t.Fatal("expected true for local definition")
		}
		if sym.Module != "LOCAL-MIB" || sym.Name != "enterprises" {
			t.Errorf("got %v, want {LOCAL-MIB, enterprises}", sym)
		}
	})

	t.Run("OidComponentNamedNumber with known name", func(t *testing.T) {
		localMod := &module.Module{
			Name: "LOCAL-MIB",
			Definitions: []module.Definition{
				&module.ValueAssignment{
					Name: "org",
					Oid: module.NewOidAssignment([]module.OidComponent{
						&module.OidComponentNumber{Value: 3},
					}, types.Synthetic),
				},
			},
		}
		localCtx := newResolverContext([]*module.Module{localMod}, nil, DefaultConfig())
		def := oidDefinition{
			mod: localMod,
			def: &module.ValueAssignment{
				Name: "test",
				Oid: module.NewOidAssignment([]module.OidComponent{
					&module.OidComponentNamedNumber{NameValue: "org", NumberValue: 3},
				}, types.Synthetic),
			},
			kind: defValueAssignment,
		}
		sym, ok := getOidParentSymbol(localCtx, def)
		if !ok {
			t.Fatal("expected true")
		}
		if sym.Name != "org" {
			t.Errorf("got name %q, want %q", sym.Name, "org")
		}
	})

	t.Run("OidComponentNamedNumber with well-known root", func(t *testing.T) {
		def := makeOidDef([]module.OidComponent{
			&module.OidComponentNamedNumber{NameValue: "iso", NumberValue: 1},
		})
		_, ok := getOidParentSymbol(ctx, def)
		if ok {
			t.Error("expected false for well-known root named number")
		}
	})

	t.Run("OidComponentNamedNumber unknown falls back to no dependency", func(t *testing.T) {
		def := makeOidDef([]module.OidComponent{
			&module.OidComponentNamedNumber{NameValue: "unknown", NumberValue: 99},
		})
		_, ok := getOidParentSymbol(ctx, def)
		if ok {
			t.Error("expected false for unknown named number (has numeric fallback)")
		}
	})

	t.Run("OidComponentQualifiedName", func(t *testing.T) {
		def := makeOidDef([]module.OidComponent{
			&module.OidComponentQualifiedName{ModuleValue: "SNMPv2-SMI", NameValue: "enterprises"},
		})
		sym, ok := getOidParentSymbol(ctx, def)
		if !ok {
			t.Fatal("expected true for qualified name")
		}
		if sym.Module != "SNMPv2-SMI" || sym.Name != "enterprises" {
			t.Errorf("got %v, want {SNMPv2-SMI, enterprises}", sym)
		}
	})

	t.Run("OidComponentQualifiedNamedNumber", func(t *testing.T) {
		def := makeOidDef([]module.OidComponent{
			&module.OidComponentQualifiedNamedNumber{ModuleValue: "RFC1155-SMI", NameValue: "private", NumberValue: 4},
		})
		sym, ok := getOidParentSymbol(ctx, def)
		if !ok {
			t.Fatal("expected true for qualified named number")
		}
		if sym.Module != "RFC1155-SMI" || sym.Name != "private" {
			t.Errorf("got %v, want {RFC1155-SMI, private}", sym)
		}
	})
}

func TestCheckSmiv2IdentifierHyphens(t *testing.T) {
	t.Run("SMIv2 with hyphen emits diagnostic", func(t *testing.T) {
		mod := &module.Module{Name: "MY-MIB", Language: types.LanguageSMIv2}
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		}, types.Synthetic)
		defs := []oidDefinition{
			{mod: mod, def: &module.ValueAssignment{Name: "my-object", Oid: oid}, kind: defValueAssignment},
		}

		// Use permissive config so SeverityWarning diagnostics are reported.
		ctx := newResolverContext(nil, nil, PermissiveConfig())
		checkSmiv2IdentifierHyphens(ctx, defs)

		found := false
		for _, d := range ctx.Diagnostics() {
			if d.Code == "identifier-hyphen-smiv2" {
				found = true
				if d.Severity != SeverityWarning {
					t.Errorf("severity = %v, want SeverityWarning", d.Severity)
				}
				if d.Module != "MY-MIB" {
					t.Errorf("module = %q, want %q", d.Module, "MY-MIB")
				}
			}
		}
		if !found {
			t.Error("expected identifier-hyphen-smiv2 diagnostic")
		}
	})

	t.Run("SMIv2 without hyphen emits nothing", func(t *testing.T) {
		mod := &module.Module{Name: "MY-MIB", Language: types.LanguageSMIv2}
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		}, types.Synthetic)
		defs := []oidDefinition{
			{mod: mod, def: &module.ValueAssignment{Name: "myObject", Oid: oid}, kind: defValueAssignment},
		}

		ctx := newResolverContext(nil, nil, DefaultConfig())
		checkSmiv2IdentifierHyphens(ctx, defs)

		if len(ctx.Diagnostics()) != 0 {
			t.Errorf("expected no diagnostics, got %d", len(ctx.Diagnostics()))
		}
	})

	t.Run("SMIv1 with hyphen emits nothing", func(t *testing.T) {
		mod := &module.Module{Name: "MY-MIB", Language: types.LanguageSMIv1}
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		}, types.Synthetic)
		defs := []oidDefinition{
			{mod: mod, def: &module.ValueAssignment{Name: "my-object", Oid: oid}, kind: defValueAssignment},
		}

		ctx := newResolverContext(nil, nil, DefaultConfig())
		checkSmiv2IdentifierHyphens(ctx, defs)

		if len(ctx.Diagnostics()) != 0 {
			t.Errorf("expected no diagnostics for SMIv1, got %d", len(ctx.Diagnostics()))
		}
	})

	t.Run("base module skipped", func(t *testing.T) {
		mod := &module.Module{Name: "SNMPv2-SMI", Language: types.LanguageSMIv2}
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
		}, types.Synthetic)
		defs := []oidDefinition{
			{mod: mod, def: &module.ValueAssignment{Name: "mib-2", Oid: oid}, kind: defValueAssignment},
		}

		ctx := newResolverContext(nil, nil, DefaultConfig())
		checkSmiv2IdentifierHyphens(ctx, defs)

		if len(ctx.Diagnostics()) != 0 {
			t.Errorf("expected no diagnostics for base module, got %d", len(ctx.Diagnostics()))
		}
	})
}

func TestResolveNumericComponent(t *testing.T) {
	t.Run("nil parent creates root-level node", func(t *testing.T) {
		ctx := newTestContext()
		node := resolveNumericComponent(ctx, nil, 1)
		if node == nil {
			t.Fatal("expected non-nil node")
		}
		if node.Arc() != 1 {
			t.Errorf("arc = %d, want 1", node.Arc())
		}
		// The node is a child of the pseudo-root, so its parent is
		// the pseudo-root (not nil). Verify it's the same node that
		// Builder.GetOrCreateRoot returns.
		if node != ctx.Mib.Root().getOrCreateChild(1) {
			t.Error("expected same node as Builder.GetOrCreateRoot(1)")
		}
	})

	t.Run("creates child of existing parent", func(t *testing.T) {
		ctx := newTestContext()
		parent := ctx.Mib.Root().getOrCreateChild(1) // iso
		child := resolveNumericComponent(ctx, parent, 3)
		if child == nil {
			t.Fatal("expected non-nil child")
		}
		if child.Arc() != 3 {
			t.Errorf("arc = %d, want 3", child.Arc())
		}
		if child.IsRoot() {
			t.Error("expected non-root (has parent)")
		}
	})

	t.Run("returns same node on repeat", func(t *testing.T) {
		ctx := newTestContext()
		parent := ctx.Mib.Root().getOrCreateChild(1)
		child1 := resolveNumericComponent(ctx, parent, 3)
		child2 := resolveNumericComponent(ctx, parent, 3)
		if child1 != child2 {
			t.Error("expected same node on repeated getOrCreateChild")
		}
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
			if ok != tt.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && node.Arc() != tt.wantArc {
				t.Errorf("arc = %d, want %d", node.Arc(), tt.wantArc)
			}
		})
	}
}

func TestLookupSmiGlobalOidRoot(t *testing.T) {
	t.Run("returns node when registered in SNMPv2-SMI", func(t *testing.T) {
		smiMod := &module.Module{Name: "SNMPv2-SMI"}
		ctx := newResolverContext([]*module.Module{smiMod}, nil, PermissiveConfig())
		ctx.ModuleIndex["SNMPv2-SMI"] = []*module.Module{smiMod}

		node := ctx.Mib.Root().getOrCreateChild(1).getOrCreateChild(3).getOrCreateChild(6).getOrCreateChild(1)
		ctx.registerModuleNodeSymbol(smiMod, "internet", node)

		got, ok := lookupSmiGlobalOidRoot(ctx, "internet")
		if !ok {
			t.Fatal("expected true")
		}
		if got != node {
			t.Error("expected same node")
		}
	})

	t.Run("returns node when registered in RFC1155-SMI", func(t *testing.T) {
		rfc1155Mod := &module.Module{Name: "RFC1155-SMI"}
		ctx := newResolverContext([]*module.Module{rfc1155Mod}, nil, PermissiveConfig())
		ctx.ModuleIndex["RFC1155-SMI"] = []*module.Module{rfc1155Mod}

		node := ctx.Mib.Root().getOrCreateChild(1).getOrCreateChild(3).getOrCreateChild(6).getOrCreateChild(1)
		ctx.registerModuleNodeSymbol(rfc1155Mod, "internet", node)

		got, ok := lookupSmiGlobalOidRoot(ctx, "internet")
		if !ok {
			t.Fatal("expected true")
		}
		if got != node {
			t.Error("expected same node")
		}
	})

	t.Run("returns false for non-global name", func(t *testing.T) {
		ctx := newTestContext()
		_, ok := lookupSmiGlobalOidRoot(ctx, "myCustomOid")
		if ok {
			t.Error("expected false for non-global name")
		}
	})

	t.Run("returns false when module not loaded", func(t *testing.T) {
		ctx := newTestContext()
		_, ok := lookupSmiGlobalOidRoot(ctx, "internet")
		if ok {
			t.Error("expected false when module not loaded")
		}
	})
}

func TestShouldPreferModule(t *testing.T) {
	t.Run("nil currentMod always prefers new", func(t *testing.T) {
		srcMod := &module.Module{Name: "NEW-MIB", Language: types.LanguageSMIv1}
		newMod := newModule("NEW-MIB")
		ctx := newTestContext()
		ctx.ModuleToResolved = map[*module.Module]*Module{srcMod: newMod}

		if !shouldPreferModule(ctx, newMod, nil, srcMod) {
			t.Error("expected true when currentMod is nil")
		}
	})

	t.Run("nil currentSrcMod prefers new", func(t *testing.T) {
		srcMod := &module.Module{Name: "NEW-MIB", Language: types.LanguageSMIv1}
		newMod := newModule("NEW-MIB")
		currentMod := newModule("OLD-MIB")
		ctx := newTestContext()
		ctx.ModuleToResolved = map[*module.Module]*Module{srcMod: newMod}
		ctx.ResolvedToModule = map[*Module]*module.Module{} // currentMod not mapped

		if !shouldPreferModule(ctx, newMod, currentMod, srcMod) {
			t.Error("expected true when currentSrcMod lookup returns nil")
		}
	})

	t.Run("SMIv2 preferred over SMIv1", func(t *testing.T) {
		newSrc := &module.Module{Name: "NEW-MIB", Language: types.LanguageSMIv2}
		oldSrc := &module.Module{Name: "OLD-MIB", Language: types.LanguageSMIv1}
		newMod := newModule("NEW-MIB")
		oldMod := newModule("OLD-MIB")

		ctx := newTestContext()
		ctx.ModuleToResolved = map[*module.Module]*Module{newSrc: newMod, oldSrc: oldMod}
		ctx.ResolvedToModule = map[*Module]*module.Module{oldMod: oldSrc, newMod: newSrc}

		if !shouldPreferModule(ctx, newMod, oldMod, newSrc) {
			t.Error("expected SMIv2 to be preferred over SMIv1")
		}
	})

	t.Run("SMIv1 not preferred over SMIv2", func(t *testing.T) {
		newSrc := &module.Module{Name: "NEW-MIB", Language: types.LanguageSMIv1}
		oldSrc := &module.Module{Name: "OLD-MIB", Language: types.LanguageSMIv2}
		newMod := newModule("NEW-MIB")
		oldMod := newModule("OLD-MIB")

		ctx := newTestContext()
		ctx.ModuleToResolved = map[*module.Module]*Module{newSrc: newMod, oldSrc: oldMod}
		ctx.ResolvedToModule = map[*Module]*module.Module{oldMod: oldSrc, newMod: newSrc}

		if shouldPreferModule(ctx, newMod, oldMod, newSrc) {
			t.Error("expected SMIv1 NOT to be preferred over SMIv2")
		}
	})

	t.Run("same language uses LAST-UPDATED tiebreaker", func(t *testing.T) {
		newSrc := &module.Module{
			Name:     "NEW-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ModuleIdentity{Name: "newMIB", LastUpdated: "200501010000Z"},
			},
		}
		oldSrc := &module.Module{
			Name:     "OLD-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ModuleIdentity{Name: "oldMIB", LastUpdated: "200001010000Z"},
			},
		}
		newMod := newModule("NEW-MIB")
		oldMod := newModule("OLD-MIB")

		ctx := newTestContext()
		ctx.ModuleToResolved = map[*module.Module]*Module{newSrc: newMod, oldSrc: oldMod}
		ctx.ResolvedToModule = map[*Module]*module.Module{oldMod: oldSrc, newMod: newSrc}

		if !shouldPreferModule(ctx, newMod, oldMod, newSrc) {
			t.Error("expected newer LAST-UPDATED to win")
		}
	})

	t.Run("same language older LAST-UPDATED loses", func(t *testing.T) {
		newSrc := &module.Module{
			Name:     "OLD-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ModuleIdentity{Name: "oldMIB", LastUpdated: "199901010000Z"},
			},
		}
		oldSrc := &module.Module{
			Name:     "NEW-MIB",
			Language: types.LanguageSMIv2,
			Definitions: []module.Definition{
				&module.ModuleIdentity{Name: "newMIB", LastUpdated: "200501010000Z"},
			},
		}
		newMod := newModule("OLD-MIB")
		oldMod := newModule("NEW-MIB")

		ctx := newTestContext()
		ctx.ModuleToResolved = map[*module.Module]*Module{newSrc: newMod, oldSrc: oldMod}
		ctx.ResolvedToModule = map[*Module]*module.Module{oldMod: oldSrc, newMod: newSrc}

		if shouldPreferModule(ctx, newMod, oldMod, newSrc) {
			t.Error("expected older LAST-UPDATED to lose")
		}
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
			resolvedMod := newModule("TEST-MIB")

			ctx := newResolverContext([]*module.Module{srcMod}, nil, DefaultConfig())
			ctx.ModuleToResolved[srcMod] = resolvedMod
			ctx.ResolvedToModule[resolvedMod] = srcMod

			node := ctx.Mib.Root().getOrCreateChild(1).getOrCreateChild(3).getOrCreateChild(6)

			oid := module.NewOidAssignment([]module.OidComponent{
				&module.OidComponentNumber{Value: 1},
			}, types.Synthetic)
			def := oidDefinition{
				mod:  srcMod,
				def:  &module.ValueAssignment{Name: "testNode", Oid: oid},
				kind: tt.kind,
			}

			finalizeOidDefinition(ctx, def, node, "testNode")

			if node.Kind() != tt.wantKind {
				t.Errorf("kind = %v, want %v", node.Kind(), tt.wantKind)
			}
			if node.Name() != "testNode" {
				t.Errorf("name = %q, want %q", node.Name(), "testNode")
			}
			if node.Module() != resolvedMod {
				t.Error("expected module to be set to resolvedMod")
			}
		})
	}
}

func TestOidDefinitionDefName(t *testing.T) {
	oid := module.NewOidAssignment([]module.OidComponent{
		&module.OidComponentNumber{Value: 1},
	}, types.Synthetic)

	def := oidDefinition{
		mod:  &module.Module{Name: "TEST-MIB"},
		def:  &module.ObjectType{Name: "sysDescr", Oid: oid},
		kind: defObjectType,
	}

	if got := def.defName(); got != "sysDescr" {
		t.Errorf("defName() = %q, want %q", got, "sysDescr")
	}
}

func TestOidDefinitionOid(t *testing.T) {
	t.Run("returns oid when present", func(t *testing.T) {
		oid := module.NewOidAssignment([]module.OidComponent{
			&module.OidComponentNumber{Value: 1},
			&module.OidComponentNumber{Value: 3},
		}, types.Synthetic)

		def := oidDefinition{
			mod:  &module.Module{Name: "TEST-MIB"},
			def:  &module.ValueAssignment{Name: "test", Oid: oid},
			kind: defValueAssignment,
		}
		got := def.oid()
		if got == nil {
			t.Fatal("expected non-nil oid")
		}
		if len(got.Components) != 2 {
			t.Errorf("got %d components, want 2", len(got.Components))
		}
	})

	t.Run("returns nil for typedef", func(t *testing.T) {
		def := oidDefinition{
			mod:  &module.Module{Name: "TEST-MIB"},
			def:  &module.TypeDef{Name: "MyType"},
			kind: defValueAssignment,
		}
		if def.oid() != nil {
			t.Error("expected nil oid for TypeDef")
		}
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
		if _, ok := smiGlobalOidRoots[name]; !ok {
			t.Errorf("expected %q in smiGlobalOidRoots", name)
		}
	}

	if _, ok := smiGlobalOidRoots["iso"]; ok {
		t.Error("iso should not be in smiGlobalOidRoots (it is a well-known root)")
	}
}

func TestGetOidParentSymbolPermissiveSmiGlobal(t *testing.T) {
	// In permissive mode, unresolved names that are SMI global OID roots
	// should reference SNMPv2-SMI.
	mod := &module.Module{Name: "VENDOR-MIB"}
	ctx := newResolverContext([]*module.Module{mod}, nil, PermissiveConfig())

	oid := module.NewOidAssignment([]module.OidComponent{
		&module.OidComponentName{NameValue: "enterprises"},
		&module.OidComponentNumber{Value: 42},
	}, types.Synthetic)

	def := oidDefinition{
		mod:  mod,
		def:  &module.ValueAssignment{Name: "vendorRoot", Oid: oid},
		kind: defValueAssignment,
	}

	sym, ok := getOidParentSymbol(ctx, def)
	if !ok {
		t.Fatal("expected true in permissive mode for SMI global root")
	}
	if sym.Module != "SNMPv2-SMI" || sym.Name != "enterprises" {
		t.Errorf("got %v, want {SNMPv2-SMI, enterprises}", sym)
	}
}

func TestGetOidParentSymbolStrictNoSmiGlobal(t *testing.T) {
	// In strict mode, unresolved SMI global names should not be resolved.
	mod := &module.Module{Name: "VENDOR-MIB"}
	ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())

	oid := module.NewOidAssignment([]module.OidComponent{
		&module.OidComponentName{NameValue: "enterprises"},
		&module.OidComponentNumber{Value: 42},
	}, types.Synthetic)

	def := oidDefinition{
		mod:  mod,
		def:  &module.ValueAssignment{Name: "vendorRoot", Oid: oid},
		kind: defValueAssignment,
	}

	_, ok := getOidParentSymbol(ctx, def)
	if ok {
		t.Error("expected false in strict mode for unimported SMI global")
	}
}

func TestCollectOidDefinitionsKindMapping(t *testing.T) {
	// Verify that each definition type maps to the correct definitionKind.
	mod := &module.Module{Name: "TEST-MIB"}

	oid := module.NewOidAssignment([]module.OidComponent{
		&module.OidComponentNumber{Value: 1},
	}, types.Synthetic)

	mod.Definitions = []module.Definition{
		&module.ObjectType{Name: "obj", Oid: oid},
		&module.ModuleIdentity{Name: "modId", Oid: oid},
		&module.ObjectIdentity{Name: "objId", Oid: oid},
		&module.Notification{Name: "notif", Oid: &oid},
		&module.ValueAssignment{Name: "val", Oid: oid},
		&module.ObjectGroup{Name: "grp", Oid: oid},
		&module.NotificationGroup{Name: "notifGrp", Oid: oid},
		&module.ModuleCompliance{Name: "comp", Oid: oid},
		&module.AgentCapabilities{Name: "cap", Oid: oid},
	}

	ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
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

	if len(defs.oidDefs) != len(expected) {
		t.Fatalf("got %d oid defs, want %d", len(defs.oidDefs), len(expected))
	}

	for i, exp := range expected {
		d := defs.oidDefs[i]
		if d.defName() != exp.name {
			t.Errorf("[%d] name = %q, want %q", i, d.defName(), exp.name)
		}
		if d.kind != exp.kind {
			t.Errorf("[%d] kind = %d, want %d", i, d.kind, exp.kind)
		}
	}
}

func TestTrapTypeRef(t *testing.T) {
	notif := &module.Notification{
		Name:     "myTrap",
		TrapInfo: &module.TrapInfo{Enterprise: "enterprises", TrapNumber: 5},
		Span:     types.Span{Start: 10, End: 20},
	}
	ref := trapTypeRef{mod: &module.Module{Name: "TEST-MIB"}, notif: notif}

	if got := ref.defName(); got != "myTrap" {
		t.Errorf("defName() = %q, want %q", got, "myTrap")
	}

	enterprise, trapNum, span, ok := ref.trapInfo()
	if !ok {
		t.Fatal("expected ok = true")
	}
	if enterprise != "enterprises" {
		t.Errorf("enterprise = %q, want %q", enterprise, "enterprises")
	}
	if trapNum != 5 {
		t.Errorf("trapNumber = %d, want 5", trapNum)
	}
	if span.Start != 10 || span.End != 20 {
		t.Errorf("span = %v, want {10, 20}", span)
	}
}

func TestTrapTypeRefNilTrapInfo(t *testing.T) {
	notif := &module.Notification{Name: "noTrap"}
	ref := trapTypeRef{mod: &module.Module{Name: "TEST-MIB"}, notif: notif}

	_, _, _, ok := ref.trapInfo()
	if ok {
		t.Error("expected ok = false for nil TrapInfo")
	}
}

func TestFinalizeModuleIdentityOIDOnlySetForPreferred(t *testing.T) {
	// When two modules define MODULE-IDENTITY at the same OID node,
	// only the preferred module should have SetOID called.

	v2Src := &module.Module{Name: "NEW-MIB", Language: types.LanguageSMIv2}
	v1Src := &module.Module{Name: "OLD-MIB", Language: types.LanguageSMIv1}
	v2Mod := newModule("NEW-MIB")
	v1Mod := newModule("OLD-MIB")

	ctx := newResolverContext([]*module.Module{v2Src, v1Src}, nil, DefaultConfig())
	ctx.ModuleToResolved[v2Src] = v2Mod
	ctx.ModuleToResolved[v1Src] = v1Mod
	ctx.ResolvedToModule[v2Mod] = v2Src
	ctx.ResolvedToModule[v1Mod] = v1Src

	node := ctx.Mib.Root().getOrCreateChild(1).getOrCreateChild(3).getOrCreateChild(6).getOrCreateChild(1).getOrCreateChild(2)

	oid := module.NewOidAssignment([]module.OidComponent{
		&module.OidComponentNumber{Value: 1},
	}, types.Synthetic)

	// First: finalize the preferred module (SMIv2) - should get OID
	v2Def := oidDefinition{
		mod:  v2Src,
		def:  &module.ModuleIdentity{Name: "newMIB", Oid: oid},
		kind: defModuleIdentity,
	}
	finalizeOidDefinition(ctx, v2Def, node, "newMIB")

	if v2Mod.OID() == nil {
		t.Fatal("preferred module should have OID set")
	}

	// Second: finalize the non-preferred module (SMIv1) at the same node
	v1Def := oidDefinition{
		mod:  v1Src,
		def:  &module.ModuleIdentity{Name: "oldMIB", Oid: oid},
		kind: defModuleIdentity,
	}
	finalizeOidDefinition(ctx, v1Def, node, "oldMIB")

	// The non-preferred module should NOT have its OID set
	if v1Mod.OID() != nil {
		t.Errorf("non-preferred module should not have OID set, got %v", v1Mod.OID())
	}

	// The preferred module should still have OID set
	if v2Mod.OID() == nil {
		t.Error("preferred module OID should still be set")
	}
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
	resolvedMod := newModule("TEST-V1-MIB")

	ctx := newResolverContext([]*module.Module{srcMod}, nil, DefaultConfig())
	ctx.ModuleToResolved[srcMod] = resolvedMod
	ctx.ResolvedToModule[resolvedMod] = srcMod

	// Build snmpTraps node at 1.3.6.1.6.3.1.1.5
	snmpTrapsNode := ctx.Mib.Root().
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
				Name:     tt.name,
				TrapInfo: &module.TrapInfo{Enterprise: "snmpTraps", TrapNumber: tt.trapNumber},
			},
		})
	}

	resolveTrapTypeDefinitions(ctx, defs)

	for _, tt := range genericTraps {
		node, ok := ctx.LookupNodeForModule(srcMod, tt.name)
		if !ok {
			t.Errorf("%s: not resolved", tt.name)
			continue
		}
		got := node.OID()
		if !got.Equal(tt.wantOID) {
			t.Errorf("%s: OID = %v, want %v", tt.name, got, tt.wantOID)
		}
		if node.Kind() != KindNotification {
			t.Errorf("%s: kind = %v, want KindNotification", tt.name, node.Kind())
		}
	}
}

func TestResolveTrapTypeDefinitions_EnterpriseSpecific(t *testing.T) {
	// Enterprise-specific traps use the enterprise.0.trapNumber convention
	// per RFC 3584 section 3.

	srcMod := &module.Module{Name: "VENDOR-MIB", Language: types.LanguageSMIv1}
	resolvedMod := newModule("VENDOR-MIB")

	ctx := newResolverContext([]*module.Module{srcMod}, nil, DefaultConfig())
	ctx.ModuleToResolved[srcMod] = resolvedMod
	ctx.ResolvedToModule[resolvedMod] = srcMod

	// Build vendor enterprise node at 1.3.6.1.4.1.9 (e.g. Cisco)
	enterpriseNode := ctx.Mib.Root().
		getOrCreateChild(1). // iso
		getOrCreateChild(3). // org
		getOrCreateChild(6). // dod
		getOrCreateChild(1). // internet
		getOrCreateChild(4). // private
		getOrCreateChild(1). // enterprises
		getOrCreateChild(9)  // cisco
	enterpriseNode.setName("cisco")
	ctx.registerModuleNodeSymbol(srcMod, "cisco", enterpriseNode)

	defs := []trapTypeRef{
		{
			mod: srcMod,
			notif: &module.Notification{
				Name:     "vendorTrap",
				TrapInfo: &module.TrapInfo{Enterprise: "cisco", TrapNumber: 42},
			},
		},
	}

	resolveTrapTypeDefinitions(ctx, defs)

	node, ok := ctx.LookupNodeForModule(srcMod, "vendorTrap")
	if !ok {
		t.Fatal("vendorTrap: not resolved")
	}
	// enterprise.0.trapNumber = cisco.0.42
	wantOID := OID{1, 3, 6, 1, 4, 1, 9, 0, 42}
	got := node.OID()
	if !got.Equal(wantOID) {
		t.Errorf("vendorTrap: OID = %v, want %v", got, wantOID)
	}
	if node.Kind() != KindNotification {
		t.Errorf("vendorTrap: kind = %v, want KindNotification", node.Kind())
	}
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
			if got := isSnmpTrapsOID(tt.oid); got != tt.want {
				t.Errorf("isSnmpTrapsOID(%v) = %v, want %v", tt.oid, got, tt.want)
			}
		})
	}
}

// Ensure we use the graph.Symbol type correctly in tests.
var _ = graph.Symbol{Module: "test", Name: "test"}
