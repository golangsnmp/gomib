package gomib

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

func TestLoadSingleMIB(t *testing.T) {
	src := mustDirTree(t, "testdata/corpus/primary")

	ctx := context.Background()
	m, err := Load(ctx, WithSource(src), WithModules("IF-MIB"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	testutil.NotNil(t, m, "m should not be nil")
	testutil.Greater(t, len(m.Modules()), 0, "should have loaded modules")
	testutil.Greater(t, len(m.Objects()), 0, "should have resolved objects")

	ifMIB := m.Module("IF-MIB")
	testutil.NotNil(t, ifMIB, "IF-MIB module should be found")

	ifIndex := m.Object("ifIndex")
	testutil.NotNil(t, ifIndex, "ifIndex object should be found")
}

func TestLoadAllCorpus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping corpus load in short mode")
	}

	src := mustDirTree(t, "testdata/corpus/primary")

	ctx := context.Background()
	m, err := Load(ctx, WithSource(src))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	testutil.Greater(t, len(m.Modules()), 50, "should have loaded many modules")
	testutil.Greater(t, len(m.Objects()), 1000, "should have resolved many objects")

	// Semantic spot-checks: verify key modules and objects survived resolution
	testutil.NotNil(t, m.Module("IF-MIB"), "IF-MIB should be present")
	testutil.NotNil(t, m.Module("SNMPv2-MIB"), "SNMPv2-MIB should be present")
	testutil.NotNil(t, m.Object("ifIndex"), "ifIndex should be resolved")
	testutil.NotNil(t, m.Object("sysDescr"), "sysDescr should be resolved")
	testutil.Equal(t, "1.3.6.1.2.1.1.1", m.Object("sysDescr").OID().String(),
		"sysDescr OID should be correct")

	t.Logf("Loaded %d modules, %d objects, %d types",
		len(m.Modules()), len(m.Objects()), len(m.Types()))
}

func TestDirSource(t *testing.T) {
	src, err := Dir("testdata/corpus/primary/ietf")
	if err != nil {
		t.Fatalf("Dir failed: %v", err)
	}

	names, err := src.ListModules()
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}
	testutil.Greater(t, len(names), 0, "should list modules")
}

func TestDirTreeSource(t *testing.T) {
	src, err := DirTree("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("DirTree failed: %v", err)
	}

	names, err := src.ListModules()
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}
	testutil.Greater(t, len(names), 10, "should list many modules recursively")
}

func TestMultiSource(t *testing.T) {
	primary := mustDirTree(t, "testdata/corpus/primary")
	problems := mustDirTree(t, "testdata/corpus/problems")

	ctx := context.Background()
	m, err := Load(ctx, WithSource(primary, problems), WithModules("IF-MIB"))
	if err != nil {
		t.Fatalf("Load from multi source failed: %v", err)
	}
	testutil.NotNil(t, m.Module("IF-MIB"), "should find IF-MIB from primary source")
}

func TestLoadNonexistentModule(t *testing.T) {
	src := mustDirTree(t, "testdata/corpus/primary")

	ctx := context.Background()
	m, err := Load(ctx, WithSource(src), WithModules("TOTALLY-FAKE-MIB-THAT-DOES-NOT-EXIST"))
	testutil.Error(t, err, "Load should error for nonexistent module")
	testutil.NotNil(t, m, "Load should return a Mib even for nonexistent module")
	testutil.True(t, errors.Is(err, ErrMissingModules), "error should wrap ErrMissingModules")

	mod := m.Module("TOTALLY-FAKE-MIB-THAT-DOES-NOT-EXIST")
	testutil.Nil(t, mod, "nonexistent module should not be in the result")
}

func TestLoadMissingModuleWithValidModule(t *testing.T) {
	src := mustDirTree(t, "testdata/corpus/primary")

	ctx := context.Background()
	m, err := Load(ctx, WithSource(src), WithModules("IF-MIB", "NONEXISTENT-MIB"))
	testutil.Error(t, err, "Load should error for missing module")
	testutil.True(t, errors.Is(err, ErrMissingModules), "error should wrap ErrMissingModules")
	testutil.NotNil(t, m, "Load should return a Mib with partial results")
	testutil.NotNil(t, m.Module("IF-MIB"), "IF-MIB should still be loaded")
	testutil.Nil(t, m.Module("NONEXISTENT-MIB"), "NONEXISTENT-MIB should not be found")
}

