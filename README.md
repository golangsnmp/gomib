# gomib

[![CI](https://github.com/golangsnmp/gomib/actions/workflows/ci.yml/badge.svg)](https://github.com/golangsnmp/gomib/actions/workflows/ci.yml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/golangsnmp/gomib)](https://pkg.go.dev/github.com/golangsnmp/gomib)
[![Go Report Card](https://goreportcard.com/badge/github.com/golangsnmp/gomib)](https://goreportcard.com/report/github.com/golangsnmp/gomib)

Go library for parsing and querying SNMP MIB files.

Supports SMIv1 and SMIv2 modules. Loads MIBs from directories, directory trees, or embedded filesystems. Resolves imports, builds the OID tree, and provides typed access to objects, types, notifications, and conformance definitions.

## Install

```
go get github.com/golangsnmp/gomib
```

Requires Go 1.24+.

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/golangsnmp/gomib"
)

func main() {
    m, err := gomib.Load(context.Background(),
        gomib.WithSystemPaths(),
        gomib.WithModules("IF-MIB"),
    )
    if err != nil {
        log.Fatal(err)
    }

    obj := m.Object("ifIndex")
    if obj != nil {
        fmt.Printf("%s  %s  %s  %s\n", obj.Name(), obj.OID(), obj.Type().Name(), obj.Access())
        // ifIndex  1.3.6.1.2.1.2.2.1.1  InterfaceIndex  read-only
    }
}
```

## Loading MIBs

```go
// Load everything from a source (parses all files)
m, err := gomib.Load(ctx, gomib.WithSource(src))

// Load only specific modules and their transitive dependencies.
// Faster when the source contains hundreds of MIBs but you only need a few.
m, err := gomib.Load(ctx, gomib.WithSource(src), gomib.WithModules("IF-MIB", "IP-MIB"))
```

### Sources

`Dir` recursively indexes a directory tree using content-derived module names. `File`/`Files` load individual files by path. `FS` wraps an `fs.FS` (useful with `embed.FS`). `Multi` tries multiple sources in order.

```go
// Directory tree (indexed once at construction)
src, err := gomib.Dir("/usr/share/snmp/mibs")

// Individual files (module names auto-detected from content)
src, err := gomib.File("/path/to/IF-MIB.mib")
src, err  = gomib.Files("/path/to/IF-MIB.mib", "/path/to/IP-MIB.mib")

// Embedded filesystem
//go:embed mibs
var mibFS embed.FS
src := gomib.FS("embedded", mibFS)

// Combine sources (first match wins)
src := gomib.Multi(systemSrc, vendorSrc)
```

`MustDir` panics on error for use in `var` blocks.

Files are matched by extension: no extension, `.mib`, `.smi`, `.txt`, `.my`. Override with `WithExtensions`. Non-MIB files are filtered during loading by checking for `DEFINITIONS` and `::=` in the content.

`ScanModuleNames` extracts module names from raw bytes without a full parse, useful for indexing:

```go
names := gomib.ScanModuleNames(content) // e.g. ["IF-MIB"]
```

Sources can also list the modules they know about without parsing them:

```go
src, _ := gomib.Dir("/usr/share/snmp/mibs")
names, _ := src.ListModules() // e.g. ["IF-MIB", "IP-MIB", ...]
```

### Options

```go
gomib.Load(ctx,
    gomib.WithSource(src),
    gomib.WithSystemPaths(),                             // discover net-snmp/libsmi paths
    gomib.WithLogger(slog.Default()),                    // enable debug/trace logging
    gomib.WithResolverStrictness(mib.ResolverPermissive),    // strictness preset
    gomib.WithDiagnosticConfig(mib.DiagnosticConfig{   // fine-grained control
        Reporting: mib.ReportingDefault,
        FailAt: mib.SeverityError,
        Ignore: []string{"identifier-underscore"},
    }),
)
```

## Querying

Lookup methods take a plain name and return nil if not found:

```go
obj := m.Object("ifIndex")
node := m.Node("ifEntry")
typ := m.Type("DisplayString")
```

For qualified lookup, scope through the module (see [Module-scoped queries](#module-scoped-queries) below).

Other lookup methods: `Node`, `Type`, `Notification`, `Group`, `Compliance`, `Capability`.

`Symbol` provides a unified lookup across all definition types:

```go
sym := m.Symbol("ifIndex")       // returns a Symbol wrapping any definition kind
sym.Name()                       // "ifIndex"
sym.Object()                     // *Object (or nil if not an OBJECT-TYPE)
sym.Node()                       // *Node
sym.IsZero()                     // true if nothing was found
```

### Resolve

`Resolve` accepts any common format - plain name, qualified name, or numeric OID - and returns the matching node:

```go
node := m.Resolve("ifDescr")                     // plain name
node  = m.Resolve("IF-MIB::ifDescr")             // qualified
node  = m.Resolve("1.3.6.1.2.1.2.2.1.2")        // numeric OID
node  = m.Resolve(".1.3.6.1.2.1.2.2.1.2")       // leading dot
```

`ResolveOID` converts any of those formats to a numeric OID, and also handles instance-suffixed forms:

```go
oid, err := m.ResolveOID("IF-MIB::ifDescr.5")    // OID{1,3,6,1,2,1,2,2,1,2,5}
oid, err  = m.ResolveOID("ifDescr.5")             // same result
oid, err  = m.ResolveOID("1.3.6.1.2.1.2.2.1.2")  // numeric pass-through
```

`ResolveOID` is the inverse of `FormatOID`:

```go
formatted := m.FormatOID(oid)       // "IF-MIB::ifDescr.5"
back, _   := m.ResolveOID(formatted) // round-trips to the same OID
```

### OID lookups

For cases where you already know the format, lower-level methods avoid parsing:

```go
oid, _ := mib.ParseOID("1.3.6.1.2.1.2.2.1.1")
node := m.NodeByOID(oid)            // exact match
node  = m.LongestPrefixByOID(oid)   // longest matching prefix
```

`LookupInstance` returns both the matched node and the instance suffix, which is the standard pattern for processing SNMP varbinds:

```go
lookup := m.LookupInstance(oid)
lookup.Node().Name()   // "ifIndex"
lookup.Suffix()        // OID{5} (trailing arcs after the matched node)
```

### Module-scoped queries

```go
mod := m.Module("IF-MIB")
obj := mod.Object("ifIndex")
typ := mod.Type("InterfaceIndex")
```

### Module metadata and imports

`Module` exposes both MODULE-IDENTITY metadata and import-resolution details:

```go
mod := m.Module("IF-MIB")

mod.Name()         // "IF-MIB"
mod.Language()     // SMIv2
mod.SourcePath()   // file path, or "" for embedded base modules
mod.IsBase()       // false for user/system modules, true for built-ins
mod.OID()          // module identity OID
mod.Organization() // ORGANIZATION clause
mod.ContactInfo()  // CONTACT-INFO clause
mod.Description()  // DESCRIPTION clause
mod.LastUpdated()  // LAST-UPDATED value
mod.Revisions()    // []Revision
mod.Imports()      // []Import from the IMPORTS clause

mod.ImportsSymbol("DisplayString") // true if listed in IMPORTS
mod.IsImportUsed("DisplayString")  // true if referenced during resolution
mod.ImportSource("DisplayString")  // resolved source module
```

Base modules are always present and define the SMI language itself:

```go
base := m.Module("SNMPv2-SMI")
base.IsBase()      // true
base.SourcePath()  // ""
base.Node("enterprises").OID() // 1.3.6.1.4.1
```

Module introspection methods:

```go
mod.DefinesSymbol("ifIndex")        // true if module defines this name
mod.ImportsSymbol("DisplayString")  // true if module imports this name
mod.IsImportUsed("DisplayString")   // true if import was referenced during resolution
mod.ImportSource("DisplayString")   // *Module for the source of the import

for sym := range mod.Definitions() {  // iterate all definitions as Symbol values
    fmt.Println(sym.Name())
}
```

### Collections

```go
m.Objects()        // all OBJECT-TYPE definitions
m.Tables()         // tables only
m.Scalars()        // scalars only
m.Columns()        // columns only
m.Rows()           // rows only
m.Types()          // all type definitions
m.Notifications()  // all notifications
m.Groups()         // all groups
m.Compliances()    // all compliances
m.Capabilities()   // all capabilities
m.Modules()        // all loaded modules
m.AllSymbols()     // all definitions across all modules (iter.Seq[Symbol])
```

### OID tree iteration

```go
for node := range m.Nodes() {
    fmt.Println(node.OID(), node.Name(), node.Kind())
}

// Subtree iteration
node := m.Node("ifEntry")
for child := range node.Subtree() {
    fmt.Println(child.Name())
}

// Node metadata (populated for base module OIDs and OBJECT-IDENTITY definitions)
node.Description() // OID assignment description text
node.Reference()   // reference string
```

## Objects

Each `Object` carries its type, access level, status, and position in the OID tree:

```go
obj := m.Object("ifType")

obj.Name()        // "ifType"
obj.OID()         // 1.3.6.1.2.1.2.2.1.3
obj.Kind()        // column
obj.Access()      // read-only
obj.Status()      // current
obj.Type().Name() // "IANAifType"
obj.Units()       // "" (empty if not set)
obj.Description() // "The type of interface..."
```

### Tables

```go
table := m.Object("ifTable")
table.IsTable()  // true

row := table.Entry()           // ifEntry
cols := row.Columns()          // [ifIndex, ifDescr, ifType, ...]
idxs := row.EffectiveIndexes() // handles AUGMENTS

for _, idx := range idxs {
    fmt.Printf("INDEX %s (implied=%v)\n", idx.Object.Name(), idx.Implied)
}
```

Navigate from any level: `obj.Table()` returns the containing table, `obj.Row()` returns the containing row.

### Effective constraints

Constraints can be defined inline on an object or inherited through its type chain. Effective SIZE and value ranges are intersections of every applicable constraint in the parent chain and the base type domain, so they can be narrower than any one declared list:

```go
ranges := obj.EffectiveRanges() // intersected value ranges
sizes := obj.EffectiveSizes()   // intersected SIZE constraints

obj.EffectiveRangesConstrained() // any applicable value constraint exists
obj.EffectiveSizesConstrained()  // any applicable SIZE constraint exists
obj.EffectiveEnums()             // nearest enum values
obj.EffectiveBits()              // nearest BITS values
obj.EffectiveDisplayHint()       // nearest display hint string
```

An empty effective slice is ambiguous by itself. If the corresponding `Effective*Constrained()` method is false, the value is unconstrained; if it is true, the declared constraints have an empty intersection. The same methods are available on `Type`.

Each `mib.Range` is inclusive and has `Min` and `Max` endpoints of type `mib.RangeBound`. Endpoint kinds preserve signed values, unsigned values through `math.MaxUint64`, symbolic `MIN`/`MAX`, and unresolved source text:

```go
for i := range ranges {
    r := &ranges[i]
    if !r.IsResolved() { // symbolic or raw endpoint remains
        fmt.Println(r.Min.Kind, r.Min.String(), r.Max.Kind, r.Max.String())
        continue
    }

    if min, ok := r.Min.AsInt64(); ok {
        fmt.Println("signed-compatible minimum:", min)
    }
    if max, ok := r.Max.AsUint64(); ok {
        fmt.Println("unsigned-compatible maximum:", max)
    }
}
```

`AsInt64()` accepts signed endpoints and unsigned endpoints no greater than `math.MaxInt64`. `AsUint64()` accepts unsigned endpoints and non-negative signed endpoints. Both return `ok == false` for unrepresentable numeric values, symbolic `MIN`/`MAX`, and raw endpoints. Inspect `RangeBound.Kind` (`RangeBoundSigned`, `RangeBoundUnsigned`, `RangeBoundMin`, `RangeBoundMax`, or `RangeBoundRaw`) when the distinction matters.

## Types

Types form chains: a textual convention references a parent type, which may reference another, down to a base SMI type.

```go
typ := m.Type("DisplayString")

typ.Name()                // "DisplayString"
typ.IsTextualConvention() // true
typ.Base()                // OCTET STRING
typ.DisplayHint()         // "255a"
typ.Sizes()               // [{0 255}]
typ.Parent().Name()       // base type reference

// Walk the chain
for t := typ; t != nil; t = t.Parent() {
    fmt.Printf("%s (base: %s)\n", t.Name(), t.Base())
}

// Effective values resolve through the chain
typ.EffectiveBase()              // underlying base type
typ.EffectiveDisplayHint()       // first non-empty hint in chain
typ.EffectiveEnums()             // first non-empty enum set
typ.EffectiveRanges()            // intersection across the chain
typ.EffectiveSizes()             // SIZE intersection across the chain
typ.EffectiveRangesConstrained() // distinguishes absent from empty
typ.EffectiveSizesConstrained()  // distinguishes absent from empty
```

Classification helpers: `IsCounter()`, `IsGauge()`, `IsString()`, `IsEnumeration()`, `IsBits()`.

## Notifications

```go
notif := m.Notification("linkDown")

notif.Name()        // "linkDown"
notif.OID()         // 1.3.6.1.6.3.1.1.5.3
notif.Status()      // current
notif.Description() // "A linkDown trap..."

for _, obj := range notif.Objects() {
    fmt.Printf("  varbind: %s (%s)\n", obj.Name(), obj.OID())
}
```

SMIv1 `TRAP-TYPE` definitions populate `TrapInfo()`:

```go
if trap := notif.TrapInfo(); trap != nil {
    fmt.Printf("enterprise=%s trap-number=%d\n", trap.Enterprise, trap.TrapNumber)
}
```

## Conformance

`gomib` also exposes `OBJECT-GROUP`, `NOTIFICATION-GROUP`, `MODULE-COMPLIANCE`, and `AGENT-CAPABILITIES`:

```go
grp := m.Group("ifGeneralInformationGroup")
grp.IsNotificationGroup() // false for OBJECT-GROUP
for _, member := range grp.Members() {
    fmt.Println(member.Name(), member.OID())
}

comp := m.Compliance("ifCompliance3")
for _, mod := range comp.Modules() {
    fmt.Println("MODULE", mod.ModuleName) // empty = current module
    fmt.Println("mandatory groups:", mod.MandatoryGroups)
}

cap := m.Capability("someAgentCaps")
if cap != nil {
    fmt.Println(cap.ProductRelease())
    for _, sup := range cap.Supports() {
        fmt.Println("SUPPORTS", sup.ModuleName, sup.Includes)
    }
}
```

## Diagnostics

Loading produces diagnostics for issues found during parsing and resolution.

```go
m, err := gomib.Load(ctx, gomib.WithSource(src))

// Check for errors
if m.HasErrors() {
    fmt.Println("errors found")
}

// Inspect all diagnostics
for _, d := range m.Diagnostics() {
    fmt.Printf("[%s] %s: %s (line %d)\n", d.Severity, d.Module, d.Message, d.Line)
}

// Check unresolved references
for _, ref := range m.Unresolved() {
    fmt.Printf("unresolved %s: %s in %s\n", ref.Kind, ref.Symbol, ref.Module)
}
```

### Strictness and Reporting

Resolver strictness controls fallback behavior:

| Resolver strictness | Constant | Behavior |
|---|---|---|
| Strict | `ResolverStrict` | Tier 1 only (deterministic scope/import resolution) |
| Normal | `ResolverNormal` | Tier 1 + Tier 2 (constrained assumptions) |
| Permissive | `ResolverPermissive` | Tier 1 + Tier 2 + Tier 3 (global search fallbacks) |

```go
m, _ := gomib.Load(ctx, gomib.WithSource(src), gomib.WithResolverStrictness(mib.ResolverPermissive))
```

Diagnostic reporting is independent and configured with `WithDiagnosticConfig`:

```go
gomib.WithDiagnosticConfig(mib.DiagnosticConfig{
    Reporting: mib.ReportingDefault,
    FailAt: mib.SeverityError,              // fail on Error or worse
    Ignore: []string{"identifier-underscore"}, // suppress specific codes
    Overrides: map[string]mib.Severity{
        "import-not-found": mib.SeverityWarning, // downgrade to warning
    },
})
```

CLI equivalents follow the same split:

- `gomib load --strict` and `gomib load --permissive` change resolver behavior.
- `gomib load --report silent|quiet|default|verbose` changes diagnostic output.
- `gomib get` and `gomib find` default to permissive resolution with silent reporting.
- `gomib lint` is separate from resolver strictness and filters diagnostics by severity.

## Syntax and Tokenization

The `syntax` package provides direct access to the lexer, parser, and concrete syntax tree (CST) for tooling, language servers, linters, or syntax highlighting.

### Tokenization

```go
import "github.com/golangsnmp/gomib/syntax"

source := []byte(`ifIndex OBJECT-TYPE SYNTAX InterfaceIndex`)
tokens := syntax.Tokenize(source)

for _, tok := range tokens {
    if tok.Kind == syntax.TokEOF {
        break
    }
    text := source[tok.Span.Start:tok.Span.End]
    fmt.Printf("%-20s %s\n", tok.Kind.LibsmiName(), text)
}
```

Token kind classification methods on `TokenKind`:

```go
tok.Kind.IsKeyword()             // any keyword
tok.Kind.IsMacroKeyword()        // OBJECT-TYPE, MODULE-IDENTITY, etc.
tok.Kind.IsTypeKeyword()         // INTEGER, Counter32, OCTET, etc.
tok.Kind.IsClauseKeyword()       // SYNTAX, MAX-ACCESS, STATUS, etc.
tok.Kind.IsIdentifier()          // uppercase or lowercase identifier
tok.Kind.IsStructuralKeyword()   // DEFINITIONS, BEGIN, END, IMPORTS, etc.
tok.Kind.IsStatusAccessKeyword() // current, deprecated, read-only, etc.
```

### Parsing to CST

`Parse` produces a lossless concrete syntax tree that preserves every byte of the original source, including whitespace and comments. This is useful for tooling that needs structural information without running the full resolution pipeline.

```go
file, diags := syntax.Parse(source)

for _, mod := range file.Modules {
    name := string(source[mod.Name.Span.Start:mod.Name.Span.End])
    fmt.Printf("module %s: %d definitions\n", name, len(mod.Body))

    for _, def := range mod.Body {
        switch d := def.(type) {
        case *syntax.ObjectTypeNode:
            fmt.Printf("  OBJECT-TYPE %s\n", source[d.Name.Span.Start:d.Name.Span.End])
        case *syntax.ModuleIdentityNode:
            fmt.Printf("  MODULE-IDENTITY %s\n", source[d.Name.Span.Start:d.Name.Span.End])
        }
    }
}

// Round-trip: reconstructing from the CST reproduces the original source exactly
reconstructed := syntax.ReconstructText(file, source)
```

### Line tables

`LineTable` provides bidirectional conversion between byte offsets (used by spans) and line/column numbers (used by editors and LSP):

```go
lt := syntax.BuildLineTable(source)
line, col := lt.LineCol(tok.Span.Start)       // byte offset -> line/col
offset, ok := lt.Offset(cursorLine, cursorCol) // line/col -> byte offset
```

### Span queries on resolved modules

The resolved model provides span-based queries for language server features like go-to-definition and hover:

```go
mod := m.Module("IF-MIB")
offset, ok := mod.Offset(cursorLine, cursorCol) // line/col -> byte offset
if ok {
    sc := mod.SpanContext(offset)
    switch sc.Kind {
    case mib.SpanContextImport:
        fmt.Printf("import %s from %s\n", sc.Name, sc.Module)
    case mib.SpanContextOidRef:
        fmt.Printf("OID reference to %s\n", sc.Name)
    case mib.SpanContextSyntax:
        fmt.Printf("SYNTAX type %s\n", sc.Name)
    case mib.SpanContextDefinition:
        fmt.Printf("definition %s in %s\n", sc.Name, sc.Module)
    }
}
```

## CLI

The `cmd/gomib` tool provides a command-line interface for MIB operations:

```
gomib load IF-MIB                    # load and show statistics
gomib get -m IF-MIB ifIndex          # query by name
gomib get -m IF-MIB 1.3.6.1.2.1.2   # query by OID
gomib inspect ifDescr                # deep-dive with type chain, provenance, diagnostics
gomib dump IF-MIB                    # JSON output
gomib lint IF-MIB                    # check for issues
gomib find -p testdata/corpus/primary 'if*'        # search by pattern
gomib normalize IF-MIB               # emit canonical SMIv2 text
gomib trace -m IF-MIB ifEntry        # trace resolution
gomib paths                          # show search paths
gomib list                           # list available modules
```

Use `-p PATH` to specify MIB search paths (repeatable). Without `-p`, paths are discovered from net-snmp and libsmi configuration (config files, `MIBDIRS`/`SMIPATH` env vars, standard default directories).

See [cmd/gomib/README.md](cmd/gomib/README.md) for full command reference.
