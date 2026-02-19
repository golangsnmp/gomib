package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golangsnmp/gomib"
)

const pathsUsage = `gomib paths - Show MIB search paths

Usage:
  gomib paths [options]

Shows the MIB search paths that would be used. When -p paths are specified,
shows those. Otherwise shows system-discovered paths (net-snmp + libsmi config).

Options:
  -h, --help   Show help

Examples:
  gomib paths
  gomib paths -p /usr/share/snmp/mibs
`

func (c *cli) cmdPaths(args []string) int {
	fs := flag.NewFlagSet("paths", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, pathsUsage) }

	help := fs.Bool("h", false, "show help")
	fs.BoolVar(help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *help || c.helpFlag {
		_, _ = fmt.Fprint(os.Stdout, pathsUsage)
		return 0
	}

	var paths []string
	if len(c.paths) > 0 {
		paths = c.paths
	} else {
		paths = gomib.DiscoverSystemPaths()
	}

	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "no search paths found")
		return 0
	}

	for _, p := range paths {
		fmt.Println(p)
	}
	return 0
}
