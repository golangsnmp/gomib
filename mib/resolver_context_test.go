package mib

import (
	"sync/atomic"
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

// Test-only helpers for wellKnownTypes classification.
func isASN1Primitive(name string) bool   { return wellKnownTypes[name] == typeClassASN1Primitive }
func isSmiGlobalType(name string) bool   { return wellKnownTypes[name] == typeClassSmiGlobal }
func isSmiV1GlobalType(name string) bool { return wellKnownTypes[name] == typeClassSmiV1Global }
func isSNMPv2TCType(name string) bool    { return wellKnownTypes[name] == typeClassSNMPv2TC }

// testNodeArc is an atomic counter for creating unique test nodes.
// Atomic to avoid races if tests run in parallel.
var testNodeArc atomic.Uint32

// newTestNode creates a named *Node for testing. Each call returns
// a distinct node (unique arc under a shared root).
func newTestNode(name string) *Node {
	arc := testNodeArc.Add(1)
	m := newMib()
	n := m.Root().getOrCreateChild(arc)
	n.setName(name)
	return n
}

func TestRecordUnresolvedSeverityConsistency(t *testing.T) {
	// All RecordUnresolved* methods should emit diagnostics at SeverityError.
	// Unresolved references represent failed symbol resolution regardless of
	// category, so the severity should be uniform.

	mod := &module.Module{Name: "TEST-MIB"}
	span := types.Span{}

	tests := []struct {
		name string
		code string
		emit func(c *resolverContext)
	}{
		{
			name: "import",
			code: "import-not-found",
			emit: func(c *resolverContext) {
				c.RecordUnresolvedImport(mod, "OTHER-MIB", "someSymbol", "not found", span)
			},
		},
		{
			name: "import module not found",
			code: "import-module-not-found",
			emit: func(c *resolverContext) {
				c.RecordUnresolvedImport(mod, "MISSING-MIB", "someSymbol", "module_not_found", span)
			},
		},
		{
			name: "type",
			code: "type-unknown",
			emit: func(c *resolverContext) {
				c.RecordUnresolvedType(mod, "myType", "UnknownType", span)
			},
		},
		{
			name: "oid",
			code: "oid-orphan",
			emit: func(c *resolverContext) {
				c.RecordUnresolvedOid(mod, "myObject", "unknownParent", span)
			},
		},
		{
			name: "index",
			code: "index-unresolved",
			emit: func(c *resolverContext) {
				c.RecordUnresolvedIndex(mod, "myRow", "missingIndex", span)
			},
		},
		{
			name: "notification object",
			code: "objects-unresolved",
			emit: func(c *resolverContext) {
				c.RecordUnresolvedNotificationObject(mod, "myNotif", "missingObject", span)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestContext()
			tt.emit(ctx)

			diags := ctx.Diagnostics()
			diag := hasDiag(t, diags, tt.code)
			testutil.Equal(t, SeverityError, diag.Severity, "diagnostic %q severity", tt.code)
			testutil.Equal(t, "TEST-MIB", diag.Module, "diagnostic %q module", tt.code)
		})
	}
}

func TestIsASN1Primitive(t *testing.T) {
	positives := []string{"INTEGER", "OCTET STRING", "OBJECT IDENTIFIER", "BITS"}
	for _, name := range positives {
		testutil.True(t, isASN1Primitive(name), "isASN1Primitive() = false, want true")
	}

	negatives := []string{
		"Integer32", "Counter32", "DisplayString", "integer",
		"OCTETSTRING", "OBJECT-IDENTIFIER", "", "Counter",
	}
	for _, name := range negatives {
		testutil.False(t, isASN1Primitive(name), "isASN1Primitive(%q) = true, want false", name)
	}
}

func TestIsSmiGlobalType(t *testing.T) {
	positives := []string{
		"Integer32", "Counter32", "Counter64", "Gauge32",
		"Unsigned32", "TimeTicks", "IpAddress", "Opaque",
	}
	for _, name := range positives {
		testutil.True(t, isSmiGlobalType(name), "isSmiGlobalType() = false, want true")
	}

	negatives := []string{
		"INTEGER", "Counter", "Gauge", "DisplayString",
		"integer32", "NetworkAddress", "",
	}
	for _, name := range negatives {
		testutil.False(t, isSmiGlobalType(name), "isSmiGlobalType(%q) = true, want false", name)
	}
}

func TestIsSmiV1GlobalType(t *testing.T) {
	positives := []string{"Counter", "Gauge", "NetworkAddress"}
	for _, name := range positives {
		testutil.True(t, isSmiV1GlobalType(name), "isSmiV1GlobalType() = false, want true")
	}

	negatives := []string{
		"Counter32", "Gauge32", "IpAddress", "INTEGER",
		"counter", "TimeTicks", "",
	}
	for _, name := range negatives {
		testutil.False(t, isSmiV1GlobalType(name), "isSmiV1GlobalType(%q) = true, want false", name)
	}
}

func TestIsSNMPv2TCType(t *testing.T) {
	positives := []string{
		"DisplayString", "TruthValue", "PhysAddress", "MacAddress",
		"RowStatus", "TimeStamp", "TimeInterval", "DateAndTime",
		"StorageType", "TestAndIncr", "AutonomousType",
		"VariablePointer", "RowPointer", "InstancePointer",
		"TDomain", "TAddress",
	}
	for _, name := range positives {
		testutil.True(t, isSNMPv2TCType(name), "isSNMPv2TCType() = false, want true")
	}

	negatives := []string{
		"INTEGER", "Counter32", "IpAddress", "displaystring",
		"Counter", "Gauge32", "",
	}
	for _, name := range negatives {
		testutil.False(t, isSNMPv2TCType(name), "isSNMPv2TCType(%q) = true, want false", name)
	}
}

func TestLookupInModuleScope(t *testing.T) {
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	nodeX := newTestNode("x")

	tests := []struct {
		name    string
		symbols map[*module.Module]map[string]*Node
		imports map[*module.Module]map[string]*module.Module
		wantOK  bool
		want    *Node
	}{
		{
			name:    "direct",
			symbols: map[*module.Module]map[string]*Node{modA: {"x": nodeX}},
			imports: map[*module.Module]map[string]*module.Module{},
			wantOK:  true,
			want:    nodeX,
		},
		{
			name:    "import chain",
			symbols: map[*module.Module]map[string]*Node{modB: {"x": nodeX}},
			imports: map[*module.Module]map[string]*module.Module{modA: {"x": modB}},
			wantOK:  true,
			want:    nodeX,
		},
		{
			name:    "import target lacks symbol",
			symbols: map[*module.Module]map[string]*Node{},
			imports: map[*module.Module]map[string]*module.Module{modA: {"x": modB}},
			wantOK:  false,
		},
		{
			name:    "not found",
			symbols: map[*module.Module]map[string]*Node{},
			imports: map[*module.Module]map[string]*module.Module{},
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := lookupInModuleScope(modA, "x",
				func(m *module.Module) map[string]*Node { return tt.symbols[m] },
				func(m *module.Module) map[string]*module.Module { return tt.imports[m] },
				nil,
			)
			testutil.Equal(t, tt.wantOK, ok, "ok")
			if tt.wantOK {
				testutil.Equal(t, tt.want, got, "result")
			}
		})
	}
}

