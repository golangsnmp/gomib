// Command gomib is a CLI tool for loading, querying, and dumping MIB modules.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/cmd/internal/cliutil"
	"github.com/golangsnmp/gomib/mib"
)

// Exit codes.
const (
	exitOK              = 0 // success
	exitError           = 1 // user error, processing failure, or severe diagnostic
	exitStrictViolation = 2 // strict mode found errors or unresolved refs
)

const usage = `gomib - MIB parser and query tool

Usage:
  gomib <command> [options] [arguments]

Commands:
  load    Load and resolve MIB modules
  lint    Check modules for issues (linter mode)
  get     Query OID or name lookups
  dump    Output modules or subtrees as JSON
  trace   Trace symbol resolution for debugging
  paths   Show MIB search paths
  list    List available module names
  find    Search for names across loaded MIBs
  version Show version

Common options:
  -p, --path PATH   Add MIB search path (repeatable)
  -v, --verbose     Enable debug logging
  -vv               Enable trace logging (implies -v)
  -h, --help        Show help

Examples:
  gomib load IF-MIB
  gomib get -m IF-MIB ifIndex
  gomib dump IF-MIB
  gomib trace -m IF-MIB ifEntry
  gomib paths
  gomib list -p testdata/corpus/primary
`

type cli struct {
	verbose  int
	paths    []string
	helpFlag bool
}

func main() {
	os.Exit(run())
}

func run() int {
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	var c cli
	args := os.Args[1:]
	var cmdArgs []string
	var cmd string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			c.helpFlag = true
		case arg == "-v" || arg == "--verbose":
			if c.verbose < 1 {
				c.verbose = 1
			}
		case arg == "-vv":
			c.verbose = 2
		case arg == "--no-color":
			// reserved for future use
		case arg == "-p" || arg == "--path":
			if i+1 < len(args) {
				i++
				c.paths = append(c.paths, args[i])
			}
		case strings.HasPrefix(arg, "-p"):
			c.paths = append(c.paths, arg[2:])
		case strings.HasPrefix(arg, "--path="):
			c.paths = append(c.paths, arg[7:])
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

	if c.helpFlag && cmd == "" {
		_, _ = fmt.Fprint(os.Stdout, usage)
		return 0
	}

	if cmd == "" {
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 1
	}

	switch cmd {
	case "load":
		return c.cmdLoad(cmdArgs)
	case "lint":
		return c.cmdLint(cmdArgs)
	case "get":
		return c.cmdGet(cmdArgs)
	case "dump":
		return c.cmdDump(cmdArgs)
	case "trace":
		return c.cmdTrace(cmdArgs)
	case "paths":
		return c.cmdPaths(cmdArgs)
	case "list":
		return c.cmdList(cmdArgs)
	case "find":
		return c.cmdFind(cmdArgs)
	case "version":
		printVersion()
		return 0
	case "help":
		_, _ = fmt.Fprint(os.Stdout, usage)
		return 0
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 1
	}
}

func (c *cli) setupLogger() *slog.Logger {
	if c.verbose == 0 {
		return nil
	}
	level := slog.LevelDebug
	if c.verbose >= 2 {
		level = gomib.LevelTrace
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}

func (c *cli) loadMib(modules []string) (*mib.Mib, error) {
	return c.loadMibWithOpts(modules)
}

// buildSources returns the composed source list from -p paths or system path
// discovery. Returns (nil, true) when no explicit paths are set, indicating
// that WithSystemPaths() should be used instead.
func (c *cli) buildSources() ([]gomib.Source, bool, error) {
	if len(c.paths) == 0 {
		return nil, true, nil
	}
	var sources []gomib.Source
	for _, p := range c.paths {
		if src, err := gomib.DirTree(p); err == nil {
			sources = append(sources, src)
		} else {
			fmt.Fprintf(os.Stderr, "warning: cannot access path %s: %v\n", p, err)
		}
	}
	if len(sources) == 0 {
		return nil, false, gomib.ErrNoSources
	}
	return sources, false, nil
}

func (c *cli) loadMibWithOpts(modules []string, extraOpts ...gomib.LoadOption) (*mib.Mib, error) {
	var opts []gomib.LoadOption

	sources, useSystem, err := c.buildSources()
	if err != nil {
		return nil, err
	}
	if useSystem {
		opts = append(opts, gomib.WithSystemPaths())
	} else {
		opts = append(opts, gomib.WithSource(sources...))
	}

	if logger := c.setupLogger(); logger != nil {
		opts = append(opts, gomib.WithLogger(logger))
	}
	opts = append(opts, extraOpts...)

	if len(modules) > 0 {
		opts = append(opts, gomib.WithModules(modules...))
	}
	return gomib.Load(context.Background(), opts...)
}

func printVersion() {
	version := "(devel)"
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		version = info.Main.Version
	}
	fmt.Printf("gomib %s\n", version)
}

func printError(format string, args ...any) {
	cliutil.PrintError(format, args...)
}
