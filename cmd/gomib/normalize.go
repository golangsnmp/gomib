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
  gomib normalize [options] [MODULE...]

Normalizes all available modules when no MODULE is specified.

Options:
  -o, --output DIR         Write each module to a file in DIR (default: stdout)
  --no-conformance         Omit conformance constructs
  --no-descriptions        Omit DESCRIPTION clauses
  --no-sequences           Omit reconstructed SEQUENCE types
  --strict                 Resolver strictness: strict (tier-1 only)
  --permissive             Resolver strictness: permissive (tier-1/2/3)
  -h, --help               Show help

Examples:
  gomib normalize IF-MIB
  gomib normalize --no-conformance IF-MIB
  gomib normalize -o /tmp/normalized IF-MIB SNMPv2-MIB
  gomib normalize -o /tmp/normalized -p testdata/corpus/primary
`

func (c *cli) cmdNormalize(args []string) int {
	fs := flag.NewFlagSet("normalize", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(c.stderr, normalizeUsage) }

	outputDir := fs.String("o", "", "write each module to a file in DIR")
	fs.StringVar(outputDir, "output", "", "write each module to a file in DIR")
	noConformance := fs.Bool("no-conformance", false, "omit conformance constructs")
	noDescriptions := fs.Bool("no-descriptions", false, "omit DESCRIPTION clauses")
	noSequences := fs.Bool("no-sequences", false, "omit reconstructed SEQUENCE types")
	strict, permissive := addStrictnessFlags(fs)
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, normalizeUsage) {
		return exitOK
	}

	if code, failed := c.validateStrictnessFlags(*strict, *permissive); failed {
		return code
	}

	modules := fs.Args()
	loadAll := len(modules) == 0

	var loadModules []string
	if !loadAll {
		loadModules = modules
	}
	m, loadErr := c.loadMibWithOpts(loadModules, strictnessOpts(*strict, *permissive)...)
	if loadErr != nil && m == nil {
		c.printError("failed to load: %v", loadErr)
		return exitError
	}

	if loadAll {
		for _, mod := range m.Modules() {
			if mod.IsBase() {
				continue
			}
			modules = append(modules, mod.Name())
		}
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
		cleanDir := filepath.Clean(*outputDir)
		if err := os.MkdirAll(cleanDir, 0o750); err != nil {
			c.printError("failed to create output directory: %v", err)
			return exitError
		}
		for _, name := range modules {
			path := filepath.Join(cleanDir, filepath.Base(name))
			f, err := os.Create(path) // #nosec G304,G703 -- outputDir is a user-provided CLI flag
			if err != nil {
				c.printError("failed to create %s: %v", path, err)
				return exitError
			}
			if err := smiwrite.Write(f, m, name, opts...); err != nil {
				_ = f.Close()
				c.printError("failed to write %s: %v", name, err)
				return exitError
			}
			if err := f.Close(); err != nil {
				c.printError("failed to close %s: %v", path, err)
				return exitError
			}
		}
	} else {
		if err := smiwrite.WriteAll(c.stdout, m, modules, opts...); err != nil {
			c.printError("failed to write: %v", err)
			return exitError
		}
	}

	return exitOK
}
