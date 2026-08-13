package gomib_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

func TestLoadSingleMIB(t *testing.T) {
	src := requireDir(t, testutil.PrimaryCorpusDir(t))

	ctx := context.Background()
	m, err := gomib.Load(ctx, gomib.WithSource(src), gomib.WithModules("IF-MIB"))
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

	src := requireDir(t, testutil.PrimaryCorpusDir(t))

	ctx := context.Background()
	m, err := gomib.Load(ctx, gomib.WithSource(src))
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
	src, err := gomib.Dir(testutil.PrimaryCorpusDir(t) + "/ietf")
	if err != nil {
		t.Fatalf("Dir failed: %v", err)
	}

	names, err := src.ListModules()
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}
	testutil.Greater(t, len(names), 0, "should list modules")
}

func TestDirSourceRecursive(t *testing.T) {
	src, err := gomib.Dir(testutil.PrimaryCorpusDir(t))
	if err != nil {
		t.Fatalf("Dir failed: %v", err)
	}

	names, err := src.ListModules()
	if err != nil {
		t.Fatalf("ListModules failed: %v", err)
	}
	testutil.Greater(t, len(names), 10, "should list many modules recursively")
}

func TestMultiSource(t *testing.T) {
	primary := requireDir(t, testutil.PrimaryCorpusDir(t))
	problems := requireDir(t, testutil.ProblemsCorpusDir(t))

	ctx := context.Background()
	m, err := gomib.Load(ctx, gomib.WithSource(primary, problems), gomib.WithModules("IF-MIB"))
	if err != nil {
		t.Fatalf("Load from multi source failed: %v", err)
	}
	testutil.NotNil(t, m.Module("IF-MIB"), "should find IF-MIB from primary source")
}

func TestLoadNonexistentModule(t *testing.T) {
	src := requireDir(t, testutil.PrimaryCorpusDir(t))

	ctx := context.Background()
	m, err := gomib.Load(ctx, gomib.WithSource(src), gomib.WithModules("TOTALLY-FAKE-MIB-THAT-DOES-NOT-EXIST"))
	testutil.Error(t, err, "Load should error for nonexistent module")
	testutil.NotNil(t, m, "Load should return a Mib even for nonexistent module")
	testutil.True(t, errors.Is(err, gomib.ErrMissingModules), "error should wrap gomib.ErrMissingModules")

	mod := m.Module("TOTALLY-FAKE-MIB-THAT-DOES-NOT-EXIST")
	testutil.Nil(t, mod, "nonexistent module should not be in the result")
}

func TestLoadMissingModuleWithValidModule(t *testing.T) {
	src := requireDir(t, testutil.PrimaryCorpusDir(t))

	ctx := context.Background()
	m, err := gomib.Load(ctx, gomib.WithSource(src), gomib.WithModules("IF-MIB", "NONEXISTENT-MIB"))
	testutil.Error(t, err, "Load should error for missing module")
	testutil.True(t, errors.Is(err, gomib.ErrMissingModules), "error should wrap gomib.ErrMissingModules")
	testutil.NotNil(t, m, "Load should return a Mib with partial results")
	testutil.NotNil(t, m.Module("IF-MIB"), "IF-MIB should still be loaded")
	testutil.Nil(t, m.Module("NONEXISTENT-MIB"), "NONEXISTENT-MIB should not be found")
}

func TestLoadNoSources(t *testing.T) {
	ctx := context.Background()
	_, err := gomib.Load(ctx)
	testutil.Error(t, err, "loading with no sources should fail")
}

