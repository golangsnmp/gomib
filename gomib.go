package gomib

import (
	"context"
	"errors"
	"log/slog"

	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

// ErrNoSources is returned when Load is called with no sources.
var ErrNoSources = errors.New("no MIB sources provided")

// ErrMissingModules is returned when WithModules names are not found in any source.
// The Mib is still returned with whatever modules could be loaded.
var ErrMissingModules = errors.New("requested modules not found")

// ErrDiagnosticThreshold is returned when diagnostics exceed the configured FailAt severity.
// The Mib is still returned with all resolved data.
var ErrDiagnosticThreshold = errors.New("diagnostic threshold exceeded")

// LevelTrace is a custom log level more verbose than Debug.
// Use for per-item iteration logging (tokens, OID nodes, imports).
// Enable with: &slog.HandlerOptions{Level: slog.Level(-8)}
const LevelTrace = types.LevelTrace

// LoadOption configures Load.
type LoadOption func(*loadConfig)

type loadConfig struct {
	logger             *slog.Logger
	systemPaths        bool
	resolverStrictness mib.ResolverStrictness
	diagConfig         mib.DiagnosticConfig
	sources            []Source
	modules            []string
	hasModules         bool // true when WithModules was called (even with empty list)
}

// WithLogger sets the logger for debug/trace output.
// If not set, no logging occurs (zero overhead).
func WithLogger(logger *slog.Logger) LoadOption {
	return func(c *loadConfig) { c.logger = logger }
}

// WithDiagnosticConfig sets the diagnostic reporting and failure configuration.
//
// DiagnosticConfig controls independent diagnostic policies:
//   - Reporting: the baseline severity retained in [mib.Mib.Diagnostics].
//   - Overrides: the effective severity stored on matching diagnostics.
//   - Ignore: which non-fatal diagnostics are suppressed entirely.
//   - FailAt: which stored severity causes [Load] to return [ErrDiagnosticThreshold].
//
// This is separate from [WithResolverStrictness], which controls how the
// resolver handles ambiguous or missing references. A permissive resolver
// still collects diagnostics, and those diagnostics are still subject to
// the FailAt threshold. Callers loading large vendor MIB sets typically
// want both permissive resolution and a relaxed FailAt (e.g. SeverityFatal).
func WithDiagnosticConfig(cfg mib.DiagnosticConfig) LoadOption {
	return func(c *loadConfig) { c.diagConfig = cfg }
}

// WithResolverStrictness sets resolver fallback behavior for ambiguous or
// missing references.
//
// This controls which heuristic fallbacks the resolver may use:
//   - [mib.ResolverStrict]: tier-1 only, no fallbacks. Unresolved references
//     are left unresolved.
//   - [mib.ResolverNormal]: tier-1 + tier-2 constrained fallbacks (default).
//   - [mib.ResolverPermissive]: tier-1 + tier-2 + tier-3 global fallbacks.
//     Best effort for vendor MIBs with spec violations.
//
// This does not affect diagnostic failure policy. A permissive resolver
// still produces diagnostics that may exceed the [mib.DiagnosticConfig.FailAt]
// threshold. To also suppress failure on non-fatal diagnostics, configure
// [WithDiagnosticConfig] with FailAt set to [mib.SeverityFatal].
func WithResolverStrictness(level mib.ResolverStrictness) LoadOption {
	return func(c *loadConfig) { c.resolverStrictness = level }
}

// WithSource appends one or more MIB sources to the load configuration.
// Sources are searched in the order they are added.
func WithSource(src ...Source) LoadOption {
	return func(c *loadConfig) { c.sources = append(c.sources, src...) }
}

// WithModules restricts loading to the named modules and their dependencies.
// Omit to load all modules from the configured sources.
func WithModules(names ...string) LoadOption {
	return func(c *loadConfig) {
		c.modules = append(c.modules, names...)
		c.hasModules = true
	}
}

// Load loads MIB modules from configured sources and resolves them.
//
// Example:
//
//	m, err := gomib.Load(ctx,
//	    gomib.WithSource(gomib.MustDir("/usr/share/snmp/mibs")),
//	    gomib.WithModules("IF-MIB", "IP-MIB"),
//	)
//
//	m, err := gomib.Load(ctx, gomib.WithSystemPaths())
func Load(ctx context.Context, opts ...LoadOption) (*mib.Mib, error) {
	cfg := loadConfig{
		resolverStrictness: mib.ResolverNormal,
		diagConfig:         mib.DefaultConfig(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	sources := cfg.sources
	if cfg.systemPaths {
		sources = append(sources, discoverSystemSources(types.Logger{L: cfg.logger})...)
	}
	// Embedded source provides base modules when no local copy is found.
	sources = append(sources, embeddedSource{})

	if len(cfg.sources) == 0 && !cfg.systemPaths && !cfg.hasModules {
		return nil, ErrNoSources
	}

	if cfg.hasModules {
		return loadModulesByName(ctx, sources, cfg.modules, &cfg)
	}
	return loadAllModules(ctx, sources, &cfg)
}

// ScanModuleNames extracts module names from raw MIB file bytes by finding
// identifiers that precede "DEFINITIONS ::=". This is a lightweight scan,
// not a full parse. ASN.1 comments are skipped so that commented-out module
// headers are not indexed. Returns nil if no module headers are found.
func ScanModuleNames(content []byte) []string {
	return scanModuleNames(content)
}
