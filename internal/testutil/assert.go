// Package testutil provides test assertion helpers.
package testutil

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Equal fails the test if got != want.
func Equal[T comparable](t testing.TB, want, got T, msgAndArgs ...any) {
	t.Helper()
	if got != want {
		t.Fatalf("%s\n  got:  %v\n  want: %v", formatMsg(msgAndArgs), got, want)
	}
}

// SliceEqual fails the test if the slices are not equal.
func SliceEqual[T comparable](t testing.TB, want, got []T, msgAndArgs ...any) {
	t.Helper()
	if !slices.Equal(want, got) {
		t.Fatalf("%s\n  got:  %v (len %d)\n  want: %v (len %d)",
			formatMsg(msgAndArgs), got, len(got), want, len(want))
	}
}

// NoError fails the test if err is not nil.
func NoError(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", formatMsg(msgAndArgs), err)
	}
}

// Error fails the test if err is nil.
func Error(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error, got nil", formatMsg(msgAndArgs))
	}
}

// NotNil fails the test if v is nil.
func NotNil(t testing.TB, v any, msgAndArgs ...any) {
	t.Helper()
	if isNil(v) {
		t.Fatalf("%s: expected non-nil, got nil", formatMsg(msgAndArgs))
	}
}

// Nil fails the test if v is not nil.
func Nil(t testing.TB, v any, msgAndArgs ...any) {
	t.Helper()
	if !isNil(v) {
		t.Fatalf("%s: expected nil, got %v", formatMsg(msgAndArgs), v)
	}
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return rv.IsNil()
	}
	return false
}

// NotEmpty fails the test if the slice is empty.
func NotEmpty[T any](t testing.TB, s []T, msgAndArgs ...any) {
	t.Helper()
	if len(s) == 0 {
		t.Fatalf("%s: expected non-empty slice, got empty", formatMsg(msgAndArgs))
	}
}

// Len fails the test if len(s) != want.
func Len[T any](t testing.TB, s []T, want int, msgAndArgs ...any) {
	t.Helper()
	if len(s) != want {
		t.Fatalf("%s: expected len %d, got %d", formatMsg(msgAndArgs), want, len(s))
	}
}

// True fails the test if cond is false.
func True(t testing.TB, cond bool, msgAndArgs ...any) {
	t.Helper()
	if !cond {
		t.Fatalf("%s: expected true, got false", formatMsg(msgAndArgs))
	}
}

// False fails the test if cond is true.
func False(t testing.TB, cond bool, msgAndArgs ...any) {
	t.Helper()
	if cond {
		t.Fatalf("%s: expected false, got true", formatMsg(msgAndArgs))
	}
}

// Contains fails the test if s does not contain substr.
func Contains(t testing.TB, s, substr string, msgAndArgs ...any) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("%s: expected %q to contain %q", formatMsg(msgAndArgs), s, substr)
	}
}

// Greater fails the test if a <= b.
func Greater[T cmp.Ordered](t testing.TB, a, b T, msgAndArgs ...any) {
	t.Helper()
	if cmp.Compare(a, b) <= 0 {
		t.Fatalf("%s: expected %v > %v", formatMsg(msgAndArgs), a, b)
	}
}

// Fail fails the test immediately with the given message.
func Fail(t testing.TB, msgAndArgs ...any) {
	t.Helper()
	t.Fatal(formatMsg(msgAndArgs))
}

func formatMsg(msgAndArgs []any) string {
	if len(msgAndArgs) == 0 {
		return "assertion failed"
	}
	msg, ok := msgAndArgs[0].(string)
	if !ok {
		return "assertion failed"
	}
	if len(msgAndArgs) == 1 {
		return msg
	}
	return fmt.Sprintf(msg, msgAndArgs[1:]...)
}