func TestLoadNoSources(t *testing.T) {
	ctx := context.Background()
	_, err := Load(ctx)
	testutil.Error(t, err, "loading with no sources should fail")
}

func TestNodeByName(t *testing.T) {
	m := loadTestMIB(t)

	node := m.Node("ifIndex")
	testutil.NotNil(t, node, "should find ifIndex by name")
}

func TestNodeNotFound(t *testing.T) {
	m := loadTestMIB(t)

	node := m.Node("totallyNonExistentSymbol")
	testutil.Nil(t, node, "nonexistent symbol should return nil")
}

func TestObjectByName(t *testing.T) {
	m := loadTestMIB(t)

	obj := m.Object("sysDescr")
	testutil.NotNil(t, obj, "should find sysDescr by name")
	testutil.Equal(t, "sysDescr", obj.Name(), "object name")
}

func TestType(t *testing.T) {
	m := loadTestMIB(t)

	typ := m.Type("DisplayString")
	testutil.NotNil(t, typ, "Type(DisplayString)")
	testutil.Equal(t, "DisplayString", typ.Name(), "type name")
	testutil.True(t, typ.IsTextualConvention(), "DisplayString should be a TC")
}

func TestNotification(t *testing.T) {
	m := loadTestMIB(t)

	notif := m.Notification("linkDown")
	testutil.NotNil(t, notif, "Notification(linkDown)")
	testutil.Equal(t, "linkDown", notif.Name(), "notification name")
}

func TestModulesCollection(t *testing.T) {
	m := loadTestMIB(t)

	mods := m.Modules()
	testutil.Greater(t, len(mods), 0, "should have modules")

	found := false
	for _, mod := range mods {
		if mod.Name() == "IF-MIB" {
			found = true
			break
		}
	}
	testutil.True(t, found, "should find IF-MIB in modules list")
}

func TestNodesIteration(t *testing.T) {
	m := loadTestMIB(t)

	count := 0
	for range m.Nodes() {
		count++
	}
	testutil.Greater(t, count, 0, "should have nodes")
	testutil.Equal(t, m.NodeCount(), count, "NodeCount should match iteration count")
}

func TestObjectsCollection(t *testing.T) {
	m := loadTestMIB(t)

	objs := m.Objects()
	testutil.Equal(t, len(m.Objects()), len(objs), "Objects() should be consistent")
}

func TestTablesAndScalars(t *testing.T) {
	m := loadTestMIB(t)

	tables := m.Tables()
	scalars := m.Scalars()

	testutil.Greater(t, len(tables), 0, "should have tables (IF-MIB has ifTable)")
	testutil.Greater(t, len(scalars), 0, "should have scalars (SNMPv2-MIB has sysDescr)")
}

func TestStrictMIBsPassAtStrictLevel(t *testing.T) {
	corpus := mustDirTree(t, "testdata/corpus/primary")
	strict := mustDirTree(t, "testdata/strictness/strict")

	tests := []string{"STRICT-TEST-MIB", "STRICT-TABLE-MIB"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			m, err := Load(ctx, WithSource(corpus, strict), WithModules(name), WithStrictness(mib.StrictnessStrict))
			if err != nil {
				t.Fatalf("Load failed: %v", err)
			}

			diags := m.Diagnostics()
			if len(diags) > 0 {
				for _, d := range diags {
					t.Errorf("unexpected diagnostic: [%s] %s: %s", d.Code, d.Severity, d.Message)
				}
			}
		})
	}
}

func TestUnderscoreViolationEmitsDiagnostic(t *testing.T) {
	m := loadViolationMIB(t, "UNDERSCORE-TEST-MIB", mib.StrictnessStrict)

	underscoreDiags := countDiagnostics(m, "identifier-underscore")
	testutil.Equal(t, 2, underscoreDiags, "expected 2 identifier-underscore diagnostics")

	m = loadViolationMIB(t, "UNDERSCORE-TEST-MIB", mib.StrictnessPermissive)

	underscoreDiags = countDiagnostics(m, "identifier-underscore")
	testutil.Equal(t, 0, underscoreDiags, "expected no identifier-underscore diagnostics in permissive mode")
}

func TestHyphenEndViolationEmitsDiagnostic(t *testing.T) {
	m := loadViolationMIB(t, "HYPHEN-END-TEST-MIB", mib.StrictnessStrict)

	hyphenDiags := countDiagnostics(m, "identifier-hyphen-end")
	testutil.Equal(t, 1, hyphenDiags, "expected 1 identifier-hyphen-end diagnostic")
}

