package gomib

import (
	"context"
	"errors"
	"log/slog"
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
	extensions  []string
	noHeuristic bool
}

// WithLogger sets the logger for debug/trace output.
// If not set, no logging occurs (zero overhead).
func WithLogger(logger *slog.Logger) LoadOption {
	return func(c *loadConfig) { c.logger = logger }
}

// Load loads all MIB modules from the given source and resolves them.
// Use Multi() to combine multiple sources.
//
// Example:
//
//	mib, err := gomib.Load(
//	    gomib.DirTree("/usr/share/snmp/mibs"),
//	    gomib.WithLogger(slog.Default()),
//	)
//
//	// Multiple sources:
//	mib, err := gomib.Load(
//	    gomib.Multi(gomib.DirTree("/usr/share/snmp/mibs"), gomib.Dir("./custom")),
//	)
func Load(source Source, opts ...LoadOption) (*Mib, error) {
	cfg := loadConfig{
		extensions: DefaultExtensions,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	var sources []Source
	if source != nil {
		sources = append(sources, source)
	}
	return loadFromSources(sources, nil, cfg)
}

// LoadModules loads specific MIB modules by name, along with their dependencies.
// Use Multi() to combine multiple sources.
//
// Example:
//
//	mib, err := gomib.LoadModules(
//	    []string{"IF-MIB", "IP-MIB"},
//	    gomib.DirTree("/usr/share/snmp/mibs"),
//	)
func LoadModules(names []string, source Source, opts ...LoadOption) (*Mib, error) {
	cfg := loadConfig{
		extensions: DefaultExtensions,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	var sources []Source
	if source != nil {
		sources = append(sources, source)
	}
	return loadFromSources(sources, names, cfg)
}

// loadFromSources is the internal implementation.
// If names is nil, loads all modules from sources.
// If names is non-nil, loads only those modules (plus dependencies).
func loadFromSources(sources []Source, names []string, cfg loadConfig) (*Mib, error) {
	if len(sources) == 0 {
		return nil, ErrNoSources
	}

	loadCfg := LoadConfig{
		Logger:      cfg.logger,
		NoHeuristic: cfg.noHeuristic,
	}

	if names != nil {
		return loadModulesByName(sources, names, loadCfg)
	}
	return loadAllModules(sources, loadCfg)
}

// logEnabled returns true if logging is enabled at the given level.
func logEnabled(logger *slog.Logger, level slog.Level) bool {
	return logger != nil && logger.Enabled(context.Background(), level)
}
