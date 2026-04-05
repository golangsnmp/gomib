package parser_test

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/cst"
	"github.com/golangsnmp/gomib/internal/cst/lower"
	cstparser "github.com/golangsnmp/gomib/internal/cst/parser"
	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/internal/types"
)

func FuzzCSTParseModule(f *testing.F) {
	f.Add([]byte("TEST DEFINITIONS ::= BEGIN END"))
	f.Add([]byte(`TEST DEFINITIONS ::= BEGIN
IMPORTS MODULE-IDENTITY FROM SNMPv2-SMI;
END`))
	f.Add([]byte(`TEST DEFINITIONS ::= BEGIN
IMPORTS OBJECT-TYPE, Integer32 FROM SNMPv2-SMI;
testObj OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "test"
    ::= { testObj 1 }
END`))
	f.Add([]byte(`TEST DEFINITIONS ::= BEGIN
IMPORTS TEXTUAL-CONVENTION, DisplayString FROM SNMPv2-TC;
MyTC ::= TEXTUAL-CONVENTION
    STATUS current
    DESCRIPTION "test"
    SYNTAX DisplayString (SIZE (0..255))
END`))
	f.Add([]byte(`TEST DEFINITIONS ::= BEGIN
IMPORTS NOTIFICATION-TYPE FROM SNMPv2-SMI;
testNotif NOTIFICATION-TYPE
    STATUS current
    DESCRIPTION "test"
    ::= { testNotif 1 }
END`))
	f.Add([]byte(`TEST DEFINITIONS ::= BEGIN
testTrap TRAP-TYPE
    ENTERPRISE testEnterprise
    DESCRIPTION "test"
    ::= 1
END`))
	f.Add([]byte(`TEST DEFINITIONS ::= BEGIN
IMPORTS OBJECT-TYPE, Integer32 FROM SNMPv2-SMI;
testObj OBJECT-TYPE
    SYNTAX INTEGER { up(1), down(2) }
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "test"
    DEFVAL { up }
    ::= { testObj 1 }
END`))

	testutil.AddProblemCorpusSeeds(f)
	testutil.AddPrimaryCorpusSeeds(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}

		cfg := types.DefaultConfig()
		p := cstparser.New(data, nil, cfg)
		file := p.ParseModule()

		// Must not panic during lowering.
		_ = lower.Lower(file, data, p.Diagnostics(), cfg)

		// Round-trip check: if there are no parse errors, reconstructed
		// text must match original.
		hasError := false
		for _, d := range p.Diagnostics() {
			if d.Code == types.DiagParseError {
				hasError = true
				break
			}
		}
		if !hasError {
			got := cst.ReconstructText(file, data)
			if string(got) != string(data) {
				t.Errorf("round-trip mismatch")
			}
		}
	})
}

func FuzzCSTConstraintParsing(f *testing.F) {
	f.Add([]byte("(0..255)"))
	f.Add([]byte("(SIZE (0..255))"))
	f.Add([]byte("(SIZE (0..255 | 512..1024))"))
	f.Add([]byte("(0..100 | 200..300)"))
	f.Add([]byte("(SIZE (0..maxMessageSize))"))
	f.Add([]byte("(-128..127)"))
	f.Add([]byte("(0..4294967295)"))
	f.Add([]byte("(0..18446744073709551615)"))
	f.Add([]byte("(SIZE (0..65535))"))

	f.Add([]byte("0"))
	f.Add([]byte(`""`))
	f.Add([]byte("'0000'H"))
	f.Add([]byte("'00000000'B"))
	f.Add([]byte("up"))
	f.Add([]byte("{ up, down }"))
	f.Add([]byte("{ 1 3 6 1 }"))
	f.Add([]byte("{ sysName 0 }"))
	f.Add([]byte("true"))
	f.Add([]byte(`"default value"`))
	f.Add([]byte("65535"))
	f.Add([]byte("-1"))

	testutil.AddProblemCorpusSeeds(f)
	testutil.AddPrimaryCorpusSeeds(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<16 {
			return
		}

		cfg := types.DefaultConfig()

		// Inject into constraint position.
		constraintSrc := []byte(`TEST DEFINITIONS ::= BEGIN
IMPORTS OBJECT-TYPE, Integer32 FROM SNMPv2-SMI;
testObj OBJECT-TYPE
    SYNTAX Integer32 ` + string(data) + `
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "test"
    ::= { testObj 1 }
END`)
		p := cstparser.New(constraintSrc, nil, cfg)
		file := p.ParseModule()
		_ = lower.Lower(file, constraintSrc, p.Diagnostics(), cfg)

		// Inject into DEFVAL position.
		defvalSrc := []byte(`TEST DEFINITIONS ::= BEGIN
IMPORTS OBJECT-TYPE, Integer32 FROM SNMPv2-SMI;
testObj OBJECT-TYPE
    SYNTAX Integer32
    MAX-ACCESS read-only
    STATUS current
    DESCRIPTION "test"
    DEFVAL { ` + string(data) + ` }
    ::= { testObj 1 }
END`)
		p2 := cstparser.New(defvalSrc, nil, cfg)
		file2 := p2.ParseModule()
		_ = lower.Lower(file2, defvalSrc, p2.Diagnostics(), cfg)
	})
}
