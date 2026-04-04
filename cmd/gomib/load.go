package main

import (
	"flag"
	"fmt"

	"github.com/golangsnmp/gomib/mib"
)

const loadUsage = `gomib load - Load and resolve MIB modules

Usage:
  gomib load [options] [MODULE...]

Loads all available modules when no MODULE is specified.

Options:
  --strict        Resolver strictness: strict (tier-1 only)
  --permissive    Resolver strictness: permissive (tier-1/2/3)
  --report LEVEL  Diagnostic reporting: silent|quiet|default|verbose
  --stats         Show detailed statistics
  -h, --help      Show help

Examples:
  gomib load IF-MIB
  gomib load IF-MIB SNMPv2-MIB
  gomib load -p testdata/corpus/primary
  gomib load -v IF-MIB                 # Debug logging
  gomib load --strict IF-MIB           # strict resolver behavior
  gomib load --permissive IF-MIB       # permissive resolver behavior
  gomib load --report verbose IF-MIB   # verbose diagnostics
  gomib load --stats IF-MIB            # Show detailed stats
`

func (c *cli) cmdLoad(args []string) int {
	fs := flag.NewFlagSet("load", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(c.stderr, loadUsage) }

	strict, permissive := addStrictnessFlags(fs)
	report := fs.String("report", "default", "diagnostic reporting: silent|quiet|default|verbose")
	stats := fs.Bool("stats", false, "show detailed statistics")
	help := addHelpFlag(fs)

	if err := fs.Parse(args); err != nil {
		return exitError
	}

	if c.checkHelp(help, loadUsage) {
		return exitOK
	}

	if code, failed := c.validateStrictnessFlags(*strict, *permissive); failed {
		return code
	}

	modules := fs.Args()
	if len(modules) == 0 {
		modules = nil
	}

	opts := strictnessOpts(*strict, *permissive)
	reportOption, ok := c.reportOpt(*report)
	if !ok {
		return exitError
	}
	opts = append(opts, reportOption)

	m, loadErr := c.loadMibWithOpts(modules, opts...)
	if loadErr != nil && m == nil {
		c.printError("failed to load: %v", loadErr)
		return exitError
	}

	if *stats {
		c.printDetailedStats(m)
	} else {
		fmt.Fprintf(c.stdout, "Loaded %d modules (%d types, %d objects, %d notifications)\n",
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
		fmt.Fprintln(c.stderr)
		fmt.Fprintln(c.stderr, "Diagnostics:")
		for _, d := range diags {
			c.printDiagnostic(d)
		}
	}

	unresolved := m.Unresolved()
	if len(unresolved) > 0 {
		fmt.Fprintln(c.stderr)
		fmt.Fprintln(c.stderr, "Unresolved references:")
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
			fmt.Fprintf(c.stderr, "  %d imports\n", importCount)
		}
		if typeCount > 0 {
			fmt.Fprintf(c.stderr, "  %d types\n", typeCount)
		}
		if objectCount > 0 {
			fmt.Fprintf(c.stderr, "  %d objects\n", objectCount)
		}
	}

	if loadErr != nil {
		c.printError("%v", loadErr)
		return exitError
	}
	if hasErrors {
		return exitIssue
	}
	if *strict && len(unresolved) > 0 {
		return exitIssue
	}
	return exitOK
}

func (c *cli) printDiagnostic(d mib.Diagnostic) {
	prefix := "  " + d.Severity.String() + ": "
	if d.Code != "" {
		prefix += "[" + d.Code + "] "
	}
	if d.Module != "" {
		if d.Line > 0 {
			fmt.Fprintf(c.stderr, "%s%s:%d: %s\n", prefix, d.Module, d.Line, d.Message)
		} else {
			fmt.Fprintf(c.stderr, "%s%s: %s\n", prefix, d.Module, d.Message)
		}
	} else {
		fmt.Fprintf(c.stderr, "%s%s\n", prefix, d.Message)
	}
}

func (c *cli) printDetailedStats(m *mib.Mib) {
	fmt.Fprintln(c.stdout, "Statistics:")
	fmt.Fprintf(c.stdout, "  Modules:        %d\n", len(m.Modules()))
	fmt.Fprintf(c.stdout, "  Types:          %d\n", len(m.Types()))
	fmt.Fprintf(c.stdout, "  Objects:        %d\n", len(m.Objects()))
	fmt.Fprintf(c.stdout, "  Notifications:  %d\n", len(m.Notifications()))
	fmt.Fprintf(c.stdout, "  OID nodes:      %d\n", m.NodeCount())
	fmt.Fprintf(c.stdout, "  Diagnostics:    %d\n", len(m.Diagnostics()))

	kindCounts := make(map[mib.Kind]int)
	for node := range m.Nodes() {
		kindCounts[node.Kind()]++
	}

	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "Nodes by kind:")
	for kind := mib.KindInternal; kind <= mib.KindCapability; kind++ {
		if count := kindCounts[kind]; count > 0 {
			fmt.Fprintf(c.stdout, "  %-15s %d\n", kind.String()+":", count)
		}
	}
}
