// Example: notifications - explore SNMP notifications (traps).
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

	fmt.Printf("Loaded %d notifications total\n\n", mib.NotificationCount())

	// Look up a specific notification
	fmt.Println("=== SNMPv2-MIB notifications ===")
	snmpMod := mib.Module("SNMPv2-MIB")
	if snmpMod != nil {
		for _, notif := range snmpMod.Notifications() {
			fmt.Printf("%s (%s)\n", notif.Name, notif.OID())
			if len(notif.Objects) > 0 {
				fmt.Printf("  Objects:\n")
				for _, obj := range notif.Objects {
					fmt.Printf("    - %s\n", obj.Name)
				}
			}
		}
	}

	// Find notification by name
	fmt.Println("\n=== Lookup notification by name ===")
	coldStart := mib.Notification("coldStart")
	if coldStart != nil {
		fmt.Printf("coldStart:\n")
		fmt.Printf("  OID: %s\n", coldStart.OID())
		fmt.Printf("  Module: %s\n", coldStart.Module.Name)
		fmt.Printf("  Status: %s\n", coldStart.Status)
		if coldStart.Description != "" {
			desc := coldStart.Description
			if len(desc) > 100 {
				desc = desc[:97] + "..."
			}
			fmt.Printf("  Description: %s\n", desc)
		}
	}

	// List notifications by module
	fmt.Println("\n=== Notifications per module (top 5) ===")
	modCounts := make(map[string]int)
	for _, notif := range mib.Notifications() {
		if notif.Module != nil {
			modCounts[notif.Module.Name]++
		}
	}

	type modCount struct {
		name  string
		count int
	}
	var sorted []modCount
	for name, count := range modCounts {
		sorted = append(sorted, modCount{name, count})
	}
	// Simple bubble sort for top 5
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].count > sorted[i].count {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	for i := 0; i < min(5, len(sorted)); i++ {
		fmt.Printf("  %s: %d notifications\n", sorted[i].name, sorted[i].count)
	}

	// Find notifications with many objects
	fmt.Println("\n=== Notifications with most objects ===")
	type notifInfo struct {
		name    string
		module  string
		objects int
	}
	var withObjects []notifInfo
	for _, notif := range mib.Notifications() {
		if len(notif.Objects) > 2 {
			withObjects = append(withObjects, notifInfo{
				name:    notif.Name,
				module:  notif.Module.Name,
				objects: len(notif.Objects),
			})
		}
	}
	// Sort by object count
	for i := 0; i < len(withObjects)-1; i++ {
		for j := i + 1; j < len(withObjects); j++ {
			if withObjects[j].objects > withObjects[i].objects {
				withObjects[i], withObjects[j] = withObjects[j], withObjects[i]
			}
		}
	}
	for i := 0; i < min(5, len(withObjects)); i++ {
		n := withObjects[i]
		fmt.Printf("  %s::%s (%d objects)\n", n.module, n.name, n.objects)
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