func TestLookupNodeForModule(t *testing.T) {
	ctx := newTestContext()
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	nodeX := newTestNode("x")

	// Register node in B, import from A -> B.
	ctx.moduleSymbolToNode[modB] = map[string]*Node{"x": nodeX}
	ctx.moduleImports[modA] = map[string]*module.Module{"x": modB}

	got, ok := ctx.LookupNodeForModule(modA, "x")
	testutil.True(t, ok, "LookupNodeForModule: expected ok")
	testutil.Equal(t, nodeX, got, "LookupNodeForModule: expected nodeX")

	_, ok = ctx.LookupNodeForModule(modA, "y")
	testutil.False(t, ok, "LookupNodeForModule: expected false for unknown symbol")
}

func TestLookupObjectInModuleScope(t *testing.T) {
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}

	t.Run("prefers current module object", func(t *testing.T) {
		ctx := newTestContext()
		resolvedA := newModule("A")
		resolvedB := newModule("B")
		ctx.moduleToResolved[modA] = resolvedA
		ctx.moduleToResolved[modB] = resolvedB
		ctx.moduleImports[modA] = map[string]*module.Module{"sharedObj": modB}

		localObj := newObject("sharedObj")
		importedObj := newObject("sharedObj")
		resolvedA.addObject(localObj)
		resolvedB.addObject(importedObj)

		got := ctx.lookupObjectInModuleScope(modA, "sharedObj")
		testutil.Equal(t, localObj, got, "lookup should prefer the current module object")
		testutil.Nil(t, ctx.usedImports[modA], "local lookups should not mark imports used")
	})

	t.Run("resolves imported object and marks usage", func(t *testing.T) {
		ctx := newTestContext()
		resolvedA := newModule("A")
		resolvedB := newModule("B")
		ctx.moduleToResolved[modA] = resolvedA
		ctx.moduleToResolved[modB] = resolvedB
		ctx.moduleImports[modA] = map[string]*module.Module{"importedObj": modB}

		importedObj := newObject("importedObj")
		resolvedB.addObject(importedObj)

		got := ctx.lookupObjectInModuleScope(modA, "importedObj")
		testutil.Equal(t, importedObj, got, "lookup should return imported object")
		testutil.True(t, ctx.usedImports[modA] != nil, "imported lookup should create usage tracking")
		_, ok := ctx.usedImports[modA]["importedObj"]
		testutil.True(t, ok, "imported lookup should mark the symbol as used")
	})

	t.Run("returns nil when imported module lacks object", func(t *testing.T) {
		ctx := newTestContext()
		resolvedA := newModule("A")
		resolvedB := newModule("B")
		ctx.moduleToResolved[modA] = resolvedA
		ctx.moduleToResolved[modB] = resolvedB
		ctx.moduleImports[modA] = map[string]*module.Module{"missingObj": modB}

		got := ctx.lookupObjectInModuleScope(modA, "missingObj")
		testutil.Nil(t, got, "lookup should fail when the imported module has no object")
		testutil.Nil(t, ctx.usedImports[modA], "failed imported lookups should not mark usage")
	})
}