func TestLongIdentifierViolationEmitsDiagnostic(t *testing.T) {
	m := loadViolationMIB(t, "LONG-IDENT-TEST-MIB", mib.StrictnessStrict)

	lengthDiags := countDiagnostics(m, "identifier-length-64")
	testutil.Equal(t, 1, lengthDiags, "expected 1 identifier-length-64 diagnostic")
}

func TestUppercaseIdentifierEmitsDiagnostic(t *testing.T) {
	corpus := mustDirTree(t, "testdata/corpus/primary")
	problems := mustDirTree(t, "testdata/corpus/problems")

	ctx := context.Background()

	m, err := Load(ctx, WithSource(corpus, problems), WithModules("PROBLEM-NAMING-MIB"), WithStrictness(mib.StrictnessNormal))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	caseDiags := countDiagnostics(m, "bad-identifier-case")
	testutil.Equal(t, 4, caseDiags, "expected 4 bad-identifier-case diagnostics in normal mode")

	node := m.Node("NetEngine8000SysOid")
	testutil.NotNil(t, node, "uppercase identifier should resolve in normal mode")

	m, err = Load(ctx, WithSource(corpus, problems), WithModules("PROBLEM-NAMING-MIB"), WithStrictness(mib.StrictnessPermissive))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	caseDiags = countDiagnostics(m, "bad-identifier-case")
	testutil.Equal(t, 0, caseDiags, "expected no bad-identifier-case diagnostics in permissive mode")
}

func TestMissingModuleIdentityEmitsDiagnostic(t *testing.T) {
	m := loadViolationMIB(t, "MISSING-IDENTITY-MIB", mib.StrictnessStrict)

	identityDiags := countDiagnostics(m, "missing-module-identity")
	testutil.Equal(t, 1, identityDiags, "expected 1 missing-module-identity diagnostic")
}

func TestMissingImportFailsInStrictMode(t *testing.T) {
	m := loadViolationMIB(t, "MISSING-IMPORT-TEST-MIB", mib.StrictnessStrict)

	unresolved := m.Unresolved()
	var oidUnresolved int
	for _, u := range unresolved {
		if u.Kind == mib.UnresolvedOID {
			oidUnresolved++
		}
	}
	testutil.Greater(t, oidUnresolved, 0, "strict mode should have unresolved OID references")

	testObj := m.Object("testObject")
	testutil.Nil(t, testObj, "testObject should not resolve in strict mode")
}

func TestMissingImportWorksInPermissiveMode(t *testing.T) {
	m := loadViolationMIB(t, "MISSING-IMPORT-TEST-MIB", mib.StrictnessPermissive)

	unresolved := m.Unresolved()
	var oidUnresolved int
	for _, u := range unresolved {
		if u.Kind == mib.UnresolvedOID && u.Module == "MISSING-IMPORT-TEST-MIB" {
			oidUnresolved++
		}
	}
	testutil.Equal(t, 0, oidUnresolved, "permissive mode should resolve enterprises via fallback")

	testObj := m.Object("testObject")
	testutil.NotNil(t, testObj, "testObject should resolve in permissive mode")
}

func TestMissingImportFailsInNormalMode(t *testing.T) {
	m := loadViolationMIB(t, "MISSING-IMPORT-TEST-MIB", mib.StrictnessNormal)

	unresolved := m.Unresolved()
	var oidUnresolved int
	for _, u := range unresolved {
		if u.Kind == mib.UnresolvedOID && u.Module == "MISSING-IMPORT-TEST-MIB" {
			oidUnresolved++
		}
	}
	testutil.Greater(t, oidUnresolved, 0, "normal mode should have unresolved OID references")
}

