package main

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/golangsnmp/gomib"
	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/mib"
)

func TestExportRangesPreservesEndpointSemantics(t *testing.T) {
	bounds := exportRanges([]mib.Range{
		{Min: mib.RangeBound{Kind: mib.RangeBoundMin}, Max: mib.RangeBound{Kind: mib.RangeBoundMax}},
		{Min: mib.NewUnsignedRangeBound(0), Max: mib.NewUnsignedRangeBound(^uint64(0))},
	})
	if bounds[0].Min != "MIN" || bounds[0].Max != "MAX" || bounds[0].MinKind != "min" || bounds[0].MaxKind != "max" {
		t.Fatalf("symbolic export = %#v", bounds[0])
	}
	if bounds[1].Max != "18446744073709551615" || bounds[1].MaxKind != "unsigned" {
		t.Fatalf("unsigned export = %#v", bounds[1])
	}
}

func TestExportRangesPreservesRawEndpoints(t *testing.T) {
	got := exportRanges([]mib.Range{{Min: mib.NewRawRangeBound("'0G'H"), Max: mib.NewRawRangeBound("'10000000000000000'H")}})
	if len(got) != 1 {
		t.Fatalf("exported ranges = %d, want 1", len(got))
	}
	if got[0].Min != "'0G'H" || got[0].Max != "'10000000000000000'H" {
		t.Errorf("exported range = %#v, want raw endpoints", got[0])
	}
}

func TestExportTypeBackedIndexUsesEffectiveConstraints(t *testing.T) {
	m := model.NewMib()
	parent := model.NewType("ParentRange")
	model.SetTypeBase(parent, model.BaseInteger32)
	model.SetTypeRanges(parent, []model.Range{{
		Min: model.NewSignedRangeBound(1),
		Max: model.NewSignedRangeBound(10),
	}})
	model.AddMibType(m, parent)

	inherited := model.NewType("InheritedRange")
	model.SetTypeParent(inherited, parent)
	model.AddMibType(m, inherited)

	disjoint := model.NewType("DisjointRange")
	model.SetTypeParent(disjoint, parent)
	model.SetTypeRanges(disjoint, []model.Range{{
		Min: model.NewSignedRangeBound(20),
		Max: model.NewSignedRangeBound(30),
	}})
	model.AddMibType(m, disjoint)

	got := exportIndexEntries(m, []mib.IndexEntry{
		{TypeName: "InheritedRange"},
		{TypeName: "DisjointRange"},
	})
	inheritedConstraints := got[0].Syntax.Constraints
	if !inheritedConstraints.RangesConstrained || len(inheritedConstraints.Ranges) != 1 ||
		inheritedConstraints.Ranges[0].Min != "1" || inheritedConstraints.Ranges[0].Max != "10" {
		t.Errorf("inherited index constraints = %#v", inheritedConstraints)
	}
	disjointConstraints := got[1].Syntax.Constraints
	if !disjointConstraints.RangesConstrained || len(disjointConstraints.Ranges) != 0 {
		t.Errorf("disjoint index constraints = %#v, want constrained empty range", disjointConstraints)
	}
}

func TestScopedExportRefsDoNotFallBackGlobally(t *testing.T) {
	m := loadScopedExportTestMIB(t)
	export := buildExportPayload(m, "normal")

	var compliance *ExportCompliance
	for i := range export.Compliances {
		if export.Compliances[i].Name == "scopedCompliance" {
			compliance = &export.Compliances[i]
			break
		}
	}
	if compliance == nil {
		t.Fatal("scopedCompliance not exported")
		return
	}
	if len(compliance.Modules) != 1 {
		t.Fatalf("compliance modules = %d, want 1", len(compliance.Modules))
	}
	cm := compliance.Modules[0]
	assertUnresolvedScopedRef(t, cm.MandatoryGroups[0], "collidingMandatory")
	assertUnresolvedScopedRef(t, cm.Groups[0].Group, "collidingOptional")
	assertUnresolvedScopedRef(t, cm.Objects[0].Object, "collidingObject")

	var capability *ExportCapability
	for i := range export.Capabilities {
		if export.Capabilities[i].Name == "scopedCapability" {
			capability = &export.Capabilities[i]
			break
		}
	}
	if capability == nil {
		t.Fatal("scopedCapability not exported")
		return
	}
	if len(capability.Supports) != 1 {
		t.Fatalf("capability supports = %d, want 1", len(capability.Supports))
	}
	supports := capability.Supports[0]
	assertUnresolvedScopedRef(t, supports.Includes[0], "collidingInclude")
	assertUnresolvedScopedRef(t, supports.ObjectVariations[0].Object, "collidingVariationObject")
	assertUnresolvedScopedRef(t, supports.ObjectVariations[0].CreationRequires[0], "collidingCreation")
	assertUnresolvedScopedRef(t, supports.NotificationVariations[0].Notification, "collidingNotification")
}

