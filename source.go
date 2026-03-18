package gomib

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	posixpath "path"
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
	Content []byte
	// Path is used in diagnostic messages to identify the source.
	Path string
}

// Source provides access to MIB files for the loading pipeline.
// Implementations are passed to [WithSource] and searched in order
// during [Load] to locate module files by name. The standard
// implementations are [Dir], [FS], and [Multi].
type Source interface {
	// Find returns the MIB content for the named module,
	// or fs.ErrNotExist if the module is not available.
	Find(name string) (FindResult, error)

	// ListModules returns all module names known to this source.
	ListModules() ([]string, error)
}

// SourceOption modifies source behavior.
type SourceOption func(*sourceConfig)

type sourceConfig struct {
	extensions []string
}

func defaultSourceConfig() sourceConfig {
	return sourceConfig{
		extensions: DefaultExtensions(),
	}
}

// WithExtensions overrides the default file extensions used to match MIB files.
// Extensions are normalized to lowercase with a leading dot (e.g. "mib"
// becomes ".mib"). An empty string matches files with no extension.
func WithExtensions(exts ...string) SourceOption {
	return func(c *sourceConfig) {
		c.extensions = make([]string, len(exts))
		for i, ext := range exts {
			ext = strings.ToLower(ext)
			if ext != "" && !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			c.extensions[i] = ext
		}
	}
}

// validateDir checks that path exists and is a directory.
func validateDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return &os.PathError{Op: "open", Path: path, Err: os.ErrInvalid}
	}
	return nil
}

// Dir creates a Source that recursively indexes a directory tree.
// It walks the tree once at construction and builds a name->path index
// using content-derived module names (not filenames).
// First match wins for duplicate names.
func Dir(root string, opts ...SourceOption) (Source, error) {
	if err := validateDir(root); err != nil {
		return nil, err
	}
	src := FS(root, os.DirFS(root), opts...)
	// Trigger eager indexing so walk errors are returned at construction time.
	if _, err := src.ListModules(); err != nil {
		return nil, err
	}
	return src, nil
}

// MustDir is like Dir but panics on error.
func MustDir(root string, opts ...SourceOption) Source {
	src, err := Dir(root, opts...)
	if err != nil {
		panic(err)
	}
	return src
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
// The name is used as a prefix in diagnostic paths. The filesystem
// is lazily indexed on first use. Errors are deferred to the first
// Find or ListModules call.
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
	fullPath := posixpath.Join(s.name, path)
	content, err := fs.ReadFile(s.fsys, path)
	if err != nil {
		return FindResult{Path: fullPath}, err
	}
	return FindResult{Content: content, Path: fullPath}, nil
}

func (s *fsSource) ListModules() ([]string, error) {
	s.once.Do(func() {
		s.index, s.err = s.buildIndex()
	})
	if s.err != nil {
		return nil, s.err
	}
	return slices.Sorted(maps.Keys(s.index)), nil
}