func TestLookups(t *testing.T) {
	m := loadTestMIB(t)

	t.Run("Node", func(t *testing.T) {
		testutil.NotNil(t, m.Node("ifIndex"), "should find ifIndex by name")
		testutil.Nil(t, m.Node("totallyNonExistentSymbol"), "nonexistent symbol should return nil")
	})

	t.Run("Object", func(t *testing.T) {
		obj := m.Object("sysDescr")
		testutil.NotNil(t, obj, "should find sysDescr by name")
		testutil.Equal(t, "sysDescr", obj.Name(), "object name")
	})

	t.Run("Type", func(t *testing.T) {
		typ := m.Type("DisplayString")
		testutil.NotNil(t, typ, "Type(DisplayString)")
		testutil.Equal(t, "DisplayString", typ.Name(), "type name")
		testutil.True(t, typ.IsTextualConvention(), "DisplayString should be a TC")
	})

	t.Run("Notification", func(t *testing.T) {
		notif := m.Notification("linkDown")
		testutil.NotNil(t, notif, "Notification(linkDown)")
		testutil.Equal(t, "linkDown", notif.Name(), "notification name")
	})

	t.Run("Collections", func(t *testing.T) {
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

		objects1 := m.Objects()
		objects2 := m.Objects()
		testutil.Equal(t, len(objects1), len(objects2), "Objects() should be deterministic across calls")
		testutil.Greater(t, len(objects1), 0, "should have objects")
		testutil.Greater(t, len(m.Tables()), 0, "should have tables")
		testutil.Greater(t, len(m.Scalars()), 0, "should have scalars")
	})

	t.Run("NodesIteration", func(t *testing.T) {
		count := 0
		for range m.Nodes() {
			count++
		}
		testutil.Greater(t, count, 0, "should have nodes")
		testutil.Equal(t, m.NodeCount(), count, "NodeCount should match iteration count")
	})
}

