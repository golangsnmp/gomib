package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestCheckNodeParentKinds_RowUnderTable(t *testing.T) {
	// Valid: row's parent is a table. Should produce no diagnostic.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testTable"},
			Syntax:  &module.TypeSyntaxSequenceOf{EntryType: "TestEntry"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testEntry"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "TestEntry"},
			Status:  types.StatusCurrent,
			Index:   []module.IndexItem{{Object: "testIndex"}},
			Oid:     testOid("testTable", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagParentRow)
}

func TestCheckNodeParentKinds_RowUnderScalar(t *testing.T) {
	// Invalid: row's parent is a scalar, not a table.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "badParent"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "badRow"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "TestEntry"},
			Status:  types.StatusCurrent,
			Index:   []module.IndexItem{{Object: "testCol"}},
			Oid:     testOid("badParent", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagParentRow)
}

func TestCheckNodeParentKinds_TableUnderTable(t *testing.T) {
	// Invalid: table's parent is another table.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "outerTable"},
			Syntax:  &module.TypeSyntaxSequenceOf{EntryType: "OuterEntry"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "innerTable"},
			Syntax:  &module.TypeSyntaxSequenceOf{EntryType: "InnerEntry"},
			Status:  types.StatusCurrent,
			Oid:     testOid("outerTable", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagParentTable)
}

func TestCheckNodeParentKinds_ScalarUnderRow(t *testing.T) {
	// A scalar child of a row is reclassified as a column by inferNodeKinds.
	// So this should NOT trigger parent-scalar.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testTable"},
			Syntax:  &module.TypeSyntaxSequenceOf{EntryType: "TestEntry"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testEntry"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "TestEntry"},
			Status:  types.StatusCurrent,
			Index:   []module.IndexItem{{Object: "testCol"}},
			Oid:     testOid("testTable", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testCol"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testEntry", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagParentScalar, types.DiagParentColumn)
}

func TestCheckNodeParentKinds_NotificationUnderTable(t *testing.T) {
	// Invalid: notification's parent is a table.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testTable"},
			Syntax:  &module.TypeSyntaxSequenceOf{EntryType: "TestEntry"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
		&module.Notification{
			DefBase: module.DefBase{Name: "badNotif"},
			Status:  types.StatusCurrent,
			Oid: func() *module.OidAssignment {
				oid := testOid("testTable", 2)
				return &oid
			}(),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagParentNotification)
}

func TestCheckNodeParentKinds_GroupUnderTable(t *testing.T) {
	// Invalid: group's parent is a table.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testTable"},
			Syntax:  &module.TypeSyntaxSequenceOf{EntryType: "TestEntry"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
		&module.ObjectGroup{
			DefBase: module.DefBase{Name: "badGroup"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testTable", 2),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagParentGroup)
}

func TestCheckNodeParentKinds_RowSubidOne(t *testing.T) {
	// Valid: row at sub-identifier 1 under its table. No diagnostic expected.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testTable"},
			Syntax:  &module.TypeSyntaxSequenceOf{EntryType: "TestEntry"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testEntry"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "TestEntry"},
			Status:  types.StatusCurrent,
			Index:   []module.IndexItem{{Object: "testIndex"}},
			Oid:     testOid("testTable", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagRowSubidentifierOne)
}

func TestCheckNodeParentKinds_RowSubidNotOne(t *testing.T) {
	// Invalid: row at sub-identifier 2 under its table.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testTable"},
			Syntax:  &module.TypeSyntaxSequenceOf{EntryType: "TestEntry"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testEntry"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "TestEntry"},
			Status:  types.StatusCurrent,
			Index:   []module.IndexItem{{Object: "testIndex"}},
			Oid:     testOid("testTable", 2),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagRowSubidentifierOne)
}

func TestCheckIndexElementNoSize(t *testing.T) {
	t.Run("OCTET STRING index without SIZE emits diagnostic", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod

		idxObj := newObject("strIndex")
		typ := newType("DisplayString")
		typ.setBase(BaseOctetString)
		idxObj.setType(typ)
		resolvedMod.addObject(idxObj)

		root := ctx.mib.Root()
		idxNode := buildOIDPath(root, 1, 1, 1, 1)
		idxNode.setName("strIndex")
		idxNode.setObject(idxObj)
		ctx.registerModuleNodeSymbol(mod, "strIndex", idxNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Index:   []module.IndexItem{{Object: "strIndex"}},
			}},
		}

		checkIndexConstraints(ctx, objRefs)

		d := hasDiag(t, ctx.Diagnostics(), types.DiagIndexElementNoSize)
		testutil.Contains(t, d.Message, "strIndex", "diagnostic message")
	})

	t.Run("OCTET STRING index with SIZE is ok", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod

		idxObj := newObject("strIndex")
		typ := newType("DisplayString")
		typ.setBase(BaseOctetString)
		idxObj.setType(typ)
		idxObj.setEffectiveSizes([]Range{{Min: 0, Max: 255}})
		resolvedMod.addObject(idxObj)

		root := ctx.mib.Root()
		idxNode := buildOIDPath(root, 1, 1, 1, 1)
		idxNode.setName("strIndex")
		idxNode.setObject(idxObj)
		ctx.registerModuleNodeSymbol(mod, "strIndex", idxNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "myEntry"},
				Index:   []module.IndexItem{{Object: "strIndex"}},
			}},
		}

		checkIndexConstraints(ctx, objRefs)

		noDiag(t, ctx.Diagnostics(), types.DiagIndexElementNoSize)
	})
}

func TestIsLegalIndexBasetype(t *testing.T) {
	legal := []BaseType{
		BaseInteger32, BaseUnsigned32, BaseGauge32,
		BaseTimeTicks, BaseIpAddress, BaseOctetString, BaseOpaque,
		BaseBits, BaseObjectIdentifier,
	}
	for _, b := range legal {
		testutil.True(t, isLegalIndexBasetype(b), b.String()+" should be legal")
	}

	illegal := []BaseType{BaseCounter32, BaseCounter64, BaseSequence, BaseUnknown}
	for _, b := range illegal {
		testutil.False(t, isLegalIndexBasetype(b), b.String()+" should be illegal")
	}
}

func TestCheckIndexIllegalBasetype(t *testing.T) {
	ctx := newTestContext()
	mod := &module.Module{Name: "TEST-MIB"}
	ctx.modules = append(ctx.modules, mod)
	resolvedMod := newModule(mod.Name)
	ctx.moduleToResolved[mod] = resolvedMod

	// Create index object with Counter64 base type (illegal for INDEX).
	idxObj := newObject("badIndex")
	typ := newType("Counter64")
	typ.setBase(BaseCounter64)
	idxObj.setType(typ)
	resolvedMod.addObject(idxObj)

	root := ctx.mib.Root()
	idxNode := buildOIDPath(root, 1, 1, 1, 1)
	idxNode.setName("badIndex")
	idxNode.setObject(idxObj)
	ctx.registerModuleNodeSymbol(mod, "badIndex", idxNode)

	objRefs := []objectTypeRef{
		{mod: mod, obj: &module.ObjectType{
			DefBase: module.DefBase{Name: "myEntry"},
			Index:   []module.IndexItem{{Object: "badIndex"}},
		}},
	}

	checkIndexConstraints(ctx, objRefs)

	d := hasDiag(t, ctx.Diagnostics(), types.DiagIndexIllegalBasetype)
	testutil.Contains(t, d.Message, "badIndex", "diagnostic message")
	testutil.Contains(t, d.Message, "Counter64", "diagnostic should mention type")
}

func TestCheckIndexOIDLength(t *testing.T) {
	t.Run("exceeds 128 sub-identifiers", func(t *testing.T) {
		ctx := newTestContextWithConfig(VerboseConfig())
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod

		// Create an index object with a variable-length OCTET STRING
		// with SIZE (0..255). This contributes 255+1=256 sub-ids
		// (max size + length prefix).
		idxObj := newObject("bigIndex")
		typ := newType("BigString")
		typ.setBase(BaseOctetString)
		idxObj.setType(typ)
		idxObj.setEffectiveSizes([]Range{{Min: 0, Max: 255}})
		resolvedMod.addObject(idxObj)

		root := ctx.mib.Root()
		idxNode := buildOIDPath(root, 1, 3, 6, 1, 2, 1, 99, 1, 1, 1)
		idxNode.setName("bigIndex")
		idxNode.setObject(idxObj)
		ctx.registerModuleNodeSymbol(mod, "bigIndex", idxNode)

		// Create the row object at a path of length 9 (1.3.6.1.2.1.99.1.1).
		rowObj := newObject("testEntry")
		resolvedMod.addObject(rowObj)
		rowObj.setIndex([]IndexEntry{{
			Object:   idxObj,
			Encoding: IndexEncodingLengthPrefixed,
		}})

		rowNode := buildOIDPath(root, 1, 3, 6, 1, 2, 1, 99, 1, 1)
		rowNode.setName("testEntry")
		rowNode.setObject(rowObj)
		ctx.registerModuleNodeSymbol(mod, "testEntry", rowNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "testEntry"},
				Index:   []module.IndexItem{{Object: "bigIndex"}},
			}},
		}

		checkIndexConstraints(ctx, objRefs)

		d := hasDiag(t, ctx.Diagnostics(), types.DiagIndexExceedsTooLarge)
		testutil.Contains(t, d.Message, "testEntry", "diagnostic message")
	})

	t.Run("within 128 sub-identifiers", func(t *testing.T) {
		ctx := newTestContext()
		mod := &module.Module{Name: "TEST-MIB"}
		ctx.modules = append(ctx.modules, mod)
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod

		// Integer index: contributes 1 sub-id.
		idxObj := newObject("intIndex")
		typ := newType("IntIdx")
		typ.setBase(BaseInteger32)
		idxObj.setType(typ)
		idxObj.setEffectiveRanges([]Range{{Min: 0, Max: 255}})
		resolvedMod.addObject(idxObj)

		root := ctx.mib.Root()
		idxNode := buildOIDPath(root, 1, 3, 6, 1, 2, 1, 100, 1, 1, 1)
		idxNode.setName("intIndex")
		idxNode.setObject(idxObj)
		ctx.registerModuleNodeSymbol(mod, "intIndex", idxNode)

		// Row OID: 1.3.6.1.2.1.100.1.1 = 9 arcs.
		// Index: 1 sub-id. Total: 9 + 1 = 10. Well within 128.
		rowObj := newObject("okEntry")
		resolvedMod.addObject(rowObj)
		rowObj.setIndex([]IndexEntry{{
			Object:   idxObj,
			Encoding: IndexEncodingInteger,
		}})

		rowNode := buildOIDPath(root, 1, 3, 6, 1, 2, 1, 100, 1, 1)
		rowNode.setName("okEntry")
		rowNode.setObject(rowObj)
		ctx.registerModuleNodeSymbol(mod, "okEntry", rowNode)

		objRefs := []objectTypeRef{
			{mod: mod, obj: &module.ObjectType{
				DefBase: module.DefBase{Name: "okEntry"},
				Index:   []module.IndexItem{{Object: "intIndex"}},
			}},
		}

		checkIndexConstraints(ctx, objRefs)

		noDiag(t, ctx.Diagnostics(), types.DiagIndexExceedsTooLarge)
	})
}

