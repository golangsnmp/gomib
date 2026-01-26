// Package resolver provides multi-phase MIB resolution.
//
// Resolution transforms parsed MIB modules into a fully resolved model where all
// symbolic references are concrete, OIDs are computed, and types are linked.
//
// # Resolution Phases
//
// The resolver executes the following phases in order:
//
//  1. Registration: Index modules and their definitions
//  2. Imports: Resolve import references across modules
//  3. Types: Build the type graph and compute base types
//  4. OIDs: Build the OID trie from symbolic references
//  5. Semantics: Infer node kinds (table, row, column, scalar) and create objects
//
// # Usage
//
//	resolver := resolver.New(logger)
//	mib := resolver.Resolve(modules)
package resolver

import (
	"github.com/golangsnmp/gomib/mib"
	"log/slog"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/types"
)

// resolver resolves modules into a Mib.
type resolver struct {
	types.Logger
}

// Resolve resolves parsed modules into a Mib.
// If logger is nil, logging is disabled (zero overhead).
func Resolve(mods []*module.Module, logger *slog.Logger) *mib.Mib {
	r := &resolver{Logger: types.Logger{L: logger}}
	return r.resolve(mods)
}

// resolve resolves modules into a Mib.
func (r *resolver) resolve(mods []*module.Module) *mib.Mib {
	ctx := newResolverContext(mods, r.L)

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "register"))
	registerModules(ctx)
	r.Log(slog.LevelDebug, "phase complete", slog.String("phase", "register"),
		slog.Int("modules", ctx.Builder.ModuleCount()))

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "imports"))
	resolveImports(ctx)

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "types"))
	resolveTypes(ctx)
	r.Log(slog.LevelDebug, "phase complete", slog.String("phase", "types"),
		slog.Int("types", ctx.Builder.TypeCount()))

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "oids"))
	resolveOids(ctx)
	r.Log(slog.LevelDebug, "phase complete", slog.String("phase", "oids"),
		slog.Int("nodes", ctx.Builder.NodeCount()))

	r.Log(slog.LevelDebug, "starting phase", slog.String("phase", "semantics"))
	analyzeSemantics(ctx)
	r.Log(slog.LevelDebug, "phase complete", slog.String("phase", "semantics"),
		slog.Int("objects", ctx.Builder.ObjectCount()),
		slog.Int("notifications", ctx.Builder.NotificationCount()))

	ctx.DropModules()

	// Finalize unresolved references
	ctx.FinalizeUnresolved()

	// Log warnings for unresolved references
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

	r.Log(slog.LevelInfo, "resolution complete",
		slog.Int("modules", ctx.Builder.ModuleCount()),
		slog.Int("types", ctx.Builder.TypeCount()),
		slog.Int("nodes", ctx.Builder.NodeCount()))

	return ctx.Builder.Mib()
}
