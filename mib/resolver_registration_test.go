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
	baseNames := module.BaseModuleNames()

	t.Run("with user modules", func(t *testing.T) {
		userMod := &module.Module{Name: "MY-MIB", Language: types.LanguageSMIv2}
		addDefs(userMod, []module.Definition{
			&module.ObjectType{DefBase: module.DefBase{Name: "myObject", Span: types.Synthetic}},
		})
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

		registerModules(ctx, []*module.Module{userMod})

		testutil.Equal(t, len(baseNames)+1, len(ctx.modules), "module count (base=%d + user=1)", len(baseNames))
		for i, name := range baseNames {
			testutil.Equal(t, name, ctx.modules[i].Name, "Modules[].Name")
		}
		last := ctx.modules[len(ctx.modules)-1]
		testutil.Equal(t, "MY-MIB", last.Name, "last module")
	})

	t.Run("no user modules", func(t *testing.T) {
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

		registerModules(ctx, nil)

		testutil.Len(t, ctx.modules, len(baseNames), "modules")
		for i, name := range baseNames {
			testutil.Equal(t, name, ctx.modules[i].Name, "Modules[].Name")
		}
	})
}

func TestRegisterModules_UserBaseNamesFiltered(t *testing.T) {
	t.Run("single base name shadowed", func(t *testing.T) {
		userSNMP := &module.Module{
			Name:     "SNMPv2-SMI",
			Language: types.LanguageSMIv2,
		}
		userMod := &module.Module{
			Name:     "MY-MIB",
			Language: types.LanguageSMIv2,
		}
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

		registerModules(ctx, []*module.Module{userSNMP, userMod})

		count := 0
		for _, mod := range ctx.modules {
			if mod.Name == "SNMPv2-SMI" {
				count++
			}
		}
		testutil.Equal(t, 1, count, "found")

		found := false
		for _, mod := range ctx.modules {
			if mod.Name == "MY-MIB" {
				found = true
				break
			}
		}
		testutil.True(t, found, "MY-MIB not found in ctx.modules")
	})

	t.Run("all base names shadowed", func(t *testing.T) {
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

		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())
		registerModules(ctx, userMods)

		nameCounts := make(map[string]int)
		for _, mod := range ctx.modules {
			nameCounts[mod.Name]++
		}
		for _, name := range baseNames {
			testutil.Equal(t, 1, nameCounts[name], "module")
		}
		testutil.Equal(t, 1, nameCounts["REAL-MIB"], "REAL-MIB appears")
	})
}

func TestRegisterModules_ModuleIndexPopulated(t *testing.T) {
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv2,
	}
	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

	registerModules(ctx, []*module.Module{userMod})

	// Every module should be indexed by name
	for _, mod := range ctx.modules {
		entries, ok := ctx.moduleIndex[mod.Name]
		testutil.True(t, ok, "ModuleIndex missing entry for %q", mod.Name)
		testutil.True(t, slices.Contains(entries, mod), "ModuleIndex[] does not contain the module pointer")
	}
}

func TestRegisterModules_BaseModulePointersCached(t *testing.T) {
	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

	registerModules(ctx, nil)

	testutil.NotNil(t, ctx.snmpv2SMIModule, "Snmpv2SMIModule")
	testutil.Equal(t, "SNMPv2-SMI", ctx.snmpv2SMIModule.Name, "Snmpv2SMIModule.Name")

	testutil.NotNil(t, ctx.rfc1155SMIModule, "Rfc1155SMIModule")
	testutil.Equal(t, "RFC1155-SMI", ctx.rfc1155SMIModule.Name, "Rfc1155SMIModule.Name")

	testutil.NotNil(t, ctx.snmpv2TCModule, "Snmpv2TCModule")
	testutil.Equal(t, "SNMPv2-TC", ctx.snmpv2TCModule.Name, "Snmpv2TCModule.Name")
}