func TestIndexElementSubIds(t *testing.T) {
	tests := []struct {
		name   string
		entry  IndexEntry
		want   int
		wantOK bool
	}{
		{
			name:   "integer",
			entry:  IndexEntry{Object: makeTypedObject(BaseInteger32, nil), Encoding: IndexEncodingInteger},
			want:   1,
			wantOK: true,
		},
		{
			name:   "ip address",
			entry:  IndexEntry{Object: makeTypedObject(BaseIpAddress, nil), Encoding: IndexEncodingIpAddress},
			want:   4,
			wantOK: true,
		},
		{
			name:   "fixed string SIZE 8",
			entry:  IndexEntry{Object: makeTypedObject(BaseOctetString, &[]Range{{Min: 8, Max: 8}}), Encoding: IndexEncodingFixedString},
			want:   8,
			wantOK: true,
		},
		{
			name:   "variable string SIZE 0..32",
			entry:  IndexEntry{Object: makeTypedObject(BaseOctetString, &[]Range{{Min: 0, Max: 32}}), Encoding: IndexEncodingLengthPrefixed},
			want:   33,
			wantOK: true,
		},
		{
			name:   "implied string SIZE 0..32",
			entry:  IndexEntry{Object: makeTypedObject(BaseOctetString, &[]Range{{Min: 0, Max: 32}}), Encoding: IndexEncodingImplied},
			want:   32,
			wantOK: true,
		},
		{
			name:   "OID length-prefixed",
			entry:  IndexEntry{Object: makeTypedObject(BaseObjectIdentifier, nil), Encoding: IndexEncodingLengthPrefixed},
			want:   129,
			wantOK: true,
		},
		{
			name:   "OID implied",
			entry:  IndexEntry{Object: makeTypedObject(BaseObjectIdentifier, nil), Encoding: IndexEncodingImplied},
			want:   128,
			wantOK: true,
		},
		{
			name:   "nil object",
			entry:  IndexEntry{Encoding: IndexEncodingInteger},
			want:   0,
			wantOK: false,
		},
		{
			name:   "unknown encoding",
			entry:  IndexEntry{Object: makeTypedObject(BaseOctetString, nil), Encoding: IndexEncodingUnknown},
			want:   0,
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := indexElementSubIds(tt.entry)
			testutil.Equal(t, tt.wantOK, ok, "ok")
			testutil.Equal(t, tt.want, got, "sub-id count")
		})
	}
}

func TestCheckAccessValidity(t *testing.T) {
	t.Run("accessible-for-notify invalid in SMIv1", func(t *testing.T) {
		mod := testSMIv1Module(nil,
			&module.ObjectType{
				DefBase: module.DefBase{Name: "badAccess"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Access:  types.AccessAccessibleForNotify,
				Status:  types.StatusMandatory,
				Oid:     testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagAccessInvalidSMIv1)
	})

	t.Run("read-create invalid in SMIv1", func(t *testing.T) {
		mod := testSMIv1Module(nil,
			&module.ObjectType{
				DefBase: module.DefBase{Name: "badAccess"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Access:  types.AccessReadCreate,
				Status:  types.StatusMandatory,
				Oid:     testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagAccessInvalidSMIv1)
	})

	t.Run("write-only invalid in SMIv2", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
				module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "badAccess"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Access:        types.AccessWriteOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagAccessWriteOnlySMIv2)
	})

	t.Run("read-write valid in SMIv1", func(t *testing.T) {
		mod := testSMIv1Module(nil,
			&module.ObjectType{
				DefBase: module.DefBase{Name: "validObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Access:  types.AccessReadWrite,
				Status:  types.StatusMandatory,
				Oid:     testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagAccessInvalidSMIv1, types.DiagAccessWriteOnlySMIv2)
	})
}

func TestCheckMaxAccessInSMIv1(t *testing.T) {
	mod := testSMIv1Module(nil,
		&module.ObjectType{
			DefBase:       module.DefBase{Name: "badKeyword"},
			Syntax:        &module.TypeSyntaxTypeRef{Name: "INTEGER"},
			Access:        types.AccessReadOnly,
			AccessKeyword: types.AccessKeywordMaxAccess,
			Status:        types.StatusMandatory,
			Oid:           testOid("testRoot", 1),
		},
	)
	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagMaxAccessInSMIv1)
}

func TestCheckAccessInSMIv2(t *testing.T) {
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase:       module.DefBase{Name: "badKeyword"},
			Syntax:        &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Access:        types.AccessReadOnly,
			AccessKeyword: types.AccessKeywordAccess,
			Status:        types.StatusCurrent,
			Oid:           testOid("testRoot", 1),
		},
	)
	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagAccessInSMIv2)
}

func TestCheckAccessTableIllegal(t *testing.T) {
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase:       module.DefBase{Name: "badTable"},
			Syntax:        &module.TypeSyntaxSequenceOf{EntryType: "TestEntry"},
			Access:        types.AccessReadOnly,
			AccessKeyword: types.AccessKeywordMaxAccess,
			Status:        types.StatusCurrent,
			Oid:           testOid("testRoot", 1),
		},
	)
	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagAccessTableIllegal)
}

func TestCheckAccessRowIllegal(t *testing.T) {
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase:       module.DefBase{Name: "testTable"},
			Syntax:        &module.TypeSyntaxSequenceOf{EntryType: "TestEntry"},
			Access:        types.AccessNotAccessible,
			AccessKeyword: types.AccessKeywordMaxAccess,
			Status:        types.StatusCurrent,
			Oid:           testOid("testRoot", 1),
		},
		&module.ObjectType{
			DefBase:       module.DefBase{Name: "testEntry"},
			Syntax:        &module.TypeSyntaxTypeRef{Name: "TestEntry"},
			Access:        types.AccessReadOnly,
			AccessKeyword: types.AccessKeywordMaxAccess,
			Status:        types.StatusCurrent,
			Index:         []module.IndexItem{{Object: "testCol"}},
			Oid:           testOid("testTable", 1),
		},
	)
	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagAccessRowIllegal)
}

func TestCheckScalarNotCreatable(t *testing.T) {
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase:       module.DefBase{Name: "badScalar"},
			Syntax:        &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Access:        types.AccessReadCreate,
			AccessKeyword: types.AccessKeywordMaxAccess,
			Status:        types.StatusCurrent,
			Oid:           testOid("testRoot", 1),
		},
	)
	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagScalarNotCreatable)
}

func TestCheckAccessCounter(t *testing.T) {
	t.Run("read-write illegal for Counter32", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
				module.NewImport("SNMPv2-SMI", "Counter32", types.Span{}),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "badCounter"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "Counter32"},
				Access:        types.AccessReadWrite,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagAccessCounterIllegal)
	})

	t.Run("read-only valid for Counter32", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
				module.NewImport("SNMPv2-SMI", "Counter32", types.Span{}),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "goodCounter"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "Counter32"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagAccessCounterIllegal)
	})
}

func TestCheckStatusValidity(t *testing.T) {
	t.Run("current invalid in SMIv1", func(t *testing.T) {
		mod := testSMIv1Module(nil,
			&module.ObjectType{
				DefBase: module.DefBase{Name: "badStatus"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Access:  types.AccessReadOnly,
				Status:  types.StatusCurrent,
				Oid:     testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagStatusInvalidSMIv1)
	})

	t.Run("mandatory invalid in SMIv2", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
				module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "badStatus"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusMandatory,
				Oid:           testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagStatusInvalidSMIv2)
	})

	t.Run("mandatory valid in SMIv1", func(t *testing.T) {
		mod := testSMIv1Module(nil,
			&module.ObjectType{
				DefBase: module.DefBase{Name: "validObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Access:  types.AccessReadOnly,
				Status:  types.StatusMandatory,
				Oid:     testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagStatusInvalidSMIv1, types.DiagStatusInvalidSMIv2)
	})
}

func TestCheckTypeStatus(t *testing.T) {
	t.Run("deprecated type emits diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "DeprecatedType"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:              types.StatusDeprecated,
				Description:         "deprecated type for testing",
				IsTextualConvention: true,
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "usesDeprecated"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "DeprecatedType"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagTypeStatusDeprecated)
	})

	t.Run("obsolete type emits diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "ObsoleteType"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:              types.StatusObsolete,
				Description:         "obsolete type for testing",
				IsTextualConvention: true,
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "usesObsolete"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "ObsoleteType"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagTypeStatusObsolete)
	})

	t.Run("current type no diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
				module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "validObj"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagTypeStatusDeprecated, types.DiagTypeStatusObsolete)
	})
}

