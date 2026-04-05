package resolver

import (
	"log/slog"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

type resolver struct {
	types.Logger
	strictness model.ResolverStrictness
	diagConfig model.DiagnosticConfig
}

// Resolve transforms normalized modules into a fully resolved [Mib].
//
// Resolution runs these phases in order:
//
//  1. Registration: index modules and definitions
//  2. Imports: resolve imported symbols across modules
//  3. Types: build type links and compute base types
//  4. OIDs: build the OID tree from symbolic references
//  5. Semantics: infer node kinds and create objects
//
// If logger is nil, logging is disabled. If strictness is nil, defaults to
// [model.ResolverNormal]. If diagConfig is nil, defaults to [DefaultConfig].
func Resolve(mods []*module.Module, logger *slog.Logger, strictness *model.ResolverStrictness, diagConfig *model.DiagnosticConfig) *model.Mib {
	resolverStrictness := model.ResolverNormal
	if strictness != nil {
		resolverStrictness = *strictness
	}
	cfg := model.DefaultConfig()
	if diagConfig != nil {
		cfg = *diagConfig
	}
	r := &resolver{
		Logger:     types.Logger{L: logger},
		strictness: resolverStrictness,
		diagConfig: cfg,
	}
	return r.resolve(mods)
}

func (r *resolver) resolve(mods []*module.Module) *model.Mib {
	ctx := newResolverContext(r.L, r.strictness, r.diagConfig)

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "register"))
	registerModules(ctx, mods)
	r.Log(slog.LevelDebug, "phase complete", slog.String("phase", "register"),
		slog.Int("modules", len(ctx.mib.Modules())))

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "imports"))
	resolveImports(ctx)
	resolveTransitiveImports(ctx)
	r.Log(slog.LevelDebug, "phase complete", slog.String("phase", "imports"))

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "types"))
	resolveTypes(ctx)
	r.Log(slog.LevelDebug, "phase complete", slog.String("phase", "types"),
		slog.Int("types", len(ctx.mib.Types())))

	checkBasetypeImports(ctx)

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "oids"))
	resolveOids(ctx)
	r.Log(slog.LevelDebug, "phase complete", slog.String("phase", "oids"))

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "semantics"))
	resolveSemantics(ctx)
	nodeCount := 0
	for range ctx.mib.Nodes() {
		nodeCount++
	}
	r.Log(slog.LevelDebug, "phase complete", slog.String("phase", "semantics"),
		slog.Int("nodes", nodeCount))

	checkUnusedImports(ctx)
	checkObsoleteImports(ctx)
	copyUsedImportsToModules(ctx)
	copyResolvedImportsToModules(ctx)

	ctx.FinalizeDiagnostics()

	unresolvedCounts := map[model.UnresolvedKind]int{}
	for _, u := range ctx.mib.Unresolved() {
		unresolvedCounts[u.Kind]++
	}
	for _, kind := range []model.UnresolvedKind{
		model.UnresolvedImport,
		model.UnresolvedType,
		model.UnresolvedOID,
		model.UnresolvedIndex,
		model.UnresolvedNotificationObject,
	} {
		if n := unresolvedCounts[kind]; n > 0 {
			r.Log(slog.LevelWarn, "unresolved "+kind.String(), slog.Int("count", n))
		}
	}

	model.SetMibNodeCount(ctx.mib, nodeCount)
	m := ctx.mib

	r.Log(slog.LevelInfo, "resolution complete",
		slog.Int("modules", len(m.Modules())),
		slog.Int("types", len(m.Types())),
		slog.Int("nodes", m.NodeCount()))

	return m
}
