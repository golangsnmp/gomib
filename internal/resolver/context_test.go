package resolver

import (
	"sync/atomic"
	"testing"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

// testNodeArc is an atomic counter for creating unique test nodes.
// Atomic to avoid races if tests run in parallel.
var testNodeArc atomic.Uint32

// newTestNode creates a named *model.Node for testing. Each call returns
// a distinct node (unique arc under a shared root).
func newTestNode(name string) *model.Node {
	arc := testNodeArc.Add(1)
	m := model.NewMib()
	n := model.GetOrCreateChild(m.Root(), arc)
	model.SetNodeName(n, name)
	return n
}

func TestRecordUnresolvedSeverityConsistency(t *testing.T) {
	// All recordUnresolved calls should emit diagnostics at model.SeverityError.
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
				c.recordUnresolved(types.DiagImportNotFound, mod, span,
					"unresolved import: \"someSymbol\" from \"OTHER-MIB\" (not found)",
					model.UnresolvedRef{Kind: model.UnresolvedImport, Symbol: "someSymbol", Module: "TEST-MIB", Reason: "not found"})
			},
		},
		{
			name: "import module not found",
			code: "import-module-not-found",
			emit: func(c *resolverContext) {
				c.recordUnresolved(types.DiagImportModuleNotFound, mod, span,
					"unresolved import: \"someSymbol\" from \"MISSING-MIB\" (module_not_found)",
					model.UnresolvedRef{Kind: model.UnresolvedImport, Symbol: "someSymbol", Module: "TEST-MIB", Reason: reasonModuleNotFound})
			},
		},
		{
			name: "type",
			code: "type-unknown",
			emit: func(c *resolverContext) {
				c.recordUnresolved(types.DiagTypeUnknown, mod, span,
					"unresolved type: \"myType\" references unknown type \"UnknownType\"",
					model.UnresolvedRef{Kind: model.UnresolvedType, Symbol: "UnknownType", Module: "TEST-MIB", Reason: reasonUnknownType})
			},
		},
		{
			name: "oid",
			code: "oid-orphan",
			emit: func(c *resolverContext) {
				c.recordUnresolved(types.DiagOidOrphan, mod, span,
					"unresolved OID: \"myObject\" references unknown parent \"unknownParent\"",
					model.UnresolvedRef{Kind: model.UnresolvedOID, Symbol: "unknownParent", Module: "TEST-MIB", Reason: reasonUnknownParent})
			},
		},
		{
			name: "index",
			code: "index-unresolved",
			emit: func(c *resolverContext) {
				c.recordUnresolved(types.DiagIndexUnresolved, mod, span,
					"unresolved INDEX: \"myRow\" references unknown object \"missingIndex\"",
					model.UnresolvedRef{Kind: model.UnresolvedIndex, Symbol: "missingIndex", Module: "TEST-MIB", Reason: reasonUnknownIndex})
			},
		},
		{
			name: "notification object",
			code: "objects-unresolved",
			emit: func(c *resolverContext) {
				c.recordUnresolved(types.DiagObjectsUnresolved, mod, span,
					"unresolved OBJECTS: \"myNotif\" references unknown object \"missingObject\"",
					model.UnresolvedRef{Kind: model.UnresolvedNotificationObject, Symbol: "missingObject", Module: "TEST-MIB", Reason: reasonUnknownObject})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestContext()
			tt.emit(ctx)

			diags := ctx.Diagnostics()
			diag := hasDiag(t, diags, tt.code)
			testutil.Equal(t, model.SeverityError, diag.Severity, "diagnostic %q severity", tt.code)
			testutil.Equal(t, "TEST-MIB", diag.Module, "diagnostic %q module", tt.code)
		})
	}
}

func TestRegisterImport(t *testing.T) {
	ctx := newTestContext()
	modA := &module.Module{Name: "A"}
	modB := &module.Module{Name: "B"}

	ctx.registerImport(modA, "foo", modB)

	imports := ctx.importSources[modA]
	testutil.NotNil(t, imports, "expected imports map to be created")
	testutil.Equal(t, modB, imports["foo"], "expected import to point to modB")

	// Register a second import in the same module.
	modC := &module.Module{Name: "C"}
	ctx.registerImport(modA, "bar", modC)
	testutil.Equal(t, modC, ctx.importSources[modA]["bar"], "expected second import to point to modC")
}

func TestRegisterModuleNodeSymbol(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "A"}
	node := newTestNode("sysDescr")

	ctx.registerModuleNodeSymbol(mod, "sysDescr", node)

	symbols := ctx.nodeSymbols[mod]
	testutil.NotNil(t, symbols, "expected symbol map to be created")
	testutil.Equal(t, node, symbols["sysDescr"], "expected registered node")

	// Overwrite should succeed.
	node2 := newTestNode("sysDescr")
	ctx.registerModuleNodeSymbol(mod, "sysDescr", node2)
	testutil.Equal(t, node2, ctx.nodeSymbols[mod]["sysDescr"], "expected overwritten node")
}

