package gomib

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"maps"
	"runtime"
	"slices"
	"sync"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/parser"
	"github.com/golangsnmp/gomib/internal/types"
	"github.com/golangsnmp/gomib/mib"
)

func componentLogger(logger *slog.Logger, component string) *slog.Logger {
	if logger == nil {
		return nil
	}
	return logger.With(slog.String("component", component))
}

// loadAllModules loads all MIB files from sources in parallel.
func loadAllModules(ctx context.Context, sources []Source, cfg *loadConfig) (*mib.Mib, error) {
	log := types.Logger{L: cfg.logger}

	type sourceModule struct {
		source Source
		name   string
	}

	// Deduplicate module names across sources: first source wins, matching
	// the precedence used by findModule() and Multi.Find().
	seen := make(map[string]struct{})
	var allModules []sourceModule
	for _, src := range sources {
		names, err := src.ListModules()
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				allModules = append(allModules, sourceModule{source: src, name: name})
			}
		}
	}

	if len(allModules) == 0 {
		return mib.Resolve(nil, nil, nil), nil
	}

	log.Log(slog.LevelInfo, "parallel loading",
		slog.Int("modules", len(allModules)))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan *module.Module, len(allModules))

	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once
	sem := make(chan struct{}, runtime.NumCPU())

	for _, sm := range allModules {
		wg.Add(1)
		go func(sm sourceModule) {
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

			result, err := sm.source.Find(sm.name)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					log.Log(slog.LevelDebug, "module not found",
						slog.String("module", sm.name),
						slog.Any("error", err))
				} else {
					log.Log(slog.LevelWarn, "module read error",
						slog.String("module", sm.name),
						slog.Any("error", err))
					errOnce.Do(func() {
						firstErr = err
						cancel()
					})
				}
				return
			}

			mod := decodeModule(result.Content, result.Path, sm.name, cfg)
			if mod != nil {
				results <- mod
			}
		}(sm)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	modules := make(map[string]*module.Module)
	for mod := range results {
		if _, exists := modules[mod.Name]; !exists {
			modules[mod.Name] = mod
		}
	}

	if firstErr != nil {
		return nil, firstErr
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	mods := collectModules(modules)

	log.Log(slog.LevelInfo, "parallel loading complete",
		slog.Int("modules", len(mods)))

	m := mib.Resolve(mods, componentLogger(cfg.logger, "resolver"), &cfg.diagConfig)
	return m, checkLoadResult(m, cfg, nil)
}

func loadModulesByName(ctx context.Context, sources []Source, names []string, cfg *loadConfig) (*mib.Mib, error) {
	log := types.Logger{L: cfg.logger}

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

		result, err := findModule(sources, name)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			log.Log(slog.LevelDebug, "module not found",
				slog.String("module", name))
			return nil // skip missing modules
		}

		mod := decodeModule(result.Content, result.Path, name, cfg)
		if mod == nil {
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

	mods := collectModules(modules)

	m := mib.Resolve(mods, componentLogger(cfg.logger, "resolver"), &cfg.diagConfig)
	return m, checkLoadResult(m, cfg, names)
}

func findModule(sources []Source, name string) (FindResult, error) {
	return Multi(sources...).Find(name)
}

// collectModules adds missing base modules to the map, deduplicates,
// and returns the modules sorted by name.
func collectModules(modules map[string]*module.Module) []*module.Module {
	for _, name := range module.BaseModuleNames() {
		if _, ok := modules[name]; !ok {
			if base := module.GetBaseModule(name); base != nil {
				modules[name] = base
			}
		}
	}
	mods := dedup(slices.Collect(maps.Values(modules)))
	slices.SortFunc(mods, func(a, b *module.Module) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return mods
}

// decodeModule runs the heuristic/parse/lower pipeline on raw MIB content.
// Returns nil if the content doesn't look like a MIB.
func decodeModule(content []byte, sourcePath, name string, cfg *loadConfig) *module.Module {
	log := types.Logger{L: cfg.logger}

	if !looksLikeMIBContent(content) {
		log.Log(slog.LevelDebug, "content rejected by heuristic",
			slog.String("module", name))
		return nil
	}

	p := parser.New(content, componentLogger(cfg.logger, "parser"), cfg.diagConfig)
	ast := p.ParseModule()

	mod := module.Lower(ast, content, componentLogger(cfg.logger, "module"), cfg.diagConfig)
	if mod != nil {
		mod.SourcePath = sourcePath
	}
	return mod
}

var (
	sigDefinitions = []byte("DEFINITIONS")
	sigAssign      = []byte("::=")
)

const heuristicMaxProbeSize = 128 * 1024

func looksLikeMIBContent(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	probeLen := min(heuristicMaxProbeSize, len(content))
	probe := content[:probeLen]

	if bytes.IndexByte(probe, 0) >= 0 {
		return false
	}

	return bytes.Contains(probe, sigDefinitions) && bytes.Contains(probe, sigAssign)
}
