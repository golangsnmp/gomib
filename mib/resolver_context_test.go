package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

func newTestContext() *resolverContext {
	return newResolverContext(nil, nil, DefaultConfig())
}

// testNodeArc is a counter for creating unique test nodes.
var testNodeArc uint32

// newTestNode creates a named *Node for testing. Each call returns
// a distinct node (unique arc under a shared root).
func newTestNode(name string) *Node {
	testNodeArc++
	m := newMib()
	n := m.Root().getOrCreateChild(testNodeArc)
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
			var found bool
			for _, d := range diags {
				if d.Code == tt.code {
					found = true
					if d.Severity != SeverityError {
						t.Errorf("diagnostic %q has severity %d, want %d (SeverityError)",
							tt.code, d.Severity, SeverityError)
					}
					if d.Module != "TEST-MIB" {
						t.Errorf("diagnostic %q has module %q, want %q",
							tt.code, d.Module, "TEST-MIB")
					}
				}
			}
			if !found {
				t.Errorf("no diagnostic with code %q emitted", tt.code)
			}
		})
	}
}

func TestIsASN1Primitive(t *testing.T) {
	positives := []string{"INTEGER", "OCTET STRING", "OBJECT IDENTIFIER", "BITS"}
	for _, name := range positives {
		if !isASN1Primitive(name) {
			t.Errorf("isASN1Primitive(%q) = false, want true", name)
		}
	}

	negatives := []string{
		"Integer32", "Counter32", "DisplayString", "integer",
		"OCTETSTRING", "OBJECT-IDENTIFIER", "", "Counter",
	}
	for _, name := range negatives {
		if isASN1Primitive(name) {
			t.Errorf("isASN1Primitive(%q) = true, want false", name)
		}
	}
}

func TestIsSmiGlobalType(t *testing.T) {
	positives := []string{
		"Integer32", "Counter32", "Counter64", "Gauge32",
		"Unsigned32", "TimeTicks", "IpAddress", "Opaque",
	}
	for _, name := range positives {
		if !isSmiGlobalType(name) {
			t.Errorf("isSmiGlobalType(%q) = false, want true", name)
		}
	}

	negatives := []string{
		"INTEGER", "Counter", "Gauge", "DisplayString",
		"integer32", "NetworkAddress", "",
	}
	for _, name := range negatives {
		if isSmiGlobalType(name) {
			t.Errorf("isSmiGlobalType(%q) = true, want false", name)
		}
	}
}

func TestIsSmiV1GlobalType(t *testing.T) {
	positives := []string{"Counter", "Gauge", "NetworkAddress"}
	for _, name := range positives {
		if !isSmiV1GlobalType(name) {
			t.Errorf("isSmiV1GlobalType(%q) = false, want true", name)
		}
	}

	negatives := []string{
		"Counter32", "Gauge32", "IpAddress", "INTEGER",
		"counter", "TimeTicks", "",
	}
	for _, name := range negatives {
		if isSmiV1GlobalType(name) {
			t.Errorf("isSmiV1GlobalType(%q) = true, want false", name)
		}
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
		if !isSNMPv2TCType(name) {
			t.Errorf("isSNMPv2TCType(%q) = false, want true", name)
		}
	}

	negatives := []string{
		"INTEGER", "Counter32", "IpAddress", "displaystring",
		"Counter", "Gauge32", "",
	}
	for _, name := range negatives {
		if isSNMPv2TCType(name) {
			t.Errorf("isSNMPv2TCType(%q) = true, want false", name)
		}
	}
}

func TestLookupInModuleScope_Direct(t *testing.T) {
	// Symbol found directly in the starting module.
	modA := &module.Module{Name: "A"}
	nodeX := newTestNode("x")

	symbols := map[*module.Module]map[string]*Node{
		modA: {"x": nodeX},
	}
	imports := map[*module.Module]map[string]*module.Module{}

	got, ok := lookupInModuleScope(modA, "x",
		func(m *module.Module) map[string]*Node { return symbols[m] },
		func(m *module.Module) map[string]*module.Module { return imports[m] },
	)
	if !ok || got != nodeX {
		t.Fatalf("expected to find nodeX directly, got ok=%v node=%v", ok, got)
	}
}

