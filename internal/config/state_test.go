package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateManager_NewWithDefaults(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewStateManager(dir)
	if err != nil {
		t.Fatalf("NewStateManager: %v", err)
	}
	if sm == nil {
		t.Fatal("NewStateManager returned nil")
	}
	if sm.state == nil {
		t.Fatal("state is nil")
	}
}

func TestStateManager_SaveAndReload(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewStateManager(dir)
	if err != nil {
		t.Fatalf("NewStateManager: %v", err)
	}

	if err := sm.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file was created
	statePath := filepath.Join(dir, "state.yaml")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state.yaml not found: %v", err)
	}

	// Create new manager from the same directory
	sm2, err := NewStateManager(dir)
	if err != nil {
		t.Fatalf("NewStateManager (reload): %v", err)
	}
	if sm2.state == nil {
		t.Fatal("reloaded state is nil")
	}
}

func TestStateManager_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "state")
	sm, err := NewStateManager(dir)
	if err != nil {
		t.Fatalf("NewStateManager: %v", err)
	}
	if sm == nil {
		t.Fatal("NewStateManager returned nil")
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected directory")
	}
}
