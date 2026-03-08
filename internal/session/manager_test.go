package session

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/takaaki-s/claude-code-valet/internal/config"
)

// newTestManager creates a Manager backed by temporary directories and a mock tmux runner.
func newTestManager(t *testing.T) (*Manager, *mockTmuxRunner) {
	t.Helper()
	dir := t.TempDir()
	configDir := t.TempDir()
	configMgr, err := config.NewManager(configDir)
	if err != nil {
		t.Fatalf("config.NewManager failed: %v", err)
	}
	mgr, err := NewManager(dir, configDir, configMgr)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	mock := newMockTmuxRunner()
	mgr.SetTmuxClient(mock)
	return mgr, mock
}

// ---------------------------------------------------------------------------
// CreateWithOptions tests
// ---------------------------------------------------------------------------

func TestManager_CreateWithOptions_Success(t *testing.T) {
	mgr, _ := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{
		WorkDir: "/tmp/project-alpha",
		Name:    "alpha",
	})
	if err != nil {
		t.Fatalf("CreateWithOptions failed: %v", err)
	}
	if sess.ID == "" {
		t.Error("expected non-empty ID")
	}
	if sess.Name != "alpha" {
		t.Errorf("Name = %q, want %q", sess.Name, "alpha")
	}
	if sess.WorkDir != "/tmp/project-alpha" {
		t.Errorf("WorkDir = %q, want %q", sess.WorkDir, "/tmp/project-alpha")
	}
	if sess.Status != StatusStopped {
		t.Errorf("Status = %q, want %q", sess.Status, StatusStopped)
	}
	if sess.ClaudeSessionID == "" {
		t.Error("expected non-empty ClaudeSessionID")
	}
}

func TestManager_CreateWithOptions_DuplicateWorkDir(t *testing.T) {
	mgr, _ := newTestManager(t)

	_, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/dup-dir", Name: "first"})
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/dup-dir", Name: "second"})
	if err == nil {
		t.Fatal("expected error for duplicate WorkDir, got nil")
	}
}

func TestManager_CreateWithOptions_DuplicateName(t *testing.T) {
	mgr, _ := newTestManager(t)

	_, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/dir-a", Name: "samename"})
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/dir-b", Name: "samename"})
	if err == nil {
		t.Fatal("expected error for duplicate Name, got nil")
	}
}

func TestManager_CreateWithOptions_EmptyWorkDir(t *testing.T) {
	mgr, _ := newTestManager(t)

	_, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "", Name: "nodir"})
	if err == nil {
		t.Fatal("expected error for empty WorkDir, got nil")
	}
}

func TestManager_CreateWithOptions_DefaultName(t *testing.T) {
	mgr, _ := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/home/user/my-project"})
	if err != nil {
		t.Fatalf("CreateWithOptions failed: %v", err)
	}
	want := filepath.Base("/home/user/my-project")
	if sess.Name != want {
		t.Errorf("Name = %q, want %q (filepath.Base of WorkDir)", sess.Name, want)
	}
}

// ---------------------------------------------------------------------------
// Get tests
// ---------------------------------------------------------------------------

