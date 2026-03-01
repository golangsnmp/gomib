package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golangsnmp/gomib/smiwrite"
)

const normalizeUsage = `gomib normalize - Emit modules as canonical SMIv2 text

Usage:
  gomib normalize [options] MODULE...

Options:
  -o, --output DIR         Write each module to a file in DIR (default: stdout)
  --no-conformance         Omit conformance constructs
  --no-descriptions        Omit DESCRIPTION clauses
  --no-sequences           Omit reconstructed SEQUENCE types
  -h, --help               Show help

Examples:
  gomib normalize IF-MIB
  gomib normalize --no-conformance IF-MIB
  gomib normalize -o /tmp/normalized IF-MIB SNMPv2-MIB
`

func (c *cli) cmdNormalize(args []string) int {
	fs := flag.NewFlagSet("normalize", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, normalizeUsage) }

	outputDir := fs.String("o", "", "write each module to a file in DIR")
	fs.StringVar(outputDir, "output", "", "write each module to a file in DIR")
	noConformance := fs.Bool("no-conformance", false, "omit conformance constructs")
	noDescriptions := fs.Bool("no-descriptions", false, "omit DESCRIPTION clauses")
	noSequences := fs.Bool("no-sequences", false, "omit reconstructed SEQUENCE types")
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, normalizeUsage) {
		return exitOK
	}

	modules := fs.Args()
	if len(modules) == 0 {
		printError("no modules specified")
		fmt.Fprint(os.Stderr, normalizeUsage)
		return exitError
	}

	m, err := c.loadMib(modules)
	if err != nil {
		printError("failed to load: %v", err)
		return exitError
	}

	var opts []smiwrite.Option
	if *noConformance {
		opts = append(opts, smiwrite.WithConformance(false))
	}
	if *noDescriptions {
		opts = append(opts, smiwrite.WithDescriptions(false))
	}
	if *noSequences {
		opts = append(opts, smiwrite.WithSequences(false))
	}

	if *outputDir != "" {
		if err := os.MkdirAll(*outputDir, 0o750); err != nil {
			printError("failed to create output directory: %v", err)
			return exitError
		}
		for _, name := range modules {
			path := filepath.Join(*outputDir, name)
			f, err := os.Create(path)
			if err != nil {
				printError("failed to create %s: %v", path, err)
				return exitError
			}
			if err := smiwrite.Write(f, m, name, opts...); err != nil {
				_ = f.Close()
				printError("failed to write %s: %v", name, err)
				return exitError
			}
			if err := f.Close(); err != nil {
				printError("failed to close %s: %v", path, err)
				return exitError
			}
		}
	} else {
		if err := smiwrite.WriteAll(os.Stdout, m, modules, opts...); err != nil {
			printError("failed to write: %v", err)
			return exitError
		}
	}

	return exitOK
}