func TestRegisterModules_DefinitionNamesCached(t *testing.T) {
	t.Run("user module", func(t *testing.T) {
		userMod := &module.Module{Name: "MY-MIB", Language: types.LanguageSMIv2}
		addDefs(userMod, []module.Definition{
			&module.ObjectType{DefBase: module.DefBase{Name: "fooObject", Span: types.Synthetic}},
			&module.TypeDef{DefBase: module.DefBase{Name: "BarType", Span: types.Synthetic}},
		})
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

		registerModules(ctx, []*module.Module{userMod})

		var found *module.Module
		for _, mod := range ctx.modules {
			if mod.Name == "MY-MIB" {
				found = mod
				break
			}
		}
		testutil.NotNil(t, found, "MY-MIB not found in ctx.modules")

		defNames := ctx.defNames[found]
		testutil.NotNil(t, defNames, "ModuleDefNames[MY-MIB] is nil")
		_, hasFoo := defNames["fooObject"]
		testutil.True(t, hasFoo, "fooObject not in ModuleDefNames")
		_, hasBar := defNames["BarType"]
		testutil.True(t, hasBar, "BarType not in ModuleDefNames")
		_, hasNonExistent := defNames["nonExistent"]
		testutil.False(t, hasNonExistent, "nonExistent should not be in ModuleDefNames")
	})

	t.Run("base module", func(t *testing.T) {
		ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

		registerModules(ctx, nil)

		var snmpv2smi *module.Module
		for _, mod := range ctx.modules {
			if mod.Name == "SNMPv2-SMI" {
				snmpv2smi = mod
				break
			}
		}
		testutil.NotNil(t, snmpv2smi, "SNMPv2-SMI not found")

		defNames := ctx.defNames[snmpv2smi]
		testutil.NotNil(t, defNames, "ModuleDefNames[SNMPv2-SMI] is nil")
		for _, name := range []string{"internet", "Integer32", "Counter32", "enterprises"} {
			_, ok := defNames[name]
			testutil.True(t, ok, "%q not in SNMPv2-SMI definition names", name)
		}
	})
}

func TestRegisterModules_ModuleToResolvedMapping(t *testing.T) {
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv2,
	}
	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

	registerModules(ctx, []*module.Module{userMod})

	for _, mod := range ctx.modules {
		resolved, ok := ctx.moduleToResolved[mod]
		testutil.True(t, ok, "ModuleToResolved missing entry for %q", mod.Name)
		testutil.Equal(t, mod.Name, resolved.Name(), "resolved name")
		// Check reverse mapping
		reverse, ok := ctx.resolvedToModule[resolved]
		testutil.True(t, ok, "ResolvedToModule missing entry for %q", mod.Name)
		testutil.Equal(t, mod, reverse, "reverse mapping for")
	}
}

func TestRegisterModules_LanguageSetOnResolved(t *testing.T) {
	userMod := &module.Module{
		Name:     "MY-MIB",
		Language: types.LanguageSMIv1,
	}
	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

	registerModules(ctx, []*module.Module{userMod})

	var found *module.Module
	for _, mod := range ctx.modules {
		if mod.Name == "MY-MIB" {
			found = mod
			break
		}
	}
	testutil.NotNil(t, found, "MY-MIB not found")

	resolved := ctx.moduleToResolved[found]
	testutil.NotNil(t, resolved, "no resolved module for MY-MIB")
	testutil.Equal(t, LanguageSMIv1, resolved.Language(), "resolved language")
}

