package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestCheckGroupMembership_NoGroups(t *testing.T) {
	// Module with no groups should produce no group-membership diagnostics.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		},
	}
	addDefs(mod, []module.Definition{
		&module.ModuleIdentity{
			DefBase: module.DefBase{Name: "testMIB"},
			Oid:     testOid("enterprises", 99999),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testScalar"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
			Access:  types.AccessReadOnly,
			Status:  types.StatusCurrent,
			Oid:     testOid("testMIB", 1),
		},
	})
	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagGroupMembership)
}

func TestCheckGroupMembership_AllCovered(t *testing.T) {
	// Module with groups that cover all accessible objects - no diagnostic.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
	}
	addDefs(mod, []module.Definition{
		&module.ModuleIdentity{
			DefBase: module.DefBase{Name: "testMIB"},
			Oid:     testOid("enterprises", 99999),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testScalar"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Access:  types.AccessReadOnly,
			Status:  types.StatusCurrent,
			Oid:     testOid("testMIB", 1),
		},
		&module.Notification{
			DefBase: module.DefBase{Name: "testNotif"},
			Status:  types.StatusCurrent,
			Oid:     &module.OidAssignment{Components: testOid("testMIB", 2).Components, Span: types.Span{}},
		},
		&module.ObjectGroup{
			DefBase: module.DefBase{Name: "testObjGroup"},
			Objects: []string{"testScalar"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testMIB", 3),
		},
		&module.NotificationGroup{
			DefBase:       module.DefBase{Name: "testNotifGroup"},
			Notifications: []string{"testNotif"},
			Status:        types.StatusCurrent,
			Oid:           testOid("testMIB", 4),
		},
	})
	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagGroupMembership)
}