func TestCheckGroupMembership_NoGroups(t *testing.T) {
	// Module with no groups should produce no group-membership diagnostics.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
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
		},
	}
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
		Definitions: []module.Definition{
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
		},
	}
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
		Definitions: []module.Definition{
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
		},
	}
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
		Definitions: []module.Definition{
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
		},
	}
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
		Definitions: []module.Definition{
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
		},
	}
	m := resolveStrict(mod)
	d := hasDiag(t, m.Diagnostics(), types.DiagGroupMembership)
	testutil.Contains(t, d.Message, "testNotif2", "diagnostic message")
}

func TestCheckComplianceStatus_GroupStatus(t *testing.T) {
	t.Run("obsolete group in current compliance emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
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
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("obsGroup")
		ctx.registerModuleNodeSymbol(mod, "obsGroup", grpNode)

		objNode := buildOIDPath(root, 1, 2)
		objNode.setName("someObj")
		objNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "someObj", objNode)
		obj := newObject("someObj")
		obj.setStatus(types.StatusObsolete)
		objNode.setObject(obj)

		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedGroups(ctx)
		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagComplianceGroupStatus)
	})

	t.Run("current group in current compliance no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
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
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("curGroup")
		ctx.registerModuleNodeSymbol(mod, "curGroup", grpNode)

		objNode := buildOIDPath(root, 1, 2)
		objNode.setName("someObj")
		objNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "someObj", objNode)
		obj := newObject("someObj")
		obj.setStatus(types.StatusCurrent)
		objNode.setObject(obj)

		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedGroups(ctx)
		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagComplianceGroupStatus)
	})

	t.Run("optional GROUP clause with obsolete group emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
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
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("optGroup")
		ctx.registerModuleNodeSymbol(mod, "optGroup", grpNode)

		objNode := buildOIDPath(root, 1, 2)
		objNode.setName("someObj")
		objNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "someObj", objNode)
		obj := newObject("someObj")
		obj.setStatus(types.StatusObsolete)
		objNode.setObject(obj)

		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedGroups(ctx)
		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagComplianceGroupStatus)
	})

	t.Run("smiv1 compliance status skips check", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
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
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("obsGroup")
		ctx.registerModuleNodeSymbol(mod, "obsGroup", grpNode)

		objNode := buildOIDPath(root, 1, 2)
		objNode.setName("someObj")
		objNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "someObj", objNode)
		obj := newObject("someObj")
		obj.setStatus(types.StatusObsolete)
		objNode.setObject(obj)

		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
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
			Definitions: []module.Definition{
				&module.ModuleCompliance{
					DefBase: module.DefBase{Name: "testCompliance"},
					Status:  types.StatusCurrent,
					Modules: []module.ComplianceModule{
						{Objects: []module.ComplianceObject{{Object: "obsObj"}}},
					},
					Oid: testOid("testNode", 10),
				},
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		objNode := buildOIDPath(root, 1, 2)
		objNode.setName("obsObj")
		objNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "obsObj", objNode)
		obj := newObject("obsObj")
		obj.setStatus(types.StatusObsolete)
		objNode.setObject(obj)

		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagComplianceObjectStatus)
	})

	t.Run("current object in current compliance no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.ModuleCompliance{
					DefBase: module.DefBase{Name: "testCompliance"},
					Status:  types.StatusCurrent,
					Modules: []module.ComplianceModule{
						{Objects: []module.ComplianceObject{{Object: "curObj"}}},
					},
					Oid: testOid("testNode", 10),
				},
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		objNode := buildOIDPath(root, 1, 2)
		objNode.setName("curObj")
		objNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "curObj", objNode)
		obj := newObject("curObj")
		obj.setStatus(types.StatusCurrent)
		objNode.setObject(obj)

		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagComplianceObjectStatus)
	})

	t.Run("deprecated object in deprecated compliance no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.ModuleCompliance{
					DefBase: module.DefBase{Name: "testCompliance"},
					Status:  types.StatusDeprecated,
					Modules: []module.ComplianceModule{
						{Objects: []module.ComplianceObject{{Object: "depObj"}}},
					},
					Oid: testOid("testNode", 10),
				},
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		objNode := buildOIDPath(root, 1, 2)
		objNode.setName("depObj")
		objNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "depObj", objNode)
		obj := newObject("depObj")
		obj.setStatus(types.StatusDeprecated)
		objNode.setObject(obj)

		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStatus(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagComplianceObjectStatus)
	})

	t.Run("smiv1 object status skips check", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.ModuleCompliance{
					DefBase: module.DefBase{Name: "testCompliance"},
					Status:  types.StatusCurrent,
					Modules: []module.ComplianceModule{
						{Objects: []module.ComplianceObject{{Object: "mandObj"}}},
					},
					Oid: testOid("testNode", 10),
				},
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		objNode := buildOIDPath(root, 1, 2)
		objNode.setName("mandObj")
		objNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "mandObj", objNode)
		obj := newObject("mandObj")
		obj.setStatus(types.StatusMandatory)
		objNode.setObject(obj)

		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
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
			Definitions: []module.Definition{
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
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()
		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagComplianceGroupInvalid)
	})

	t.Run("duplicate refinement emits refinement-exists", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
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
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()
		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagRefinementExists)
	})

	t.Run("duplicate optional group emits optional-group-exists", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
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
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()
		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagOptionalGroupExists)
	})

	t.Run("refinement not in any group emits refinement-not-listed", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
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
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("myGroup")
		ctx.registerModuleNodeSymbol(mod, "myGroup", grpNode)

		memberNode := buildOIDPath(root, 1, 2)
		memberNode.setName("memberObj")
		memberNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "memberObj", memberNode)
		memberObj := newObject("memberObj")
		memberObj.setStatus(types.StatusCurrent)
		memberNode.setObject(memberObj)

		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedGroups(ctx)
		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagRefinementNotListed)
	})

	t.Run("refinement in mandatory group no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
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
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("myGroup")
		ctx.registerModuleNodeSymbol(mod, "myGroup", grpNode)

		memberNode := buildOIDPath(root, 1, 2)
		memberNode.setName("listedObj")
		memberNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "listedObj", memberNode)
		memberObj := newObject("listedObj")
		memberObj.setStatus(types.StatusCurrent)
		memberNode.setObject(memberObj)

		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedGroups(ctx)
		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagRefinementNotListed)
	})

	t.Run("no duplicates no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
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
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()
		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
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
			Definitions: []module.Definition{
				&module.ObjectGroup{
					DefBase: module.DefBase{Name: "testGroup"},
					Objects: []string{"importedObj"},
					Status:  types.StatusCurrent,
				},
			},
		}

		ctx := newComplianceTestContext(modA, modB)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("testGroup")
		ctx.registerModuleNodeSymbol(modB, "testGroup", grpNode)

		// Register importedObj in modA (source) but not in modB (group's module).
		objNode := buildOIDPath(root, 1, 2)
		objNode.setName("importedObj")
		objNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(modA, "importedObj", objNode)
		// Make it available in modB via import.
		if ctx.moduleImports[modB] == nil {
			ctx.moduleImports[modB] = make(map[string]*module.Module)
		}
		ctx.moduleImports[modB]["importedObj"] = modA

		checkGroupMemberLocality(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagComplianceMemberNotLocal)
	})

	t.Run("local member no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.ObjectGroup{
					DefBase: module.DefBase{Name: "testGroup"},
					Objects: []string{"localObj"},
					Status:  types.StatusCurrent,
				},
			},
		}

		ctx := newComplianceTestContext(mod)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("testGroup")
		ctx.registerModuleNodeSymbol(mod, "testGroup", grpNode)

		objNode := buildOIDPath(root, 1, 2)
		objNode.setName("localObj")
		objNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "localObj", objNode)

		checkGroupMemberLocality(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagComplianceMemberNotLocal)
	})

	t.Run("notification group imported member emits diagnostic", func(t *testing.T) {
		modA := &module.Module{Name: "SOURCE-MIB"}
		modB := &module.Module{
			Name: "TEST-MIB",
			Definitions: []module.Definition{
				&module.NotificationGroup{
					DefBase:       module.DefBase{Name: "testNotifGroup"},
					Notifications: []string{"importedNotif"},
					Status:        types.StatusCurrent,
				},
			},
		}

		ctx := newComplianceTestContext(modA, modB)

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("testNotifGroup")
		ctx.registerModuleNodeSymbol(modB, "testNotifGroup", grpNode)

		notifNode := buildOIDPath(root, 1, 2)
		notifNode.setName("importedNotif")
		notifNode.setKind(types.KindNotification)
		ctx.registerModuleNodeSymbol(modA, "importedNotif", notifNode)
		if ctx.moduleImports[modB] == nil {
			ctx.moduleImports[modB] = make(map[string]*module.Module)
		}
		ctx.moduleImports[modB]["importedNotif"] = modA

		checkGroupMemberLocality(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagComplianceMemberNotLocal)
	})
}

// makeTypedObject creates a minimal Object with a type of the given base type
// and optional sizes. Used for unit testing index helpers.
func makeTypedObject(base BaseType, sizes *[]Range) *Object {
	obj := newObject("x")
	typ := newType("xType")
	typ.setBase(base)
	obj.setType(typ)
	if sizes != nil {
		obj.setEffectiveSizes(*sizes)
	}
	return obj
}

func TestCheckUnusedImports(t *testing.T) {
	t.Run("unused import emits diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
				module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
				module.NewImport("SNMPv2-TC", "DisplayString", types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:  types.StatusCurrent,
				Oid:     testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		d := hasDiag(t, m.Diagnostics(), types.DiagImportUnused)
		testutil.True(t, d.Message != "", "expected non-empty message")
	})

	t.Run("all used no diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
				module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:  types.StatusCurrent,
				Oid:     testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagImportUnused)
	})
}

func TestCheckBasetypeNotImported(t *testing.T) {
	t.Run("not imported emits diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(nil,
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:  types.StatusCurrent,
				Oid:     testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		d := hasDiag(t, m.Diagnostics(), types.DiagBasetypeNotImported)
		testutil.True(t, d.Message != "", "expected non-empty message")
	})

	t.Run("imported no diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:  types.StatusCurrent,
				Oid:     testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagBasetypeNotImported)
	})
}

