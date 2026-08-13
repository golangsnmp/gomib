package gomib_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/mib"
)

// TestDuplicateOIDNamesReturnDefinitionMetadata exercises duplicate-OID
// lookups through the public API.
func TestDuplicateOIDNamesReturnDefinitionMetadata(t *testing.T) {
	src := gomib.FS("duplicate-oids", fstest.MapFS{
		"preferred.mib": &fstest.MapFile{Data: []byte(`
PREFERRED-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises FROM SNMPv2-SMI;

preferredMIB MODULE-IDENTITY
    LAST-UPDATED "202501010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Preferred module"
    REVISION "202501010000Z"
    DESCRIPTION "Preferred revision"
    ::= { enterprises 99999 }

preferredObject OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "preferred metadata"
    ::= { enterprises 99999 1 }
END
`)},
		"legacy.mib": &fstest.MapFile{Data: []byte(`
LEGACY-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises FROM SNMPv2-SMI;

legacyMIB MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Legacy module"
    REVISION "200001010000Z"
    DESCRIPTION "Legacy revision"
    ::= { enterprises 99998 }

legacyObject OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-only
    STATUS deprecated
    DESCRIPTION "legacy metadata"
    ::= { enterprises 99999 1 }
END
`)},
	})

	diagConfig := mib.DefaultConfig()
	diagConfig.FailAt = mib.SeverityFatal
	m, err := gomib.Load(context.Background(),
		gomib.WithSource(src),
		gomib.WithModules("PREFERRED-MIB", "LEGACY-MIB"),
		gomib.WithDiagnosticConfig(diagConfig),
	)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	preferred := m.Object("preferredObject")
	legacy := m.Object("legacyObject")
	if preferred == nil || legacy == nil {
		t.Fatalf("global objects: preferred=%v legacy=%v", preferred, legacy)
	}
	if preferred.Description() != "preferred metadata" {
		t.Errorf("preferred description = %q", preferred.Description())
	}
	if legacy.Description() != "legacy metadata" {
		t.Errorf("legacy description = %q", legacy.Description())
	}
	if preferred.Node() != legacy.Node() {
		t.Error("duplicate-OID objects should share a node")
	}

	preferredMod := m.Module("PREFERRED-MIB")
	legacyMod := m.Module("LEGACY-MIB")
	if preferredMod == nil || legacyMod == nil {
		t.Fatal("defining modules not found")
	}
	if preferredMod.Object("preferredObject") != preferred {
		t.Error("preferred module lookup did not return its definition")
	}
	if legacyMod.Object("legacyObject") != legacy {
		t.Error("legacy module lookup did not return its definition")
	}
	if m.Symbol("preferredObject").Object() != preferred || m.Symbol("legacyObject").Object() != legacy {
		t.Error("global symbol lookup did not return exact-name object metadata")
	}
	if m.Resolve("PREFERRED-MIB::preferredObject") != preferred.Node() ||
		m.Resolve("LEGACY-MIB::legacyObject") != legacy.Node() {
		t.Error("qualified node lookup did not return the shared node")
	}
	if legacyMod.Node("preferredObject") != nil || !legacyMod.Symbol("preferredObject").IsZero() ||
		m.Resolve("LEGACY-MIB::preferredObject") != nil {
		t.Error("legacy module exposed the preferred module's definition name")
	}
	if preferredMod.Node("legacyObject") != nil || !preferredMod.Symbol("legacyObject").IsZero() ||
		m.Resolve("PREFERRED-MIB::legacyObject") != nil {
		t.Error("preferred module exposed the legacy module's definition name")
	}

	sharedNode := preferred.Node()
	if sharedNode.Object() != preferred {
		t.Error("tree node did not retain the preferred object")
	}
	if sharedNode.Name() != "preferredObject" || sharedNode.Module() != preferredMod {
		t.Errorf("preferred tree metadata = %s::%s", sharedNode.Module().Name(), sharedNode.Name())
	}
}

