# gomib

Pure Go MIB parser. Parses SMIv1 and SMIv2 MIB files into a queryable model.

## Install

```bash
go get github.com/golangsnmp/gomib
```

Requires Go 1.24+.

## Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/golangsnmp/gomib"
)

func main() {
    // Load all MIBs from a directory tree
    source, _ := gomib.DirTree("/usr/share/snmp/mibs")
    mib, err := gomib.Load(source)
    if err != nil {
        log.Fatal(err)
    }

    // Look up an object
    obj := mib.Object("sysDescr")
    fmt.Println(obj.OID()) // 1.3.6.1.2.1.1.1

    // Look up by qualified name
    ifIndex := mib.ObjectByQualified("IF-MIB::ifIndex")
    fmt.Println(ifIndex.Kind()) // column

    // Look up by OID
    node := mib.Node("1.3.6.1.2.1.2.2.1.1")
    fmt.Println(node.Name) // ifIndex

    // Walk the tree
    mib.Walk(func(n *gomib.Node) bool {
        if n.Kind == gomib.KindTable {
            fmt.Println(n.Name, n.OID())
        }
        return true
    })
}
```

## Loading MIBs

```go
// Load all MIBs from a directory tree
source, err := gomib.DirTree("/path/to/mibs")
mib, err := gomib.Load(source)

// Load specific modules (with dependencies)
mib, err := gomib.LoadModules([]string{"IF-MIB", "IP-MIB"}, source)

// Multiple search paths
src1, _ := gomib.DirTree("/usr/share/snmp/mibs")
src2, _ := gomib.Dir("/opt/vendor/mibs")
mib, err := gomib.Load(gomib.Multi(src1, src2))

// With debug logging
logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
mib, err := gomib.Load(source, gomib.WithLogger(logger))
```

## Queries

```go
// By name
obj := mib.Object("ifIndex")
typ := mib.Type("DisplayString")
notif := mib.Notification("coldStart")

// By qualified name (MODULE::name)
obj := mib.ObjectByQualified("IF-MIB::ifIndex")

// By OID
node := mib.Node("1.3.6.1.2.1.2.2.1.1")
node := mib.NodeByOID(gomib.Oid{1, 3, 6, 1, 2, 1, 2, 2, 1, 1})

// Flexible lookup (tries qualified, OID, then name)
node := mib.FindNode("IF-MIB::ifIndex")
node := mib.FindNode("1.3.6.1.2.1.2.2.1.1")
node := mib.FindNode("ifIndex")
```

## Model

The resolved model contains:

- **Node** - Point in the OID tree with Kind (scalar, table, row, column, etc.)
- **Object** - OBJECT-TYPE definition with type, access, status, constraints
- **Type** - Type definition or textual convention
- **Notification** - NOTIFICATION-TYPE or TRAP-TYPE
- **Module** - MIB module metadata

```go
obj := mib.Object("ifIndex")
obj.OID()           // 1.3.6.1.2.1.2.2.1.1
obj.Kind()          // KindColumn
obj.Type.Base       // BaseInteger32
obj.Access          // AccessReadOnly
obj.Status          // StatusCurrent
obj.NamedValues     // enum values if any
obj.Size            // SIZE constraint if any
obj.ValueRange      // value range if any
```

## CLI

```bash
go install github.com/golangsnmp/gomib/cmd/gomib@latest

# Load and summarize
gomib load -p /usr/share/snmp/mibs IF-MIB

# Query an object
gomib get -m IF-MIB ifIndex

# Dump as JSON
gomib dump IF-MIB
```

## Examples

See [examples/](examples/) for runnable code.

## License

MIT
