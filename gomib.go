package gomib

import (
	"context"
	"log/slog"
)

// LevelTrace is a custom log level more verbose than Debug.
// Use for per-item iteration logging (tokens, OID nodes, imports).
// Enable with: &slog.HandlerOptions{Level: slog.Level(-8)}
const LevelTrace = slog.Level(-8)

// LoadOption configures Load and LoadModules.
type LoadOption interface {
	apply(*loadConfig)
}

type loadConfig struct {
	logger      *slog.Logger
	extensions  []string
	noHeuristic bool
}

type loggerOption struct{ logger *slog.Logger }

func (o loggerOption) apply(c *loadConfig) { c.logger = o.logger }

// WithLogger sets the logger for debug/trace output.
// If not set, no logging occurs (zero overhead).
func WithLogger(logger *slog.Logger) LoadOption {
	return loggerOption{logger: logger}
}

// Load loads all MIB modules from the given sources and resolves them.
// Sources are searched in order; the first match for each module name wins.
//
// Example:
//
//	mib, err := gomib.Load(
//	    gomib.DirTree("/usr/share/snmp/mibs"),
//	    gomib.WithLogger(slog.Default()),
//	)
func Load(args ...any) (*Mib, error) {
	var sources []Source
	var opts []LoadOption

	for _, arg := range args {
		switch v := arg.(type) {
		case Source:
			sources = append(sources, v)
		case LoadOption:
			opts = append(opts, v)
		}
	}

	cfg := loadConfig{
		extensions: DefaultExtensions,
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	return loadFromSources(sources, nil, cfg)
}

// LoadModules loads specific MIB modules by name, along with their dependencies.
//
// Example:
//
//	mib, err := gomib.LoadModules(
//	    []string{"IF-MIB", "IP-MIB"},
//	    gomib.DirTree("/usr/share/snmp/mibs"),
//	)
func LoadModules(names []string, args ...any) (*Mib, error) {
	var sources []Source
	var opts []LoadOption

	for _, arg := range args {
		switch v := arg.(type) {
		case Source:
			sources = append(sources, v)
		case LoadOption:
			opts = append(opts, v)
		}
	}

	cfg := loadConfig{
		extensions: DefaultExtensions,
	}
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	return loadFromSources(sources, names, cfg)
}

// loadFromSources is the internal implementation.
// If names is nil, loads all modules from sources.
// If names is non-nil, loads only those modules (plus dependencies).
func loadFromSources(sources []Source, names []string, cfg loadConfig) (*Mib, error) {
	if len(sources) == 0 {
		return NewMib(), nil
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

// --- Logging helpers ---

// ctx is a package-level context for logging.
var ctx = context.Background()

// logEnabled returns true if logging is enabled at the given level.
func logEnabled(logger *slog.Logger, level slog.Level) bool {
	return logger != nil && logger.Enabled(ctx, level)
}