func TestManager_Get_Found(t *testing.T) {
	mgr, _ := newTestManager(t)

	created, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/get-test", Name: "getme"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	got, ok := mgr.Get(created.ID)
	if !ok {
		t.Fatal("Get returned ok=false for existing session")
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
}

func TestManager_Get_NotFound(t *testing.T) {
	mgr, _ := newTestManager(t)

	_, ok := mgr.Get("nonexistent-id")
	if ok {
		t.Fatal("Get returned ok=true for nonexistent session")
	}
}

// ---------------------------------------------------------------------------
// List tests
// ---------------------------------------------------------------------------

func TestManager_List(t *testing.T) {
	mgr, _ := newTestManager(t)

	_, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/list-1", Name: "first"})
	if err != nil {
		t.Fatalf("create first failed: %v", err)
	}
	// Ensure distinct CreatedAt timestamps.
	time.Sleep(2 * time.Millisecond)
	_, err = mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/list-2", Name: "second"})
	if err != nil {
		t.Fatalf("create second failed: %v", err)
	}

	infos := mgr.List()
	if len(infos) != 2 {
		t.Fatalf("List returned %d items, want 2", len(infos))
	}
	// Sorted by CreatedAt ascending
	if infos[0].Name != "first" {
		t.Errorf("first item Name = %q, want %q", infos[0].Name, "first")
	}
	if infos[1].Name != "second" {
		t.Errorf("second item Name = %q, want %q", infos[1].Name, "second")
	}
}

// ---------------------------------------------------------------------------
// SetStatus / SetStatusWithError tests
// ---------------------------------------------------------------------------

func TestManager_SetStatus(t *testing.T) {
	mgr, _ := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/status-test", Name: "s1"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	mgr.SetStatus(sess.ID, StatusThinking)

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if got.Status != StatusThinking {
		t.Errorf("Status = %q, want %q", got.Status, StatusThinking)
	}
}

func TestManager_SetStatusWithError(t *testing.T) {
	mgr, _ := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/err-test", Name: "e1"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	mgr.SetStatusWithError(sess.ID, StatusStopped, "something went wrong")

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if got.Status != StatusStopped {
		t.Errorf("Status = %q, want %q", got.Status, StatusStopped)
	}
	if got.ErrorMessage != "something went wrong" {
		t.Errorf("ErrorMessage = %q, want %q", got.ErrorMessage, "something went wrong")
	}
}

// ---------------------------------------------------------------------------
// SetWorkDir tests
// ---------------------------------------------------------------------------

func TestManager_SetWorkDir(t *testing.T) {
	mgr, _ := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/wd-old", Name: "wd"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if err := mgr.SetWorkDir(sess.ID, "/tmp/wd-new"); err != nil {
		t.Fatalf("SetWorkDir failed: %v", err)
	}

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if got.WorkDir != "/tmp/wd-new" {
		t.Errorf("WorkDir = %q, want %q", got.WorkDir, "/tmp/wd-new")
	}
}

func TestManager_SetWorkDir_DuplicateWorkDir(t *testing.T) {
	mgr, _ := newTestManager(t)

	_, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/wd-dup", Name: "d1"})
	if err != nil {
		t.Fatalf("create first failed: %v", err)
	}
	s2, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/wd-other", Name: "d2"})
	if err != nil {
		t.Fatalf("create second failed: %v", err)
	}

	err = mgr.SetWorkDir(s2.ID, "/tmp/wd-dup")
	if err == nil {
		t.Fatal("expected error when setting WorkDir to one already in use, got nil")
	}
}

// ---------------------------------------------------------------------------
// CountActive tests
// ---------------------------------------------------------------------------

func TestManager_CountActive(t *testing.T) {
	mgr, _ := newTestManager(t)

	s1, _ := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/ca-1", Name: "ca1"})
	s2, _ := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/ca-2", Name: "ca2"})
	s3, _ := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/ca-3", Name: "ca3"})

	// All start as StatusStopped; set two to active statuses.
	mgr.SetStatus(s2.ID, StatusThinking)
	mgr.SetStatus(s3.ID, StatusRunning)
	// s1 remains StatusStopped

	_ = s1 // keep compiler happy

	count := mgr.CountActive()
	if count != 2 {
		t.Errorf("CountActive() = %d, want 2", count)
	}
}

// ---------------------------------------------------------------------------
// HandleHookEvent tests
// ---------------------------------------------------------------------------

func TestManager_HandleHookEvent_UserPromptSubmit(t *testing.T) {
	mgr, _ := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/hook-ups", Name: "hups"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	mgr.HandleHookEvent(sess.ClaudeSessionID, sess.ID, "UserPromptSubmit", "", "")

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if got.Status != StatusThinking {
		t.Errorf("Status = %q, want %q", got.Status, StatusThinking)
	}
}

func TestManager_HandleHookEvent_Stop(t *testing.T) {
	mgr, _ := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/hook-stop", Name: "hstop"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	// Set to thinking first
	mgr.SetStatus(sess.ID, StatusThinking)

	mgr.HandleHookEvent(sess.ClaudeSessionID, sess.ID, "Stop", "", "")

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if got.Status != StatusIdle {
		t.Errorf("Status = %q, want %q", got.Status, StatusIdle)
	}
}

func TestManager_HandleHookEvent_Notification_Permission(t *testing.T) {
	mgr, _ := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/hook-perm", Name: "hperm"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	mgr.HandleHookEvent(sess.ClaudeSessionID, sess.ID, "Notification", "permission_prompt", "")

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if got.Status != StatusPermission {
		t.Errorf("Status = %q, want %q", got.Status, StatusPermission)
	}
}

