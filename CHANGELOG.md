# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/golangsnmp/gomib/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/golangsnmp/gomib/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/golangsnmp/gomib/compare/v0.7.1...v0.8.0