func TestStrictMIBsPassAtStrictLevel(t *testing.T) {
	corpus := requireDir(t, testutil.PrimaryCorpusDir(t))
	strict := requireDir(t, testutil.TestdataDir(t, "strictness", "strict"))

	tests := []string{"STRICT-TEST-MIB", "STRICT-TABLE-MIB"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			m, err := gomib.Load(ctx, gomib.WithSource(corpus, strict), gomib.WithModules(name), gomib.WithResolverStrictness(mib.ResolverStrict))
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
	m := loadViolationMIB(t, "UNDERSCORE-TEST-MIB", mib.ResolverStrict)

	underscoreDiags := countDiagnostics(m, "identifier-underscore")
	testutil.Equal(t, 2, underscoreDiags, "expected 2 identifier-underscore diagnostics")

	m = loadViolationMIB(t, "UNDERSCORE-TEST-MIB", mib.ResolverPermissive)

	underscoreDiags = countDiagnostics(m, "identifier-underscore")
	testutil.Equal(t, 2, underscoreDiags, "resolver mode should not change identifier diagnostic reporting")
}

func TestHyphenEndViolationEmitsDiagnostic(t *testing.T) {
	m := loadViolationMIB(t, "HYPHEN-END-TEST-MIB", mib.ResolverStrict)

	hyphenDiags := countDiagnostics(m, "identifier-hyphen-end")
	testutil.Equal(t, 1, hyphenDiags, "expected 1 identifier-hyphen-end diagnostic")
}

func TestLongIdentifierViolationEmitsDiagnostic(t *testing.T) {
	m := loadViolationMIB(t, "LONG-IDENT-TEST-MIB", mib.ResolverStrict)

	lengthDiags := countDiagnostics(m, "identifier-length-64")
	testutil.Equal(t, 1, lengthDiags, "expected 1 identifier-length-64 diagnostic")
}

func TestUppercaseIdentifierEmitsDiagnostic(t *testing.T) {
	corpus := requireDir(t, testutil.PrimaryCorpusDir(t))
	problems := requireDir(t, testutil.ProblemsCorpusDir(t))

	ctx := context.Background()
	diag := mib.DiagnosticConfig{Reporting: mib.ReportingVerbose, FailAt: mib.SeverityFatal}

	m, err := gomib.Load(ctx,
		gomib.WithSource(corpus, problems),
		gomib.WithModules("PROBLEM-NAMING-MIB"),
		gomib.WithResolverStrictness(mib.ResolverNormal),
		gomib.WithDiagnosticConfig(diag),
	)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	caseDiags := countDiagnostics(m, "bad-identifier-case")
	testutil.Equal(t, 4, caseDiags, "expected 4 bad-identifier-case diagnostics in normal mode")

	node := m.Node("NetEngine8000SysOid")
	testutil.NotNil(t, node, "uppercase identifier should resolve in normal mode")

	m, err = gomib.Load(ctx,
		gomib.WithSource(corpus, problems),
		gomib.WithModules("PROBLEM-NAMING-MIB"),
		gomib.WithResolverStrictness(mib.ResolverPermissive),
		gomib.WithDiagnosticConfig(diag),
	)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	caseDiags = countDiagnostics(m, "bad-identifier-case")
	testutil.Equal(t, 4, caseDiags, "resolver mode should not change identifier case diagnostic reporting")
}

func TestMissingModuleIdentityEmitsDiagnostic(t *testing.T) {
	m := loadViolationMIB(t, "MISSING-IDENTITY-MIB", mib.ResolverStrict)

	identityDiags := countDiagnostics(m, "missing-module-identity")
	testutil.Equal(t, 1, identityDiags, "expected 1 missing-module-identity diagnostic")
}

func TestMissingImportFailsInStrictMode(t *testing.T) {
	m := loadViolationMIB(t, "MISSING-IMPORT-TEST-MIB", mib.ResolverStrict)

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
	m := loadViolationMIB(t, "MISSING-IMPORT-TEST-MIB", mib.ResolverPermissive)

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

func TestMissingImportWorksInNormalMode(t *testing.T) {
	m := loadViolationMIB(t, "MISSING-IMPORT-TEST-MIB", mib.ResolverNormal)

	unresolved := m.Unresolved()
	var oidUnresolved int
	for _, u := range unresolved {
		if u.Kind == mib.UnresolvedOID && u.Module == "MISSING-IMPORT-TEST-MIB" {
			oidUnresolved++
		}
	}
	testutil.Equal(t, 0, oidUnresolved, "normal mode should resolve enterprises via constrained fallback")

	testObj := m.Object("testObject")
	testutil.NotNil(t, testObj, "testObject should resolve in normal mode")
}

func TestDiagnosticThresholdEnforced(t *testing.T) {
	corpus := requireDir(t, testutil.PrimaryCorpusDir(t))
	violations := requireDir(t, testutil.TestdataDir(t, "strictness", "violations"))

	ctx := context.Background()

	// MISSING-IMPORT-TEST-MIB produces Error-level diagnostics for unresolved OIDs.
	// With FailAt=mib.SeverityError, Load should return the Mib and an error.
	cfg := mib.DiagnosticConfig{
		Reporting: mib.ReportingVerbose,
		FailAt:    mib.SeverityError,
	}
	m, err := gomib.Load(ctx,
		gomib.WithSource(corpus, violations),
		gomib.WithModules("MISSING-IMPORT-TEST-MIB"),
		gomib.WithResolverStrictness(mib.ResolverStrict),
		gomib.WithDiagnosticConfig(cfg),
	)
	testutil.Error(t, err, "Load should error when diagnostics exceed FailAt threshold")
	testutil.True(t, errors.Is(err, gomib.ErrDiagnosticThreshold), "error should wrap gomib.ErrDiagnosticThreshold")
	testutil.NotNil(t, m, "Load should return non-nil Mib even on threshold failure")
	testutil.NotNil(t, m.Module("MISSING-IMPORT-TEST-MIB"), "module should still be loaded")
}

func TestDiagnosticThresholdNotTriggered(t *testing.T) {
	// Same MIB but with default FailAt=SeveritySevere.
	// Error-level diagnostics should not trigger failure.
	m := loadViolationMIB(t, "MISSING-IMPORT-TEST-MIB", mib.ResolverStrict)
	testutil.NotNil(t, m, "Mib should be returned")
}

func TestDiagnosticOverridePromotionCausesLoadFailure(t *testing.T) {
	const source = `
OVERRIDE-PROMOTION-MIB DEFINITIONS ::= BEGIN
VendorEnum ::= INTEGER { vendorSentinel(4294967295) }
END
`
	src := &fakeSource{modules: map[string]fakeModule{
		"OVERRIDE-PROMOTION-MIB": {content: []byte(source)},
	}}
	cfg := mib.DefaultConfig()
	cfg.Overrides = map[string]mib.Severity{types.DiagEnumValueOutOfRange: mib.SeveritySevere}

	m, err := gomib.Load(context.Background(),
		gomib.WithSource(src),
		gomib.WithModules("OVERRIDE-PROMOTION-MIB"),
		gomib.WithDiagnosticConfig(cfg),
	)
	testutil.True(t, errors.Is(err, gomib.ErrDiagnosticThreshold), "promotion should cause threshold failure, got %v", err)
	testutil.NotNil(t, m, "Load should return the Mib on threshold failure")
	diags := moduleDiagnostics(m, "OVERRIDE-PROMOTION-MIB")
	for _, d := range diags {
		if d.Code == types.DiagEnumValueOutOfRange {
			testutil.Equal(t, mib.SeveritySevere, d.Severity, "promoted severity")
			return
		}
	}
	t.Fatal("expected promoted enum bounds diagnostic")
}

func TestDiagnosticOverrideRetainedDemotionAvoidsLoadFailure(t *testing.T) {
	const source = `
OVERRIDE-DEMOTION-MIB DEFINITIONS ::= BEGIN
BadEnum ::= INTEGER { duplicate(1), duplicate(2) }
END
`
	src := &fakeSource{modules: map[string]fakeModule{
		"OVERRIDE-DEMOTION-MIB": {content: []byte(source)},
	}}
	cfg := mib.QuietConfig()
	cfg.FailAt = mib.SeverityError
	cfg.Overrides = map[string]mib.Severity{types.DiagEnumNameRedefinition: mib.SeverityInfo}

	m, err := gomib.Load(context.Background(),
		gomib.WithSource(src),
		gomib.WithModules("OVERRIDE-DEMOTION-MIB"),
		gomib.WithDiagnosticConfig(cfg),
	)
	testutil.NoError(t, err, "retained demotion should avoid threshold failure")
	for _, d := range moduleDiagnostics(m, "OVERRIDE-DEMOTION-MIB") {
		if d.Code == types.DiagEnumNameRedefinition {
			testutil.Equal(t, mib.SeverityInfo, d.Severity, "demoted severity")
			return
		}
	}
	t.Fatal("expected demoted diagnostic to be retained")
}

func TestDiagnosticOverrideIgnoreAndFatalPolicies(t *testing.T) {
	const source = `
OVERRIDE-POLICY-MIB DEFINITIONS ::= BEGIN
VendorEnum ::= INTEGER { vendorSentinel(4294967295) }
END
`
	src := &fakeSource{modules: map[string]fakeModule{
		"OVERRIDE-POLICY-MIB": {content: []byte(source)},
	}}

	t.Run("ignore suppresses non-fatal promotion", func(t *testing.T) {
		cfg := mib.VerboseConfig()
		cfg.FailAt = mib.SeveritySevere
		cfg.Overrides = map[string]mib.Severity{types.DiagEnumValueOutOfRange: mib.SeveritySevere}
		cfg.Ignore = []string{types.DiagEnumValueOutOfRange}
		m, err := gomib.Load(context.Background(),
			gomib.WithSource(src),
			gomib.WithModules("OVERRIDE-POLICY-MIB"),
			gomib.WithDiagnosticConfig(cfg),
		)
		testutil.NoError(t, err, "ignored non-fatal diagnostic should not fail")
		for _, d := range m.Diagnostics() {
			if d.Code == types.DiagEnumValueOutOfRange {
				t.Fatal("ignored diagnostic should not be retained")
			}
		}
	})

	t.Run("effective fatal bypasses ignore and reporting", func(t *testing.T) {
		cfg := mib.SilentConfig()
		cfg.Overrides = map[string]mib.Severity{types.DiagEnumValueOutOfRange: mib.SeverityFatal}
		cfg.Ignore = []string{types.DiagEnumValueOutOfRange}
		m, err := gomib.Load(context.Background(),
			gomib.WithSource(src),
			gomib.WithModules("OVERRIDE-POLICY-MIB"),
			gomib.WithDiagnosticConfig(cfg),
		)
		testutil.True(t, errors.Is(err, gomib.ErrDiagnosticThreshold), "fatal diagnostic should fail, got %v", err)
		for _, d := range m.Diagnostics() {
			if d.Code == types.DiagEnumValueOutOfRange {
				testutil.Equal(t, mib.SeverityFatal, d.Severity, "fatal severity")
				return
			}
		}
		t.Fatal("effective fatal diagnostic should be retained")
	})
}

// fakeSource is a test gomib.Source that returns pre-configured results.
type fakeSource struct {
	prefix  string // path prefix (default "fake")
	modules map[string]fakeModule
}

type fakeModule struct {
	findErr error  // error from Find
	content []byte // content to return from Find (if findErr is nil)
}

func (f *fakeSource) Find(name string) (gomib.FindResult, error) {
	m, ok := f.modules[name]
	if !ok {
		return gomib.FindResult{}, fs.ErrNotExist
	}
	if m.findErr != nil {
		return gomib.FindResult{}, m.findErr
	}
	prefix := f.prefix
	if prefix == "" {
		prefix = "fake"
	}
	return gomib.FindResult{Content: m.content, Path: prefix + ":" + name}, nil
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
		m, err := gomib.Load(ctx, gomib.WithSource(src1, src2), gomib.WithModules("TEST-PRECEDENCE-MIB"),
			gomib.WithDiagnosticConfig(mib.SilentConfig()))
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
		m, err := gomib.Load(ctx, gomib.WithSource(src1, src2),
			gomib.WithDiagnosticConfig(mib.SilentConfig()))
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		mod := m.Module("TEST-PRECEDENCE-MIB")
		testutil.NotNil(t, mod, "module should be found")
		testutil.Equal(t, "src1:TEST-PRECEDENCE-MIB", mod.SourcePath(),
			"full-load should pick first source")
	})
}

