# gomib

Go library for parsing and querying SNMP MIB files.

Supports SMIv1, SMIv2, and SPPI modules. Loads MIBs from directories, directory trees, or embedded filesystems. Resolves imports, builds the OID tree, and provides typed access to objects, types, notifications, and conformance definitions.

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

`Dir` searches a single flat directory. `DirTree` recursively indexes a directory tree. `FS` wraps an `fs.FS` (useful with `embed.FS`). `Multi` tries multiple sources in order.

```go
// Single directory
src, err := gomib.Dir("/usr/share/snmp/mibs")

// Recursive tree (indexed once at construction)
src, err := gomib.DirTree("/usr/share/snmp/mibs")

// Embedded filesystem
//go:embed mibs
var mibFS embed.FS
src := gomib.FS("embedded", mibFS)

// Combine sources (first match wins)
src := gomib.Multi(systemSrc, vendorSrc)
```

`Must` variants (`MustDir`, `MustDirTree`) panic on error for use in `var` blocks.

Files are matched by extension: no extension, `.mib`, `.smi`, `.txt`, `.my`. Override with `WithExtensions`. Non-MIB files are filtered during loading by checking for `DEFINITIONS` and `::=` in the content.

### Options

```go
gomib.Load(ctx,
    gomib.WithSource(src),
    gomib.WithSystemPaths(),                             // discover net-snmp/libsmi paths
    gomib.WithLogger(slog.Default()),                    // enable debug/trace logging
    gomib.WithStrictness(mib.StrictnessPermissive),    // strictness preset
    gomib.WithDiagnosticConfig(mib.DiagnosticConfig{   // fine-grained control
        Level:  mib.StrictnessNormal,
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

### OID lookups

```go
oid, _ := mib.ParseOID("1.3.6.1.2.1.2.2.1.1")
node := m.NodeByOID(oid)            // exact match
node  = m.LongestPrefixByOID(oid)   // longest matching prefix
```

### Module-scoped queries

```go
mod := m.Module("IF-MIB")
obj := mod.Object("ifIndex")
typ := mod.Type("InterfaceIndex")
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

Constraints can be defined inline on the object or inherited through the type chain. The `Effective*` methods walk both:

```go
obj.EffectiveEnums()       // enum values
obj.EffectiveBits()        // BITS values
obj.EffectiveRanges()      // value ranges
obj.EffectiveSizes()       // size constraints
obj.EffectiveDisplayHint() // display hint string
```

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
typ.EffectiveBase()        // underlying base type
typ.EffectiveDisplayHint() // first non-empty hint in chain
typ.EffectiveEnums()       // first non-empty enum set
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

### Strictness levels

Four presets control how strictly MIBs are validated:

| Level | Constant | Behavior |
|-------|----------|----------|
| Strict | `StrictnessStrict` | RFC-only, no fallbacks |
| Normal | `StrictnessNormal` | Default, safe fallbacks for common issues |
| Permissive | `StrictnessPermissive` | Tolerant of vendor MIB violations |
| Silent | `StrictnessSilent` | Accept everything, suppress diagnostics |

```go
m, _ := gomib.Load(ctx, gomib.WithSource(src), gomib.WithStrictness(mib.StrictnessPermissive))
```

For fine-grained control, use `WithDiagnosticConfig`:

```go
gomib.WithDiagnosticConfig(mib.DiagnosticConfig{
    Level:  mib.StrictnessNormal,
    FailAt: mib.SeverityError,              // fail on Error or worse
    Ignore: []string{"identifier-underscore"}, // suppress specific codes
    Overrides: map[string]mib.Severity{
        "import-not-found": mib.SeverityWarning, // downgrade to warning
    },
})
```

## CLI

The `cmd/gomib` tool provides a command-line interface for MIB operations:

```
gomib load IF-MIB                    # load and show statistics
gomib get -m IF-MIB ifIndex          # query by name
gomib get -m IF-MIB 1.3.6.1.2.1.2   # query by OID
gomib dump IF-MIB                    # JSON output
gomib lint IF-MIB                    # check for issues
gomib trace -m IF-MIB ifEntry        # trace resolution
```

Use `-p PATH` to specify MIB search paths (repeatable). Without `-p`, paths are discovered from net-snmp and libsmi configuration (config files, `MIBDIRS`/`SMIPATH` env vars, standard default directories).
