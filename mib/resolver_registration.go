package mib

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/golangsnmp/gomib/internal/module"
)

// registerModules indexes all modules and seeds the resolver context.
// Synthetic base modules are prepended to user modules so that later
// phases can resolve primitives and well-known types.
func registerModules(ctx *resolverContext) {
	baseModules := module.CreateBaseModules()

	ctx.Log(slog.LevelDebug, "loaded base modules", slog.Int("count", len(baseModules)))

	var userModules []*module.Module
	for _, mod := range ctx.modules {
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
		resolved.setImports(groupImports(mod.Imports))

		for _, def := range mod.Definitions {
			mi, ok := def.(*module.ModuleIdentity)
			if !ok {
				continue
			}
			resolved.setOrganization(mi.Organization)
			resolved.setContactInfo(mi.ContactInfo)
			resolved.setDescription(mi.Description)
			resolved.setLastUpdated(mi.LastUpdated)
			resolved.setRevisions(convertRevisions(mi.Revisions))
			break
		}

		ctx.mib.addModule(resolved)
		ctx.moduleToResolved[mod] = resolved
		ctx.resolvedToModule[resolved] = mod

		// Collect diagnostics from parsing and lowering
		for _, d := range mod.Diagnostics {
			ctx.mib.addDiagnostic(d)
		}

		// Cache pointers to base modules used by the type resolution
		// fallback chain (LookupTypeForModule, LookupType). Many vendor
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
		// ModuleOidDefNames is pre-populated for user modules in
		// newResolverContext, so only build it for base modules.
		defNames := make(map[string]struct{}, len(mod.Definitions))
		var oidDefNames map[string]struct{}
		if _, exists := ctx.moduleOidDefNames[mod]; !exists {
			oidDefNames = make(map[string]struct{})
		}
		for _, def := range mod.Definitions {
			name := def.DefinitionName()
			defNames[name] = struct{}{}
			if oidDefNames != nil && def.DefinitionOid() != nil {
				oidDefNames[name] = struct{}{}
			}
		}
		ctx.moduleDefNames[mod] = defNames
		if oidDefNames != nil {
			ctx.moduleOidDefNames[mod] = oidDefNames
		}

		if ctx.TraceEnabled() {
			ctx.Trace("registered module",
				slog.String("name", mod.Name),
				slog.Int("definitions", len(mod.Definitions)))
		}
	}
}

// groupImports converts flat per-symbol imports into grouped-by-module form.
func groupImports(raw []module.Import) []Import {
	if len(raw) == 0 {
		return nil
	}
	var order []string
	grouped := make(map[string][]ImportSymbol)
	for _, imp := range raw {
		if _, ok := grouped[imp.Module]; !ok {
			order = append(order, imp.Module)
		}
		grouped[imp.Module] = append(grouped[imp.Module], ImportSymbol{
			Name: imp.Symbol,
			Span: imp.Span,
		})
	}
	result := make([]Import, 0, len(order))
	for _, modName := range order {
		syms := grouped[modName]
		slices.SortFunc(syms, func(a, b ImportSymbol) int {
			return cmp.Compare(a.Name, b.Name)
		})
		result = append(result, Import{Module: modName, Symbols: syms})
	}
	return result
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
