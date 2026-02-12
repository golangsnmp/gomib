// Command gomib is a CLI tool for loading, querying, and dumping MIB modules.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/golangsnmp/gomib"
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
`

var (
	verbose  int
	paths    []string
	helpFlag bool
)

func main() {
	os.Exit(run())
}

func run() int {
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	args := os.Args[1:]
	var cmdArgs []string
	var cmd string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			helpFlag = true
		case arg == "-v" || arg == "--verbose":
			if verbose < 1 {
				verbose = 1
			}
		case arg == "-vv":
			verbose = 2
		case arg == "--no-color":
			// reserved for future use
		case arg == "-p" || arg == "--path":
			if i+1 < len(args) {
				i++
				paths = append(paths, args[i])
			}
		case strings.HasPrefix(arg, "-p"):
			paths = append(paths, arg[2:])
		case strings.HasPrefix(arg, "--path="):
			paths = append(paths, arg[7:])
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

	if helpFlag && cmd == "" {
		_, _ = fmt.Fprint(os.Stdout, usage)
		return 0
	}

	if cmd == "" {
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 1
	}

	switch cmd {
	case "load":
		return cmdLoad(cmdArgs)
	case "lint":
		return cmdLint(cmdArgs)
	case "get":
		return cmdGet(cmdArgs)
	case "dump":
		return cmdDump(cmdArgs)
	case "trace":
		return cmdTrace(cmdArgs)
	case "help":
		_, _ = fmt.Fprint(os.Stdout, usage)
		return 0
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		_, _ = fmt.Fprint(os.Stderr, usage)
		return 1
	}
}

func setupLogger() *slog.Logger {
	if verbose == 0 {
		return nil
	}
	level := slog.LevelDebug
	if verbose >= 2 {
		level = gomib.LevelTrace
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}

func loadMib(modules []string) (gomib.Mib, error) {
	return loadMibWithOpts(modules)
}

func loadMibWithOpts(modules []string, extraOpts ...gomib.LoadOption) (gomib.Mib, error) {
	var source gomib.Source
	var opts []gomib.LoadOption

	if len(paths) > 0 {
		var sources []gomib.Source
		for _, p := range paths {
			if src, err := gomib.DirTree(p); err == nil {
				sources = append(sources, src)
			} else {
				fmt.Fprintf(os.Stderr, "warning: cannot access path %s: %v\n", p, err)
			}
		}
		if len(sources) == 0 {
			return nil, gomib.ErrNoSources
		}
		if len(sources) == 1 {
			source = sources[0]
		} else {
			source = gomib.Multi(sources...)
		}
	} else {
		opts = append(opts, gomib.WithSystemPaths())
	}

	if logger := setupLogger(); logger != nil {
		opts = append(opts, gomib.WithLogger(logger))
	}
	opts = append(opts, extraOpts...)

	if len(modules) > 0 {
		return gomib.LoadModules(context.Background(), modules, source, opts...)
	}
	return gomib.Load(context.Background(), source, opts...)
}

func printError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}