func TestLookupInModuleScope_ImportChain(t *testing.T) {
	// A imports "x" from B, B has "x" registered.
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	nodeX := newTestNode("x")

	symbols := map[*module.Module]map[string]*Node{
		modB: {"x": nodeX},
	}
	imports := map[*module.Module]map[string]*module.Module{
		modA: {"x": modB},
	}

	got, ok := lookupInModuleScope(modA, "x",
		func(m *module.Module) map[string]*Node { return symbols[m] },
		func(m *module.Module) map[string]*module.Module { return imports[m] },
	)
	if !ok || got != nodeX {
		t.Fatalf("expected to find nodeX via import chain, got ok=%v node=%v", ok, got)
	}
}

func TestLookupInModuleScope_ImportTargetLacksSymbol(t *testing.T) {
	// Import points to a module that doesn't have the symbol registered.
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}

	symbols := map[*module.Module]map[string]*Node{}
	imports := map[*module.Module]map[string]*module.Module{
		modA: {"x": modB},
	}

	_, ok := lookupInModuleScope(modA, "x",
		func(m *module.Module) map[string]*Node { return symbols[m] },
		func(m *module.Module) map[string]*module.Module { return imports[m] },
	)
	if ok {
		t.Fatal("expected false when import target lacks the symbol")
	}
}

func TestLookupInModuleScope_NotFound(t *testing.T) {
	modA := &module.Module{Name: "A"}

	symbols := map[*module.Module]map[string]*Node{}
	imports := map[*module.Module]map[string]*module.Module{}

	_, ok := lookupInModuleScope(modA, "x",
		func(m *module.Module) map[string]*Node { return symbols[m] },
		func(m *module.Module) map[string]*module.Module { return imports[m] },
	)
	if ok {
		t.Fatal("expected not found for missing symbol")
	}
}

func TestLookupNodeForModule(t *testing.T) {
	ctx := newTestContext()
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	nodeX := newTestNode("x")

	// Register node in B, import from A -> B.
	ctx.ModuleSymbolToNode[modB] = map[string]*Node{"x": nodeX}
	ctx.ModuleImports[modA] = map[string]*module.Module{"x": modB}

	got, ok := ctx.LookupNodeForModule(modA, "x")
	if !ok || got != nodeX {
		t.Fatalf("LookupNodeForModule: expected nodeX, got ok=%v", ok)
	}

	_, ok = ctx.LookupNodeForModule(modA, "y")
	if ok {
		t.Fatal("LookupNodeForModule: expected false for unknown symbol")
	}
}

func TestLookupNodeInModule(t *testing.T) {
	ctx := newTestContext()
	modA := &module.Module{Name: "MY-MIB"}
	nodeX := newTestNode("x")

	ctx.ModuleIndex["MY-MIB"] = []*module.Module{modA}
	ctx.ModuleSymbolToNode[modA] = map[string]*Node{"x": nodeX}

	got, ok := ctx.LookupNodeInModule("MY-MIB", "x")
	if !ok || got != nodeX {
		t.Fatalf("LookupNodeInModule: expected nodeX, got ok=%v", ok)
	}

	_, ok = ctx.LookupNodeInModule("OTHER-MIB", "x")
	if ok {
		t.Fatal("LookupNodeInModule: expected false for unknown module")
	}
}

func TestLookupNodeInModule_MultipleVersions(t *testing.T) {
	ctx := newTestContext()
	modV1 := &module.Module{Name: "MY-MIB"}
	modV2 := &module.Module{Name: "MY-MIB"}
	nodeX := newTestNode("x")

	// Only the second version has the symbol.
	ctx.ModuleIndex["MY-MIB"] = []*module.Module{modV1, modV2}
	ctx.ModuleSymbolToNode[modV2] = map[string]*Node{"x": nodeX}

	got, ok := ctx.LookupNodeInModule("MY-MIB", "x")
	if !ok || got != nodeX {
		t.Fatalf("expected to find nodeX in second version, got ok=%v", ok)
	}
}

