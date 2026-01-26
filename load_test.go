package gomib_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/golangsnmp/gomib"
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
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	mib, err := gomib.Load(src)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Basic sanity checks
	modules := mib.Modules()
	if len(modules) == 0 {
		t.Error("Expected at least some modules")
	}
	t.Logf("Loaded %d modules", len(modules))

	objects := mib.Objects()
	t.Logf("Found %d objects", len(objects))

	types := mib.Types()
	t.Logf("Found %d types", len(types))

	// Test lookup
	obj := mib.Object("sysDescr")
	if obj != nil {
		t.Logf("sysDescr found: OID=%s, Module=%s", obj.OID(), obj.Module.Name)
	}

	// Test OID lookup
	node := mib.Node("1.3.6.1.2.1.1.1")
	if node != nil {
		t.Logf("Node 1.3.6.1.2.1.1.1: Name=%s, Kind=%v", node.Name, node.Kind)
	}
}

func TestLoadModulesIntegration(t *testing.T) {
	mibPath := findMibDir(t)

	src, err := gomib.DirTree(mibPath)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	mib, err := gomib.LoadModules([]string{"IF-MIB"}, src)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Check we got IF-MIB
	mod := mib.Module("IF-MIB")
	if mod == nil {
		t.Log("IF-MIB not found (might be missing from source)")
		return
	}
	t.Logf("Loaded IF-MIB: %s", mod.Name)

	// Check for ifOperStatus
	obj := mib.Object("ifOperStatus")
	if obj != nil {
		t.Logf("ifOperStatus found: OID=%s", obj.OID())
	}
}

func TestLoadNoSource(t *testing.T) {
	// Loading with nil source should return ErrNoSources
	_, err := gomib.Load(nil)
	if err != gomib.ErrNoSources {
		t.Errorf("Load(nil) = %v, want ErrNoSources", err)
	}
}

func TestFindNode(t *testing.T) {
	mibPath := findMibDir(t)

	src, err := gomib.DirTree(mibPath)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	mib, err := gomib.LoadModules([]string{"SNMPv2-MIB"}, src)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

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
				if node != nil {
					t.Errorf("FindNode(%q) = %q, want nil", tt.query, node.Name)
				}
			} else {
				if node == nil {
					t.Errorf("FindNode(%q) = nil, want %q", tt.query, tt.wantName)
				} else if node.Name != tt.wantName {
					t.Errorf("FindNode(%q) = %q, want %q", tt.query, node.Name, tt.wantName)
				}
			}
		})
	}
}
