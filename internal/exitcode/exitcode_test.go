package exitcode

import (
	"errors"
	"testing"
)

func TestExitError_Error(t *testing.T) {
	err := &ExitError{Code: SessionNotFound, Message: "session not found: foo"}
	if err.Error() != "session not found: foo" {
		t.Errorf("Error() = %q, want %q", err.Error(), "session not found: foo")
	}
}

func TestExitError_Is(t *testing.T) {
	err := Errorf(Timeout, "timed out after %ds", 30)
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal("expected errors.As to match *ExitError")
	}
	if exitErr.Code != Timeout {
		t.Errorf("Code = %d, want %d", exitErr.Code, Timeout)
	}
}

func TestExitError_Wrap(t *testing.T) {
	inner := errors.New("connection refused")
	err := Wrap(inner, DaemonNotRunning, "daemon is not running")
	if err.Error() != "daemon is not running: connection refused" {
		t.Errorf("Error() = %q, want %q", err.Error(), "daemon is not running: connection refused")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal("expected errors.As to match *ExitError")
	}
	if exitErr.Code != DaemonNotRunning {
		t.Errorf("Code = %d, want %d", exitErr.Code, DaemonNotRunning)
	}
	if !errors.Is(err, inner) {
		t.Error("expected errors.Is to match inner error")
	}
}

func TestConstants(t *testing.T) {
	if Success != 0 {
		t.Errorf("Success = %d, want 0", Success)
	}
	if GeneralError != 1 {
		t.Errorf("GeneralError = %d, want 1", GeneralError)
	}
	if SessionNotFound != 2 {
		t.Errorf("SessionNotFound = %d, want 2", SessionNotFound)
	}
	if DaemonNotRunning != 3 {
		t.Errorf("DaemonNotRunning = %d, want 3", DaemonNotRunning)
	}
	if Timeout != 4 {
		t.Errorf("Timeout = %d, want 4", Timeout)
	}
}
