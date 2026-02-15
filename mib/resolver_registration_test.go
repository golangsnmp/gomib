package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestConvertRevisions(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		got := convertRevisions(nil)
		if len(got) != 0 {
			t.Errorf("convertRevisions(nil) returned %d items, want 0", len(got))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got := convertRevisions([]module.Revision{})
		if len(got) != 0 {
			t.Errorf("convertRevisions([]) returned %d items, want 0", len(got))
		}
	})

	t.Run("single revision", func(t *testing.T) {
		input := []module.Revision{
			{Date: "2024-01-15", Description: "Initial version"},
		}
		got := convertRevisions(input)
		if len(got) != 1 {
			t.Fatalf("got %d revisions, want 1", len(got))
		}
		if got[0].Date != "2024-01-15" {
			t.Errorf("Date = %q, want %q", got[0].Date, "2024-01-15")
		}
		if got[0].Description != "Initial version" {
			t.Errorf("Description = %q, want %q", got[0].Description, "Initial version")
		}
	})

	t.Run("multiple revisions", func(t *testing.T) {
		input := []module.Revision{
			{Date: "2024-06-01", Description: "Added new objects"},
			{Date: "2024-01-15", Description: "Initial version"},
			{Date: "2023-12-01", Description: "Draft"},
		}
		got := convertRevisions(input)
		if len(got) != 3 {
			t.Fatalf("got %d revisions, want 3", len(got))
		}
		for i, r := range input {
			if got[i].Date != r.Date {
				t.Errorf("revision[%d].Date = %q, want %q", i, got[i].Date, r.Date)
			}
			if got[i].Description != r.Description {
				t.Errorf("revision[%d].Description = %q, want %q", i, got[i].Description, r.Description)
			}
		}
	})
}

func TestRegisterModules_BaseModulesPrepended(t *testing.T) {
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv2,
		Definitions: []module.Definition{
			&module.ObjectType{Name: "myObject", Span: types.Synthetic},
		},
	}
	ctx := newResolverContext([]*module.Module{userMod}, nil, DefaultConfig())

	registerModules(ctx)

	// Base modules should come first, user module last
	baseNames := module.BaseModuleNames()
	if len(ctx.Modules) != len(baseNames)+1 {
		t.Fatalf("got %d modules, want %d (base=%d + user=1)",
			len(ctx.Modules), len(baseNames)+1, len(baseNames))
	}
	for i, name := range baseNames {
		if ctx.Modules[i].Name != name {
			t.Errorf("Modules[%d].Name = %q, want %q", i, ctx.Modules[i].Name, name)
		}
	}
	last := ctx.Modules[len(ctx.Modules)-1]
	if last.Name != "MY-MIB" {
		t.Errorf("last module = %q, want %q", last.Name, "MY-MIB")
	}
}

func TestRegisterModules_UserModulesWithBaseNamesFiltered(t *testing.T) {
	// If a user provides a module with a base module name, it should be dropped.
	userSNMP := &module.Module{
		Name:     "SNMPv2-SMI",
		Language: types.LanguageSMIv2,
	}
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv2,
	}
	ctx := newResolverContext([]*module.Module{userSNMP, userMod}, nil, DefaultConfig())

	registerModules(ctx)

	// The user's SNMPv2-SMI should be replaced by the base version.
	// Count how many SNMPv2-SMI modules exist.
	count := 0
	for _, mod := range ctx.Modules {
		if mod.Name == "SNMPv2-SMI" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("found %d SNMPv2-SMI modules, want exactly 1 (base only)", count)
	}

	// MY-MIB should still be present
	found := false
	for _, mod := range ctx.Modules {
		if mod.Name == "MY-MIB" {
			found = true
			break
		}
	}
	if !found {
		t.Error("MY-MIB not found in ctx.Modules")
	}
}

