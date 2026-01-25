// Package integration provides integration tests against the MIB test corpus.
//
// These tests load the full testdata/corpus/primary/ folder and make assertions
// against the resolved model. Test cases should be cross-validated with net-snmp
// (snmptranslate) to ensure assertions are grounded against a known reference.
//
// # Adding Test Cases
//
// 1. Use snmptranslate to verify the expected value
// 2. Add the test case to the appropriate file (oid_test.go, types_test.go, etc.)
// 3. Document the snmptranslate command used for verification in the NetSnmp field
//
// # File Organization
//
//   - corpus_test.go: Shared infrastructure and basic load test
//   - oid_test.go: OID resolution (name -> dotted OID)
//   - types_test.go: Type resolution, base types, textual conventions
//   - access_test.go: Access levels
//   - tables_test.go: Table/row/column structure, INDEX, AUGMENTS
//   - notifications_test.go: NOTIFICATION-TYPE, TRAP-TYPE
//   - enums_test.go: Enumerated INTEGERs, BITS definitions
//   - constraints_test.go: SIZE and value range constraints
package integration

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/stretchr/testify/require"
)

// corpusModel holds the shared resolved model for all tests.
// Loaded once via loadCorpus().
var (
	corpusModel *gomib.Mib
	corpusOnce  sync.Once
	corpusErr   error
)

// corpusPath returns the path to the test corpus.
func corpusPath() string {
	return filepath.Join("..", "testdata", "corpus", "primary")
}

// loadCorpus loads the entire test corpus once and caches the result.
// All tests share the same resolved model for efficiency.
func loadCorpus(t *testing.T) *gomib.Mib {
	t.Helper()

	corpusOnce.Do(func() {
		path := corpusPath()
		if _, err := os.Stat(path); os.IsNotExist(err) {
			corpusErr = err
			return
		}

		corpusModel, corpusErr = gomib.Load(gomib.DirTree(path))
	})

	if corpusErr != nil {
		t.Fatalf("failed to load corpus: %v", corpusErr)
	}
	if corpusModel == nil {
		t.Fatal("corpus model is nil")
	}

	return corpusModel
}

// getNode is a helper that retrieves a node by qualified name and fails if not found.
func getNode(t *testing.T, m *gomib.Mib, module, name string) *gomib.Node {
	t.Helper()
	qname := module + "::" + name
	obj := m.ObjectByQualified(qname)
	if obj != nil && obj.Node != nil {
		return obj.Node
	}
	// Try notification
	notif := m.Notification(name)
	if notif != nil && notif.Node != nil && notif.Module != nil && notif.Module.Name == module {
		return notif.Node
	}
	// Try by name and filter by module
	nodes := getNodesByName(m, name)
	for _, node := range nodes {
		if node.Module != nil && node.Module.Name == module {
			return node
		}
	}
	require.Fail(t, "node %s::%s should exist", module, name)
	return nil
}

// getNodesByName returns all nodes with the given name.
func getNodesByName(m *gomib.Mib, name string) []*gomib.Node {
	var nodes []*gomib.Node
	m.Walk(func(n *gomib.Node) bool {
		if n.Name == name {
			nodes = append(nodes, n)
		}
		return true
	})
	return nodes
}

// getObject is a helper that retrieves an object by qualified name and fails if not found.
func getObject(t *testing.T, m *gomib.Mib, module, name string) *gomib.Object {
	t.Helper()
	qname := module + "::" + name
	obj := m.ObjectByQualified(qname)
	require.NotNil(t, obj, "object %s::%s should exist", module, name)
	return obj
}

// TestCorpusLoads verifies the corpus loads without fatal errors.
func TestCorpusLoads(t *testing.T) {
	m := loadCorpus(t)

	// Basic sanity checks
	require.Greater(t, m.ModuleCount(), 0, "should have loaded modules")
	require.Greater(t, m.NodeCount(), 0, "should have OID nodes")
	require.Greater(t, m.ObjectCount(), 0, "should have objects")
	require.Greater(t, m.TypeCount(), 0, "should have types")

	t.Logf("Corpus: %d modules, %d nodes, %d objects, %d types",
		m.ModuleCount(), m.NodeCount(), m.ObjectCount(), m.TypeCount())
}
