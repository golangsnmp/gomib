package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

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
		}
		addDefs(mod, []module.Definition{
			&module.ModuleIdentity{
				DefBase:     module.DefBase{Name: "testMIB"},
				Description: "test",
				Oid: module.NewOidAssignment([]module.OidComponent{
					{Name: "enterprises"},
					{Number: 99999, HasNumber: true},
					{Number: 1, HasNumber: true},
				}, types.Span{}),
			},
		})
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
		}
		addDefs(mod, []module.Definition{
			&module.ModuleIdentity{
				DefBase:     module.DefBase{Name: "testMIB"},
				Description: "test",
				Oid:         testOid("testEnterprise", 1),
			},
			&module.ValueAssignment{
				DefBase: module.DefBase{Name: "testEnterprise"},
				Oid:     testOid("enterprises", 99999),
			},
		})
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
		}
		addDefs(mod, []module.Definition{
			&module.ModuleIdentity{
				DefBase:     module.DefBase{Name: "testMIB"},
				Description: "test",
				Oid: module.NewOidAssignment([]module.OidComponent{
					{Name: "iso", Number: 1, HasNumber: true},
					{Name: "org", Number: 3, HasNumber: true},
					{Name: "dod", Number: 6, HasNumber: true},
					{Name: "internet", Number: 1, HasNumber: true},
					{Name: "mgmt", Number: 2, HasNumber: true},
					{Number: 99, HasNumber: true},
				}, types.Span{}),
			},
		})
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
		}
		addDefs(mod, []module.Definition{
			&module.ModuleIdentity{
				DefBase:     module.DefBase{Name: "testMIB"},
				Description: "test",
				Oid:         testOid("mib-2", 999),
			},
		})
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
		}
		addDefs(mod, []module.Definition{
			&module.ModuleIdentity{
				DefBase:     module.DefBase{Name: "testMIB"},
				Description: "test",
				Oid:         testOid("enterprises", 12345),
			},
		})
		m := resolveStrict(mod)
		noDiag(t, m.Diagnostics(), types.DiagModuleIdentityReg)
	})
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
		mod.TypeDefs = []*module.TypeDef{
			{
				DefBase: module.DefBase{Name: "UnusedType"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			},
		}

		ctx := newTestContextForModules(model.VerboseConfig(), mod)
		checkTypeUnreferenced(ctx)

		hasDiag(t, ctx.Diagnostics(), types.DiagTypeUnreferenced)
	})

	t.Run("skips locally referenced types", func(t *testing.T) {
		mod := module.NewModule("TEST-MIB", types.Span{})
		mod.TypeDefs = []*module.TypeDef{
			{
				DefBase: module.DefBase{Name: "ParentType"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			},
			{
				DefBase: module.DefBase{Name: "ChildType"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "ParentType"},
			},
		}
		mod.ObjectTypes = []*module.ObjectType{
			{
				DefBase: module.DefBase{Name: "testObject"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "ChildType"},
			},
		}

		ctx := newTestContextForModules(model.VerboseConfig(), mod)
		checkTypeUnreferenced(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagTypeUnreferenced)
	})

	t.Run("treats imported type as referenced in source module", func(t *testing.T) {
		source := module.NewModule("SOURCE-MIB", types.Span{})
		source.TypeDefs = []*module.TypeDef{
			{
				DefBase: module.DefBase{Name: "ExportedType"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			},
		}

		user := module.NewModule("USER-MIB", types.Span{})
		user.Imports = []module.Import{
			module.NewImport("SOURCE-MIB", "ExportedType", types.Span{}),
		}
		user.ObjectTypes = []*module.ObjectType{
			{
				DefBase: module.DefBase{Name: "testObject"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "Integer32"},
			},
		}

		ctx := newTestContextForModules(model.VerboseConfig(), source, user)
		checkTypeUnreferenced(ctx)

		noDiag(t, ctx.Diagnostics(), types.DiagTypeUnreferenced)
	})
}

func TestCheckIdentifierCaseMatch(t *testing.T) {
	t.Run("case-only collision emits diagnostic", func(t *testing.T) {
		mod := module.NewModule("TEST-MIB", types.Span{})
		mod.ValueAssignments = []*module.ValueAssignment{
			{DefBase: module.DefBase{Name: "testNode"}},
		}
		mod.ObjectIdentities = []*module.ObjectIdentity{
			{DefBase: module.DefBase{Name: "TestNode"}},
		}

		ctx := newTestContextForModules(model.VerboseConfig(), mod)
		checkIdentifierCaseMatch(ctx)

		d := hasDiag(t, ctx.Diagnostics(), types.DiagIdentifierCaseMatch)
		testutil.Contains(t, d.Message, "TestNode", "diagnostic message")
		testutil.Contains(t, d.Message, "testNode", "diagnostic message")
	})

	t.Run("sequence row naming convention is skipped", func(t *testing.T) {
		mod := module.NewModule("TEST-MIB", types.Span{})
		mod.TypeDefs = []*module.TypeDef{
			{
				DefBase: module.DefBase{Name: "TestEntry"},
				Syntax: &module.TypeSyntaxSequence{
					Fields: []module.SequenceField{{Name: "index", Syntax: &module.TypeSyntaxTypeRef{Name: "Integer32"}}},
				},
			},
		}
		mod.ObjectTypes = []*module.ObjectType{
			{
				DefBase: module.DefBase{Name: "testEntry"},
				Syntax:  &module.TypeSyntaxTypeRef{Name: "TestEntry"},
			},
		}

		ctx := newTestContextForModules(model.VerboseConfig(), mod)
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
