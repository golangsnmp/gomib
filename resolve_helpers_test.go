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
	loadedMib      mib.Mib
	loadErr        error
)

// loadTestMIB loads all fixture modules once (via sync.Once) and returns
// the shared Mib, so that tests sharing the same fixture set avoid
// redundant parsing.
func loadTestMIB(t testing.TB) mib.Mib {
	t.Helper()
	loadOnce.Do(func() {
		src, err := DirTree("testdata/corpus/primary")
		if err != nil {
			loadErr = err
			return
		}
		loadedMib, loadErr = LoadModules(context.Background(), fixtureModules, src)
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