func TestLookupNodeGlobal(t *testing.T) {
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	nodeX := newTestNode("x")
	nodeY := newTestNode("y")

	ctx := newResolverContext([]*module.Module{modA, modB}, nil, DefaultConfig())
	ctx.ModuleSymbolToNode[modA] = map[string]*Node{"x": nodeX}
	ctx.ModuleSymbolToNode[modB] = map[string]*Node{"y": nodeY}

	got, ok := ctx.LookupNodeGlobal("x")
	if !ok || got != nodeX {
		t.Fatalf("LookupNodeGlobal(x): expected nodeX, got ok=%v", ok)
	}

	got, ok = ctx.LookupNodeGlobal("y")
	if !ok || got != nodeY {
		t.Fatalf("LookupNodeGlobal(y): expected nodeY, got ok=%v", ok)
	}

	_, ok = ctx.LookupNodeGlobal("z")
	if ok {
		t.Fatal("LookupNodeGlobal(z): expected false")
	}
}

func TestLookupNodeGlobal_DeterministicOrder(t *testing.T) {
	// When the same name appears in multiple modules, the first module wins.
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	nodeA := newTestNode("x")
	nodeB := newTestNode("x")

	ctx := newResolverContext([]*module.Module{modA, modB}, nil, DefaultConfig())
	ctx.ModuleSymbolToNode[modA] = map[string]*Node{"x": nodeA}
	ctx.ModuleSymbolToNode[modB] = map[string]*Node{"x": nodeB}

	got, ok := ctx.LookupNodeGlobal("x")
	if !ok || got != nodeA {
		t.Fatal("LookupNodeGlobal should return the first module's node")
	}
}

func TestLookupTypeForModule(t *testing.T) {
	// Test type lookup via import chain.
	ctx := newTestContext()
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}
	typeX := newType("MyType")

	ctx.ModuleSymbolToType[modB] = map[string]*Type{"MyType": typeX}
	ctx.ModuleImports[modA] = map[string]*module.Module{"MyType": modB}

	got, ok := ctx.LookupTypeForModule(modA, "MyType")
	if !ok || got != typeX {
		t.Fatalf("LookupTypeForModule: expected typeX, got ok=%v", ok)
	}
}

func TestLookupTypeForModule_ASN1Fallback(t *testing.T) {
	// ASN.1 primitives should resolve even without explicit import.
	ctx := newTestContext()
	modA := &module.Module{Name: "A"}
	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	intType := newType("INTEGER")

	ctx.Snmpv2SMIModule = smiMod
	ctx.ModuleSymbolToType[smiMod] = map[string]*Type{"INTEGER": intType}

	got, ok := ctx.LookupTypeForModule(modA, "INTEGER")
	if !ok || got != intType {
		t.Fatalf("expected ASN.1 primitive fallback, got ok=%v", ok)
	}
}

func TestLookupTypeForModule_PermissiveFallbacks(t *testing.T) {
	// In permissive mode, SMI global types, SMIv1 types, and TC types resolve.
	ctx := newResolverContext(nil, nil, PermissiveConfig())
	modA := &module.Module{Name: "A"}

	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	rfc1155Mod := &module.Module{Name: "RFC1155-SMI"}
	tcMod := &module.Module{Name: "SNMPv2-TC"}

	counter32 := newType("Counter32")
	counter := newType("Counter")
	displayString := newType("DisplayString")

	ctx.Snmpv2SMIModule = smiMod
	ctx.Rfc1155SMIModule = rfc1155Mod
	ctx.Snmpv2TCModule = tcMod
	ctx.ModuleSymbolToType[smiMod] = map[string]*Type{"Counter32": counter32}
	ctx.ModuleSymbolToType[rfc1155Mod] = map[string]*Type{"Counter": counter}
	ctx.ModuleSymbolToType[tcMod] = map[string]*Type{"DisplayString": displayString}

	tests := []struct {
		name string
		want *Type
	}{
		{"Counter32", counter32},
		{"Counter", counter},
		{"DisplayString", displayString},
	}
	for _, tt := range tests {
		got, ok := ctx.LookupTypeForModule(modA, tt.name)
		if !ok || got != tt.want {
			t.Errorf("LookupTypeForModule(%q) permissive: ok=%v, got=%v", tt.name, ok, got)
		}
	}
}

