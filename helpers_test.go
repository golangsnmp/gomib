package gomib_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/golangsnmp/gomib"
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
		src, err := gomib.Dir(testutil.PrimaryCorpusDir(t))
		if err != nil {
			loadErr = err
			return
		}
		loadedMib, loadErr = gomib.Load(context.Background(), gomib.WithSource(src), gomib.WithModules(fixtureModules...))
	})
	if loadErr != nil {
		t.Fatalf("failed to load test MIBs: %v", loadErr)
	}
	return loadedMib
}

func fixturePath(t testing.TB, module string) string {
	t.Helper()
	return testutil.TestdataDir(t, "fixtures", "netsnmp", module+".json")
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
	return testutil.LoadFixture(t, fixturePath(t, module))
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

// requireDir calls Dir and fails the test on error.
func requireDir(t testing.TB, path string) gomib.Source {
	t.Helper()
	src, err := gomib.Dir(path)
	if err != nil {
		t.Fatalf("gomib.Dir(%s) failed: %v", path, err)
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

// requireModuleDiagnostic asserts that exactly wantCount diagnostics with the
// given module and code exist, and that each contains all specified message
// fragments. Returns the matching diagnostics for further inspection.
func requireModuleDiagnostic(t testing.TB, m *mib.Mib, module, code string, wantCount int, fragments ...string) []mib.Diagnostic { //nolint:unparam // module varies by caller
	t.Helper()
	var matched []mib.Diagnostic
	for _, d := range m.Diagnostics() {
		if d.Module == module && d.Code == code {
			matched = append(matched, d)
		}
	}
	if len(matched) != wantCount {
		t.Fatalf("expected %d %s diagnostics for %s, got %d", wantCount, code, module, len(matched))
	}
	for _, d := range matched {
		for _, frag := range fragments {
			if !strings.Contains(d.Message, frag) {
				t.Errorf("diagnostic %s message should contain %q, got: %s", code, frag, d.Message)
			}
		}
	}
	return matched
}

// conformanceNodeSpec describes a conformance entity (group, compliance,
// capability) for shared structural tests.
type conformanceNodeSpec struct {
	name       string     // entity name, e.g. "snmpGroup"
	module     string     // defining module, e.g. "SNMPv2-MIB"
	kind       mib.Kind   // expected node kind
	status     mib.Status // expected status
	wrongOwner string     // module that should NOT contain this entity
}

// checkConformanceNode runs the common structural checks shared across groups,
// compliances, and capabilities: global lookup, module-scoped lookup,
// OID-based lookup, node backreference, module backreference, and metadata.
//
// lookupGlobal returns the entity by name from the Mib.
// lookupScoped returns the entity by name from a Module.
// getNode returns the entity's Node.
// getModule returns the entity's Module.
// getOID returns the entity's OID.
// nodeAccessor returns the entity from its Node (e.g. node.Group()).
func checkConformanceNode[E any](
	t *testing.T,
	m *mib.Mib,
	spec conformanceNodeSpec,
	lookupGlobal func(*mib.Mib, string) E,
	lookupScoped func(*mib.Module, string) E,
	getNode func(E) *mib.Node,
	getModule func(E) *mib.Module,
	getOID func(E) mib.OID,
	getStatus func(E) mib.Status,
	getDescription func(E) string,
	nodeAccessor func(*mib.Node) E,
	isNilEntity func(E) bool,
) {
	t.Helper()

	t.Run("GlobalLookup", func(t *testing.T) {
		e := lookupGlobal(m, spec.name)
		if isNilEntity(e) {
			t.Fatalf("%s not found via global lookup", spec.name)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		e := lookupGlobal(m, "noSuch"+spec.name)
		testutil.True(t, isNilEntity(e), "non-existent name should return nil")
	})

	t.Run("ScopedLookup", func(t *testing.T) {
		mod := m.Module(spec.module)
		if mod == nil {
			t.Fatalf("module %s not found", spec.module)
		}
		e := lookupScoped(mod, spec.name)
		if isNilEntity(e) {
			t.Fatalf("%s not found in %s", spec.name, spec.module)
		}
		if spec.wrongOwner != "" {
			wrong := m.Module(spec.wrongOwner)
			if wrong != nil {
				testutil.True(t, isNilEntity(lookupScoped(wrong, spec.name)),
					"%s should not be in %s", spec.name, spec.wrongOwner)
			}
		}
	})

	t.Run("OIDLookup", func(t *testing.T) {
		e := lookupGlobal(m, spec.name)
		if isNilEntity(e) {
			t.Fatalf("%s not found", spec.name)
		}
		oid := getOID(e)
		testutil.Greater(t, len(oid), 0, "OID should not be empty")
		nd := m.NodeByOID(oid)
		testutil.NotNil(t, nd, "NodeByOID should find node")
		fromNode := nodeAccessor(nd)
		if isNilEntity(fromNode) {
			t.Fatalf("node accessor returned nil for %s", spec.name)
		}
	})

	t.Run("Metadata", func(t *testing.T) {
		e := lookupGlobal(m, spec.name)
		if isNilEntity(e) {
			t.Fatalf("%s not found", spec.name)
		}

		node := getNode(e)
		testutil.NotNil(t, node, "Node() should not be nil")
		testutil.Equal(t, spec.name, node.Name(), "node name")
		testutil.Equal(t, spec.kind, node.Kind(), "node kind")

		mod := getModule(e)
		testutil.NotNil(t, mod, "Module() should not be nil")
		testutil.Equal(t, spec.module, mod.Name(), "module name")

		testutil.Equal(t, spec.status, getStatus(e), "status")
		testutil.Greater(t, len(getDescription(e)), 0, "should have a description")
	})

	t.Run("NodeBackreference", func(t *testing.T) {
		nd := m.Node(spec.name)
		if nd == nil {
			t.Fatalf("node %s not found", spec.name)
		}
		e := nodeAccessor(nd)
		if isNilEntity(e) {
			t.Fatalf("node accessor returned nil for %s", spec.name)
		}
	})

	t.Run("NodeModule", func(t *testing.T) {
		nd := m.Node(spec.name)
		if nd == nil {
			t.Fatalf("node %s not found", spec.name)
		}
		mod := nd.Module()
		testutil.NotNil(t, mod, "node should have a Module()")
		testutil.Equal(t, spec.module, mod.Name(), "node module name")
	})
}

func isNilGroup(g *mib.Group) bool { return g == nil }

func isNilCompliance(c *mib.Compliance) bool { return c == nil }

func isNilCapability(c *mib.Capability) bool { return c == nil }

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
func loadCorpusMIB(t testing.TB, name string, opts ...gomib.LoadOption) *mib.Mib {
	t.Helper()
	corpus := requireDir(t, testutil.PrimaryCorpusDir(t))
	allOpts := append([]gomib.LoadOption{gomib.WithSource(corpus), gomib.WithModules(name)}, opts...)
	m, err := gomib.Load(context.Background(), allOpts...)
	if err != nil {
		t.Fatalf("gomib.Load(%s) failed: %v", name, err)
	}
	return m
}

// loadAtStrictness loads a module from both the primary corpus and the
// problems directory at a specific strictness level.
func loadAtStrictness(t testing.TB, name string, level mib.ResolverStrictness) *mib.Mib {
	t.Helper()
	corpus := requireDir(t, testutil.PrimaryCorpusDir(t))
	problems := requireDir(t, testutil.ProblemsCorpusDir(t))
	diag := mib.DiagnosticConfig{Reporting: mib.ReportingVerbose, FailAt: mib.SeverityFatal}
	m, err := gomib.Load(context.Background(),
		gomib.WithSource(corpus, problems),
		gomib.WithModules(name),
		gomib.WithResolverStrictness(level),
		gomib.WithDiagnosticConfig(diag),
	)
	if err != nil {
		t.Fatalf("gomib.Load(%s, %s) failed: %v", name, level, err)
	}
	return m
}

// loadViolationMIB loads a module from the primary corpus and the strictness
// violations directory at a specific strictness level.
func loadViolationMIB(t testing.TB, name string, level mib.ResolverStrictness) *mib.Mib {
	t.Helper()
	corpus := requireDir(t, testutil.PrimaryCorpusDir(t))
	violations := requireDir(t, testutil.TestdataDir(t, "strictness", "violations"))
	diag := mib.DiagnosticConfig{Reporting: mib.ReportingVerbose, FailAt: mib.SeverityFatal}
	m, err := gomib.Load(context.Background(),
		gomib.WithSource(corpus, violations),
		gomib.WithModules(name),
		gomib.WithResolverStrictness(level),
		gomib.WithDiagnosticConfig(diag),
	)
	if err != nil {
		t.Fatalf("gomib.Load(%s, %s) failed: %v", name, level, err)
	}
	return m
}
