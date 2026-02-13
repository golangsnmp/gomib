//go:build cgo

// Command gomib-libsmi compares gomib against libsmi for lexer/parser cross-validation.
// Build with: CGO_ENABLED=1 go build -tags cgo ./cmd/gomib-libsmi
//
//nolint:errcheck // CLI tool, fmt output errors are not critical
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/golangsnmp/gomib"
)

const usage = `gomib-libsmi - Compare gomib against libsmi (smilint)

Usage:
  gomib-libsmi <command> [options] [arguments]

Commands:
  diag       Compare parser diagnostics at different strictness levels
  accept     Batch test which MIBs pass/fail in each parser
  compare    Semantic comparison between gomib and libsmi

Common options:
  -p, --path PATH   Add MIB search path (repeatable)
  -o, --output FILE Write output to file instead of stdout
  -json             Output in JSON format
  -h, --help        Show help

Examples:
  gomib-libsmi diag -p ./testdata/corpus/primary -level 2 IF-MIB
  gomib-libsmi accept -p ./testdata/corpus/primary
  gomib-libsmi compare -p ./testdata/corpus/primary IF-MIB SNMPv2-MIB
`

var (
	paths      []string
	outputFile string
	jsonOutput bool
	helpFlag   bool
)

func main() {
	os.Exit(run())
}

func run() int {
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	args := os.Args[1:]
	var cmdArgs []string
	var cmd string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			helpFlag = true
		case arg == "-json":
			jsonOutput = true
		case arg == "-p" || arg == "--path":
			if i+1 < len(args) {
				i++
				paths = append(paths, args[i])
			}
		case strings.HasPrefix(arg, "-p"):
			paths = append(paths, arg[2:])
		case strings.HasPrefix(arg, "--path="):
			paths = append(paths, arg[7:])
		case arg == "-o" || arg == "--output":
			if i+1 < len(args) {
				i++
				outputFile = args[i]
			}
		case strings.HasPrefix(arg, "-o"):
			outputFile = arg[2:]
		case strings.HasPrefix(arg, "--output="):
			outputFile = arg[9:]
		case len(arg) > 0 && arg[0] == '-':
			cmdArgs = append(cmdArgs, arg)
		default:
			if cmd == "" {
				cmd = arg
			} else {
				cmdArgs = append(cmdArgs, arg)
			}
		}
	}

	if helpFlag && cmd == "" {
		fmt.Fprint(os.Stdout, usage)
		return 0
	}

	if cmd == "" {
		fmt.Fprint(os.Stderr, usage)
		return 1
	}

	switch cmd {
	case "diag":
		return cmdDiag(cmdArgs)
	case "accept":
		return cmdAccept(cmdArgs)
	case "compare":
		return cmdCompare(cmdArgs)
	case "help":
		fmt.Fprint(os.Stdout, usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		fmt.Fprint(os.Stderr, usage)
		return 1
	}
}

func getOutput() (*os.File, func(), error) {
	if outputFile == "" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(outputFile)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { _ = f.Close() }, nil
}

func getMIBPaths() []string {
	return paths
}

func printError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}

// buildSource creates a gomib.Source from multiple directory paths.
// Returns nil if no valid paths are found.
func buildSource(mibPaths []string) gomib.Source {
	var sources []gomib.Source
	for _, p := range mibPaths {
		src, err := gomib.DirTree(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping path %s: %v\n", p, err)
			continue
		}
		sources = append(sources, src)
	}
	if len(sources) == 0 {
		return nil
	}
	if len(sources) == 1 {
		return sources[0]
	}
	return gomib.Multi(sources...)
}