func TestCheckGroupMembership_MissingObject(t *testing.T) {
	// Module with OBJECT-GROUP that does not cover testScalar2.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
	}
	addDefs(mod, []module.Definition{
		&module.ModuleIdentity{
			DefBase: module.DefBase{Name: "testMIB"},
			Oid:     testOid("enterprises", 99999),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testScalar1"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Access:  types.AccessReadOnly,
			Status:  types.StatusCurrent,
			Oid:     testOid("testMIB", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testScalar2"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Access:  types.AccessReadWrite,
			Status:  types.StatusCurrent,
			Oid:     testOid("testMIB", 2),
		},
		&module.ObjectGroup{
			DefBase: module.DefBase{Name: "testObjGroup"},
			Objects: []string{"testScalar1"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testMIB", 3),
		},
	})
	m := resolveStrict(mod)
	d := hasDiag(t, m.Diagnostics(), types.DiagGroupMembership)
	testutil.Contains(t, d.Message, "testScalar2", "diagnostic message")
}

func TestCheckGroupMembership_NotAccessibleExcluded(t *testing.T) {
	// not-accessible objects should not trigger group-membership.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
	}
	addDefs(mod, []module.Definition{
		&module.ModuleIdentity{
			DefBase: module.DefBase{Name: "testMIB"},
			Oid:     testOid("enterprises", 99999),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testTable"},
			Syntax:  &module.TypeSyntaxSequenceOf{EntryType: "TestEntry"},
			Access:  types.AccessNotAccessible,
			Status:  types.StatusCurrent,
			Oid:     testOid("testMIB", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testEntry"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "TestEntry"},
			Access:  types.AccessNotAccessible,
			Status:  types.StatusCurrent,
			Index:   []module.IndexItem{{Object: "testIndex"}},
			Oid:     testOid("testTable", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testIndex"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Access:  types.AccessNotAccessible,
			Status:  types.StatusCurrent,
			Oid:     testOid("testEntry", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testValue"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Access:  types.AccessReadOnly,
			Status:  types.StatusCurrent,
			Oid:     testOid("testEntry", 2),
		},
		&module.ObjectGroup{
			DefBase: module.DefBase{Name: "testGroup"},
			Objects: []string{"testValue"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testMIB", 2),
		},
	})
	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagGroupMembership)
}

func TestCheckGroupMembership_MissingNotification(t *testing.T) {
	// Module with NOTIFICATION-GROUP that doesn't cover testNotif2.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		},
	}
	addDefs(mod, []module.Definition{
		&module.ModuleIdentity{
			DefBase: module.DefBase{Name: "testMIB"},
			Oid:     testOid("enterprises", 99999),
		},
		&module.Notification{
			DefBase: module.DefBase{Name: "testNotif1"},
			Status:  types.StatusCurrent,
			Oid:     &module.OidAssignment{Components: testOid("testMIB", 1).Components, Span: types.Span{}},
		},
		&module.Notification{
			DefBase: module.DefBase{Name: "testNotif2"},
			Status:  types.StatusCurrent,
			Oid:     &module.OidAssignment{Components: testOid("testMIB", 2).Components, Span: types.Span{}},
		},
		&module.NotificationGroup{
			DefBase:       module.DefBase{Name: "testNotifGroup"},
			Notifications: []string{"testNotif1"},
			Status:        types.StatusCurrent,
			Oid:           testOid("testMIB", 3),
		},
	})
	m := resolveStrict(mod)
	d := hasDiag(t, m.Diagnostics(), types.DiagGroupMembership)
	testutil.Contains(t, d.Message, "testNotif2", "diagnostic message")
}

func TestCheckComplianceStatus_GroupStatus(t *testing.T) {
	t.Run("obsolete group in current compliance emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "obsGroup"},
				Objects: []string{"someObj"},
				Status:  types.StatusObsolete,
			},
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusCurrent,
				Modules: []module.ComplianceModule{
					{MandatoryGroups: []string{"obsGroup"}},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		model.SetNodeName(grpNode, "obsGroup")
		ctx.registerModuleNodeSymbol(mod, "obsGroup", grpNode)

		objNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(objNode, "someObj")
		model.SetNodeKind(objNode, types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "someObj", objNode)
		obj := model.NewObject("someObj")
		model.SetObjectStatus(obj, types.StatusObsolete)
		model.SetNodeObject(objNode, obj)

		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedGroups(ctx)
		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagComplianceGroupStatus)
	})

	t.Run("current group in current compliance no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "curGroup"},
				Objects: []string{"someObj"},
				Status:  types.StatusCurrent,
			},
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusCurrent,
				Modules: []module.ComplianceModule{
					{MandatoryGroups: []string{"curGroup"}},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		model.SetNodeName(grpNode, "curGroup")
		ctx.registerModuleNodeSymbol(mod, "curGroup", grpNode)

		objNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(objNode, "someObj")
		model.SetNodeKind(objNode, types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "someObj", objNode)
		obj := model.NewObject("someObj")
		model.SetObjectStatus(obj, types.StatusCurrent)
		model.SetNodeObject(objNode, obj)

		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedGroups(ctx)
		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagComplianceGroupStatus)
	})

	t.Run("optional GROUP clause with obsolete group emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "optGroup"},
				Objects: []string{"someObj"},
				Status:  types.StatusObsolete,
			},
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusCurrent,
				Modules: []module.ComplianceModule{
					{Groups: []module.ComplianceGroup{{Group: "optGroup"}}},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		model.SetNodeName(grpNode, "optGroup")
		ctx.registerModuleNodeSymbol(mod, "optGroup", grpNode)

		objNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(objNode, "someObj")
		model.SetNodeKind(objNode, types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "someObj", objNode)
		obj := model.NewObject("someObj")
		model.SetObjectStatus(obj, types.StatusObsolete)
		model.SetNodeObject(objNode, obj)

		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedGroups(ctx)
		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagComplianceGroupStatus)
	})

	t.Run("smiv1 compliance status skips check", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "obsGroup"},
				Objects: []string{"someObj"},
				Status:  types.StatusObsolete,
			},
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusMandatory,
				Modules: []module.ComplianceModule{
					{MandatoryGroups: []string{"obsGroup"}},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		model.SetNodeName(grpNode, "obsGroup")
		ctx.registerModuleNodeSymbol(mod, "obsGroup", grpNode)

		objNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(objNode, "someObj")
		model.SetNodeKind(objNode, types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "someObj", objNode)
		obj := model.NewObject("someObj")
		model.SetObjectStatus(obj, types.StatusObsolete)
		model.SetNodeObject(objNode, obj)

		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedGroups(ctx)
		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagComplianceGroupStatus)
	})
}

