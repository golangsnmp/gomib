package resolver

import (
	"log/slog"

	"github.com/golangsnmp/gomib/internal/mibimpl"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/mib"
)

// registerModules registers modules and definitions.
func registerModules(ctx *ResolverContext) {
	baseModules := module.CreateBaseModules()

	if ctx.TraceEnabled() {
		ctx.Trace("loaded base modules", slog.Int("count", len(baseModules)))
	}

	var userModules []*module.Module
	for _, mod := range ctx.Modules {
		if module.IsBaseModule(mod.Name) {
			continue
		}
		userModules = append(userModules, mod)
	}

	ctx.Modules = append(baseModules, userModules...)

	for _, mod := range ctx.Modules {
		resolved := mibimpl.NewModule(mod.Name)
		resolved.SetLanguage(convertLanguage(mod.Language))

		ctx.Builder.AddModule(resolved)
		ctx.ModuleToResolved[mod] = resolved

		if mod.Name == "SNMPv2-SMI" {
			ctx.Snmpv2SMIModule = mod
		}

		ctx.ModuleIndex[mod.Name] = append(ctx.ModuleIndex[mod.Name], mod)

		// Cache definition names for faster import resolution
		defNames := make(map[string]struct{}, len(mod.Definitions))
		for _, def := range mod.Definitions {
			defNames[def.DefinitionName()] = struct{}{}
		}
		ctx.ModuleDefNames[mod] = defNames

		if ctx.TraceEnabled() {
			ctx.Trace("registered module",
				slog.String("name", mod.Name),
				slog.Int("definitions", len(mod.Definitions)))
		}
	}
}

func convertLanguage(lang module.SmiLanguage) mib.Language {
	switch lang {
	case module.SmiLanguageSMIv1:
		return mib.LanguageSMIv1
	case module.SmiLanguageSMIv2:
		return mib.LanguageSMIv2
	default:
		return mib.LanguageSMIv1
	}
}
