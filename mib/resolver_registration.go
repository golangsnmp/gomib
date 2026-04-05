package mib

import (
	"log/slog"
	"slices"

	"github.com/golangsnmp/gomib/internal/module"
)

// registerModules indexes all modules and seeds the resolver context.
// Synthetic base modules are prepended to user modules so that later
// phases can resolve primitives and well-known types. The result is
// stored in ctx.modules.
func registerModules(ctx *resolverContext, inputModules []*module.Module) {
	baseModules := module.CreateBaseModules()

	ctx.Log(slog.LevelDebug, "loaded base modules", slog.Int("count", len(baseModules)))

	var userModules []*module.Module
	for _, mod := range inputModules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		userModules = append(userModules, mod)
	}

	ctx.modules = slices.Concat(baseModules, userModules)

	for _, mod := range ctx.modules {
		resolved := newModule(mod.Name)
		resolved.setLineTable(mod.LineTable)
		resolved.setSourcePath(mod.SourcePath)
		resolved.setBase(module.IsBaseModule(mod.Name))
		resolved.setLanguage(mod.Language)
		resolved.setRawImports(mod.Imports)

		if len(mod.ModuleIdentities) > 0 {
			mi := mod.ModuleIdentities[0]
			resolved.setOrganization(mi.Organization)
			resolved.setContactInfo(mi.ContactInfo)
			resolved.setDescription(mi.Description)
			resolved.setLastUpdated(mi.LastUpdated)
			resolved.setRevisions(convertRevisions(mi.Revisions))
		}

		ctx.mib.addModule(resolved)
		ctx.moduleToResolved[mod] = resolved
		ctx.resolvedToModule[resolved] = mod

		// Collect diagnostics from parsing and validation
		for _, d := range mod.Diagnostics {
			ctx.mib.addDiagnostic(d)
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

func convertRevisions(revs []module.Revision) []Revision {
	result := make([]Revision, len(revs))
	for i, r := range revs {
		result[i] = Revision{
			Date:        r.Date,
			Description: r.Description,
			Span:        r.Span,
		}
	}
	return result
}
