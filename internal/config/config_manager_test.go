package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigManager_NewWithDefaults(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	cfg := m.Get()
	if cfg == nil {
		t.Fatal("Get() returned nil")
	}

	// Default config should have no hosts
	if len(cfg.Hosts) != 0 {
		t.Errorf("default hosts: got %d, want 0", len(cfg.Hosts))
	}
}

func TestConfigManager_SaveAndReload(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	// Modify config via viper and save
	m.v.Set("hosts", []map[string]any{
		{"id": "ec2", "type": "ssh", "host": "ec2-host"},
	})
	if err := m.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err != nil {
		t.Fatalf("config.yaml not found: %v", err)
	}

	// Create new manager from the same directory
	m2, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager (reload): %v", err)
	}

	hosts := m2.GetHosts()
	if len(hosts) != 1 {
		t.Fatalf("hosts after reload: got %d, want 1", len(hosts))
	}
	if hosts[0].ID != "ec2" {
		t.Errorf("host ID: got %q, want %q", hosts[0].ID, "ec2")
	}
}

func TestConfigManager_GetHost_Found(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	m.mu.Lock()
	m.config.Hosts = []HostConfig{
		{ID: "ec2", Type: "ssh", Host: "ec2-host"},
		{ID: "docker-dev", Type: "docker", Container: "my-container"},
	}
	m.mu.Unlock()

	h := m.GetHost("docker-dev")
	if h == nil {
		t.Fatal("GetHost returned nil for existing host")
	}
	if h.Type != "docker" {
		t.Errorf("Type: got %q, want %q", h.Type, "docker")
	}
	if h.Container != "my-container" {
		t.Errorf("Container: got %q, want %q", h.Container, "my-container")
	}
}

func TestConfigManager_GetHost_NotFound(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	h := m.GetHost("nonexistent")
	if h != nil {
		t.Errorf("GetHost: expected nil for nonexistent host, got %+v", h)
	}
}

func TestConfigManager_GetKeybindings_Defaults(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	kb := m.GetKeybindings()
	defaults := DefaultKeybindings()

	if len(kb.Up) != len(defaults.Up) || kb.Up[0] != defaults.Up[0] {
		t.Errorf("Up: got %v, want %v", kb.Up, defaults.Up)
	}
	if len(kb.Down) != len(defaults.Down) || kb.Down[0] != defaults.Down[0] {
		t.Errorf("Down: got %v, want %v", kb.Down, defaults.Down)
	}
	if len(kb.Detach) != len(defaults.Detach) || kb.Detach[0] != defaults.Detach[0] {
		t.Errorf("Detach: got %v, want %v", kb.Detach, defaults.Detach)
	}
}

func TestConfigManager_GetKeybindings_PartialOverride(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	// Override only Up, everything else should use defaults
	m.mu.Lock()
	m.config.Keybindings.Up = []string{"w"}
	m.mu.Unlock()

	kb := m.GetKeybindings()

	if len(kb.Up) != 1 || kb.Up[0] != "w" {
		t.Errorf("Up: got %v, want [w]", kb.Up)
	}
	// Down should be default
	defaults := DefaultKeybindings()
	if len(kb.Down) != len(defaults.Down) {
		t.Errorf("Down should use defaults: got %v", kb.Down)
	}
}

func TestConfigManager_GetDetachKey_Default(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	b := m.GetDetachKey()
	if b != 0x1d { // ctrl+]
		t.Errorf("GetDetachKey: got 0x%02x, want 0x1d", b)
	}

	hint := m.GetDetachKeyHint()
	if hint != "Ctrl+]" {
		t.Errorf("GetDetachKeyHint: got %q, want %q", hint, "Ctrl+]")
	}

	tmuxKey := m.GetDetachKeyTmux()
	if tmuxKey != "C-]" {
		t.Errorf("GetDetachKeyTmux: got %q, want %q", tmuxKey, "C-]")
	}
}

func TestConfigManager_GetDetachKey_Configured(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	m.mu.Lock()
	m.config.Keybindings.Detach = []string{"ctrl+g"}
	m.mu.Unlock()

	b := m.GetDetachKey()
	if b != 0x07 {
		t.Errorf("GetDetachKey: got 0x%02x, want 0x07", b)
	}

	hint := m.GetDetachKeyHint()
	if hint != "Ctrl+G" {
		t.Errorf("GetDetachKeyHint: got %q, want %q", hint, "Ctrl+G")
	}

	csi := m.GetDetachKeyCSIu()
	expected := []byte("\x1b[103;5u")
	if string(csi) != string(expected) {
		t.Errorf("GetDetachKeyCSIu: got %v, want %v", csi, expected)
	}
}

func TestConfigManager_GetShell(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	shell := m.GetShell()
	if shell == "" {
		t.Error("GetShell returned empty string")
	}
}
