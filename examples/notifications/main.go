// Load an embedded module and inspect notifications, groups, compliances, and capabilities.
package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"strings"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/mib"
)

//go:embed mibs
var mibFS embed.FS

func main() {
	src := gomib.FS("embedded", mibFS)

	m, err := gomib.Load(context.Background(),
		gomib.WithSource(src),
		gomib.WithModules("EXAMPLE-FULL-MIB"),
	)
	if err != nil {
		log.Fatal(err)
	}

	notif := m.Notification("exStatusChange")
	if notif == nil {
		log.Fatal("exStatusChange not found")
	}
	fmt.Printf("Notification: %s\n", notif.Name())
	fmt.Printf("  OID:         %s\n", notif.OID())
	fmt.Printf("  Status:      %s\n", notif.Status())
	fmt.Printf("  Description: %s\n", notif.Description())
	fmt.Printf("  Objects:     %s\n", joinObjects(notif.Objects()))
	fmt.Println()

	group := m.Group("exScalarGroup")
	if group == nil {
		log.Fatal("exScalarGroup not found")
	}
	fmt.Printf("Group: %s\n", group.Name())
	fmt.Printf("  OID:                 %s\n", group.OID())
	fmt.Printf("  Notification group:  %v\n", group.IsNotificationGroup())
	fmt.Printf("  Members:             %s\n", joinNodes(group.Members()))
	fmt.Println()

	notifGroup := m.Group("exNotifGroup")
	if notifGroup == nil {
		log.Fatal("exNotifGroup not found")
	}
	fmt.Printf("Notification group: %s\n", notifGroup.Name())
	fmt.Printf("  Members: %s\n", joinNodes(notifGroup.Members()))
	fmt.Println()

	comp := m.Compliance("exBasicCompliance")
	if comp == nil {
		log.Fatal("exBasicCompliance not found")
	}
	fmt.Printf("Compliance: %s\n", comp.Name())
	fmt.Printf("  OID:      %s\n", comp.OID())
	for _, mod := range comp.Modules() {
		name := mod.ModuleName
		if name == "" {
			name = "(current module)"
		}
		fmt.Printf("  MODULE %s\n", name)
		fmt.Printf("    mandatory groups: %s\n", strings.Join(mod.MandatoryGroups, ", "))
		for _, grp := range mod.Groups {
			fmt.Printf("    GROUP %s\n", grp.Group)
		}
	}
	fmt.Println()

	cap := m.Capability("exAgentCaps")
	if cap == nil {
		log.Fatal("exAgentCaps not found")
	}
	fmt.Printf("Capability: %s\n", cap.Name())
	fmt.Printf("  OID:              %s\n", cap.OID())
	fmt.Printf("  Product release:  %s\n", cap.ProductRelease())
	for _, sup := range cap.Supports() {
		fmt.Printf("  SUPPORTS %s\n", sup.ModuleName)
		fmt.Printf("    includes: %s\n", strings.Join(sup.Includes, ", "))
		for i := range sup.ObjectVariations {
			v := &sup.ObjectVariations[i]
			access := ""
			if v.Access != nil {
				access = v.Access.String()
			}
			fmt.Printf("    object variation %-18s access=%s\n", v.Object, access)
		}
		for _, v := range sup.NotificationVariations {
			access := ""
			if v.Access != nil {
				access = v.Access.String()
			}
			fmt.Printf("    notification variation %-11s access=%s\n", v.Notification, access)
		}
	}
}

func joinObjects(objs []*mib.Object) string {
	names := make([]string, 0, len(objs))
	for _, obj := range objs {
		names = append(names, obj.Name())
	}
	return strings.Join(names, ", ")
}

func joinNodes(nodes []*mib.Node) string {
	names := make([]string, 0, len(nodes))
	for _, node := range nodes {
		names = append(names, node.Name())
	}
	return strings.Join(names, ", ")
}
