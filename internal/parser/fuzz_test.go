package parser

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func FuzzParseModule(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("EXAMPLE-MIB DEFINITIONS ::= BEGIN\nEND"))
	f.Add([]byte("EXAMPLE-MIB DEFINITIONS ::= BEGIN\n\tIMPORTS\n\t\tMODULE-IDENTITY, OBJECT-TYPE FROM SNMPv2-SMI\n\t\tDisplayString FROM SNMPv2-TC;\n\tEND"))
	f.Add([]byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "A test object"
			::= { test 1 }
		END`))
	f.Add([]byte(`TEST-MIB DEFINITIONS ::= BEGIN
		TestTC ::= TEXTUAL-CONVENTION
			DISPLAY-HINT "255a"
			STATUS current
			DESCRIPTION ""
			SYNTAX OCTET STRING (SIZE (0..255))
		END`))
	f.Add([]byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testNotif NOTIFICATION-TYPE
			OBJECTS { testObj }
			STATUS current
			DESCRIPTION "test"
			::= { test 2 }
		END`))
	f.Add([]byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testTrap TRAP-TYPE
			ENTERPRISE testEnterprise
			VARIABLES { testObj }
			DESCRIPTION "trap"
			::= 1
		END`))
	f.Add([]byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testTable OBJECT-TYPE
			SYNTAX SEQUENCE OF TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "table"
			::= { test 3 }
		testEntry OBJECT-TYPE
			SYNTAX TestEntry
			MAX-ACCESS not-accessible
			STATUS current
			DESCRIPTION "row"
			INDEX { testIndex }
			::= { testTable 1 }
		TestEntry ::= SEQUENCE {
			testIndex Integer32
		}
		END`))
	f.Add([]byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testId MODULE-IDENTITY
			LAST-UPDATED "200601010000Z"
			ORGANIZATION "Test"
			CONTACT-INFO "test"
			DESCRIPTION "test"
			REVISION "200601010000Z"
				DESCRIPTION "Initial version"
			::= { enterprises 99999 }
		END`))
	f.Add([]byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX INTEGER { up(1), down(2), testing(3) }
			MAX-ACCESS read-write
			STATUS current
			DESCRIPTION "enumerated"
			DEFVAL { up }
			::= { test 1 }
		END`))
	f.Add([]byte(`TEST-MIB DEFINITIONS ::= BEGIN
		testObj OBJECT-TYPE
			SYNTAX OCTET STRING (SIZE (0..255))
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION "hex defval"
			DEFVAL { 'deadbeef'H }
			::= { test 1 }
		END`))

	testutil.AddProblemCorpusSeeds(f)
	testutil.AddPrimaryCorpusSeeds(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Skip pathologically large inputs to avoid excessive memory use.
		if len(data) > 1<<20 {
			return
		}

		p := New(data, nil, types.DefaultConfig())
		mods := p.ParseModule()

		if len(mods) == 0 {
			t.Fatal("ParseModule returned empty slice")
		}
		for _, mod := range mods {
			for _, d := range mod.Diagnostics {
				if d.Code == "" {
					t.Fatal("diagnostic has empty Code")
				}
			}

			// Validate module span.
			inputLen := types.ByteOffset(len(data))
			if mod.Span.Start > mod.Span.End {
				t.Fatalf("module span start %d > end %d", mod.Span.Start, mod.Span.End)
			}
			if mod.Span.End > inputLen {
				t.Fatalf("module span end %d > input length %d", mod.Span.End, inputLen)
			}

			// Validate definition spans are ordered and within input bounds.
			for def := range mod.AllDefinitions() {
				span := def.DefinitionSpan()
				if span.Start > span.End {
					t.Fatalf("definition span start %d > end %d", span.Start, span.End)
				}
				if span.End > inputLen {
					t.Fatalf("definition span end %d > input length %d", span.End, inputLen)
				}
				// Definition name should be non-empty for valid definitions.
				if name := def.DefinitionName(); name != "" {
					nameSpan := def.DefinitionSpan()
					if nameSpan.Start > nameSpan.End {
						t.Fatalf("definition name %q span start %d > end %d",
							name, nameSpan.Start, nameSpan.End)
					}
				}
			}

			// Validate import spans.
			for _, imp := range mod.Imports {
				if imp.Span.Start > imp.Span.End {
					t.Fatalf("import span start %d > end %d", imp.Span.Start, imp.Span.End)
				}
				if imp.Span.End > inputLen {
					t.Fatalf("import span end %d > input length %d", imp.Span.End, inputLen)
				}
			}

			// Validate diagnostics have codes (no span validation since
			// Diagnostics use line/column, not byte spans).
			for _, d := range mod.Diagnostics {
				if d.Code == "" {
					t.Fatal("diagnostic has empty Code")
				}
				if d.Line < 0 {
					t.Fatalf("diagnostic line %d is negative", d.Line)
				}
			}
		}
	})
}

