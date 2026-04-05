package resolver

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/parser"
	"github.com/golangsnmp/gomib/internal/types"
)

// FuzzFormatOID fuzzes the FormatOID method against a resolved MIB tree.
// This catches panics in the suffix slicing and string formatting logic
// that only surface with specific OID prefix/suffix combinations.
func FuzzFormatOID(f *testing.F) {
	f.Add("1.3.6.1.2.1")
	f.Add("1.3.6.1.2.1.1.1.0")
	f.Add("1.3.6.1.4.1.99999.1.2.3")
	f.Add("0.0")
	f.Add("1")
	f.Add("")
	f.Add("1.3.6.1.2.1.999.999.999")

	// Build a small resolved MIB to test FormatOID against.
	src := []byte(`TEST-MIB DEFINITIONS ::= BEGIN
		IMPORTS MODULE-IDENTITY, OBJECT-TYPE, Integer32, enterprises FROM SNMPv2-SMI;
		testMIB MODULE-IDENTITY
			LAST-UPDATED "200601010000Z"
			ORGANIZATION ""
			CONTACT-INFO ""
			DESCRIPTION ""
			::= { enterprises 99999 }
		testObj OBJECT-TYPE
			SYNTAX Integer32
			MAX-ACCESS read-only
			STATUS current
			DESCRIPTION ""
			::= { testMIB 1 }
		END`)
	cfg := types.DefaultConfig()
	p := parser.New(src, nil, cfg)
	mods := p.ParseModule()
	mod := mods[0]
	module.ValidateModule(mod, src, nil, cfg)
	resolverCfg := model.DefaultConfig()
	mibTree := Resolve([]*module.Module{mod}, nil, nil, &resolverCfg)

	f.Fuzz(func(t *testing.T, s string) {
		oid, err := model.ParseOID(s)
		if err != nil {
			return
		}

		// FormatOID must not panic regardless of input.
		result := mibTree.FormatOID(oid)
		if result == "" && len(oid) > 0 {
			t.Fatal("FormatOID returned empty string for non-empty OID")
		}

		// LongestPrefixByOID must not panic.
		_ = mibTree.LongestPrefixByOID(oid)

		// NodeByOID must not panic.
		_ = mibTree.NodeByOID(oid)
	})
}
