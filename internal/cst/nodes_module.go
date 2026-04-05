package cst

import "github.com/golangsnmp/gomib/internal/types"

// ModuleFile is the root CST node for a parsed file.
type ModuleFile struct {
	Modules []ModuleNode
	EOF     SyntaxToken
	Span    types.Span
}

// ModuleNode is a single DEFINITIONS ::= BEGIN...END block.
type ModuleNode struct {
	Name        SyntaxToken      // module name
	Definitions SyntaxToken      // DEFINITIONS keyword
	Assign      SyntaxToken      // ::=
	Begin       SyntaxToken      // BEGIN keyword
	Imports     *ImportsNode     // nil if no IMPORTS section
	Body        []DefinitionNode // definition list
	End         SyntaxToken      // END keyword
	Span        types.Span
}

// ImportsNode is the IMPORTS...; section of a module.
type ImportsNode struct {
	Keyword SyntaxToken // IMPORTS keyword
	Groups  []ImportGroupNode
	Semi    SyntaxToken // closing semicolon
	Span    types.Span
}

// ImportGroupNode is one "symbol, symbol FROM ModuleName" group.
type ImportGroupNode struct {
	Symbols []SyntaxToken
	Commas  []SyntaxToken
	From    SyntaxToken // FROM keyword
	Module  SyntaxToken // module name
	Span    types.Span
}

// DefinitionNode is implemented by all definition CST node types.
type DefinitionNode interface {
	definitionNode()
	DefinitionSpan() types.Span
}
