// Tokenize a small MIB snippet and print each token's kind and text.
package main

import (
	"fmt"

	"github.com/golangsnmp/gomib/token"
)

func main() {
	source := []byte(`
EXAMPLE-MIB DEFINITIONS ::= BEGIN

IMPORTS
    MODULE-IDENTITY, Integer32
        FROM SNMPv2-SMI
    DisplayString
        FROM SNMPv2-TC;

exampleMIB MODULE-IDENTITY
    LAST-UPDATED "200601010000Z"
    ORGANIZATION "Example Inc."
    CONTACT-INFO "support@example.com"
    DESCRIPTION  "An example MIB for tokenization."
    ::= { enterprises 99999 }

exampleString OBJECT-TYPE
    SYNTAX      DisplayString (SIZE (0..255))
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION "An example string object."
    ::= { exampleMIB 1 }

END
`)

	tokens := token.Tokenize(source)

	// Print all tokens
	fmt.Println("=== All tokens ===")
	for _, tok := range tokens {
		if tok.Kind == token.EOF {
			break
		}
		text := source[tok.Span.Start:tok.Span.End]
		fmt.Printf("  %-22s %q\n", tok.Kind.LibsmiName(), text)
	}

	// Filter: only macro keywords (OBJECT-TYPE, MODULE-IDENTITY, etc.)
	fmt.Println("\n=== Macro keywords ===")
	for _, tok := range tokens {
		if tok.Kind.IsMacroKeyword() {
			fmt.Printf("  %s\n", source[tok.Span.Start:tok.Span.End])
		}
	}

	// Filter: only identifiers
	fmt.Println("\n=== Identifiers ===")
	for _, tok := range tokens {
		if tok.Kind.IsIdentifier() {
			fmt.Printf("  %-22s %s\n", tok.Kind.LibsmiName(), source[tok.Span.Start:tok.Span.End])
		}
	}
}
