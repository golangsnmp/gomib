# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.13.2] - 2026-08-13

### Changed

- Upgrade `actions/setup-go` from v6 to v7 in CI, fuzzing, API compatibility, and release workflows

## [0.13.1] - 2026-08-13

### Fixed

- Use native path separators for `Dir` source result paths on Windows while preserving slash-separated paths for `FS` source labels
- Make cross-platform CI checks independent of host-installed net-snmp directories and compatible with the current Go 1.26 lint baseline

## [0.13.0] - 2026-08-13

### Breaking Changes

- Change `mib.Range.Min` and `mib.Range.Max` from `int64` to `RangeBound`, preserving signed, unsigned, symbolic `MIN`/`MAX`, and unresolved raw endpoints; callers must inspect `RangeBound.Kind` or use `AsInt64()`/`AsUint64()` instead of reading integer endpoints directly
- Change `Range.IsResolved()` to a pointer receiver; calls on non-addressable `Range` values must first assign or take an address
- Change `Type.EffectiveSizes()`, `Type.EffectiveRanges()`, and the corresponding `Object` methods to intersect constraints across the complete parent chain and base type domain instead of returning the nearest non-empty list; use the new `EffectiveSizesConstrained()` and `EffectiveRangesConstrained()` methods to distinguish no constraint from an empty intersection

### Added

- Add `RangeBound`, `RangeBoundKind`, endpoint constructors, conversion helpers, comparison, and concrete-value inspection to the public `mib` package
- Add diagnostics for empty constraint intersections, type dependency cycles, overflowing generic trap numbers, and INTEGER enumeration values outside the Integer32 range

### Changed

- Preserve full-width unsigned, symbolic `MIN`/`MAX`, and malformed raw range endpoints in resolved constraints, schema output, and SMI output
- Infer SMI language from base-module identity, imports, and strong syntax evidence instead of defaulting modules without SMIv2 imports to SMIv1
- Index objects, notifications, groups, compliances, and capabilities by definition name so lookups remain correct when OID node labels differ or collide
- Validate decoded module candidates before applying file and source precedence, including candidates discovered from changed files or duplicate advertisements
- Store configured diagnostic severity overrides on emitted diagnostics and use those effective severities for failure checks
- Expand literal `$HOME` placeholders in net-snmp configuration and `MIBDIRS` paths

### Fixed

- Fix MACRO bodies containing quoted `END`, repeated `EXPORTS` clauses, and semicolons inside `EXPORTS` comments
- Fix forwarded imports being accepted when the final module does not define the symbol, and retain unresolved references in their module scope in exported output
- Fix invalid or impossible `LAST-UPDATED` values influencing duplicate-module preference
- Fix typed CLI `find` filters returning non-object definition kinds

## [0.12.0] - 2026-07-23

### Added

- Add exclusive end line and column positions to diagnostics when a source range is available

## [0.11.0] - 2026-04-07

### Breaking Changes

- Remove `token/` package; all token types, constants, and `Tokenize()` now live in the new `syntax/` package
- `Module.Definitions()` now returns `iter.Seq[Symbol]` instead of a slice
- Remove `DropModules` method
- Remove `Module.Definitions` field from internal IR (replaced by typed slices: Objects, Types, Notifications, Groups, Compliances, Capabilities)

### Added

- Add `syntax/` package with lossless CST, token types, and source-level query APIs
  - `Parse()` produces a full lossless concrete syntax tree (`ModuleFile`)
  - `Tokenize()` lexes source into token stream (moved from `token/`)
  - `CursorContextAt()` classifies syntactic context at a byte offset (module, definition, clause, imports, comment, string)
  - `TokenAt()` finds the CST token at a byte offset (for precise identifier extraction)
  - `BuildLineTable()` / `LineTable.LineCol()` / `LineTable.Offset()` for line/column mapping
  - `ClauseKind` enum for SYNTAX, STATUS, DESCRIPTION, INDEX, etc.
  - `MacroName()` returns the macro keyword for a definition node
  - `SyntaxBaseTypeNames` lists valid SYNTAX clause base type names
  - All CST node types exported (modules, definitions, clauses, type syntax, OID components, conformance)
  - `SyntaxToken` and `Trivia` types for whitespace/comment-preserving round-trips via `ReconstructText()`
- Add `SymbolKind` type and constants to `mib/` package (Object, Type, TextualConvention, Notification, Group, NotificationGroup, Compliance, Capability, Node)
- Add `SpanContext` and `SpanContextKind` types to `mib/` package (Definition, Import, OidRef, Syntax)
- Add `Module.SpanContext()` for offset-based span classification
- Add `Module.Offset()` for line/col to byte offset conversion
- Add RFC-sourced descriptions and references to all base module types and OID nodes
- Add `Node.Descendants()` iterator
- Add per-item spans in resolved model for INCLUDES and CREATION-REQUIRES clauses
- Add source context to load errors from `fsSource` and `loadAllModules`

### Changed

- Replace direct-to-IR parser with lossless CST pipeline (parse -> lower -> validate)
- Replace hand-constructed base module IR with embedded SMI source files
- Split `mib/` into `internal/model` and `internal/resolver` (public re-exports in `mib/` preserved)
- Eliminate AST package and lowering pass (CST lowers directly to Module IR)
- Replace `OidComponent` interface hierarchy with flat struct
- Downgrade conformance diagnostics (`DiagGroupMemberUnresolved`, `DiagGroupObjectsNotification`, `DiagGroupNotificationsObject`) from Error to Minor severity
- Use per-clause spans for resolver diagnostics (ACCESS, DEFVAL, SYNTAX, DISPLAY-HINT, STATUS, INDEX now point at the specific clause, not the whole definition)
- Use per-member spans for group and notification diagnostics (individual member names instead of whole definition)
- Convert SNMP administrative OID nodes to OBJECT-IDENTITY with RFC descriptions
- Use windowed 2-digit year interpretation in date parsing (>= 70 maps to 19xx, < 70 maps to 20xx)
- Thread stdout/stderr through CLI struct, removing global mutation from tests
- Unify Node() and Symbol() name-collision priority logic
- Simplify import format to flat storage

