package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/mib"
)

const loadUsage = `gomib load - Load and resolve MIB modules

Usage:
  gomib load [options] MODULE...

Options:
  --strict        Resolver strictness: strict (tier-1 only)
  --permissive    Resolver strictness: permissive (tier-1/2/3)
  --report LEVEL  Diagnostic reporting: silent|quiet|default|verbose
  --stats         Show detailed statistics
  -h, --help      Show help

Examples:
  gomib load IF-MIB
  gomib load IF-MIB SNMPv2-MIB
  gomib load -v IF-MIB                 # Debug logging
  gomib load -vv IF-MIB                # Trace logging
  gomib load --strict IF-MIB           # strict resolver behavior
  gomib load --permissive IF-MIB       # permissive resolver behavior
  gomib load --report verbose IF-MIB   # verbose diagnostics
  gomib load --stats IF-MIB            # Show detailed stats
`

func (c *cli) cmdLoad(args []string) int {
	fs := flag.NewFlagSet("load", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, loadUsage) }

	strict := fs.Bool("strict", false, "use strict RFC compliance mode")
	permissive := fs.Bool("permissive", false, "use permissive mode for vendor MIBs")
	report := fs.String("report", "default", "diagnostic reporting: silent|quiet|default|verbose")
	stats := fs.Bool("stats", false, "show detailed statistics")
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, loadUsage) {
		return exitOK
	}

	modules := fs.Args()
	if len(modules) == 0 {
		printError("no modules specified")
		fmt.Fprint(os.Stderr, loadUsage)
		return exitError
	}

	var opts []gomib.LoadOption
	switch {
	case *strict:
		opts = append(opts, gomib.WithResolverStrictness(mib.ResolverStrict))
	case *permissive:
		opts = append(opts, gomib.WithResolverStrictness(mib.ResolverPermissive))
	}
	switch *report {
	case "silent":
		opts = append(opts, gomib.WithDiagnosticConfig(mib.SilentConfig()))
	case "quiet":
		opts = append(opts, gomib.WithDiagnosticConfig(mib.QuietConfig()))
	case "default":
		opts = append(opts, gomib.WithDiagnosticConfig(mib.DefaultConfig()))
	case "verbose":
		opts = append(opts, gomib.WithDiagnosticConfig(mib.VerboseConfig()))
	default:
		printError("invalid --report value %q (use silent|quiet|default|verbose)", *report)
		return exitError
	}

	m, loadErr := c.loadMibWithOpts(modules, opts...)
	if loadErr != nil && m == nil {
		printError("failed to load: %v", loadErr)
		return exitError
	}

	if *stats {
		printDetailedStats(m)
	} else {
		fmt.Printf("Loaded %d modules (%d types, %d objects, %d notifications)\n",
			len(m.Modules()), len(m.Types()), len(m.Objects()), len(m.Notifications()))
	}

	diags := m.Diagnostics()
	hasErrors := false
	for _, d := range diags {
		if d.Severity.AtLeast(mib.SeverityError) {
			hasErrors = true
			break
		}
	}

	if len(diags) > 0 {
		fmt.Println()
		fmt.Println("Diagnostics:")
		for _, d := range diags {
			printDiagnostic(d)
		}
	}

	unresolved := m.Unresolved()
	if len(unresolved) > 0 {
		fmt.Println()
		fmt.Println("Unresolved references:")
		importCount := 0
		typeCount := 0
		objectCount := 0
		for _, u := range unresolved {
			switch u.Kind {
			case mib.UnresolvedImport:
				importCount++
			case mib.UnresolvedType:
				typeCount++
			case mib.UnresolvedOID, mib.UnresolvedIndex, mib.UnresolvedNotificationObject:
				objectCount++
			}
		}
		if importCount > 0 {
			fmt.Printf("  %d imports\n", importCount)
		}
		if typeCount > 0 {
			fmt.Printf("  %d types\n", typeCount)
		}
		if objectCount > 0 {
			fmt.Printf("  %d objects\n", objectCount)
		}
	}

	if loadErr != nil {
		printError("%v", loadErr)
		return exitError
	}
	if hasErrors {
		return exitError
	}
	if *strict && len(unresolved) > 0 {
		return exitStrictViolation
	}
	return exitOK
}

func printDiagnostic(d mib.Diagnostic) {
	prefix := "  " + d.Severity.String() + ": "
	if d.Code != "" {
		prefix += "[" + d.Code + "] "
	}
	if d.Module != "" {
		if d.Line > 0 {
			fmt.Printf("%s%s:%d: %s\n", prefix, d.Module, d.Line, d.Message)
		} else {
			fmt.Printf("%s%s: %s\n", prefix, d.Module, d.Message)
		}
	} else {
		fmt.Printf("%s%s\n", prefix, d.Message)
	}
}

func printDetailedStats(m *mib.Mib) {
	fmt.Println("Statistics:")
	fmt.Printf("  Modules:        %d\n", len(m.Modules()))
	fmt.Printf("  Types:          %d\n", len(m.Types()))
	fmt.Printf("  Objects:        %d\n", len(m.Objects()))
	fmt.Printf("  Notifications:  %d\n", len(m.Notifications()))
	fmt.Printf("  OID nodes:      %d\n", m.NodeCount())
	fmt.Printf("  Diagnostics:    %d\n", len(m.Diagnostics()))

	kindCounts := make(map[mib.Kind]int)
	for node := range m.Nodes() {
		kindCounts[node.Kind()]++
	}

	fmt.Println()
	fmt.Println("Nodes by kind:")
	for kind := mib.KindInternal; kind <= mib.KindCapability; kind++ {
		if count := kindCounts[kind]; count > 0 {
			fmt.Printf("  %-15s %d\n", kind.String()+":", count)
		}
	}
}
