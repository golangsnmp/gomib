package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golangsnmp/gomib"
)

var discoverSystemPaths = gomib.DiscoverSystemPaths

const pathsUsage = `gomib paths - Show MIB search paths

Usage:
  gomib paths [options]

Shows MIB search paths with annotations. When -p is given, custom paths are
shown first, followed by system-discovered paths.

Options:
  -h, --help   Show help

Examples:
  gomib paths
  gomib paths -p /usr/share/snmp/mibs
`

func (c *cli) cmdPaths(args []string) int {
	fs := flag.NewFlagSet("paths", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, pathsUsage) }

	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, pathsUsage) {
		return exitOK
	}

	systemPaths := discoverSystemPaths()
	hasOutput := false

	if len(c.paths) > 0 {
		for _, p := range c.paths {
			fmt.Printf("%s (custom)\n", p)
		}
		hasOutput = true
	}

	if len(systemPaths) > 0 {
		for _, p := range systemPaths {
			fmt.Printf("%s (system)\n", p)
		}
		hasOutput = true
	}

	if !hasOutput {
		fmt.Fprintln(os.Stderr, "no search paths found")
		return exitIssue
	}

	return exitOK
}
