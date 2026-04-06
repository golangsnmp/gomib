package mib

import (
	"log/slog"

	"github.com/golangsnmp/gomib/internal/model"
	"github.com/golangsnmp/gomib/internal/module"
	"github.com/golangsnmp/gomib/internal/resolver"
)

// --- Core model types ---

type (
	Mib          = model.Mib
	Node         = model.Node
	Module       = model.Module
	Object       = model.Object
	Type         = model.Type
	Notification = model.Notification
	Group        = model.Group
	Compliance   = model.Compliance
	Capability   = model.Capability
	Symbol       = model.Symbol
	SymbolKind   = model.SymbolKind
	OidLookup    = model.OidLookup
)

// --- OID ---

type OID = model.OID

func ParseOID(s string) (OID, error) { return model.ParseOID(s) }

// --- Value types ---

type (
	Range             = model.Range
	NamedValue        = model.NamedValue
	Revision          = model.Revision
	IndexEntry        = model.IndexEntry
	Import            = model.Import
	ImportSymbol      = model.ImportSymbol
	OidRef            = model.OidRef
	TrapInfo          = model.TrapInfo
	SyntaxConstraints = model.SyntaxConstraints
)

// --- DefVal ---

type (
	DefValKind = model.DefValKind
	DefVal     = model.DefVal
)

const (
	SymbolKindNone              = model.SymbolKindNone
	SymbolKindObject            = model.SymbolKindObject
	SymbolKindType              = model.SymbolKindType
	SymbolKindTextualConvention = model.SymbolKindTextualConvention
	SymbolKindNotification      = model.SymbolKindNotification
	SymbolKindGroup             = model.SymbolKindGroup
	SymbolKindNotificationGroup = model.SymbolKindNotificationGroup
	SymbolKindCompliance        = model.SymbolKindCompliance
	SymbolKindCapability        = model.SymbolKindCapability
	SymbolKindNode              = model.SymbolKindNode
)

const (
	DefValKindUnset  = model.DefValKindUnset
	DefValKindInt    = model.DefValKindInt
	DefValKindUint   = model.DefValKindUint
	DefValKindString = model.DefValKindString
	DefValKindBytes  = model.DefValKindBytes
	DefValKindEnum   = model.DefValKindEnum
	DefValKindBits   = model.DefValKindBits
	DefValKindOID    = model.DefValKindOID
)

// DefValAs extracts the default value as the concrete type T.
func DefValAs[T any](d DefVal) (T, bool) { return model.DefValAs[T](d) }

// --- Unresolved references ---

type (
	UnresolvedKind = model.UnresolvedKind
	UnresolvedRef  = model.UnresolvedRef
)

const (
	UnresolvedImport             = model.UnresolvedImport
	UnresolvedType               = model.UnresolvedType
	UnresolvedOID                = model.UnresolvedOID
	UnresolvedIndex              = model.UnresolvedIndex
	UnresolvedNotificationObject = model.UnresolvedNotificationObject
)

// --- Display hint ---

type (
	DisplayHint     = model.DisplayHint
	IntegerHint     = model.IntegerHint
	IntegerFormat   = model.IntegerFormat
	OctetStringHint = model.OctetStringHint
	OctetSegment    = model.OctetSegment
	OctetFormat     = model.OctetFormat
	HexCase         = model.HexCase
)

const (
	HexUpper = model.HexUpper
	HexLower = model.HexLower
)

const (
	IntegerDecimal = model.IntegerDecimal
	IntegerHex     = model.IntegerHex
	IntegerOctal   = model.IntegerOctal
	IntegerBinary  = model.IntegerBinary
)

const (
	OctetDecimal = model.OctetDecimal
	OctetHex     = model.OctetHex
	OctetOctal   = model.OctetOctal
	OctetAscii   = model.OctetAscii
	OctetUtf8    = model.OctetUtf8
)

func ParseDisplayHint(hint string) *DisplayHint { return model.ParseDisplayHint(hint) }
func IsValidIntegerHint(hint string) bool       { return model.IsValidIntegerHint(hint) }
func IsValidOctetStringHint(hint string) bool   { return model.IsValidOctetStringHint(hint) }
func FormatInteger(hint string, value int64, hexCase HexCase) (string, bool) {
	return model.FormatInteger(hint, value, hexCase)
}
func ScaleInteger(hint string, value int64) (float64, bool) { return model.ScaleInteger(hint, value) }
func FormatOctets(hint string, data []byte, hexCase HexCase) (string, bool) {
	return model.FormatOctets(hint, data, hexCase)
}

// --- Index decoding ---

type (
	IndexValueKind = model.IndexValueKind
	IndexValue     = model.IndexValue
	DecodedIndex   = model.DecodedIndex
)

const (
	IndexValueInteger          = model.IndexValueInteger
	IndexValueIpAddress        = model.IndexValueIpAddress
	IndexValueOctetString      = model.IndexValueOctetString
	IndexValueObjectIdentifier = model.IndexValueObjectIdentifier
)

func DecodeSuffix(indexes []IndexEntry, suffix OID) []DecodedIndex {
	return model.DecodeSuffix(indexes, suffix)
}

// NameRefNames returns just the name strings from a slice of NameRefs.
var NameRefNames = model.NameRefNames

// --- Compliance / Capability value types ---

type (
	ComplianceModule      = model.ComplianceModule
	ComplianceGroup       = model.ComplianceGroup
	ComplianceObject      = model.ComplianceObject
	CapabilitiesModule    = model.CapabilitiesModule
	ObjectVariation       = model.ObjectVariation
	NotificationVariation = model.NotificationVariation
	NameRef               = model.NameRef
)

// --- Span context ---

type (
	SpanContext     = model.SpanContext
	SpanContextKind = model.SpanContextKind
)

const (
	SpanContextNone       = model.SpanContextNone
	SpanContextDefinition = model.SpanContextDefinition
	SpanContextImport     = model.SpanContextImport
	SpanContextOidRef     = model.SpanContextOidRef
	SpanContextSyntax     = model.SpanContextSyntax
)

// --- Resolve entry point ---

// Resolve transforms normalized modules into a fully resolved [Mib].
//
// Resolution runs these phases in order:
//
//  1. Registration: index modules and definitions
//  2. Imports: resolve imported symbols across modules
//  3. Types: build type links and compute base types
//  4. OIDs: build the OID tree from symbolic references
//  5. Semantics: infer node kinds and create objects
//
// If logger is nil, logging is disabled. If strictness is nil, defaults to
// [ResolverNormal]. If diagConfig is nil, defaults to [DefaultConfig].
func Resolve(mods []*module.Module, logger *slog.Logger, strictness *ResolverStrictness, diagConfig *DiagnosticConfig) *Mib {
	return resolver.Resolve(mods, logger, strictness, diagConfig)
}