// findModule is a test helper that searches sources in order for the named module.
func findModule(sources []gomib.Source, name string) (gomib.FindResult, error) {
	return gomib.Multi(sources...).Find(name)
}

func TestFindModuleReturnsContent(t *testing.T) {
	want := []byte("test content")
	src := &fakeSource{modules: map[string]fakeModule{
		"MOD": {content: want},
	}}
	got, err := findModule([]gomib.Source{src}, "MOD")
	testutil.NoError(t, err, "findModule")
	testutil.Equal(t, string(want), string(got.Content), "content")
}

func TestFindModuleSkipsNotExist(t *testing.T) {
	want := []byte("from second source")
	src1 := &fakeSource{modules: map[string]fakeModule{}}
	src2 := &fakeSource{modules: map[string]fakeModule{
		"MOD": {content: want},
	}}
	got, err := findModule([]gomib.Source{src1, src2}, "MOD")
	testutil.NoError(t, err, "findModule")
	testutil.Equal(t, string(want), string(got.Content), "content from second source")
}

func TestFindModuleNotFound(t *testing.T) {
	src := &fakeSource{modules: map[string]fakeModule{}}
	_, err := findModule([]gomib.Source{src}, "MISSING")
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
	_, err := findModule([]gomib.Source{src1, src2}, "MOD")
	testutil.True(t, errors.Is(err, permErr),
		"should propagate Find error, got %v", err)
}

