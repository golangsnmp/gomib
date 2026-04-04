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
	// the precedence used by Multi.Find().
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
		return mib.Resolve(nil, nil, nil, nil), nil
	}

	log.Log(slog.LevelInfo, "parallel loading",
		slog.Int("modules", len(allModules)))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan *module.Module, len(allModules))

	// Cache decoded files by path so multi-module files are only
	// parsed once even when multiple goroutines request different
	// module names from the same file.
	type cachedDecode struct {
		once sync.Once
		mods []*module.Module
	}
	var pathCache sync.Map // result.Path -> *cachedDecode

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

			entry, _ := pathCache.LoadOrStore(result.Path, &cachedDecode{})
			cd := entry.(*cachedDecode)
			cd.once.Do(func() {
				cd.mods = decodeModules(result.Content, result.Path, cfg)
			})
			// Send only this goroutine's module to avoid N^2 sends
			// for multi-module files. The consumer deduplicates by name.
			found := false
			for _, mod := range cd.mods {
				if mod.Name == sm.name {
					results <- mod
					found = true
					break
				}
			}
			if !found {
				slog.Debug("module not found in decoded file",
					slog.String("module", sm.name),
					slog.String("path", result.Path))
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

	m := mib.Resolve(mods, componentLogger(cfg.logger, "resolver"), &cfg.resolverStrictness, &cfg.diagConfig)
	return m, checkLoadResult(m, cfg, nil)
}

func loadModulesByName(ctx context.Context, sources []Source, names []string, cfg *loadConfig) (*mib.Mib, error) {
	log := types.Logger{L: cfg.logger}

	modules := make(map[string]*module.Module)
	combined := Multi(sources...)

	// Cache decoded files by path so multi-module files are only
	// parsed once. Sibling modules are found through Find (which
	// respects source precedence) rather than eagerly cached.
	fileCache := make(map[string][]*module.Module) // path -> decoded modules

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

		result, err := combined.Find(name)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			log.Log(slog.LevelDebug, "module not found",
				slog.String("module", name))
			return nil // skip missing modules
		}

		mods, ok := fileCache[result.Path]
		if !ok {
			mods = decodeModules(result.Content, result.Path, cfg)
			fileCache[result.Path] = mods
		}
		if len(mods) == 0 {
			return nil
		}

		// Store only the requested module. Sibling modules from the
		// same file are loaded through Find when needed, so source
		// precedence is respected per-module.
		var target *module.Module
		for _, mod := range mods {
			if mod.Name == name {
				target = mod
				break
			}
		}
		if target == nil {
			return nil
		}
		modules[name] = target

		for _, imp := range target.Imports {
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

	m := mib.Resolve(mods, componentLogger(cfg.logger, "resolver"), &cfg.resolverStrictness, &cfg.diagConfig)
	return m, checkLoadResult(m, cfg, names)
}

// collectModules adds missing base modules to the map and returns the
// modules sorted by name.
func collectModules(modules map[string]*module.Module) []*module.Module {
	for _, name := range module.BaseModuleNames() {
		if _, ok := modules[name]; !ok {
			if base := module.GetBaseModule(name); base != nil {
				modules[name] = base
			}
		}
	}
	mods := slices.Collect(maps.Values(modules))
	slices.SortFunc(mods, func(a, b *module.Module) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return mods
}

// decodeModules runs the heuristic/parse/lower pipeline on raw MIB content.
// A single file may contain multiple modules. Returns nil if the content
// doesn't look like a MIB.
func decodeModules(content []byte, sourcePath string, cfg *loadConfig) []*module.Module {
	log := types.Logger{L: cfg.logger}

	if !looksLikeMIBContent(content) {
		log.Log(slog.LevelDebug, "content rejected by heuristic",
			slog.String("path", sourcePath))
		return nil
	}

	p := parser.New(content, componentLogger(cfg.logger, "parser"), cfg.diagConfig)
	astModules := p.ParseModule()

	var mods []*module.Module
	for _, am := range astModules {
		mod := module.Lower(am, content, componentLogger(cfg.logger, "module"), cfg.diagConfig)
		if mod != nil {
			mod.SourcePath = sourcePath
			mods = append(mods, mod)
		}
	}
	return mods
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