func TestCheckComplianceStatus_ObjectStatus(t *testing.T) {
	t.Run("obsolete object in current compliance emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusCurrent,
				Modules: []module.ComplianceModule{
					{Objects: []module.ComplianceObject{{Object: "obsObj"}}},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		objNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(objNode, "obsObj")
		model.SetNodeKind(objNode, types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "obsObj", objNode)
		obj := model.NewObject("obsObj")
		model.SetObjectStatus(obj, types.StatusObsolete)
		model.SetNodeObject(objNode, obj)

		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagComplianceObjectStatus)
	})

	t.Run("current object in current compliance no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusCurrent,
				Modules: []module.ComplianceModule{
					{Objects: []module.ComplianceObject{{Object: "curObj"}}},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		objNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(objNode, "curObj")
		model.SetNodeKind(objNode, types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "curObj", objNode)
		obj := model.NewObject("curObj")
		model.SetObjectStatus(obj, types.StatusCurrent)
		model.SetNodeObject(objNode, obj)

		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagComplianceObjectStatus)
	})

	t.Run("deprecated object in deprecated compliance no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusDeprecated,
				Modules: []module.ComplianceModule{
					{Objects: []module.ComplianceObject{{Object: "depObj"}}},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		objNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(objNode, "depObj")
		model.SetNodeKind(objNode, types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "depObj", objNode)
		obj := model.NewObject("depObj")
		model.SetObjectStatus(obj, types.StatusDeprecated)
		model.SetNodeObject(objNode, obj)

		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagComplianceObjectStatus)
	})

	t.Run("smiv1 object status skips check", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusCurrent,
				Modules: []module.ComplianceModule{
					{Objects: []module.ComplianceObject{{Object: "mandObj"}}},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		objNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(objNode, "mandObj")
		model.SetNodeKind(objNode, types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "mandObj", objNode)
		obj := model.NewObject("mandObj")
		model.SetObjectStatus(obj, types.StatusMandatory)
		model.SetNodeObject(objNode, obj)

		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagComplianceObjectStatus)
	})
}

func TestCheckComplianceStructure(t *testing.T) {
	t.Run("group both mandatory and optional emits compliance-group-invalid", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "sharedGroup"},
				Objects: []string{"someObj"},
				Status:  types.StatusCurrent,
			},
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusCurrent,
				Modules: []module.ComplianceModule{
					{
						MandatoryGroups: []string{"sharedGroup"},
						Groups:          []module.ComplianceGroup{{Group: "sharedGroup"}},
					},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()
		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagComplianceGroupInvalid)
	})

	t.Run("duplicate refinement emits refinement-exists", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusCurrent,
				Modules: []module.ComplianceModule{
					{
						Objects: []module.ComplianceObject{
							{Object: "dupObj"},
							{Object: "dupObj"},
						},
					},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()
		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagRefinementExists)
	})

	t.Run("duplicate optional group emits optional-group-exists", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusCurrent,
				Modules: []module.ComplianceModule{
					{
						Groups: []module.ComplianceGroup{
							{Group: "dupGroup"},
							{Group: "dupGroup"},
						},
					},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()
		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagOptionalGroupExists)
	})

	t.Run("refinement not in any group emits refinement-not-listed", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "myGroup"},
				Objects: []string{"memberObj"},
				Status:  types.StatusCurrent,
			},
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusCurrent,
				Modules: []module.ComplianceModule{
					{
						MandatoryGroups: []string{"myGroup"},
						Objects:         []module.ComplianceObject{{Object: "unlistedObj"}},
					},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		model.SetNodeName(grpNode, "myGroup")
		ctx.registerModuleNodeSymbol(mod, "myGroup", grpNode)

		memberNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(memberNode, "memberObj")
		model.SetNodeKind(memberNode, types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "memberObj", memberNode)
		memberObj := model.NewObject("memberObj")
		model.SetObjectStatus(memberObj, types.StatusCurrent)
		model.SetNodeObject(memberNode, memberObj)

		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedGroups(ctx)
		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagRefinementNotListed)
	})

	t.Run("refinement in mandatory group no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "myGroup"},
				Objects: []string{"listedObj"},
				Status:  types.StatusCurrent,
			},
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusCurrent,
				Modules: []module.ComplianceModule{
					{
						MandatoryGroups: []string{"myGroup"},
						Objects:         []module.ComplianceObject{{Object: "listedObj"}},
					},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		model.SetNodeName(grpNode, "myGroup")
		ctx.registerModuleNodeSymbol(mod, "myGroup", grpNode)

		memberNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(memberNode, "listedObj")
		model.SetNodeKind(memberNode, types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "listedObj", memberNode)
		memberObj := model.NewObject("listedObj")
		model.SetObjectStatus(memberObj, types.StatusCurrent)
		model.SetNodeObject(memberNode, memberObj)

		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedGroups(ctx)
		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagRefinementNotListed)
	})

	t.Run("no duplicates no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Status:  types.StatusCurrent,
				Modules: []module.ComplianceModule{
					{
						MandatoryGroups: []string{"groupA"},
						Groups:          []module.ComplianceGroup{{Group: "groupB"}},
						Objects:         []module.ComplianceObject{{Object: "objA"}, {Object: "objB"}},
					},
				},
				Oid: testOid("testNode", 10),
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()
		compNode := buildOIDPath(root, 1, 10)
		model.SetNodeName(compNode, "testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagComplianceGroupInvalid, types.DiagRefinementExists, types.DiagOptionalGroupExists)
	})
}

