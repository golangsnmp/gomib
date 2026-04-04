package module

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestLower_ObjectIdentity(t *testing.T) {
	source := `OID-IDENT-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-IDENTITY, enterprises
        FROM SNMPv2-SMI;

oidIdentTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testIdentity OBJECT-IDENTITY
    STATUS current
    DESCRIPTION "A test identity node"
    REFERENCE "RFC 9999"
    ::= { oidIdentTest 1 }

END
`

	mod := lowerModule(t, source)
	oi := requireDef[*ObjectIdentity](t, mod, "testIdentity")

	testutil.Equal(t, "testIdentity", oi.Name, "Name")
	testutil.Equal(t, types.StatusCurrent, oi.Status, "Status")
	testutil.Equal(t, "A test identity node", oi.Description, "Description")
	testutil.Equal(t, "RFC 9999", oi.Reference, "Reference")
	testutil.True(t, len(oi.Oid.Components) >= 2, "expected at least 2 OID components, got %d", len(oi.Oid.Components))

	nameComp, ok := oi.Oid.Components[0].(*OidComponentName)
	testutil.True(t, ok, "Oid[0] should be OidComponentName")
	testutil.Equal(t, "oidIdentTest", nameComp.NameValue, "Oid[0] name")

	numComp, ok := oi.Oid.Components[1].(*OidComponentNumber)
	testutil.True(t, ok, "Oid[1] should be OidComponentNumber")
	testutil.Equal(t, uint32(1), numComp.Value, "Oid[1] value")
}

func TestLower_NotificationType(t *testing.T) {
	source := `NOTIF-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, NOTIFICATION-TYPE, Integer32, enterprises
        FROM SNMPv2-SMI;

notifTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testObj OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "Test"
    ::= { notifTest 1 }

testNotification NOTIFICATION-TYPE
    OBJECTS { testObj }
    STATUS current
    DESCRIPTION "A test notification"
    REFERENCE "RFC 9999"
    ::= { notifTest 2 }

END
`

	mod := lowerModule(t, source)
	notif := requireDef[*Notification](t, mod, "testNotification")

	testutil.Equal(t, "testNotification", notif.Name, "Name")
	testutil.SliceEqual(t, []string{"testObj"}, notif.Objects, "Objects")
	testutil.Equal(t, types.StatusCurrent, notif.Status, "Status")
	testutil.Equal(t, "A test notification", notif.Description, "Description")
	testutil.Equal(t, "RFC 9999", notif.Reference, "Reference")
	testutil.Nil(t, notif.TrapInfo, "TrapInfo should be nil for NOTIFICATION-TYPE")
	testutil.NotNil(t, notif.Oid, "Oid should be non-nil for NOTIFICATION-TYPE")
}

func TestLower_TrapType(t *testing.T) {
	source := `TRAP-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    enterprises
        FROM RFC1155-SMI;

trapTestEnterprise OBJECT IDENTIFIER ::= { enterprises 99999 }

testTrap TRAP-TYPE
    ENTERPRISE trapTestEnterprise
    VARIABLES { }
    DESCRIPTION "A test trap"
    ::= 7

END
`

	mod := lowerModule(t, source)
	notif := requireDef[*Notification](t, mod, "testTrap")

	testutil.Equal(t, "testTrap", notif.Name, "Name")
	testutil.Equal(t, types.StatusCurrent, notif.Status, "Status (TRAP-TYPE defaults to current)")
	testutil.Equal(t, "A test trap", notif.Description, "Description")
	testutil.NotNil(t, notif.TrapInfo, "TrapInfo should be non-nil for TRAP-TYPE")
	testutil.Equal(t, "trapTestEnterprise", notif.TrapInfo.Enterprise, "TrapInfo.Enterprise")
	testutil.Equal(t, uint32(7), notif.TrapInfo.TrapNumber, "TrapInfo.TrapNumber")
	testutil.Nil(t, notif.Oid, "Oid should be nil for TRAP-TYPE")
	testutil.Len(t, notif.Objects, 0, "Objects should be empty for empty VARIABLES clause")
}

func TestLower_TextualConvention(t *testing.T) {
	source := `TC-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, Integer32, enterprises
        FROM SNMPv2-SMI
    TEXTUAL-CONVENTION
        FROM SNMPv2-TC;

tcTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

MyString ::= TEXTUAL-CONVENTION
    DISPLAY-HINT "255a"
    STATUS current
    DESCRIPTION "A custom string type"
    REFERENCE "RFC 9999"
    SYNTAX OCTET STRING (SIZE (0..255))

END
`

	mod := lowerModule(t, source)
	td := requireDef[*TypeDef](t, mod, "MyString")

	testutil.Equal(t, "MyString", td.Name, "Name")
	testutil.True(t, td.IsTextualConvention, "IsTextualConvention")
	testutil.Equal(t, "255a", td.DisplayHint, "DisplayHint")
	testutil.Equal(t, types.StatusCurrent, td.Status, "Status")
	testutil.Equal(t, "A custom string type", td.Description, "Description")
	testutil.Equal(t, "RFC 9999", td.Reference, "Reference")
	testutil.Nil(t, td.BaseType, "BaseType should be nil")

	_, ok := td.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, ok, "Syntax should be *TypeSyntaxConstrained, got %T", td.Syntax)
}