func TestRegisterModules_ModuleIndexPopulated(t *testing.T) {
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv2,
	}
	ctx := newResolverContext([]*module.Module{userMod}, nil, DefaultConfig())

	registerModules(ctx)

	// Every module should be indexed by name
	for _, mod := range ctx.Modules {
		entries, ok := ctx.ModuleIndex[mod.Name]
		if !ok {
			t.Errorf("ModuleIndex missing entry for %q", mod.Name)
			continue
		}
		found := false
		for _, entry := range entries {
			if entry == mod {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ModuleIndex[%q] does not contain the module pointer", mod.Name)
		}
	}
}

func TestRegisterModules_BaseModulePointersCached(t *testing.T) {
	ctx := newResolverContext(nil, nil, DefaultConfig())

	registerModules(ctx)

	if ctx.Snmpv2SMIModule == nil {
		t.Error("Snmpv2SMIModule is nil")
	} else if ctx.Snmpv2SMIModule.Name != "SNMPv2-SMI" {
		t.Errorf("Snmpv2SMIModule.Name = %q, want %q", ctx.Snmpv2SMIModule.Name, "SNMPv2-SMI")
	}

	if ctx.Rfc1155SMIModule == nil {
		t.Error("Rfc1155SMIModule is nil")
	} else if ctx.Rfc1155SMIModule.Name != "RFC1155-SMI" {
		t.Errorf("Rfc1155SMIModule.Name = %q, want %q", ctx.Rfc1155SMIModule.Name, "RFC1155-SMI")
	}

	if ctx.Snmpv2TCModule == nil {
		t.Error("Snmpv2TCModule is nil")
	} else if ctx.Snmpv2TCModule.Name != "SNMPv2-TC" {
		t.Errorf("Snmpv2TCModule.Name = %q, want %q", ctx.Snmpv2TCModule.Name, "SNMPv2-TC")
	}
}

func TestRegisterModules_DefinitionNamesCached(t *testing.T) {
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv2,
		Definitions: []module.Definition{
			&module.ObjectType{Name: "fooObject", Span: types.Synthetic},
			&module.TypeDef{Name: "BarType", Span: types.Synthetic},
		},
	}
	ctx := newResolverContext([]*module.Module{userMod}, nil, DefaultConfig())

	registerModules(ctx)

	// Find the user module pointer (the original may have been replaced)
	var found *module.Module
	for _, mod := range ctx.Modules {
		if mod.Name == "MY-MIB" {
			found = mod
			break
		}
	}
	if found == nil {
		t.Fatal("MY-MIB not found in ctx.Modules")
	}

	defNames := ctx.ModuleDefNames[found]
	if defNames == nil {
		t.Fatal("ModuleDefNames[MY-MIB] is nil")
	}
	if _, ok := defNames["fooObject"]; !ok {
		t.Error("fooObject not in ModuleDefNames")
	}
	if _, ok := defNames["BarType"]; !ok {
		t.Error("BarType not in ModuleDefNames")
	}
	if _, ok := defNames["nonExistent"]; ok {
		t.Error("nonExistent should not be in ModuleDefNames")
	}
}

func TestRegisterModules_ModuleToResolvedMapping(t *testing.T) {
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv2,
	}
	ctx := newResolverContext([]*module.Module{userMod}, nil, DefaultConfig())

	registerModules(ctx)

	for _, mod := range ctx.Modules {
		resolved, ok := ctx.ModuleToResolved[mod]
		if !ok {
			t.Errorf("ModuleToResolved missing entry for %q", mod.Name)
			continue
		}
		if resolved.Name() != mod.Name {
			t.Errorf("resolved name %q != module name %q", resolved.Name(), mod.Name)
		}
		// Check reverse mapping
		reverse, ok := ctx.ResolvedToModule[resolved]
		if !ok {
			t.Errorf("ResolvedToModule missing entry for %q", mod.Name)
			continue
		}
		if reverse != mod {
			t.Errorf("reverse mapping for %q points to wrong module", mod.Name)
		}
	}
}

