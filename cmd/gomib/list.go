package main

import (
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/golangsnmp/gomib"
)

const listUsage = `gomib list - List available module names

Usage:
  gomib list [options]

Lists all available module names from configured sources without loading or
parsing them.

Options:
  --count         Print only the module count
  --format FMT    Output format: text, json (default: text)
  -h, --help      Show help

Examples:
  gomib list -p testdata/corpus/primary
  gomib list -p testdata/corpus/primary --count
  gomib list -p testdata/corpus/primary --format json
`

func (c *cli) cmdList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, listUsage) }

	count := fs.Bool("count", false, "print only module count")
	format := fs.String("format", "text", "output format: text, json")
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, listUsage) {
		return exitOK
	}

	sources, useSystem, err := c.buildSources()
	if err != nil {
		printError("%v", err)
		return exitIssue
	}
	if useSystem {
		sources = gomib.DiscoverSystemSources()
	}

	if len(sources) == 0 {
		printError("no sources available")
		return exitIssue
	}

	src := gomib.Multi(sources...)
	names, err := src.ListModules()
	if err != nil {
		printError("listing modules: %v", err)
		return exitIssue
	}

	slices.Sort(names)

	if *count {
		fmt.Println(len(names))
		return exitOK
	}

	switch *format {
	case "json":
		if err := writeJSON(os.Stdout, names, true); err != nil {
			printError("encoding JSON: %v", err)
			return exitIssue
		}
	case "text", "":
		for _, name := range names {
			fmt.Println(name)
		}
	default:
		printError("unknown format: %s", *format)
		return exitIssue
	}
	return exitOK
}
