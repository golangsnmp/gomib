package gomib

import (
	"context"
	"errors"
	"log/slog"

	"github.com/golangsnmp/gomib/mib"
)

// ErrNoSources is returned when Load is called with no sources.
var ErrNoSources = errors.New("no MIB sources provided")

// LevelTrace is a custom log level more verbose than Debug.
// Use for per-item iteration logging (tokens, OID nodes, imports).
// Enable with: &slog.HandlerOptions{Level: slog.Level(-8)}
const LevelTrace = slog.Level(-8)

// LoadOption configures Load and LoadModules.
type LoadOption func(*loadConfig)

type loadConfig struct {
	logger      *slog.Logger
	systemPaths bool
	diagConfig  mib.DiagnosticConfig
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

// Load loads all MIB modules from the given source and resolves them.
// Use Multi() to combine multiple sources.
//
// Example:
//
//	mib, err := gomib.Load(ctx,
//	    gomib.DirTree("/usr/share/snmp/mibs"),
//	    gomib.WithLogger(slog.Default()),
//	)
//
//	// Multiple sources:
//	mib, err := gomib.Load(ctx,
//	    gomib.Multi(gomib.DirTree("/usr/share/snmp/mibs"), gomib.Dir("./custom")),
//	)
func Load(ctx context.Context, source Source, opts ...LoadOption) (*Mib, error) {
	cfg := loadConfig{
		diagConfig: mib.DefaultConfig(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	var sources []Source
	if source != nil {
		sources = append(sources, source)
	}
	return loadFromSources(ctx, sources, nil, cfg)
}

// LoadModules loads specific MIB modules by name, along with their dependencies.
// Use Multi() to combine multiple sources.
//
// Example:
//
//	mib, err := gomib.LoadModules(ctx,
//	    []string{"IF-MIB", "IP-MIB"},
//	    gomib.DirTree("/usr/share/snmp/mibs"),
//	)
func LoadModules(ctx context.Context, names []string, source Source, opts ...LoadOption) (*Mib, error) {
	cfg := loadConfig{
		diagConfig: mib.DefaultConfig(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	var sources []Source
	if source != nil {
		sources = append(sources, source)
	}
	return loadFromSources(ctx, sources, names, cfg)
}

// loadFromSources loads all modules if names is nil, or only named
// modules (plus dependencies) if names is non-nil.
func loadFromSources(ctx context.Context, sources []Source, names []string, cfg loadConfig) (*Mib, error) {
	if cfg.systemPaths {
		sources = append(sources, discoverSystemSources()...)
	}
	if len(sources) == 0 {
		return nil, ErrNoSources
	}

	if names != nil {
		return loadModulesByName(ctx, sources, names, cfg)
	}
	return loadAllModules(ctx, sources, cfg)
}

func logEnabled(logger *slog.Logger, level slog.Level) bool {
	return logger != nil && logger.Enabled(context.Background(), level)
}
