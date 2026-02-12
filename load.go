package gomib

import (
	"bytes"
	"cmp"
	"context"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"sync"

	"github.com/golangsnmp/gomib/internal/mibimpl"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/parser"
	"github.com/golangsnmp/gomib/internal/resolver"
)

func componentLogger(logger *slog.Logger, component string) *slog.Logger {
	if logger == nil {
		return nil
	}
	return logger.With(slog.String("component", component))
}

// loadAllModules loads all MIB files from sources in parallel.
func loadAllModules(ctx context.Context, sources []Source, cfg loadConfig) (Mib, error) {
	if len(sources) == 0 {
		return nil, ErrNoSources
	}

	logger := cfg.logger

	var allFiles []string
	for _, src := range sources {
		files, err := src.ListFiles()
		if err != nil {
			return nil, err
		}
		allFiles = append(allFiles, files...)
	}

	if len(allFiles) == 0 {
		return mibimpl.EmptyMib(), nil
	}

	if logEnabled(logger, slog.LevelInfo) {
		logger.LogAttrs(ctx, slog.LevelInfo, "parallel loading",
			slog.Int("files", len(allFiles)))
	}

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

			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

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

			p := parser.New(content, componentLogger(logger, "parser"), cfg.diagConfig)
			ast := p.ParseModule()
			if ast == nil {
				return
			}

			mod := module.Lower(ast, componentLogger(logger, "module"), cfg.diagConfig)
			if mod != nil {
				results <- parseResult{mod: mod}
			}
		}(file)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	modules := make(map[string]*module.Module)
	for r := range results {
		if _, exists := modules[r.mod.Name]; !exists {
			modules[r.mod.Name] = r.mod
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	for _, name := range module.BaseModuleNames() {
		if _, ok := modules[name]; !ok {
			if base := module.GetBaseModule(name); base != nil {
				modules[name] = base
			}
		}
	}

	var mods []*module.Module
	for _, mod := range modules {
		mods = append(mods, mod)
	}
	slices.SortFunc(mods, func(a, b *module.Module) int {
		return cmp.Compare(a.Name, b.Name)
	})

	if logEnabled(logger, slog.LevelInfo) {
		logger.LogAttrs(ctx, slog.LevelInfo, "parallel loading complete",
			slog.Int("modules", len(mods)))
	}

	return resolver.Resolve(mods, componentLogger(logger, "resolver"), &cfg.diagConfig), nil
}

func loadModulesByName(ctx context.Context, sources []Source, names []string, cfg loadConfig) (Mib, error) {
	logger := cfg.logger

	heuristic := defaultHeuristic()
	if cfg.noHeuristic {
		heuristic.enabled = false
	}

	modules := make(map[string]*module.Module)
	loading := make(map[string]struct{})

	var loadOne func(name string) error
	loadOne = func(name string) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if _, ok := modules[name]; ok {
			return nil
		}

		if base := module.GetBaseModule(name); base != nil {
			modules[name] = base
			return nil
		}

		if _, inProgress := loading[name]; inProgress {
			return nil
		}
		loading[name] = struct{}{}
		defer delete(loading, name)

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

		p := parser.New(content, componentLogger(logger, "parser"), cfg.diagConfig)
		ast := p.ParseModule()
		if ast == nil {
			if logEnabled(logger, slog.LevelDebug) {
				logger.LogAttrs(ctx, slog.LevelDebug, "parse failed",
					slog.String("module", name))
			}
			return nil
		}

		mod := module.Lower(ast, componentLogger(logger, "module"), cfg.diagConfig)
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

		for _, imp := range mod.Imports {
			if err := loadOne(imp.Module); err != nil {
				return err
			}
		}

		return nil
	}

	for _, name := range names {
		if err := loadOne(name); err != nil {
			return nil, err
		}
	}

	for _, name := range module.BaseModuleNames() {
		if _, ok := modules[name]; !ok {
			if base := module.GetBaseModule(name); base != nil {
				modules[name] = base
			}
		}
	}

	// Deduplicate since multiple names may map to the same module.
	seen := make(map[*module.Module]struct{})
	var mods []*module.Module
	for _, mod := range modules {
		if _, exists := seen[mod]; !exists {
			seen[mod] = struct{}{}
			mods = append(mods, mod)
		}
	}
	slices.SortFunc(mods, func(a, b *module.Module) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return resolver.Resolve(mods, componentLogger(logger, "resolver"), &cfg.diagConfig), nil
}

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

	checkLen := h.binaryCheckSize
	if checkLen > len(content) {
		checkLen = len(content)
	}
	for _, b := range content[:checkLen] {
		if b == 0 {
			return false
		}
	}

	probeLen := h.maxProbeSize
	if probeLen > len(content) {
		probeLen = len(content)
	}
	probe := content[:probeLen]

	if bytes.IndexByte(probe, 0) >= 0 {
		return false
	}

	return bytes.Contains(probe, sigDefinitions) && bytes.Contains(probe, sigAssign)
}
