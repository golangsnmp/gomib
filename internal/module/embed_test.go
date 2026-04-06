package module

import (
	"testing"
)

func TestEmbeddedContent(t *testing.T) {
	for _, name := range BaseModuleNames() {
		content := EmbeddedContent(name)
		if content == nil {
			t.Errorf("EmbeddedContent(%q) returned nil", name)
			continue
		}
		if len(content) == 0 {
			t.Errorf("EmbeddedContent(%q) returned empty content", name)
		}
	}
}

func TestEmbeddedContent_NotFound(t *testing.T) {
	if content := EmbeddedContent("NONEXISTENT"); content != nil {
		t.Error("EmbeddedContent for unknown module should return nil")
	}
}

func TestEmbeddedModuleNames(t *testing.T) {
	names := EmbeddedModuleNames()
	if len(names) < 7 {
		t.Errorf("EmbeddedModuleNames() returned %d names, want at least 7", len(names))
	}
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	for _, base := range BaseModuleNames() {
		if !nameSet[base] {
			t.Errorf("EmbeddedModuleNames() missing %q", base)
		}
	}
}
