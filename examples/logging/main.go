// Example: logging - enable debug logging to see what gomib is doing.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/golangsnmp/gomib"
)

func main() {
	corpusPath := findCorpus()

	// Create source from directory tree
	source, err := gomib.DirTree(corpusPath)
	if err != nil {
		log.Fatalf("failed to open MIB directory: %v", err)
	}

	// Create a logger at debug level
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	fmt.Println("Loading with debug logging enabled...")
	fmt.Println("(Log output goes to stderr)")
	fmt.Println()

	// Load with logging - this will show module loading, resolution phases, etc.
	mib, err := gomib.LoadModules(context.Background(),
		[]string{"IF-MIB"},
		source,
		gomib.WithLogger(logger),
	)
	if err != nil {
		log.Fatalf("failed to load: %v", err)
	}

	fmt.Printf("\nLoaded %d modules\n", mib.ModuleCount())

	// For even more detail, use trace level
	fmt.Println("\n--- Trace level example ---")
	traceLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: gomib.LevelTrace, // -8, more verbose than Debug
	}))

	// Load a single small module to see trace output
	_, _ = gomib.LoadModules(context.Background(),
		[]string{"SNMPv2-SMI"},
		source,
		gomib.WithLogger(traceLogger),
	)
}

func findCorpus() string {
	candidates := []string{
		"testdata/corpus/primary",
		"../testdata/corpus/primary",
		"gomib/testdata/corpus/primary",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	log.Fatal("could not find test corpus")
	return ""
}
