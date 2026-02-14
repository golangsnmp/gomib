package gomib

import (
	"errors"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// DefaultExtensions returns the file extensions recognized as MIB files.
// Empty string matches files with no extension (e.g., "IF-MIB").
func DefaultExtensions() []string {
	return []string{"", ".mib", ".smi", ".txt", ".my"}
}

// FindResult holds the content and location of a found MIB file.
type FindResult struct {
	Reader io.ReadCloser
	// Path is used in diagnostic messages to identify the source.
	Path string
}

// Source provides access to MIB files for loading.
type Source interface {
	// Find returns the MIB content for the named module,
	// or fs.ErrNotExist if the module is not available.
	Find(name string) (FindResult, error)

	// ListModules returns all module names known to this source.
	ListModules() ([]string, error)
}

// SourceOption modifies source behavior (extensions, heuristics).
type SourceOption func(*sourceConfig)

type sourceConfig struct {
	extensions  []string
	noHeuristic bool
}

func defaultSourceConfig() sourceConfig {
	return sourceConfig{
		extensions: DefaultExtensions(),
	}
}

// WithExtensions overrides the default file extensions used to match MIB files.
func WithExtensions(exts ...string) SourceOption {
	return func(c *sourceConfig) {
		c.extensions = exts
	}
}

// WithNoHeuristic disables the DEFINITIONS/::= content check,
// treating all matched files as MIB sources.
func WithNoHeuristic() SourceOption {
	return func(c *sourceConfig) {
		c.noHeuristic = true
	}
}

type dirSource struct {
	path   string
	config sourceConfig
}

// Dir creates a Source that searches a single directory (no recursion).
// Files are looked up lazily on each Find() call.
func Dir(path string, opts ...SourceOption) (Source, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrInvalid}
	}
	cfg := defaultSourceConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &dirSource{path: path, config: cfg}, nil
}

// MustDir is like Dir but panics on error.
func MustDir(path string, opts ...SourceOption) Source {
	src, err := Dir(path, opts...)
	if err != nil {
		panic(err)
	}
	return src
}

func (s *dirSource) Find(name string) (FindResult, error) {
	for _, ext := range s.config.extensions {
		fullPath := filepath.Join(s.path, name+ext)
		f, err := os.Open(fullPath)
		if err == nil {
			return FindResult{Reader: f, Path: fullPath}, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return FindResult{Path: fullPath}, err
		}
	}
	return FindResult{}, fs.ErrNotExist
}

func (s *dirSource) ListModules() ([]string, error) {
	extSet := makeExtensionSet(s.config.extensions)
	seen := make(map[string]struct{})
	var names []string

	entries, err := os.ReadDir(s.path)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if hasValidExtension(entry.Name(), extSet) {
			name := moduleNameFromPath(entry.Name())
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				names = append(names, name)
			}
		}
	}
	return names, nil
}

type treeSource struct {
	index  map[string]string // module name -> file path
	config sourceConfig
}

// DirTree creates a Source that recursively indexes a directory tree.
// It walks the tree once at construction and builds a name->path index.
// First match wins for duplicate names.
func DirTree(root string, opts ...SourceOption) (Source, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "open", Path: root, Err: os.ErrInvalid}
	}

	cfg := defaultSourceConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	index, err := buildTreeIndex(cfg.extensions, func(fn fs.WalkDirFunc) error {
		return filepath.WalkDir(root, fn)
	})
	if err != nil {
		return nil, err
	}

	return &treeSource{index: index, config: cfg}, nil
}

// MustDirTree is like DirTree but panics on error.
func MustDirTree(root string, opts ...SourceOption) Source {
	src, err := DirTree(root, opts...)
	if err != nil {
		panic(err)
	}
	return src
}

func (s *treeSource) Find(name string) (FindResult, error) {
	path, ok := s.index[name]
	if !ok {
		return FindResult{}, fs.ErrNotExist
	}
	f, err := os.Open(path)
	if err != nil {
		return FindResult{Path: path}, err
	}
	return FindResult{Reader: f, Path: path}, nil
}

func (s *treeSource) ListModules() ([]string, error) {
	return slices.Collect(maps.Keys(s.index)), nil
}

type fsSource struct {
	name   string
	fsys   fs.FS
	config sourceConfig

	once  sync.Once
	index map[string]string
	err   error
}

// FS creates a Source backed by an fs.FS (e.g., embed.FS).
// The name is used in diagnostic paths. The filesystem is lazily
// indexed on first use.
//
// Unlike Dir and DirTree, FS does not return an error at construction time.
// This is intentional: embed.FS cannot be walked until the program runs,
// so validation is deferred to the first Find or ListModules call.
func FS(name string, fsys fs.FS, opts ...SourceOption) Source {
	cfg := defaultSourceConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &fsSource{
		name:   name,
		fsys:   fsys,
		config: cfg,
	}
}

func (s *fsSource) Find(name string) (FindResult, error) {
	s.once.Do(func() {
		s.index, s.err = s.buildIndex()
	})
	if s.err != nil {
		return FindResult{}, s.err
	}

	path, ok := s.index[name]
	if !ok {
		return FindResult{}, fs.ErrNotExist
	}
	fullPath := s.name + ":" + path
	f, err := s.fsys.Open(path)
	if err != nil {
		return FindResult{Path: fullPath}, err
	}
	return FindResult{Reader: f, Path: fullPath}, nil
}

func (s *fsSource) ListModules() ([]string, error) {
	s.once.Do(func() {
		s.index, s.err = s.buildIndex()
	})
	if s.err != nil {
		return nil, s.err
	}
	return slices.Collect(maps.Keys(s.index)), nil
}

func (s *fsSource) buildIndex() (map[string]string, error) {
	return buildTreeIndex(s.config.extensions, func(fn fs.WalkDirFunc) error {
		return fs.WalkDir(s.fsys, ".", fn)
	})
}

type multiSource struct {
	sources []Source
}

// Multi combines multiple sources into one.
// Find() tries each source in order, returning the first match.
func Multi(sources ...Source) Source {
	return &multiSource{sources: sources}
}

func (s *multiSource) Find(name string) (FindResult, error) {
	for _, src := range s.sources {
		result, err := src.Find(name)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return result, err
		}
	}
	return FindResult{}, fs.ErrNotExist
}

func (s *multiSource) ListModules() ([]string, error) {
	seen := make(map[string]struct{})
	var names []string
	for _, src := range s.sources {
		n, err := src.ListModules()
		if err != nil {
			return nil, err
		}
		for _, name := range n {
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				names = append(names, name)
			}
		}
	}
	return names, nil
}

func makeExtensionSet(extensions []string) map[string]struct{} {
	set := make(map[string]struct{}, len(extensions))
	for _, ext := range extensions {
		set[strings.ToLower(ext)] = struct{}{}
	}
	return set
}

func hasValidExtension(path string, extSet map[string]struct{}) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := extSet[ext]
	return ok
}

func moduleNameFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// buildTreeIndex walks a file tree and builds a module name -> path index.
// First match wins for duplicate names.
func buildTreeIndex(extensions []string, walkFn func(fs.WalkDirFunc) error) (map[string]string, error) {
	extSet := makeExtensionSet(extensions)
	index := make(map[string]string)

	err := walkFn(func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !hasValidExtension(path, extSet) {
			return nil
		}

		name := moduleNameFromPath(path)
		if _, exists := index[name]; !exists {
			index[name] = path
		}
		return nil
	})
	return index, err
}