func TestDiagnosticThresholdEnforced(t *testing.T) {
	corpus := mustDirTree(t, "testdata/corpus/primary")
	violations := mustDirTree(t, "testdata/strictness/violations")

	ctx := context.Background()

	// MISSING-IMPORT-TEST-MIB produces Error-level diagnostics for unresolved OIDs.
	// With FailAt=mib.SeverityError, Load should return the Mib and an error.
	cfg := mib.DiagnosticConfig{
		Level:  mib.StrictnessStrict,
		FailAt: mib.SeverityError,
	}
	m, err := Load(ctx, WithSource(corpus, violations), WithModules("MISSING-IMPORT-TEST-MIB"), WithDiagnosticConfig(cfg))
	testutil.Error(t, err, "Load should error when diagnostics exceed FailAt threshold")
	testutil.True(t, errors.Is(err, ErrDiagnosticThreshold), "error should wrap ErrDiagnosticThreshold")
	testutil.NotNil(t, m, "Load should return non-nil Mib even on threshold failure")
	testutil.NotNil(t, m.Module("MISSING-IMPORT-TEST-MIB"), "module should still be loaded")
}

func TestDiagnosticThresholdNotTriggered(t *testing.T) {
	// Same MIB but with default FailAt=SeveritySevere.
	// Error-level diagnostics should not trigger failure.
	m := loadViolationMIB(t, "MISSING-IMPORT-TEST-MIB", mib.StrictnessStrict)
	testutil.NotNil(t, m, "Mib should be returned")
}

// fakeSource is a test Source that returns pre-configured results.
type fakeSource struct {
	prefix  string // path prefix (default "fake")
	modules map[string]fakeModule
}

type fakeModule struct {
	findErr error  // error from Find
	content []byte // content to return from Find (if findErr is nil)
}

func (f *fakeSource) Find(name string) (FindResult, error) {
	m, ok := f.modules[name]
	if !ok {
		return FindResult{}, fs.ErrNotExist
	}
	if m.findErr != nil {
		return FindResult{}, m.findErr
	}
	prefix := f.prefix
	if prefix == "" {
		prefix = "fake"
	}
	return FindResult{Content: m.content, Path: prefix + ":" + name}, nil
}

func (f *fakeSource) ListModules() ([]string, error) {
	var names []string
	for name := range f.modules {
		names = append(names, name)
	}
	return names, nil
}

func TestSourcePrecedenceDeterministic(t *testing.T) {
	// Two sources both provide the same module name. The first source
	// should always win, regardless of load path (WithModules or full-load).
	mibContent := []byte(`
TEST-PRECEDENCE-MIB DEFINITIONS ::= BEGIN
IMPORTS
    MODULE-IDENTITY, Integer32
        FROM SNMPv2-SMI;

testPrecedenceMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "test"
    CONTACT-INFO "test"
    DESCRIPTION  "test"
    ::= { 1 3 6 1 4 1 99999 }

END
`)

	src1 := &fakeSource{
		prefix:  "src1",
		modules: map[string]fakeModule{"TEST-PRECEDENCE-MIB": {content: mibContent}},
	}
	src2 := &fakeSource{
		prefix:  "src2",
		modules: map[string]fakeModule{"TEST-PRECEDENCE-MIB": {content: mibContent}},
	}

	ctx := context.Background()

	// WithModules path (loadModulesByName via findModule)
	t.Run("WithModules", func(t *testing.T) {
		m, err := Load(ctx, WithSource(src1, src2), WithModules("TEST-PRECEDENCE-MIB"),
			WithStrictness(mib.StrictnessSilent))
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		mod := m.Module("TEST-PRECEDENCE-MIB")
		testutil.NotNil(t, mod, "module should be found")
		testutil.Equal(t, "src1:TEST-PRECEDENCE-MIB", mod.SourcePath(),
			"WithModules should pick first source")
	})

	// Full-load path (loadAllModules)
	t.Run("FullLoad", func(t *testing.T) {
		m, err := Load(ctx, WithSource(src1, src2),
			WithStrictness(mib.StrictnessSilent))
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		mod := m.Module("TEST-PRECEDENCE-MIB")
		testutil.NotNil(t, mod, "module should be found")
		testutil.Equal(t, "src1:TEST-PRECEDENCE-MIB", mod.SourcePath(),
			"full-load should pick first source")
	})
}

func TestFindModuleReturnsContent(t *testing.T) {
	want := []byte("test content")
	src := &fakeSource{modules: map[string]fakeModule{
		"MOD": {content: want},
	}}
	got, err := findModule([]Source{src}, "MOD")
	testutil.NoError(t, err, "findModule")
	testutil.Equal(t, string(want), string(got.Content), "content")
}