// --- description-missing ---

func TestCheckDescriptionMissing(t *testing.T) {
	t.Run("missing in SMIv2 emits diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
			},
			&module.ObjectType{
				DefBase:     module.DefBase{Name: "noDesc"},
				Syntax:      &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:      types.StatusCurrent,
				Description: "",
				Oid:         testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagDescriptionMissing)
	})

	t.Run("SMIv1 no diagnostic", func(t *testing.T) {
		mod := testSMIv1Module(
			[]module.Import{
				module.NewImport("RFC1155-SMI", "INTEGER", types.Span{}),
			},
			&module.ObjectType{
				DefBase:     module.DefBase{Name: "noDesc"},
				Syntax:      &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Status:      types.StatusMandatory,
				Description: "",
				Oid:         testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagDescriptionMissing)
	})

	t.Run("with description no diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
			},
			&module.ObjectType{
				DefBase:        module.DefBase{Name: "hasDesc"},
				Syntax:         &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:         types.StatusCurrent,
				Description:    "This object has a description",
				HasDescription: true,
				Oid:            testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagDescriptionMissing)
	})
}

func TestCheckTCNested(t *testing.T) {
	t.Run("TC from TC emits diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "ParentTC"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "OCTET STRING"},
				Status:              types.StatusCurrent,
				Description:         "A parent TC",
				IsTextualConvention: true,
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "ChildTC"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "ParentTC"},
				Status:              types.StatusCurrent,
				Description:         "A nested TC",
				IsTextualConvention: true,
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagTCNested)
	})

	t.Run("TC from base type no diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "MyString"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "OCTET STRING"},
				Status:              types.StatusCurrent,
				Description:         "A TC from base type",
				IsTextualConvention: true,
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagTCNested)
	})
}

func TestCheckTypeAssignmentSMIv2(t *testing.T) {
	t.Run("plain typedef emits diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(nil,
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "MyType"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Status:              types.StatusCurrent,
				IsTextualConvention: false,
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagTypeAssignmentSMIv2)
	})

	t.Run("TC no diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "MyTC"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "OCTET STRING"},
				Status:              types.StatusCurrent,
				Description:         "A proper TC",
				IsTextualConvention: true,
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagTypeAssignmentSMIv2)
	})
}

// --- table/row naming ---

func TestCheckTableRowNaming_BadTableName(t *testing.T) {
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase:     module.DefBase{Name: "badTbl"},
			Syntax:      &module.TypeSyntaxSequenceOf{EntryType: "BadRow"},
			Status:      types.StatusCurrent,
			Description: "A table without Table suffix",
			Oid:         testOid("testRoot", 1),
		},
		&module.ObjectType{
			DefBase:     module.DefBase{Name: "badRow"},
			Syntax:      &module.TypeSyntaxTypeRef{Name: "BadRow"},
			Status:      types.StatusCurrent,
			Description: "A row without Entry suffix",
			Index:       []module.IndexItem{{Object: "badIdx"}},
			Oid:         testOid("badTbl", 1),
		},
		&module.ObjectType{
			DefBase:     module.DefBase{Name: "badIdx"},
			Syntax:      &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Status:      types.StatusCurrent,
			Description: "Index",
			Oid:         testOid("badRow", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagTableNameTable)
	hasDiag(t, m.Diagnostics(), types.DiagRowNameEntry)
}

func TestCheckTableRowNaming_NoDiag(t *testing.T) {
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase:     module.DefBase{Name: "goodTable"},
			Syntax:      &module.TypeSyntaxSequenceOf{EntryType: "GoodEntry"},
			Status:      types.StatusCurrent,
			Description: "A properly named table",
			Oid:         testOid("testRoot", 1),
		},
		&module.ObjectType{
			DefBase:     module.DefBase{Name: "goodEntry"},
			Syntax:      &module.TypeSyntaxTypeRef{Name: "GoodEntry"},
			Status:      types.StatusCurrent,
			Description: "A properly named entry",
			Index:       []module.IndexItem{{Object: "goodIndex"}},
			Oid:         testOid("goodTable", 1),
		},
		&module.ObjectType{
			DefBase:     module.DefBase{Name: "goodIndex"},
			Syntax:      &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Status:      types.StatusCurrent,
			Description: "Index",
			Oid:         testOid("goodEntry", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagTableNameTable, types.DiagRowNameEntry, types.DiagRowNameTableName)
}

func TestCheckTableRowNaming_PrefixMismatch(t *testing.T) {
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase:     module.DefBase{Name: "fooTable"},
			Syntax:      &module.TypeSyntaxSequenceOf{EntryType: "BarEntry"},
			Status:      types.StatusCurrent,
			Description: "Table with mismatched prefix",
			Oid:         testOid("testRoot", 1),
		},
		&module.ObjectType{
			DefBase:     module.DefBase{Name: "barEntry"},
			Syntax:      &module.TypeSyntaxTypeRef{Name: "BarEntry"},
			Status:      types.StatusCurrent,
			Description: "Entry with mismatched prefix",
			Index:       []module.IndexItem{{Object: "barIndex"}},
			Oid:         testOid("fooTable", 1),
		},
		&module.ObjectType{
			DefBase:     module.DefBase{Name: "barIndex"},
			Syntax:      &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Status:      types.StatusCurrent,
			Description: "Index",
			Oid:         testOid("barEntry", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagRowNameTableName)
}

func TestCheckNamedNumbersAscending(t *testing.T) {
	t.Run("descending order emits diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(nil,
			&module.ObjectType{
				DefBase: module.DefBase{Name: "badOrder"},
				Syntax: &module.TypeSyntaxIntegerEnum{
					NamedNumbers: []module.NamedNumber{
						{Name: "high", Value: 5},
						{Name: "low", Value: 1},
					},
				},
				Status:      types.StatusCurrent,
				Description: "Descending enum",
				Oid:         testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagNamedNumbersAscending)
	})

	t.Run("ascending order no diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(nil,
			&module.ObjectType{
				DefBase: module.DefBase{Name: "goodOrder"},
				Syntax: &module.TypeSyntaxIntegerEnum{
					NamedNumbers: []module.NamedNumber{
						{Name: "low", Value: 1},
						{Name: "high", Value: 5},
					},
				},
				Status:      types.StatusCurrent,
				Description: "Ascending enum",
				Oid:         testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagNamedNumbersAscending)
	})
}

func TestCheckHyphenInLabel(t *testing.T) {
	t.Run("hyphen in SMIv2 emits diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(nil,
			&module.ObjectType{
				DefBase: module.DefBase{Name: "badLabel"},
				Syntax: &module.TypeSyntaxIntegerEnum{
					NamedNumbers: []module.NamedNumber{
						{Name: "my-value", Value: 1},
					},
				},
				Status:      types.StatusCurrent,
				Description: "Enum with hyphenated label",
				Oid:         testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagHyphenInLabel)
	})

	t.Run("hyphen in SMIv1 no diagnostic", func(t *testing.T) {
		mod := testSMIv1Module(nil,
			&module.ObjectType{
				DefBase: module.DefBase{Name: "okLabel"},
				Syntax: &module.TypeSyntaxIntegerEnum{
					NamedNumbers: []module.NamedNumber{
						{Name: "my-value", Value: 1},
					},
				},
				Status:      types.StatusMandatory,
				Description: "Hyphen is OK in SMIv1",
				Oid:         testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagHyphenInLabel)
	})

	t.Run("no hyphen no diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(nil,
			&module.ObjectType{
				DefBase: module.DefBase{Name: "goodLabel"},
				Syntax: &module.TypeSyntaxIntegerEnum{
					NamedNumbers: []module.NamedNumber{
						{Name: "myValue", Value: 1},
					},
				},
				Status:      types.StatusCurrent,
				Description: "Clean label",
				Oid:         testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagHyphenInLabel)
	})
}

func TestValidateDisplayHintInteger(t *testing.T) {
	valid := []string{"d", "x", "o", "b", "d-1", "d-2", "d-10"}
	for _, h := range valid {
		if !validateDisplayHintInteger(h) {
			t.Errorf("expected valid integer hint %q", h)
		}
	}

	invalid := []string{"", "a", "t", "dd", "d-", "d-x", "x1", "d--2", "D", "X"}
	for _, h := range invalid {
		if validateDisplayHintInteger(h) {
			t.Errorf("expected invalid integer hint %q", h)
		}
	}
}

func TestValidateDisplayHintOctetString(t *testing.T) {
	valid := []string{
		"255a",                         // DisplayString
		"1x:",                          // PhysAddress - hex with colon separator
		"2d-1d-1d,1d:1d:1d.1d,1a1d:1d", // DateAndTime
		"1d.1d.1d.1d",                  // IpAddress
		"2x:2x:2x:2x:2x:2x:2x:2x",      // IPv6 address
		"1x",                           // single hex octet
		"4d",                           // 4-octet decimal
		"*1x:",                         // repeat indicator
		"*1x:.",                        // repeat with separator and terminator
		"0a[1a",                        // zero-width prefix, last spec consumes
		"0a[2x]0a:2d",                  // TransportAddress style, last spec consumes
	}
	for _, h := range valid {
		if !validateDisplayHintOctetString(h) {
			t.Errorf("expected valid octet string hint %q", h)
		}
	}

	invalid := []string{
		"",     // empty
		"a",    // no octet count
		"1",    // missing format char
		"1z",   // bad format char
		"*",    // repeat without count
		"*a",   // repeat without digits before format
		"1x*",  // * at position where digit expected for next spec
		"0a",   // zero-width final spec would loop forever
		"0d",   // zero-width final spec (decimal)
		"1d0a", // last spec is zero-width
	}
	for _, h := range invalid {
		if validateDisplayHintOctetString(h) {
			t.Errorf("expected invalid octet string hint %q", h)
		}
	}
}

