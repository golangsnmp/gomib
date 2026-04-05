package mib

import (
	"cmp"
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
		// If Definitions is populated but typed slices are empty, backfill the
		// typed slices. This handles modules constructed via struct literals
		// without using AddDefinition. Base modules are always built correctly.
		if !module.IsBaseModule(mod.Name) && len(mod.Definitions) > 0 && mod.DefinitionCount() == 0 {
			for _, def := range mod.Definitions {
				switch d := def.(type) {
				case *module.ObjectType:
					mod.ObjectTypes = append(mod.ObjectTypes, d)
				case *module.TypeDef:
					mod.TypeDefs = append(mod.TypeDefs, d)
				case *module.Notification:
					mod.Notifications = append(mod.Notifications, d)
				case *module.ModuleIdentity:
					mod.ModuleIdentities = append(mod.ModuleIdentities, d)
				case *module.ObjectIdentity:
					mod.ObjectIdentities = append(mod.ObjectIdentities, d)
				case *module.ValueAssignment:
					mod.ValueAssignments = append(mod.ValueAssignments, d)
				case *module.ObjectGroup:
					mod.ObjectGroups = append(mod.ObjectGroups, d)
				case *module.NotificationGroup:
					mod.NotificationGroups = append(mod.NotificationGroups, d)
				case *module.ModuleCompliance:
					mod.Compliances = append(mod.Compliances, d)
				case *module.AgentCapabilities:
					mod.Capabilities = append(mod.Capabilities, d)
				}
			}
		}
	}

	for _, mod := range ctx.modules {
		resolved := newModule(mod.Name)
		resolved.setLineTable(mod.LineTable)
		resolved.setSourcePath(mod.SourcePath)
		resolved.setBase(module.IsBaseModule(mod.Name))
		resolved.setLanguage(mod.Language)
		resolved.setImports(groupImports(mod.Imports))

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