func TestLookupTypeForModule_StrictNoFallback(t *testing.T) {
	// In strict mode, SMI global types should not resolve without import.
	ctx := newResolverContext(nil, nil, StrictConfig())
	modA := &module.Module{Name: "A"}

	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	counter32 := newType("Counter32")

	ctx.Snmpv2SMIModule = smiMod
	ctx.ModuleSymbolToType[smiMod] = map[string]*Type{"Counter32": counter32}

	// Counter32 is not an ASN.1 primitive, so strict mode should not find it.
	_, ok := ctx.LookupTypeForModule(modA, "Counter32")
	if ok {
		t.Fatal("expected strict mode to not resolve Counter32 without import")
	}

	// ASN.1 primitives should still resolve in strict mode.
	intType := newType("INTEGER")
	ctx.ModuleSymbolToType[smiMod]["INTEGER"] = intType

	got, ok := ctx.LookupTypeForModule(modA, "INTEGER")
	if !ok || got != intType {
		t.Fatal("expected ASN.1 primitive to resolve even in strict mode")
	}
}

func TestLookupType_Permissive(t *testing.T) {
	// LookupType with no module context, permissive mode.
	ctx := newResolverContext(nil, nil, PermissiveConfig())

	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	rfc1155Mod := &module.Module{Name: "RFC1155-SMI"}
	tcMod := &module.Module{Name: "SNMPv2-TC"}

	intType := newType("INTEGER")
	counter32 := newType("Counter32")
	gauge := newType("Gauge")
	truthValue := newType("TruthValue")

	ctx.Snmpv2SMIModule = smiMod
	ctx.Rfc1155SMIModule = rfc1155Mod
	ctx.Snmpv2TCModule = tcMod
	ctx.ModuleSymbolToType[smiMod] = map[string]*Type{
		"INTEGER":   intType,
		"Counter32": counter32,
	}
	ctx.ModuleSymbolToType[rfc1155Mod] = map[string]*Type{"Gauge": gauge}
	ctx.ModuleSymbolToType[tcMod] = map[string]*Type{"TruthValue": truthValue}

	tests := []struct {
		name string
		want *Type
	}{
		{"INTEGER", intType},
		{"Counter32", counter32},
		{"Gauge", gauge},
		{"TruthValue", truthValue},
	}
	for _, tt := range tests {
		got, ok := ctx.LookupType(tt.name)
		if !ok || got != tt.want {
			t.Errorf("LookupType(%q): ok=%v, got=%v", tt.name, ok, got)
		}
	}
}

func TestLookupType_StrictOnlyPrimitives(t *testing.T) {
	// Strict mode: only ASN.1 primitives, no global search.
	ctx := newResolverContext(nil, nil, StrictConfig())

	smiMod := &module.Module{Name: "SNMPv2-SMI"}
	intType := newType("INTEGER")
	counter32 := newType("Counter32")

	ctx.Snmpv2SMIModule = smiMod
	ctx.ModuleSymbolToType[smiMod] = map[string]*Type{
		"INTEGER":   intType,
		"Counter32": counter32,
	}

	got, ok := ctx.LookupType("INTEGER")
	if !ok || got != intType {
		t.Fatal("expected ASN.1 primitive to resolve in strict mode")
	}

	_, ok = ctx.LookupType("Counter32")
	if ok {
		t.Fatal("expected strict mode to not allow global search for Counter32")
	}
}

func TestLookupType_GlobalModuleScan(t *testing.T) {
	// In permissive mode, LookupType scans all modules for unknown types.
	modA := &module.Module{Name: "A"}
	vendorType := newType("VendorSpecialType")

	ctx := newResolverContext([]*module.Module{modA}, nil, PermissiveConfig())
	ctx.Snmpv2SMIModule = &module.Module{Name: "SNMPv2-SMI"}
	ctx.ModuleSymbolToType[modA] = map[string]*Type{"VendorSpecialType": vendorType}

	got, ok := ctx.LookupType("VendorSpecialType")
	if !ok || got != vendorType {
		t.Fatal("expected global module scan to find vendor type in permissive mode")
	}
}

func TestRegisterImport(t *testing.T) {
	ctx := newTestContext()
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}

	ctx.registerImport(modA, "foo", modB)

	imports := ctx.ModuleImports[modA]
	if imports == nil {
		t.Fatal("expected imports map to be created")
	}
	if imports["foo"] != modB {
		t.Fatal("expected import to point to modB")
	}

	// Register a second import in the same module.
	modC := &module.Module{Name: "C"}
	ctx.registerImport(modA, "bar", modC)
	if ctx.ModuleImports[modA]["bar"] != modC {
		t.Fatal("expected second import to point to modC")
	}
}

