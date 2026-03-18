// Show different query patterns: by name, module-scoped, OID, and prefix matching.
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
		src, err = gomib.Dir(*path)
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

	// Resolve accepts any format: plain name, qualified name, or numeric OID
	fmt.Println("=== Resolve ===")
	for _, query := range []string{"ifIndex", "IF-MIB::ifDescr", "1.3.6.1.2.1.2.2.1.3"} {
		node := m.Resolve(query)
		if node != nil {
			fmt.Printf("  %-30s -> %-20s %s\n", query, node.Name(), node.OID())
		}
	}

	// ResolveOID handles instance-suffixed forms and is the inverse of FormatOID
	fmt.Println("\n=== ResolveOID ===")
	for _, query := range []string{"ifDescr.5", "IF-MIB::ifIndex.17", "1.3.6.1.2.1.2.2.1.2"} {
		oid, err := m.ResolveOID(query)
		if err != nil {
			fmt.Printf("  %-30s -> error: %v\n", query, err)
			continue
		}
		fmt.Printf("  %-30s -> %s\n", query, oid)
	}

	// FormatOID/ResolveOID round-trip
	fmt.Println("\n=== FormatOID / ResolveOID round-trip ===")
	oid, _ := m.ResolveOID("IF-MIB::ifDescr.5")
	formatted := m.FormatOID(oid)
	back, _ := m.ResolveOID(formatted)
	fmt.Printf("  ResolveOID  -> %s\n", oid)
	fmt.Printf("  FormatOID   -> %s\n", formatted)
	fmt.Printf("  round-trip  -> %s\n", back)

	// Query by name
	fmt.Println("\n=== By name ===")
	obj := m.Object("ifIndex")
	if obj != nil {
		fmt.Printf("%-20s %s\n", obj.Name(), obj.OID())
	}

	// Qualified lookup via Module().Object()
	fmt.Println("\n=== Module-scoped object ===")
	mod := m.Module("IF-MIB")
	if mod != nil {
		obj = mod.Object("ifDescr")
		if obj != nil {
			fmt.Printf("%-20s %s\n", obj.Name(), obj.OID())
		}
	}

	// OID-based lookup via NodeByOID().Object()
	fmt.Println("\n=== By OID ===")
	oid, _ = mib.ParseOID("1.3.6.1.2.1.2.2.1.3")
	node := m.NodeByOID(oid)
	if node != nil && node.Object() != nil {
		obj = node.Object()
		fmt.Printf("%-20s %s\n", obj.Name(), obj.OID())
	}

	// Exact OID lookup
	fmt.Println("\n=== NodeByOID (exact) ===")
	oid, _ = mib.ParseOID("1.3.6.1.2.1.2.2.1.1")
	node = m.NodeByOID(oid)
	if node != nil {
		fmt.Printf("%-20s %s  kind=%s\n", node.Name(), node.OID(), node.Kind())
	}

	// Longest prefix match (useful for instance OIDs like ifDescr.1)
	fmt.Println("\n=== LongestPrefixByOID ===")
	instanceOID, _ := mib.ParseOID("1.3.6.1.2.1.2.2.1.2.17")
	prefix := m.LongestPrefixByOID(instanceOID)
	if prefix != nil {
		fmt.Printf("%-20s %s  (matched from %s)\n", prefix.Name(), prefix.OID(), instanceOID)
	}

	// LookupInstance returns both the matched node and the instance suffix.
	// This is the standard pattern for processing SNMP varbinds.
	fmt.Println("\n=== LookupInstance ===")
	lookup := m.LookupInstance(instanceOID)
	fmt.Printf("node=%-16s suffix=%s\n", lookup.Node().Name(), lookup.Suffix())

	// Module-scoped type lookup
	fmt.Println("\n=== Module-scoped type ===")
	if mod != nil {
		typ := mod.Type("InterfaceIndex")
		if typ != nil {
			fmt.Printf("type %-16s base=%s  tc=%v\n", typ.Name(), typ.Base(), typ.IsTextualConvention())
		}
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

	// OID tree iteration (Go range-over-func)
	fmt.Println("\n=== Subtree iteration ===")
	entry := m.Node("ifEntry")
	if entry != nil {
		for nd := range entry.Subtree() {
			fmt.Printf("  %-24s %s  %s\n", nd.Name(), nd.OID(), nd.Kind())
		}
	}
}