func TestFindScopedObject(t *testing.T) {
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}

	t.Run("returns object from module scope", func(t *testing.T) {
		ctx := newTestContext()
		resolvedA := newModule("A")
		ctx.moduleToResolved[modA] = resolvedA
		nodeX := newTestNode("x")
		ctx.moduleSymbolToNode[modA] = map[string]*Node{"x": nodeX}

		obj := newObject("x")
		resolvedA.addObject(obj)

		got, nodeFound := ctx.findScopedObject(modA, "x")
		testutil.Equal(t, obj, got, "should return module-scoped object")
		testutil.True(t, nodeFound, "nodeFound should be true")
	})

	t.Run("returns nodeFound true with nil object when node has no object", func(t *testing.T) {
		ctx := newTestContext()
		resolvedA := newModule("A")
		ctx.moduleToResolved[modA] = resolvedA
		nodeX := newTestNode("x")
		ctx.moduleSymbolToNode[modA] = map[string]*Node{"x": nodeX}

		got, nodeFound := ctx.findScopedObject(modA, "x")
		testutil.Nil(t, got, "should return nil when node has no object")
		testutil.True(t, nodeFound, "nodeFound should be true when node exists")
	})

	t.Run("falls back to global lookup when permissive", func(t *testing.T) {
		ctx := newResolverContext(nil, ResolverPermissive, DefaultConfig())
		resolvedA := newModule("A")
		resolvedB := newModule("B")
		ctx.moduleToResolved[modA] = resolvedA
		ctx.moduleToResolved[modB] = resolvedB
		ctx.modules = []*module.Module{modA, modB}

		nodeY := newTestNode("y")
		obj := newObject("y")
		nodeY.setObject(obj)
		ctx.moduleSymbolToNode[modB] = map[string]*Node{"y": nodeY}

		got, nodeFound := ctx.findScopedObject(modA, "y")
		testutil.Equal(t, obj, got, "should return object via global fallback")
		testutil.True(t, nodeFound, "nodeFound should be true")
	})

	t.Run("does not use global fallback when normal strictness", func(t *testing.T) {
		ctx := newTestContext() // ResolverNormal
		resolvedA := newModule("A")
		resolvedB := newModule("B")
		ctx.moduleToResolved[modA] = resolvedA
		ctx.moduleToResolved[modB] = resolvedB
		ctx.modules = []*module.Module{modA, modB}

		nodeY := newTestNode("y")
		obj := newObject("y")
		nodeY.setObject(obj)
		ctx.moduleSymbolToNode[modB] = map[string]*Node{"y": nodeY}

		got, nodeFound := ctx.findScopedObject(modA, "y")
		testutil.Nil(t, got, "should not find object without global fallback")
		testutil.False(t, nodeFound, "nodeFound should be false")
	})

	t.Run("returns false for completely unknown name", func(t *testing.T) {
		ctx := newTestContext()
		resolvedA := newModule("A")
		ctx.moduleToResolved[modA] = resolvedA

		got, nodeFound := ctx.findScopedObject(modA, "nonexistent")
		testutil.Nil(t, got, "should return nil for unknown name")
		testutil.False(t, nodeFound, "nodeFound should be false for unknown name")
	})
}

