package mib

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

// testOid builds an OID assignment as { parentName subid }.
func testOid(parentName string, subid uint32) module.OidAssignment {
	return module.NewOidAssignment([]module.OidComponent{
		&module.OidComponentName{NameValue: parentName},
		&module.OidComponentNumber{Value: subid},
	}, types.Span{})
}

func TestCheckNodeParentKinds_RowUnderTable(t *testing.T) {
	// Valid: row's parent is a table. Should produce no diagnostic.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagParentRow {
			t.Fatalf("unexpected %s diagnostic: %s", types.DiagParentRow, d.Message)
		}
	}
}

func TestCheckNodeParentKinds_RowUnderScalar(t *testing.T) {
	// Invalid: row's parent is a scalar, not a table.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagParentRow {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s diagnostic for row under scalar", types.DiagParentRow)
}

func TestCheckNodeParentKinds_TableUnderTable(t *testing.T) {
	// Invalid: table's parent is another table.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagParentTable {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s diagnostic for table under table", types.DiagParentTable)
}

func TestCheckNodeParentKinds_ScalarUnderRow(t *testing.T) {
	// A scalar child of a row is reclassified as a column by inferNodeKinds.
	// So this should NOT trigger parent-scalar.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagParentScalar || d.Code == types.DiagParentColumn {
			t.Fatalf("unexpected parent diagnostic: %s: %s", d.Code, d.Message)
		}
	}
}

func TestCheckNodeParentKinds_NotificationUnderTable(t *testing.T) {
	// Invalid: notification's parent is a table.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagParentNotification {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s diagnostic for notification under table", types.DiagParentNotification)
}

func TestCheckNodeParentKinds_GroupUnderTable(t *testing.T) {
	// Invalid: group's parent is a table.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagParentGroup {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s diagnostic for group under table", types.DiagParentGroup)
}

func TestCheckNodeParentKinds_RowSubidOne(t *testing.T) {
	// Valid: row at sub-identifier 1 under its table. No diagnostic expected.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagRowSubidentifierOne {
			t.Fatalf("unexpected %s diagnostic: %s", types.DiagRowSubidentifierOne, d.Message)
		}
	}
}

func TestCheckNodeParentKinds_RowSubidNotOne(t *testing.T) {
	// Invalid: row at sub-identifier 2 under its table.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagRowSubidentifierOne {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s diagnostic for row with sub-id != 1", types.DiagRowSubidentifierOne)
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

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagIndexElementNoSize {
				testutil.Contains(t, d.Message, "strIndex", "diagnostic message")
				found = true
			}
		}
		testutil.True(t, found, "expected index-element-no-size diagnostic")
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

		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagIndexElementNoSize {
				t.Fatalf("unexpected index-element-no-size diagnostic: %s", d.Message)
			}
		}
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

	var found bool
	for _, d := range ctx.Diagnostics() {
		if d.Code == types.DiagIndexIllegalBasetype {
			testutil.Contains(t, d.Message, "badIndex", "diagnostic message")
			testutil.Contains(t, d.Message, "Counter64", "diagnostic should mention type")
			found = true
		}
	}
	testutil.True(t, found, "expected index-illegal-basetype diagnostic")
}

