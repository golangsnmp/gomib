package mib

import (
	"strings"
	"testing"
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
			if got != tt.want {
				t.Errorf("Severity(%d).String() = %q, want %q", tt.sev, got, tt.want)
			}
		})
	}
}

func TestStrictnessLevelString(t *testing.T) {
	tests := []struct {
		level StrictnessLevel
		want  string
	}{
		{StrictnessStrict, "strict"},
		{StrictnessNormal, "normal"},
		{StrictnessPermissive, "permissive"},
		{StrictnessSilent, "silent"},
		{StrictnessLevel(99), "StrictnessLevel(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.want {
				t.Errorf("StrictnessLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
			}
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
		{KindCapabilities, "capabilities"},
		{Kind(99), "Kind(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.kind.String()
			if got != tt.want {
				t.Errorf("Kind(%d).String() = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestKindIsObjectType(t *testing.T) {
	objectTypes := []Kind{KindScalar, KindTable, KindRow, KindColumn}
	nonObjectTypes := []Kind{KindUnknown, KindInternal, KindNode, KindNotification, KindGroup, KindCompliance, KindCapabilities}

	for _, k := range objectTypes {
		if !k.IsObjectType() {
			t.Errorf("%s should be IsObjectType()", k)
		}
	}
	for _, k := range nonObjectTypes {
		if k.IsObjectType() {
			t.Errorf("%s should not be IsObjectType()", k)
		}
	}
}

func TestKindIsConformance(t *testing.T) {
	conformance := []Kind{KindGroup, KindCompliance, KindCapabilities}
	nonConformance := []Kind{KindUnknown, KindInternal, KindNode, KindScalar, KindTable, KindRow, KindColumn, KindNotification}

	for _, k := range conformance {
		if !k.IsConformance() {
			t.Errorf("%s should be IsConformance()", k)
		}
	}
	for _, k := range nonConformance {
		if k.IsConformance() {
			t.Errorf("%s should not be IsConformance()", k)
		}
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
		{AccessInstall, "install"},
		{AccessInstallNotify, "install-notify"},
		{AccessReportOnly, "report-only"},
		{AccessNotImplemented, "not-implemented"},
		{Access(99), "Access(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.access.String()
			if got != tt.want {
				t.Errorf("Access(%d).String() = %q, want %q", tt.access, got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestStatusIsSMIv1(t *testing.T) {
	smiv1 := []Status{StatusMandatory, StatusOptional}
	notSMIv1 := []Status{StatusCurrent, StatusDeprecated, StatusObsolete}

	for _, s := range smiv1 {
		if !s.IsSMIv1() {
			t.Errorf("%s should be IsSMIv1()", s)
		}
	}
	for _, s := range notSMIv1 {
		if s.IsSMIv1() {
			t.Errorf("%s should not be IsSMIv1()", s)
		}
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
		{LanguageSPPI, "SPPI"},
		{Language(99), "Language(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.lang.String()
			if got != tt.want {
				t.Errorf("Language(%d).String() = %q, want %q", tt.lang, got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("BaseType(%d).String() = %q, want %q", tt.bt, got, tt.want)
			}
		})
	}
}
