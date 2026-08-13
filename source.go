package gomib

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	posixpath "path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/golangsnmp/gomib/internal/lexer"
	"github.com/golangsnmp/gomib/internal/types"
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
	index map[string][]string
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
	var result FindResult
	found, err := s.visitCandidates(name, func(candidate FindResult) bool {
		result = candidate
		return true
	})
	if err != nil {
		return result, err
	}
	if !found {
		return FindResult{}, fs.ErrNotExist
	}
	return result, nil
}

func (s *fsSource) visitCandidates(name string, visit func(FindResult) bool) (bool, error) {
	s.once.Do(func() {
		s.index, s.err = s.buildIndex()
	})
	if s.err != nil {
		return false, s.err
	}

	for _, path := range s.index[name] {
		fullPath := posixpath.Join(s.name, path)
		content, err := fs.ReadFile(s.fsys, path)
		if err != nil {
			return false, err
		}
		// Revalidate indexed candidates because their files may have changed
		// since lazy indexing (or eager Dir construction).
		if slices.Contains(scanModuleNames(content), name) && visit(FindResult{Content: content, Path: fullPath}) {
			return true, nil
		}
	}
	return false, nil
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

func (s *fsSource) buildIndex() (map[string][]string, error) {
	idx, err := buildTreeIndex(s.config.extensions, func(fn fs.WalkDirFunc) error {
		return fs.WalkDir(s.fsys, ".", fn)
	}, func(path string) ([]byte, error) {
		return fs.ReadFile(s.fsys, path)
	})
	if err != nil {
		return nil, fmt.Errorf("indexing %s: %w", s.name, err)
	}
	return idx, nil
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
	var result FindResult
	found, err := visitSourceCandidates(s, name, "", func(candidate FindResult, _ string) bool {
		result = candidate
		return true
	})
	if err != nil {
		return result, err
	}
	if !found {
		return FindResult{}, fs.ErrNotExist
	}
	return result, nil
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
// token sequences that form module headers. This is a lightweight lexical
// scan, not a full parse. Comments and quoted strings cannot advertise modules.
// Returns nil if no module headers are found.
func scanModuleNames(content []byte) []string {
	l := lexer.New(content, nil, types.DiagnosticConfig{})
	tokens, _ := l.Tokenize()

	var names []string
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Kind != lexer.TokUppercaseIdent {
			continue
		}

		nameToken := tokens[i]
		j := nextScanToken(tokens, i+1)

		// Some old ASN.1 modules include an obsolete module OID between
		// the module name and DEFINITIONS.
		if j < len(tokens) && tokens[j].Kind == lexer.TokLBrace {
			depth := 0
			for j < len(tokens) {
				switch tokens[j].Kind {
				case lexer.TokLBrace:
					depth++
				case lexer.TokRBrace:
					depth--
				}
				j++
				if depth == 0 {
					break
				}
			}
			j = nextScanToken(tokens, j)
		}

		if j >= len(tokens) || tokens[j].Kind != lexer.TokKwDefinitions {
			continue
		}
		j = nextScanToken(tokens, j+1)

		// Accept the ASN.1 tag-default form supported by the previous
		// scanner, while requiring each word to be a complete token.
		if j+1 < len(tokens) &&
			(string(content[tokens[j].Span.Start:tokens[j].Span.End]) == "IMPLICIT" ||
				string(content[tokens[j].Span.Start:tokens[j].Span.End]) == "EXPLICIT") &&
			string(content[tokens[j+1].Span.Start:tokens[j+1].Span.End]) == "TAGS" {
			j = nextScanToken(tokens, j+2)
		}

		if j >= len(tokens) || tokens[j].Kind != lexer.TokColonColonEqual {
			continue
		}

		names = append(names, string(content[nameToken.Span.Start:nameToken.Span.End]))
	}
	return names
}

func nextScanToken(tokens []lexer.Token, i int) int {
	for i < len(tokens) && tokens[i].Kind == lexer.TokComment {
		i++
	}
	return i
}

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
	index := make(map[string][]fileEntry)
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
			index[name] = append(index[name], fileEntry{path: path, content: content})
		}
	}
	return &fileSource{index: index}, nil
}

type fileEntry struct {
	path    string
	content []byte
}

type fileSource struct {
	index map[string][]fileEntry
}

func (s *fileSource) Find(name string) (FindResult, error) {
	entries := s.index[name]
	if len(entries) == 0 {
		return FindResult{}, fs.ErrNotExist
	}
	return FindResult{Content: entries[0].content, Path: entries[0].path}, nil
}

func (s *fileSource) visitCandidates(name string, visit func(FindResult) bool) bool {
	for _, entry := range s.index[name] {
		if visit(FindResult{Content: entry.content, Path: entry.path}) {
			return true
		}
	}
	return false
}

func (s *fileSource) ListModules() ([]string, error) {
	return slices.Sorted(maps.Keys(s.index)), nil
}

// visitSourceCandidates exposes every candidate from built-in aggregate sources
// to the loader without changing the public Source API. The namespace identifies
// a source position so equal diagnostic paths from different sources do not share
// decoded-content cache entries.
func visitSourceCandidates(src Source, name, namespace string, visit func(FindResult, string) bool) (bool, error) {
	switch s := src.(type) {
	case *fsSource:
		return s.visitCandidates(name, func(result FindResult) bool {
			return visit(result, namespace)
		})
	case *fileSource:
		return s.visitCandidates(name, func(result FindResult) bool {
			return visit(result, namespace)
		}), nil
	case *multiSource:
		for i, child := range s.sources {
			childNamespace := namespace + "/" + strconv.Itoa(i)
			found, err := visitSourceCandidates(child, name, childNamespace, visit)
			if err != nil {
				return false, err
			}
			if found {
				return true, nil
			}
		}
		return false, nil
	default:
		result, err := src.Find(name)
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return visit(result, namespace), nil
	}
}

// buildTreeIndex walks a file tree and builds a module name -> candidate paths index.
func buildTreeIndex(extensions []string, walkFn func(fs.WalkDirFunc) error, readFn func(path string) ([]byte, error)) (map[string][]string, error) {
	extSet := makeExtensionSet(extensions)
	index := make(map[string][]string)

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

		content, err := readFn(path)
		if err != nil {
			return nil //nolint:nilerr // skip unreadable files without aborting the walk
		}

		names := scanModuleNames(content)
		for _, name := range names {
			index[name] = append(index[name], path)
		}
		return nil
	})
	return index, err
}
