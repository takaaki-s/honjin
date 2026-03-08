package host

import (
	"strings"
	"testing"

	"github.com/takaaki-s/claude-code-valet/internal/config"
)

func TestSSHControlPath(t *testing.T) {
	got := SSHControlPath("ec2")
	want := "/tmp/ccvalet-ssh-ctrl-ec2"
	if got != want {
		t.Errorf("SSHControlPath(%q) = %q, want %q", "ec2", got, want)
	}
}

func TestSSHControlPath_DifferentHost(t *testing.T) {
	got := SSHControlPath("docker-dev")
	want := "/tmp/ccvalet-ssh-ctrl-docker-dev"
	if got != want {
		t.Errorf("SSHControlPath(%q) = %q, want %q", "docker-dev", got, want)
	}
}

func TestAttachCommandString_SSH(t *testing.T) {
	cfg := config.HostConfig{
		ID:      "ec2",
		Type:    "ssh",
		Host:    "ec2-host",
		SSHOpts: []string{"-p", "2222"},
	}
	target := "my-session"
	got := AttachCommandString(cfg, target)

	// Verify it starts with ssh command
	if !strings.HasPrefix(got, "ssh ") {
		t.Errorf("SSH command should start with 'ssh ', got %q", got)
	}

	// Verify ControlMaster=no is present
	if !strings.Contains(got, "ControlMaster=no") {
		t.Errorf("SSH command should contain ControlMaster=no, got %q", got)
	}

	// Verify ControlPath references the right socket
	expectedCtrlPath := SSHControlPath("ec2")
	if !strings.Contains(got, "ControlPath="+expectedCtrlPath) {
		t.Errorf("SSH command should contain ControlPath=%s, got %q", expectedCtrlPath, got)
	}

	// Verify ClearAllForwardings=yes
	if !strings.Contains(got, "ClearAllForwardings=yes") {
		t.Errorf("SSH command should contain ClearAllForwardings=yes, got %q", got)
	}

	// Verify SSH opts are included
	if !strings.Contains(got, "-p 2222") {
		t.Errorf("SSH command should contain SSHOpts '-p 2222', got %q", got)
	}

	// Verify -t and host
	if !strings.Contains(got, "-t ec2-host") {
		t.Errorf("SSH command should contain '-t ec2-host', got %q", got)
	}

	// Verify remote tmux command is quoted
	expectedRemote := "tmux -L ccvalet attach -t my-session"
	if !strings.Contains(got, "'"+expectedRemote+"'") {
		t.Errorf("SSH command should contain quoted remote command '%s', got %q", expectedRemote, got)
	}
}

func TestAttachCommandString_Docker(t *testing.T) {
	cfg := config.HostConfig{
		ID:        "dev",
		Type:      "docker",
		Container: "my-container",
	}
	target := "my-session"
	got := AttachCommandString(cfg, target)

	want := "docker exec -it my-container tmux -L ccvalet attach -t my-session"
	if got != want {
		t.Errorf("AttachCommandString(docker) = %q, want %q", got, want)
	}
}

func TestAttachCommandString_Local(t *testing.T) {
	cfg := config.HostConfig{
		ID: "local",
		// Type is empty, so default branch is used
	}
	target := "my-session"
	got := AttachCommandString(cfg, target)

	want := "tmux -L ccvalet attach -t my-session"
	if got != want {
		t.Errorf("AttachCommandString(local) = %q, want %q", got, want)
	}
}

func TestAttachCommandString_SSHNoOpts(t *testing.T) {
	cfg := config.HostConfig{
		ID:   "simple",
		Type: "ssh",
		Host: "simple-host",
	}
	target := "sess1"
	got := AttachCommandString(cfg, target)

	// Should not have extra spaces from empty SSHOpts
	if !strings.Contains(got, "-t simple-host") {
		t.Errorf("SSH command without opts should have '-t simple-host', got %q", got)
	}

	expectedRemote := "tmux -L ccvalet attach -t sess1"
	if !strings.Contains(got, "'"+expectedRemote+"'") {
		t.Errorf("SSH command should contain quoted remote command, got %q", got)
	}
}
