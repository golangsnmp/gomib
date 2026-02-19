package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/golangsnmp/gomib"
)

const listUsage = `gomib list - List available module names

Usage:
  gomib list [options]

Lists all available module names from configured sources without loading or
parsing them.

Options:
  --count      Print only the module count
  --json       Output as JSON array
  -h, --help   Show help

Examples:
  gomib list -p testdata/corpus/primary
  gomib list -p testdata/corpus/primary --count
  gomib list -p testdata/corpus/primary --json
`

func (c *cli) cmdList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, listUsage) }

	count := fs.Bool("count", false, "print only module count")
	jsonOut := fs.Bool("json", false, "output as JSON array")
	help := fs.Bool("h", false, "show help")
	fs.BoolVar(help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *help || c.helpFlag {
		_, _ = fmt.Fprint(os.Stdout, listUsage)
		return 0
	}

	sources, useSystem, err := c.buildSources()
	if err != nil {
		printError("%v", err)
		return exitError
	}
	if useSystem {
		sources = gomib.DiscoverSystemSources()
	}

	if len(sources) == 0 {
		printError("no sources available")
		return exitError
	}

	src := gomib.Multi(sources...)
	names, err := src.ListModules()
	if err != nil {
		printError("listing modules: %v", err)
		return exitError
	}

	sort.Strings(names)

	if *count {
		fmt.Println(len(names))
		return 0
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(names); err != nil {
			printError("encoding JSON: %v", err)
			return exitError
		}
		return 0
	}

	for _, name := range names {
		fmt.Println(name)
	}
	return 0
}
