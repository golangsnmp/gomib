package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golangsnmp/gomib"
)

const loadUsage = `gomib load - Load and resolve MIB modules

Usage:
  gomib load [options] MODULE...

Options:
  --strict      Exit non-zero if any unresolved references
  --stats       Show detailed statistics
  -h, --help    Show help

Examples:
  gomib load IF-MIB
  gomib load IF-MIB SNMPv2-MIB
  gomib load -v IF-MIB                 # Debug logging
  gomib load -vv IF-MIB                # Trace logging
  gomib load --stats IF-MIB            # Show detailed stats
`

func cmdLoad(args []string) int {
	fs := flag.NewFlagSet("load", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, loadUsage) }

	strict := fs.Bool("strict", false, "exit non-zero if unresolved references")
	stats := fs.Bool("stats", false, "show detailed statistics")
	help := fs.Bool("h", false, "show help")
	fs.BoolVar(help, "help", false, "show help")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *help || helpFlag {
		_, _ = fmt.Fprint(os.Stdout, loadUsage)
		return 0
	}

	modules := fs.Args()
	if len(modules) == 0 {
		printError("no modules specified")
		fmt.Fprint(os.Stderr, loadUsage)
		return 1
	}

	mib, err := loadMib(modules)
	if err != nil {
		printError("failed to load: %v", err)
		return 1
	}

	// Print summary
	if *stats {
		printDetailedStats(mib)
	} else {
		fmt.Printf("Loaded %d modules (%d types, %d objects, %d notifications)\n",
			mib.ModuleCount(), mib.TypeCount(), mib.ObjectCount(), mib.NotificationCount())
	}

	// Print diagnostics
	diags := mib.Diagnostics()
	hasWarnings := false
	hasErrors := false
	for _, d := range diags {
		switch d.Severity {
		case gomib.SeverityWarning:
			hasWarnings = true
		case gomib.SeverityError:
			hasErrors = true
		}
	}

	if hasWarnings || hasErrors {
		fmt.Println()
		fmt.Println("Diagnostics:")
		for _, d := range diags {
			printDiagnostic(d)
		}
	}

	// Print unresolved references summary
	unresolved := mib.Unresolved()
	if len(unresolved) > 0 {
		fmt.Println()
		fmt.Println("Unresolved references:")
		importCount := 0
		typeCount := 0
		objectCount := 0
		for _, u := range unresolved {
			switch u.Kind {
			case "import":
				importCount++
			case "type":
				typeCount++
			case "object", "oid", "index", "notification":
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

	// Exit code
	if hasErrors {
		return 1
	}
	if *strict && len(unresolved) > 0 {
		return 2
	}
	return 0
}

func printDiagnostic(d gomib.Diagnostic) {
	var prefix string
	switch d.Severity {
	case gomib.SeverityError:
		prefix = "  error: "
	case gomib.SeverityWarning:
		prefix = "  warning: "
	default:
		prefix = "  "
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

func printDetailedStats(m *gomib.Mib) {
	fmt.Println("Statistics:")
	fmt.Printf("  Modules:        %d\n", m.ModuleCount())
	fmt.Printf("  Types:          %d\n", m.TypeCount())
	fmt.Printf("  Objects:        %d\n", m.ObjectCount())
	fmt.Printf("  Notifications:  %d\n", m.NotificationCount())
	fmt.Printf("  OID nodes:      %d\n", m.NodeCount())
	fmt.Printf("  Diagnostics:    %d\n", len(m.Diagnostics()))

	// Count nodes by kind
	kindCounts := make(map[gomib.Kind]int)
	m.Walk(func(node *gomib.Node) bool {
		kindCounts[node.Kind]++
		return true
	})

	fmt.Println()
	fmt.Println("Nodes by kind:")
	for kind := gomib.KindInternal; kind <= gomib.KindCapabilities; kind++ {
		if count := kindCounts[kind]; count > 0 {
			fmt.Printf("  %-15s %d\n", kind.String()+":", count)
		}
	}
}