func TestLookupNodeInModule(t *testing.T) {
	ctx := newTestContext()
	modA := &module.Module{Name: "MY-MIB"}
	nodeX := newTestNode("x")

	ctx.moduleIndex["MY-MIB"] = []*module.Module{modA}
	ctx.moduleSymbolToNode[modA] = map[string]*Node{"x": nodeX}

	got, ok := ctx.LookupNodeInModule("MY-MIB", "x")
	testutil.True(t, ok, "LookupNodeInModule: expected ok")
	testutil.Equal(t, nodeX, got, "LookupNodeInModule: expected nodeX")

	_, ok = ctx.LookupNodeInModule("OTHER-MIB", "x")
	testutil.False(t, ok, "LookupNodeInModule: expected false for unknown module")
}

func TestLookupNodeInModule_MultipleVersions(t *testing.T) {
	ctx := newTestContext()
	modV1 := &module.Module{Name: "MY-MIB"}
	modV2 := &module.Module{Name: "MY-MIB"}
	nodeX := newTestNode("x")

	// Only the second version has the symbol.
	ctx.moduleIndex["MY-MIB"] = []*module.Module{modV1, modV2}
	ctx.moduleSymbolToNode[modV2] = map[string]*Node{"x": nodeX}

	got, ok := ctx.LookupNodeInModule("MY-MIB", "x")
	testutil.True(t, ok, "expected to find nodeX in second version")
	testutil.Equal(t, nodeX, got, "expected nodeX in second version")
}

func TestLookupNodeGlobal(t *testing.T) {
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	nodeX := newTestNode("x")
	nodeY := newTestNode("y")
	nodeXdup := newTestNode("x")

	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
	ctx.modules = []*module.Module{modA, modB}
	ctx.moduleSymbolToNode[modA] = map[string]*Node{"x": nodeX}
	ctx.moduleSymbolToNode[modB] = map[string]*Node{"x": nodeXdup, "y": nodeY}

	got, ok := ctx.LookupNodeGlobal("x")
	testutil.True(t, ok, "LookupNodeGlobal(x): expected ok")
	testutil.Equal(t, nodeX, got, "LookupNodeGlobal(x): first module should win")

	got, ok = ctx.LookupNodeGlobal("y")
	testutil.True(t, ok, "LookupNodeGlobal(y): expected ok")
	testutil.Equal(t, nodeY, got, "LookupNodeGlobal(y): expected nodeY")

	_, ok = ctx.LookupNodeGlobal("z")
	testutil.False(t, ok, "LookupNodeGlobal(z): expected false")
}