func TestCheckInvalidFormat(t *testing.T) {
	t.Run("bad integer hint", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
				module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "BadHint"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "Integer32"},
				DisplayHint:         "zz",
				Status:              types.StatusCurrent,
				Description:         "bad integer hint",
				IsTextualConvention: true,
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagInvalidFormat)
	})

	t.Run("valid integer hint", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
				module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "GoodHint"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "Integer32"},
				DisplayHint:         "d-2",
				Status:              types.StatusCurrent,
				Description:         "valid integer hint",
				IsTextualConvention: true,
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagInvalidFormat)
	})

	t.Run("bad octet string hint", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "BadOSHint"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "OCTET STRING"},
				DisplayHint:         "xyz",
				Status:              types.StatusCurrent,
				Description:         "bad octet string hint",
				IsTextualConvention: true,
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagInvalidFormat)
	})

	t.Run("valid octet string hint", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "GoodOSHint"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "OCTET STRING"},
				DisplayHint:         "1x:",
				Status:              types.StatusCurrent,
				Description:         "valid octet string hint",
				IsTextualConvention: true,
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagInvalidFormat)
	})

	t.Run("unsupported basetype", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "Counter32", types.Span{}),
				module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "BadCounterHint"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "Counter32"},
				DisplayHint:         "d",
				Status:              types.StatusCurrent,
				Description:         "counter with hint",
				IsTextualConvention: true,
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagInvalidFormat)
	})
}

func TestCheckTypeWithoutFormat(t *testing.T) {
	t.Run("octet string textual convention without DISPLAY-HINT emits diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "DisplayStringLike"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "OCTET STRING"},
				Status:              types.StatusCurrent,
				Description:         "textual convention without display hint",
				IsTextualConvention: true,
			},
		)

		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagTypeWithoutFormat)
	})

	t.Run("counter textual convention without DISPLAY-HINT is allowed", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "Counter32", types.Span{}),
				module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "CounterLike"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "Counter32"},
				Status:              types.StatusCurrent,
				Description:         "counter textual convention without display hint",
				IsTextualConvention: true,
			},
		)

		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagTypeWithoutFormat)
	})
}

func TestCheckNodeImplicit(t *testing.T) {
	t.Run("implicit node emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name:     "TEST-MIB",
			Language: types.LanguageSMIv2,
			Imports: []module.Import{
				module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
				module.NewImport("SNMPv2-SMI", "MODULE-IDENTITY", types.Span{}),
			},
			Definitions: []module.Definition{
				&module.ModuleIdentity{
					DefBase:     module.DefBase{Name: "testMIB"},
					Description: "test",
					Oid: module.NewOidAssignment([]module.OidComponent{
						&module.OidComponentName{NameValue: "enterprises"},
						&module.OidComponentNumber{Value: 99999},
						&module.OidComponentNumber{Value: 1},
					}, types.Span{}),
				},
			},
		}
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagNodeImplicit)
	})

	t.Run("named node no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name:     "TEST-MIB",
			Language: types.LanguageSMIv2,
			Imports: []module.Import{
				module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
				module.NewImport("SNMPv2-SMI", "MODULE-IDENTITY", types.Span{}),
			},
			Definitions: []module.Definition{
				&module.ModuleIdentity{
					DefBase:     module.DefBase{Name: "testMIB"},
					Description: "test",
					Oid:         testOid("testEnterprise", 1),
				},
				&module.ValueAssignment{
					DefBase: module.DefBase{Name: "testEnterprise"},
					Oid:     testOid("enterprises", 99999),
				},
			},
		}
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagNodeImplicit)
	})
}

func TestCheckModuleIdentityRegistration(t *testing.T) {
	t.Run("uncontrolled mgmt emits diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name:     "TEST-MIB",
			Language: types.LanguageSMIv2,
			Imports: []module.Import{
				module.NewImport("SNMPv2-SMI", "MODULE-IDENTITY", types.Span{}),
			},
			Definitions: []module.Definition{
				&module.ModuleIdentity{
					DefBase:     module.DefBase{Name: "testMIB"},
					Description: "test",
					Oid: module.NewOidAssignment([]module.OidComponent{
						&module.OidComponentNamedNumber{NameValue: "iso", NumberValue: 1},
						&module.OidComponentNamedNumber{NameValue: "org", NumberValue: 3},
						&module.OidComponentNamedNumber{NameValue: "dod", NumberValue: 6},
						&module.OidComponentNamedNumber{NameValue: "internet", NumberValue: 1},
						&module.OidComponentNamedNumber{NameValue: "mgmt", NumberValue: 2},
						&module.OidComponentNumber{Value: 99},
					}, types.Span{}),
				},
			},
		}
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagModuleIdentityReg)
	})

	t.Run("mib-2 no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name:     "TEST-MIB",
			Language: types.LanguageSMIv2,
			Imports: []module.Import{
				module.NewImport("SNMPv2-SMI", "MODULE-IDENTITY", types.Span{}),
				module.NewImport("SNMPv2-SMI", "mib-2", types.Span{}),
			},
			Definitions: []module.Definition{
				&module.ModuleIdentity{
					DefBase:     module.DefBase{Name: "testMIB"},
					Description: "test",
					Oid:         testOid("mib-2", 999),
				},
			},
		}
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagModuleIdentityReg)
	})

	t.Run("enterprises no diagnostic", func(t *testing.T) {
		mod := &module.Module{
			Name:     "TEST-MIB",
			Language: types.LanguageSMIv2,
			Imports: []module.Import{
				module.NewImport("SNMPv2-SMI", "MODULE-IDENTITY", types.Span{}),
				module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			},
			Definitions: []module.Definition{
				&module.ModuleIdentity{
					DefBase:     module.DefBase{Name: "testMIB"},
					Description: "test",
					Oid:         testOid("enterprises", 12345),
				},
			},
		}
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagModuleIdentityReg)
	})
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

func TestCheckIndexAccessAndDefval(t *testing.T) {
	tests := []struct {
		name     string
		lang     types.Language
		access   Access
		defval   bool
		strict   bool // use StrictConfig (needed for warning-level diags)
		wantCode string
	}{
		{"SMIv2 read-only emits index-accessible", types.LanguageSMIv2, AccessReadOnly, false, false, types.DiagIndexAccessible},
		{"SMIv2 not-accessible is ok", types.LanguageSMIv2, AccessNotAccessible, false, false, ""},
		{"SMIv1 not-accessible emits index-not-accessible", types.LanguageSMIv1, AccessNotAccessible, false, false, types.DiagIndexNotAccessible},
		{"SMIv1 read-only is ok", types.LanguageSMIv1, AccessReadOnly, false, false, ""},
		{"INDEX with DEFVAL emits index-defval", types.LanguageSMIv2, AccessNotAccessible, true, true, types.DiagIndexDefval},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx *resolverContext
			if tt.strict {
				ctx = newTestContextWithConfig(VerboseConfig())
			} else {
				ctx = newTestContext()
			}
			mod := &module.Module{Name: "TEST-MIB", Language: tt.lang}
			ctx.modules = append(ctx.modules, mod)
			resolvedMod := newModule(mod.Name)
			ctx.moduleToResolved[mod] = resolvedMod

			idxObj := newObject("myIndex")
			typ := newType("Integer32")
			typ.setBase(BaseInteger32)
			idxObj.setType(typ)
			idxObj.setAccess(tt.access)
			idxObj.setEffectiveRanges([]Range{{Min: 1, Max: 100}})
			if tt.defval {
				dv := newDefValInt(1, "1")
				idxObj.setDefaultValue(&dv)
			}
			resolvedMod.addObject(idxObj)

			root := ctx.mib.Root()
			idxNode := buildOIDPath(root, 1, 1, 1, 1)
			idxNode.setName("myIndex")
			idxNode.setObject(idxObj)
			ctx.registerModuleNodeSymbol(mod, "myIndex", idxNode)

			objRefs := []objectTypeRef{
				{mod: mod, obj: &module.ObjectType{
					DefBase: module.DefBase{Name: "myEntry"},
					Index:   []module.IndexItem{{Object: "myIndex"}},
				}},
			}

			checkIndexConstraints(ctx, objRefs)

			if tt.wantCode != "" {
				hasDiag(t, ctx.Diagnostics(), tt.wantCode)
			} else {
				noDiag(t, ctx.Diagnostics(), types.DiagIndexAccessible, types.DiagIndexNotAccessible)
			}
		})
	}
}

func TestCheckAccessWriteOnlySMIv1(t *testing.T) {
	mod := testSMIv1Module(
		[]module.Import{
			module.NewImport("RFC1155-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase:       module.DefBase{Name: "woObject"},
			Syntax:        &module.TypeSyntaxTypeRef{Name: "INTEGER"},
			Access:        types.AccessWriteOnly,
			AccessKeyword: types.AccessKeywordAccess,
			Status:        types.StatusMandatory,
			Oid:           testOid("testRoot", 1),
		},
	)
	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagAccessWriteOnlySMIv1)
}

