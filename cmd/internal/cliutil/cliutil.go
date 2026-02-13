// Package cliutil provides shared CLI utilities for gomib command-line tools.
package cliutil

import (
	"fmt"
	"os"
	"strings"
)

// CGOFlags holds the common flags shared by CGO cross-validation tools.
type CGOFlags struct {
	Paths      []string
	OutputFile string
	JSONOutput bool
	HelpFlag   bool
}

// ParseCGOArgs parses global flags and extracts the subcommand from args.
// Flags handled: -p/--path, -o/--output, -json, -h/--help.
// Unrecognized flags are passed through to the subcommand.
func ParseCGOArgs(args []string) (flags CGOFlags, cmd string, cmdArgs []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			flags.HelpFlag = true
		case arg == "-json":
			flags.JSONOutput = true
		case arg == "-p" || arg == "--path":
			if i+1 < len(args) {
				i++
				flags.Paths = append(flags.Paths, args[i])
			}
		case strings.HasPrefix(arg, "-p"):
			flags.Paths = append(flags.Paths, arg[2:])
		case strings.HasPrefix(arg, "--path="):
			flags.Paths = append(flags.Paths, arg[7:])
		case arg == "-o" || arg == "--output":
			if i+1 < len(args) {
				i++
				flags.OutputFile = args[i]
			}
		case strings.HasPrefix(arg, "-o"):
			flags.OutputFile = arg[2:]
		case strings.HasPrefix(arg, "--output="):
			flags.OutputFile = arg[9:]
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
	return
}

// GetOutput opens the output file or returns stdout.
func GetOutput(outputFile string) (*os.File, func(), error) {
	if outputFile == "" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(outputFile)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { _ = f.Close() }, nil
}

// PrintError writes a formatted error message to stderr.
func PrintError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}
