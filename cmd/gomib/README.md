# gomib CLI

MIB parser and query tool.

```
gomib <command> [options] [arguments]
```

## Global Options

```
-p, --path PATH   Add MIB search path (repeatable)
-v, --verbose     Enable debug logging
-vv               Enable trace logging (implies -v)
--version         Show version
-h, --help        Show help
```

When no `-p` paths are given, gomib discovers system MIB paths from net-snmp and libsmi configuration.

## Commands

### paths

Show the MIB search paths that would be used. Shows both custom (-p) and system-discovered paths with annotations.

```
gomib paths
gomib paths -p /usr/share/snmp/mibs
```

### list

List available module names from configured sources without loading or parsing them.

```
gomib list -p testdata/corpus/primary
gomib list -p testdata/corpus/primary --count
gomib list -p testdata/corpus/primary --format json
```

Flags: `--count` (print count only), `--format` (text/json).

### load

Load and resolve MIB modules. Loads all available modules when none specified.

```
gomib load IF-MIB
gomib load --strict IF-MIB
gomib load --permissive IF-MIB
gomib load --stats IF-MIB
gomib load --report verbose IF-MIB
gomib load -p testdata/corpus/primary
```

Flags: `--strict` (RFC compliance), `--permissive` (vendor MIBs), `--report LEVEL` (silent/quiet/default/verbose), `--stats` (detailed statistics).

### get

Query OID or name lookups. Accepts numeric OIDs, plain names, or qualified names (MODULE::name). Modules selected via -m flag, defaults to all. Uses permissive+silent strictness by default.

```
gomib get -m IF-MIB ifIndex
gomib get -m IF-MIB 1.3.6.1.2.1.2.2.1.1
gomib get IF-MIB::ifIndex
gomib get -m IF-MIB -t ifTable
gomib get -m IF-MIB --format json ifIndex
gomib get -m IF-MIB --full ifIndex
```

Flags: `-m MODULE` (repeatable), `-t`/`--tree` (show subtree), `--max-depth N` (implies `--tree`), `--format` (text/json), `--full` (full descriptions in text output), `--strict`, `--permissive`.

### dump

Export resolved MIB data as canonical JSON. Dumps all available modules when none specified.

```
gomib dump IF-MIB
gomib dump -o 1.3.6.1.2.1.2 IF-MIB
gomib dump -p testdata/corpus/primary
gomib dump --no-descriptions --compact IF-MIB | jq '.modules'
gomib dump --report verbose IF-MIB
```

Flags: `-o OID` (subtree filter), `--compact` (minified), `--no-descriptions`, `--report LEVEL` (silent/quiet/default/verbose), `--strict` (RFC compliance), `--permissive` (vendor MIBs).

### normalize

Emit modules as canonical SMIv2 text. Normalizes SMIv1/v2 differences, OID syntax, status values, and access levels into a consistent output format. Normalizes all available modules when none specified.

```
gomib normalize IF-MIB
gomib normalize --no-conformance IF-MIB
gomib normalize --no-descriptions IF-MIB
gomib normalize --no-sequences IF-MIB
gomib normalize --permissive -o /tmp/normalized -p /path/to/mibs
gomib normalize -o /tmp/normalized IF-MIB SNMPv2-MIB
```

Flags: `-o DIR` (write each module to a file in DIR), `--no-conformance` (omit conformance constructs), `--no-descriptions` (omit DESCRIPTION clauses), `--no-sequences` (omit reconstructed SEQUENCE types), `--strict` (RFC compliance), `--permissive` (vendor MIBs).

When no modules are specified, synthetic base modules (SNMPv2-SMI, SNMPv2-TC, etc.) are excluded from output since consumers provide their own built-in definitions.

### lint

Check modules for issues. Lints all available modules when none specified.

```
gomib lint IF-MIB
gomib lint -p testdata/corpus/primary
gomib lint --level 6 IF-MIB
gomib lint --format json IF-MIB
gomib lint --format sarif IF-MIB
gomib lint --ignore "identifier-*" IF-MIB
gomib lint --summary IF-MIB
gomib lint --list-codes
```

Flags: `--level N` (report up to severity N, 0-6, higher is more verbose), `--fail-on N`, `--ignore CODE` (repeatable, supports globs), `--only CODE`, `--format` (text/json/sarif/compact), `--group-by` (module/code/severity), `--summary`, `--quiet`, `--list-codes` (show all diagnostic codes).

### find

Search for names across loaded MIBs using glob patterns. Searches objects, notifications, groups, compliances, capabilities, and OID tree nodes. Modules selected via -m flag, defaults to all. Uses permissive+silent strictness by default.

```
gomib find -p testdata/corpus/primary 'if*'
gomib find -p testdata/corpus/primary --kind table '*'
gomib find -p testdata/corpus/primary --type Counter32 'if*'
gomib find -m IF-MIB 'if*'
gomib find -m IF-MIB -p testdata/corpus/primary --count 'if*'
gomib find --format json -m IF-MIB '*Entry'
```

Flags: `-m MODULE` (repeatable), `--kind` (scalar/table/row/column/notification/node/group/compliance/capability/module-identity/object-identity), `--type` (base type filter for objects), `--count` (print count only), `--format` (text/json), `--strict`, `--permissive`.

### trace

Trace symbol resolution for debugging. Shows where a symbol is defined, how it resolves, and any related issues. Modules selected via -m flag, defaults to all.

```
gomib trace -m IF-MIB ifIndex
gomib trace -m IF-MIB ifEntry
gomib trace -p testdata/corpus/primary ifEntry
```

Flags: `-m MODULE` (repeatable), `--strict`, `--permissive`.

### version

Show version information.

```
gomib version
```

## Exit Codes

- 0 - success (clean load, match found, no issues)
- 1 - negative result (issues/violations found, no match)
- 2 - operational error (load failure, parse error)

Output-only commands (dump, list, paths) use 0/1 only.