func TestCheckIndexOIDLength(t *testing.T) {
	t.Run("exceeds 128 sub-identifiers", func(t *testing.T) {
		ctx := newTestContextWithConfig(StrictConfig())
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

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagIndexExceedsTooLarge {
				testutil.Contains(t, d.Message, "testEntry", "diagnostic message")
				found = true
			}
		}
		testutil.True(t, found, "expected index-exceeds-too-large diagnostic")
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

		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagIndexExceedsTooLarge {
				t.Fatalf("unexpected index-exceeds-too-large diagnostic: %s", d.Message)
			}
		}
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

func TestCheckAccessInvalidSMIv1(t *testing.T) {
	// accessible-for-notify is SMIv2-only.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv1,
		Imports: []module.Import{
			module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "badAccess"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Access:  types.AccessAccessibleForNotify,
				Status:  types.StatusMandatory,
				Oid:     testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagAccessInvalidSMIv1 {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for accessible-for-notify in SMIv1", types.DiagAccessInvalidSMIv1)
}

func TestCheckAccessInvalidSMIv1_ReadCreate(t *testing.T) {
	// read-create is SMIv2-only.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv1,
		Imports: []module.Import{
			module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "badAccess"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Access:  types.AccessReadCreate,
				Status:  types.StatusMandatory,
				Oid:     testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagAccessInvalidSMIv1 {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for read-create in SMIv1", types.DiagAccessInvalidSMIv1)
}

func TestCheckAccessWriteOnlySMIv2(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "badAccess"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Access:        types.AccessWriteOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagAccessWriteOnlySMIv2 {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for write-only in SMIv2", types.DiagAccessWriteOnlySMIv2)
}

func TestCheckAccessValid_NoFalsePositive(t *testing.T) {
	// read-write in SMIv1 should not produce any access diagnostic.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv1,
		Imports: []module.Import{
			module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "validObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Access:  types.AccessReadWrite,
				Status:  types.StatusMandatory,
				Oid:     testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagAccessInvalidSMIv1 || d.Code == types.DiagAccessWriteOnlySMIv2 {
			t.Fatalf("unexpected access diagnostic: %s: %s", d.Code, d.Message)
		}
	}
}

func TestCheckMaxAccessInSMIv1(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv1,
		Imports: []module.Import{
			module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "badKeyword"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusMandatory,
				Oid:           testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagMaxAccessInSMIv1 {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for MAX-ACCESS in SMIv1", types.DiagMaxAccessInSMIv1)
}

func TestCheckAccessInSMIv2(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "badKeyword"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagAccessInSMIv2 {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for ACCESS in SMIv2", types.DiagAccessInSMIv2)
}

func TestCheckAccessTableIllegal(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "badTable"},
				Syntax:        &module.TypeSyntaxSequenceOf{EntryType: "TestEntry"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagAccessTableIllegal {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for table with read-only access", types.DiagAccessTableIllegal)
}

func TestCheckAccessRowIllegal(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagAccessRowIllegal {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for row with read-only access", types.DiagAccessRowIllegal)
}

func TestCheckScalarNotCreatable(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "badScalar"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Access:        types.AccessReadCreate,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagScalarNotCreatable {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for scalar with read-create", types.DiagScalarNotCreatable)
}

func TestCheckAccessCounterIllegal(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Counter32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "badCounter"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "Counter32"},
				Access:        types.AccessReadWrite,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagAccessCounterIllegal {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for counter with read-write", types.DiagAccessCounterIllegal)
}

func TestCheckAccessCounterReadOnly_NoFalsePositive(t *testing.T) {
	// Counter32 with read-only should not trigger.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Counter32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "goodCounter"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "Counter32"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagAccessCounterIllegal {
			t.Fatalf("unexpected %s diagnostic for read-only counter", types.DiagAccessCounterIllegal)
		}
	}
}

func TestCheckStatusInvalidSMIv1(t *testing.T) {
	// "current" is SMIv2-only.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv1,
		Imports: []module.Import{
			module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "badStatus"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Access:  types.AccessReadOnly,
				Status:  types.StatusCurrent,
				Oid:     testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagStatusInvalidSMIv1 {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for current status in SMIv1", types.DiagStatusInvalidSMIv1)
}

func TestCheckStatusInvalidSMIv2(t *testing.T) {
	// "mandatory" is SMIv1-only.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "badStatus"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusMandatory,
				Oid:           testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagStatusInvalidSMIv2 {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for mandatory status in SMIv2", types.DiagStatusInvalidSMIv2)
}

func TestCheckStatusValid_NoFalsePositive(t *testing.T) {
	// "mandatory" in SMIv1 should not produce status diagnostic.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv1,
		Imports: []module.Import{
			module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "validObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Access:  types.AccessReadOnly,
				Status:  types.StatusMandatory,
				Oid:     testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagStatusInvalidSMIv1 || d.Code == types.DiagStatusInvalidSMIv2 {
			t.Fatalf("unexpected status diagnostic: %s: %s", d.Code, d.Message)
		}
	}
}

func TestCheckTypeStatusDeprecated(t *testing.T) {
	// Define a deprecated TC, then use it in an object.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagTypeStatusDeprecated {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for object using deprecated type", types.DiagTypeStatusDeprecated)
}

func TestCheckTypeStatusObsolete(t *testing.T) {
	// Define an obsolete TC, then use it in an object.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagTypeStatusObsolete {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s for object using obsolete type", types.DiagTypeStatusObsolete)
}

func TestCheckTypeStatusCurrent_NoFalsePositive(t *testing.T) {
	// A current type should not trigger type-status diagnostics.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase:       module.DefBase{Name: "validObj"},
				Syntax:        &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Access:        types.AccessReadOnly,
				AccessKeyword: types.AccessKeywordMaxAccess,
				Status:        types.StatusCurrent,
				Oid:           testOid("testRoot", 1),
			},
		},
	}
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagTypeStatusDeprecated || d.Code == types.DiagTypeStatusObsolete {
			t.Fatalf("unexpected type status diagnostic: %s: %s", d.Code, d.Message)
		}
	}
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
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagGroupMembership {
			t.Fatalf("unexpected group-membership diagnostic: %s", d.Message)
		}
	}
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
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagGroupMembership {
			t.Fatalf("unexpected group-membership diagnostic: %s", d.Message)
		}
	}
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
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagGroupMembership && d.Module == "TEST-MIB" {
			if d.Message == `"testScalar2" is not in any OBJECT-GROUP` {
				found = true
			}
		}
	}
	testutil.True(t, found, "expected group-membership diagnostic for testScalar2")
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
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagGroupMembership {
			t.Fatalf("unexpected group-membership diagnostic: %s", d.Message)
		}
	}
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
	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagGroupMembership && d.Module == "TEST-MIB" {
			if d.Message == `"testNotif2" is not in any NOTIFICATION-GROUP` {
				found = true
			}
		}
	}
	testutil.True(t, found, "expected group-membership diagnostic for testNotif2")
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

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

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagComplianceGroupStatus {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagComplianceGroupStatus diagnostic")
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

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

		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagComplianceGroupStatus {
				t.Fatal("unexpected DiagComplianceGroupStatus diagnostic")
			}
		}
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

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

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagComplianceGroupStatus {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagComplianceGroupStatus for optional group")
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

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

		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagComplianceGroupStatus {
				t.Fatal("unexpected DiagComplianceGroupStatus for SMIv1 compliance")
			}
		}
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

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

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagComplianceObjectStatus {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagComplianceObjectStatus diagnostic")
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

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

		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagComplianceObjectStatus {
				t.Fatal("unexpected DiagComplianceObjectStatus diagnostic")
			}
		}
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

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

		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagComplianceObjectStatus {
				t.Fatal("unexpected DiagComplianceObjectStatus for matching status")
			}
		}
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

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

		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagComplianceObjectStatus {
				t.Fatal("unexpected DiagComplianceObjectStatus for SMIv1 object status")
			}
		}
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

		root := ctx.mib.Root()
		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagComplianceGroupInvalid {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagComplianceGroupInvalid diagnostic")
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

		root := ctx.mib.Root()
		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagRefinementExists {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagRefinementExists diagnostic")
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

		root := ctx.mib.Root()
		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagOptionalGroupExists {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagOptionalGroupExists diagnostic")
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

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

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagRefinementNotListed {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagRefinementNotListed diagnostic")
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

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

		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagRefinementNotListed {
				t.Fatal("unexpected DiagRefinementNotListed diagnostic")
			}
		}
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

		root := ctx.mib.Root()
		compNode := buildOIDPath(root, 1, 10)
		compNode.setName("testCompliance")
		ctx.registerModuleNodeSymbol(mod, "testCompliance", compNode)

		createResolvedCompliances(ctx)
		checkComplianceStructure(ctx)

		for _, d := range ctx.Diagnostics() {
			switch d.Code {
			case types.DiagComplianceGroupInvalid, types.DiagRefinementExists, types.DiagOptionalGroupExists:
				t.Fatalf("unexpected diagnostic: %s", d.Code)
			}
		}
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

		ctx := newResolverContext([]*module.Module{modA, modB}, nil, StrictConfig())
		ctx.moduleIndex[modA.Name] = []*module.Module{modA}
		ctx.moduleIndex[modB.Name] = []*module.Module{modB}
		resolvedModB := newModule(modB.Name)
		ctx.moduleToResolved[modB] = resolvedModB
		ctx.resolvedToModule[resolvedModB] = modB

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

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagComplianceMemberNotLocal {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagComplianceMemberNotLocal diagnostic")
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

		ctx := newResolverContext([]*module.Module{mod}, nil, StrictConfig())
		ctx.moduleIndex[mod.Name] = []*module.Module{mod}
		resolvedMod := newModule(mod.Name)
		ctx.moduleToResolved[mod] = resolvedMod
		ctx.resolvedToModule[resolvedMod] = mod

		root := ctx.mib.Root()

		grpNode := buildOIDPath(root, 1, 1)
		grpNode.setName("testGroup")
		ctx.registerModuleNodeSymbol(mod, "testGroup", grpNode)

		objNode := buildOIDPath(root, 1, 2)
		objNode.setName("localObj")
		objNode.setKind(types.KindScalar)
		ctx.registerModuleNodeSymbol(mod, "localObj", objNode)

		checkGroupMemberLocality(ctx)

		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagComplianceMemberNotLocal {
				t.Fatal("unexpected DiagComplianceMemberNotLocal diagnostic")
			}
		}
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

		ctx := newResolverContext([]*module.Module{modA, modB}, nil, StrictConfig())
		ctx.moduleIndex[modA.Name] = []*module.Module{modA}
		ctx.moduleIndex[modB.Name] = []*module.Module{modB}
		resolvedModB := newModule(modB.Name)
		ctx.moduleToResolved[modB] = resolvedModB
		ctx.resolvedToModule[resolvedModB] = modB

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

		var found bool
		for _, d := range ctx.Diagnostics() {
			if d.Code == types.DiagComplianceMemberNotLocal {
				found = true
				break
			}
		}
		testutil.True(t, found, "expected DiagComplianceMemberNotLocal for notification group")
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
	// DisplayString is imported but never referenced in any definition.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
			module.NewImport("SNMPv2-TC", "DisplayString", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:  types.StatusCurrent,
				Oid:     testOid("testRoot", 1),
			},
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)

	found := false
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagImportUnused {
			found = true
			testutil.True(t, d.Message != "", "expected non-empty message")
		}
	}
	testutil.True(t, found, "expected %s diagnostic for unused DisplayString import", types.DiagImportUnused)
}

func TestCheckUnusedImports_AllUsed(t *testing.T) {
	// All imports are used; no unused import diagnostic expected.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
			module.NewImport("SNMPv2-SMI", "OBJECT-TYPE", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:  types.StatusCurrent,
				Oid:     testOid("testRoot", 1),
			},
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)

	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagImportUnused {
			t.Fatalf("unexpected %s diagnostic: %s", types.DiagImportUnused, d.Message)
		}
	}
}

func TestCheckBasetypeNotImported(t *testing.T) {
	// Integer32 used without import from SNMPv2-SMI.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:  types.StatusCurrent,
				Oid:     testOid("testRoot", 1),
			},
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)

	found := false
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagBasetypeNotImported {
			found = true
			testutil.True(t, d.Message != "", "expected non-empty message")
		}
	}
	testutil.True(t, found, "expected %s diagnostic for Integer32", types.DiagBasetypeNotImported)
}

