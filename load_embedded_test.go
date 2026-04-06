package gomib_test

import (
	"context"
	"testing"

	"github.com/golangsnmp/gomib"
)

func TestLoadWithEmbeddedBaseDeps(t *testing.T) {
	src, err := gomib.Dir("testdata/corpus/primary")
	if err != nil {
		t.Fatal(err)
	}
	m, err := gomib.Load(context.Background(),
		gomib.WithSource(src),
		gomib.WithModules("SNMPv2-MIB"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if m.Module("SNMPv2-SMI") == nil {
		t.Error("SNMPv2-SMI not loaded")
	}
	if m.Module("SNMPv2-TC") == nil {
		t.Error("SNMPv2-TC not loaded")
	}
}

func TestLoadEmbeddedSourcePath(t *testing.T) {
	m, err := gomib.Load(context.Background(),
		gomib.WithModules("SNMPv2-SMI"),
	)
	if err != nil {
		t.Fatal(err)
	}
	mod := m.Module("SNMPv2-SMI")
	if mod == nil {
		t.Fatal("SNMPv2-SMI not loaded from embedded source")
	}
	if mod.SourcePath() != "embedded:SNMPv2-SMI" {
		t.Errorf("SourcePath = %q, want %q", mod.SourcePath(), "embedded:SNMPv2-SMI")
	}
}

func TestLoadLocalOverridesEmbedded(t *testing.T) {
	src, err := gomib.Dir("testdata/corpus/primary")
	if err != nil {
		t.Fatal(err)
	}
	m, err := gomib.Load(context.Background(),
		gomib.WithSource(src),
		gomib.WithModules("SNMPv2-SMI"),
	)
	if err != nil {
		t.Fatal(err)
	}
	mod := m.Module("SNMPv2-SMI")
	if mod == nil {
		t.Fatal("SNMPv2-SMI not loaded")
	}
	if mod.SourcePath() == "embedded:SNMPv2-SMI" {
		t.Error("local copy should take precedence over embedded source")
	}
}
