# gomib

Pure Go SNMP MIB parser. Parses SMIv1 and SMIv2 MIB files into a queryable model.

## Why gomib?

**Permissive parsing.** Many MIB files have syntax errors or vendor quirks. gomib uses error recovery to load files that strict parsers reject.

**Fully resolved.** `Load()` parses all files, resolves imports, computes OIDs, and infers node kinds. The returned model has no lazy evaluation or deferred errors.

**Efficient queries.** OID lookups are O(depth) via trie. Name lookups are indexed.

## Install

```bash
go get github.com/golangsnmp/gomib
```

Requires Go 1.24+.

## Quick Start

```go
// Load MIBs from a directory tree
source, _ := gomib.DirTree("/usr/share/snmp/mibs")
mib, _ := gomib.Load(ctx, source)

// Find* methods accept name, qualified name, or OID string
obj := mib.FindObject("IF-MIB::ifIndex")
obj := mib.FindObject("1.3.6.1.2.1.2.2.1.1")

// Resolve SNMP instance OIDs to their defining object
node := mib.LongestPrefixByOID(gomib.Oid{1,3,6,1,2,1,2,2,1,1,5}) // ifIndex
```

### Loading Options

```go
// Load specific modules (resolves dependencies automatically)
mib, _ := gomib.LoadModules(ctx, []string{"IF-MIB", "IP-MIB"}, source)

// Multiple search paths
mib, _ := gomib.Load(ctx, gomib.Multi(src1, src2))

// With debug logging
mib, _ := gomib.Load(ctx, source, gomib.WithLogger(slog.Default()))
```

## Working with Objects

```go
obj := mib.FindObject("ifOperStatus")

// Basic properties
obj.OID()    // 1.3.6.1.2.1.2.2.1.8
obj.Kind()   // KindColumn
obj.Access() // AccessReadOnly
obj.Status() // StatusCurrent

// Type info (walks type chain for effective values)
obj.Type().Base()          // BaseInteger32
obj.EffectiveEnums()       // [{up 1} {down 2} {testing 3} ...]
obj.EffectiveDisplayHint() // from textual convention
obj.EffectiveSizes()       // SIZE constraints
obj.EffectiveRanges()      // value range constraints

// Look up enum by label
if nv, ok := obj.Enum("up"); ok {
    fmt.Println(nv.Value) // 1
}

// Quick type checks
obj.Type().IsEnumeration() // true (has enum values)
obj.Type().IsCounter()     // false
```

## Working with Tables

```go
// Get all tables
for _, tbl := range mib.Tables() {
    fmt.Println(tbl.Name(), tbl.OID())
}

// Navigate from table to columns
tbl := mib.FindObject("ifTable")
tbl.Entry().Name()           // ifEntry (the row object)
tbl.Entry().EffectiveIndexes() // [{ifIndex, false}]
for _, col := range tbl.Columns() {
    fmt.Println(col.Name(), col.Type().Base())
}

// Navigate from column up to table
col := mib.FindObject("ifInOctets")
col.IsColumn()       // true
col.Row().Name()     // ifEntry
col.Table().Name()   // ifTable
```

## Tree Traversal

```go
// Iterate all nodes
for node := range mib.Nodes() {
    if node.Kind() == gomib.KindScalar {
        fmt.Println(node.Name())
    }
}

// Traverse subtree
ifMIB := mib.FindNode("IF-MIB::ifMIB")
for node := range ifMIB.Descendants() {
    fmt.Println(node.OID(), node.Name())
}
```

## CLI

```bash
go install github.com/golangsnmp/gomib/cmd/gomib@latest

gomib load -p /usr/share/snmp/mibs IF-MIB    # summarize
gomib get -m IF-MIB ifIndex                   # query object
gomib dump IF-MIB                             # dump as JSON
```

## More Examples

See [examples/](examples/) for runnable code covering modules, types, notifications, and more.

For SNMP protocol support, pair gomib with [gosnmp](https://github.com/gosnmp/gosnmp).
