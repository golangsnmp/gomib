// Package gomib loads and resolves SNMP MIB modules.
//
// Call [Load] with one or more [Source] values to parse MIB files,
// resolve cross-module imports, build the OID tree, and return a
// read-only [mib.Mib] containing the merged result.
//
// # Overview
//
// Most callers should stay at this package layer:
//
//   - [Load] runs the full parse, lower, and resolve pipeline.
//   - [Source] implementations such as [Dir], [File], [Files], [FS],
//     and [Multi] control where modules are found.
//   - [WithModules] limits loading to selected modules and their
//     transitive dependencies.
//   - [WithSystemPaths] discovers net-snmp and libsmi search paths.
//
// # Query Model
//
// The resolved query model lives in the [github.com/golangsnmp/gomib/mib]
// package. Callers typically load through [Load] here, then query the
// resulting [mib.Mib] for modules, objects, types, notifications, table
// structure, and OID lookups.
//
// # Lower-Level Access
//
// For lower-level lexical access without full loading, use the
// [github.com/golangsnmp/gomib/token] package.
package gomib
