// Example: walk - traverse the OID tree.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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

	// Walk entire tree and count node kinds
	fmt.Println("=== Node kind counts ===")
	counts := make(map[gomib.Kind]int)
	mib.Walk(func(n *gomib.Node) bool {
		counts[n.Kind]++
		return true
	})
	for kind, count := range counts {
		fmt.Printf("  %s: %d\n", kind, count)
	}

	// Navigate to a specific subtree
	fmt.Println("\n=== System subtree (1.3.6.1.2.1.1) ===")
	system := mib.Node("1.3.6.1.2.1.1")
	if system != nil {
		printTree(system, 0, 3)
	}

	// Walk from a specific node
	fmt.Println("\n=== Walk IF-MIB interfaces (max 10 objects) ===")
	ifMIB := mib.Node("1.3.6.1.2.1.2")
	if ifMIB != nil {
		count := 0
		ifMIB.Walk(func(n *gomib.Node) bool {
			if n.Object != nil {
				fmt.Printf("  %s (%s) - %s\n", n.Name, n.OID(), n.Kind)
				count++
			}
			return count < 10
		})
	}

	// Find all tables
	fmt.Println("\n=== Tables in IF-MIB ===")
	ifMod := mib.Module("IF-MIB")
	if ifMod != nil {
		for _, obj := range ifMod.Objects() {
			if obj.Kind() == gomib.KindTable {
				fmt.Printf("  %s (%s)\n", obj.Name, obj.OID())
			}
		}
	}
}

// printTree prints a subtree with indentation up to maxDepth levels.
func printTree(n *gomib.Node, depth, maxDepth int) {
	if depth > maxDepth {
		return
	}
	indent := strings.Repeat("  ", depth)
	name := n.Name
	if name == "" {
		name = fmt.Sprintf("(%d)", n.Arc())
	}
	fmt.Printf("%s%s [%s]\n", indent, name, n.Kind)
	for _, child := range n.Children() {
		printTree(child, depth+1, maxDepth)
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
