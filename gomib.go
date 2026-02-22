// Package gomib loads and resolves SNMP MIB modules.
//
// Call [Load] with one or more [Source] values to parse MIB files,
// resolve cross-module imports, build the OID tree, and return a
// read-only [mib.Mib] containing the merged result.
package gomib

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

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
const LevelTrace = slog.Level(-8)

// LoadOption configures Load.
type LoadOption func(*loadConfig)

type loadConfig struct {
	logger      *slog.Logger
	systemPaths bool
	diagConfig  mib.DiagnosticConfig
	sources     []Source
	modules     []string
	hasModules  bool // true when WithModules was called (even with empty list)
}

// WithLogger sets the logger for debug/trace output.
// If not set, no logging occurs (zero overhead).
func WithLogger(logger *slog.Logger) LoadOption {
	return func(c *loadConfig) { c.logger = logger }
}

// WithDiagnosticConfig sets the diagnostic configuration for strictness control.
// If not set, defaults to Normal strictness (report Minor and above, fail on Severe).
func WithDiagnosticConfig(cfg mib.DiagnosticConfig) LoadOption {
	return func(c *loadConfig) { c.diagConfig = cfg }
}

// WithStrictness sets the strictness level using a preset configuration.
// Convenience wrapper for WithDiagnosticConfig with preset configs.
func WithStrictness(level mib.StrictnessLevel) LoadOption {
	return func(c *loadConfig) {
		switch level {
		case mib.StrictnessStrict:
			c.diagConfig = mib.StrictConfig()
		case mib.StrictnessNormal:
			c.diagConfig = mib.DefaultConfig()
		case mib.StrictnessPermissive:
			c.diagConfig = mib.PermissiveConfig()
		case mib.StrictnessSilent:
			c.diagConfig = mib.DiagnosticConfig{
				Level:  mib.StrictnessSilent,
				FailAt: mib.SeverityFatal,
			}
		default:
			c.diagConfig = mib.DefaultConfig()
		}
	}
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
//	    gomib.WithSource(gomib.MustDirTree("/usr/share/snmp/mibs")),
//	    gomib.WithModules("IF-MIB", "IP-MIB"),
//	)
//
//	m, err := gomib.Load(ctx, gomib.WithSystemPaths())
func Load(ctx context.Context, opts ...LoadOption) (*mib.Mib, error) {
	cfg := loadConfig{
		diagConfig: mib.DefaultConfig(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	sources := cfg.sources
	if cfg.systemPaths {
		sources = append(sources, discoverSystemSources(types.Logger{L: cfg.logger})...)
	}
	if len(sources) == 0 {
		return nil, ErrNoSources
	}

	if cfg.hasModules {
		return loadModulesByName(ctx, sources, cfg.modules, cfg)
	}
	return loadAllModules(ctx, sources, cfg)
}

// checkLoadResult checks the resolved Mib for diagnostic threshold violations
// and missing requested modules. Returns nil if no issues found.
func checkLoadResult(m *mib.Mib, cfg loadConfig, requestedModules []string) error {
	var errs []error

	// Check for missing requested modules
	if len(requestedModules) > 0 {
		var missing []string
		for _, name := range requestedModules {
			if m.Module(name) == nil {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			errs = append(errs, fmt.Errorf("%w: %s", ErrMissingModules, strings.Join(missing, ", ")))
		}
	}

	// Check FailAt threshold
	for _, d := range m.Diagnostics() {
		if cfg.diagConfig.ShouldFail(d.Severity) {
			errs = append(errs, fmt.Errorf("%w: %s", ErrDiagnosticThreshold, d))
			break
		}
	}

	return errors.Join(errs...)
}

func logEnabled(logger *slog.Logger, level slog.Level) bool {
	return logger != nil && logger.Enabled(context.Background(), level)
}
