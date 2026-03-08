package daemon

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
)

// setupTestServer creates a Server and Client connected via a Unix socket in t.TempDir().
// The server runs in a background goroutine and is stopped on test cleanup.
// We manage the listener directly to avoid data races in Server.Start()/Stop().
func setupTestServer(t *testing.T) (*Server, *Client) {
	t.Helper()

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")
	dataDir := filepath.Join(tmpDir, "sessions")
	configDir := filepath.Join(tmpDir, "config")

	server, err := NewServer(socketPath, dataDir, configDir)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	// Pre-create the listener ourselves so we own it and avoid races.
	os.Remove(socketPath)
	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	// Use a local variable for the accept loop to avoid racing on server.listener.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed
			}
			go server.handleConnection(conn)
		}
	}()

	client := NewClient(socketPath)

	t.Cleanup(func() {
		listener.Close()
		<-done // wait for accept loop to exit
		os.Remove(socketPath)
	})

	return server, client
}

func TestIntegration_ClientIsRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, client := setupTestServer(t)

	if !client.IsRunning() {
		t.Error("client.IsRunning() = false, want true")
	}
}

func TestIntegration_NewAndList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, client := setupTestServer(t)

	// Create a session (without starting it, since tmux isn't available)
	info, err := client.NewWithOptions(NewOptions{
		Name:    "test-session",
		WorkDir: "/tmp/test-project",
		Start:   false,
	})
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}
	if info.Name != "test-session" {
		t.Errorf("Name: got %q, want %q", info.Name, "test-session")
	}
	if info.WorkDir != "/tmp/test-project" {
		t.Errorf("WorkDir: got %q, want %q", info.WorkDir, "/tmp/test-project")
	}
	if info.ID == "" {
		t.Error("ID is empty")
	}

	// List sessions
	sessions, err := client.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("List: got %d sessions, want 1", len(sessions))
	}
	if sessions[0].Name != "test-session" {
		t.Errorf("Listed session Name: got %q, want %q", sessions[0].Name, "test-session")
	}
}

func TestIntegration_CreateMultipleAndList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, client := setupTestServer(t)

	for _, name := range []string{"alpha", "beta", "gamma"} {
		_, err := client.NewWithOptions(NewOptions{
			Name:    name,
			WorkDir: "/tmp/" + name,
			Start:   false,
		})
		if err != nil {
			t.Fatalf("NewWithOptions(%s): %v", name, err)
		}
	}

	sessions, err := client.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("List: got %d, want 3", len(sessions))
	}
}

func TestIntegration_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, client := setupTestServer(t)

	info, err := client.NewWithOptions(NewOptions{
		Name:    "to-delete",
		WorkDir: "/tmp/to-delete",
		Start:   false,
	})
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}

	if err := client.Delete(info.ID, ""); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	sessions, err := client.List()
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("List after delete: got %d, want 0", len(sessions))
	}
}

func TestIntegration_HookEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, client := setupTestServer(t)

	info, err := client.NewWithOptions(NewOptions{
		Name:    "hook-test",
		WorkDir: "/tmp/hook-test",
		Start:   false,
	})
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}

	// Send a hook event
	err = client.SendHook(HookRequest{
		SessionID:        info.ClaudeSessionID,
		CcvaletSessionID: info.ID,
		HookEventName:    "UserPromptSubmit",
	})
	if err != nil {
		t.Fatalf("SendHook: %v", err)
	}

	// Verify status changed
	sessions, err := client.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("List: got %d, want 1", len(sessions))
	}
	if string(sessions[0].Status) != "thinking" {
		t.Errorf("Status after hook: got %q, want %q", sessions[0].Status, "thinking")
	}
}

func TestIntegration_NotificationHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, client := setupTestServer(t)

	entries, err := client.NotificationHistory()
	if err != nil {
		t.Fatalf("NotificationHistory: %v", err)
	}
	// Empty history at start
	if len(entries) != 0 {
		t.Errorf("NotificationHistory: got %d entries, want 0", len(entries))
	}
}

func TestIntegration_UnknownAction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, client := setupTestServer(t)

	// Send an unknown action directly
	req := Request{Action: "nonexistent"}
	resp, err := client.send(req)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if resp.Success {
		t.Error("expected Success=false for unknown action")
	}
	if resp.Error == "" {
		t.Error("expected non-empty error for unknown action")
	}
}

func TestIntegration_DuplicateWorkDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, client := setupTestServer(t)

	_, err := client.NewWithOptions(NewOptions{
		Name:    "first",
		WorkDir: "/tmp/same-dir",
		Start:   false,
	})
	if err != nil {
		t.Fatalf("first NewWithOptions: %v", err)
	}

	_, err = client.NewWithOptions(NewOptions{
		Name:    "second",
		WorkDir: "/tmp/same-dir",
		Start:   false,
	})
	if err == nil {
		t.Error("expected error for duplicate WorkDir")
	}
}

func TestIntegration_ListHosts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, client := setupTestServer(t)

	hosts, err := client.ListHosts()
	if err != nil {
		t.Fatalf("ListHosts: %v", err)
	}
	// Without remote hosts configured, should return empty or just local
	_ = hosts // No assertion on count since it depends on configuration
}

func TestIntegration_HookStopTriggersIdle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, client := setupTestServer(t)

	info, err := client.NewWithOptions(NewOptions{
		Name:    "stop-test",
		WorkDir: "/tmp/stop-test",
		Start:   false,
	})
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}

	// First make it "thinking"
	if err := client.SendHook(HookRequest{
		CcvaletSessionID: info.ID,
		HookEventName:    "UserPromptSubmit",
	}); err != nil {
		t.Fatalf("SendHook(UserPromptSubmit): %v", err)
	}

	// Then send "Stop" to transition to idle
	if err := client.SendHook(HookRequest{
		CcvaletSessionID: info.ID,
		HookEventName:    "Stop",
	}); err != nil {
		t.Fatalf("SendHook(Stop): %v", err)
	}

	sessions, err := client.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("List: got %d, want 1", len(sessions))
	}
	if string(sessions[0].Status) != "idle" {
		t.Errorf("Status: got %q, want %q", sessions[0].Status, "idle")
	}
}

func TestIntegration_HookPermission(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, client := setupTestServer(t)

	info, err := client.NewWithOptions(NewOptions{
		Name:    "perm-test",
		WorkDir: "/tmp/perm-test",
		Start:   false,
	})
	if err != nil {
		t.Fatalf("NewWithOptions: %v", err)
	}

	if err := client.SendHook(HookRequest{
		CcvaletSessionID: info.ID,
		HookEventName:    "Notification",
		NotificationType: "permission_prompt",
	}); err != nil {
		t.Fatalf("SendHook: %v", err)
	}

	sessions, err := client.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if string(sessions[0].Status) != "permission" {
		t.Errorf("Status: got %q, want %q", sessions[0].Status, "permission")
	}
}

// Verify Request/Response JSON serialization
func TestRequestResponse_JSON(t *testing.T) {
	req := Request{
		Action: "new",
		Data:   json.RawMessage(`{"name":"test","work_dir":"/tmp"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal request: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal request: %v", err)
	}
	if decoded.Action != "new" {
		t.Errorf("Action: got %q, want %q", decoded.Action, "new")
	}
}