func TestLookupTypeForModule(t *testing.T) {
	// Test type lookup via import chain.
	ctx := newTestContext()
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	typeX := newType("MyType")

	ctx.moduleSymbolToType[modB] = map[string]*Type{"MyType": typeX}
	ctx.moduleImports[modA] = map[string]*module.Module{"MyType": modB}

	got, ok := ctx.LookupTypeForModule(modA, "MyType")
	testutil.True(t, ok, "LookupTypeForModule: expected ok")
	testutil.Equal(t, typeX, got, "LookupTypeForModule: expected typeX")
}

func TestLookupTypeForModule_ASN1Fallback(t *testing.T) {
	// ASN.1 primitives should resolve even without explicit import.
	ctx := newTestContext()
	modA := &module.Module{Name: "A"}
	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	intType := newType("INTEGER")

	ctx.snmpv2SMIModule = smiMod
	ctx.moduleSymbolToType[smiMod] = map[string]*Type{"INTEGER": intType}

	got, ok := ctx.LookupTypeForModule(modA, "INTEGER")
	testutil.True(t, ok, "expected ASN.1 primitive fallback")
	testutil.Equal(t, intType, got, "expected ASN.1 primitive fallback")
}

func TestTypeLookupFallbacks_Normal(t *testing.T) {
	// In Normal mode, well-known types resolve via both LookupType and
	// LookupTypeForModule without explicit imports.
	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
	modA := &module.Module{Name: "A"}

	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	rfc1155Mod := &module.Module{Name: "RFC1155-SMI"}
	tcMod := &module.Module{Name: "SNMPv2-TC"}

	intType := newType("INTEGER")
	counter32 := newType("Counter32")
	counter := newType("Counter")
	displayString := newType("DisplayString")

	ctx.snmpv2SMIModule = smiMod
	ctx.rfc1155SMIModule = rfc1155Mod
	ctx.snmpv2TCModule = tcMod
	ctx.moduleSymbolToType[smiMod] = map[string]*Type{
		"INTEGER":   intType,
		"Counter32": counter32,
	}
	ctx.moduleSymbolToType[rfc1155Mod] = map[string]*Type{"Counter": counter}
	ctx.moduleSymbolToType[tcMod] = map[string]*Type{"DisplayString": displayString}

	tests := []struct {
		name string
		want *Type
	}{
		{"INTEGER", intType},
		{"Counter32", counter32},
		{"Counter", counter},
		{"DisplayString", displayString},
	}
	for _, tt := range tests {
		got, ok := ctx.LookupType(tt.name)
		testutil.True(t, ok, "LookupType(%q): expected ok", tt.name)
		testutil.Equal(t, tt.want, got, "LookupType(%q)", tt.name)

		got, ok = ctx.LookupTypeForModule(modA, tt.name)
		testutil.True(t, ok, "LookupTypeForModule(%q): expected ok", tt.name)
		testutil.Equal(t, tt.want, got, "LookupTypeForModule(%q)", tt.name)
	}
}

