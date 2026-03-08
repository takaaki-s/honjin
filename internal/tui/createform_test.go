package tui

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/takaaki-s/claude-code-valet/internal/daemon"
	"github.com/takaaki-s/claude-code-valet/internal/session"
)

// --- hasMultipleHosts ---

func TestCreateFormModel_HasMultipleHosts(t *testing.T) {
	t.Run("empty hosts returns false", func(t *testing.T) {
		m := CreateFormModel{}
		if m.hasMultipleHosts() {
			t.Error("hasMultipleHosts() should return false for empty hosts")
		}
	})

	t.Run("single host returns false", func(t *testing.T) {
		m := CreateFormModel{
			hosts: []daemon.HostInfo{
				{ID: "local", Type: "local", Connected: true},
			},
		}
		if m.hasMultipleHosts() {
			t.Error("hasMultipleHosts() should return false for single host")
		}
	})

	t.Run("multiple hosts returns true", func(t *testing.T) {
		m := CreateFormModel{
			hosts: []daemon.HostInfo{
				{ID: "local", Type: "local", Connected: true},
				{ID: "remote-dev", Type: "ssh", Connected: true},
			},
		}
		if !m.hasMultipleHosts() {
			t.Error("hasMultipleHosts() should return true for multiple hosts")
		}
	})
}

// --- filterHosts ---

func TestCreateFormModel_FilterHosts(t *testing.T) {
	makeModel := func(hosts []daemon.HostInfo, query string) *CreateFormModel {
		hi := textinput.New()
		hi.SetValue(query)
		return &CreateFormModel{
			hosts:     hosts,
			hostInput: hi,
		}
	}

	allHosts := []daemon.HostInfo{
		{ID: "local", Type: "local", Connected: true},
		{ID: "remote-dev", Type: "ssh", Connected: true},
		{ID: "remote-staging", Type: "ssh", Connected: false},
	}

	t.Run("empty query returns all hosts", func(t *testing.T) {
		m := makeModel(allHosts, "")
		m.filterHosts()

		if len(m.filteredHosts) != 3 {
			t.Fatalf("filterHosts('') returned %d hosts, want 3", len(m.filteredHosts))
		}
		if !m.hostDropdownOpen {
			t.Error("hostDropdownOpen should be true for empty query")
		}
		if m.hostSelectedIndex != 0 {
			t.Errorf("hostSelectedIndex should be 0, got %d", m.hostSelectedIndex)
		}
	})

	t.Run("partial match case insensitive", func(t *testing.T) {
		m := makeModel(allHosts, "Remote")
		m.filterHosts()

		if len(m.filteredHosts) != 2 {
			t.Fatalf("filterHosts('Remote') returned %d hosts, want 2: %v", len(m.filteredHosts), m.filteredHosts)
		}
		for _, h := range m.filteredHosts {
			if h.ID != "remote-dev" && h.ID != "remote-staging" {
				t.Errorf("unexpected host %q in filtered results", h.ID)
			}
		}
		if !m.hostDropdownOpen {
			t.Error("hostDropdownOpen should be true when matches exist")
		}
		if m.hostSelectedIndex != 0 {
			t.Errorf("hostSelectedIndex should be 0, got %d", m.hostSelectedIndex)
		}
	})

	t.Run("no matches closes dropdown", func(t *testing.T) {
		m := makeModel(allHosts, "nonexistent")
		m.filterHosts()

		if len(m.filteredHosts) != 0 {
			t.Fatalf("filterHosts('nonexistent') returned %d hosts, want 0", len(m.filteredHosts))
		}
		if m.hostDropdownOpen {
			t.Error("hostDropdownOpen should be false when no matches")
		}
	})

	t.Run("match resets hostSelectedIndex to 0", func(t *testing.T) {
		m := makeModel(allHosts, "")
		m.hostSelectedIndex = 2
		m.filterHosts()

		if m.hostSelectedIndex != 0 {
			t.Errorf("hostSelectedIndex should be reset to 0, got %d", m.hostSelectedIndex)
		}
	})
}