func TestCheckBasetypeNotImported_WhenImported(t *testing.T) {
	// Integer32 is properly imported; no diagnostic expected.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase: module.DefBase{Name: "testObj"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:  types.StatusCurrent,
				Oid:     testOid("testRoot", 1),
			},
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)

	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagBasetypeNotImported {
			t.Fatalf("unexpected %s diagnostic: %s", types.DiagBasetypeNotImported, d.Message)
		}
	}
}

// --- description-missing ---

func TestCheckDescriptionMissing_ObjectType(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase:     module.DefBase{Name: "noDesc"},
				Syntax:      &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:      types.StatusCurrent,
				Description: "",
				Oid:         testOid("testRoot", 1),
			},
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagDescriptionMissing {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s diagnostic for OBJECT-TYPE without DESCRIPTION", types.DiagDescriptionMissing)
}

func TestCheckDescriptionMissing_SMIv1_NoDiag(t *testing.T) {
	// SMIv1 modules should not trigger description-missing.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv1,
		Imports: []module.Import{
			module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
			module.NewImport("RFC1155-SMI", "INTEGER", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase:     module.DefBase{Name: "noDesc"},
				Syntax:      &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Status:      types.StatusMandatory,
				Description: "",
				Oid:         testOid("testRoot", 1),
			},
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagDescriptionMissing {
			t.Fatalf("unexpected %s diagnostic for SMIv1 module: %s", types.DiagDescriptionMissing, d.Message)
		}
	}
}