func TestManager_HandleHookEvent_UnknownSession(t *testing.T) {
	mgr, _ := newTestManager(t)

	// Should not panic when both IDs are unknown.
	mgr.HandleHookEvent("unknown-cc-id", "unknown-valet-id", "Stop", "", "")
}

func TestManager_HandleHookEvent_CWDUpdate(t *testing.T) {
	mgr, _ := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/hook-cwd", Name: "hcwd"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	newCwd := "/tmp/hook-cwd-changed"
	mgr.HandleHookEvent(sess.ClaudeSessionID, sess.ID, "UserPromptSubmit", "", newCwd)

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if got.WorkDir != newCwd {
		t.Errorf("WorkDir = %q, want %q", got.WorkDir, newCwd)
	}
	if got.CurrentWorkDir != newCwd {
		t.Errorf("CurrentWorkDir = %q, want %q", got.CurrentWorkDir, newCwd)
	}
}

// ---------------------------------------------------------------------------
// Kill tests
// ---------------------------------------------------------------------------

func TestManager_Kill(t *testing.T) {
	mgr, mock := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/kill-test", Name: "killme"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Simulate a running session with tmux integration.
	mgr.mu.Lock()
	sess.TmuxWindowName = "ccvalet_" + sess.ID
	sess.TmuxPaneID = "%42"
	sess.Status = StatusRunning
	mgr.mu.Unlock()

	if err := mgr.Kill(sess.ID); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false after Kill")
	}
	if got.Status != StatusStopped {
		t.Errorf("Status = %q, want %q", got.Status, StatusStopped)
	}
	if !mock.hasCalledWith("KillPane", "%42") {
		t.Error("expected KillPane to be called with %42")
	}
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

func TestManager_Delete(t *testing.T) {
	mgr, _ := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/del-test", Name: "delme"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if err := mgr.Delete(sess.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Session should no longer be accessible via Get.
	_, ok := mgr.Get(sess.ID)
	if ok {
		t.Fatal("Get returned ok=true after Delete")
	}

	// Store should also have removed the file.
	_, err = mgr.store.Load(sess.ID)
	if err == nil {
		t.Fatal("expected store.Load to return error after Delete, got nil")
	}
}

// ---------------------------------------------------------------------------
// RecoverTmuxSessions tests
// ---------------------------------------------------------------------------

func TestManager_RecoverTmuxSessions_Live(t *testing.T) {
	mgr, mock := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/recover-live", Name: "rlive"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	innerName := "ccvalet_" + sess.ID
	mgr.mu.Lock()
	sess.TmuxWindowName = innerName
	sess.TmuxPaneID = "%10"
	mgr.mu.Unlock()

	// Configure mock: session exists and pane is alive.
	mock.sessions[innerName] = true
	mock.deadPanes["%10"] = false

	mgr.RecoverTmuxSessions()

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if got.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", got.Status, StatusRunning)
	}
}

func TestManager_RecoverTmuxSessions_DeadPane(t *testing.T) {
	mgr, mock := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/recover-dead", Name: "rdead"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	innerName := "ccvalet_" + sess.ID
	mgr.mu.Lock()
	sess.TmuxWindowName = innerName
	sess.TmuxPaneID = "%11"
	mgr.mu.Unlock()

	// Configure mock: session exists but pane is dead.
	mock.sessions[innerName] = true
	mock.deadPanes["%11"] = true

	mgr.RecoverTmuxSessions()

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if got.Status != StatusStopped {
		t.Errorf("Status = %q, want %q", got.Status, StatusStopped)
	}
	// TmuxWindowName should be kept (window preserved via remain-on-exit).
	if got.TmuxWindowName == "" {
		t.Error("expected TmuxWindowName to be kept after dead pane recovery")
	}
}

func TestManager_RecoverTmuxSessions_NoTmux(t *testing.T) {
	mgr, _ := newTestManager(t)

	// Explicitly set tmuxClient to nil to simulate no tmux available.
	mgr.SetTmuxClient(nil)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/recover-notmux", Name: "rnotmux"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	mgr.mu.Lock()
	sess.TmuxWindowName = "ccvalet_" + sess.ID
	mgr.mu.Unlock()

	// Should be a no-op and not panic.
	mgr.RecoverTmuxSessions()

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	// Status should remain unchanged (StatusStopped from creation).
	if got.Status != StatusStopped {
		t.Errorf("Status = %q, want %q", got.Status, StatusStopped)
	}
}

// ---------------------------------------------------------------------------
// FindByClaudeSessionID tests
// ---------------------------------------------------------------------------

