package gomib

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DefaultExtensions are the file extensions recognized as MIB files.
// Empty string matches files with no extension (e.g., "IF-MIB").
var DefaultExtensions = []string{"", ".mib", ".smi", ".txt", ".my"}

// Source finds MIB files by module name.
type Source interface {
	// Find locates a module by name.
	// Returns the file content, source path for diagnostics, or fs.ErrNotExist if not found.
	Find(name string) (io.ReadCloser, string, error)

	// ListFiles returns all MIB file paths known to this source.
	// Used for parallel loading.
	ListFiles() ([]string, error)
}

// SourceOption configures a source.
type SourceOption func(*sourceConfig)

type sourceConfig struct {
	extensions  []string
	noHeuristic bool
}

func defaultSourceConfig() sourceConfig {
	return sourceConfig{
		extensions: DefaultExtensions,
	}
}

// WithExtensions sets the file extensions to recognize for this source.
func WithExtensions(exts ...string) SourceOption {
	return func(c *sourceConfig) {
		c.extensions = exts
	}
}

// WithNoHeuristic disables content validation for this source.
func WithNoHeuristic() SourceOption {
	return func(c *sourceConfig) {
		c.noHeuristic = true
	}
}

// --- Dir Source (single directory, lazy) ---

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

func (s *dirSource) Find(name string) (io.ReadCloser, string, error) {
	for _, ext := range s.config.extensions {
		fullPath := filepath.Join(s.path, name+ext)
		f, err := os.Open(fullPath)
		if err == nil {
			return f, fullPath, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fullPath, err
		}
	}
	return nil, "", fs.ErrNotExist
}

func (s *dirSource) ListFiles() ([]string, error) {
	extSet := makeExtensionSet(s.config.extensions)
	var files []string

	entries, err := os.ReadDir(s.path)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(s.path, entry.Name())
		if hasValidExtension(path, extSet) {
			files = append(files, path)
		}
	}
	return files, nil
}

// --- DirTree Source (recursive directory, indexed) ---

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

	extSet := makeExtensionSet(cfg.extensions)
	index := make(map[string]string)

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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

func (s *treeSource) Find(name string) (io.ReadCloser, string, error) {
	path, ok := s.index[name]
	if !ok {
		return nil, "", fs.ErrNotExist
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, path, err
	}
	return f, path, nil
}

func (s *treeSource) ListFiles() ([]string, error) {
	files := make([]string, 0, len(s.index))
	for _, path := range s.index {
		files = append(files, path)
	}
	return files, nil
}

// --- FS Source (for embed.FS, testing, http filesystems) ---

type fsSource struct {
	name   string
	fsys   fs.FS
	config sourceConfig

	once  sync.Once
	index map[string]string
	err   error
}

// FS creates a Source backed by an fs.FS (e.g., embed.FS).
// The name is used for error messages and path reporting.
// It lazily indexes the filesystem on first Find() call.
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

func (s *fsSource) Find(name string) (io.ReadCloser, string, error) {
	s.once.Do(func() {
		s.index, s.err = s.buildIndex()
	})
	if s.err != nil {
		return nil, "", s.err
	}

	path, ok := s.index[name]
	if !ok {
		return nil, "", fs.ErrNotExist
	}
	f, err := s.fsys.Open(path)
	if err != nil {
		return nil, s.name + ":" + path, err
	}
	return f, s.name + ":" + path, nil
}

func (s *fsSource) ListFiles() ([]string, error) {
	s.once.Do(func() {
		s.index, s.err = s.buildIndex()
	})
	if s.err != nil {
		return nil, s.err
	}

	files := make([]string, 0, len(s.index))
	for _, path := range s.index {
		files = append(files, s.name+":"+path)
	}
	return files, nil
}

func (s *fsSource) buildIndex() (map[string]string, error) {
	extSet := makeExtensionSet(s.config.extensions)
	index := make(map[string]string)

	err := fs.WalkDir(s.fsys, ".", func(path string, d fs.DirEntry, err error) error {
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

// --- Multi Source (combines multiple sources) ---

type multiSource struct {
	sources []Source
}

// Multi combines multiple sources into one.
// Find() tries each source in order, returning the first match.
func Multi(sources ...Source) Source {
	return &multiSource{sources: sources}
}

func (s *multiSource) Find(name string) (io.ReadCloser, string, error) {
	for _, src := range s.sources {
		r, path, err := src.Find(name)
		if err == nil {
			return r, path, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, path, err
		}
	}
	return nil, "", fs.ErrNotExist
}

func (s *multiSource) ListFiles() ([]string, error) {
	var files []string
	for _, src := range s.sources {
		f, err := src.ListFiles()
		if err != nil {
			return nil, err
		}
		files = append(files, f...)
	}
	return files, nil
}

// --- Helpers ---

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
