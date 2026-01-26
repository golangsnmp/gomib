package integration

import (
	"testing"

	"github.com/golangsnmp/gomib/internal/testutil"
	"github.com/golangsnmp/gomib/mib"
)

// DefValTestCase defines a test case for DEFVAL verification.
type DefValTestCase struct {
	Name       string // object name
	Module     string // module name
	WantKind   mib.DefValKind
	WantString string // expected String() output
	WantRaw    string // expected Raw() output
}

var defvalTests = []DefValTestCase{
	// OID reference: DEFVAL { zeroDotZero }
	{
		Name:       "syntheticSystemObjectID",
		Module:     "SYNTHETIC-MIB",
		WantKind:   mib.DefValKindOID,
		WantString: "0.0",
		WantRaw:    "zeroDotZero",
	},
	// Enum: DEFVAL { enabled }
	{
		Name:       "syntheticTrapEnable",
		Module:     "SYNTHETIC-MIB",
		WantKind:   mib.DefValKindEnum,
		WantString: "enabled",
		WantRaw:    "enabled",
	},
	// TC-based enum: DEFVAL { unknown }
	{
		Name:       "syntheticSimpleStatus",
		Module:     "SYNTHETIC-MIB",
		WantKind:   mib.DefValKindEnum,
		WantString: "unknown",
		WantRaw:    "unknown",
	},
	// Integer: DEFVAL { 42 }
	{
		Name:       "syntheticDefvalInteger",
		Module:     "SYNTHETIC-MIB",
		WantKind:   mib.DefValKindInt,
		WantString: "42",
		WantRaw:    "42",
	},
	// String: DEFVAL { "default" }
	{
		Name:       "syntheticDefvalString",
		Module:     "SYNTHETIC-MIB",
		WantKind:   mib.DefValKindString,
		WantString: `"default"`,
		WantRaw:    `"default"`,
	},
	// Hex string: DEFVAL { 'DEADBEEF'H } - interpreted as integer
	{
		Name:       "syntheticDefvalHex",
		Module:     "SYNTHETIC-MIB",
		WantKind:   mib.DefValKindBytes,
		WantString: "3735928559", // 0xDEADBEEF
		WantRaw:    "'DEADBEEF'H",
	},
	// Empty hex string: DEFVAL { ''H }
	{
		Name:       "syntheticDefvalEmptyHex",
		Module:     "SYNTHETIC-MIB",
		WantKind:   mib.DefValKindBytes,
		WantString: "0",
		WantRaw:    "''H",
	},
	// BITS: DEFVAL { { flag0, flag2 } }
	{
		Name:       "syntheticDefvalBits",
		Module:     "SYNTHETIC-MIB",
		WantKind:   mib.DefValKindBits,
		WantString: "{ flag0, flag2 }",
		WantRaw:    "{ flag0, flag2 }",
	},
}

func TestDefValResolution(t *testing.T) {
	if len(defvalTests) == 0 {
		t.Skip("no defval test cases defined yet")
	}

	m := loadCorpus(t)

	for _, tc := range defvalTests {
		t.Run(tc.Module+"::"+tc.Name, func(t *testing.T) {
			obj := getObject(t, m, tc.Module, tc.Name)

			dv := obj.DefaultValue()
			if dv.IsZero() {
				t.Fatal("DefaultValue() is zero, expected a value")
			}

			testutil.Equal(t, tc.WantKind, dv.Kind(), "Kind mismatch")
			testutil.Equal(t, tc.WantString, dv.String(), "String() mismatch")
			testutil.Equal(t, tc.WantRaw, dv.Raw(), "Raw() mismatch")
		})
	}
}

func TestDefValGenericAccess(t *testing.T) {
	m := loadCorpus(t)

	// Test DefValAs[string] for enum
	obj := getObject(t, m, "SYNTHETIC-MIB", "syntheticTrapEnable")
	dv := obj.DefaultValue()

	if v, ok := mib.DefValAs[string](dv); !ok {
		t.Error("DefValAs[string] failed for enum DefVal")
	} else {
		testutil.Equal(t, "enabled", v, "DefValAs[string] value")
	}

	// Wrong type should return false
	if _, ok := mib.DefValAs[int64](dv); ok {
		t.Error("DefValAs[int64] should fail for enum DefVal")
	}
}
