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
	populateTypedSlices(mod, []module.Definition{
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
	})
	return mod
}

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

	mod := &module.Module{Name: "TEST-MIB", Language: types.LanguageSMIv2, Imports: imports}
	populateTypedSlices(mod, defs)
	return mod
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
