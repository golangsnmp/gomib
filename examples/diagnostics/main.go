// Load MIBs at different strictness levels and show diagnostic output.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/mib"
)

func main() {
	path := flag.String("p", "", "MIB search path (default: system paths)")
	flag.Parse()

	var src gomib.Source
	if *path != "" {
		var err error
		src, err = gomib.DirTree(*path)
		if err != nil {
			log.Fatal(err)
		}
	}

	levels := []struct {
		name  string
		level mib.StrictnessLevel
	}{
		{"Strict", mib.StrictnessStrict},
		{"Normal", mib.StrictnessNormal},
		{"Permissive", mib.StrictnessPermissive},
	}

	for _, l := range levels {
		fmt.Printf("=== %s ===\n", l.name)
		var loadOpts []gomib.LoadOption
		if src != nil {
			loadOpts = append(loadOpts, gomib.WithSource(src))
		}
		loadOpts = append(loadOpts, gomib.WithModules("IF-MIB"), gomib.WithSystemPaths(), gomib.WithStrictness(l.level))
		m, err := gomib.Load(context.Background(), loadOpts...)
		if err != nil {
			fmt.Printf("  load error: %v\n\n", err)
			continue
		}

		diags := m.Diagnostics()
		fmt.Printf("  modules:     %d\n", len(m.Modules()))
		fmt.Printf("  diagnostics: %d\n", len(diags))
		fmt.Printf("  has errors:  %v\n", m.HasErrors())

		for i, d := range diags {
			if i >= 10 {
				fmt.Printf("  ... and %d more\n", len(diags)-10)
				break
			}
			loc := ""
			if d.Line > 0 {
				loc = fmt.Sprintf(" (line %d)", d.Line)
			}
			fmt.Printf("  [%s] %s: %s%s\n", d.Severity, d.Module, d.Message, loc)
		}

		unresolved := m.Unresolved()
		if len(unresolved) > 0 {
			fmt.Printf("  unresolved refs: %d\n", len(unresolved))
			for i, ref := range unresolved {
				if i >= 5 {
					fmt.Printf("  ... and %d more\n", len(unresolved)-5)
					break
				}
				fmt.Printf("    %s %q in %s\n", ref.Kind, ref.Symbol, ref.Module)
			}
		}
		fmt.Println()
	}

	// Fine-grained diagnostic config
	fmt.Println("=== Custom config ===")
	var customOpts []gomib.LoadOption
	if src != nil {
		customOpts = append(customOpts, gomib.WithSource(src))
	}
	customOpts = append(customOpts, gomib.WithModules("IF-MIB"), gomib.WithSystemPaths(),
		gomib.WithDiagnosticConfig(mib.DiagnosticConfig{
			Level:  mib.StrictnessNormal,
			FailAt: mib.SeverityFatal,
			Ignore: []string{"identifier-*"},
		}))
	m, err := gomib.Load(context.Background(), customOpts...)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  diagnostics (ignoring identifier-*): %d\n", len(m.Diagnostics()))
}