// testInetAddressMIB creates a minimal INET-ADDRESS-MIB module with the
// type definitions needed for address pairing tests.
func testInetAddressMIB() *module.Module {
	mod := module.NewModule("INET-ADDRESS-MIB", types.Synthetic)
	mod.Language = types.LanguageSMIv2
	mod.Imports = []module.Import{
		module.NewImport("SNMPv2-SMI", "MODULE-IDENTITY", types.Span{}),
		module.NewImport("SNMPv2-SMI", "mib-2", types.Span{}),
		module.NewImport("SNMPv2-SMI", "Unsigned32", types.Span{}),
		module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
	}
	mod.Definitions = []module.Definition{
		&module.ModuleIdentity{
			DefBase: module.DefBase{Name: "inetAddressMIB"},
			Oid: module.NewOidAssignment([]module.OidComponent{
				&module.OidComponentName{NameValue: "mib-2"},
				&module.OidComponentNumber{Value: 76},
			}, types.Span{}),
		},
		// InetAddressType ::= TEXTUAL-CONVENTION SYNTAX INTEGER { unknown(0), ipv4(1), ipv6(2), ... }
		&module.TypeDef{
			DefBase: module.DefBase{Name: "InetAddressType"},
			Syntax: &module.TypeSyntaxIntegerEnum{
				NamedNumbers: []module.NamedNumber{
					{Name: "unknown", Value: 0},
					{Name: "ipv4", Value: 1},
					{Name: "ipv6", Value: 2},
					{Name: "ipv4z", Value: 3},
					{Name: "ipv6z", Value: 4},
					{Name: "dns", Value: 16},
				},
			},
			Status:              types.StatusCurrent,
			IsTextualConvention: true,
		},
		// InetAddress ::= TEXTUAL-CONVENTION SYNTAX OCTET STRING (SIZE (0..255))
		&module.TypeDef{
			DefBase: module.DefBase{Name: "InetAddress"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxOctetString{},
				Constraint: &module.ConstraintSize{Ranges: []module.Range{module.NewRangeUnsigned(0, 255, types.Span{})}},
			},
			Status:              types.StatusCurrent,
			IsTextualConvention: true,
		},
		// InetAddressIPv4 ::= TEXTUAL-CONVENTION SYNTAX OCTET STRING (SIZE (4))
		&module.TypeDef{
			DefBase: module.DefBase{Name: "InetAddressIPv4"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxOctetString{},
				Constraint: &module.ConstraintSize{Ranges: []module.Range{{Min: &module.RangeValueUnsigned{Value: 4}}}},
			},
			Status:              types.StatusCurrent,
			IsTextualConvention: true,
		},
		// InetAddressIPv6 ::= TEXTUAL-CONVENTION SYNTAX OCTET STRING (SIZE (16))
		&module.TypeDef{
			DefBase: module.DefBase{Name: "InetAddressIPv6"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxOctetString{},
				Constraint: &module.ConstraintSize{Ranges: []module.Range{{Min: &module.RangeValueUnsigned{Value: 16}}}},
			},
			Status:              types.StatusCurrent,
			IsTextualConvention: true,
		},
	}
	return mod
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
	mod.Definitions = []module.Definition{
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
	}
	return mod
}

func TestCheckIpAddressDeprecation(t *testing.T) {
	t.Run("IpAddress in SMIv2 emits diagnostic", func(t *testing.T) {
		mod := testSMIv2Module(
			[]module.Import{
				module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
				module.NewImport("SNMPv2-SMI", "IpAddress", types.Span{}),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "myAddr"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "IpAddress"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		hasDiag(t, m.Diagnostics(), types.DiagIpAddressInSyntax)
	})

	t.Run("IpAddress in SMIv1 does not emit diagnostic", func(t *testing.T) {
		mod := testSMIv1Module(
			[]module.Import{
				module.NewImport("RFC1155-SMI", "OBJECT-TYPE", types.Span{}),
				module.NewImport("RFC1155-SMI", "IpAddress", types.Span{}),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "myAddr"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "IpAddress"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordAccess,
				Status:        types.StatusMandatory,
				Oid:           testOid("testRoot", 1),
			},
		)
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagIpAddressInSyntax)
	})
}

// --- checkRangeConstraints tests ---

func TestCheckRangeConstraints_SizeIllegalOnInteger(t *testing.T) {
	// SIZE constraint on an Integer32 type should emit size-illegal.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "BadSize"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Constraint: &module.ConstraintSize{Ranges: []module.Range{module.NewRangeSigned(0, 255, types.Span{})}},
			},
			Status: types.StatusCurrent,
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagSizeIllegal)
}

func TestCheckRangeConstraints_SizeValidOnOctetString(t *testing.T) {
	// SIZE constraint on OCTET STRING is valid, no diagnostic.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "goodSize"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxOctetString{},
				Constraint: &module.ConstraintSize{Ranges: []module.Range{module.NewRangeSigned(0, 255, types.Span{})}},
			},
			Access: types.AccessReadOnly,
			Status: types.StatusCurrent,
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagSizeIllegal)
}

func TestCheckRangeConstraints_RangeIllegalOnOctetString(t *testing.T) {
	// Value range constraint on OCTET STRING should emit range-illegal.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "BadRange"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxOctetString{},
				Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(0, 255, types.Span{})}},
			},
			Status: types.StatusCurrent,
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagRangeIllegal)
}

func TestCheckRangeConstraints_CounterRange(t *testing.T) {
	// Range constraint on Counter32 should emit counter-range-illegal.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "Counter32", types.Span{}),
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "BadCounter"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxTypeRef{Name: "Counter32"},
				Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(0, 100, types.Span{})}},
			},
			Status: types.StatusCurrent,
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagCounterRangeIllegal)
}

func TestCheckRangeConstraints_TimeticksRange(t *testing.T) {
	// Range constraint on TimeTicks should emit timeticks-range-illegal.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "TimeTicks", types.Span{}),
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "BadTicks"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxTypeRef{Name: "TimeTicks"},
				Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(0, 100, types.Span{})}},
			},
			Status: types.StatusCurrent,
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagTimeticksRangeIllegal)
}

func TestCheckRangeConstraints_ExchangedLimits(t *testing.T) {
	// Range with min > max should emit range-exchanged.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "Exchanged"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(100, 0, types.Span{})}},
			},
			Status: types.StatusCurrent,
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagRangeExchanged)
}

func TestCheckRangeConstraints_RangeBoundsExceeded(t *testing.T) {
	// Range exceeding Integer32 bounds should emit range-bounds.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "OutOfBounds"},
			Syntax: &module.TypeSyntaxConstrained{
				Base: &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Constraint: &module.ConstraintRange{Ranges: []module.Range{
					module.NewRangeSigned(0, 3000000000, types.Span{}),
				}},
			},
			Status: types.StatusCurrent,
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagRangeBounds)
}

func TestCheckRangeConstraints_RangeOverlap(t *testing.T) {
	// Overlapping ranges should emit range-overlap.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "Overlap"},
			Syntax: &module.TypeSyntaxConstrained{
				Base: &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Constraint: &module.ConstraintRange{Ranges: []module.Range{
					module.NewRangeSigned(0, 10, types.Span{}),
					module.NewRangeSigned(5, 20, types.Span{}),
				}},
			},
			Status: types.StatusCurrent,
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagRangeOverlap)
}

func TestCheckRangeConstraints_RangeAscending(t *testing.T) {
	// Ranges not in ascending order should emit range-ascending.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "Descending"},
			Syntax: &module.TypeSyntaxConstrained{
				Base: &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Constraint: &module.ConstraintRange{Ranges: []module.Range{
					module.NewRangeSigned(10, 20, types.Span{}),
					module.NewRangeSigned(0, 5, types.Span{}),
				}},
			},
			Status: types.StatusCurrent,
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagRangeAscending)
}

func TestCheckRangeConstraints_ValidRange(t *testing.T) {
	// Valid non-overlapping ascending ranges should produce no range diagnostics.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "GoodRange"},
			Syntax: &module.TypeSyntaxConstrained{
				Base: &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Constraint: &module.ConstraintRange{Ranges: []module.Range{
					module.NewRangeSigned(0, 10, types.Span{}),
					module.NewRangeSigned(20, 30, types.Span{}),
				}},
			},
			Status: types.StatusCurrent,
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(),
		types.DiagRangeExchanged,
		types.DiagRangeBounds,
		types.DiagRangeOverlap,
		types.DiagRangeAscending,
		types.DiagSizeIllegal,
		types.DiagRangeIllegal,
	)
}

func TestCheckRangeConstraints_ObjectTypeSizeIllegal(t *testing.T) {
	// SIZE constraint on an OBJECT-TYPE with Integer32 syntax.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "badObjSize"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Constraint: &module.ConstraintSize{Ranges: []module.Range{module.NewRangeSigned(0, 10, types.Span{})}},
			},
			Access: types.AccessReadOnly,
			Status: types.StatusCurrent,
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagSizeIllegal)
}

func TestCheckRangeConstraints_ObjectTypeRangeOnGauge(t *testing.T) {
	// Valid range constraint on Gauge32 OBJECT-TYPE - no diagnostic.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Gauge32", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "goodGauge"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxTypeRef{Name: "Gauge32"},
				Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeUnsigned(0, 100, types.Span{})}},
			},
			Access: types.AccessReadOnly,
			Status: types.StatusCurrent,
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(),
		types.DiagRangeIllegal,
		types.DiagCounterRangeIllegal,
		types.DiagRangeBounds,
	)
}

// --- checkDefvalConstraints tests ---

func TestCheckDefvalConstraints_CounterDefval(t *testing.T) {
	// DEFVAL on Counter32 should emit counter-defval-illegal.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Counter32", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "badCounter"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Counter32"},
			Access:  types.AccessReadOnly,
			Status:  types.StatusCurrent,
			DefVal:  &module.DefValInteger{Value: 0},
			Oid:     testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagCounterDefvalIllegal)
}

func TestCheckDefvalConstraints_BasetypeExceeded(t *testing.T) {
	// DEFVAL exceeding Integer32 range should emit defval-basetype.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "bigVal"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Access:  types.AccessReadOnly,
			Status:  types.StatusCurrent,
			DefVal:  &module.DefValInteger{Value: 3000000000},
			Oid:     testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagDefvalBasetype)
}

func TestCheckDefvalConstraints_OutsideRange(t *testing.T) {
	// DEFVAL outside RANGE constraint should emit defval-range.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "outOfRange"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(1, 10, types.Span{})}},
			},
			Access: types.AccessReadOnly,
			Status: types.StatusCurrent,
			DefVal: &module.DefValInteger{Value: 50},
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagDefvalRange)
}

func TestCheckDefvalConstraints_WithinRange(t *testing.T) {
	// DEFVAL within RANGE constraint should not emit defval-range.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "inRange"},
			Syntax: &module.TypeSyntaxConstrained{
				Base:       &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Constraint: &module.ConstraintRange{Ranges: []module.Range{module.NewRangeSigned(1, 10, types.Span{})}},
			},
			Access: types.AccessReadOnly,
			Status: types.StatusCurrent,
			DefVal: &module.DefValInteger{Value: 5},
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagDefvalRange, types.DiagDefvalBasetype)
}

