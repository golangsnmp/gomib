package testutil

import (
	"os"
	"testing"
)

// mockTB captures whether a test failure occurred.
type mockTB struct {
	testing.TB // embedded for unimplemented methods
	failed     bool
}

func (m *mockTB) Helper()                           {}
func (m *mockTB) Fatal(args ...any)                 { m.failed = true }
func (m *mockTB) Fatalf(format string, args ...any) { m.failed = true }

func TestEqual(t *testing.T) {
	m := &mockTB{}

	Equal(m, 1, 1)
	if m.failed {
		t.Error("Equal(1, 1) should pass")
	}

	m.failed = false
	Equal(m, "foo", "foo")
	if m.failed {
		t.Error("Equal(foo, foo) should pass")
	}

	m.failed = false
	Equal(m, 1, 2)
	if !m.failed {
		t.Error("Equal(1, 2) should fail")
	}
}

func TestSliceEqual(t *testing.T) {
	m := &mockTB{}

	SliceEqual(m, []int{1, 2, 3}, []int{1, 2, 3})
	if m.failed {
		t.Error("equal slices should pass")
	}

	m.failed = false
	SliceEqual(m, []int{}, []int{})
	if m.failed {
		t.Error("empty slices should pass")
	}

	m.failed = false
	SliceEqual(m, []int{1, 2}, []int{1, 2, 3})
	if !m.failed {
		t.Error("different length slices should fail")
	}

	m.failed = false
	SliceEqual(m, []int{1, 2, 3}, []int{1, 9, 3})
	if !m.failed {
		t.Error("different content should fail")
	}
}

func TestNoError(t *testing.T) {
	m := &mockTB{}

	NoError(m, nil)
	if m.failed {
		t.Error("NoError(nil) should pass")
	}

	m.failed = false
	NoError(m, os.ErrNotExist)
	if !m.failed {
		t.Error("NoError(err) should fail")
	}
}

func TestError(t *testing.T) {
	m := &mockTB{}

	Error(m, os.ErrNotExist)
	if m.failed {
		t.Error("Error(err) should pass")
	}

	m.failed = false
	Error(m, nil)
	if !m.failed {
		t.Error("Error(nil) should fail")
	}
}

func TestNil(t *testing.T) {
	m := &mockTB{}

	var nilPtr *int
	var nilSlice []int
	var nilMap map[string]int
	var nilChan chan int
	var nilFunc func()
	var nilInterface error

	// All should pass
	for _, v := range []any{nil, nilPtr, nilSlice, nilMap, nilChan, nilFunc, nilInterface} {
		m.failed = false
		Nil(m, v)
		if m.failed {
			t.Errorf("Nil(%T) should pass", v)
		}
	}

	// Typed nil in interface (the gotcha case)
	m.failed = false
	var typedNil error = (*os.PathError)(nil)
	Nil(m, typedNil)
	if m.failed {
		t.Error("Nil(typed nil in interface) should pass")
	}

	// Non-nil values should fail
	m.failed = false
	Nil(m, 42)
	if !m.failed {
		t.Error("Nil(42) should fail")
	}

	m.failed = false
	Nil(m, []int{})
	if !m.failed {
		t.Error("Nil(empty slice) should fail")
	}
}

func TestNotNil(t *testing.T) {
	m := &mockTB{}

	NotNil(m, 42)
	if m.failed {
		t.Error("NotNil(42) should pass")
	}

	m.failed = false
	NotNil(m, []int{})
	if m.failed {
		t.Error("NotNil(empty slice) should pass")
	}

	m.failed = false
	NotNil(m, make(map[string]int))
	if m.failed {
		t.Error("NotNil(empty map) should pass")
	}

	m.failed = false
	var nilPtr *int
	NotNil(m, nilPtr)
	if !m.failed {
		t.Error("NotNil(nil ptr) should fail")
	}
}

func TestNotEmpty(t *testing.T) {
	m := &mockTB{}

	NotEmpty(m, []int{1})
	if m.failed {
		t.Error("NotEmpty([1]) should pass")
	}

	m.failed = false
	NotEmpty(m, []int{})
	if !m.failed {
		t.Error("NotEmpty([]) should fail")
	}
}

func TestLen(t *testing.T) {
	m := &mockTB{}

	Len(m, []int{1, 2, 3}, 3)
	if m.failed {
		t.Error("Len([1,2,3], 3) should pass")
	}

	m.failed = false
	Len(m, []int{1, 2, 3}, 5)
	if !m.failed {
		t.Error("Len([1,2,3], 5) should fail")
	}
}

func TestTrueFalse(t *testing.T) {
	m := &mockTB{}

	True(m, true)
	if m.failed {
		t.Error("True(true) should pass")
	}

	m.failed = false
	True(m, false)
	if !m.failed {
		t.Error("True(false) should fail")
	}

	m.failed = false
	False(m, false)
	if m.failed {
		t.Error("False(false) should pass")
	}

	m.failed = false
	False(m, true)
	if !m.failed {
		t.Error("False(true) should fail")
	}
}

func TestContains(t *testing.T) {
	m := &mockTB{}

	Contains(m, "hello world", "world")
	if m.failed {
		t.Error("Contains(hello world, world) should pass")
	}

	m.failed = false
	Contains(m, "hello world", "foo")
	if !m.failed {
		t.Error("Contains(hello world, foo) should fail")
	}
}

func TestGreater(t *testing.T) {
	m := &mockTB{}

	Greater(m, 5, 3)
	if m.failed {
		t.Error("Greater(5, 3) should pass")
	}

	m.failed = false
	Greater(m, 3.14, 2.71)
	if m.failed {
		t.Error("Greater(3.14, 2.71) should pass")
	}

	m.failed = false
	Greater(m, "b", "a")
	if m.failed {
		t.Error("Greater(b, a) should pass")
	}

	m.failed = false
	Greater(m, 3, 5)
	if !m.failed {
		t.Error("Greater(3, 5) should fail")
	}

	m.failed = false
	Greater(m, 3, 3)
	if !m.failed {
		t.Error("Greater(3, 3) should fail")
	}
}

func TestFail(t *testing.T) {
	m := &mockTB{}

	Fail(m, "some message")
	if !m.failed {
		t.Error("Fail should always fail")
	}
}

func TestFormatMsg(t *testing.T) {
	if got := formatMsg(nil); got != "assertion failed" {
		t.Errorf("formatMsg(nil) = %q, want %q", got, "assertion failed")
	}

	if got := formatMsg([]any{"custom"}); got != "custom" {
		t.Errorf("formatMsg([custom]) = %q, want %q", got, "custom")
	}

	if got := formatMsg([]any{"value is %d", 42}); got != "value is 42" {
		t.Errorf("formatMsg with args = %q, want %q", got, "value is 42")
	}

	if got := formatMsg([]any{123}); got != "assertion failed" {
		t.Errorf("formatMsg(non-string) = %q, want %q", got, "assertion failed")
	}
}