func TestRegisterModuleTypeSymbol(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "A"}
	typ := model.NewType("MyType")

	ctx.registerModuleTypeSymbol(mod, "MyType", typ)

	symbols := ctx.typeSymbols[mod]
	testutil.NotNil(t, symbols, "expected symbol map to be created")
	testutil.Equal(t, typ, symbols["MyType"], "expected registered type")
}

func TestEmitDiagnostic(t *testing.T) {
	tests := []struct {
		name   string
		config model.DiagnosticConfig
		code   string // must be a registered diagnostic code
		want   int
	}{
		{
			name:   "default reports error",
			config: model.DefaultConfig(),
			code:   types.DiagParseError, // model.SeverityError
			want:   1,
		},
		{
			name:   "default reports minor",
			config: model.DefaultConfig(),
			code:   types.DiagGroupNotAccessible, // model.SeverityMinor
			want:   1,
		},
		{
			name:   "default suppresses style",
			config: model.DefaultConfig(),
			code:   types.DiagIdentifierUnderscore, // model.SeverityStyle
			want:   0,
		},
		{
			name:   "default suppresses warning",
			config: model.DefaultConfig(),
			code:   types.DiagIdentifierHyphenSMI, // model.SeverityWarning
			want:   0,
		},
		{
			name:   "verbose reports warning",
			config: model.VerboseConfig(),
			code:   types.DiagIdentifierHyphenSMI, // model.SeverityWarning
			want:   1,
		},
		{
			name: "silent suppresses non-fatal",
			config: model.DiagnosticConfig{
				Reporting: model.ReportingSilent,
			},
			code: types.DiagParseError, // model.SeverityError
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newResolverContext(nil, model.ResolverNormal, tt.config)
			mod := &module.Module{Name: "MOD"}
			ctx.EmitDiagnostic(tt.code, mod, types.Span{}, "test message")
			got := len(ctx.Diagnostics())
			testutil.Equal(t, tt.want, got, "diagnostics")
		})
	}
}

func TestShouldReport_FatalAlwaysReported(t *testing.T) {
	// No diagnostic codes map to model.SeverityFatal, but ShouldReport must still
	// pass fatal-severity diagnostics even in silent mode.
	config := model.DiagnosticConfig{Reporting: model.ReportingSilent}
	testutil.True(t, config.ShouldReport("any-code", model.SeverityFatal),
		"fatal should be reported even in silent mode")
}

func TestEmitDiagnostic_IgnoredCode(t *testing.T) {
	config := model.DiagnosticConfig{
		Reporting: model.ReportingVerbose,
		Ignore:    []string{"import-*"},
	}
	ctx := newResolverContext(nil, model.ResolverNormal, config)
	mod := &module.Module{Name: "MOD"}
	ctx.EmitDiagnostic(types.DiagImportNotFound, mod, types.Span{}, "ignored")
	testutil.Len(t, ctx.Diagnostics(), 0, "expected ignored code to produce no diagnostics")

	// Non-matching code should still be reported.
	ctx.EmitDiagnostic(types.DiagParseError, mod, types.Span{}, "not ignored")
	testutil.Len(t, ctx.Diagnostics(), 1, "expected non-ignored code to produce a diagnostic")
}

func TestEmitDiagnostic_PersistsEffectiveSeverity(t *testing.T) {
	config := model.QuietConfig()
	config.Overrides = map[string]model.Severity{types.DiagGroupNotAccessible: model.SeveritySevere}
	ctx := newResolverContext(nil, model.ResolverNormal, config)
	ctx.EmitDiagnostic(types.DiagGroupNotAccessible, &module.Module{Name: "MOD"}, types.Span{}, "promoted")

	diags := ctx.Diagnostics()
	testutil.Len(t, diags, 1, "expected promoted diagnostic")
	testutil.Equal(t, model.SeveritySevere, diags[0].Severity, "effective severity")
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
	span := types.NewSpan(94, 95) // line 10, columns 5-6
	ctx.EmitDiagnostic(types.DiagGroupNotAccessible, mod, span, "something happened")

	diags := ctx.Diagnostics()
	testutil.Len(t, diags, 1, "expected 1 diagnostic, got")
	d := diags[0]
	testutil.Equal(t, types.DiagGroupNotAccessible, d.Code, "Code")
	testutil.Equal(t, model.SeverityMinor, d.Severity, "Severity")
	testutil.Equal(t, "TEST-MIB", d.Module, "Module")
	testutil.Equal(t, 10, d.Line, "Line")
	testutil.Equal(t, 5, d.Column, "Column")
	testutil.Equal(t, 10, d.EndLine, "EndLine")
	testutil.Equal(t, 6, d.EndColumn, "EndColumn")
	testutil.Equal(t, "something happened", d.Message, "Message")
}

func TestEmitDiagnostic_NoUsableEnd(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{
		Name:      "TEST-MIB",
		LineTable: types.BuildLineTable([]byte("first\nsecond\n")),
	}
	ctx.EmitDiagnostic(types.DiagGroupNotAccessible, mod, types.NewSpan(8, 8), "point diagnostic")

	diags := ctx.Diagnostics()
	testutil.Len(t, diags, 1, "expected 1 diagnostic, got")
	d := diags[0]
	testutil.Equal(t, 2, d.Line, "Line")
	testutil.Equal(t, 3, d.Column, "Column")
	testutil.Equal(t, 0, d.EndLine, "EndLine")
	testutil.Equal(t, 0, d.EndColumn, "EndColumn")
}

