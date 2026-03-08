package testutil

import (
	"strings"
	"testing"
)

// Equal asserts that got equals want.
func Equal[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// NoError asserts that err is nil.
func NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ErrorContains asserts that err is non-nil and its message contains substr.
func ErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Errorf("error %q does not contain %q", err.Error(), substr)
	}
}

// True asserts that v is true.
func True(t *testing.T, v bool, msg string) {
	t.Helper()
	if !v {
		t.Errorf("expected true: %s", msg)
	}
}

// False asserts that v is false.
func False(t *testing.T, v bool, msg string) {
	t.Helper()
	if v {
		t.Errorf("expected false: %s", msg)
	}
}
