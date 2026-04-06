package gomib

import (
	"errors"
	"io/fs"
	"slices"
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
)

func TestEmbeddedSource_Find(t *testing.T) {
	src := embeddedSource{}

	result, err := src.Find("SNMPv2-SMI")
	if err != nil {
		t.Fatalf("Find(SNMPv2-SMI) error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Error("Find(SNMPv2-SMI) returned empty content")
	}
	if result.Path != "embedded:SNMPv2-SMI" {
		t.Errorf("Path = %q, want %q", result.Path, "embedded:SNMPv2-SMI")
	}
}

func TestEmbeddedSource_Find_NotFound(t *testing.T) {
	src := embeddedSource{}

	_, err := src.Find("NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for nonexistent module")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error = %v, want fs.ErrNotExist", err)
	}
}

func TestEmbeddedSource_ListModules(t *testing.T) {
	src := embeddedSource{}

	names, err := src.ListModules()
	if err != nil {
		t.Fatalf("ListModules() error: %v", err)
	}
	if len(names) < 7 {
		t.Errorf("ListModules() returned %d names, want at least 7", len(names))
	}
	for _, base := range module.BaseModuleNames() {
		if !slices.Contains(names, base) {
			t.Errorf("ListModules() missing %q", base)
		}
	}
}
