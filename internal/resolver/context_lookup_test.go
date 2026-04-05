package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestLookupInScope(t *testing.T) {
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	nodeX := newTestNode("x")

	tests := []struct {
		name    string
		symbols symbolTable[*model.Node]
		imports symbolTable[*module.Module]
		wantOK  bool
		want    *model.Node
	}{
		{
			name:    "direct",
			symbols: symbolTable[*model.Node]{modA: {"x": nodeX}},
			imports: symbolTable[*module.Module]{},
			wantOK:  true,
			want:    nodeX,
		},
		{
			name:    "import chain",
			symbols: symbolTable[*model.Node]{modB: {"x": nodeX}},
			imports: symbolTable[*module.Module]{modA: {"x": modB}},
			wantOK:  true,
			want:    nodeX,
		},
		{
			name:    "import target lacks symbol",
			symbols: symbolTable[*model.Node]{},
			imports: symbolTable[*module.Module]{modA: {"x": modB}},
			wantOK:  false,
		},
		{
			name:    "not found",
			symbols: symbolTable[*model.Node]{},
			imports: symbolTable[*module.Module]{},
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := lookupInScope(modA, "x",
				tt.symbols,
				tt.imports,
				nil,
			)
			testutil.Equal(t, tt.wantOK, ok, "ok")
			if tt.wantOK {
				testutil.Equal(t, tt.want, got, "result")
			}
		})
	}
}

func TestLookupNode(t *testing.T) {
	ctx := newTestContext()
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	nodeX := newTestNode("x")

	// Register node in B, import from A -> B.
	ctx.nodeSymbols[modB] = map[string]*model.Node{"x": nodeX}
	ctx.importSources[modA] = map[string]*module.Module{"x": modB}

	got, ok := ctx.lookupNode(modA, "x")
	testutil.True(t, ok, "lookupNode: expected ok")
	testutil.Equal(t, nodeX, got, "lookupNode: expected nodeX")

	_, ok = ctx.lookupNode(modA, "y")
	testutil.False(t, ok, "lookupNode: expected false for unknown symbol")
}

func TestLookupObject(t *testing.T) {
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}

	t.Run("prefers current module object", func(t *testing.T) {
		ctx := newTestContext()
		resolvedA := model.NewModule("A")
		resolvedB := model.NewModule("B")
		ctx.moduleToResolved[modA] = resolvedA
		ctx.moduleToResolved[modB] = resolvedB
		ctx.importSources[modA] = map[string]*module.Module{"sharedObj": modB}

		localObj := model.NewObject("sharedObj")
		importedObj := model.NewObject("sharedObj")
		model.AddModuleObject(resolvedA, localObj)
		model.AddModuleObject(resolvedB, importedObj)

		got := ctx.lookupObject(modA, "sharedObj")
		testutil.Equal(t, localObj, got, "lookup should prefer the current module object")
		testutil.Nil(t, ctx.usedImports.forModule(modA), "local lookups should not mark imports used")
	})

	t.Run("resolves imported object and marks usage", func(t *testing.T) {
		ctx := newTestContext()
		resolvedA := model.NewModule("A")
		resolvedB := model.NewModule("B")
		ctx.moduleToResolved[modA] = resolvedA
		ctx.moduleToResolved[modB] = resolvedB
		ctx.importSources[modA] = map[string]*module.Module{"importedObj": modB}

		importedObj := model.NewObject("importedObj")
		model.AddModuleObject(resolvedB, importedObj)

		got := ctx.lookupObject(modA, "importedObj")
		testutil.Equal(t, importedObj, got, "lookup should return imported object")
		testutil.True(t, ctx.usedImports.forModule(modA) != nil, "imported lookup should create usage tracking")
		testutil.True(t, ctx.usedImports.has(modA, "importedObj"), "imported lookup should mark the symbol as used")
	})

	t.Run("returns nil when imported module lacks object", func(t *testing.T) {
		ctx := newTestContext()
		resolvedA := model.NewModule("A")
		resolvedB := model.NewModule("B")
		ctx.moduleToResolved[modA] = resolvedA
		ctx.moduleToResolved[modB] = resolvedB
		ctx.importSources[modA] = map[string]*module.Module{"missingObj": modB}

		got := ctx.lookupObject(modA, "missingObj")
		testutil.Nil(t, got, "lookup should fail when the imported module has no object")
		testutil.Nil(t, ctx.usedImports.forModule(modA), "failed imported lookups should not mark usage")
	})
}

