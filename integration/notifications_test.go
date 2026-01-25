package integration

import (
	"testing"

	"github.com/golangsnmp/gomib"
	"github.com/stretchr/testify/require"
)

// NotificationTestCase defines a test case for notification verification.
// Verify expected values with: snmptranslate -m <MODULE> -Td <name>
type NotificationTestCase struct {
	Name    string   // notification name
	Module  string   // module name
	Oid     string   // expected OID
	Objects []string // expected OBJECTS names (empty to skip check)
	NetSnmp string
}

// notificationTests contains notification test cases.
//
// Verified against net-snmp 5.9.4: snmptranslate -Td -m SYNTHETIC-MIB SYNTHETIC-MIB::<name>
var notificationTests = []NotificationTestCase{
	// syntheticConfigChange: OBJECTS { syntheticSystemDescription, syntheticConfigSerial }
	{Name: "syntheticConfigChange", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.14.0.1",
		Objects: []string{"syntheticSystemDescription", "syntheticConfigSerial"},
		NetSnmp: "OBJECTS { syntheticSystemDescription, syntheticConfigSerial }"},

	// syntheticFailure: OBJECTS { syntheticSimpleStatus }
	{Name: "syntheticFailure", Module: "SYNTHETIC-MIB", Oid: "1.3.6.1.2.1.999.14.0.2",
		Objects: []string{"syntheticSimpleStatus"},
		NetSnmp: "OBJECTS { syntheticSimpleStatus }"},
}

func TestNotifications(t *testing.T) {
	if len(notificationTests) == 0 {
		t.Skip("no notification test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range notificationTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			node := getNode(t, m, tc.Module, tc.Name)
			require.Equal(t, gomib.KindNotification, node.Kind, "should be a notification")

			got := node.OID().String()
			require.Equal(t, tc.Oid, got, "OID mismatch")

			if len(tc.Objects) > 0 {
				notif := node.Notif
				require.NotNil(t, notif, "should have a notification object")
				require.Len(t, notif.Objects, len(tc.Objects), "OBJECTS count mismatch")

				for i, expectedName := range tc.Objects {
					require.NotNil(t, notif.Objects[i], "OBJECTS[%d] should be resolved", i)
					require.Equal(t, expectedName, notif.Objects[i].Name, "OBJECTS[%d] name mismatch", i)
				}
			}
		})
	}
}

// TrapTestCase defines a test case for SMIv1 TRAP-TYPE verification.
type TrapTestCase struct {
	Name       string // trap name
	Module     string
	Enterprise string // enterprise OID name
	SpecificId int    // specific trap number
	NetSnmp    string
}

// trapTests contains SMIv1 trap test cases.
var trapTests = []TrapTestCase{
	// SMIv1 TRAP-TYPE examples would go here
}

func TestTraps(t *testing.T) {
	if len(trapTests) == 0 {
		t.Skip("no trap test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range trapTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			node := getNode(t, m, tc.Module, tc.Name)
			require.Equal(t, gomib.KindNotification, node.Kind, "trap should be notification kind")
			// Additional trap-specific assertions can be added
		})
	}
}
