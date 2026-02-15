//go:build cgo

// Command gomib-libsmi compares gomib against libsmi for lexer/parser cross-validation.
// Build with: CGO_ENABLED=1 go build -tags cgo ./cmd/gomib-libsmi
//
//nolint:errcheck // CLI tool, fmt output errors are not critical
package main

import (
	"fmt"
	"os"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/cmd/internal/cliutil"
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
)

func main() {
	os.Exit(run())
}

func run() int {
	flags, cmd, cmdArgs := cliutil.ParseCGOArgs(os.Args[1:])
	paths = flags.Paths
	outputFile = flags.OutputFile
	jsonOutput = flags.JSONOutput

	if flags.HelpFlag && cmd == "" {
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
	return cliutil.GetOutput(outputFile)
}

func getMIBPaths() []string {
	return paths
}

func printError(format string, args ...any) {
	cliutil.PrintError(format, args...)
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
