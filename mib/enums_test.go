package mib

import (
	"strings"
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
)

func TestSeverityString(t *testing.T) {
	tests := []struct {
		sev  Severity
		want string
	}{
		{SeverityFatal, "fatal"},
		{SeveritySevere, "severe"},
		{SeverityError, "error"},
		{SeverityMinor, "minor"},
		{SeverityStyle, "style"},
		{SeverityWarning, "warning"},
		{SeverityInfo, "info"},
		{Severity(99), "Severity(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.sev.String()
			testutil.Equal(t, tt.want, got, "Severity().String()")
		})
	}
}

func TestResolverStrictnessString(t *testing.T) {
	tests := []struct {
		level ResolverStrictness
		want  string
	}{
		{ResolverStrict, "strict"},
		{ResolverNormal, "normal"},
		{ResolverPermissive, "permissive"},
		{ResolverStrictness(99), "ResolverStrictness(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.level.String()
			testutil.Equal(t, tt.want, got, "ResolverStrictness().String()")
		})
	}
}

func TestReportingLevelString(t *testing.T) {
	tests := []struct {
		level ReportingLevel
		want  string
	}{
		{ReportingSilent, "silent"},
		{ReportingQuiet, "quiet"},
		{ReportingDefault, "default"},
		{ReportingVerbose, "verbose"},
		{ReportingLevel(99), "ReportingLevel(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.level.String()
			testutil.Equal(t, tt.want, got, "ReportingLevel().String()")
		})
	}
}

func TestKindString(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindUnknown, "unknown"},
		{KindInternal, "internal"},
		{KindNode, "node"},
		{KindScalar, "scalar"},
		{KindTable, "table"},
		{KindRow, "row"},
		{KindColumn, "column"},
		{KindNotification, "notification"},
		{KindGroup, "group"},
		{KindCompliance, "compliance"},
		{KindCapability, "capabilities"},
		{Kind(99), "Kind(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.kind.String()
			testutil.Equal(t, tt.want, got, "Kind().String()")
		})
	}
}

func TestKindIsObjectType(t *testing.T) {
	objectTypes := []Kind{KindScalar, KindTable, KindRow, KindColumn}
	nonObjectTypes := []Kind{KindUnknown, KindInternal, KindNode, KindNotification, KindGroup, KindCompliance, KindCapability}

	for _, k := range objectTypes {
		testutil.True(t, k.IsObjectType(), "should be IsObjectType()")
	}
	for _, k := range nonObjectTypes {
		testutil.False(t, k.IsObjectType(), "should not be IsObjectType()")
	}
}

func TestKindIsConformance(t *testing.T) {
	conformance := []Kind{KindGroup, KindCompliance, KindCapability}
	nonConformance := []Kind{KindUnknown, KindInternal, KindNode, KindScalar, KindTable, KindRow, KindColumn, KindNotification}

	for _, k := range conformance {
		testutil.True(t, k.IsConformance(), "should be IsConformance()")
	}
	for _, k := range nonConformance {
		testutil.False(t, k.IsConformance(), "should not be IsConformance()")
	}
}

func TestAccessString(t *testing.T) {
	tests := []struct {
		access Access
		want   string
	}{
		{AccessNotAccessible, "not-accessible"},
		{AccessAccessibleForNotify, "accessible-for-notify"},
		{AccessReadOnly, "read-only"},
		{AccessReadWrite, "read-write"},
		{AccessReadCreate, "read-create"},
		{AccessWriteOnly, "write-only"},
		{AccessNotImplemented, "not-implemented"},
		{Access(99), "Access(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.access.String()
			testutil.Equal(t, tt.want, got, "Access().String()")
		})
	}
}

func TestStatusString(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusCurrent, "current"},
		{StatusDeprecated, "deprecated"},
		{StatusObsolete, "obsolete"},
		{StatusMandatory, "mandatory"},
		{StatusOptional, "optional"},
		{Status(99), "Status(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.status.String()
			testutil.Equal(t, tt.want, got, "Status().String()")
		})
	}
}

func TestStatusIsSMIv1(t *testing.T) {
	smiv1 := []Status{StatusMandatory, StatusOptional}
	notSMIv1 := []Status{StatusCurrent, StatusDeprecated, StatusObsolete}

	for _, s := range smiv1 {
		testutil.True(t, s.IsSMIv1(), "should be IsSMIv1()")
	}
	for _, s := range notSMIv1 {
		testutil.False(t, s.IsSMIv1(), "should not be IsSMIv1()")
	}
}

func TestLanguageString(t *testing.T) {
	tests := []struct {
		lang Language
		want string
	}{
		{LanguageUnknown, "unknown"},
		{LanguageSMIv1, "SMIv1"},
		{LanguageSMIv2, "SMIv2"},
		{Language(99), "Language(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.lang.String()
			testutil.Equal(t, tt.want, got, "Language().String()")
		})
	}
}

func TestBaseTypeString(t *testing.T) {
	tests := []struct {
		bt   BaseType
		want string
	}{
		{BaseUnknown, "unknown"},
		{BaseInteger32, "Integer32"},
		{BaseUnsigned32, "Unsigned32"},
		{BaseCounter32, "Counter32"},
		{BaseCounter64, "Counter64"},
		{BaseGauge32, "Gauge32"},
		{BaseTimeTicks, "TimeTicks"},
		{BaseIpAddress, "IpAddress"},
		{BaseOctetString, "OCTET STRING"},
		{BaseObjectIdentifier, "OBJECT IDENTIFIER"},
		{BaseBits, "BITS"},
		{BaseOpaque, "Opaque"},
		{BaseSequence, "SEQUENCE"},
		{BaseType(99), "BaseType(99)"},
	}

	for _, tt := range tests {
		t.Run(strings.ReplaceAll(tt.want, " ", "_"), func(t *testing.T) {
			got := tt.bt.String()
			testutil.Equal(t, tt.want, got, "BaseType().String()")
		})
	}
}
