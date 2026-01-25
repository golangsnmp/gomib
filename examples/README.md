# gomib examples

These examples demonstrate how to use the gomib package to parse and query MIB files.

## Running examples

From the `gomib/` directory:

```bash
go run ./examples/basic/
go run ./examples/lookup/
go run ./examples/walk/
go run ./examples/types/
go run ./examples/tables/
go run ./examples/notifications/
go run ./examples/modules/
go run ./examples/logging/
```

All examples use the test corpus in `testdata/corpus/primary/`.

## Examples

### basic

Load all MIBs and print a summary of what was loaded.

### lookup

Demonstrates different ways to find objects:
- By simple name (`sysDescr`)
- By qualified name (`IF-MIB::ifIndex`)
- By OID string (`1.3.6.1.2.1.1.1`)
- Using `FindNode()` which tries multiple formats

### walk

Traverse the OID tree:
- Count nodes by kind
- Print a subtree
- Walk from a specific node
- Find all tables in a module

### types

Explore type definitions:
- Look up textual conventions
- Examine enumerated types
- Find objects with constraints
- List Counter64 objects

### tables

Explore SNMP table structure:
- Table/entry/column hierarchy
- Index columns (including compound indices)
- AUGMENTS relationships

### notifications

Work with SNMP notifications (traps):
- Find notifications by name
- List notification objects
- Count notifications per module

### modules

Load specific modules:
- `LoadModules()` loads named modules plus dependencies
- Explore module metadata
- List objects and types per module

### logging

Enable debug logging to see internal operations:
- Debug level shows phase transitions
- Trace level shows per-item details
