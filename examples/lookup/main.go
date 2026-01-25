// Example: lookup - look up objects by name, OID, and qualified name.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golangsnmp/gomib"
)

func main() {
	source, err := gomib.DirTree(findCorpus())
	if err != nil {
		log.Fatalf("failed to open MIB directory: %v", err)
	}

	mib, err := gomib.Load(source)
	if err != nil {
		log.Fatalf("failed to load MIBs: %v", err)
	}

	// Look up by simple name
	fmt.Println("=== Lookup by name ===")
	obj := mib.Object("sysDescr")
	if obj != nil {
		fmt.Printf("sysDescr: OID=%s, Access=%s, Type=%s\n",
			obj.OID(), obj.Access, obj.Type.Name)
	}

	// Look up by qualified name (MODULE::name)
	fmt.Println("\n=== Lookup by qualified name ===")
	obj = mib.ObjectByQualified("IF-MIB::ifIndex")
	if obj != nil {
		fmt.Printf("IF-MIB::ifIndex: OID=%s, Kind=%s\n",
			obj.OID(), obj.Kind())
	}

	// Look up by OID string
	fmt.Println("\n=== Lookup by OID ===")
	node := mib.Node("1.3.6.1.2.1.1.1") // sysDescr
	if node != nil {
		fmt.Printf("1.3.6.1.2.1.1.1: Name=%s, Kind=%s\n",
			node.Name, node.Kind)
	}

	// FindNode handles multiple query formats
	fmt.Println("\n=== FindNode (flexible lookup) ===")
	queries := []string{
		"sysUpTime",           // simple name
		"IF-MIB::ifNumber",    // qualified name
		"1.3.6.1.2.1.2.2.1.1", // numeric OID
		".1.3.6.1.2.1.1.3",    // partial OID with leading dot
	}
	for _, q := range queries {
		n := mib.FindNode(q)
		if n != nil {
			fmt.Printf("  %q -> %s\n", q, n)
		} else {
			fmt.Printf("  %q -> not found\n", q)
		}
	}

	// Objects with same name from different modules
	fmt.Println("\n=== Multiple objects with same name ===")
	objs := mib.ObjectsByName("sysDescr")
	fmt.Printf("Found %d objects named 'sysDescr'\n", len(objs))
	for _, o := range objs {
		fmt.Printf("  %s::%s OID=%s\n", o.Module.Name, o.Name, o.OID())
	}
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
