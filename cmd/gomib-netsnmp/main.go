//go:build cgo

// Command gomib-netsnmp compares gomib against net-snmp for cross-validation.
// Build with: CGO_ENABLED=1 go build -tags cgo ./cmd/gomib-netsnmp
//
//nolint:errcheck // CLI tool, fmt output errors are not critical
package main

import (
	"fmt"
	"os"

	"github.com/golangsnmp/gomib/cmd/internal/cliutil"
)

const usage = `gomib-netsnmp - Compare gomib against net-snmp

Usage:
  gomib-netsnmp <command> [options] [arguments]

Commands:
  compare    Full semantic comparison between gomib and net-snmp
  tables     Table-focused comparison (INDEX, AUGMENTS, columns)
  testgen    Generate Go test cases from net-snmp ground truth
  validate   Validate existing test cases against net-snmp
  fixturegen Generate JSON fixture files from net-snmp ground truth

Common options:
  -p, --path PATH   Add MIB search path (repeatable)
  -o, --output FILE Write output to file instead of stdout
  -json             Output in JSON format
  -h, --help        Show help

Examples:
  gomib-netsnmp compare -p /usr/share/snmp/mibs IF-MIB
  gomib-netsnmp tables -p ./testdata/corpus/primary SYNTHETIC-MIB
  gomib-netsnmp testgen -type tables -p ./testdata SYNTHETIC-MIB
  gomib-netsnmp validate -p ./testdata
`

var (
	paths      []string
	outputFile string
	jsonOutput bool
)

func main() {
	os.Exit(run())
}

func run() int {
	flags, cmd, cmdArgs := cliutil.ParseCGOArgs(os.Args[1:])
	paths = flags.Paths
	outputFile = flags.OutputFile
	jsonOutput = flags.JSONOutput

	if flags.HelpFlag && cmd == "" {
		fmt.Fprint(os.Stdout, usage)
		return 0
	}

	if cmd == "" {
		fmt.Fprint(os.Stderr, usage)
		return 1
	}

	switch cmd {
	case "compare":
		return cmdCompare(cmdArgs)
	case "tables":
		return cmdTables(cmdArgs)
	case "testgen":
		return cmdTestgen(cmdArgs)
	case "validate":
		return cmdValidate(cmdArgs)
	case "fixturegen":
		return cmdFixturegen(cmdArgs)
	case "help":
		fmt.Fprint(os.Stdout, usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		fmt.Fprint(os.Stderr, usage)
		return 1
	}
}

func getOutput() (*os.File, func(), error) {
	return cliutil.GetOutput(outputFile)
}

func getMIBPaths() []string {
	return paths
}

func printError(format string, args ...any) {
	cliutil.PrintError(format, args...)
}
