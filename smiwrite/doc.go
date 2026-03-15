// Package smiwrite emits resolved MIB models as canonical SMIv2 text.
//
// The emitter normalizes source variation such as SMIv1 versus SMIv2 forms,
// alternative OID syntax, bare type assignments, and TRAP-TYPE constructs
// into a single consistent SMIv2 output format.
//
// # Overview
//
// Use [Write] or [WriteAll] with a resolved
// [github.com/golangsnmp/gomib/mib.Mib] model when you need stable,
// generated module text for normalization, fixture generation, or
// cross-tool comparison.
package smiwrite
