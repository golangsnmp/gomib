package gomib_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/internal/testutil"
)

// findMibDir tries to locate a directory with MIB files for testing
func findMibDir(t *testing.T) string {
	// Check common locations
	candidates := []string{
		"/usr/share/snmp/mibs",
		"/usr/local/share/snmp/mibs",
	}

	// Also check relative to the test directory
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "testdata", "corpus", "primary"),
			filepath.Join(cwd, "testdata", "mibs"),
			filepath.Join(cwd, "..", "testdata", "mibs"),
		)
	}

	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			// Check if it has any files
			entries, err := os.ReadDir(path)
			if err == nil && len(entries) > 0 {
				return path
			}
		}
	}

	t.Skip("No MIB directory found for testing")
	return ""
}

func TestLoadIntegration(t *testing.T) {
	mibPath := findMibDir(t)

	src, err := gomib.DirTree(mibPath)
	testutil.NoError(t, err, "create source")

	mib, err := gomib.Load(context.Background(), src)
	testutil.NoError(t, err, "load")

	// Basic sanity checks
	modules := mib.Modules()
	testutil.NotEmpty(t, modules, "expected at least some modules")
	t.Logf("Loaded %d modules", len(modules))

	objects := mib.Objects()
	t.Logf("Found %d objects", len(objects))

	types := mib.Types()
	t.Logf("Found %d types", len(types))

	// Test lookup
	obj := mib.FindObject("sysDescr")
	if obj != nil {
		t.Logf("sysDescr found: OID=%s, Module=%s", obj.OID(), obj.Module().Name())
	}

	// Test OID lookup
	node := mib.FindNode("1.3.6.1.2.1.1.1")
	if node != nil {
		t.Logf("Node 1.3.6.1.2.1.1.1: Name=%s, Kind=%v", node.Name(), node.Kind())
	}
}

func TestLoadModulesIntegration(t *testing.T) {
	mibPath := findMibDir(t)

	src, err := gomib.DirTree(mibPath)
	testutil.NoError(t, err, "create source")

	mib, err := gomib.LoadModules(context.Background(), []string{"IF-MIB"}, src)
	testutil.NoError(t, err, "load IF-MIB")

	// Check we got IF-MIB
	mod := mib.Module("IF-MIB")
	if mod == nil {
		t.Log("IF-MIB not found (might be missing from source)")
		return
	}
	t.Logf("Loaded IF-MIB: %s", mod.Name())

	// Check for ifOperStatus
	obj := mib.FindObject("ifOperStatus")
	if obj != nil {
		t.Logf("ifOperStatus found: OID=%s", obj.OID())
	}
}

func TestLoadNoSource(t *testing.T) {
	// Loading with nil source should return ErrNoSources
	_, err := gomib.Load(context.Background(), nil)
	testutil.Equal(t, gomib.ErrNoSources, err, "Load(nil)")
}

func TestFindNode(t *testing.T) {
	mibPath := findMibDir(t)

	src, err := gomib.DirTree(mibPath)
	testutil.NoError(t, err, "create source")

	mib, err := gomib.LoadModules(context.Background(), []string{"SNMPv2-MIB"}, src)
	testutil.NoError(t, err, "load SNMPv2-MIB")

	// Check that SNMPv2-MIB loaded
	if mib.Module("SNMPv2-MIB") == nil {
		t.Skip("SNMPv2-MIB not found")
	}

	tests := []struct {
		name     string
		query    string
		wantName string
	}{
		{"qualified name", "SNMPv2-MIB::sysDescr", "sysDescr"},
		{"simple name", "sysDescr", "sysDescr"},
		{"numeric OID", "1.3.6.1.2.1.1.1", "sysDescr"},
		{"partial OID", ".1.3.6.1.2.1.1.1", "sysDescr"},
		{"nonexistent", "nonexistent-object-xyz", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := mib.FindNode(tt.query)
			if tt.wantName == "" {
				testutil.Nil(t, node, "FindNode(%q)", tt.query)
			} else {
				testutil.NotNil(t, node, "FindNode(%q)", tt.query)
				testutil.Equal(t, tt.wantName, node.Name(), "FindNode(%q) name", tt.query)
			}
		})
	}
}

func TestNodesIterator(t *testing.T) {
	mibPath := findMibDir(t)

	src, err := gomib.DirTree(mibPath)
	testutil.NoError(t, err, "create source")

	mib, err := gomib.LoadModules(context.Background(), []string{"SNMPv2-MIB"}, src)
	testutil.NoError(t, err, "load SNMPv2-MIB")

	if mib.Module("SNMPv2-MIB") == nil {
		t.Skip("SNMPv2-MIB not found")
	}

	// Test Mib.Nodes() iterator
	nodeCount := 0
	for range mib.Nodes() {
		nodeCount++
	}

	t.Logf("Iterator visited %d nodes", nodeCount)
	testutil.Greater(t, nodeCount, 0, "Nodes() should return nodes")

	// Test early termination
	earlyCount := 0
	for range mib.Nodes() {
		earlyCount++
		if earlyCount >= 5 {
			break
		}
	}
	testutil.Equal(t, 5, earlyCount, "early termination count")

	// Test Node.Descendants()
	sysNode := mib.FindNode("system")
	if sysNode != nil {
		var descendants []gomib.Node
		for n := range sysNode.Descendants() {
			descendants = append(descendants, n)
		}
		testutil.NotEmpty(t, descendants, "system node Descendants()")
		t.Logf("system node has %d descendants", len(descendants))
	}
}
