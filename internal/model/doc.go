// Package model defines the resolved MIB data model types.
//
// Types in this package have unexported fields with exported accessor methods.
// The internal/resolver package uses the exported setter functions (SetNode*,
// SetObject*, etc.) to build the model during resolution. The mib/ package
// re-exports these types as type aliases for the public API.
package model
