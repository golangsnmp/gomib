// Show type chain walking, textual conventions, and effective constraints.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/golangsnmp/gomib"
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

	m, err := gomib.LoadModules(context.Background(), []string{"IF-MIB"}, src, gomib.WithSystemPaths())
	if err != nil {
		log.Fatal(err)
	}

	// Walk the type chain for DisplayString
	fmt.Println("=== Type chain: DisplayString ===")
	typ := m.Type("DisplayString")
	for t := typ; t != nil; t = t.Parent() {
		fmt.Printf("  %s (base: %s)\n", t.Name(), t.Base())
		if t.IsTextualConvention() {
			fmt.Println("    textual convention")
		}
		if t.DisplayHint() != "" {
			fmt.Printf("    hint: %q\n", t.DisplayHint())
		}
		if len(t.Sizes()) > 0 {
			fmt.Printf("    sizes: %v\n", t.Sizes())
		}
	}

	// Effective values resolve through the chain
	fmt.Println("\n=== Effective values: DisplayString ===")
	if typ != nil {
		fmt.Printf("  EffectiveBase:        %s\n", typ.EffectiveBase())
		fmt.Printf("  EffectiveDisplayHint: %q\n", typ.EffectiveDisplayHint())
		fmt.Printf("  EffectiveSizes:       %v\n", typ.EffectiveSizes())
	}

	// Enumeration type (ifType -> IANAifType)
	fmt.Println("\n=== Enum type: ifType ===")
	obj := m.Object("ifType")
	if obj != nil {
		fmt.Printf("  object type: %s\n", obj.Type().Name())
		fmt.Printf("  is enum:     %v\n", obj.Type().IsEnumeration())
		enums := obj.EffectiveEnums()
		fmt.Printf("  values:      %d total\n", len(enums))
		for i, e := range enums {
			if i >= 10 {
				fmt.Printf("    ... and %d more\n", len(enums)-10)
				break
			}
			fmt.Printf("    %s(%d)\n", e.Label, e.Value)
		}
	}

	// InterfaceIndex - TC with range constraint
	fmt.Println("\n=== Textual convention: InterfaceIndex ===")
	tc := m.Type("InterfaceIndex")
	if tc != nil {
		fmt.Printf("  TC:     %v\n", tc.IsTextualConvention())
		fmt.Printf("  Base:   %s\n", tc.Base())
		fmt.Printf("  Ranges: %v\n", tc.EffectiveRanges())
		fmt.Printf("  Hint:   %q\n", tc.EffectiveDisplayHint())
	}

	// Classification helpers
	fmt.Println("\n=== Type classification ===")
	for _, name := range []string{"ifIndex", "ifDescr", "ifType", "ifInOctets", "ifSpeed"} {
		o := m.Object(name)
		if o == nil || o.Type() == nil {
			continue
		}
		t := o.Type()
		var flags []string
		if t.IsCounter() {
			flags = append(flags, "counter")
		}
		if t.IsGauge() {
			flags = append(flags, "gauge")
		}
		if t.IsString() {
			flags = append(flags, "string")
		}
		if t.IsEnumeration() {
			flags = append(flags, "enum")
		}
		if t.IsBits() {
			flags = append(flags, "bits")
		}
		fmt.Printf("  %-16s %-20s %v\n", name, t.Name(), flags)
	}
}
