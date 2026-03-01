//go:build cgo

// Command gomib-libsmi compares gomib against libsmi for lexer/parser cross-validation.
// Build with: CGO_ENABLED=1 go build -tags cgo ./cmd/gomib-libsmi
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

type cli struct {
	paths      []string
	outputFile string
	jsonOutput bool
}

func main() {
	os.Exit(run())
}

func run() int {
	flags, cmd, cmdArgs, errMsg := cliutil.ParseCGOArgs(os.Args[1:])
	if errMsg != "" {
		cliutil.PrintError("%s", errMsg)
		return 1
	}
	c := cli{
		paths:      flags.Paths,
		outputFile: flags.OutputFile,
		jsonOutput: flags.JSONOutput,
	}

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
		return c.cmdDiag(cmdArgs)
	case "accept":
		return c.cmdAccept(cmdArgs)
	case "compare":
		return c.cmdCompare(cmdArgs)
	case "help":
		fmt.Fprint(os.Stdout, usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		fmt.Fprint(os.Stderr, usage)
		return 1
	}
}

func (c *cli) getOutput() (*os.File, func(), error) {
	return cliutil.GetOutput(c.outputFile)
}

// buildSource creates a gomib.Source from multiple directory paths.
// Returns nil if no valid paths are found.
func buildSource(mibPaths []string) gomib.Source {
	sources := cliutil.BuildSources(mibPaths)
	if len(sources) == 0 {
		return nil
	}
	if len(sources) == 1 {
		return sources[0]
	}
	return gomib.Multi(sources...)
}