func TestCheckDefvalConstraints_EnumLabelInvalid(t *testing.T) {
	// DEFVAL with enum label not in the type should emit defval-enum.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "badEnum"},
			Syntax: &module.TypeSyntaxIntegerEnum{
				NamedNumbers: []module.NamedNumber{
					{Name: "up", Value: 1},
					{Name: "down", Value: 2},
				},
			},
			Access: types.AccessReadOnly,
			Status: types.StatusCurrent,
			DefVal: &module.DefValEnum{Name: "unknown"},
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagDefvalEnum)
}

func TestCheckDefvalConstraints_EnumLabelValid(t *testing.T) {
	// DEFVAL with valid enum label should not emit defval-enum.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "goodEnum"},
			Syntax: &module.TypeSyntaxIntegerEnum{
				NamedNumbers: []module.NamedNumber{
					{Name: "up", Value: 1},
					{Name: "down", Value: 2},
				},
			},
			Access: types.AccessReadOnly,
			Status: types.StatusCurrent,
			DefVal: &module.DefValEnum{Name: "up"},
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagDefvalEnum)
}

func TestCheckDefvalConstraints_BitsLabelInvalid(t *testing.T) {
	// DEFVAL with BITS label not in the type should emit defval-bits.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "badBits"},
			Syntax: &module.TypeSyntaxBits{
				NamedBits: []module.NamedBit{
					{Name: "read", Position: 0},
					{Name: "write", Position: 1},
				},
			},
			Access: types.AccessReadOnly,
			Status: types.StatusCurrent,
			DefVal: &module.DefValBits{Labels: []string{"execute"}},
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagDefvalBits)
}

func TestCheckDefvalConstraints_BitsLabelValid(t *testing.T) {
	// DEFVAL with valid BITS labels should not emit defval-bits.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "goodBits"},
			Syntax: &module.TypeSyntaxBits{
				NamedBits: []module.NamedBit{
					{Name: "read", Position: 0},
					{Name: "write", Position: 1},
				},
			},
			Access: types.AccessReadOnly,
			Status: types.StatusCurrent,
			DefVal: &module.DefValBits{Labels: []string{"read", "write"}},
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagDefvalBits)
}

// --- checkSequenceFields tests ---

// buildTableModule constructs a module with a table, row, and columns.
// The SEQUENCE type has the given fields, while columns are the actual
// OBJECT-TYPE children of the row.
func buildTableModule(seqFields []module.SequenceField, columns []string) *module.Module {
	imports := []module.Import{
		module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
	}

	defs := []module.Definition{
		&module.ValueAssignment{
			DefBase: module.DefBase{Name: "testRoot"},
			Oid:     testOid("enterprises", 99999),
		},
		// SEQUENCE type definition
		&module.TypeDef{
			DefBase: module.DefBase{Name: "TestEntry"},
			Syntax:  &module.TypeSyntaxSequence{Fields: seqFields},
		},
		// Table
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testTable"},
			Syntax:  &module.TypeSyntaxSequenceOf{EntryType: "TestEntry"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
		// Row
		&module.ObjectType{
			DefBase: module.DefBase{Name: "testEntry"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "TestEntry"},
			Status:  types.StatusCurrent,
			Index:   []module.IndexItem{{Object: "col1"}},
			Oid:     testOid("testTable", 1),
		},
	}

	// Add columns in order
	for i, name := range columns {
		defs = append(defs, &module.ObjectType{
			DefBase: module.DefBase{Name: name},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Access:  types.AccessReadOnly,
			Status:  types.StatusCurrent,
			Oid:     testOid("testEntry", uint32(i+1)),
		})
	}

	return &module.Module{
		Name:        "TEST-MIB",
		Language:    types.LanguageSMIv2,
		Imports:     imports,
		Definitions: defs,
	}
}

func TestCheckSequenceFields_Valid(t *testing.T) {
	// SEQUENCE fields match columns exactly - no diagnostics.
	fields := []module.SequenceField{
		{Name: "col1", Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"}},
		{Name: "col2", Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"}},
	}
	mod := buildTableModule(fields, []string{"col1", "col2"})

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(),
		types.DiagSequenceNoColumn,
		types.DiagSequenceMissingColumn,
		types.DiagSequenceOrder,
		types.DiagSequenceTypeMismatch,
	)
}

func TestCheckSequenceFields_NoColumn(t *testing.T) {
	// SEQUENCE field "col3" has no matching column object.
	fields := []module.SequenceField{
		{Name: "col1", Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"}},
		{Name: "col3", Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"}},
	}
	mod := buildTableModule(fields, []string{"col1", "col2"})

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagSequenceNoColumn)
}

func TestCheckSequenceFields_MissingColumn(t *testing.T) {
	// Column "col2" exists but is not listed in the SEQUENCE.
	fields := []module.SequenceField{
		{Name: "col1", Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"}},
	}
	mod := buildTableModule(fields, []string{"col1", "col2"})

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagSequenceMissingColumn)
}

func TestCheckSequenceFields_OrderMismatch(t *testing.T) {
	// SEQUENCE lists col2 before col1, but OID order is col1(1), col2(2).
	fields := []module.SequenceField{
		{Name: "col2", Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"}},
		{Name: "col1", Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"}},
	}
	mod := buildTableModule(fields, []string{"col1", "col2"})

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagSequenceOrder)
}

func TestCheckSequenceFields_TypeMismatch(t *testing.T) {
	// SEQUENCE field says DisplayString but column object uses Integer32.
	fields := []module.SequenceField{
		{Name: "col1", Syntax: &module.TypeSyntaxTypeRef{Name: "DisplayString"}},
		{Name: "col2", Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"}},
	}
	mod := buildTableModule(fields, []string{"col1", "col2"})

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagSequenceTypeMismatch)
}

// --- checkEnumSubtyping tests ---

func TestCheckEnumSubtyping_InvalidValue(t *testing.T) {
	// Subtype enum with a named value not in parent should emit subtype-enumeration-illegal.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		// Parent type with enum values
		&module.TypeDef{
			DefBase: module.DefBase{Name: "ParentEnum"},
			Syntax: &module.TypeSyntaxIntegerEnum{
				NamedNumbers: []module.NamedNumber{
					{Name: "up", Value: 1},
					{Name: "down", Value: 2},
				},
			},
			Status:              types.StatusCurrent,
			IsTextualConvention: true,
		},
		// Subtype referencing ParentEnum with invalid value "testing(3)"
		&module.ObjectType{
			DefBase: module.DefBase{Name: "badSubtype"},
			Syntax: &module.TypeSyntaxIntegerEnum{
				Base: "ParentEnum",
				NamedNumbers: []module.NamedNumber{
					{Name: "up", Value: 1},
					{Name: "testing", Value: 3},
				},
			},
			Access: types.AccessReadOnly,
			Status: types.StatusCurrent,
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagSubtypeEnumIllegal)
}

func TestCheckEnumSubtyping_ValidSubset(t *testing.T) {
	// Subtype enum with values all present in parent - no diagnostic.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "ParentEnum"},
			Syntax: &module.TypeSyntaxIntegerEnum{
				NamedNumbers: []module.NamedNumber{
					{Name: "up", Value: 1},
					{Name: "down", Value: 2},
					{Name: "testing", Value: 3},
				},
			},
			Status:              types.StatusCurrent,
			IsTextualConvention: true,
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "goodSubtype"},
			Syntax: &module.TypeSyntaxIntegerEnum{
				Base: "ParentEnum",
				NamedNumbers: []module.NamedNumber{
					{Name: "up", Value: 1},
					{Name: "down", Value: 2},
				},
			},
			Access: types.AccessReadOnly,
			Status: types.StatusCurrent,
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagSubtypeEnumIllegal)
}

// --- checkObsoleteImports tests ---

func TestCheckObsoleteImports_RFC1155InSMIv2(t *testing.T) {
	// An SMIv2 module importing from RFC1155-SMI should emit obsolete-import.
	mod := buildTestModule(testModuleConfig{
		language:    types.LanguageSMIv2,
		baseImport:  module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		includeRoot: true,
	})

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagObsoleteImport)
}

func TestCheckObsoleteImports_RFC1213DisplayStringInSMIv2(t *testing.T) {
	// SMIv2 module importing DisplayString from RFC1213-MIB should emit obsolete-import.
	mod := buildTestModule(testModuleConfig{
		language:   types.LanguageSMIv2,
		baseImport: module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		extraImports: []module.Import{
			module.NewImport("RFC1213-MIB", "DisplayString", types.Span{}),
		},
		includeRoot: true,
	})

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagObsoleteImport)
}

func TestCheckObsoleteImports_SMIv1ModuleSkipped(t *testing.T) {
	// An SMIv1 module importing from RFC1155-SMI should NOT emit obsolete-import.
	mod := buildTestModule(testModuleConfig{
		language:    types.LanguageSMIv1,
		baseImport:  module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		includeRoot: true,
	})

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagObsoleteImport)
}

func TestCheckObsoleteImports_RFC1065InSMIv2(t *testing.T) {
	// SMIv2 module importing from RFC1065-SMI should emit obsolete-import.
	mod := buildTestModule(testModuleConfig{
		language:    types.LanguageSMIv2,
		baseImport:  module.NewImport("RFC1065-SMI", "enterprises", types.Span{}),
		includeRoot: true,
	})

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagObsoleteImport)
}

// --- checkIntegerMisuse ---

func TestCheckIntegerMisuse_ObjectType(t *testing.T) {
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "badObj"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagIntegerInSMIv2)
}

func TestCheckIntegerMisuse_TypeDef(t *testing.T) {
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.TypeDef{
			DefBase: module.DefBase{Name: "BadType"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagIntegerInSMIv2)
}

func TestCheckIntegerMisuse_ConstrainedInteger(t *testing.T) {
	// INTEGER with range constraint should still trigger.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "rangeObj"},
			Syntax: &module.TypeSyntaxConstrained{
				Base: &module.TypeSyntaxTypeRef{Name: "INTEGER"},
			},
			Status: types.StatusCurrent,
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagIntegerInSMIv2)
}

