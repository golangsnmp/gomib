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
	exitError           = 1 // user error, processing failure, or error-severity diagnostic
	exitStrictViolation = 2 // strict mode found errors or unresolved refs
)

const usage = `gomib - MIB parser and query tool

Usage:
  gomib <command> [options] [arguments]

Commands:
  load       Load and resolve MIB modules
  lint       Check modules for issues (linter mode)
  get        Query OID or name lookups
  dump       Output modules or subtrees as JSON
  normalize  Emit modules as canonical SMIv2 text
  trace      Trace symbol resolution for debugging
  paths      Show MIB search paths
  list       List available module names
  find       Search for names across loaded MIBs
  version    Show version

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
		case arg == "-p" || arg == "--path":
			if i+1 >= len(args) {
				printError("%s requires a value", arg)
				return exitError
			}
			i++
			c.paths = append(c.paths, args[i])
		case strings.HasPrefix(arg, "-p"):
			c.paths = append(c.paths, arg[2:])
		case strings.HasPrefix(arg, "--path="):
			c.paths = append(c.paths, arg[7:])
		case arg != "" && arg[0] == '-':
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
		return exitOK
	}

	if cmd == "" {
		_, _ = fmt.Fprint(os.Stderr, usage)
		return exitError
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
	case "normalize":
		return c.cmdNormalize(cmdArgs)
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
		return exitOK
	case "help":
		_, _ = fmt.Fprint(os.Stdout, usage)
		return exitOK
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		_, _ = fmt.Fprint(os.Stderr, usage)
		return exitError
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
	sources := cliutil.BuildSources(c.paths)
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

// addHelpFlag registers -h and --help on the given FlagSet.
func addHelpFlag(fs *flag.FlagSet) *bool {
	help := fs.Bool("h", false, "show help")
	fs.BoolVar(help, "help", false, "show help")
	return help
}

// addModuleFlags registers -m/--module (repeatable) and --all flags.
func addModuleFlags(fs *flag.FlagSet) (modules *moduleList, loadAll *bool) {
	var m moduleList
	fs.Var(&m, "m", "module to load")
	fs.Var(&m, "module", "module to load")
	return &m, fs.Bool("all", false, "load all MIBs from search path")
}

// checkHelp prints usage and returns true if help was requested.
func (c *cli) checkHelp(help *bool, usageText string) bool {
	if *help || c.helpFlag {
		_, _ = fmt.Fprint(os.Stdout, usageText)
		return true
	}
	return false
}

// requireModuleOrAll validates that either -m or --all was specified.
// Returns an exit code and true if the validation failed.
func requireModuleOrAll(modules moduleList, loadAll bool, usageText string) (int, bool) {
	if !loadAll && len(modules) == 0 {
		printError("specify -m MODULE or --all")
		fmt.Fprint(os.Stderr, usageText)
		return exitError, true
	}
	return exitOK, false
}

// modulesToLoad returns the module list for loading, or nil if --all is set.
func modulesToLoad(modules moduleList, loadAll bool) []string {
	if loadAll {
		return nil
	}
	return modules
}
