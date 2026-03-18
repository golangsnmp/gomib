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

// Exit codes following the grep/shellcheck convention.
const (
	exitOK    = 0 // success (clean load, match found, no issues)
	exitIssue = 1 // negative result (issues/violations found, no match)
	exitError = 2 // operational error (load failure, parse error)
)

const usage = `gomib - MIB parser and query tool

Usage:
  gomib <command> [options] [arguments]

Commands:
  load       Load and resolve MIB modules
  lint       Check modules for issues (linter mode)
  get        Query OID or name lookups
  dump       Export resolved MIB data as canonical JSON
  normalize  Emit modules as canonical SMIv2 text
  trace      Trace symbol resolution for debugging
  paths      Show MIB search paths
  list       List available module names
  find       Search for names across loaded MIBs
  version    Show version
  help       Show help

Common options:
  -p, --path PATH   Add MIB search path (repeatable)
  -v, --verbose     Enable debug logging
  -vv               Enable trace logging (implies -v)
  --version         Show version
  -h, --help        Show help

Examples:
  gomib load IF-MIB
  gomib get -m IF-MIB ifIndex
  gomib dump IF-MIB
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
	versionFlag := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			c.helpFlag = true
		case (arg == "--version" || arg == "-V") && cmd == "":
			versionFlag = true
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
	if versionFlag && cmd == "" {
		printVersion()
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

// addModuleFlag registers -m/--module (repeatable) on the given FlagSet.
func addModuleFlag(fs *flag.FlagSet) *moduleList {
	var m moduleList
	fs.Var(&m, "m", "module to load")
	fs.Var(&m, "module", "module to load")
	return &m
}

// checkHelp prints usage and returns true if help was requested.
func (c *cli) checkHelp(help *bool, usageText string) bool {
	if *help || c.helpFlag {
		_, _ = fmt.Fprint(os.Stdout, usageText)
		return true
	}
	return false
}

// addStrictnessFlags registers --strict and --permissive on the given FlagSet.
func addStrictnessFlags(fs *flag.FlagSet) (strict, permissive *bool) {
	return fs.Bool("strict", false, "use strict RFC compliance mode"),
		fs.Bool("permissive", false, "use permissive mode for vendor MIBs")
}

func validateStrictnessFlags(strict, permissive bool) (int, bool) {
	if strict && permissive {
		printError("--strict and --permissive are mutually exclusive")
		return exitError, true
	}
	return exitOK, false
}

// strictnessOpts returns LoadOptions for the given --strict/--permissive flags.
//
// --strict sets resolver strictness to tier-1 only (no fallbacks).
// --permissive sets resolver strictness to tier-3 (all fallbacks) and relaxes
// the diagnostic failure threshold so only fatal parse errors cause failure.
// This makes --permissive a "best effort" mode suitable for loading large
// vendor MIB sets that may contain spec violations.
func strictnessOpts(strict, permissive bool) []gomib.LoadOption {
	switch {
	case strict:
		return []gomib.LoadOption{
			gomib.WithResolverStrictness(mib.ResolverStrict),
		}
	case permissive:
		return permissiveSilentOpts()
	default:
		return nil
	}
}

// reportOpt returns the LoadOption for the given --report flag value.
// Returns false if the value is invalid (after printing an error).
func reportOpt(report string) (gomib.LoadOption, bool) {
	switch report {
	case "silent":
		return gomib.WithDiagnosticConfig(mib.SilentConfig()), true
	case "quiet":
		return gomib.WithDiagnosticConfig(mib.QuietConfig()), true
	case "default":
		return gomib.WithDiagnosticConfig(mib.DefaultConfig()), true
	case "verbose":
		return gomib.WithDiagnosticConfig(mib.VerboseConfig()), true
	default:
		printError("invalid --report value %q (use silent|quiet|default|verbose)", report)
		return nil, false
	}
}

// permissiveSilentOpts returns LoadOptions for permissive+silent mode,
// the default for query commands (get, find).
func permissiveSilentOpts() []gomib.LoadOption {
	return []gomib.LoadOption{
		gomib.WithResolverStrictness(mib.ResolverPermissive),
		gomib.WithDiagnosticConfig(mib.SilentConfig()),
	}
}
