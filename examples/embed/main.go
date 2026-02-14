// Use embed.FS with gomib.FS() to load MIBs embedded in the binary.
//
// Combines embedded vendor MIBs with system paths for base modules (SNMPv2-SMI, etc.)
// using WithSystemPaths(). In a real application, embed your vendor-specific MIBs and
// rely on system-installed MIBs for standard dependencies.
package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/golangsnmp/gomib"
)

//go:embed mibs
var mibFS embed.FS

func main() {
	embedded := gomib.FS("embedded", mibFS)

	// Embedded MIBs checked first, then fall back to system MIBs for imports
	m, err := gomib.LoadModules(context.Background(),
		[]string{"EXAMPLE-MIB"},
		embedded,
		gomib.WithSystemPaths(),
		gomib.WithStrictness(gomib.StrictnessPermissive),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d modules\n", m.ModuleCount())
	for _, mod := range m.Modules() {
		fmt.Printf("  %-24s %s  objects=%d\n", mod.Name(), mod.Language(), len(mod.Objects()))
	}

	// The embedded EXAMPLE-MIB is available alongside standard modules
	fmt.Println()
	obj := m.Object("exampleName")
	if obj != nil {
		fmt.Printf("%s  %s\n", obj.Name(), obj.OID())
		fmt.Printf("  type:   %s (base: %s)\n", obj.Type().Name(), obj.Type().EffectiveBase())
		fmt.Printf("  access: %s\n", obj.Access())
	}

	obj = m.Object("exampleCount")
	if obj != nil {
		fmt.Printf("\n%s  %s\n", obj.Name(), obj.OID())
		fmt.Printf("  type:   %s (base: %s)\n", obj.Type().Name(), obj.Type().EffectiveBase())
		fmt.Printf("  access: %s\n", obj.Access())
	}
}