func TestRegisterModules_LanguageSetOnResolved(t *testing.T) {
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv1,
	}
	ctx := newResolverContext([]*module.Module{userMod}, nil, DefaultConfig())

	registerModules(ctx)

	var found *module.Module
	for _, mod := range ctx.Modules {
		if mod.Name == "MY-MIB" {
			found = mod
			break
		}
	}
	if found == nil {
		t.Fatal("MY-MIB not found")
	}

	resolved := ctx.ModuleToResolved[found]
	if resolved == nil {
		t.Fatal("no resolved module for MY-MIB")
	}
	if resolved.Language() != LanguageSMIv1 {
		t.Errorf("resolved language = %v, want SMIv1", resolved.Language())
	}
}

func TestRegisterModules_ModuleIdentityExtracted(t *testing.T) {
	mi := &module.ModuleIdentity{
		Name:         "myMIB",
		Organization: "ACME Corp",
		ContactInfo:  "support@acme.example",
		Description:  "Test MIB module",
		Revisions: []module.Revision{
			{Date: "2024-06-01", Description: "Rev 2"},
			{Date: "2024-01-15", Description: "Rev 1"},
		},
		Oid:  module.NewOidAssignment(nil, types.Synthetic),
		Span: types.Synthetic,
	}
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv2,
		Definitions: []module.Definition{
			// Some other definition before MODULE-IDENTITY
			&module.ObjectType{Name: "someObj", Span: types.Synthetic},
			mi,
		},
	}
	ctx := newResolverContext([]*module.Module{userMod}, nil, DefaultConfig())

	registerModules(ctx)

	var found *module.Module
	for _, mod := range ctx.Modules {
		if mod.Name == "MY-MIB" {
			found = mod
			break
		}
	}
	if found == nil {
		t.Fatal("MY-MIB not found")
	}

	resolved := ctx.ModuleToResolved[found]
	if resolved == nil {
		t.Fatal("no resolved module for MY-MIB")
	}

	if resolved.Organization() != "ACME Corp" {
		t.Errorf("Organization = %q, want %q", resolved.Organization(), "ACME Corp")
	}
	if resolved.ContactInfo() != "support@acme.example" {
		t.Errorf("ContactInfo = %q, want %q", resolved.ContactInfo(), "support@acme.example")
	}
	if resolved.Description() != "Test MIB module" {
		t.Errorf("Description = %q, want %q", resolved.Description(), "Test MIB module")
	}
	revs := resolved.Revisions()
	if len(revs) != 2 {
		t.Fatalf("got %d revisions, want 2", len(revs))
	}
	if revs[0].Date != "2024-06-01" || revs[0].Description != "Rev 2" {
		t.Errorf("revision[0] = %+v, want Date=2024-06-01 Description=Rev 2", revs[0])
	}
	if revs[1].Date != "2024-01-15" || revs[1].Description != "Rev 1" {
		t.Errorf("revision[1] = %+v, want Date=2024-01-15 Description=Rev 1", revs[1])
	}
}

func TestRegisterModules_NoModuleIdentity(t *testing.T) {
	// A module without MODULE-IDENTITY should have empty metadata on the resolved module.
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv1,
		Definitions: []module.Definition{
			&module.ObjectType{Name: "someObj", Span: types.Synthetic},
		},
	}
	ctx := newResolverContext([]*module.Module{userMod}, nil, DefaultConfig())

	registerModules(ctx)

	var found *module.Module
	for _, mod := range ctx.Modules {
		if mod.Name == "MY-MIB" {
			found = mod
			break
		}
	}
	if found == nil {
		t.Fatal("MY-MIB not found")
	}

	resolved := ctx.ModuleToResolved[found]
	if resolved.Organization() != "" {
		t.Errorf("Organization = %q, want empty", resolved.Organization())
	}
	if resolved.ContactInfo() != "" {
		t.Errorf("ContactInfo = %q, want empty", resolved.ContactInfo())
	}
	if resolved.Description() != "" {
		t.Errorf("Description = %q, want empty", resolved.Description())
	}
	if len(resolved.Revisions()) != 0 {
		t.Errorf("Revisions has %d entries, want 0", len(resolved.Revisions()))
	}
}

