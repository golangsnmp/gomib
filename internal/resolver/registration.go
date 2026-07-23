package resolver

import (
	"log/slog"
	"slices"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
)

// registerModules indexes all modules and seeds the resolver context.
// Base modules are separated from user modules and prepended so that later
// phases can resolve primitives and well-known types. The caller is
// responsible for including base modules in inputModules (the load pipeline
// provides them via embeddedSource). The result is stored in ctx.modules.
func registerModules(ctx *resolverContext, inputModules []*module.Module) {
	// Separate base modules from user modules.
	var baseModules, userModules []*module.Module
	for _, mod := range inputModules {
		if module.IsBaseModule(mod.Name) {
			baseModules = append(baseModules, mod)
		} else {
			userModules = append(userModules, mod)
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
		for i := range mod.Diagnostics {
			model.AddMibDiagnostic(ctx.mib, &mod.Diagnostics[i])
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
