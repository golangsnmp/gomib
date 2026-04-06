package resolver

import (
	"log/slog"
	"slices"
	"sync"

	"github.com/golangsnmp/gomib/internal/cst/lower"
	cstparser "github.com/golangsnmp/gomib/internal/cst/parser"
	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// embeddedBaseModules holds the parsed base modules from the embedded SMI source.
// Parsed once on first use.
var embeddedBaseModules = sync.OnceValue(func() []*module.Module {
	var result []*module.Module
	cfg := types.DefaultConfig()
	for _, name := range module.BaseModuleNames() {
		content := module.EmbeddedContent(name)
		if content == nil {
			continue
		}
		p := cstparser.New(content, nil, cfg)
		file := p.ParseModule()
		mods := lower.Lower(file, content, p.Diagnostics(), cfg)
		for _, mod := range mods {
			if mod.Name == name {
				result = append(result, mod)
				break
			}
		}
	}
	return result
})

// registerModules indexes all modules and seeds the resolver context.
// Base modules are separated from user modules and prepended so that later
// phases can resolve primitives and well-known types. The result is
// stored in ctx.modules.
func registerModules(ctx *resolverContext, inputModules []*module.Module) {
	// Index which base modules are already present in the input.
	inputBaseNames := make(map[string]struct{})
	for _, mod := range inputModules {
		if module.IsBaseModule(mod.Name) {
			inputBaseNames[mod.Name] = struct{}{}
		}
	}

	// Separate base modules from user modules.
	var baseModules, userModules []*module.Module
	for _, mod := range inputModules {
		if module.IsBaseModule(mod.Name) {
			baseModules = append(baseModules, mod)
		} else {
			userModules = append(userModules, mod)
		}
	}

	// Fill in any missing base modules from the embedded source.
	for _, mod := range embeddedBaseModules() {
		if _, ok := inputBaseNames[mod.Name]; !ok {
			baseModules = append(baseModules, mod)
		}
	}

	ctx.Log(slog.LevelDebug, "loaded base modules", slog.Int("count", len(baseModules)))

	ctx.modules = slices.Concat(baseModules, userModules)

	for _, mod := range ctx.modules {
		resolved := model.NewModule(mod.Name)
		model.SetModuleLineTable(resolved, mod.LineTable)
		model.SetModuleSourcePath(resolved, mod.SourcePath)
		model.SetModuleBase(resolved, module.IsBaseModule(mod.Name))
		model.SetModuleLanguage(resolved, mod.Language)
		model.SetModuleRawImports(resolved, mod.Imports)

		if len(mod.ModuleIdentities) > 0 {
			mi := mod.ModuleIdentities[0]
			model.SetModuleOrganization(resolved, mi.Organization)
			model.SetModuleContactInfo(resolved, mi.ContactInfo)
			model.SetModuleDescription(resolved, mi.Description)
			model.SetModuleLastUpdated(resolved, mi.LastUpdated)
			model.SetModuleRevisions(resolved, convertRevisions(mi.Revisions))
		}

		model.AddMibModule(ctx.mib, resolved)
		ctx.moduleToResolved[mod] = resolved
		ctx.resolvedToModule[resolved] = mod

		// Collect diagnostics from parsing and validation
		for _, d := range mod.Diagnostics {
			model.AddMibDiagnostic(ctx.mib, d)
		}

		// Cache pointers to base modules used by the type resolution
		// fallback chain (resolveTypeForModule, resolveType). Many vendor
		// MIBs use types from these modules without importing them, so
		// Normal+ fallback paths need direct access.
		switch mod.Name {
		case moduleSNMPv2SMI:
			ctx.snmpv2SMIModule = mod
		case moduleRFC1155SMI:
			ctx.rfc1155SMIModule = mod
		case moduleSNMPv2TC:
			ctx.snmpv2TCModule = mod
		}

		ctx.moduleIndex[mod.Name] = append(ctx.moduleIndex[mod.Name], mod)

		// Cache definition names for faster import/OID resolution.
		defNames := make(map[string]struct{}, mod.DefinitionCount())
		oidDefNames := make(map[string]struct{})
		for def := range mod.AllDefinitions() {
			name := def.DefinitionName()
			defNames[name] = struct{}{}
			if def.DefinitionOid() != nil {
				oidDefNames[name] = struct{}{}
			}
		}
		ctx.defNames[mod] = defNames
		ctx.oidDefNames[mod] = oidDefNames

		if ctx.TraceEnabled() {
			ctx.Trace("registered module",
				slog.String("name", mod.Name),
				slog.Int("definitions", mod.DefinitionCount()))
		}
	}
}

func convertRevisions(revs []module.Revision) []model.Revision {
	result := make([]model.Revision, len(revs))
	for i, r := range revs {
		result[i] = model.Revision{
			Date:        r.Date,
			Description: r.Description,
			Span:        r.Span,
		}
	}
	return result
}