func TestRecordUnresolved_WritesToMib(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	span := types.Span{}

	ctx.recordUnresolved(types.DiagImportModuleNotFound, mod, span, "import msg",
		model.UnresolvedRef{Kind: model.UnresolvedImport, Symbol: "sym1", Module: "TEST-MIB", Reason: reasonModuleNotFound})
	ctx.recordUnresolved(types.DiagTypeUnknown, mod, span, "type msg",
		model.UnresolvedRef{Kind: model.UnresolvedType, Symbol: "UnknownType", Module: "TEST-MIB", Reason: reasonUnknownType})
	ctx.recordUnresolved(types.DiagOidOrphan, mod, span, "oid msg",
		model.UnresolvedRef{Kind: model.UnresolvedOID, Symbol: "parent1", Module: "TEST-MIB", Reason: reasonUnknownParent})
	ctx.recordUnresolved(types.DiagIndexUnresolved, mod, span, "index msg",
		model.UnresolvedRef{Kind: model.UnresolvedIndex, Symbol: "idx1", Module: "TEST-MIB", Reason: reasonUnknownIndex})
	ctx.recordUnresolved(types.DiagObjectsUnresolved, mod, span, "notif msg",
		model.UnresolvedRef{Kind: model.UnresolvedNotificationObject, Symbol: "obj2", Module: "TEST-MIB", Reason: reasonUnknownObject})

	unresolved := ctx.mib.Unresolved()
	testutil.Len(t, unresolved, 5, "expected 5 unresolved refs")

	kindCounts := map[model.UnresolvedKind]int{}
	reasonByKind := map[model.UnresolvedKind]string{}
	for _, u := range unresolved {
		kindCounts[u.Kind]++
		reasonByKind[u.Kind] = u.Reason
		testutil.Equal(t, "TEST-MIB", u.Module, "module name")
	}

	expectedKinds := []model.UnresolvedKind{model.UnresolvedImport, model.UnresolvedType, model.UnresolvedOID, model.UnresolvedIndex, model.UnresolvedNotificationObject}
	for _, k := range expectedKinds {
		testutil.Equal(t, 1, kindCounts[k], "count for kind")
	}

	testutil.Equal(t, reasonModuleNotFound, reasonByKind[model.UnresolvedImport], "import reason")
	testutil.Equal(t, reasonUnknownType, reasonByKind[model.UnresolvedType], "type reason")
	testutil.Equal(t, reasonUnknownParent, reasonByKind[model.UnresolvedOID], "oid reason")
	testutil.Equal(t, reasonUnknownIndex, reasonByKind[model.UnresolvedIndex], "index reason")
	testutil.Equal(t, reasonUnknownObject, reasonByKind[model.UnresolvedNotificationObject], "notif object reason")

	testutil.Len(t, ctx.Diagnostics(), 5, "expected 5 diagnostics")
}

func TestRecordUnresolved_NilModule(t *testing.T) {
	ctx := newTestContext()
	span := types.Span{}

	ctx.recordUnresolved(types.DiagImportModuleNotFound, nil, span, "import msg",
		model.UnresolvedRef{Kind: model.UnresolvedImport, Symbol: "sym1", Module: "", Reason: reasonModuleNotFound})
	ctx.recordUnresolved(types.DiagTypeUnknown, nil, span, "type msg",
		model.UnresolvedRef{Kind: model.UnresolvedType, Symbol: "UnknownType", Module: "", Reason: reasonUnknownType})
	ctx.recordUnresolved(types.DiagOidOrphan, nil, span, "oid msg",
		model.UnresolvedRef{Kind: model.UnresolvedOID, Symbol: "parent1", Module: "", Reason: reasonUnknownParent})
	ctx.recordUnresolved(types.DiagIndexUnresolved, nil, span, "index msg",
		model.UnresolvedRef{Kind: model.UnresolvedIndex, Symbol: "idx1", Module: "", Reason: reasonUnknownIndex})
	ctx.recordUnresolved(types.DiagObjectsUnresolved, nil, span, "notif msg",
		model.UnresolvedRef{Kind: model.UnresolvedNotificationObject, Symbol: "obj2", Module: "", Reason: reasonUnknownObject})

	for _, u := range ctx.mib.Unresolved() {
		testutil.Equal(t, "", u.Module, "module should be empty for nil")
		testutil.True(t, u.Reason != "", "expected non-empty reason for kind %s", u.Kind)
	}
}

func TestFinalizeDiagnostics(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	ctx.EmitDiagnostic(types.DiagParseError, mod, types.Span{}, "test error")
	ctx.FinalizeDiagnostics()

	diags := ctx.mib.Diagnostics()
	testutil.Len(t, diags, 1, "expected 1 diagnostic copied to mib")
	testutil.Equal(t, "test error", diags[0].Message, "message")
}