func TestRegisterModuleNodeSymbol(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "A"}
	node := newTestNode("sysDescr")

	ctx.registerModuleNodeSymbol(mod, "sysDescr", node)

	symbols := ctx.ModuleSymbolToNode[mod]
	if symbols == nil {
		t.Fatal("expected symbol map to be created")
	}
	if symbols["sysDescr"] != node {
		t.Fatal("expected registered node")
	}

	// Overwrite should succeed.
	node2 := newTestNode("sysDescr")
	ctx.registerModuleNodeSymbol(mod, "sysDescr", node2)
	if ctx.ModuleSymbolToNode[mod]["sysDescr"] != node2 {
		t.Fatal("expected overwritten node")
	}
}

func TestRegisterModuleTypeSymbol(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "A"}
	typ := newType("MyType")

	ctx.registerModuleTypeSymbol(mod, "MyType", typ)

	symbols := ctx.ModuleSymbolToType[mod]
	if symbols == nil {
		t.Fatal("expected symbol map to be created")
	}
	if symbols["MyType"] != typ {
		t.Fatal("expected registered type")
	}
}

func TestEmitDiagnostic(t *testing.T) {
	tests := []struct {
		name     string
		config   DiagnosticConfig
		severity Severity
		want     int
	}{
		{
			name:     "default reports error",
			config:   DefaultConfig(),
			severity: SeverityError,
			want:     1,
		},
		{
			name:     "default reports minor",
			config:   DefaultConfig(),
			severity: SeverityMinor,
			want:     1,
		},
		{
			name:     "default suppresses style",
			config:   DefaultConfig(),
			severity: SeverityStyle,
			want:     0,
		},
		{
			name:     "default suppresses warning",
			config:   DefaultConfig(),
			severity: SeverityWarning,
			want:     0,
		},
		{
			name:     "default suppresses info",
			config:   DefaultConfig(),
			severity: SeverityInfo,
			want:     0,
		},
		{
			name:     "strict reports info",
			config:   StrictConfig(),
			severity: SeverityInfo,
			want:     1,
		},
		{
			name:     "permissive reports warning",
			config:   PermissiveConfig(),
			severity: SeverityWarning,
			want:     1,
		},
		{
			name:     "permissive suppresses info",
			config:   PermissiveConfig(),
			severity: SeverityInfo,
			want:     0,
		},
		{
			name: "silent suppresses everything",
			config: DiagnosticConfig{
				Level: StrictnessSilent,
			},
			severity: SeverityFatal,
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newResolverContext(nil, nil, tt.config)
			mod := &module.Module{Name: "MOD"}
			ctx.EmitDiagnostic("test-code", tt.severity, mod, types.Span{}, "test message")
			got := len(ctx.Diagnostics())
			if got != tt.want {
				t.Errorf("got %d diagnostics, want %d", got, tt.want)
			}
		})
	}
}

func TestEmitDiagnostic_IgnoredCode(t *testing.T) {
	config := DiagnosticConfig{
		Level:  StrictnessStrict,
		Ignore: []string{"test-*"},
	}
	ctx := newResolverContext(nil, nil, config)
	mod := &module.Module{Name: "MOD"}
	ctx.EmitDiagnostic("test-foo", SeverityError, mod, types.Span{}, "ignored")
	if len(ctx.Diagnostics()) != 0 {
		t.Fatal("expected ignored code to produce no diagnostics")
	}

	// Non-matching code should still be reported.
	ctx.EmitDiagnostic("other-code", SeverityError, mod, types.Span{}, "not ignored")
	if len(ctx.Diagnostics()) != 1 {
		t.Fatal("expected non-ignored code to produce a diagnostic")
	}
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
	ctx.EmitDiagnostic("my-code", SeverityMinor, mod, span, "something happened")

	diags := ctx.Diagnostics()
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	d := diags[0]
	if d.Code != "my-code" {
		t.Errorf("Code = %q, want %q", d.Code, "my-code")
	}
	if d.Severity != SeverityMinor {
		t.Errorf("Severity = %d, want %d", d.Severity, SeverityMinor)
	}
	if d.Module != "TEST-MIB" {
		t.Errorf("Module = %q, want %q", d.Module, "TEST-MIB")
	}
	if d.Line != 10 {
		t.Errorf("Line = %d, want 10", d.Line)
	}
	if d.Column != 5 {
		t.Errorf("Column = %d, want 5", d.Column)
	}
	if d.Message != "something happened" {
		t.Errorf("Message = %q, want %q", d.Message, "something happened")
	}
}