func loadInvalidMIB(t testing.TB, name string, level mib.ResolverStrictness) *mib.Mib {
	t.Helper()
	corpus := requireDir(t, testutil.PrimaryCorpusDir(t))
	invalid := requireDir(t, testutil.TestdataDir(t, "strictness", "invalid"))
	ctx := context.Background()
	diag := mib.DiagnosticConfig{Reporting: mib.ReportingVerbose, FailAt: mib.SeverityFatal}
	m, err := gomib.Load(ctx,
		gomib.WithSource(corpus, invalid),
		gomib.WithModules(name),
		gomib.WithResolverStrictness(level),
		gomib.WithDiagnosticConfig(diag),
	)
	if err != nil {
		t.Fatalf("gomib.Load(%s) failed: %v", name, err)
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
		level mib.ResolverStrictness
	}{
		{"strict", mib.ResolverStrict},
		{"normal", mib.ResolverNormal},
		{"permissive", mib.ResolverPermissive},
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
		level mib.ResolverStrictness
	}{
		{"strict", mib.ResolverStrict},
		{"normal", mib.ResolverNormal},
		{"permissive", mib.ResolverPermissive},
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
	m := loadInvalidMIB(t, "INVALID-DUPLICATE-OID-MIB", mib.ResolverPermissive)

	objs := moduleObjects(m, "INVALID-DUPLICATE-OID-MIB")
	testutil.Equal(t, 2, len(objs),
		"both duplicate-OID objects should load")

	if len(objs) == 2 {
		testutil.Equal(t, objs[0].OID().String(), objs[1].OID().String(),
			"duplicate objects should share the same OID")
	}
}

func TestMultiModuleFileLoadAll(t *testing.T) {
	// PROBLEM-MULTIMOD.mib contains PROBLEM-MULTIMOD-BASE-MIB and
	// PROBLEM-MULTIMOD-MIB. Both should be discovered and loaded.
	corpus := requireDir(t, testutil.PrimaryCorpusDir(t))
	problems := requireDir(t, testutil.ProblemsCorpusDir(t))

	ctx := context.Background()
	m, err := gomib.Load(ctx, gomib.WithSource(corpus, problems))
	testutil.NoError(t, err, "Load")

	testutil.NotNil(t, m.Module("PROBLEM-MULTIMOD-BASE-MIB"),
		"base module from multi-module file should be loaded")
	testutil.NotNil(t, m.Module("PROBLEM-MULTIMOD-MIB"),
		"leaf module from multi-module file should be loaded")
}

func TestMultiModuleFileLoadByName(t *testing.T) {
	// Loading the leaf module by name should also pick up the base module
	// (it's in the same file, and the leaf imports from the base).
	corpus := requireDir(t, testutil.PrimaryCorpusDir(t))
	problems := requireDir(t, testutil.ProblemsCorpusDir(t))

	ctx := context.Background()
	m, err := gomib.Load(ctx, gomib.WithSource(corpus, problems),
		gomib.WithModules("PROBLEM-MULTIMOD-MIB"))
	testutil.NoError(t, err, "Load")

	testutil.NotNil(t, m.Module("PROBLEM-MULTIMOD-MIB"),
		"leaf module should be loaded")
	testutil.NotNil(t, m.Module("PROBLEM-MULTIMOD-BASE-MIB"),
		"base module should be loaded (same file + import dependency)")

	// The leaf module's scalar should resolve.
	obj := m.Object("problemMultimodScalar")
	testutil.NotNil(t, obj, "problemMultimodScalar should resolve")
}

func TestMultiModuleFileLoadBaseByName(t *testing.T) {
	// Loading the base module by name should find it even though the
	// filename doesn't match.
	corpus := requireDir(t, testutil.PrimaryCorpusDir(t))
	problems := requireDir(t, testutil.ProblemsCorpusDir(t))

	ctx := context.Background()
	m, err := gomib.Load(ctx, gomib.WithSource(corpus, problems),
		gomib.WithModules("PROBLEM-MULTIMOD-BASE-MIB"))
	testutil.NoError(t, err, "Load")

	testutil.NotNil(t, m.Module("PROBLEM-MULTIMOD-BASE-MIB"),
		"base module should be found by content-derived name")
}