func (s *fsSource) buildIndex() (map[string]string, error) {
	return buildTreeIndex(s.config.extensions, func(fn fs.WalkDirFunc) error {
		return fs.WalkDir(s.fsys, ".", fn)
	}, func(path string) ([]byte, error) {
		return fs.ReadFile(s.fsys, path)
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
	var names []string
	for _, src := range s.sources {
		n, err := src.ListModules()
		if err != nil {
			return nil, err
		}
		names = append(names, n...)
	}
	return dedup(names), nil
}

func makeExtensionSet(extensions []string) map[string]struct{} {
	set := make(map[string]struct{}, len(extensions))
	for _, ext := range extensions {
		set[ext] = struct{}{}
	}
	return set
}

func hasValidExtension(path string, extSet map[string]struct{}) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := extSet[ext]
	return ok
}

// scanModuleNames extracts module names from raw MIB file bytes by finding
// identifiers that precede "DEFINITIONS ::=". This is a lightweight scan,
// not a full parse. ASN.1 comments (-- to end of line or next --) are
// skipped so that commented-out module headers are not indexed.
// Returns nil if no module headers are found.
func scanModuleNames(content []byte) []string {
	var names []string
	rest := content
	for {
		idx := bytes.Index(rest, sigDefinitions)
		if idx < 0 {
			break
		}
		// Absolute offset of this DEFINITIONS in content.
		absOff := len(content) - len(rest) + idx

		// Check that DEFINITIONS is not inside an ASN.1 comment by
		// scanning from the start of the line, toggling on each --.
		if inLineComment(content, absOff) {
			rest = rest[idx+len(sigDefinitions):]
			continue
		}

		// Require ::= somewhere after DEFINITIONS (possibly with
		// intervening tag defaults like IMPLICIT TAGS).
		after := rest[idx+len(sigDefinitions):]
		window := after
		if len(window) > 100 {
			window = window[:100]
		}
		if !bytes.Contains(window, sigAssign) {
			rest = rest[idx+len(sigDefinitions):]
			continue
		}

		// Walk backwards from DEFINITIONS to find the identifier.
		// Skip whitespace and comment lines between identifier and DEFINITIONS.
		pos := idx - 1
		for pos >= 0 {
			if rest[pos] == ' ' || rest[pos] == '\t' || rest[pos] == '\r' || rest[pos] == '\n' {
				pos--
				continue
			}
			// If we landed inside an ASN.1 comment, skip to the start of
			// the line and continue (handles comment lines between the
			// module name and DEFINITIONS, e.g. Emacs mode-line comments).
			absPos := len(content) - len(rest) + pos
			if inLineComment(content, absPos) {
				for pos >= 0 && rest[pos] != '\n' {
					pos--
				}
				continue
			}
			break
		}
		if pos < 0 {
			rest = rest[idx+len(sigDefinitions):]
			continue
		}
		// Collect the identifier characters backwards.
		end := pos + 1
		for pos >= 0 && isIdentChar(rest[pos]) {
			pos--
		}
		start := pos + 1
		if start < end {
			name := string(rest[start:end])
			// Module names must start with an uppercase letter.
			if name[0] >= 'A' && name[0] <= 'Z' {
				names = append(names, name)
			}
		}
		rest = rest[idx+len(sigDefinitions):]
	}
	return names
}

// inLineComment reports whether the byte at pos in content is inside an
// ASN.1 comment. It scans from the start of the line containing pos,
// toggling on each "--" sequence.
func inLineComment(content []byte, pos int) bool {
	lineStart := pos
	for lineStart > 0 && content[lineStart-1] != '\n' {
		lineStart--
	}
	inComment := false
	i := lineStart
	for i < pos {
		if i+1 < len(content) && content[i] == '-' && content[i+1] == '-' {
			inComment = !inComment
			i += 2
			continue
		}
		i++
	}
	return inComment
}

// isIdentChar returns true for characters valid in SMI identifiers.
func isIdentChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '-' || b == '_'
}

// buildTreeIndex walks a file tree and builds a module name -> path index.
// Module names are derived from file content (scanning for DEFINITIONS headers),
// not from filenames. Multiple module names from one file each get their own
// index entry pointing to the same path. First match wins for duplicate names.
// The readFn parameter provides file content for scanning.
// File creates a Source from a single MIB file on disk.
// The module name is extracted from the file content by scanning for
// "DEFINITIONS ::=" headers, just like [Dir] does for directory trees.
// The caller does not need to know or provide the module name.
func File(path string) (Source, error) {
	return Files(path)
}

// Files creates a Source from multiple MIB files on disk.
// Module names are extracted from each file's content by scanning for
// "DEFINITIONS ::=" headers. When duplicate module names appear across
// files, the first file wins.
func Files(paths ...string) (Source, error) {
	index := make(map[string]fileEntry)
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		names := scanModuleNames(content)
		if len(names) == 0 {
			return nil, fmt.Errorf("no module definition found in %s", path)
		}
		for _, name := range names {
			if _, exists := index[name]; !exists {
				index[name] = fileEntry{path: path, content: content}
			}
		}
	}
	return &fileSource{index: index}, nil
}

type fileEntry struct {
	path    string
	content []byte
}

type fileSource struct {
	index map[string]fileEntry
}

func (s *fileSource) Find(name string) (FindResult, error) {
	entry, ok := s.index[name]
	if !ok {
		return FindResult{}, fs.ErrNotExist
	}
	return FindResult{Content: entry.content, Path: entry.path}, nil
}

func (s *fileSource) ListModules() ([]string, error) {
	return slices.Sorted(maps.Keys(s.index)), nil
}

// buildTreeIndex walks a file tree and builds a module name -> path index.
func buildTreeIndex(extensions []string, walkFn func(fs.WalkDirFunc) error, readFn func(path string) ([]byte, error)) (map[string]string, error) {
	extSet := makeExtensionSet(extensions)
	index := make(map[string]string)

	err := walkFn(func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Debug("buildTreeIndex: skipping entry", "path", path, "error", err)
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

		content, readErr := readFn(path)
		if readErr != nil {
			slog.Debug("buildTreeIndex: cannot read file", "path", path, "error", readErr)
			return nil
		}

		names := scanModuleNames(content)
		for _, name := range names {
			if _, exists := index[name]; !exists {
				index[name] = path
			}
		}
		return nil
	})
	return index, err
}