func TestFinalizeUnresolved(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	span := types.Span{}

	ctx.RecordUnresolvedImport(mod, "OTHER", "sym1", "not found", span)
	ctx.RecordUnresolvedType(mod, "ref1", "UnknownType", span)
	ctx.RecordUnresolvedOid(mod, "obj1", "parent1", span)
	ctx.RecordUnresolvedIndex(mod, "row1", "idx1", span)
	ctx.RecordUnresolvedNotificationObject(mod, "notif1", "obj2", span)

	ctx.FinalizeUnresolved()

	result := ctx.Mib
	unresolved := result.Unresolved()

	// We expect 5 unresolved refs.
	if len(unresolved) != 5 {
		t.Fatalf("expected 5 unresolved refs, got %d", len(unresolved))
	}

	// Verify each kind is present.
	kindCounts := map[UnresolvedKind]int{}
	for _, u := range unresolved {
		kindCounts[u.Kind]++
		if u.Module != "TEST-MIB" {
			t.Errorf("unresolved ref kind=%s has module=%q, want TEST-MIB", u.Kind, u.Module)
		}
	}

	expectedKinds := []UnresolvedKind{UnresolvedImport, UnresolvedType, UnresolvedOID, UnresolvedIndex, UnresolvedNotificationObject}
	for _, k := range expectedKinds {
		if kindCounts[k] != 1 {
			t.Errorf("expected 1 unresolved ref of kind %s, got %d", k, kindCounts[k])
		}
	}

	// Diagnostics should also be copied.
	diags := result.Diagnostics()
	if len(diags) != 5 {
		t.Errorf("expected 5 diagnostics, got %d", len(diags))
	}
}

func TestFinalizeUnresolved_NilModule(t *testing.T) {
	ctx := newTestContext()
	span := types.Span{}

	ctx.RecordUnresolvedImport(nil, "OTHER", "sym1", "not found", span)
	ctx.RecordUnresolvedType(nil, "ref1", "UnknownType", span)
	ctx.RecordUnresolvedOid(nil, "obj1", "parent1", span)
	ctx.RecordUnresolvedIndex(nil, "row1", "idx1", span)
	ctx.RecordUnresolvedNotificationObject(nil, "notif1", "obj2", span)

	ctx.FinalizeUnresolved()

	result := ctx.Mib
	for _, u := range result.Unresolved() {
		if u.Module != "" {
			t.Errorf("unresolved ref kind=%q has module=%q, want empty string for nil module", u.Kind, u.Module)
		}
	}
}

func TestDropModules(t *testing.T) {
	mod := &module.Module{Name: "A"}
	ctx := newResolverContext([]*module.Module{mod}, nil, DefaultConfig())
	ctx.ModuleIndex["A"] = []*module.Module{mod}
	ctx.ModuleDefNames[mod] = map[string]struct{}{"foo": {}}

	if ctx.Modules == nil {
		t.Fatal("expected Modules to be set before DropModules")
	}

	ctx.DropModules()

	if ctx.Modules != nil {
		t.Error("expected Modules to be nil after DropModules")
	}
	if ctx.ModuleIndex != nil {
		t.Error("expected ModuleIndex to be nil after DropModules")
	}
	if ctx.ModuleDefNames != nil {
		t.Error("expected ModuleDefNames to be nil after DropModules")
	}

	// Other maps should be untouched.
	if ctx.ModuleSymbolToNode == nil {
		t.Error("expected ModuleSymbolToNode to survive DropModules")
	}
	if ctx.ModuleSymbolToType == nil {
		t.Error("expected ModuleSymbolToType to survive DropModules")
	}
	if ctx.ModuleImports == nil {
		t.Error("expected ModuleImports to survive DropModules")
	}
}

func TestDiagnosticConfig_Getter(t *testing.T) {
	config := PermissiveConfig()
	ctx := newResolverContext(nil, nil, config)
	got := ctx.DiagnosticConfig()
	if got.Level != config.Level {
		t.Errorf("DiagnosticConfig().Level = %v, want %v", got.Level, config.Level)
	}
}