func TestManager_FindByClaudeSessionID(t *testing.T) {
	mgr, _ := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/find-cc", Name: "findcc"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Find by the ClaudeSessionID that was auto-generated during creation.
	got, ok := mgr.FindByClaudeSessionID(sess.ClaudeSessionID)
	if !ok {
		t.Fatal("FindByClaudeSessionID returned ok=false for existing session")
	}
	if got.ID != sess.ID {
		t.Errorf("ID = %q, want %q", got.ID, sess.ID)
	}

	// Find with a non-existent ClaudeSessionID should return nil.
	got2, ok2 := mgr.FindByClaudeSessionID("nonexistent-cc-id")
	if ok2 {
		t.Fatal("FindByClaudeSessionID returned ok=true for non-existent ClaudeSessionID")
	}
	if got2 != nil {
		t.Errorf("expected nil session, got %+v", got2)
	}
}

func TestManager_FindByClaudeSessionID_NotFound(t *testing.T) {
	mgr, _ := newTestManager(t)

	// Empty manager: should return nil, false.
	got, ok := mgr.FindByClaudeSessionID("does-not-exist")
	if ok {
		t.Fatal("FindByClaudeSessionID returned ok=true on empty manager")
	}
	if got != nil {
		t.Errorf("expected nil session, got %+v", got)
	}
}

// ---------------------------------------------------------------------------
// StartBackground tests
// ---------------------------------------------------------------------------

func TestManager_StartBackground(t *testing.T) {
	mgr, mock := newTestManager(t)

	// Use a real temp directory so os.Stat in startSessionTmux passes.
	workDir := t.TempDir()

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: workDir, Name: "bg"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Configure mock so GetPaneID returns a valid pane ID for the inner session.
	innerName := "sess-" + sess.ID
	mock.paneIDs[innerName] = "%99"

	if err := mgr.StartBackground(sess.ID); err != nil {
		t.Fatalf("StartBackground failed: %v", err)
	}

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false after StartBackground")
	}
	if got.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", got.Status, StatusRunning)
	}
	if got.TmuxWindowName != innerName {
		t.Errorf("TmuxWindowName = %q, want %q", got.TmuxWindowName, innerName)
	}

	// Verify mock tmux calls.
	if !mock.hasCalledWith("NewSessionWithCmdInDir", innerName) {
		t.Error("expected NewSessionWithCmdInDir to be called with inner session name")
	}
}

func TestManager_StartBackground_NotFound(t *testing.T) {
	mgr, _ := newTestManager(t)

	err := mgr.StartBackground("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for non-existent session ID, got nil")
	}
}

func TestManager_StartBackground_AlreadyRunning(t *testing.T) {
	mgr, mock := newTestManager(t)

	workDir := t.TempDir()
	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: workDir, Name: "already"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Simulate a session that's already running (has TmuxWindowName and non-stopped status).
	mgr.mu.Lock()
	sess.TmuxWindowName = "sess-" + sess.ID
	sess.Status = StatusRunning
	mgr.mu.Unlock()

	// StartBackground should succeed without creating a new tmux session.
	if err := mgr.StartBackground(sess.ID); err != nil {
		t.Fatalf("StartBackground failed: %v", err)
	}

	// NewSessionWithCmdInDir should NOT have been called.
	if mock.hasCalledWith("NewSessionWithCmdInDir", "sess-"+sess.ID) {
		t.Error("expected NewSessionWithCmdInDir NOT to be called for already running session")
	}
}

// ---------------------------------------------------------------------------
// SetStatus extended tests
// ---------------------------------------------------------------------------

func TestManager_SetStatus_NonExistent(t *testing.T) {
	mgr, _ := newTestManager(t)

	// Setting status on a non-existent session should not panic.
	mgr.SetStatus("nonexistent-id", StatusThinking)

	// Verify no sessions were created.
	infos := mgr.List()
	if len(infos) != 0 {
		t.Errorf("List returned %d items, want 0", len(infos))
	}
}

func TestManager_SetStatus_Persisted(t *testing.T) {
	mgr, _ := newTestManager(t)

	sess, err := mgr.CreateWithOptions(CreateOptions{WorkDir: "/tmp/setstatus-persist", Name: "sp"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	mgr.SetStatus(sess.ID, StatusThinking)

	got, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned ok=false")
	}
	if got.Status != StatusThinking {
		t.Errorf("Status = %q, want %q", got.Status, StatusThinking)
	}
}