func TestLower_TypeAssignment(t *testing.T) {
	source := `TYPE-TEST-MIB DEFINITIONS ::= BEGIN

MyInt ::= INTEGER (0..100)

END
`

	mod := lowerModule(t, source)
	td := requireDef[*TypeDef](t, mod, "MyInt")

	testutil.Equal(t, "MyInt", td.Name, "Name")
	testutil.False(t, td.IsTextualConvention, "IsTextualConvention")
	testutil.Equal(t, "", td.DisplayHint, "DisplayHint")
	testutil.Equal(t, types.StatusCurrent, td.Status, "Status (default for type assignments)")
	testutil.Equal(t, "", td.Description, "Description")
	testutil.Equal(t, "", td.Reference, "Reference")

	_, ok := td.Syntax.(*TypeSyntaxConstrained)
	testutil.True(t, ok, "Syntax should be *TypeSyntaxConstrained, got %T", td.Syntax)
}

func TestLower_ValueAssignment(t *testing.T) {
	source := `VAL-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    enterprises
        FROM RFC1155-SMI;

testOid OBJECT IDENTIFIER ::= { enterprises 99999 }

END
`

	mod := lowerModule(t, source)
	va := requireDef[*ValueAssignment](t, mod, "testOid")

	testutil.Equal(t, "testOid", va.Name, "Name")
	testutil.Len(t, va.Oid.Components, 2, "Oid component count")

	nameComp, ok := va.Oid.Components[0].(*OidComponentName)
	testutil.True(t, ok, "Oid[0] should be OidComponentName")
	testutil.Equal(t, "enterprises", nameComp.NameValue, "Oid[0] name")

	numComp, ok := va.Oid.Components[1].(*OidComponentNumber)
	testutil.True(t, ok, "Oid[1] should be OidComponentNumber")
	testutil.Equal(t, uint32(99999), numComp.Value, "Oid[1] value")
}

func TestLower_ObjectGroup(t *testing.T) {
	source := `GROUP-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises
        FROM SNMPv2-SMI
    OBJECT-GROUP
        FROM SNMPv2-CONF;

groupTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testGroup OBJECT-GROUP
    OBJECTS { obj1, obj2, obj3 }
    STATUS current
    DESCRIPTION "Test group"
    REFERENCE "RFC 9999"
    ::= { groupTest 1 }

END
`

	mod := lowerModule(t, source)
	og := requireDef[*ObjectGroup](t, mod, "testGroup")

	testutil.Equal(t, "testGroup", og.Name, "Name")
	testutil.SliceEqual(t, []string{"obj1", "obj2", "obj3"}, og.Objects, "Objects")
	testutil.Equal(t, types.StatusCurrent, og.Status, "Status")
	testutil.Equal(t, "Test group", og.Description, "Description")
	testutil.Equal(t, "RFC 9999", og.Reference, "Reference")
}

func TestLower_NotificationGroup(t *testing.T) {
	source := `NOTIFGROUP-TEST-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, enterprises
        FROM SNMPv2-SMI
    NOTIFICATION-GROUP
        FROM SNMPv2-CONF;

notifGroupTest MODULE-IDENTITY
    LAST-UPDATED "200001010000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Test"
    REVISION "200001010000Z"
    DESCRIPTION "Test"
    ::= { enterprises 99999 }

testNotifGroup NOTIFICATION-GROUP
    NOTIFICATIONS { notif1, notif2 }
    STATUS deprecated
    DESCRIPTION "Test notification group"
    ::= { notifGroupTest 1 }

END
`

	mod := lowerModule(t, source)
	ng := requireDef[*NotificationGroup](t, mod, "testNotifGroup")

	testutil.Equal(t, "testNotifGroup", ng.Name, "Name")
	testutil.SliceEqual(t, []string{"notif1", "notif2"}, ng.Notifications, "Notifications")
	testutil.Equal(t, types.StatusDeprecated, ng.Status, "Status")
	testutil.Equal(t, "Test notification group", ng.Description, "Description")
	testutil.Equal(t, "", ng.Reference, "Reference (absent)")
}