func TestCheckGroupMemberLocality(t *testing.T) {
	t.Run("imported member emits compliance-member-not-local", func(t *testing.T) {
		modA := &module.Module{Name: "SOURCE-MIB"}
		modB := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(modB, []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "testGroup"},
				Objects: []string{"importedObj"},
				Status:  types.StatusCurrent,
			},
		})

		ctx := newComplianceTestContext(modA, modB)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		model.SetNodeName(grpNode, "testGroup")
		ctx.registerModuleNodeSymbol(modB, "testGroup", grpNode)

		// Register importedObj in modA (source) but not in modB (group's module).
		objNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(objNode, "importedObj")
		model.SetNodeKind(objNode, types.KindScalar)
		ctx.registerModuleNodeSymbol(modA, "importedObj", objNode)
		// Make it available in modB via import.
		if ctx.importSources[modB] == nil {
			ctx.importSources[modB] = make(map[string]*module.Module)
		}
		ctx.importSources[modB]["importedObj"] = modA

		checkGroupMemberLocality(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagComplianceMemberNotLocal)
	})

	t.Run("local member no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(mod, []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "testGroup"},
				Objects: []string{"localObj"},
				Status:  types.StatusCurrent,
			},
		})

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		model.SetNodeName(grpNode, "testGroup")
		ctx.registerModuleNodeSymbol(mod, "testGroup", grpNode)

		objNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(objNode, "localObj")
		model.SetNodeKind(objNode, types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "localObj", objNode)

		checkGroupMemberLocality(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagComplianceMemberNotLocal)
	})

	t.Run("notification group imported member emits diagnostic", func(t *testing.T) {
		modA := &module.Module{Name: "SOURCE-MIB"}
		modB := &module.Module{
			Name: "TEST-MIB",
		}
		addDefs(modB, []module.Definition{
			&module.NotificationGroup{
				DefBase:       module.DefBase{Name: "testNotifGroup"},
				Notifications: []string{"importedNotif"},
				Status:        types.StatusCurrent,
			},
		})

		ctx := newComplianceTestContext(modA, modB)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		model.SetNodeName(grpNode, "testNotifGroup")
		ctx.registerModuleNodeSymbol(modB, "testNotifGroup", grpNode)

		notifNode := buildOIDPath(root, 1, 2)
		model.SetNodeName(notifNode, "importedNotif")
		model.SetNodeKind(notifNode, types.KindNotification)
		ctx.registerModuleNodeSymbol(modA, "importedNotif", notifNode)
		if ctx.importSources[modB] == nil {
			ctx.importSources[modB] = make(map[string]*module.Module)
		}
		ctx.importSources[modB]["importedNotif"] = modA

		checkGroupMemberLocality(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagComplianceMemberNotLocal)
	})
}

// makeTypedObject creates a minimal Object with a type of the given base type
// and optional sizes. Used for unit testing index helpers.
func makeTypedObject(base model.BaseType, sizes *[]model.Range) *model.Object {
	obj := model.NewObject("x")
	typ := model.NewType("xType")
	model.SetTypeBase(typ, base)
	model.SetObjectType(obj, typ)
	if sizes != nil {
		model.SetObjectEffectiveSizes(obj, *sizes)
	}
	return obj
}