// TestLoadAndQueryMib is an end-to-end test exercising the public API from
// an external package perspective. It verifies that Load, Module, Object,
// Node, and Type lookups work through the exported surface.
func TestLoadAndQueryMib(t *testing.T) {
	src, err := gomib.Dir("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("Dir failed: %v", err)
	}

	ctx := context.Background()
	m, err := gomib.Load(ctx,
		gomib.WithSource(src),
		gomib.WithModules("IF-MIB", "SNMPv2-MIB"),
	)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify modules were loaded
	ifMIB := m.Module("IF-MIB")
	if ifMIB == nil {
		t.Fatal("IF-MIB module should be found")
	}
	if ifMIB.Name() != "IF-MIB" {
		t.Errorf("Module name = %q, want %q", ifMIB.Name(), "IF-MIB")
	}

	snmpMIB := m.Module("SNMPv2-MIB")
	if snmpMIB == nil {
		t.Fatal("SNMPv2-MIB module should be found")
	}

	// Query objects
	ifIndex := m.Object("ifIndex")
	if ifIndex == nil {
		t.Fatal("ifIndex should be found")
	}
	if ifIndex.Name() != "ifIndex" {
		t.Errorf("Object name = %q, want %q", ifIndex.Name(), "ifIndex")
	}
	if ifIndex.OID() == nil {
		t.Fatal("ifIndex should have an OID")
	}

	sysDescr := m.Object("sysDescr")
	if sysDescr == nil {
		t.Fatal("sysDescr should be found")
	}
	if sysDescr.Type() == nil {
		t.Fatal("sysDescr should have a resolved type")
	}

	// Query node by name
	nd := m.Node("ifIndex")
	if nd == nil {
		t.Fatal("Node(ifIndex) should return a node")
	}
	if nd.Object() == nil {
		t.Fatal("ifIndex node should have an Object")
	}

	// Query type
	typ := m.Type("DisplayString")
	if typ == nil {
		t.Fatal("DisplayString type should be found")
	}
	if !typ.IsTextualConvention() {
		t.Error("DisplayString should be a textual convention")
	}

	// Collections
	if len(m.Objects()) == 0 {
		t.Error("should have objects")
	}
	if len(m.Types()) == 0 {
		t.Error("should have types")
	}
	if len(m.Modules()) == 0 {
		t.Error("should have modules")
	}
	if len(m.Tables()) == 0 {
		t.Error("should have tables")
	}

	// Node iteration
	count := 0
	for range m.Nodes() {
		count++
	}
	if count != m.NodeCount() {
		t.Errorf("Nodes() count = %d, NodeCount() = %d, should match", count, m.NodeCount())
	}

	// NodeByOID lookup
	ndByOID := m.NodeByOID(mib.OID{1, 3, 6, 1, 2, 1, 2, 2, 1, 1})
	if ndByOID == nil {
		t.Fatal("NodeByOID should find ifIndex")
	}
	if ndByOID.Name() != "ifIndex" {
		t.Errorf("NodeByOID name = %q, want %q", ndByOID.Name(), "ifIndex")
	}
	if m.NodeByOID(mib.OID{99, 99, 99}) != nil {
		t.Error("NodeByOID should return nil for non-existent OID")
	}

	// FormatOID
	formatted := m.FormatOID(mib.OID{1, 3, 6, 1, 2, 1, 2, 2, 1, 1, 5})
	if formatted == "" {
		t.Fatal("FormatOID should return non-empty string")
	}
	if !strings.Contains(formatted, "ifIndex") {
		t.Errorf("FormatOID result should contain 'ifIndex', got %q", formatted)
	}
}

func TestWithSystemPaths(t *testing.T) {
	if os.Getenv("GOMIB_INTEGRATION") == "" {
		t.Skip("skipping host-dependent test; set GOMIB_INTEGRATION=1 to run")
	}

	ctx := context.Background()

	m, err := gomib.Load(ctx, gomib.WithSystemPaths())
	if err != nil {
		if errors.Is(err, gomib.ErrNoSources) {
			t.Skip("no system MIB paths found on this host")
		}
		if !errors.Is(err, gomib.ErrDiagnosticThreshold) {
			t.Fatalf("unexpected error: %v", err)
		}
		// ErrDiagnosticThreshold is acceptable (vendor MIBs may have issues),
		// but the Mib should still be returned with resolved data.
	}

	if m == nil {
		t.Fatal("Mib should not be nil")
	}
	if m.NodeCount() == 0 {
		t.Error("system paths should yield at least some resolved nodes")
	}
	if len(m.Modules()) == 0 {
		t.Error("system paths should yield at least one loaded module")
	}
}

func TestWithSystemPathsCombinedWithExplicitSource(t *testing.T) {
	src, err := gomib.Dir("testdata/corpus/primary")
	if err != nil {
		t.Fatalf("Dir failed: %v", err)
	}

	ctx := context.Background()
	m, err := gomib.Load(ctx,
		gomib.WithSource(src),
		gomib.WithSystemPaths(),
		gomib.WithModules("IF-MIB"),
	)
	if err != nil {
		t.Fatalf("Load with explicit source + system paths failed: %v", err)
	}
	if m.Module("IF-MIB") == nil {
		t.Error("IF-MIB should be found when combining explicit source with system paths")
	}
}