func TestResolveNamedRefPreservesUnscopedLookup(t *testing.T) {
	m := loadScopedExportTestMIB(t)

	ref := resolveNamedRef(m, "", "collidingObject")
	if ref.Module != "COLLISION-MIB" || ref.Name != "collidingObject" || ref.OID == "" {
		t.Fatalf("unscoped ref = %+v, want resolved COLLISION-MIB::collidingObject", ref)
	}
}

func assertUnresolvedScopedRef(t *testing.T, ref ExportObjRef, name string) {
	t.Helper()
	const module = "TARGET-MIB"
	if ref.Module != module || ref.Name != name || ref.OID != "" {
		t.Errorf("ref = %+v, want unresolved %s::%s", ref, module, name)
	}
}

func loadScopedExportTestMIB(t *testing.T) *mib.Mib {
	t.Helper()

	files := fstest.MapFS{
		"COLLISION-MIB.mib": &fstest.MapFile{Data: []byte(collisionMIB)},
		"TARGET-MIB.mib":    &fstest.MapFile{Data: []byte(targetMIB)},
		"SCOPED-MIB.mib":    &fstest.MapFile{Data: []byte(scopedMIB)},
	}
	cfg := mib.DefaultConfig()
	cfg.FailAt = mib.SeverityFatal
	m, err := gomib.Load(
		context.Background(),
		gomib.WithSource(gomib.FS("scoped-export-test", files)),
		gomib.WithDiagnosticConfig(cfg),
	)
	if err != nil {
		t.Fatalf("load test MIBs: %v", err)
	}
	return m
}

const collisionMIB = `COLLISION-MIB DEFINITIONS ::= BEGIN
IMPORTS
    MODULE-IDENTITY, OBJECT-TYPE, NOTIFICATION-TYPE, Integer32, enterprises
        FROM SNMPv2-SMI
    OBJECT-GROUP
        FROM SNMPv2-CONF;

collisionMIB MODULE-IDENTITY
    LAST-UPDATED "202603190000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Defines symbols that collide with missing scoped references."
    ::= { enterprises 99990 }

collidingObject OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "Collision."
    ::= { collisionMIB 1 }

collidingVariationObject OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-create
    STATUS current
    DESCRIPTION "Collision."
    ::= { collisionMIB 2 }

collidingCreation OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-create
    STATUS current
    DESCRIPTION "Collision."
    ::= { collisionMIB 3 }

collidingMandatory OBJECT-GROUP
    OBJECTS { collidingObject }
    STATUS current
    DESCRIPTION "Collision."
    ::= { collisionMIB 4 }

collidingOptional OBJECT-GROUP
    OBJECTS { collidingObject }
    STATUS current
    DESCRIPTION "Collision."
    ::= { collisionMIB 5 }

collidingInclude OBJECT-GROUP
    OBJECTS { collidingObject }
    STATUS current
    DESCRIPTION "Collision."
    ::= { collisionMIB 6 }

collidingNotification NOTIFICATION-TYPE
    OBJECTS { collidingObject }
    STATUS current
    DESCRIPTION "Collision."
    ::= { collisionMIB 0 1 }
END
`

const targetMIB = `TARGET-MIB DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY, enterprises FROM SNMPv2-SMI;

targetMIB MODULE-IDENTITY
    LAST-UPDATED "202603190000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "Intentionally lacks the referenced symbols."
    ::= { enterprises 99991 }
END
`

const scopedMIB = `SCOPED-MIB DEFINITIONS ::= BEGIN
IMPORTS
    MODULE-IDENTITY, Integer32, enterprises
        FROM SNMPv2-SMI
    MODULE-COMPLIANCE, AGENT-CAPABILITIES
        FROM SNMPv2-CONF;

scopedMIB MODULE-IDENTITY
    LAST-UPDATED "202603190000Z"
    ORGANIZATION "Test"
    CONTACT-INFO "Test"
    DESCRIPTION "References missing symbols in another module."
    ::= { enterprises 99992 }

scopedCompliance MODULE-COMPLIANCE
    STATUS current
    DESCRIPTION "Scoped references."
    MODULE TARGET-MIB
        MANDATORY-GROUPS { collidingMandatory }
        GROUP collidingOptional
            DESCRIPTION "Optional collision."
        OBJECT collidingObject
            MIN-ACCESS read-only
            DESCRIPTION "Object collision."
    ::= { scopedMIB 1 }

scopedCapability AGENT-CAPABILITIES
    PRODUCT-RELEASE "test"
    STATUS current
    DESCRIPTION "Scoped references."
    SUPPORTS TARGET-MIB
        INCLUDES { collidingInclude }
        VARIATION collidingVariationObject
            SYNTAX Integer32
            CREATION-REQUIRES { collidingCreation }
            DESCRIPTION "Object variation collision."
        VARIATION collidingNotification
            ACCESS not-implemented
            DESCRIPTION "Notification variation collision."
    ::= { scopedMIB 2 }
END
`