### Fixed

- Fix parser recovery to stop at lowercase type assignments (was skipping valid definitions during error recovery)
- Fix CLI pipe deadlock on Windows (concurrent pipe draining instead of sequential)
- Fix `resolveUltimateDefiner` silently returning wrong module on broken import chains
- Fix span loss during lowering for OCTET STRING and OBJECT IDENTIFIER type syntax
- Fix `Notification.TrapInfo()` returning internal struct instead of clone
- Fix hex/binary string error spans not including the bad suffix character
- Fix `BaseSequence` missing from schema export

## [0.10.0] - 2026-03-19

### Added

- Add inspect subcommand for deep-dive symbol inspection (type chains, import provenance, group membership, scoped diagnostics)
- Add NodesByName for multi-result name lookup
- Add Type.EffectiveTC to walk type chain for nearest textual convention
- Add IndexEntry.FixedSize for fixed-width encoding queries
- Add index suffix decoding for table instance OIDs (DecodeSuffix, IndexValue, DecodedIndex types)
- Extend Object.EffectiveIndexes to delegate to parent row on columns
- Add global fallbacks to linkObjectIndexes in permissive mode (INDEX and AUGMENTS resolution)
- Add tests for global fallback code paths and corpus test helpers

### Changed

- Simplify Node.Module to return OID-phase owner directly, removing entity-cascading priority chain
- Move module list out of newResolverContext, making registerModules the single owner of ctx.modules
- Deduplicate find command search loops with findFilter struct and collectMatches helper
- Clean up minor CLI and resolver duplication (reportOpt helper, oidComponentNameAndModule extraction)
- Document intentional moduleOid drop in lowering for MODULE-COMPLIANCE and AGENT-CAPABILITIES

### Fixed

- Fix scanModuleNames skipping files with comments before DEFINITIONS
- Fix dump command exit code for hard load failure (return exitError instead of exitIssue)
- Fix silent test degradation in corpus helpers (fatal on failure instead of returning empty)
- Fix exit codes for fs.Parse errors in list, paths, dump commands

## [0.9.0] - 2026-03-18

### Added

- Add display hint parsing and formatting API (ParseDisplayHint, FormatInteger, FormatOctets, ScaleInteger)
- Add parsed display hint accessors on Object and Type
- Add LookupInstance method and OidLookup type for combined node + instance suffix lookup
- Add File and Files source constructors for loading individual MIB files by path
- Add CI, pkg.go.dev, and Go Report Card badges to README

### Changed

- Refactor FormatOID to use LookupInstance internally
- Move display hint validation from resolver into public displayhint API

## [0.8.0] - 2026-03-15

### Breaking Changes

- Remove v1 prefix from export schema types and functions (V1Foo -> ExportFoo, v1Foo -> exportFoo)
- Remove --all flag from CLI commands, default to all when no modules given
- Swap CLI exit codes: 1 for negative result, 2 for operational error
- Move CLI load diagnostics from stdout to stderr
- Switch CLI get output from header+indent to key-value format
- Change CLI get and find to use -m only for module selection, no positional args

### Added

- Register semantic nodes (OBJECT-TYPE, NOTIFICATION-TYPE, groups, compliances, capabilities) on module in resolver, fixing qualified name resolution
- Add qualified name round-trip test across all nodes (643 nodes)
- Expand CLI find kind filter from 5 to 11 values (add node, group, compliance, capability, module-identity, object-identity)
- Expand CLI find search scope to groups, compliances, capabilities, tree
- Add --format (text, json) to CLI find and list commands
- Add --report and --no-descriptions to CLI dump command
- Add examples/modules and examples/notifications
- Add doc.go files with expanded godoc for gomib, mib, smiwrite, token packages

### Changed

- Default CLI get and find to permissive+silent strictness
- Show both custom and system paths with annotations in CLI paths command
- Rename CLI list --json to --format json
- Expand README with module metadata, conformance queries, TRAP-TYPE, and CLI strictness/reporting sections
- Add severity level table and strictness note to cmd/gomib/README.md

### Fixed

- Fix description nulling inconsistency in dump export (use *string for nullable descriptions)
- Fix path traversal in normalize output path
- Fix godoc link to mib.DiagnosticConfig

[Unreleased]: https://github.com/golangsnmp/gomib/compare/v0.13.2...HEAD
[0.13.2]: https://github.com/golangsnmp/gomib/compare/v0.13.1...v0.13.2
[0.13.1]: https://github.com/golangsnmp/gomib/compare/v0.13.0...v0.13.1
[0.13.0]: https://github.com/golangsnmp/gomib/compare/v0.12.0...v0.13.0
[0.12.0]: https://github.com/golangsnmp/gomib/compare/v0.11.0...v0.12.0
[0.11.0]: https://github.com/golangsnmp/gomib/compare/v0.10.0...v0.11.0
[0.10.0]: https://github.com/golangsnmp/gomib/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/golangsnmp/gomib/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/golangsnmp/gomib/compare/v0.7.1...v0.8.0