func TestCheckDescriptionMissing_WithDescription_NoDiag(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.ObjectType{
				DefBase:        module.DefBase{Name: "hasDesc"},
				Syntax:         &module.TypeSyntaxTypeRef{Name: "Integer32"},
				Status:         types.StatusCurrent,
				Description:    "This object has a description",
				HasDescription: true,
				Oid:            testOid("testRoot", 1),
			},
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagDescriptionMissing {
			t.Fatalf("unexpected %s diagnostic: %s", types.DiagDescriptionMissing, d.Message)
		}
	}
}

// --- textual-convention-nested ---

func TestCheckTCNested(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagTCNested {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s diagnostic for TC derived from TC", types.DiagTCNested)
}

func TestCheckTCNested_FromBase_NoDiag(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "MyString"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "OCTET STRING"},
				Status:              types.StatusCurrent,
				Description:         "A TC from base type",
				IsTextualConvention: true,
			},
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagTCNested {
			t.Fatalf("unexpected %s diagnostic: %s", types.DiagTCNested, d.Message)
		}
	}
}

// --- type-assignment-smiv2 ---

func TestCheckTypeAssignmentSMIv2(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "MyType"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "INTEGER"},
				Status:              types.StatusCurrent,
				IsTextualConvention: false,
			},
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagTypeAssignmentSMIv2 {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s diagnostic for type assignment in SMIv2", types.DiagTypeAssignmentSMIv2)
}