// --- selectHost ---

func TestCreateFormModel_SelectHost(t *testing.T) {
	t.Run("valid index sets selectedHostID and input value", func(t *testing.T) {
		hi := textinput.New()
		m := CreateFormModel{
			hostInput:         hi,
			filteredHosts:     []daemon.HostInfo{
				{ID: "local", Type: "local", Connected: true},
				{ID: "remote-dev", Type: "ssh", Connected: true},
			},
			hostSelectedIndex: 1,
			hostDropdownOpen:  true,
		}

		m.selectHost()

		if m.selectedHostID != "remote-dev" {
			t.Errorf("selectedHostID = %q, want %q", m.selectedHostID, "remote-dev")
		}
		if m.hostInput.Value() != "remote-dev" {
			t.Errorf("hostInput.Value() = %q, want %q", m.hostInput.Value(), "remote-dev")
		}
		if m.hostDropdownOpen {
			t.Error("hostDropdownOpen should be false after selectHost()")
		}
	})

	t.Run("out of bounds index makes no change", func(t *testing.T) {
		hi := textinput.New()
		m := CreateFormModel{
			hostInput:         hi,
			selectedHostID:    "",
			filteredHosts:     []daemon.HostInfo{
				{ID: "local", Type: "local", Connected: true},
			},
			hostSelectedIndex: 5, // out of bounds
			hostDropdownOpen:  true,
		}

		m.selectHost()

		if m.selectedHostID != "" {
			t.Errorf("selectedHostID should remain empty, got %q", m.selectedHostID)
		}
		if m.hostInput.Value() != "" {
			t.Errorf("hostInput.Value() should remain empty, got %q", m.hostInput.Value())
		}
		if !m.hostDropdownOpen {
			t.Error("hostDropdownOpen should remain true when index is out of bounds")
		}
	})
}

// --- computeDirHistory (additional cases not in model_test.go) ---

func TestComputeDirHistory_TildeConversion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}
	now := time.Now()

	t.Run("local host converts home prefix to tilde in DisplayPath", func(t *testing.T) {
		sessions := []session.Info{
			{WorkDir: home + "/myproject", LastActiveAt: now},
		}

		result := computeDirHistory(sessions, "local", 10)

		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
		if result[0].Path != home+"/myproject" {
			t.Errorf("Path should remain absolute: got %q", result[0].Path)
		}
		if result[0].DisplayPath != "~/myproject" {
			t.Errorf("DisplayPath = %q, want %q", result[0].DisplayPath, "~/myproject")
		}
	})

	t.Run("remote host does not apply tilde conversion", func(t *testing.T) {
		sessions := []session.Info{
			{WorkDir: "/remote/home/project", HostID: "remote-dev", LastActiveAt: now},
		}

		result := computeDirHistory(sessions, "remote-dev", 10)

		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
		// For remote hosts, DisplayPath should match Path (no tilde conversion)
		if result[0].DisplayPath != "/remote/home/project" {
			t.Errorf("DisplayPath = %q, want %q (no tilde conversion for remote)", result[0].DisplayPath, "/remote/home/project")
		}
	})
}

func TestComputeDirHistory_EmptyHostIDNormalization(t *testing.T) {
	now := time.Now()

	t.Run("empty hostID parameter is treated as local", func(t *testing.T) {
		sessions := []session.Info{
			{WorkDir: "/home/user/proj1", HostID: "", LastActiveAt: now},
			{WorkDir: "/remote/proj", HostID: "remote-dev", LastActiveAt: now},
		}

		// Pass empty string for hostID (function normalizes to "local")
		result := computeDirHistory(sessions, "", 10)

		if len(result) != 1 {
			t.Fatalf("expected 1 local entry, got %d", len(result))
		}
		if result[0].Path != "/home/user/proj1" {
			t.Errorf("Path = %q, want %q", result[0].Path, "/home/user/proj1")
		}
	})
}
