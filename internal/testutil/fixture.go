package testutil

import (
	"encoding/json"
	"os"
	"testing"
)

// FixtureNode mirrors the NormalizedNode JSON schema from gomib-netsnmp fixtures.
type FixtureNode struct {
	OID          string         `json:"OID"`
	Name         string         `json:"Name"`
	Module       string         `json:"Module"`
	Type         string         `json:"Type"`
	Access       string         `json:"Access"`
	Status       string         `json:"Status"`
	Hint         string         `json:"Hint"`
	TCName       string         `json:"TCName"`
	Units        string         `json:"Units"`
	EnumValues   map[int]string `json:"EnumValues"`
	Indexes      []IndexInfo    `json:"Indexes"`
	Augments     string         `json:"Augments"`
	Ranges       []RangeInfo    `json:"Ranges"`
	DefaultValue string         `json:"DefaultValue"`
	Kind         string         `json:"Kind"`
	Varbinds     []string       `json:"Varbinds"`
	NodeType     string         `json:"NodeType"`
	BitValues    map[int]string `json:"BitValues"`
	Reference    string         `json:"Reference"`
}

// RangeInfo describes a range constraint from the fixture.
type RangeInfo struct {
	Low  int64 `json:"Low"`
	High int64 `json:"High"`
}

// IndexInfo describes an index component from the fixture.
type IndexInfo struct {
	Name    string `json:"Name"`
	Implied bool   `json:"Implied"`
}

// LoadFixture loads a fixture JSON file and returns nodes keyed by OID string.
func LoadFixture(t testing.TB, path string) map[string]*FixtureNode {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", path, err)
	}
	var nodes map[string]*FixtureNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		t.Fatalf("failed to parse fixture %s: %v", path, err)
	}
	return nodes
}

// LoadFixtureByName loads a fixture JSON file and returns nodes keyed by Name.
func LoadFixtureByName(t testing.TB, path string) map[string]*FixtureNode {
	t.Helper()
	byOID := LoadFixture(t, path)
	byName := make(map[string]*FixtureNode, len(byOID))
	for _, node := range byOID {
		byName[node.Name] = node
	}
	return byName
}

// FixtureObjectNodes returns only fixture nodes that are OBJECT-TYPE
// (have a non-empty Type that is not "OTHER").
func FixtureObjectNodes(nodes map[string]*FixtureNode) map[string]*FixtureNode {
	filtered := make(map[string]*FixtureNode)
	for oid, node := range nodes {
		if node.Type != "" && node.Type != "OTHER" {
			filtered[oid] = node
		}
	}
	return filtered
}

// FixtureNotificationNodes returns only fixture nodes that are NOTIFICATION-TYPE.
func FixtureNotificationNodes(nodes map[string]*FixtureNode) map[string]*FixtureNode {
	filtered := make(map[string]*FixtureNode)
	for oid, node := range nodes {
		if node.NodeType == "NOTIFICATION-TYPE" || node.NodeType == "TRAP-TYPE" {
			filtered[oid] = node
		}
	}
	return filtered
}