func TestRegisterModules_BuilderReceivesModules(t *testing.T) {
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv2,
	}
	ctx := newResolverContext([]*module.Module{userMod}, nil, DefaultConfig())

	registerModules(ctx)

	baseNames := module.BaseModuleNames()
	wantCount := len(baseNames) + 1
	if got := len(ctx.Mib.Modules()); got != wantCount {
		t.Errorf("len(Builder.Modules()) = %d, want %d", got, wantCount)
	}

	// Verify the builder can look up each module by name
	for _, name := range baseNames {
		if ctx.Mib.Module(name) == nil {
			t.Errorf("Builder.Module(%q) returned nil", name)
		}
	}
	if ctx.Mib.Module("MY-MIB") == nil {
		t.Error("Builder.Module(MY-MIB) returned nil")
	}
}

func TestRegisterModules_DiagnosticsForwarded(t *testing.T) {
	// Module-level diagnostics from parsing should be forwarded to the builder.
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv2,
		Diagnostics: []Diagnostic{
			{Severity: SeverityWarning, Code: "test-warning", Message: "test", Module: "MY-MIB"},
		},
	}
	ctx := newResolverContext([]*module.Module{userMod}, nil, DefaultConfig())

	registerModules(ctx)

	// Build and check diagnostics are present
	built := ctx.Mib
	diags := built.Diagnostics()
	found := false
	for _, d := range diags {
		if d.Code == "test-warning" {
			found = true
			break
		}
	}
	if !found {
		t.Error("module diagnostic not forwarded to builder")
	}
}

func TestRegisterModules_AllBaseModulesFiltered(t *testing.T) {
	// Provide user modules that shadow every base module name.
	baseNames := module.BaseModuleNames()
	var userMods []*module.Module
	for _, name := range baseNames {
		userMods = append(userMods, &module.Module{
			Name:     name,
			Language: types.LanguageSMIv2,
		})
	}
	userMods = append(userMods, &module.Module{
		Name:     "REAL-MIB",
		Language: types.LanguageSMIv2,
	})

	ctx := newResolverContext(userMods, nil, DefaultConfig())
	registerModules(ctx)

	// Each base name should appear exactly once (from base, not user)
	nameCounts := make(map[string]int)
	for _, mod := range ctx.Modules {
		nameCounts[mod.Name]++
	}
	for _, name := range baseNames {
		if nameCounts[name] != 1 {
			t.Errorf("module %q appears %d times, want 1", name, nameCounts[name])
		}
	}
	if nameCounts["REAL-MIB"] != 1 {
		t.Errorf("REAL-MIB appears %d times, want 1", nameCounts["REAL-MIB"])
	}
}

func TestRegisterModules_EmptyModuleList(t *testing.T) {
	// No user modules - should still register all base modules.
	ctx := newResolverContext(nil, nil, DefaultConfig())

	registerModules(ctx)

	baseNames := module.BaseModuleNames()
	if len(ctx.Modules) != len(baseNames) {
		t.Fatalf("got %d modules, want %d (base only)", len(ctx.Modules), len(baseNames))
	}
	for i, name := range baseNames {
		if ctx.Modules[i].Name != name {
			t.Errorf("Modules[%d].Name = %q, want %q", i, ctx.Modules[i].Name, name)
		}
	}
}

func TestRegisterModules_BaseModuleDefinitionNamesCached(t *testing.T) {
	// Verify that base modules have their definition names cached too.
	ctx := newResolverContext(nil, nil, DefaultConfig())

	registerModules(ctx)

	// SNMPv2-SMI should have well-known definitions like "internet", "Integer32"
	var snmpv2smi *module.Module
	for _, mod := range ctx.Modules {
		if mod.Name == "SNMPv2-SMI" {
			snmpv2smi = mod
			break
		}
	}
	if snmpv2smi == nil {
		t.Fatal("SNMPv2-SMI not found")
	}

	defNames := ctx.ModuleDefNames[snmpv2smi]
	if defNames == nil {
		t.Fatal("ModuleDefNames[SNMPv2-SMI] is nil")
	}
	// Spot-check a few well-known names
	for _, name := range []string{"internet", "Integer32", "Counter32", "enterprises"} {
		if _, ok := defNames[name]; !ok {
			t.Errorf("%q not in SNMPv2-SMI definition names", name)
		}
	}
}