func TestCheckRowStatusDefault(t *testing.T) {
	t.Run("illegal action emits diagnostic", func(t *testing.T) {
		mod := testTableModule(
			[]module.Import{
				module.NewImport("SNMPv2-TC", "RowStatus", types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testRowStatus"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "RowStatus"},
				Access:  types.AccessReadCreate,
				Status:  types.StatusCurrent,
				DefVal:  &module.DefValEnum{Name: "createAndGo"},
				Oid:     testOid("testEntry", 2),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagRowStatusDefault)
	})

	t.Run("valid active no diagnostic", func(t *testing.T) {
		mod := testTableModule(
			[]module.Import{
				module.NewImport("SNMPv2-TC", "RowStatus", types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testRowStatus"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "RowStatus"},
				Access:  types.AccessReadCreate,
				Status:  types.StatusCurrent,
				DefVal:  &module.DefValInteger{Value: 1},
				Oid:     testOid("testEntry", 2),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagRowStatusDefault)
	})
}

func TestCheckRowStatusAccess_NotReadCreate(t *testing.T) {
	mod := testTableModule(
		[]module.Import{
			module.NewImport("SNMPv2-TC", "RowStatus", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testRowStatus"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "RowStatus"},
			Access:  types.AccessReadWrite,
			Status:  types.StatusCurrent,
			Oid:     testOid("testEntry", 2),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagRowStatusAccess)
}

func TestCheckStorageTypeDefault(t *testing.T) {
	t.Run("permanent emits diagnostic", func(t *testing.T) {
		mod := testTableModule(
			[]module.Import{
				module.NewImport("SNMPv2-TC", "StorageType", types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testStorage"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "StorageType"},
				Access:  types.AccessReadCreate,
				Status:  types.StatusCurrent,
				DefVal:  &module.DefValEnum{Name: "permanent"},
				Oid:     testOid("testEntry", 2),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagStorageTypeDefault)
	})

	t.Run("nonVolatile no diagnostic", func(t *testing.T) {
		mod := testTableModule(
			[]module.Import{
				module.NewImport("SNMPv2-TC", "StorageType", types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testStorage"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "StorageType"},
				Access:  types.AccessReadCreate,
				Status:  types.StatusCurrent,
				DefVal:  &module.DefValInteger{Value: 3},
				Oid:     testOid("testEntry", 2),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagStorageTypeDefault)
	})
}

func TestCheckTAddressTDomain(t *testing.T) {
	t.Run("missing TDomain emits diagnostic", func(t *testing.T) {
		mod := testTableModule(
			[]module.Import{
				module.NewImport("SNMPv2-TC", "TAddress", types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testAddr"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "TAddress"},
				Access:  types.AccessReadOnly,
				Status:  types.StatusCurrent,
				Oid:     testOid("testEntry", 2),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagTAddressTDomain)
	})

	t.Run("with TDomain no diagnostic", func(t *testing.T) {
		mod := testTableModule(
			[]module.Import{
				module.NewImport("SNMPv2-TC", "TAddress", types.Span{}),
				module.NewImport("SNMPv2-TC", "TDomain", types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testDomain"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "TDomain"},
				Access:  types.AccessReadOnly,
				Status:  types.StatusCurrent,
				Oid:     testOid("testEntry", 2),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testAddr"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "TAddress"},
				Access:  types.AccessReadOnly,
				Status:  types.StatusCurrent,
				Oid:     testOid("testEntry", 3),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagTAddressTDomain)
	})
}

func TestCheckAddressPairing(t *testing.T) {
	families := []struct {
		label        string
		depMod       func() *module.Module
		depModName   string
		addrType     string
		addrTypeName string
		specificType string
		diagPairing  string
		diagSpecific string
		diagSubtyped string
		subtypeEnums []module.NamedNumber
	}{
		{
			label:        "InetAddress",
			depMod:       testInetAddressMIB,
			depModName:   "INET-ADDRESS-MIB",
			addrType:     "InetAddress",
			addrTypeName: "InetAddressType",
			specificType: "InetAddressIPv4",
			diagPairing:  types.DiagInetAddressPairing,
			diagSpecific: types.DiagInetAddressSpecific,
			diagSubtyped: types.DiagInetAddressTypeSubtyped,
			subtypeEnums: []module.NamedNumber{{Name: "ipv4", Value: 1}, {Name: "ipv6", Value: 2}},
		},
		{
			label:        "TransportAddress",
			depMod:       testTransportAddressMIB,
			depModName:   "TRANSPORT-ADDRESS-MIB",
			addrType:     "TransportAddress",
			addrTypeName: "TransportAddressType",
			specificType: "TransportAddressIPv4",
			diagPairing:  types.DiagTransportAddressPairing,
			diagSpecific: types.DiagTransportAddressSpecific,
			diagSubtyped: types.DiagTransportAddressTypeSubtyped,
			subtypeEnums: []module.NamedNumber{{Name: "udpIpv4", Value: 1}, {Name: "udpIpv6", Value: 2}},
		},
	}

	for _, fam := range families {
		t.Run(fam.label+"/missing type", func(t *testing.T) {
			mod := testTableModule(
				[]module.Import{
					module.NewImport(fam.depModName, fam.addrType, types.Span{}),
				},
				&module.ObjectType{
					DefBase: module.DefBase{Name: "testAddr"},
					Syntax:  &module.TypeSyntaxTypeRef{Name: fam.addrType},
					Access:  types.AccessReadOnly,
					Status:  types.StatusCurrent,
					Oid:     testOid("testEntry", 2),
				},
			)
			m := resolveStrict(fam.depMod(), mod)
			hasDiag(t, m.Diagnostics(), fam.diagPairing)
		})

		t.Run(fam.label+"/specific type", func(t *testing.T) {
			mod := testTableModule(
				[]module.Import{
					module.NewImport(fam.depModName, fam.specificType, types.Span{}),
				},
				&module.ObjectType{
					DefBase: module.DefBase{Name: "testAddr"},
					Syntax:  &module.TypeSyntaxTypeRef{Name: fam.specificType},
					Access:  types.AccessReadOnly,
					Status:  types.StatusCurrent,
					Oid:     testOid("testEntry", 2),
				},
			)
			m := resolveStrict(fam.depMod(), mod)
			hasDiag(t, m.Diagnostics(), fam.diagSpecific)
		})

		t.Run(fam.label+"/type subtyped", func(t *testing.T) {
			mod := testTableModule(
				[]module.Import{
					module.NewImport(fam.depModName, fam.addrType, types.Span{}),
					module.NewImport(fam.depModName, fam.addrTypeName, types.Span{}),
				},
				&module.ObjectType{
					DefBase: module.DefBase{Name: "testAddrType"},
					Syntax: &module.TypeSyntaxIntegerEnum{
						Base:         fam.addrTypeName,
						NamedNumbers: fam.subtypeEnums,
					},
					Access: types.AccessReadOnly,
					Status: types.StatusCurrent,
					Oid:    testOid("testEntry", 2),
				},
				&module.ObjectType{
					DefBase: module.DefBase{Name: "testAddr"},
					Syntax:  &module.TypeSyntaxTypeRef{Name: fam.addrType},
					Access:  types.AccessReadOnly,
					Status:  types.StatusCurrent,
					Oid:     testOid("testEntry", 3),
				},
			)
			m := resolveStrict(fam.depMod(), mod)
			hasDiag(t, m.Diagnostics(), fam.diagSubtyped)
		})
	}

	// InetAddress-specific: pairing present suppresses diagnostic.
	t.Run("InetAddress/with type OK", func(t *testing.T) {
		inetMod := testInetAddressMIB()
		mod := testTableModule(
			[]module.Import{
				module.NewImport("INET-ADDRESS-MIB", "InetAddress", types.Span{}),
				module.NewImport("INET-ADDRESS-MIB", "InetAddressType", types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testAddrType"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "InetAddressType"},
				Access:  types.AccessReadOnly,
				Status:  types.StatusCurrent,
				Oid:     testOid("testEntry", 2),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testAddr"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "InetAddress"},
				Access:  types.AccessReadOnly,
				Status:  types.StatusCurrent,
				Oid:     testOid("testEntry", 3),
			},
		)
		m := resolveStrict(inetMod, mod)
		noDiag(t, m.Diagnostics(), types.DiagInetAddressPairing)
	})
}

func testTransportAddressMIB() *module.Module {
	mod := module.NewModule("TRANSPORT-ADDRESS-MIB", types.Synthetic)
	mod.Language = types.LanguageSMIv2
	mod.Imports = []module.Import{
		module.NewImport("SNMPv2-SMI", "MODULE-IDENTITY", types.Span{}),
		module.NewImport("SNMPv2-SMI", "mib-2", types.Span{}),
		module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
	}
	addDefs(mod, []module.Definition{
		&module.ModuleIdentity{
			DefBase: module.DefBase{Name: "transportAddressMIB"},
			Oid: module.NewOidAssignment([]module.OidComponent{
				&module.OidComponentName{NameValue: "mib-2"},
				&module.OidComponentNumber{Value: 100},
			}, types.Span{}),
		},
		// TransportAddressType ::= TEXTUAL-CONVENTION SYNTAX INTEGER { udpIpv4(1), ... }
		&module.TypeDef{
			DefBase: module.DefBase{Name: "TransportAddressType"},
			Syntax: &module.TypeSyntaxIntegerEnum{
				NamedNumbers: []module.NamedNumber{
					{Name: "udpIpv4", Value: 1},
					{Name: "udpIpv6", Value: 2},
					{Name: "udpIpv4z", Value: 3},
					{Name: "udpIpv6z", Value: 4},
				},
			},
			Status:              types.StatusCurrent,
			IsTextualConvention: true,
		},
		// TransportAddress ::= TEXTUAL-CONVENTION SYNTAX OCTET STRING (SIZE (0..255))
		&module.TypeDef{
			DefBase: module.DefBase{Name: "TransportAddress"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxOctetString{},
				Constraint: &module.ConstraintSize{Ranges: []module.Range{module.NewRangeUnsigned(0, 255, types.Span{})}},
			},
			Status:              types.StatusCurrent,
			IsTextualConvention: true,
		},
		// TransportAddressIPv4 ::= TEXTUAL-CONVENTION SYNTAX OCTET STRING (SIZE (6))
		&module.TypeDef{
			DefBase: module.DefBase{Name: "TransportAddressIPv4"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxOctetString{},
				Constraint: &module.ConstraintSize{Ranges: []module.Range{{Min: &module.RangeValueUnsigned{Value: 6}}}},
			},
			Status:              types.StatusCurrent,
			IsTextualConvention: true,
		},
		// TransportAddressIPv6 ::= TEXTUAL-CONVENTION SYNTAX OCTET STRING (SIZE (18))
		&module.TypeDef{
			DefBase: module.DefBase{Name: "TransportAddressIPv6"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxOctetString{},
				Constraint: &module.ConstraintSize{Ranges: []module.Range{{Min: &module.RangeValueUnsigned{Value: 18}}}},
			},
			Status:              types.StatusCurrent,
			IsTextualConvention: true,
		},
	})
	return mod
}

func TestCheckGroupUnreferenced(t *testing.T) {
	t.Run("unreferenced groups emit diagnostics", func(t *testing.T) {
		mod := module.NewModule("TEST-MIB", types.Span{})
		mod.ObjectGroups = []*module.ObjectGroup{
			{DefBase: module.DefBase{Name: "unusedObjectGroup"}},
		}
		mod.NotificationGroups = []*module.NotificationGroup{
			{DefBase: module.DefBase{Name: "unusedNotificationGroup"}},
		}

		ctx := newTestContextForModules(model.VerboseConfig(), mod)
		checkGroupUnreferenced(ctx)

		requireDiagCount(t, ctx.Diagnostics(), types.DiagGroupUnreferenced, 2)
	})

	t.Run("compliance and capabilities references suppress diagnostics", func(t *testing.T) {
		mod := module.NewModule("TEST-MIB", types.Span{})
		mod.ObjectGroups = []*module.ObjectGroup{
			{DefBase: module.DefBase{Name: "usedObjectGroup"}},
		}
		mod.NotificationGroups = []*module.NotificationGroup{
			{DefBase: module.DefBase{Name: "usedNotificationGroup"}},
		}
		mod.Compliances = []*module.ModuleCompliance{
			{
				DefBase: module.DefBase{Name: "testCompliance"},
				Modules: []module.ComplianceModule{
					{
						MandatoryGroups: []string{"usedObjectGroup"},
					},
				},
			},
		}
		mod.Capabilities = []*module.AgentCapabilities{
			{
				DefBase: module.DefBase{Name: "testCaps"},
				Supports: []module.SupportsModule{
					{
						ModuleName: "TEST-MIB",
						Includes:   []string{"usedNotificationGroup"},
					},
				},
			},
		}

		ctx := newTestContextForModules(model.VerboseConfig(), mod)
		checkGroupUnreferenced(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagGroupUnreferenced)
	})
}

// --- checkOpaqueSMIv2 ---
