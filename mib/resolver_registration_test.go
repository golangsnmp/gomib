package mib

import (
	"slices"
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestConvertRevisions(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		got := convertRevisions(nil)
		testutil.Len(t, got, 0, "convertRevisions(nil) returned")
	})

	t.Run("empty input", func(t *testing.T) {
		got := convertRevisions([]module.Revision{})
		testutil.Len(t, got, 0, "convertRevisions([]) returned")
	})

	t.Run("single revision", func(t *testing.T) {
		input := []module.Revision{
			{Date: "2024-01-15", Description: "Initial version"},
		}
		got := convertRevisions(input)
		testutil.Len(t, got, 1, "revisions")
		testutil.Equal(t, "2024-01-15", got[0].Date, "Date")
		testutil.Equal(t, "Initial version", got[0].Description, "Description")
	})

	t.Run("multiple revisions", func(t *testing.T) {
		input := []module.Revision{
			{Date: "2024-06-01", Description: "Added new objects"},
			{Date: "2024-01-15", Description: "Initial version"},
			{Date: "2023-12-01", Description: "Draft"},
		}
		got := convertRevisions(input)
		testutil.Len(t, got, 3, "revisions")
		for i, r := range input {
			testutil.Equal(t, r.Date, got[i].Date, "revision[].Date")
			testutil.Equal(t, r.Description, got[i].Description, "revision[].Description")
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
	testutil.Equal(t, len(baseNames)+1, len(ctx.Modules), "module count (base=%d + user=1)", len(baseNames))
	for i, name := range baseNames {
		testutil.Equal(t, name, ctx.Modules[i].Name, "Modules[].Name")
	}
	last := ctx.Modules[len(ctx.Modules)-1]
	testutil.Equal(t, "MY-MIB", last.Name, "last module")
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
	testutil.Equal(t, 1, count, "found")

	// MY-MIB should still be present
	found := false
	for _, mod := range ctx.Modules {
		if mod.Name == "MY-MIB" {
			found = true
			break
		}
	}
	testutil.True(t, found, "MY-MIB not found in ctx.Modules")
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
		testutil.True(t, ok, "ModuleIndex missing entry for %q", mod.Name)
		testutil.True(t, slices.Contains(entries, mod), "ModuleIndex[] does not contain the module pointer")
	}
}

func TestRegisterModules_BaseModulePointersCached(t *testing.T) {
	ctx := newResolverContext(nil, nil, DefaultConfig())

	registerModules(ctx)

	testutil.NotNil(t, ctx.Snmpv2SMIModule, "Snmpv2SMIModule")
	testutil.Equal(t, "SNMPv2-SMI", ctx.Snmpv2SMIModule.Name, "Snmpv2SMIModule.Name")

	testutil.NotNil(t, ctx.Rfc1155SMIModule, "Rfc1155SMIModule")
	testutil.Equal(t, "RFC1155-SMI", ctx.Rfc1155SMIModule.Name, "Rfc1155SMIModule.Name")

	testutil.NotNil(t, ctx.Snmpv2TCModule, "Snmpv2TCModule")
	testutil.Equal(t, "SNMPv2-TC", ctx.Snmpv2TCModule.Name, "Snmpv2TCModule.Name")
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
	testutil.NotNil(t, found, "MY-MIB not found in ctx.Modules")

	defNames := ctx.ModuleDefNames[found]
	testutil.NotNil(t, defNames, "ModuleDefNames[MY-MIB] is nil")
	_, hasFoo := defNames["fooObject"]
	testutil.True(t, hasFoo, "fooObject not in ModuleDefNames")
	_, hasBar := defNames["BarType"]
	testutil.True(t, hasBar, "BarType not in ModuleDefNames")
	_, hasNonExistent := defNames["nonExistent"]
	testutil.False(t, hasNonExistent, "nonExistent should not be in ModuleDefNames")
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
		testutil.True(t, ok, "ModuleToResolved missing entry for %q", mod.Name)
		testutil.Equal(t, mod.Name, resolved.Name(), "resolved name")
		// Check reverse mapping
		reverse, ok := ctx.ResolvedToModule[resolved]
		testutil.True(t, ok, "ResolvedToModule missing entry for %q", mod.Name)
		testutil.Equal(t, mod, reverse, "reverse mapping for")
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
	testutil.NotNil(t, found, "MY-MIB not found")

	resolved := ctx.ModuleToResolved[found]
	testutil.NotNil(t, resolved, "no resolved module for MY-MIB")
	testutil.Equal(t, LanguageSMIv1, resolved.Language(), "resolved language")
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
	testutil.NotNil(t, found, "MY-MIB not found")

	resolved := ctx.ModuleToResolved[found]
	testutil.NotNil(t, resolved, "no resolved module for MY-MIB")

	testutil.Equal(t, "ACME Corp", resolved.Organization(), "Organization")
	testutil.Equal(t, "support@acme.example", resolved.ContactInfo(), "ContactInfo")
	testutil.Equal(t, "Test MIB module", resolved.Description(), "Description")
	revs := resolved.Revisions()
	testutil.Len(t, revs, 2, "revisions")
	testutil.Equal(t, "2024-06-01", revs[0].Date, "revision[0].Date")
	testutil.Equal(t, "Rev 2", revs[0].Description, "revision[0].Description")
	testutil.Equal(t, "2024-01-15", revs[1].Date, "revision[1].Date")
	testutil.Equal(t, "Rev 1", revs[1].Description, "revision[1].Description")
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
	testutil.NotNil(t, found, "MY-MIB not found")

	resolved := ctx.ModuleToResolved[found]
	testutil.Equal(t, "", resolved.Organization(), "Organization")
	testutil.Equal(t, "", resolved.ContactInfo(), "ContactInfo")
	testutil.Equal(t, "", resolved.Description(), "Description")
	testutil.Len(t, resolved.Revisions(), 0, "Revisions has")
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
	testutil.Len(t, ctx.Mib.Modules(), wantCount, "len(Builder.Modules())")

	// Verify the builder can look up each module by name
	for _, name := range baseNames {
		testutil.NotNil(t, ctx.Mib.Module(name), "Builder.Module() returned nil")
	}
	testutil.NotNil(t, ctx.Mib.Module("MY-MIB"), "Builder.Module(MY-MIB) returned nil")
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
	testutil.True(t, found, "module diagnostic not forwarded to builder")
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
		testutil.Equal(t, 1, nameCounts[name], "module")
	}
	testutil.Equal(t, 1, nameCounts["REAL-MIB"], "REAL-MIB appears")
}

func TestRegisterModules_EmptyModuleList(t *testing.T) {
	// No user modules - should still register all base modules.
	ctx := newResolverContext(nil, nil, DefaultConfig())

	registerModules(ctx)

	baseNames := module.BaseModuleNames()
	testutil.Len(t, ctx.Modules, len(baseNames), "modules")
	for i, name := range baseNames {
		testutil.Equal(t, name, ctx.Modules[i].Name, "Modules[].Name")
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
	testutil.NotNil(t, snmpv2smi, "SNMPv2-SMI not found")

	defNames := ctx.ModuleDefNames[snmpv2smi]
	testutil.NotNil(t, defNames, "ModuleDefNames[SNMPv2-SMI] is nil")
	// Spot-check a few well-known names
	for _, name := range []string{"internet", "Integer32", "Counter32", "enterprises"} {
		_, ok := defNames[name]
		testutil.True(t, ok, "%q not in SNMPv2-SMI definition names", name)
	}
}
