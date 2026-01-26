// Package testutil provides test assertion helpers.
package testutil

import (
	"fmt"
	"strings"
	"testing"
)

// Equal fails the test if got != want.
func Equal[T comparable](t *testing.T, want, got T, msgAndArgs ...any) {
	t.Helper()
	if got != want {
		t.Fatalf("%s\n  got:  %v\n  want: %v", formatMsg(msgAndArgs), got, want)
	}
}

// NotNil fails the test if v is nil.
func NotNil[T any](t *testing.T, v *T, msgAndArgs ...any) {
	t.Helper()
	if v == nil {
		t.Fatalf("%s: expected non-nil, got nil", formatMsg(msgAndArgs))
	}
}

// Nil fails the test if v is not nil.
func Nil[T any](t *testing.T, v *T, msgAndArgs ...any) {
	t.Helper()
	if v != nil {
		t.Fatalf("%s: expected nil, got %v", formatMsg(msgAndArgs), v)
	}
}

// NotEmpty fails the test if the slice is empty.
func NotEmpty[T any](t *testing.T, s []T, msgAndArgs ...any) {
	t.Helper()
	if len(s) == 0 {
		t.Fatalf("%s: expected non-empty slice, got empty", formatMsg(msgAndArgs))
	}
}

// Len fails the test if len(s) != want.
func Len[T any](t *testing.T, s []T, want int, msgAndArgs ...any) {
	t.Helper()
	if len(s) != want {
		t.Fatalf("%s: expected len %d, got %d", formatMsg(msgAndArgs), want, len(s))
	}
}

// True fails the test if cond is false.
func True(t *testing.T, cond bool, msgAndArgs ...any) {
	t.Helper()
	if !cond {
		t.Fatalf("%s: expected true, got false", formatMsg(msgAndArgs))
	}
}

// False fails the test if cond is true.
func False(t *testing.T, cond bool, msgAndArgs ...any) {
	t.Helper()
	if cond {
		t.Fatalf("%s: expected false, got true", formatMsg(msgAndArgs))
	}
}

// Contains fails the test if s does not contain substr.
func Contains(t *testing.T, s, substr string, msgAndArgs ...any) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("%s: expected %q to contain %q", formatMsg(msgAndArgs), s, substr)
	}
}

// Greater fails the test if a <= b.
func Greater[T ~int | ~int64 | ~uint | ~uint64](t *testing.T, a, b T, msgAndArgs ...any) {
	t.Helper()
	if a <= b {
		t.Fatalf("%s: expected %v > %v", formatMsg(msgAndArgs), a, b)
	}
}

// Fail fails the test immediately with the given message.
func Fail(t *testing.T, msgAndArgs ...any) {
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