func TestResolveObject(t *testing.T) {
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}

	t.Run("returns object from module scope", func(t *testing.T) {
		ctx := newTestContext()
		resolvedA := model.NewModule("A")
		ctx.moduleToResolved[modA] = resolvedA
		nodeX := newTestNode("x")
		ctx.nodeSymbols[modA] = map[string]*model.Node{"x": nodeX}

		obj := model.NewObject("x")
		model.AddModuleObject(resolvedA, obj)

		got, nodeFound := ctx.resolveObject(modA, "x")
		testutil.Equal(t, obj, got, "should return module-scoped object")
		testutil.True(t, nodeFound, "nodeFound should be true")
	})

	t.Run("returns nodeFound true with nil object when node has no object", func(t *testing.T) {
		ctx := newTestContext()
		resolvedA := model.NewModule("A")
		ctx.moduleToResolved[modA] = resolvedA
		nodeX := newTestNode("x")
		ctx.nodeSymbols[modA] = map[string]*model.Node{"x": nodeX}

		got, nodeFound := ctx.resolveObject(modA, "x")
		testutil.Nil(t, got, "should return nil when node has no object")
		testutil.True(t, nodeFound, "nodeFound should be true when node exists")
	})

	t.Run("falls back to global lookup when permissive", func(t *testing.T) {
		ctx := newResolverContext(nil, model.ResolverPermissive, model.DefaultConfig())
		resolvedA := model.NewModule("A")
		resolvedB := model.NewModule("B")
		ctx.moduleToResolved[modA] = resolvedA
		ctx.moduleToResolved[modB] = resolvedB
		ctx.modules = []*module.Module{modA, modB}

		nodeY := newTestNode("y")
		obj := model.NewObject("y")
		model.SetNodeObject(nodeY, obj)
		ctx.nodeSymbols[modB] = map[string]*model.Node{"y": nodeY}

		got, nodeFound := ctx.resolveObject(modA, "y")
		testutil.Equal(t, obj, got, "should return object via global fallback")
		testutil.True(t, nodeFound, "nodeFound should be true")
	})

	t.Run("does not use global fallback when normal strictness", func(t *testing.T) {
		ctx := newTestContext() // model.ResolverNormal
		resolvedA := model.NewModule("A")
		resolvedB := model.NewModule("B")
		ctx.moduleToResolved[modA] = resolvedA
		ctx.moduleToResolved[modB] = resolvedB
		ctx.modules = []*module.Module{modA, modB}

		nodeY := newTestNode("y")
		obj := model.NewObject("y")
		model.SetNodeObject(nodeY, obj)
		ctx.nodeSymbols[modB] = map[string]*model.Node{"y": nodeY}

		got, nodeFound := ctx.resolveObject(modA, "y")
		testutil.Nil(t, got, "should not find object without global fallback")
		testutil.False(t, nodeFound, "nodeFound should be false")
	})

	t.Run("returns false for completely unknown name", func(t *testing.T) {
		ctx := newTestContext()
		resolvedA := model.NewModule("A")
		ctx.moduleToResolved[modA] = resolvedA

		got, nodeFound := ctx.resolveObject(modA, "nonexistent")
		testutil.Nil(t, got, "should return nil for unknown name")
		testutil.False(t, nodeFound, "nodeFound should be false for unknown name")
	})
}

func TestLookupNodeByModuleName(t *testing.T) {
	ctx := newTestContext()
	modA := &module.Module{Name: "MY-MIB"}
	nodeX := newTestNode("x")

	ctx.moduleIndex["MY-MIB"] = []*module.Module{modA}
	ctx.nodeSymbols[modA] = map[string]*model.Node{"x": nodeX}

	got, ok := ctx.lookupNodeByModuleName("MY-MIB", "x")
	testutil.True(t, ok, "lookupNodeByModuleName: expected ok")
	testutil.Equal(t, nodeX, got, "lookupNodeByModuleName: expected nodeX")

	_, ok = ctx.lookupNodeByModuleName("OTHER-MIB", "x")
	testutil.False(t, ok, "lookupNodeByModuleName: expected false for unknown module")
}

func TestLookupNodeByModuleName_MultipleVersions(t *testing.T) {
	ctx := newTestContext()
	modV1 := &module.Module{Name: "MY-MIB"}
	modV2 := &module.Module{Name: "MY-MIB"}
	nodeX := newTestNode("x")

	// Only the second version has the symbol.
	ctx.moduleIndex["MY-MIB"] = []*module.Module{modV1, modV2}
	ctx.nodeSymbols[modV2] = map[string]*model.Node{"x": nodeX}

	got, ok := ctx.lookupNodeByModuleName("MY-MIB", "x")
	testutil.True(t, ok, "expected to find nodeX in second version")
	testutil.Equal(t, nodeX, got, "expected nodeX in second version")
}

