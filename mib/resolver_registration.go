package mib

import (
	"log/slog"

	"github.com/golangsnmp/gomib/internal/module"
)

// registerModules indexes all modules and seeds the resolver context.
// Synthetic base modules are prepended to user modules so that later
// phases can resolve primitives and well-known types.
func registerModules(ctx *resolverContext) {
	baseModules := module.CreateBaseModules()

	ctx.Log(slog.LevelDebug, "loaded base modules", slog.Int("count", len(baseModules)))

	var userModules []*module.Module
	for _, mod := range ctx.Modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		userModules = append(userModules, mod)
	}

	ctx.Modules = append(baseModules, userModules...)

	for _, mod := range ctx.Modules {
		resolved := newModule(mod.Name)
		resolved.setSourcePath(mod.SourcePath)
		resolved.setLanguage(mod.Language)

		for _, def := range mod.Definitions {
			if mi, ok := def.(*module.ModuleIdentity); ok {
				resolved.setOrganization(mi.Organization)
				resolved.setContactInfo(mi.ContactInfo)
				resolved.setDescription(mi.Description)
				resolved.setRevisions(convertRevisions(mi.Revisions))
				break
			}
		}

		ctx.Mib.addModule(resolved)
		ctx.ModuleToResolved[mod] = resolved
		ctx.ResolvedToModule[resolved] = mod

		// Collect diagnostics from parsing and lowering
		for _, d := range mod.Diagnostics {
			ctx.Mib.addDiagnostic(d)
		}

		// Cache pointers to base modules used by the type resolution
		// fallback chain (LookupTypeForModule, LookupType). Many vendor
		// MIBs use types from these modules without importing them, so
		// the resolver needs direct access for permissive-mode lookups.
		if mod.Name == "SNMPv2-SMI" {
			ctx.Snmpv2SMIModule = mod
		}
		if mod.Name == "RFC1155-SMI" {
			ctx.Rfc1155SMIModule = mod
		}
		if mod.Name == "SNMPv2-TC" {
			ctx.Snmpv2TCModule = mod
		}

		ctx.ModuleIndex[mod.Name] = append(ctx.ModuleIndex[mod.Name], mod)

		// Cache definition names for faster import/OID resolution
		defNames := make(map[string]struct{}, len(mod.Definitions))
		oidDefNames := make(map[string]struct{})
		for _, def := range mod.Definitions {
			name := def.DefinitionName()
			defNames[name] = struct{}{}
			if def.DefinitionOid() != nil {
				oidDefNames[name] = struct{}{}
			}
		}
		ctx.ModuleDefNames[mod] = defNames
		ctx.ModuleOidDefNames[mod] = oidDefNames

		if ctx.TraceEnabled() {
			ctx.Trace("registered module",
				slog.String("name", mod.Name),
				slog.Int("definitions", len(mod.Definitions)))
		}
	}
}

func convertRevisions(revs []module.Revision) []Revision {
	result := make([]Revision, len(revs))
	for i, r := range revs {
		result[i] = Revision{
			Date:        r.Date,
			Description: r.Description,
		}
	}
	return result
}
