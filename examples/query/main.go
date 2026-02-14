// Show different query patterns: by name, module-scoped, OID, and prefix matching.
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

	var opts []gomib.LoadOption
	if src != nil {
		opts = append(opts, gomib.WithSource(src))
	}
	opts = append(opts, gomib.WithModules("IF-MIB"), gomib.WithSystemPaths())
	m, err := gomib.Load(context.Background(), opts...)
	if err != nil {
		log.Fatal(err)
	}

	// Query by name
	fmt.Println("=== By name ===")
	obj := m.Object("ifIndex")
	if obj != nil {
		fmt.Printf("%-20s %s\n", obj.Name(), obj.OID())
	}

	// Qualified lookup via Module().Object()
	fmt.Println("\n=== Module-scoped object ===")
	obj = m.Module("IF-MIB").Object("ifDescr")
	if obj != nil {
		fmt.Printf("%-20s %s\n", obj.Name(), obj.OID())
	}

	// OID-based lookup via NodeByOID().Object()
	fmt.Println("\n=== By OID ===")
	oid, _ := gomib.ParseOID("1.3.6.1.2.1.2.2.1.3")
	node := m.NodeByOID(oid)
	if node != nil && node.Object() != nil {
		obj = node.Object()
		fmt.Printf("%-20s %s\n", obj.Name(), obj.OID())
	}

	// Exact OID lookup
	fmt.Println("\n=== NodeByOID (exact) ===")
	oid, _ = gomib.ParseOID("1.3.6.1.2.1.2.2.1.1")
	node = m.NodeByOID(oid)
	if node != nil {
		fmt.Printf("%-20s %s  kind=%s\n", node.Name(), node.OID(), node.Kind())
	}

	// Longest prefix match (useful for instance OIDs like ifDescr.1)
	fmt.Println("\n=== LongestPrefixByOID ===")
	instanceOID, _ := gomib.ParseOID("1.3.6.1.2.1.2.2.1.2.17")
	prefix := m.LongestPrefixByOID(instanceOID)
	if prefix != nil {
		fmt.Printf("%-20s %s  (matched from %s)\n", prefix.Name(), prefix.OID(), instanceOID)
	}

	// Module-scoped type lookup
	fmt.Println("\n=== Module-scoped type ===")
	mod := m.Module("IF-MIB")
	typ := mod.Type("InterfaceIndex")
	if typ != nil {
		fmt.Printf("type %-16s base=%s  tc=%v\n", typ.Name(), typ.Base(), typ.IsTextualConvention())
	}

	// Find different definition kinds
	fmt.Println("\n=== Other definitions ===")
	notif := m.Notification("linkDown")
	if notif != nil {
		fmt.Printf("notification %-16s %s\n", notif.Name(), notif.OID())
	}
	grp := m.Group("ifGeneralInformationGroup")
	if grp != nil {
		fmt.Printf("group        %-16s %s  members=%d\n", grp.Name(), grp.OID(), len(grp.Members()))
	}
}
