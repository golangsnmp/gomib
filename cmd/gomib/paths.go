package main

import (
	"flag"
	"fmt"
)

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
	fs.Usage = func() { fmt.Fprint(c.stderr, pathsUsage) }

	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, pathsUsage) {
		return exitOK
	}

	systemPaths := c.discoverSystemPaths()
	hasOutput := false

	if len(c.paths) > 0 {
		for _, p := range c.paths {
			fmt.Fprintf(c.stdout, "%s (custom)\n", p)
		}
		hasOutput = true
	}

	if len(systemPaths) > 0 {
		for _, p := range systemPaths {
			fmt.Fprintf(c.stdout, "%s (system)\n", p)
		}
		hasOutput = true
	}

	if !hasOutput {
		fmt.Fprintln(c.stderr, "no search paths found")
		return exitIssue
	}

	return exitOK
}