func TestFindModuleSkipsNotExist(t *testing.T) {
	want := []byte("from second source")
	src1 := &fakeSource{modules: map[string]fakeModule{}}
	src2 := &fakeSource{modules: map[string]fakeModule{
		"MOD": {content: want},
	}}
	got, err := findModule([]Source{src1, src2}, "MOD")
	testutil.NoError(t, err, "findModule")
	testutil.Equal(t, string(want), string(got.Content), "content from second source")
}

func TestFindModuleNotFound(t *testing.T) {
	src := &fakeSource{modules: map[string]fakeModule{}}
	_, err := findModule([]Source{src}, "MISSING")
	testutil.True(t, errors.Is(err, fs.ErrNotExist), "should return fs.ErrNotExist, got %v", err)
}

func TestFindModulePropagatesFindError(t *testing.T) {
	permErr := errors.New("permission denied")
	src1 := &fakeSource{modules: map[string]fakeModule{
		"MOD": {findErr: permErr},
	}}
	src2 := &fakeSource{modules: map[string]fakeModule{
		"MOD": {content: []byte("ok")},
	}}
	_, err := findModule([]Source{src1, src2}, "MOD")
	testutil.True(t, errors.Is(err, permErr),
		"should propagate Find error, got %v", err)
}

func loadInvalidMIB(t testing.TB, name string, level mib.StrictnessLevel) *mib.Mib {
	t.Helper()
	corpus := mustDirTree(t, "testdata/corpus/primary")
	invalid := mustDirTree(t, "testdata/strictness/invalid")
	ctx := context.Background()
	m, err := Load(ctx, WithSource(corpus, invalid), WithModules(name), WithStrictness(level))
	if err != nil {
		t.Fatalf("Load(%s) failed: %v", name, err)
	}
	return m
}

func moduleObjects(m *mib.Mib, moduleName string) []*mib.Object {
	var result []*mib.Object
	for _, o := range m.Objects() {
		if o.Module().Name() == moduleName {
			result = append(result, o)
		}
	}
	return result
}

func moduleDiagnostics(m *mib.Mib, moduleName string) []mib.Diagnostic {
	var result []mib.Diagnostic
	for _, d := range m.Diagnostics() {
		if d.Module == moduleName {
			result = append(result, d)
		}
	}
	return result
}

func TestInvalidSyntaxMIBProducesNoBrokenObjects(t *testing.T) {
	levels := []struct {
		name  string
		level mib.StrictnessLevel
	}{
		{"strict", mib.StrictnessStrict},
		{"normal", mib.StrictnessNormal},
		{"permissive", mib.StrictnessPermissive},
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadInvalidMIB(t, "INVALID-SYNTAX-MIB", lvl.level)

			objs := moduleObjects(m, "INVALID-SYNTAX-MIB")
			testutil.Equal(t, 0, len(objs),
				"broken OBJECT-TYPE should not produce objects at %s level", lvl.name)

			diags := moduleDiagnostics(m, "INVALID-SYNTAX-MIB")
			testutil.Greater(t, len(diags), 0,
				"should emit diagnostic for missing SYNTAX at %s level", lvl.name)
		})
	}
}

func TestInvalidTruncatedMIBProducesNoObjects(t *testing.T) {
	levels := []struct {
		name  string
		level mib.StrictnessLevel
	}{
		{"strict", mib.StrictnessStrict},
		{"normal", mib.StrictnessNormal},
		{"permissive", mib.StrictnessPermissive},
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			m := loadInvalidMIB(t, "INVALID-TRUNCATED-MIB", lvl.level)

			objs := moduleObjects(m, "INVALID-TRUNCATED-MIB")
			testutil.Equal(t, 0, len(objs),
				"truncated OBJECT-TYPE should not produce objects at %s level", lvl.name)

			diags := moduleDiagnostics(m, "INVALID-TRUNCATED-MIB")
			testutil.Greater(t, len(diags), 0,
				"should emit diagnostic for truncated definition at %s level", lvl.name)
		})
	}
}

func TestInvalidDuplicateOIDMIBBothObjectsLoad(t *testing.T) {
	m := loadInvalidMIB(t, "INVALID-DUPLICATE-OID-MIB", mib.StrictnessPermissive)

	objs := moduleObjects(m, "INVALID-DUPLICATE-OID-MIB")
	testutil.Equal(t, 2, len(objs),
		"both duplicate-OID objects should load")

	if len(objs) == 2 {
		testutil.Equal(t, objs[0].OID().String(), objs[1].OID().String(),
			"duplicate objects should share the same OID")
	}
}
