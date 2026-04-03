package mib

import (
	"log/slog"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

type resolver struct {
	types.Logger
	strictness ResolverStrictness
	diagConfig DiagnosticConfig
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
// [ResolverNormal]. If diagConfig is nil, defaults to [DefaultConfig].
func Resolve(mods []*module.Module, logger *slog.Logger, strictness *ResolverStrictness, diagConfig *DiagnosticConfig) *Mib {
	resolverStrictness := ResolverNormal
	if strictness != nil {
		resolverStrictness = *strictness
	}
	cfg := DefaultConfig()
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

func (r *resolver) resolve(mods []*module.Module) *Mib {
	ctx := newResolverContext(r.L, r.strictness, r.diagConfig)

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "register"))
	registerModules(ctx, mods)
	r.Log(slog.LevelDebug, "phase complete", slog.String("phase", "register"),
		slog.Int("modules", len(ctx.mib.modules)))

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "imports"))
	resolveImports(ctx)
	resolveTransitiveImports(ctx)
	r.Log(slog.LevelDebug, "phase complete", slog.String("phase", "imports"))

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "types"))
	resolveTypes(ctx)
	r.Log(slog.LevelDebug, "phase complete", slog.String("phase", "types"),
		slog.Int("types", len(ctx.mib.types)))

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

	ctx.DropModules()

	ctx.FinalizeUnresolved()

	if len(ctx.unresolvedImports) > 0 {
		r.Log(slog.LevelWarn, "unresolved imports",
			slog.Int("count", len(ctx.unresolvedImports)))
	}
	if len(ctx.unresolvedTypes) > 0 {
		r.Log(slog.LevelWarn, "unresolved types",
			slog.Int("count", len(ctx.unresolvedTypes)))
	}
	if len(ctx.unresolvedOids) > 0 {
		r.Log(slog.LevelWarn, "unresolved OIDs",
			slog.Int("count", len(ctx.unresolvedOids)))
	}
	if len(ctx.unresolvedIndexes) > 0 {
		r.Log(slog.LevelWarn, "unresolved indexes",
			slog.Int("count", len(ctx.unresolvedIndexes)))
	}

	ctx.mib.setNodeCount(nodeCount)
	m := ctx.mib

	r.Log(slog.LevelInfo, "resolution complete",
		slog.Int("modules", len(m.modules)),
		slog.Int("types", len(m.types)),
		slog.Int("nodes", m.NodeCount()))

	return m
}