func TestTypeLookupFallbacks_Strict(t *testing.T) {
	// In Strict mode, only ASN.1 primitives resolve without import.
	ctx := newResolverContext(nil, ResolverStrict, VerboseConfig())
	modA := &module.Module{Name: "A"}

	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	intType := newType("INTEGER")
	counter32 := newType("Counter32")

	ctx.snmpv2SMIModule = smiMod
	ctx.moduleSymbolToType[smiMod] = map[string]*Type{
		"INTEGER":   intType,
		"Counter32": counter32,
	}

	// ASN.1 primitives resolve via both entry points.
	got, ok := ctx.LookupType("INTEGER")
	testutil.True(t, ok, "LookupType(INTEGER): expected ok in strict mode")
	testutil.Equal(t, intType, got, "LookupType(INTEGER)")

	got, ok = ctx.LookupTypeForModule(modA, "INTEGER")
	testutil.True(t, ok, "LookupTypeForModule(INTEGER): expected ok in strict mode")
	testutil.Equal(t, intType, got, "LookupTypeForModule(INTEGER)")

	// Non-primitives do not resolve via either entry point.
	_, ok = ctx.LookupType("Counter32")
	testutil.False(t, ok, "LookupType(Counter32): expected false in strict mode")

	_, ok = ctx.LookupTypeForModule(modA, "Counter32")
	testutil.False(t, ok, "LookupTypeForModule(Counter32): expected false in strict mode")
}

func TestLookupType_GlobalModuleScan(t *testing.T) {
	// In permissive mode, LookupType scans all modules for unknown types.
	modA := &module.Module{Name: "A"}
	vendorType := newType("VendorSpecialType")

	ctx := newResolverContext(nil, ResolverPermissive, DefaultConfig())
	ctx.modules = []*module.Module{modA}
	ctx.snmpv2SMIModule = &module.Module{Name: "SNMPv2-SMI"}
	ctx.moduleSymbolToType[modA] = map[string]*Type{"VendorSpecialType": vendorType}

	got, ok := ctx.LookupType("VendorSpecialType")
	testutil.True(t, ok, "expected global module scan to find vendor type in permissive mode")
	testutil.Equal(t, vendorType, got, "expected global module scan to find vendor type in permissive mode")
}

func TestLookupType_GlobalModuleScan_StrictRejects(t *testing.T) {
	// In strict mode, LookupType should NOT scan all modules for unknown types.
	// A vendor type that exists in another module should not be discoverable.
	modA := &module.Module{Name: "A"}
	vendorType := newType("VendorSpecialType")

	ctx := newResolverContext(nil, ResolverStrict, VerboseConfig())
	ctx.modules = []*module.Module{modA}
	ctx.snmpv2SMIModule = &module.Module{Name: "SNMPv2-SMI"}
	ctx.moduleSymbolToType[modA] = map[string]*Type{"VendorSpecialType": vendorType}

	_, ok := ctx.LookupType("VendorSpecialType")
	testutil.False(t, ok, "strict mode should not find vendor type via global module scan")
}

func TestRegisterImport(t *testing.T) {
	ctx := newTestContext()
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}

	ctx.registerImport(modA, "foo", modB)

	imports := ctx.moduleImports[modA]
	testutil.NotNil(t, imports, "expected imports map to be created")
	testutil.Equal(t, modB, imports["foo"], "expected import to point to modB")

	// Register a second import in the same module.
	modC := &module.Module{Name: "C"}
	ctx.registerImport(modA, "bar", modC)
	testutil.Equal(t, modC, ctx.moduleImports[modA]["bar"], "expected second import to point to modC")
}

func TestRegisterModuleNodeSymbol(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "A"}
	node := newTestNode("sysDescr")

	ctx.registerModuleNodeSymbol(mod, "sysDescr", node)

	symbols := ctx.moduleSymbolToNode[mod]
	testutil.NotNil(t, symbols, "expected symbol map to be created")
	testutil.Equal(t, node, symbols["sysDescr"], "expected registered node")

	// Overwrite should succeed.
	node2 := newTestNode("sysDescr")
	ctx.registerModuleNodeSymbol(mod, "sysDescr", node2)
	testutil.Equal(t, node2, ctx.moduleSymbolToNode[mod]["sysDescr"], "expected overwritten node")
}

func TestRegisterModuleTypeSymbol(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "A"}
	typ := newType("MyType")

	ctx.registerModuleTypeSymbol(mod, "MyType", typ)

	symbols := ctx.moduleSymbolToType[mod]
	testutil.NotNil(t, symbols, "expected symbol map to be created")
	testutil.Equal(t, typ, symbols["MyType"], "expected registered type")
}

