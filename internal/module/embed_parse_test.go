package module_test

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/cst/lower"
	cstparser "github.com/golangsnmp/gomib/internal/cst/parser"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestEmbeddedBaseModulesParse(t *testing.T) {
	for _, name := range module.BaseModuleNames() {
		t.Run(name, func(t *testing.T) {
			content := module.EmbeddedContent(name)
			if content == nil {
				t.Fatalf("no embedded content for %s", name)
			}

			p := cstparser.New(content, nil, types.DefaultConfig())
			file := p.ParseModule()
			mods := lower.Lower(file, content, p.Diagnostics(), types.DefaultConfig())

			var target *module.Module
			for _, mod := range mods {
				if mod.Name == name {
					target = mod
					break
				}
			}
			if target == nil {
				t.Fatalf("module %s not found in parse result (got %d modules)", name, len(mods))
			}

			for _, d := range target.Diagnostics {
				if d.Severity == types.SeverityError {
					t.Errorf("parse error: %s", d.Message)
				}
			}
		})
	}
}

func TestEmbeddedSNMPv2SMI_Content(t *testing.T) {
	content := module.EmbeddedContent("SNMPv2-SMI")
	p := cstparser.New(content, nil, types.DefaultConfig())
	file := p.ParseModule()
	mods := lower.Lower(file, content, p.Diagnostics(), types.DefaultConfig())

	var target *module.Module
	for _, mod := range mods {
		if mod.Name == "SNMPv2-SMI" {
			target = mod
			break
		}
	}
	if target == nil {
		t.Fatal("SNMPv2-SMI not found")
	}

	if len(target.TypeDefs) == 0 {
		t.Error("SNMPv2-SMI should have type definitions")
	}

	if len(target.ValueAssignments) == 0 {
		t.Error("SNMPv2-SMI should have value assignments")
	}

	// Should have OBJECT-IDENTITY definitions (internet, enterprises, etc.)
	if len(target.ObjectIdentities) == 0 {
		t.Error("SNMPv2-SMI should have OBJECT-IDENTITY definitions")
	}
}

func TestEmbeddedBaseModules_NoMacroWarnings(t *testing.T) {
	for _, name := range module.BaseModuleNames() {
		t.Run(name, func(t *testing.T) {
			content := module.EmbeddedContent(name)
			if content == nil {
				t.Skipf("no embedded content for %s", name)
			}

			p := cstparser.New(content, nil, types.DefaultConfig())
			file := p.ParseModule()
			mods := lower.Lower(file, content, p.Diagnostics(), types.DefaultConfig())

			for _, mod := range mods {
				if mod.Name != name {
					continue
				}
				for _, d := range mod.Diagnostics {
					if d.Code == types.DiagMacroNotAllowed {
						t.Errorf("unexpected DiagMacroNotAllowed in base module %s: %s", name, d.Message)
					}
				}
			}
		})
	}
}

func TestEmbeddedSNMPv2TC_Content(t *testing.T) {
	content := module.EmbeddedContent("SNMPv2-TC")
	p := cstparser.New(content, nil, types.DefaultConfig())
	file := p.ParseModule()
	mods := lower.Lower(file, content, p.Diagnostics(), types.DefaultConfig())

	var target *module.Module
	for _, mod := range mods {
		if mod.Name == "SNMPv2-TC" {
			target = mod
			break
		}
	}
	if target == nil {
		t.Fatal("SNMPv2-TC not found")
	}

	hasTCs := false
	for _, td := range target.TypeDefs {
		if td.IsTextualConvention {
			hasTCs = true
			break
		}
	}
	if !hasTCs {
		t.Error("SNMPv2-TC should have textual convention definitions")
	}
}
