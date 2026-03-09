package gomib

import (
	"context"
	"sync"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

var (
	fixtureModules = []string{"IF-MIB", "SNMPv2-MIB", "IP-MIB", "ENTITY-MIB", "BRIDGE-MIB"}
	loadOnce       sync.Once
	loadedMib      *mib.Mib
	loadErr        error
)

// loadTestMIB loads all fixture modules once (via sync.Once) and returns
// the shared Mib, so that tests sharing the same fixture set avoid
// redundant parsing.
func loadTestMIB(t testing.TB) *mib.Mib {
	t.Helper()
	loadOnce.Do(func() {
		src, err := Dir(testutil.PrimaryCorpusDir())
		if err != nil {
			loadErr = err
			return
		}
		loadedMib, loadErr = Load(context.Background(), WithSource(src), WithModules(fixtureModules...))
	})
	if loadErr != nil {
		t.Fatalf("failed to load test MIBs: %v", loadErr)
	}
	return loadedMib
}

func fixturePath(module string) string {
	return testutil.TestdataDir("fixtures", "netsnmp", module+".json")
}

type strictnessCase struct {
	name  string
	level mib.ResolverStrictness
}

var allStrictnessCases = []strictnessCase{
	{name: "strict", level: mib.ResolverStrict},
	{name: "normal", level: mib.ResolverNormal},
	{name: "permissive", level: mib.ResolverPermissive},
}

func loadFixtureNodes(t testing.TB, module string) map[string]*testutil.FixtureNode {
	t.Helper()
	return testutil.LoadFixture(t, fixturePath(module))
}

// isObjectTypeNode filters out containers and conformance nodes that
// don't carry data type information in the fixture format.
func isObjectTypeNode(fn *testutil.FixtureNode) bool {
	switch fn.Type {
	case "", "OTHER", "NOTIFICATION-TYPE", "TRAP-TYPE", "MODULE-IDENTITY",
		"MODULE-COMPLIANCE", "OBJECT-GROUP", "NOTIFICATION-GROUP",
		"AGENT-CAPABILITIES", "OBJECT-IDENTITY":
		return false
	}
	return true
}

func isNotificationNode(fn *testutil.FixtureNode) bool {
	return fn.NodeType == "NOTIFICATION-TYPE" || fn.NodeType == "TRAP-TYPE"
}

// forEachFixtureNode iterates all fixture nodes across all fixture modules,
// calling check for each node that passes the filter.
// If filter is nil, all nodes are visited.
func forEachFixtureNode(t *testing.T, filter func(*testutil.FixtureNode) bool, check func(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode)) {
	t.Helper()
	m := loadTestMIB(t)
	for _, mod := range fixtureModules {
		t.Run(mod, func(t *testing.T) {
			fixture := loadFixtureNodes(t, mod)
			for _, fn := range fixture {
				if filter != nil && !filter(fn) {
					continue
				}
				t.Run(fn.Name, func(t *testing.T) {
					check(t, m, fn)
				})
			}
		})
	}
}

// requireFixtureObject looks up an object by fixture node name and fails the
// test if it is not found.
func requireFixtureObject(t *testing.T, m *mib.Mib, fn *testutil.FixtureNode) *mib.Object {
	t.Helper()
	obj := m.Object(fn.Name)
	if obj == nil {
		t.Fatalf("divergence: gomib does not have object %q", fn.Name)
	}
	return obj
}

func requireObject(t testing.TB, m *mib.Mib, name string) *mib.Object {
	t.Helper()
	obj := m.Object(name)
	if obj == nil {
		t.Fatalf("Object(%s) = nil", name)
	}
	return obj
}

func requireNode(t testing.TB, m *mib.Mib, name string) *mib.Node {
	t.Helper()
	node := m.Node(name)
	if node == nil {
		t.Fatalf("Node(%s) = nil", name)
	}
	return node
}

func requireNotification(t testing.TB, m *mib.Mib, name string) *mib.Notification {
	t.Helper()
	notif := m.Notification(name)
	if notif == nil {
		t.Fatalf("Notification(%s) = nil", name)
	}
	return notif
}

func requireGroup(t testing.TB, m *mib.Mib, name string) *mib.Group {
	t.Helper()
	group := m.Group(name)
	if group == nil {
		t.Fatalf("Group(%s) = nil", name)
	}
	return group
}

// mustDir calls Dir and fails the test on error.
func mustDir(t testing.TB, path string) Source {
	t.Helper()
	src, err := Dir(path)
	if err != nil {
		t.Fatalf("Dir(%s) failed: %v", path, err)
	}
	return src
}

// countDiagnostics returns the number of diagnostics with the given code.
func countDiagnostics(m *mib.Mib, code string) int {
	var n int
	for _, d := range m.Diagnostics() {
		if d.Code == code {
			n++
		}
	}
	return n
}

func countModuleDiagnostics(m *mib.Mib, module, code string) int {
	var n int
	for _, d := range m.Diagnostics() {
		if d.Module == module && d.Code == code {
			n++
		}
	}
	return n
}

// unresolvedSymbols returns unresolved symbols of a given kind for one module,
// keyed by symbol name for compact presence checks in tests.
func unresolvedSymbols(m *mib.Mib, module string, kind mib.UnresolvedKind) map[string]bool {
	result := make(map[string]bool)
	for _, u := range m.Unresolved() {
		if u.Module == module && u.Kind == kind {
			result[u.Symbol] = true
		}
	}
	return result
}

// loadCorpusMIB loads a single module from the primary corpus with optional
// Load options (e.g. WithResolverStrictness). Used for tests that need real MIBs but
// not the synthetic problem corpus.
func loadCorpusMIB(t testing.TB, name string, opts ...LoadOption) *mib.Mib {
	t.Helper()
	corpus := mustDir(t, testutil.PrimaryCorpusDir())
	allOpts := append([]LoadOption{WithSource(corpus), WithModules(name)}, opts...)
	m, err := Load(context.Background(), allOpts...)
	if err != nil {
		t.Fatalf("Load(%s) failed: %v", name, err)
	}
	return m
}

// loadAtStrictness loads a module from both the primary corpus and the
// problems directory at a specific strictness level.
func loadAtStrictness(t testing.TB, name string, level mib.ResolverStrictness) *mib.Mib {
	t.Helper()
	corpus := mustDir(t, testutil.PrimaryCorpusDir())
	problems := mustDir(t, testutil.ProblemsCorpusDir())
	diag := mib.DiagnosticConfig{Reporting: mib.ReportingVerbose, FailAt: mib.SeverityFatal}
	m, err := Load(context.Background(),
		WithSource(corpus, problems),
		WithModules(name),
		WithResolverStrictness(level),
		WithDiagnosticConfig(diag),
	)
	if err != nil {
		t.Fatalf("Load(%s, %s) failed: %v", name, level, err)
	}
	return m
}

// loadViolationMIB loads a module from the primary corpus and the strictness
// violations directory at a specific strictness level.
func loadViolationMIB(t testing.TB, name string, level mib.ResolverStrictness) *mib.Mib {
	t.Helper()
	corpus := mustDir(t, testutil.PrimaryCorpusDir())
	violations := mustDir(t, testutil.TestdataDir("strictness", "violations"))
	diag := mib.DiagnosticConfig{Reporting: mib.ReportingVerbose, FailAt: mib.SeverityFatal}
	m, err := Load(context.Background(),
		WithSource(corpus, violations),
		WithModules(name),
		WithResolverStrictness(level),
		WithDiagnosticConfig(diag),
	)
	if err != nil {
		t.Fatalf("Load(%s, %s) failed: %v", name, level, err)
	}
	return m
}
