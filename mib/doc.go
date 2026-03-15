// Package mib provides the resolved, read-only model produced by loading
// SNMP MIB modules.
//
// The central type is [Mib], which contains the merged OID tree, modules,
// objects, types, notifications, groups, compliances, capabilities, and
// diagnostics extracted from one or more MIB files.
//
// # Overview
//
// This package is the main query surface after [github.com/golangsnmp/gomib.Load]:
//
//   - [Mib] provides top-level lookup and iteration across loaded data.
//   - [Module] scopes queries and exposes import metadata.
//   - [Node] represents named OID tree nodes and supports tree navigation.
//   - [Object] models OBJECT-TYPE definitions including table, row, column,
//     index, and effective-constraint helpers.
//   - [Type] models textual conventions and type chains.
//   - [Notification], [Group], [Compliance], and [Capability] expose
//     conformance and notification definitions.
//
// # Model Semantics
//
// Model types use unexported fields with accessor methods. Small value
// structs such as [Range], [NamedValue], [Revision], [IndexEntry], and
// [TrapInfo] expose fields directly. Slice-returning accessors return copies,
// so callers may modify returned slices without affecting the model.
//
// # Resolution
//
// The package also contains the resolver entry point, [Resolve], which
// transforms normalized internal modules into a fully linked [Mib]. Most
// external callers should prefer [github.com/golangsnmp/gomib.Load] unless
// they are deliberately assembling a lower-level pipeline.
package mib
