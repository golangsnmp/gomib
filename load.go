package gomib

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/golangsnmp/gomib/internal/cst/lower"
	cstparser "github.com/golangsnmp/gomib/internal/cst/parser"
	"github.com/golangsnmp/gomib/internal/module"
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

	type sourceCandidate struct {
		source Source
		index  int
	}
	type sourceModule struct {
		name       string
		candidates []sourceCandidate
	}

	// Keep every source advertising a name until its content has been decoded.
	// Precedence is committed only when a candidate actually contains the module.
	moduleIndex := make(map[string]int)
	var allModules []sourceModule
	for sourceIndex, src := range sources {
		names, err := src.ListModules()
		if err != nil {
			return nil, fmt.Errorf("listing modules: %w", err)
		}
		seenInSource := make(map[string]struct{})
		for _, name := range names {
			if _, ok := seenInSource[name]; ok {
				continue
			}
			seenInSource[name] = struct{}{}

			candidate := sourceCandidate{source: src, index: sourceIndex}
			if i, ok := moduleIndex[name]; ok {
				allModules[i].candidates = append(allModules[i].candidates, candidate)
				continue
			}
			moduleIndex[name] = len(allModules)
			allModules = append(allModules, sourceModule{name: name, candidates: []sourceCandidate{candidate}})
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

	// Cache decoded files by source and path so multi-module files are only
	// parsed once without conflating identical diagnostic paths from different
	// sources.
	type cacheKey struct {
		source string
		path   string
	}
	type cachedDecode struct {
		once sync.Once
		mods []*module.Module
	}
	var pathCache sync.Map // cacheKey -> *cachedDecode

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

			for _, candidate := range sm.candidates {
				found, err := visitSourceCandidates(candidate.source, sm.name, strconv.Itoa(candidate.index), func(result FindResult, sourceID string) bool {
					key := cacheKey{source: sourceID, path: result.Path}
					entry, _ := pathCache.LoadOrStore(key, &cachedDecode{})
					cd := entry.(*cachedDecode)
					cd.once.Do(func() {
						cd.mods = decodeModules(result.Content, result.Path, cfg)
					})
					for _, mod := range cd.mods {
						if mod.Name == sm.name {
							results <- mod
							return true
						}
					}
					log.Log(slog.LevelDebug, "module not found in decoded file",
						slog.String("module", sm.name),
						slog.String("path", result.Path))
					return false
				})
				if err != nil {
					log.Log(slog.LevelWarn, "module read error",
						slog.String("module", sm.name),
						slog.Any("error", err))
					errOnce.Do(func() {
						firstErr = err
						cancel()
					})
					return
				}
				if found {
					return
				}
				log.Log(slog.LevelDebug, "module not found",
					slog.String("module", sm.name))
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

	// Cache decoded files by source and path so multi-module files are only
	// parsed once without conflating identical paths from different sources.
	type cacheKey struct {
		source string
		path   string
	}
	fileCache := make(map[cacheKey][]*module.Module)

	var loadOne func(name string) error
	loadOne = func(name string) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if _, ok := modules[name]; ok {
			return nil
		}

		// A Source.Find result is only a candidate until decoding confirms that
		// its content contains the requested module. Phantom advertisements do
		// not shadow later files or sources.
		var target *module.Module
		for sourceIndex, source := range sources {
			found, err := visitSourceCandidates(source, name, strconv.Itoa(sourceIndex), func(result FindResult, sourceID string) bool {
				key := cacheKey{source: sourceID, path: result.Path}
				mods, ok := fileCache[key]
				if !ok {
					mods = decodeModules(result.Content, result.Path, cfg)
					fileCache[key] = mods
				}
				for _, mod := range mods {
					if mod.Name == name {
						target = mod
						return true
					}
				}
				log.Log(slog.LevelDebug, "module not found in decoded file",
					slog.String("module", name),
					slog.String("path", result.Path))
				return false
			})
			if err != nil {
				return err
			}
			if found {
				break
			}
		}
		if target == nil {
			log.Log(slog.LevelDebug, "module not found",
				slog.String("module", name))
			return nil // skip missing modules
		}

		// Store only the requested module. Sibling modules from the same file
		// are loaded through Find so source precedence remains per-module.
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

	// Ensure base modules are loaded for the resolver.
	for _, name := range module.BaseModuleNames() {
		if err := loadOne(name); err != nil {
			return nil, err
		}
	}

	mods := collectModules(modules)

	m := mib.Resolve(mods, componentLogger(cfg.logger, "resolver"), &cfg.resolverStrictness, &cfg.diagConfig)
	return m, checkLoadResult(m, cfg, names)
}

// collectModules returns the modules sorted by name.
func collectModules(modules map[string]*module.Module) []*module.Module {
	mods := slices.Collect(maps.Values(modules))
	slices.SortFunc(mods, func(a, b *module.Module) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return mods
}

// decodeModules runs the heuristic/parse/validate pipeline on raw MIB content.
// A single file may contain multiple modules. Returns nil if the content
// doesn't look like a MIB.
func decodeModules(content []byte, sourcePath string, cfg *loadConfig) []*module.Module {
	log := types.Logger{L: cfg.logger}

	if !looksLikeMIBContent(content) {
		log.Log(slog.LevelDebug, "content rejected by heuristic",
			slog.String("path", sourcePath))
		return nil
	}

	p := cstparser.New(content, componentLogger(cfg.logger, "parser"), cfg.diagConfig)
	file := p.ParseModule()
	mods := lower.Lower(file, content, p.Diagnostics(), cfg.diagConfig)

	for _, mod := range mods {
		mod.SourcePath = sourcePath
		module.ValidateModule(mod, content, componentLogger(cfg.logger, "module"), cfg.diagConfig)
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

// checkLoadResult checks the resolved Mib for diagnostic threshold violations
// and missing requested modules. Returns nil if no issues found.
func checkLoadResult(m *mib.Mib, cfg *loadConfig, requestedModules []string) error {
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

	// Check stored effective severities against the independent FailAt threshold.
	for _, d := range m.Diagnostics() {
		if cfg.diagConfig.ShouldFail(d.Severity) {
			errs = append(errs, fmt.Errorf("%w: %s", ErrDiagnosticThreshold, d))
			break
		}
	}

	return errors.Join(errs...)
}
