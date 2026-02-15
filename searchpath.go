package gomib

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/golangsnmp/gomib/internal/types"
)

// WithSystemPaths enables automatic discovery of MIB search paths from
// net-snmp and libsmi configuration (config files, env vars, defaults).
// Discovered paths are appended after any explicit source, serving as fallback.
// When source is nil and WithSystemPaths is set, system paths alone are sufficient.
func WithSystemPaths() LoadOption {
	return func(c *loadConfig) { c.systemPaths = true }
}

type pathOp int

const (
	pathReplace pathOp = iota
	pathAppend
	pathPrepend
)

// discoverSystemSources returns Sources for all discovered system MIB directories.
func discoverSystemSources(logger types.Logger) []Source {
	dirs := discoverSystemPaths(logger)
	var sources []Source
	for _, d := range dirs {
		if src, err := Dir(d); err == nil {
			sources = append(sources, src)
		}
	}
	return sources
}

// discoverSystemPaths returns MIB directories from net-snmp and libsmi
// configuration, deduplicated and filtered to directories that exist.
func discoverSystemPaths(logger types.Logger) []string {
	var all []string
	all = append(all, discoverNetSNMPPaths(logger)...)
	all = append(all, discoverLibSMIPaths(logger)...)
	return filterExistingDirs(dedup(all))
}

func discoverNetSNMPPaths(logger types.Logger) []string {
	paths := netsnmpDefaults()
	for _, cf := range netsnmpConfigFiles() {
		paths = applyConfigFile(cf, paths, parseNetSNMPLine, logger)
	}
	if v := os.Getenv("MIBDIRS"); v != "" {
		paths = applyNetSNMPEnv(v, paths)
	}
	return paths
}

func discoverLibSMIPaths(logger types.Logger) []string {
	paths := libsmiDefaults()
	for _, cf := range libsmiConfigFiles() {
		paths = applyConfigFile(cf, paths, parseLibSMILine, logger)
	}
	if v := os.Getenv("SMIPATH"); v != "" {
		paths = applyLibSMIEnv(v, paths)
	}
	return paths
}

func netsnmpDefaults() []string {
	var paths []string
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".snmp", "mibs"))
	}
	paths = append(paths,
		"/usr/share/snmp/mibs",
		"/usr/share/snmp/mibs/iana",
		"/usr/share/snmp/mibs/ietf",
		"/usr/local/share/snmp/mibs",
	)
	return paths
}

func libsmiDefaults() []string {
	return []string{
		"/usr/share/mibs/ietf",
		"/usr/share/mibs/iana",
		"/usr/share/mibs/irtf",
		"/usr/share/mibs/site",
		"/usr/local/share/mibs/ietf",
		"/usr/local/share/mibs/iana",
		"/usr/local/share/mibs/irtf",
		"/usr/local/share/mibs/site",
	}
}

func netsnmpConfigFiles() []string {
	files := []string{"/etc/snmp/snmp.conf"}
	if home, err := os.UserHomeDir(); err == nil {
		files = append(files, filepath.Join(home, ".snmp", "snmp.conf"))
	}
	return files
}

func libsmiConfigFiles() []string {
	files := []string{"/etc/smi.conf"}
	if home, err := os.UserHomeDir(); err == nil {
		files = append(files, filepath.Join(home, ".smirc"))
	}
	return files
}

// parseNetSNMPLine parses a single snmp.conf line for mibdirs directives.
// Supports both "mibdirs +/path" (prefix on value) and "+mibdirs /path" (prefix on directive).
func parseNetSNMPLine(line string) (pathOp, []string, bool) {
	line = strings.TrimSpace(line)
	if line == "" || line[0] == '#' {
		return 0, nil, false
	}

	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, nil, false
	}

	directive := fields[0]
	value := fields[1]

	switch directive {
	case "mibdirs":
		// Value may start with + or -
		if strings.HasPrefix(value, "+") {
			return pathAppend, splitPaths(value[1:]), true
		}
		if strings.HasPrefix(value, "-") {
			return pathPrepend, splitPaths(value[1:]), true
		}
		return pathReplace, splitPaths(value), true
	case "+mibdirs":
		return pathAppend, splitPaths(value), true
	case "-mibdirs":
		return pathPrepend, splitPaths(value), true
	default:
		return 0, nil, false
	}
}

// parseLibSMILine parses a single smi.conf line for path directives.
// Lines with tag prefixes (e.g. "smilint: path ...") are skipped.
func parseLibSMILine(line string) (pathOp, []string, bool) {
	line = strings.TrimSpace(line)
	if line == "" || line[0] == '#' {
		return 0, nil, false
	}

	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, nil, false
	}

	// Skip tagged lines (e.g., "smilint: path ...")
	if strings.HasSuffix(fields[0], ":") {
		return 0, nil, false
	}

	if fields[0] != "path" {
		return 0, nil, false
	}

	op, dirs := parseColonSemantic(fields[1])
	return op, dirs, true
}

// parseColonSemantic interprets leading/trailing colon semantics.
// Leading colon = append, trailing colon = prepend, neither = replace.
func parseColonSemantic(value string) (pathOp, []string) {
	if strings.HasPrefix(value, ":") {
		return pathAppend, splitPaths(strings.TrimPrefix(value, ":"))
	}
	if strings.HasSuffix(value, ":") {
		return pathPrepend, splitPaths(strings.TrimSuffix(value, ":"))
	}
	return pathReplace, splitPaths(value)
}

func applyNetSNMPEnv(value string, current []string) []string {
	if strings.HasPrefix(value, "+") {
		return applyOp(pathAppend, splitPaths(value[1:]), current)
	}
	if strings.HasPrefix(value, "-") {
		return applyOp(pathPrepend, splitPaths(value[1:]), current)
	}
	return splitPaths(value)
}

func applyLibSMIEnv(value string, current []string) []string {
	op, dirs := parseColonSemantic(value)
	return applyOp(op, dirs, current)
}

func applyOp(op pathOp, dirs, current []string) []string {
	switch op {
	case pathAppend:
		return append(current, dirs...)
	case pathPrepend:
		return append(dirs, current...)
	default:
		return dirs
	}
}

func applyConfigFile(path string, current []string, parseLine func(string) (pathOp, []string, bool), logger types.Logger) []string {
	f, err := os.Open(path)
	if err != nil {
		return current
	}
	defer f.Close() //nolint:errcheck // best-effort config file read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		op, dirs, ok := parseLine(scanner.Text())
		if !ok {
			continue
		}
		current = applyOp(op, dirs, current)
	}
	if err := scanner.Err(); err != nil {
		logger.Log(slog.LevelDebug, "error reading config file", slog.String("path", path), slog.Any("error", err))
	}
	return current
}

func splitPaths(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, p := range strings.Split(s, ":") {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func dedup(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	var result []string
	for _, p := range paths {
		if _, ok := seen[p]; !ok {
			seen[p] = struct{}{}
			result = append(result, p)
		}
	}
	return result
}

func filterExistingDirs(paths []string) []string {
	var result []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err == nil && info.IsDir() {
			result = append(result, p)
		}
	}
	return result
}
