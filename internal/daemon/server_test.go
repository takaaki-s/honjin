package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceEnv_Replace(t *testing.T) {
	env := []string{"FOO=old", "BAR=baz"}
	got := replaceEnv(env, "FOO", "new")

	found := false
	for _, e := range got {
		if e == "FOO=new" {
			found = true
		}
		if e == "FOO=old" {
			t.Error("replaceEnv should have replaced FOO=old, but it still exists")
		}
	}
	if !found {
		t.Errorf("replaceEnv did not set FOO=new, got %v", got)
	}
	// BAR should remain unchanged
	barFound := false
	for _, e := range got {
		if e == "BAR=baz" {
			barFound = true
		}
	}
	if !barFound {
		t.Errorf("replaceEnv should preserve other entries, BAR=baz missing in %v", got)
	}
}

func TestReplaceEnv_Append(t *testing.T) {
	env := []string{"BAR=1"}
	got := replaceEnv(env, "FOO", "new")

	if len(got) != 2 {
		t.Fatalf("replaceEnv should append, got len %d: %v", len(got), got)
	}

	// BAR=1 should still be there
	if got[0] != "BAR=1" {
		t.Errorf("first element should be BAR=1, got %q", got[0])
	}
	// FOO=new should be appended
	if got[1] != "FOO=new" {
		t.Errorf("appended element should be FOO=new, got %q", got[1])
	}
}

func TestReplaceEnv_Empty(t *testing.T) {
	var env []string
	got := replaceEnv(env, "FOO", "bar")

	if len(got) != 1 {
		t.Fatalf("replaceEnv on empty env should return 1 element, got %d: %v", len(got), got)
	}
	if got[0] != "FOO=bar" {
		t.Errorf("replaceEnv on empty env should return [FOO=bar], got %v", got)
	}
}

func TestReplaceEnv_PrefixCollision(t *testing.T) {
	// Ensure "FOOBAR=x" is not matched when replacing "FOO"
	env := []string{"FOOBAR=x", "FOO=old"}
	got := replaceEnv(env, "FOO", "new")

	foobarFound := false
	fooNewFound := false
	for _, e := range got {
		if e == "FOOBAR=x" {
			foobarFound = true
		}
		if e == "FOO=new" {
			fooNewFound = true
		}
	}
	if !foobarFound {
		t.Errorf("FOOBAR=x should not be modified, got %v", got)
	}
	if !fooNewFound {
		t.Errorf("FOO=new should be present, got %v", got)
	}
}

func TestDebugLog_Disabled(t *testing.T) {
	// Save and restore original values
	origEnabled := debugEnabled
	origPath := debugLogPath
	defer func() {
		debugEnabled = origEnabled
		debugLogPath = origPath
	}()

	dir := t.TempDir()
	logFile := filepath.Join(dir, "debug.log")

	debugEnabled = false
	debugLogPath = logFile

	debugLog("this message should not appear")

	// Verify file was NOT created
	if _, err := os.Stat(logFile); err == nil {
		t.Error("debugLog created a file even though debugEnabled=false")
	}
}

func TestDebugLog_Enabled(t *testing.T) {
	// Save and restore original values
	origEnabled := debugEnabled
	origPath := debugLogPath
	defer func() {
		debugEnabled = origEnabled
		debugLogPath = origPath
	}()

	dir := t.TempDir()
	logFile := filepath.Join(dir, "debug.log")

	debugEnabled = true
	debugLogPath = logFile

	debugLog("hello %s %d", "world", 42)

	// Verify file was created and contains the message
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "hello world 42") {
		t.Errorf("log file content %q does not contain expected message", content)
	}
	// Verify timestamp format [HH:MM:SS] is present
	if !strings.Contains(content, "[") || !strings.Contains(content, "]") {
		t.Errorf("log file content %q does not contain timestamp brackets", content)
	}
}

func TestDebugLog_EmptyPath(t *testing.T) {
	// Save and restore original values
	origEnabled := debugEnabled
	origPath := debugLogPath
	defer func() {
		debugEnabled = origEnabled
		debugLogPath = origPath
	}()

	debugEnabled = true
	debugLogPath = ""

	// Should not panic
	debugLog("this should be a no-op with empty path")
}
