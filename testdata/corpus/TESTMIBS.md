# Test MIB Corpus

A curated collection of 194 real-world MIBs for integration testing.

## Structure

```
corpus/
├── primary/           # Main corpus - self-contained, all deps included
│   ├── ietf/          # IETF/RFC standards (64)
│   ├── iana/          # IANA registries (5)
│   ├── ieee/          # IEEE standards (8)
│   ├── cisco/         # Cisco/Linksys (10)
│   ├── juniper/       # Juniper (16)
│   ├── alcatel/       # Alcatel/Nokia/Timetra (34)
│   ├── huawei/        # Huawei (8)
│   ├── adva/          # ADVA/FSP/Tropic (17)
│   ├── net-snmp/      # Net-SNMP/UCD (3)
│   ├── misc/          # Other vendors (27)
│   └── synthetic/     # Purpose-built test MIBs (2)
└── <future>/          # Edge-case corpora as needed
```

## Usage

```go
// Load with recursive directory traversal
src, _ := loader.DirTree("testdata/corpus/primary")
l := loader.New(src)
l.Load("IF-MIB")  // finds ietf/IF-MIB.mib
```

```bash
# CLI
gomib load -p ./testdata/corpus/primary IF-MIB IP-MIB
```

## Selection Criteria

MIBs were selected for maximum diversity:
- SMIv1 and SMIv2 coverage
- TRAP-TYPE (SMIv1) and NOTIFICATION-TYPE (SMIv2)
- AUGMENTS, IMPLIED INDEX, deep compound indices
- TEXTUAL-CONVENTION, BITS, enumerated INTEGER
- MODULE-COMPLIANCE, AGENT-CAPABILITIES
- Range from minimal to very large MIBs (10k+ objects)

## File Naming

All files use `.mib` extension for consistency.
