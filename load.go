package gomib

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"runtime"
	"sync"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/parser"
	"github.com/golangsnmp/gomib/internal/resolver"
	"github.com/golangsnmp/gomib/mib"
)

// componentLogger returns a logger with the component attribute, or nil if logger is nil.
func componentLogger(logger *slog.Logger, component string) *slog.Logger {
	if logger == nil {
		return nil
	}
	return logger.With(slog.String("component", component))
}

// loadAllModules loads all MIB files from sources in parallel.
func loadAllModules(ctx context.Context, sources []Source, cfg loadConfig) (*Mib, error) {
	if len(sources) == 0 {
		return nil, ErrNoSources
	}

	logger := cfg.logger

	// Collect all files from sources
	var allFiles []string
	for _, src := range sources {
		files, err := src.ListFiles()
		if err != nil {
			return nil, err
		}
		allFiles = append(allFiles, files...)
	}

	if len(allFiles) == 0 {
		return mib.NewMib(), nil
	}

	if logEnabled(logger, slog.LevelInfo) {
		logger.LogAttrs(ctx, slog.LevelInfo, "parallel loading",
			slog.Int("files", len(allFiles)))
	}

	// Parse in parallel with worker pool
	type parseResult struct {
		mod *module.Module
	}
	results := make(chan parseResult, len(allFiles))

	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())

	heuristic := defaultHeuristic()
	if cfg.noHeuristic {
		heuristic.enabled = false
	}

	for _, file := range allFiles {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			// Check for cancellation before acquiring semaphore
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			// Check for cancellation after acquiring semaphore
			if ctx.Err() != nil {
				return
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return
			}

			if !heuristic.looksLikeMIBContent(content) {
				return
			}

			p := parser.New(content, componentLogger(logger, "parser"))
			ast := p.ParseModule()
			if ast == nil {
				return
			}

			mod := module.Lower(ast, componentLogger(logger, "module"))
			if mod != nil {
				results <- parseResult{mod: mod}
			}
		}(file)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect parsed modules
	modules := make(map[string]*module.Module)
	for r := range results {
		if _, exists := modules[r.mod.Name]; !exists {
			modules[r.mod.Name] = r.mod
		}
	}

	// Check for cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Add base modules
	for _, name := range module.BaseModuleNames() {
		if _, ok := modules[name]; !ok {
			if base := module.GetBaseModule(name); base != nil {
				modules[name] = base
			}
		}
	}

	// Convert map to slice
	var mods []*module.Module
	for _, mod := range modules {
		mods = append(mods, mod)
	}

	if logEnabled(logger, slog.LevelInfo) {
		logger.LogAttrs(ctx, slog.LevelInfo, "parallel loading complete",
			slog.Int("modules", len(mods)))
	}

	// Resolve
	return resolver.Resolve(mods, componentLogger(logger, "resolver")), nil
}

// loadModulesByName loads specific modules by name along with their dependencies.
func loadModulesByName(ctx context.Context, sources []Source, names []string, cfg loadConfig) (*Mib, error) {
	logger := cfg.logger

	heuristic := defaultHeuristic()
	if cfg.noHeuristic {
		heuristic.enabled = false
	}

	modules := make(map[string]*module.Module)
	loading := make(map[string]struct{}) // cycle detection

	var loadOne func(name string) error
	loadOne = func(name string) error {
		// Check for cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Already loaded?
		if _, ok := modules[name]; ok {
			return nil
		}

		// Base module?
		if base := module.GetBaseModule(name); base != nil {
			modules[name] = base
			return nil
		}

		// Cycle detection
		if _, inProgress := loading[name]; inProgress {
			return nil // silently skip cycles
		}
		loading[name] = struct{}{}
		defer delete(loading, name)

		// Find the file
		content, err := findModuleContent(sources, name)
		if err != nil {
			if logEnabled(logger, slog.LevelDebug) {
				logger.LogAttrs(ctx, slog.LevelDebug, "module not found",
					slog.String("module", name))
			}
			return nil // skip missing modules
		}

		if !heuristic.looksLikeMIBContent(content) {
			if logEnabled(logger, slog.LevelDebug) {
				logger.LogAttrs(ctx, slog.LevelDebug, "content rejected by heuristic",
					slog.String("module", name))
			}
			return nil
		}

		// Parse
		p := parser.New(content, componentLogger(logger, "parser"))
		ast := p.ParseModule()
		if ast == nil {
			if logEnabled(logger, slog.LevelDebug) {
				logger.LogAttrs(ctx, slog.LevelDebug, "parse failed",
					slog.String("module", name))
			}
			return nil
		}

		// Lower
		mod := module.Lower(ast, componentLogger(logger, "module"))
		if mod == nil {
			if logEnabled(logger, slog.LevelDebug) {
				logger.LogAttrs(ctx, slog.LevelDebug, "lowering failed",
					slog.String("module", name))
			}
			return nil
		}

		modules[mod.Name] = mod
		if mod.Name != name {
			modules[name] = mod // also cache under requested name
		}

		// Load dependencies
		for _, imp := range mod.Imports {
			if err := loadOne(imp.Module); err != nil {
				return err
			}
		}

		return nil
	}

	// Load requested modules
	for _, name := range names {
		if err := loadOne(name); err != nil {
			return nil, err
		}
	}

	// Add base modules
	for _, name := range module.BaseModuleNames() {
		if _, ok := modules[name]; !ok {
			if base := module.GetBaseModule(name); base != nil {
				modules[name] = base
			}
		}
	}

	// Convert map to slice (deduplicate)
	seen := make(map[*module.Module]struct{})
	var mods []*module.Module
	for _, mod := range modules {
		if _, exists := seen[mod]; !exists {
			seen[mod] = struct{}{}
			mods = append(mods, mod)
		}
	}

	// Resolve
	return resolver.Resolve(mods, componentLogger(logger, "resolver")), nil
}

// findModuleContent searches sources for a module and returns its content.
func findModuleContent(sources []Source, name string) ([]byte, error) {
	for _, src := range sources {
		result, err := src.Find(name)
		if err == nil {
			content, err := io.ReadAll(result.Reader)
			_ = result.Reader.Close()
			if err == nil {
				return content, nil
			}
		}
	}
	return nil, fs.ErrNotExist
}

// --- Heuristic helpers ---

var (
	sigDefinitions = []byte("DEFINITIONS")
	sigAssign      = []byte("::=")
)

type heuristicConfig struct {
	enabled         bool
	binaryCheckSize int
	maxProbeSize    int
}

func defaultHeuristic() heuristicConfig {
	return heuristicConfig{
		enabled:         true,
		binaryCheckSize: 1024,
		maxProbeSize:    128 * 1024,
	}
}

func (h *heuristicConfig) looksLikeMIBContent(content []byte) bool {
	if !h.enabled {
		return true
	}
	if len(content) == 0 {
		return false
	}

	// Binary check on header
	checkLen := h.binaryCheckSize
	if checkLen > len(content) {
		checkLen = len(content)
	}
	for _, b := range content[:checkLen] {
		if b == 0 {
			return false
		}
	}

	// Probe for signatures
	probeLen := h.maxProbeSize
	if probeLen > len(content) {
		probeLen = len(content)
	}
	probe := content[:probeLen]

	// Reject if null byte found
	if bytes.IndexByte(probe, 0) >= 0 {
		return false
	}

	return bytes.Contains(probe, sigDefinitions) && bytes.Contains(probe, sigAssign)
}