func TestCheckIntegerMisuse_Integer32NoDiag(t *testing.T) {
	// Integer32 is the correct SMIv2 form; no diagnostic.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "goodObj"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagIntegerInSMIv2)
}

func TestCheckTypeUnreferenced(t *testing.T) {
	t.Run("emits for unused local type", func(t *testing.T) {
		mod := module.NewModule("TEST-MIB", types.Span{})
		mod.Definitions = []module.Definition{
			&module.TypeDef{
				DefBase: module.DefBase{Name: "UnusedType"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			},
		}

		ctx := newTestContextForModules(VerboseConfig(), mod)
		checkTypeUnreferenced(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagTypeUnreferenced)
	})

	t.Run("skips locally referenced types", func(t *testing.T) {
		mod := module.NewModule("TEST-MIB", types.Span{})
		mod.Definitions = []module.Definition{
			&module.TypeDef{
				DefBase: module.DefBase{Name: "ParentType"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			},
			&module.TypeDef{
				DefBase: module.DefBase{Name: "ChildType"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "ParentType"},
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testObject"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "ChildType"},
			},
		}

		ctx := newTestContextForModules(VerboseConfig(), mod)
		checkTypeUnreferenced(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagTypeUnreferenced)
	})

	t.Run("treats imported type as referenced in source module", func(t *testing.T) {
		source := module.NewModule("SOURCE-MIB", types.Span{})
		source.Definitions = []module.Definition{
			&module.TypeDef{
				DefBase: module.DefBase{Name: "ExportedType"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			},
		}

		user := module.NewModule("USER-MIB", types.Span{})
		user.Imports = []module.Import{
			module.NewImport("SOURCE-MIB", "ExportedType", types.Span{}),
		}
		user.Definitions = []module.Definition{
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testObject"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			},
		}

		ctx := newTestContextForModules(VerboseConfig(), source, user)
		checkTypeUnreferenced(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagTypeUnreferenced)
	})
}

func TestCheckIdentifierCaseMatch(t *testing.T) {
	t.Run("case-only collision emits diagnostic", func(t *testing.T) {
		mod := module.NewModule("TEST-MIB", types.Span{})
		mod.Definitions = []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testNode"},
			},
			&module.ObjectIdentity{
				DefBase: module.DefBase{Name: "TestNode"},
			},
		}

		ctx := newTestContextForModules(VerboseConfig(), mod)
		checkIdentifierCaseMatch(ctx)

		d := hasDiag(t, ctx.Diagnostics(), types.DiagIdentifierCaseMatch)
		testutil.Contains(t, d.Message, "TestNode", "diagnostic message")
		testutil.Contains(t, d.Message, "testNode", "diagnostic message")
	})

	t.Run("sequence row naming convention is skipped", func(t *testing.T) {
		mod := module.NewModule("TEST-MIB", types.Span{})
		mod.Definitions = []module.Definition{
			&module.TypeDef{
				DefBase: module.DefBase{Name: "TestEntry"},
				Syntax: &module.TypeSyntaxSequence{
					Fields: []module.SequenceField{{Name: "index", Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"}}},
				},
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "TestEntry"},
			},
		}

		ctx := newTestContextForModules(VerboseConfig(), mod)
		checkIdentifierCaseMatch(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagIdentifierCaseMatch)
	})
}

func TestCheckIntegerMisuse_IntegerEnumNoDiag(t *testing.T) {
	// INTEGER with named values (enum) should not trigger.
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "enumObj"},
			Syntax: &module.TypeSyntaxIntegerEnum{
				NamedNumbers: []module.NamedNumber{{Name: "up", Value: 1}, {Name: "down", Value: 2}},
			},
			Status: types.StatusCurrent,
			Oid:    testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagIntegerInSMIv2)
}

func TestCheckIntegerMisuse_SMIv1Skipped(t *testing.T) {
	// INTEGER in SMIv1 module should not trigger.
	mod := testSMIv1Module(nil,
		&module.ObjectType{
			DefBase: module.DefBase{Name: "v1Obj"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
			Status:  types.StatusMandatory,
			Oid:     testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagIntegerInSMIv2)
}

// --- checkTrapInSMIv2 ---

func TestCheckTrapInSMIv2_TrapTypeEmitsDiag(t *testing.T) {
	mod := testSMIv2Module(nil,
		&module.Notification{
			DefBase:  module.DefBase{Name: "badTrap"},
			TrapInfo: &module.TrapInfo{Enterprise: "testRoot", TrapNumber: 1},
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagTrapInSMIv2)
}

func TestCheckTrapInSMIv2_NotificationTypeNoDiag(t *testing.T) {
	// NOTIFICATION-TYPE (not a trap) should not trigger.
	notifOid := testOid("testRoot", 1)
	mod := testSMIv2Module(nil,
		&module.Notification{
			DefBase: module.DefBase{Name: "goodNotif"},
			Status:  types.StatusCurrent,
			Oid:     &notifOid,
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagTrapInSMIv2)
}

func TestCheckTrapInSMIv2_SMIv1TrapSkipped(t *testing.T) {
	// TRAP-TYPE in SMIv1 module should not trigger.
	mod := testSMIv1Module(nil,
		&module.Notification{
			DefBase:  module.DefBase{Name: "v1Trap"},
			TrapInfo: &module.TrapInfo{Enterprise: "testRoot", TrapNumber: 1},
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagTrapInSMIv2)
}

func TestCheckGroupUnreferenced(t *testing.T) {
	t.Run("unreferenced groups emit diagnostics", func(t *testing.T) {
		mod := module.NewModule("TEST-MIB", types.Span{})
		mod.Definitions = []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "unusedObjectGroup"},
			},
			&module.NotificationGroup{
				DefBase: module.DefBase{Name: "unusedNotificationGroup"},
			},
		}

		ctx := newTestContextForModules(VerboseConfig(), mod)
		checkGroupUnreferenced(ctx)

		requireDiagCount(t, ctx.Diagnostics(), types.DiagGroupUnreferenced, 2)
	})

	t.Run("compliance and capabilities references suppress diagnostics", func(t *testing.T) {
		mod := module.NewModule("TEST-MIB", types.Span{})
		mod.Definitions = []module.Definition{
			&module.ObjectGroup{
				DefBase: module.DefBase{Name: "usedObjectGroup"},
			},
			&module.NotificationGroup{
				DefBase: module.DefBase{Name: "usedNotificationGroup"},
			},
			&module.ModuleCompliance{
				DefBase: module.DefBase{Name: "testCompliance"},
				Modules: []module.ComplianceModule{
					{
						MandatoryGroups: []string{"usedObjectGroup"},
					},
				},
			},
			&module.AgentCapabilities{
				DefBase: module.DefBase{Name: "testCaps"},
				Supports: []module.SupportsModule{
					{
						ModuleName: "TEST-MIB",
						Includes:   []string{"usedNotificationGroup"},
					},
				},
			},
		}

		ctx := newTestContextForModules(VerboseConfig(), mod)
		checkGroupUnreferenced(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagGroupUnreferenced)
	})
}

// --- checkOpaqueSMIv2 ---

func TestCheckOpaqueSMIv2_EmitsDiag(t *testing.T) {
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Opaque", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "opaqueObj"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Opaque"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagOpaqueSMIv2)
}

func TestCheckOpaqueSMIv2_NonOpaqueNoDiag(t *testing.T) {
	mod := testSMIv2Module(
		[]module.Import{
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "normalObj"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			Status:  types.StatusCurrent,
			Oid:     testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagOpaqueSMIv2)
}

func TestCheckOpaqueSMIv2_SMIv1Skipped(t *testing.T) {
	// Opaque in SMIv1 module should not trigger.
	mod := testSMIv1Module(
		[]module.Import{
			module.NewImport("RFC1155-SMI", "Opaque", types.Span{}),
		},
		&module.ObjectType{
			DefBase: module.DefBase{Name: "v1Opaque"},
			Syntax:  &module.TypeSyntaxTypeRef{Name: "Opaque"},
			Status:  types.StatusMandatory,
			Oid:     testOid("testRoot", 1),
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagOpaqueSMIv2)
}

// --- checkNotificationReversibility ---

func TestCheckNotificationReversibility_NotReversible(t *testing.T) {
	// Notification whose parent arc is not 0 should emit not-reversible.
	// Place notification directly under testRoot (arc 99999, not 0).
	notifOid := testOid("testRoot", 1)
	mod := testSMIv2Module(nil,
		&module.Notification{
			DefBase: module.DefBase{Name: "badNotif"},
			Status:  types.StatusCurrent,
			Oid:     &notifOid,
		},
	)

	m := resolveStrict(mod)
	hasDiag(t, m.Diagnostics(), types.DiagNotifNotReversible)
}

func TestCheckNotificationReversibility_Reversible(t *testing.T) {
	// Proper structure: notification under a .0. node.
	// testRoot { enterprises 99999 } -> testNotifs { testRoot 0 } -> myNotif { testNotifs 1 }
	notifsOid := testOid("testRoot", 0)
	notifOid := testOid("testNotifs", 1)
	mod := testSMIv2Module(nil,
		&module.ValueAssignment{
			DefBase: module.DefBase{Name: "testNotifs"},
			Oid:     notifsOid,
		},
		&module.Notification{
			DefBase: module.DefBase{Name: "myNotif"},
			Status:  types.StatusCurrent,
			Oid:     &notifOid,
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagNotifNotReversible)
}

func TestCheckNotificationReversibility_SMIv1Skipped(t *testing.T) {
	// SMIv1 trap should not be checked for reversibility.
	mod := testSMIv1Module(nil,
		&module.Notification{
			DefBase:  module.DefBase{Name: "v1Trap"},
			TrapInfo: &module.TrapInfo{Enterprise: "testRoot", TrapNumber: 1},
		},
	)

	m := resolveStrict(mod)
	noDiag(t, m.Diagnostics(), types.DiagNotifNotReversible, types.DiagNotifIdTooLarge)
}