func TestLookupNodeGlobal(t *testing.T) {
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	nodeX := newTestNode("x")
	nodeY := newTestNode("y")
	nodeXdup := newTestNode("x")

	ctx := newResolverContext(nil, model.ResolverNormal, model.DefaultConfig())
	ctx.modules = []*module.Module{modA, modB}
	ctx.nodeSymbols[modA] = map[string]*model.Node{"x": nodeX}
	ctx.nodeSymbols[modB] = map[string]*model.Node{"x": nodeXdup, "y": nodeY}

	got, ok := ctx.lookupNodeGlobal("x")
	testutil.True(t, ok, "lookupNodeGlobal(x): expected ok")
	testutil.Equal(t, nodeX, got, "lookupNodeGlobal(x): first module should win")

	got, ok = ctx.lookupNodeGlobal("y")
	testutil.True(t, ok, "lookupNodeGlobal(y): expected ok")
	testutil.Equal(t, nodeY, got, "lookupNodeGlobal(y): expected nodeY")

	_, ok = ctx.lookupNodeGlobal("z")
	testutil.False(t, ok, "lookupNodeGlobal(z): expected false")
}

func TestResolveTypeForModule(t *testing.T) {
	// Test type lookup via import chain.
	ctx := newTestContext()
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	typeX := model.NewType("MyType")

	ctx.typeSymbols[modB] = map[string]*model.Type{"MyType": typeX}
	ctx.importSources[modA] = map[string]*module.Module{"MyType": modB}

	got, ok := ctx.resolveTypeForModule(modA, "MyType")
	testutil.True(t, ok, "resolveTypeForModule: expected ok")
	testutil.Equal(t, typeX, got, "resolveTypeForModule: expected typeX")
}

func TestResolveTypeForModule_ASN1Fallback(t *testing.T) {
	// ASN.1 primitives should resolve even without explicit import.
	ctx := newTestContext()
	modA := &module.Module{Name: "A"}
	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	intType := model.NewType("INTEGER")

	ctx.snmpv2SMIModule = smiMod
	ctx.typeSymbols[smiMod] = map[string]*model.Type{"INTEGER": intType}

	got, ok := ctx.resolveTypeForModule(modA, "INTEGER")
	testutil.True(t, ok, "expected ASN.1 primitive fallback")
	testutil.Equal(t, intType, got, "expected ASN.1 primitive fallback")
}

func TestTypeLookupFallbacks_Normal(t *testing.T) {
	// In Normal mode, well-known types resolve via both resolveType and
	// resolveTypeForModule without explicit imports.
	ctx := newResolverContext(nil, model.ResolverNormal, model.DefaultConfig())
	modA := &module.Module{Name: "A"}

	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	rfc1155Mod := &module.Module{Name: "RFC1155-SMI"}
	tcMod := &module.Module{Name: "SNMPv2-TC"}

	intType := model.NewType("INTEGER")
	counter32 := model.NewType("Counter32")
	counter := model.NewType("Counter")
	displayString := model.NewType("DisplayString")

	ctx.snmpv2SMIModule = smiMod
	ctx.rfc1155SMIModule = rfc1155Mod
	ctx.snmpv2TCModule = tcMod
	ctx.typeSymbols[smiMod] = map[string]*model.Type{
		"INTEGER":   intType,
		"Counter32": counter32,
	}
	ctx.typeSymbols[rfc1155Mod] = map[string]*model.Type{"Counter": counter}
	ctx.typeSymbols[tcMod] = map[string]*model.Type{"DisplayString": displayString}

	tests := []struct {
		name string
		want *model.Type
	}{
		{"INTEGER", intType},
		{"Counter32", counter32},
		{"Counter", counter},
		{"DisplayString", displayString},
	}
	for _, tt := range tests {
		got, ok := ctx.resolveType(tt.name)
		testutil.True(t, ok, "resolveType(%q): expected ok", tt.name)
		testutil.Equal(t, tt.want, got, "resolveType(%q)", tt.name)

		got, ok = ctx.resolveTypeForModule(modA, tt.name)
		testutil.True(t, ok, "resolveTypeForModule(%q): expected ok", tt.name)
		testutil.Equal(t, tt.want, got, "resolveTypeForModule(%q)", tt.name)
	}
}

