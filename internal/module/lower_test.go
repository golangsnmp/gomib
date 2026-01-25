package module

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/ast"
	"github.com/golangsnmp/gomib/internal/types"
)

func TestLowerAccess(t *testing.T) {
	tests := []struct {
		input    ast.AccessValue
		expected types.Access
	}{
		{ast.AccessValueReadOnly, types.AccessReadOnly},
		{ast.AccessValueReadWrite, types.AccessReadWrite},
		{ast.AccessValueReadCreate, types.AccessReadCreate},
		{ast.AccessValueNotAccessible, types.AccessNotAccessible},
		{ast.AccessValueAccessibleForNotify, types.AccessAccessibleForNotify},
		{ast.AccessValueWriteOnly, types.AccessWriteOnly},
		// SMIv1 normalization
		{ast.AccessValueReportOnly, types.AccessReadOnly},
		{ast.AccessValueInstall, types.AccessReadWrite},
		{ast.AccessValueNotImplemented, types.AccessNotAccessible},
	}

	for _, tt := range tests {
		got := lowerAccess(tt.input)
		if got != tt.expected {
			t.Errorf("lowerAccess(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestLowerStatus(t *testing.T) {
	tests := []struct {
		input    ast.StatusValue
		expected types.Status
	}{
		{ast.StatusValueCurrent, types.StatusCurrent},
		{ast.StatusValueDeprecated, types.StatusDeprecated},
		{ast.StatusValueObsolete, types.StatusObsolete},
		// SMIv1 normalization
		{ast.StatusValueMandatory, types.StatusCurrent},
		{ast.StatusValueOptional, types.StatusDeprecated},
	}

	for _, tt := range tests {
		got := lowerStatus(tt.input)
		if got != tt.expected {
			t.Errorf("lowerStatus(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestLowerEmptyModule(t *testing.T) {
	astMod := &ast.Module{
		Name:            ast.Ident{Name: "TEST-MIB", Span: types.Span{Start: 0, End: 8}},
		DefinitionsKind: ast.DefinitionsKindDefinitions,
		Imports:         nil,
		Body:            nil,
		Span:            types.Span{Start: 0, End: 100},
	}

	mod := Lower(astMod, nil)

	if mod.Name != "TEST-MIB" {
		t.Errorf("got name %q, want %q", mod.Name, "TEST-MIB")
	}
	if mod.Language != SmiLanguageSMIv1 {
		t.Errorf("got language %v, want %v", mod.Language, SmiLanguageSMIv1)
	}
	if len(mod.Imports) != 0 {
		t.Errorf("got %d imports, want 0", len(mod.Imports))
	}
	if len(mod.Definitions) != 0 {
		t.Errorf("got %d definitions, want 0", len(mod.Definitions))
	}
}

func TestLowerSMIv2Detection(t *testing.T) {
	astMod := &ast.Module{
		Name:            ast.Ident{Name: "TEST-MIB", Span: types.Span{Start: 0, End: 8}},
		DefinitionsKind: ast.DefinitionsKindDefinitions,
		Imports: []ast.ImportClause{
			{
				Symbols: []ast.Ident{
					{Name: "MODULE-IDENTITY", Span: types.Span{Start: 20, End: 35}},
					{Name: "OBJECT-TYPE", Span: types.Span{Start: 37, End: 48}},
				},
				FromModule: ast.Ident{Name: "SNMPv2-SMI", Span: types.Span{Start: 54, End: 64}},
				Span:       types.Span{Start: 20, End: 64},
			},
		},
		Body: nil,
		Span: types.Span{Start: 0, End: 100},
	}

	mod := Lower(astMod, nil)

	if mod.Language != SmiLanguageSMIv2 {
		t.Errorf("got language %v, want %v", mod.Language, SmiLanguageSMIv2)
	}
	if len(mod.Imports) != 2 {
		t.Errorf("got %d imports, want 2", len(mod.Imports))
	}
	if mod.Imports[0].Module != "SNMPv2-SMI" {
		t.Errorf("got module %q, want %q", mod.Imports[0].Module, "SNMPv2-SMI")
	}
	if mod.Imports[0].Symbol != "MODULE-IDENTITY" {
		t.Errorf("got symbol %q, want %q", mod.Imports[0].Symbol, "MODULE-IDENTITY")
	}
}

func TestLowerValueAssignment(t *testing.T) {
	astMod := &ast.Module{
		Name:            ast.Ident{Name: "TEST-MIB", Span: types.Span{Start: 0, End: 8}},
		DefinitionsKind: ast.DefinitionsKindDefinitions,
		Body: []ast.Definition{
			&ast.ValueAssignmentDef{
				Name: ast.Ident{Name: "testOid", Span: types.Span{Start: 100, End: 107}},
				OidAssignment: ast.OidAssignment{
					Components: []ast.OidComponent{
						&ast.OidComponentName{Name: ast.Ident{Name: "enterprises", Span: types.Span{Start: 130, End: 141}}},
						&ast.OidComponentNumber{Value: 123, Span: types.Span{Start: 142, End: 145}},
					},
					Span: types.Span{Start: 128, End: 147},
				},
				Span: types.Span{Start: 100, End: 147},
			},
		},
		Span: types.Span{Start: 0, End: 200},
	}

	mod := Lower(astMod, nil)

	if len(mod.Definitions) != 1 {
		t.Fatalf("got %d definitions, want 1", len(mod.Definitions))
	}

	va, ok := mod.Definitions[0].(*ValueAssignment)
	if !ok {
		t.Fatalf("expected ValueAssignment, got %T", mod.Definitions[0])
	}

	if va.Name != "testOid" {
		t.Errorf("got name %q, want %q", va.Name, "testOid")
	}
	if len(va.Oid.Components) != 2 {
		t.Errorf("got %d OID components, want 2", len(va.Oid.Components))
	}
}

func TestLowerMacroDefinitionFiltered(t *testing.T) {
	astMod := &ast.Module{
		Name:            ast.Ident{Name: "SNMPv2-SMI", Span: types.Span{Start: 0, End: 10}},
		DefinitionsKind: ast.DefinitionsKindDefinitions,
		Body: []ast.Definition{
			&ast.MacroDefinitionDef{
				Name: ast.Ident{Name: "OBJECT-TYPE", Span: types.Span{Start: 100, End: 111}},
				Span: types.Span{Start: 100, End: 200},
			},
			&ast.ValueAssignmentDef{
				Name: ast.Ident{Name: "internet", Span: types.Span{Start: 300, End: 308}},
				OidAssignment: ast.OidAssignment{
					Components: []ast.OidComponent{
						&ast.OidComponentNumber{Value: 1, Span: types.Span{Start: 330, End: 331}},
						&ast.OidComponentNumber{Value: 3, Span: types.Span{Start: 332, End: 333}},
						&ast.OidComponentNumber{Value: 6, Span: types.Span{Start: 334, End: 335}},
						&ast.OidComponentNumber{Value: 1, Span: types.Span{Start: 336, End: 337}},
					},
					Span: types.Span{Start: 328, End: 339},
				},
				Span: types.Span{Start: 300, End: 339},
			},
		},
		Span: types.Span{Start: 0, End: 400},
	}

	mod := Lower(astMod, nil)

	// MACRO definitions should be filtered out
	if len(mod.Definitions) != 1 {
		t.Fatalf("got %d definitions, want 1 (MACRO should be filtered)", len(mod.Definitions))
	}

	va, ok := mod.Definitions[0].(*ValueAssignment)
	if !ok {
		t.Fatalf("expected ValueAssignment, got %T", mod.Definitions[0])
	}
	if va.Name != "internet" {
		t.Errorf("got name %q, want %q", va.Name, "internet")
	}
}

func TestLowerTypeSyntax(t *testing.T) {
	tests := []struct {
		name    string
		input   ast.TypeSyntax
		checkFn func(TypeSyntax) bool
	}{
		{
			name:  "TypeRef",
			input: &ast.TypeSyntaxTypeRef{Name: ast.Ident{Name: "Integer32"}},
			checkFn: func(ts TypeSyntax) bool {
				ref, ok := ts.(*TypeSyntaxTypeRef)
				return ok && ref.Name == "Integer32"
			},
		},
		{
			name:  "OctetString",
			input: &ast.TypeSyntaxOctetString{},
			checkFn: func(ts TypeSyntax) bool {
				_, ok := ts.(*TypeSyntaxOctetString)
				return ok
			},
		},
		{
			name:  "ObjectIdentifier",
			input: &ast.TypeSyntaxObjectIdentifier{},
			checkFn: func(ts TypeSyntax) bool {
				_, ok := ts.(*TypeSyntaxObjectIdentifier)
				return ok
			},
		},
		{
			name: "IntegerEnum",
			input: &ast.TypeSyntaxIntegerEnum{
				NamedNumbers: []ast.NamedNumber{
					{Name: ast.Ident{Name: "up"}, Value: 1},
					{Name: ast.Ident{Name: "down"}, Value: 2},
				},
			},
			checkFn: func(ts TypeSyntax) bool {
				ie, ok := ts.(*TypeSyntaxIntegerEnum)
				return ok && len(ie.NamedNumbers) == 2 &&
					ie.NamedNumbers[0].Name == "up" &&
					ie.NamedNumbers[1].Value == 2
			},
		},
		{
			name: "SequenceOf",
			input: &ast.TypeSyntaxSequenceOf{
				EntryType: ast.Ident{Name: "IfEntry"},
			},
			checkFn: func(ts TypeSyntax) bool {
				so, ok := ts.(*TypeSyntaxSequenceOf)
				return ok && so.EntryType == "IfEntry" && ts.IsSequenceOf()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lowerTypeSyntax(tt.input)
			if !tt.checkFn(result) {
				t.Errorf("lowerTypeSyntax failed for %s", tt.name)
			}
		})
	}
}