// FuzzConstraintParsing fuzzes constraint and DEFVAL parsing by injecting
// fuzz data into known-valid module templates. This targets the constraint
// parser (SIZE, range) and DEFVAL parser more directly than FuzzParseModule,
// which must discover valid-looking structure from random bytes.
func FuzzConstraintParsing(f *testing.F) {
	// Constraint seeds.
	f.Add([]byte("(0..255)"), true)
	f.Add([]byte("(SIZE (0..255))"), true)
	f.Add([]byte("(SIZE (0..255 | 512..1024))"), true)
	f.Add([]byte("(0..2147483647)"), true)
	f.Add([]byte("(-128..127)"), true)
	f.Add([]byte("(SIZE (0..MAX))"), true)
	f.Add([]byte("(MIN..MAX)"), true)
	f.Add([]byte("(1..10 | 20..30 | 100..200)"), true)
	f.Add([]byte("(SIZE (0 | 4 | 8..16))"), true)
	f.Add([]byte("(0..4294967295)"), true)

	// DEFVAL seeds.
	f.Add([]byte("42"), false)
	f.Add([]byte("-1"), false)
	f.Add([]byte(`"public"`), false)
	f.Add([]byte("'FF00'H"), false)
	f.Add([]byte("'1010'B"), false)
	f.Add([]byte("enabled"), false)
	f.Add([]byte("{ flag1, flag2 }"), false)
	f.Add([]byte("{ sysName 0 }"), false)
	f.Add([]byte("{ 1 3 6 1 }"), false)
	f.Add([]byte("{ }"), false)
	f.Add([]byte("0"), false)
	f.Add([]byte("''H"), false)

	f.Fuzz(func(t *testing.T, data []byte, isConstraint bool) {
		if len(data) > 1<<16 {
			return
		}

		var src []byte
		if isConstraint {
			// Inject fuzz data as a constraint on a type assignment.
			src = make([]byte, 0, len(data)+128)
			src = append(src, "T DEFINITIONS ::= BEGIN\nX ::= INTEGER "...)
			src = append(src, data...)
			src = append(src, "\nEND"...)
		} else {
			// Inject fuzz data as a DEFVAL value.
			src = make([]byte, 0, len(data)+256)
			src = append(src, "T DEFINITIONS ::= BEGIN\nx OBJECT-TYPE\n\tSYNTAX INTEGER\n\tMAX-ACCESS read-only\n\tSTATUS current\n\tDESCRIPTION \"\"\n\tDEFVAL { "...)
			src = append(src, data...)
			src = append(src, " }\n\t::= { test 1 }\nEND"...)
		}

		p := New(src, nil, types.DefaultConfig())
		mods := p.ParseModule()
		if len(mods) == 0 {
			t.Fatal("ParseModule returned empty slice")
		}
		mod := mods[0]

		// All diagnostics must have codes.
		for _, d := range mod.Diagnostics {
			if d.Code == "" {
				t.Fatal("diagnostic has empty Code")
			}
		}

		// Validate all spans are within input bounds.
		inputLen := types.ByteOffset(len(src))
		if mod.Span.End > inputLen {
			t.Fatalf("module span end %d > input length %d", mod.Span.End, inputLen)
		}
		for def := range mod.AllDefinitions() {
			span := def.DefinitionSpan()
			if span.Start > span.End {
				t.Fatalf("definition span start %d > end %d", span.Start, span.End)
			}
			if span.End > inputLen {
				t.Fatalf("definition span end %d > input length %d", span.End, inputLen)
			}
		}
	})
}
