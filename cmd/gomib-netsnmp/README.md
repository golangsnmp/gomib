# gomib-netsnmp

Cross-validation tool for comparing gomib against net-snmp.

This tool uses CGO to link against libnetsnmp for ground-truth comparison.
It is used for development and test case generation, not as a runtime dependency.

## Prerequisites

Install net-snmp development headers:

```bash
# Debian/Ubuntu
sudo apt-get install libsnmp-dev

# RHEL/CentOS/Fedora
sudo dnf install net-snmp-devel

# macOS
brew install net-snmp
```

## Building

This binary requires CGO and is not built by default:

```bash
cd gomib
CGO_ENABLED=1 go build -tags cgo ./cmd/gomib-netsnmp
```

The resulting binary `gomib-netsnmp` will be created in the current directory.

## Commands

### compare

Full semantic comparison between gomib and net-snmp:

```bash
./gomib-netsnmp compare -p /usr/share/snmp/mibs IF-MIB
./gomib-netsnmp compare -p ./testdata/corpus/primary SYNTHETIC-MIB
```

Compares:
- OID resolution
- Type mapping
- Access levels
- Enum values
- Index structures
- AUGMENTS relationships

### tables

Table-focused comparison:

```bash
./gomib-netsnmp tables -p ./testdata/corpus/primary SYNTHETIC-MIB
./gomib-netsnmp tables -detailed -p /usr/share/snmp/mibs IF-MIB
```

Compares:
- INDEX clause (names, order, IMPLIED)
- AUGMENTS targets
- Column membership

### testgen

Generate Go test cases from net-snmp ground truth:

```bash
# Generate table tests
./gomib-netsnmp testgen -type tables -p ./testdata/corpus/primary SYNTHETIC-MIB

# Generate OID tests
./gomib-netsnmp testgen -type oids -p ./testdata/corpus/primary SYNTHETIC-MIB

# Generate enum tests
./gomib-netsnmp testgen -type enums -p /usr/share/snmp/mibs IF-MIB

# Generate access tests
./gomib-netsnmp testgen -type access -p /usr/share/snmp/mibs IF-MIB

# Output to file
./gomib-netsnmp testgen -type tables -o generated_tests.go -p ./testdata SYNTHETIC-MIB
```

### validate

Validate existing test cases against net-snmp:

```bash
./gomib-netsnmp validate -p ./testdata/corpus/primary
./gomib-netsnmp validate -tests ./integration -p /usr/share/snmp/mibs
```

## Options

Common options for all commands:

| Option | Description |
|--------|-------------|
| `-p PATH` | Add MIB search path (repeatable) |
| `-o FILE` | Write output to file instead of stdout |
| `-json` | Output in JSON format |
| `-h` | Show help |

## Environment

The tool respects the `MIBDIRS` environment variable for MIB search paths,
following net-snmp conventions.

Default search paths (if no `-p` flags):
- `$MIBDIRS` (colon-separated)
- `~/.snmp/mibs`
- `/usr/share/snmp/mibs`
- `/usr/local/share/snmp/mibs`

## Example Workflow

1. Compare current implementation against net-snmp:

```bash
./gomib-netsnmp compare -p ./testdata/corpus/primary SYNTHETIC-MIB
```

2. Generate test cases for a module:

```bash
./gomib-netsnmp testgen -type tables -p ./testdata/corpus/primary SYNTHETIC-MIB > new_tests.go
```

3. Validate existing tests:

```bash
./gomib-netsnmp validate -p ./testdata/corpus/primary
```

## Notes

- Not included in default CI builds due to CGO dependency
- Results should be committed as static test cases
- The tool requires net-snmp headers and library at build time