func TestTypeLookupFallbacks_Strict(t *testing.T) {
	// In Strict mode, only ASN.1 primitives resolve without import.
	ctx := newResolverContext(nil, model.ResolverStrict, model.VerboseConfig())
	modA := &module.Module{Name: "A"}

	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	intType := model.NewType("INTEGER")
	counter32 := model.NewType("Counter32")

	ctx.snmpv2SMIModule = smiMod
	ctx.typeSymbols[smiMod] = map[string]*model.Type{
		"INTEGER":   intType,
		"Counter32": counter32,
	}

	// ASN.1 primitives resolve via both entry points.
	got, ok := ctx.resolveType("INTEGER")
	testutil.True(t, ok, "resolveType(INTEGER): expected ok in strict mode")
	testutil.Equal(t, intType, got, "resolveType(INTEGER)")

	got, ok = ctx.resolveTypeForModule(modA, "INTEGER")
	testutil.True(t, ok, "resolveTypeForModule(INTEGER): expected ok in strict mode")
	testutil.Equal(t, intType, got, "resolveTypeForModule(INTEGER)")

	// Non-primitives do not resolve via either entry point.
	_, ok = ctx.resolveType("Counter32")
	testutil.False(t, ok, "resolveType(Counter32): expected false in strict mode")

	_, ok = ctx.resolveTypeForModule(modA, "Counter32")
	testutil.False(t, ok, "resolveTypeForModule(Counter32): expected false in strict mode")
}

func TestResolveType_GlobalModuleScan(t *testing.T) {
	// In permissive mode, resolveType scans all modules for unknown types.
	modA := &module.Module{Name: "A"}
	vendorType := model.NewType("VendorSpecialType")

	ctx := newResolverContext(nil, model.ResolverPermissive, model.DefaultConfig())
	ctx.modules = []*module.Module{modA}
	ctx.snmpv2SMIModule = &module.Module{Name: "SNMPv2-SMI"}
	ctx.typeSymbols[modA] = map[string]*model.Type{"VendorSpecialType": vendorType}

	got, ok := ctx.resolveType("VendorSpecialType")
	testutil.True(t, ok, "expected global module scan to find vendor type in permissive mode")
	testutil.Equal(t, vendorType, got, "expected global module scan to find vendor type in permissive mode")
}

func TestResolveType_GlobalModuleScan_StrictRejects(t *testing.T) {
	// In strict mode, resolveType should NOT scan all modules for unknown types.
	// A vendor type that exists in another module should not be discoverable.
	modA := &module.Module{Name: "A"}
	vendorType := model.NewType("VendorSpecialType")

	ctx := newResolverContext(nil, model.ResolverStrict, model.VerboseConfig())
	ctx.modules = []*module.Module{modA}
	ctx.snmpv2SMIModule = &module.Module{Name: "SNMPv2-SMI"}
	ctx.typeSymbols[modA] = map[string]*model.Type{"VendorSpecialType": vendorType}

	_, ok := ctx.resolveType("VendorSpecialType")
	testutil.False(t, ok, "strict mode should not find vendor type via global module scan")
}

func TestCopyUsedImportsToModules_Direct(t *testing.T) {
	importer := &module.Module{Name: "IMPORTER-MIB"}
	source := &module.Module{Name: "SOURCE-MIB"}
	ctx := newTestContextForModules(model.DefaultConfig(), importer, source)

	ctx.usedImports[importer] = map[string]struct{}{
		"usedOne": {},
		"usedTwo": {},
	}
	ctx.usedImports[&module.Module{Name: "UNMAPPED-MIB"}] = map[string]struct{}{
		"ignored": {},
	}

	copyUsedImportsToModules(ctx)

	resolvedImporter := ctx.moduleToResolved[importer]
	testutil.True(t, resolvedImporter.IsImportUsed("usedOne"), "usedOne should be copied to the resolved module")
	testutil.True(t, resolvedImporter.IsImportUsed("usedTwo"), "usedTwo should be copied to the resolved module")
	testutil.False(t, resolvedImporter.IsImportUsed("ignored"), "usage from unmapped modules should be ignored")
}

func TestCopyResolvedImportsToModules_Direct(t *testing.T) {
	importer := &module.Module{Name: "IMPORTER-MIB"}
	source := &module.Module{Name: "SOURCE-MIB"}
	unmapped := &module.Module{Name: "UNMAPPED-MIB"}
	ctx := newTestContextForModules(model.DefaultConfig(), importer, source)

	ctx.importSources[importer] = map[string]*module.Module{
		"goodImport": source,
		"skipImport": unmapped,
	}

	copyResolvedImportsToModules(ctx)

	resolvedImporter := ctx.moduleToResolved[importer]
	resolvedSource := ctx.moduleToResolved[source]
	testutil.Equal(t, resolvedSource, resolvedImporter.ImportSource("goodImport"), "resolved import source")
	testutil.Nil(t, resolvedImporter.ImportSource("skipImport"), "imports without resolved module mappings should be skipped")
}

func TestDiagnosticConfig_Getter(t *testing.T) {
	config := model.DefaultConfig()
	ctx := newResolverContext(nil, model.ResolverNormal, config)
	got := ctx.DiagnosticConfig()
	testutil.Equal(t, config.Reporting, got.Reporting, "model.DiagnosticConfig().Reporting")
}