func TestCheckTypeAssignmentSMIv2_TC_NoDiag(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-TC", "TEXTUAL-CONVENTION", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
			&module.TypeDef{
				DefBase:             module.DefBase{Name: "MyTC"},
				Syntax:              &module.TypeSyntaxTypeRef{Name: "OCTET STRING"},
				Status:              types.StatusCurrent,
				Description:         "A proper TC",
				IsTextualConvention: true,
			},
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagTypeAssignmentSMIv2 {
			t.Fatalf("unexpected %s diagnostic for TC: %s", types.DiagTypeAssignmentSMIv2, d.Message)
		}
	}
}

// --- table/row naming ---

func TestCheckTableRowNaming_BadTableName(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var foundTable, foundRow bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagTableNameTable {
			foundTable = true
		}
		if d.Code == types.DiagRowNameEntry {
			foundRow = true
		}
	}
	testutil.True(t, foundTable, "expected %s diagnostic for table not ending in Table", types.DiagTableNameTable)
	testutil.True(t, foundRow, "expected %s diagnostic for row not ending in Entry", types.DiagRowNameEntry)
}

func TestCheckTableRowNaming_Good_NoDiag(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagTableNameTable || d.Code == types.DiagRowNameEntry || d.Code == types.DiagRowNameTableName {
			t.Fatalf("unexpected naming diagnostic: %s: %s", d.Code, d.Message)
		}
	}
}

func TestCheckTableRowNaming_PrefixMismatch(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
			module.NewImport("SNMPv2-SMI", "Integer32", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagRowNameTableName {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s diagnostic for prefix mismatch", types.DiagRowNameTableName)
}

// --- named-numbers-ascending ---

func TestCheckNamedNumbersAscending(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagNamedNumbersAscending {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s diagnostic for non-ascending named numbers", types.DiagNamedNumbersAscending)
}

func TestCheckNamedNumbersAscending_OK_NoDiag(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagNamedNumbersAscending {
			t.Fatalf("unexpected %s diagnostic: %s", types.DiagNamedNumbersAscending, d.Message)
		}
	}
}

// --- hyphen-in-label ---

func TestCheckHyphenInLabel(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	var found bool
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagHyphenInLabel {
			found = true
			break
		}
	}
	testutil.True(t, found, "expected %s diagnostic for hyphenated named number", types.DiagHyphenInLabel)
}

func TestCheckHyphenInLabel_SMIv1_NoDiag(t *testing.T) {
	// Hyphenated labels are allowed in SMIv1.
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv1,
		Imports: []module.Import{
			module.NewImport("RFC1155-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagHyphenInLabel {
			t.Fatalf("unexpected %s diagnostic for SMIv1 module: %s", types.DiagHyphenInLabel, d.Message)
		}
	}
}

func TestCheckHyphenInLabel_NoHyphen_NoDiag(t *testing.T) {
	mod := &module.Module{
		Name:     "TEST-MIB",
		Language: types.LanguageSMIv2,
		Imports: []module.Import{
			module.NewImport("SNMPv2-SMI", "enterprises", types.Span{}),
		},
		Definitions: []module.Definition{
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testRoot"},
				Oid:     testOid("enterprises", 99999),
			},
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
		},
	}

	cfg := StrictConfig()
	m := Resolve([]*module.Module{mod}, nil, &cfg)
	for _, d := range m.Diagnostics() {
		if d.Code == types.DiagHyphenInLabel {
			t.Fatalf("unexpected %s diagnostic: %s", types.DiagHyphenInLabel, d.Message)
		}
	}
}