func TestEmitDiagnostic(t *testing.T) {
	tests := []struct {
		name   string
		config DiagnosticConfig
		code   string // must be a registered diagnostic code
		want   int
	}{
		{
			name:   "default reports error",
			config: DefaultConfig(),
			code:   types.DiagParseError, // SeverityError
			want:   1,
		},
		{
			name:   "default reports minor",
			config: DefaultConfig(),
			code:   types.DiagGroupNotAccessible, // SeverityMinor
			want:   1,
		},
		{
			name:   "default suppresses style",
			config: DefaultConfig(),
			code:   types.DiagIdentifierUnderscore, // SeverityStyle
			want:   0,
		},
		{
			name:   "default suppresses warning",
			config: DefaultConfig(),
			code:   types.DiagIdentifierHyphenSMI, // SeverityWarning
			want:   0,
		},
		{
			name:   "verbose reports warning",
			config: VerboseConfig(),
			code:   types.DiagIdentifierHyphenSMI, // SeverityWarning
			want:   1,
		},
		{
			name: "silent suppresses non-fatal",
			config: DiagnosticConfig{
				Reporting: ReportingSilent,
			},
			code: types.DiagParseError, // SeverityError
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newResolverContext(nil, ResolverNormal, tt.config)
			mod := &module.Module{Name: "MOD"}
			ctx.EmitDiagnostic(tt.code, mod, types.Span{}, "test message")
			got := len(ctx.Diagnostics())
			testutil.Equal(t, tt.want, got, "diagnostics")
		})
	}
}

func TestShouldReport_FatalAlwaysReported(t *testing.T) {
	// No diagnostic codes map to SeverityFatal, but ShouldReport must still
	// pass fatal-severity diagnostics even in silent mode.
	config := DiagnosticConfig{Reporting: ReportingSilent}
	testutil.True(t, config.ShouldReport("any-code", SeverityFatal),
		"fatal should be reported even in silent mode")
}

func TestEmitDiagnostic_IgnoredCode(t *testing.T) {
	config := DiagnosticConfig{
		Reporting: ReportingVerbose,
		Ignore:    []string{"import-*"},
	}
	ctx := newResolverContext(nil, ResolverNormal, config)
	mod := &module.Module{Name: "MOD"}
	ctx.EmitDiagnostic(types.DiagImportNotFound, mod, types.Span{}, "ignored")
	testutil.Len(t, ctx.Diagnostics(), 0, "expected ignored code to produce no diagnostics")

	// Non-matching code should still be reported.
	ctx.EmitDiagnostic(types.DiagParseError, mod, types.Span{}, "not ignored")
	testutil.Len(t, ctx.Diagnostics(), 1, "expected non-ignored code to produce a diagnostic")
}

func TestEmitDiagnostic_Fields(t *testing.T) {
	ctx := newTestContext()
	// Build a source with 10 lines of 10 bytes each (9 chars + newline).
	// Line 10 starts at offset 90, column 5 is offset 94.
	source := make([]byte, 100)
	for i := range source {
		if (i+1)%10 == 0 {
			source[i] = '\n'
		} else {
			source[i] = 'x'
		}
	}
	mod := &module.Module{
		Name:      "TEST-MIB",
		LineTable: types.BuildLineTable(source),
	}
	span := types.NewSpan(94, 95) // line 10, column 5
	ctx.EmitDiagnostic(types.DiagGroupNotAccessible, mod, span, "something happened")

	diags := ctx.Diagnostics()
	testutil.Len(t, diags, 1, "expected 1 diagnostic, got")
	d := diags[0]
	testutil.Equal(t, types.DiagGroupNotAccessible, d.Code, "Code")
	testutil.Equal(t, SeverityMinor, d.Severity, "Severity")
	testutil.Equal(t, "TEST-MIB", d.Module, "Module")
	testutil.Equal(t, 10, d.Line, "Line")
	testutil.Equal(t, 5, d.Column, "Column")
	testutil.Equal(t, "something happened", d.Message, "Message")
}

