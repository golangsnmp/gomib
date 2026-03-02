// Package cliutil provides shared CLI utilities for gomib command-line tools.
package cliutil

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	gomib "github.com/golangsnmp/gomib"
)

// CGOFlags holds the common flags shared by CGO cross-validation tools.
type CGOFlags struct {
	Paths      []string
	OutputFile string
	JSONOutput bool
	HelpFlag   bool
}

// ParseCGOArgs parses global flags and extracts the subcommand from args.
// Flags handled: -p/--path, -o/--output, -json, -h/--help.
// Unrecognized flags are passed through to the subcommand.
// Returns a non-empty error string if a flag is missing its required value.
func ParseCGOArgs(args []string) (flags CGOFlags, cmd string, cmdArgs []string, errMsg string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			flags.HelpFlag = true
		case arg == "-json":
			flags.JSONOutput = true
		case arg == "-p" || arg == "--path":
			if i+1 >= len(args) {
				errMsg = arg + " requires a value"
				return flags, cmd, cmdArgs, errMsg
			}
			i++
			flags.Paths = append(flags.Paths, args[i])
		case strings.HasPrefix(arg, "-p"):
			flags.Paths = append(flags.Paths, arg[2:])
		case strings.HasPrefix(arg, "--path="):
			flags.Paths = append(flags.Paths, arg[7:])
		case arg == "-o" || arg == "--output":
			if i+1 >= len(args) {
				errMsg = arg + " requires a value"
				return flags, cmd, cmdArgs, errMsg
			}
			i++
			flags.OutputFile = args[i]
		case strings.HasPrefix(arg, "-o"):
			flags.OutputFile = arg[2:]
		case strings.HasPrefix(arg, "--output="):
			flags.OutputFile = arg[9:]
		case arg != "" && arg[0] == '-':
			cmdArgs = append(cmdArgs, arg)
		default:
			if cmd == "" {
				cmd = arg
			} else {
				cmdArgs = append(cmdArgs, arg)
			}
		}
	}
	return flags, cmd, cmdArgs, errMsg
}

// GetOutput opens the output file or returns stdout.
func GetOutput(outputFile string) (*os.File, func(), error) {
	if outputFile == "" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(outputFile)
	if err != nil {
		return nil, func() {}, err
	}
	return f, func() { _ = f.Close() }, nil
}

// PrintError writes a formatted error message to stderr.
func PrintError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}

// BuildSources creates gomib.Source values from directory paths using Dir.
// Invalid paths are skipped with a warning to stderr. Returns only the
// successfully created sources.
func BuildSources(paths []string) []gomib.Source {
	var sources []gomib.Source
	for _, p := range paths {
		src, err := gomib.Dir(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping path %s: %v\n", p, err)
			continue
		}
		sources = append(sources, src)
	}
	return sources
}

// ExpandDirs recursively discovers all directories under the given roots.
// Roots that don't exist or aren't directories are silently skipped.
func ExpandDirs(roots []string) []string {
	var dirs []string
	seen := make(map[string]bool)

	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}

		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil //nolint:nilerr // skip inaccessible entries
			}
			if d.IsDir() && !seen[path] {
				seen[path] = true
				dirs = append(dirs, path)
			}
			return nil
		})
	}

	return dirs
}
