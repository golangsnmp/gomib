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
-h, --help        Show help
```

When no `-p` paths are given, gomib discovers system MIB paths from net-snmp and libsmi configuration.

## Commands

### paths

Show the MIB search paths that would be used.

```
gomib paths
gomib paths -p /usr/share/snmp/mibs
```

### list

List available module names from configured sources without loading or parsing them.

```
gomib list -p testdata/corpus/primary
gomib list -p testdata/corpus/primary --count
gomib list -p testdata/corpus/primary --json
```

Flags: `--count` (print count only), `--json` (JSON array output).

### load

Load and resolve MIB modules. Reports statistics and diagnostics.

```
gomib load IF-MIB
gomib load --strict IF-MIB
gomib load --permissive IF-MIB
gomib load --stats IF-MIB
```

Flags: `--strict` (RFC compliance), `--permissive` (vendor MIBs), `--level N` (strictness 0-6), `--stats` (detailed statistics).

### get

Query OID or name lookups. Accepts numeric OIDs, plain names, or qualified names (MODULE::name).

```
gomib get -m IF-MIB ifIndex
gomib get -m IF-MIB 1.3.6.1.2.1.2.2.1.1
gomib get IF-MIB SNMPv2-MIB -- sysDescr
gomib get -m IF-MIB -t ifTable
gomib get --all -p testdata/corpus/primary ifIndex
gomib get -m IF-MIB --format json ifIndex
gomib get -m IF-MIB --full ifIndex
```

Flags: `-m MODULE` (repeatable), `--all` (load all modules from search path), `-t`/`--tree` (show subtree), `--max-depth N`, `--format` (text/json), `--full` (untruncated descriptions).

### dump

Output modules or subtrees as JSON. Includes types, objects, notifications, groups, compliances, and capabilities.

```
gomib dump IF-MIB
gomib dump -o 1.3.6.1.2.1.2 IF-MIB
gomib dump --compact IF-MIB
```

Flags: `-o OID` (subtree filter), `--compact` (minified), `--no-tree`, `--no-descriptions`.

### lint

Check modules for issues.

```
gomib lint IF-MIB
gomib lint --level 4 IF-MIB
gomib lint --format json IF-MIB
gomib lint --format sarif IF-MIB
gomib lint --ignore "identifier-*" IF-MIB
gomib lint --summary IF-MIB
gomib lint --list-codes
```

Flags: `--level N` (severity threshold, 0-6), `--fail-on N`, `--ignore CODE` (repeatable, supports globs), `--only CODE`, `--format` (text/json/sarif/compact), `--group-by` (module/code/severity), `--summary`, `--quiet`, `--list-codes` (show all diagnostic codes).

### find

Search for object names across loaded MIBs using glob patterns.

```
gomib find --all -p testdata/corpus/primary 'if*'
gomib find --all -p testdata/corpus/primary --kind table '*'
gomib find --all -p testdata/corpus/primary --type Counter32 'if*'
gomib find -m IF-MIB -p testdata/corpus/primary --count 'if*'
```

Flags: `-m MODULE` (repeatable), `--all` (load all modules), `--kind` (scalar/table/row/column/notification), `--type` (base type filter), `--count` (print count only).

### trace

Trace symbol resolution for debugging. Shows where a symbol is defined, how it resolves, and any related issues.

```
gomib trace -m IF-MIB ifIndex
gomib trace -m IF-MIB ifEntry
gomib trace --all -p testdata/corpus/primary ifEntry
```

Flags: `-m MODULE` (repeatable), `--all` (load all modules from search path).

### version

Show version information.

```
gomib version
```

## Exit Codes

- 0 - success
- 1 - user error, processing failure, or severe diagnostic
- 2 - strict mode found errors or unresolved refs