func TestFinalizeUnresolved(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	span := types.Span{}

	ctx.RecordUnresolvedImport(mod, "OTHER", "sym1", "module_not_found", span)
	ctx.RecordUnresolvedType(mod, "ref1", "UnknownType", span)
	ctx.RecordUnresolvedOid(mod, "obj1", "parent1", span)
	ctx.RecordUnresolvedIndex(mod, "row1", "idx1", span)
	ctx.RecordUnresolvedNotificationObject(mod, "notif1", "obj2", span)

	ctx.FinalizeUnresolved()

	result := ctx.mib
	unresolved := result.Unresolved()

	// We expect 5 unresolved refs.
	testutil.Len(t, unresolved, 5, "expected 5 unresolved refs, got")

	// Verify each kind is present with correct reason.
	kindCounts := map[UnresolvedKind]int{}
	reasonByKind := map[UnresolvedKind]string{}
	for _, u := range unresolved {
		kindCounts[u.Kind]++
		reasonByKind[u.Kind] = u.Reason
		testutil.Equal(t, "TEST-MIB", u.Module, "unresolved ref kind")
	}

	expectedKinds := []UnresolvedKind{UnresolvedImport, UnresolvedType, UnresolvedOID, UnresolvedIndex, UnresolvedNotificationObject}
	for _, k := range expectedKinds {
		testutil.Equal(t, 1, kindCounts[k], "expected 1 unresolved ref of kind , got")
	}

	testutil.Equal(t, "module_not_found", reasonByKind[UnresolvedImport], "import reason")
	testutil.Equal(t, "unknown_type", reasonByKind[UnresolvedType], "type reason")
	testutil.Equal(t, "unknown_parent", reasonByKind[UnresolvedOID], "oid reason")
	testutil.Equal(t, "unknown_index_object", reasonByKind[UnresolvedIndex], "index reason")
	testutil.Equal(t, "unknown_object", reasonByKind[UnresolvedNotificationObject], "notif object reason")

	// Diagnostics should also be copied.
	diags := result.Diagnostics()
	testutil.Len(t, diags, 5, "expected 5 diagnostics, got")
}

func TestFinalizeUnresolved_NilModule(t *testing.T) {
	ctx := newTestContext()
	span := types.Span{}

	ctx.RecordUnresolvedImport(nil, "OTHER", "sym1", "module_not_found", span)
	ctx.RecordUnresolvedType(nil, "ref1", "UnknownType", span)
	ctx.RecordUnresolvedOid(nil, "obj1", "parent1", span)
	ctx.RecordUnresolvedIndex(nil, "row1", "idx1", span)
	ctx.RecordUnresolvedNotificationObject(nil, "notif1", "obj2", span)

	ctx.FinalizeUnresolved()

	result := ctx.mib
	for _, u := range result.Unresolved() {
		testutil.Equal(t, "", u.Module, "unresolved ref kind")
		// Reason should still be populated even with nil module.
		testutil.True(t, u.Reason != "", "expected non-empty reason for kind %s", u.Kind)
	}
}

func TestCopyUsedImportsToModules_Direct(t *testing.T) {
	importer := &module.Module{Name: "IMPORTER-MIB"}
	source := &module.Module{Name: "SOURCE-MIB"}
	ctx := newTestContextForModules(DefaultConfig(), importer, source)

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
	ctx := newTestContextForModules(DefaultConfig(), importer, source)

	ctx.moduleImports[importer] = map[string]*module.Module{
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
	config := DefaultConfig()
	ctx := newResolverContext(nil, ResolverNormal, config)
	got := ctx.DiagnosticConfig()
	testutil.Equal(t, config.Reporting, got.Reporting, "DiagnosticConfig().Reporting")
}
