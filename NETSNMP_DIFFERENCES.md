# gomib vs net-snmp Differences

This document describes behavioral differences between gomib and net-snmp's MIB parsing. Understanding these helps when migrating code or comparing outputs.

## Module Preference

When multiple modules define the same OID (e.g., RFC1213-MIB and IP-MIB both define `ipForwarding`), the libraries return different definitions.

### net-snmp

From `snmplib/parse.c`:

- First module to define an OID sets properties (access, status, enums)
- Later modules are added to the module list but don't overwrite properties
- `MIB_REPLACE` flag (`-R` option) inverts this: last definition wins
- Load order depends on `MIBS` env var, filesystem traversal, dependency resolution

### gomib

Deterministic preference order:
1. SMIv2 modules over SMIv1
2. Newer LAST-UPDATED within same SMI version
3. Alphabetical module name as tiebreaker

### Practical impact

For `ipForwarding` (1.3.6.1.2.1.4.1):

```
RFC1213-MIB (SMIv1):
  STATUS  mandatory
  SYNTAX  INTEGER { forwarding(1), not-forwarding(2) }

IP-MIB (SMIv2):
  STATUS  current
  SYNTAX  INTEGER { forwarding(1), notForwarding(2) }
```

gomib returns IP-MIB definition. net-snmp typically returns RFC1213-MIB.

This affects: enum labels, status values, access levels for any OID defined in multiple modules.


## Status Normalization

gomib normalizes SMIv1 status values to SMIv2 equivalents:

| SMIv1 | gomib | net-snmp |
|-------|-------|----------|
| mandatory | current | mandatory |
| optional | deprecated | optional |

net-snmp preserves the original value.


## Default Values

gomib's `DefVal` provides both the interpreted value and original MIB syntax:

```go
dv := obj.DefaultValue()
dv.String()  // "0" - interpreted value
dv.Raw()     // "'00000000'H" - original syntax
dv.Value()   // []byte{0,0,0,0} - typed value

// Type-safe access
if n, ok := mib.DefValAs[int64](dv); ok {
    // use n
}
```

Hex strings (≤8 bytes) display as integers to match net-snmp's numeric interpretation.

| MIB syntax | dv.String() | dv.Raw() |
|------------|-------------|----------|
| `DEFVAL { '00000000'H }` | `0` | `'00000000'H` |
| `DEFVAL { ''H }` | `0` | `''H` |
| `DEFVAL { "public" }` | `"public"` | `"public"` |
| `DEFVAL { true }` | `true` | `true` |


## Range Values

net-snmp uses signed `int` for range bounds. Large unsigned values overflow:

```
MIB:      (0..4294967295)
gomib:    (0..4294967295)
net-snmp: (0..-1)
```

gomib uses `int64` and handles the full unsigned 32-bit range correctly.


## Node Kind

gomib infers semantic role from MIB structure:

| Kind | Meaning |
|------|---------|
| table | SEQUENCE OF container |
| row | Has INDEX or AUGMENTS |
| column | Child of row |
| scalar | Leaf, not in table |
| node | OID registration point |
| notification | NOTIFICATION-TYPE or TRAP-TYPE |

```go
if obj.IsTable() { ... }
if obj.Kind() == mib.KindColumn { ... }
```

net-snmp exposes ASN.1 macro type but not semantic role.


## Import Resolution

When resolving `IMPORTS foo, bar FROM SOME-MIB` and multiple modules named SOME-MIB exist:

**net-snmp**: Uses first module found (load order dependent).

**gomib**: Selects the candidate that exports all requested symbols. If multiple candidates qualify, prefers newer LAST-UPDATED.


## Loading Behavior

| Aspect | net-snmp | gomib |
|--------|----------|-------|
| Directory traversal | Flat only | `Dir()` flat, `DirTree()` recursive |
| Definition preference | Load order dependent | Deterministic rules |
| Error handling | Writes to stderr | `Mib.Diagnostics()` slice |
| Duplicate OIDs | First wins (or last with `-R`) | SMIv2 preferred |


## Type Names

Base type names are consistent. For textual conventions, use `Object.Type().Name()` for the TC name (e.g., "DisplayString") or `Object.Type().Base()` for the underlying type.
