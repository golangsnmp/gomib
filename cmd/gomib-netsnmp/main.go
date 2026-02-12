//go:build cgo

// Command gomib-netsnmp compares gomib against net-snmp for cross-validation.
// Build with: CGO_ENABLED=1 go build -tags cgo ./cmd/gomib-netsnmp
//
//nolint:errcheck // CLI tool, fmt output errors are not critical
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
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
	helpFlag   bool
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
		case arg == "-json":
			jsonOutput = true
		case arg == "-p" || arg == "--path":
			if i+1 < len(args) {
				i++
				paths = append(paths, args[i])
			}
		case strings.HasPrefix(arg, "-p"):
			paths = append(paths, arg[2:])
		case strings.HasPrefix(arg, "--path="):
			paths = append(paths, arg[7:])
		case arg == "-o" || arg == "--output":
			if i+1 < len(args) {
				i++
				outputFile = args[i]
			}
		case strings.HasPrefix(arg, "-o"):
			outputFile = arg[2:]
		case strings.HasPrefix(arg, "--output="):
			outputFile = arg[9:]
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
	if outputFile == "" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(outputFile)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { _ = f.Close() }, nil
}

// getMIBPaths returns MIB paths from -p flags only, with no defaults,
// to ensure fair comparison between libraries.
func getMIBPaths() []string {
	return paths
}

func printError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}