func TestRegisterModules_ModuleIdentityExtracted(t *testing.T) {
	mi := &module.ModuleIdentity{
		DefBase:      module.DefBase{Name: "myMIB", Span: types.Synthetic},
		Organization: "ACME Corp",
		ContactInfo:  "support@acme.example",
		Description:  "Test MIB module",
		Revisions: []module.Revision{
			{Date: "2024-06-01", Description: "Rev 2"},
			{Date: "2024-01-15", Description: "Rev 1"},
		},
		Oid: module.NewOidAssignment(nil, types.Synthetic),
	}
	userMod := &module.Module{Name: "MY-MIB", Language: types.LanguageSMIv2}
	addDefs(userMod, []module.Definition{
		// Some other definition before MODULE-IDENTITY
		&module.ObjectType{DefBase: module.DefBase{Name: "someObj", Span: types.Synthetic}},
		mi,
	})
	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

	registerModules(ctx, []*module.Module{userMod})

	var found *module.Module
	for _, mod := range ctx.modules {
		if mod.Name == "MY-MIB" {
			found = mod
			break
		}
	}
	testutil.NotNil(t, found, "MY-MIB not found")

	resolved := ctx.moduleToResolved[found]
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
	userMod := &module.Module{Name: "MY-MIB", Language: types.LanguageSMIv1}
	addDefs(userMod, []module.Definition{
		&module.ObjectType{DefBase: module.DefBase{Name: "someObj", Span: types.Synthetic}},
	})
	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

	registerModules(ctx, []*module.Module{userMod})

	var found *module.Module
	for _, mod := range ctx.modules {
		if mod.Name == "MY-MIB" {
			found = mod
			break
		}
	}
	testutil.NotNil(t, found, "MY-MIB not found")

	resolved := ctx.moduleToResolved[found]
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
	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

	registerModules(ctx, []*module.Module{userMod})

	baseNames := module.BaseModuleNames()
	wantCount := len(baseNames) + 1
	testutil.Len(t, ctx.mib.Modules(), wantCount, "len(Builder.Modules())")

	// Verify the builder can look up each module by name
	for _, name := range baseNames {
		testutil.NotNil(t, ctx.mib.Module(name), "Builder.Module() returned nil")
	}
	testutil.NotNil(t, ctx.mib.Module("MY-MIB"), "Builder.Module(MY-MIB) returned nil")
}

func TestModuleImports_Grouping(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		m := newModule("TEST-MIB")
		got := m.Imports()
		testutil.Len(t, got, 0, "Imports() on empty module")
	})

	t.Run("single import", func(t *testing.T) {
		m := newModule("TEST-MIB")
		m.setRawImports([]module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{Start: 10, End: 20}),
		})
		got := m.Imports()
		testutil.Len(t, got, 1, "Imports() result")
		testutil.Equal(t, got[0].Module, "SNMPv2-SMI", "module name")
		testutil.Len(t, got[0].Symbols, 1, "symbols")
		testutil.Equal(t, got[0].Symbols[0].Name, "enterprises", "symbol name")
		testutil.Equal(t, got[0].Symbols[0].Span, Span{Start: 10, End: 20}, "span preserved")
	})

	t.Run("multiple symbols from same module sorted", func(t *testing.T) {
		m := newModule("TEST-MIB")
		m.setRawImports([]module.Import{
			module.NewImport("SNMPv2-SMI", "Counter32", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		})
		got := m.Imports()
		testutil.Len(t, got, 1, "Imports() result")
		testutil.Len(t, got[0].Symbols, 3, "symbols")
		// Symbols should be sorted alphabetically.
		testutil.Equal(t, got[0].Symbols[0].Name, "Counter32", "first symbol")
		testutil.Equal(t, got[0].Symbols[1].Name, "Integer32", "second symbol")
		testutil.Equal(t, got[0].Symbols[2].Name, "enterprises", "third symbol")
	})

	t.Run("multiple modules preserve declaration order", func(t *testing.T) {
		m := newModule("TEST-MIB")
		m.setRawImports([]module.Import{
			module.NewImport("SNMPv2-TC", "DisplayString", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
			module.NewImport("SNMPv2-TC", "TruthValue", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Counter32", types.Span{}),
		})
		got := m.Imports()
		testutil.Len(t, got, 2, "grouped modules")
		// Module order follows first occurrence.
		testutil.Equal(t, got[0].Module, "SNMPv2-TC", "first module")
		testutil.Equal(t, got[1].Module, "SNMPv2-SMI", "second module")
		// Symbols within each module are sorted.
		testutil.Equal(t, got[0].Symbols[0].Name, "DisplayString", "TC first")
		testutil.Equal(t, got[0].Symbols[1].Name, "TruthValue", "TC second")
		testutil.Equal(t, got[1].Symbols[0].Name, "Counter32", "SMI first")
		testutil.Equal(t, got[1].Symbols[1].Name, "Integer32", "SMI second")
	})
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
	ctx := newResolverContext(nil, ResolverNormal, DefaultConfig())

	registerModules(ctx, []*module.Module{userMod})

	// Build and check diagnostics are present
	built := ctx.mib
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
