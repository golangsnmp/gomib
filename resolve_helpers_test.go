package gomib

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

const fixtureDir = "testdata/fixtures/netsnmp"

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
		src, err := DirTree("testdata/corpus/primary")
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
	return filepath.Join(fixtureDir, module+".json")
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

// mustDirTree calls DirTree and fails the test on error.
func mustDirTree(t testing.TB, path string) Source {
	t.Helper()
	src, err := DirTree(path)
	if err != nil {
		t.Fatalf("DirTree(%s) failed: %v", path, err)
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

// loadCorpusMIB loads a single module from the primary corpus with optional
// Load options (e.g. WithStrictness). Used for tests that need real MIBs but
// not the synthetic problem corpus.
func loadCorpusMIB(t testing.TB, name string, opts ...LoadOption) *mib.Mib {
	t.Helper()
	corpus := mustDirTree(t, "testdata/corpus/primary")
	allOpts := append([]LoadOption{WithSource(corpus), WithModules(name)}, opts...)
	m, err := Load(context.Background(), allOpts...)
	if err != nil {
		t.Fatalf("Load(%s) failed: %v", name, err)
	}
	return m
}

// loadAtStrictness loads a module from both the primary corpus and the
// problems directory at a specific strictness level.
func loadAtStrictness(t testing.TB, name string, level mib.StrictnessLevel) *mib.Mib {
	t.Helper()
	corpus := mustDirTree(t, "testdata/corpus/primary")
	problems := mustDirTree(t, "testdata/corpus/problems")
	m, err := Load(context.Background(), WithSource(corpus, problems), WithModules(name), WithStrictness(level))
	if err != nil {
		t.Fatalf("Load(%s, %s) failed: %v", name, level, err)
	}
	return m
}

// loadViolationMIB loads a module from the primary corpus and the strictness
// violations directory at a specific strictness level.
func loadViolationMIB(t testing.TB, name string, level mib.StrictnessLevel) *mib.Mib {
	t.Helper()
	corpus := mustDirTree(t, "testdata/corpus/primary")
	violations := mustDirTree(t, "testdata/strictness/violations")
	m, err := Load(context.Background(), WithSource(corpus, violations), WithModules(name), WithStrictness(level))
	if err != nil {
		t.Fatalf("Load(%s, %s) failed: %v", name, level, err)
	}
	return m
}
