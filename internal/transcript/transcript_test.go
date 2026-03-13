package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- TruncateMessage ---

func TestTruncateMessage_WithinLimit(t *testing.T) {
	got := TruncateMessage("hello", 10)
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}

func TestTruncateMessage_ExactBoundary(t *testing.T) {
	got := TruncateMessage("hello", 5)
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}

func TestTruncateMessage_Truncated(t *testing.T) {
	got := TruncateMessage("hello world", 8)
	// maxLen=8, so first 5 chars + "..." = "hello..."
	want := "hello..."
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestTruncateMessage_VeryShortMax(t *testing.T) {
	// maxLen <= 3 returns first maxLen chars without "..."
	got := TruncateMessage("hello", 3)
	want := "hel"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}

	got2 := TruncateMessage("hello", 1)
	want2 := "h"
	if got2 != want2 {
		t.Errorf("expected %q, got %q", want2, got2)
	}
}

// --- TruncateMessageFromEnd ---

func TestTruncateMessageFromEnd_WithinLimit(t *testing.T) {
	got := TruncateMessageFromEnd("hello", 10)
	if got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}

func TestTruncateMessageFromEnd_Truncated(t *testing.T) {
	got := TruncateMessageFromEnd("hello world", 8)
	// "..." + last 5 chars = "...world"
	want := "...world"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestTruncateMessageFromEnd_VeryShortMax(t *testing.T) {
	// maxLen <= 3 returns last maxLen chars without "..."
	got := TruncateMessageFromEnd("hello", 3)
	want := "llo"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}

	got2 := TruncateMessageFromEnd("hello", 1)
	want2 := "o"
	if got2 != want2 {
		t.Errorf("expected %q, got %q", want2, got2)
	}
}

// --- encodePathForClaude ---

func TestEncodePathForClaude(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"/Users/foo/bar", "-Users-foo-bar"},
		{"/home/user/project", "-home-user-project"},
		{"relative/path", "relative-path"},
		{"/", "-"},
	}
	for _, tc := range cases {
		got := encodePathForClaude(tc.input)
		if got != tc.want {
			t.Errorf("encodePathForClaude(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- cleanContent ---

func TestCleanContent(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"newlines replaced", "hello\nworld", "hello world"},
		{"tabs replaced", "hello\tworld", "hello world"},
		{"carriage return removed", "hello\rworld", "helloworld"},
		{"multiple spaces collapsed", "hello    world", "hello world"},
		{"trimming", "  hello  ", "hello"},
		{"combined", " hello\n\tworld  foo  ", "hello world foo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cleanContent(tc.input)
			if got != tc.want {
				t.Errorf("cleanContent(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// --- extractContent ---

func TestExtractContent_UserString(t *testing.T) {
	entry := &transcriptEntry{
		Type: "user",
		Message: msgObject{
			Role:    "user",
			Content: "hello world",
		},
	}
	got := extractContent(entry)
	if got != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", got)
	}
}

func TestExtractContent_AssistantBlocks(t *testing.T) {
	// Simulate what json.Unmarshal produces for []contentBlock
	blocks := []any{
		map[string]any{"type": "text", "text": "first"},
		map[string]any{"type": "text", "text": "second"},
	}
	entry := &transcriptEntry{
		Type: "assistant",
		Message: msgObject{
			Role:    "assistant",
			Content: blocks,
		},
	}
	got := extractContent(entry)
	want := "first second"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestExtractContent_NilContent(t *testing.T) {
	entry := &transcriptEntry{
		Type: "user",
		Message: msgObject{
			Role:    "user",
			Content: nil,
		},
	}
	got := extractContent(entry)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// --- Reader ---

// writeJSONL writes JSONL entries to a file.
func writeJSONL(t *testing.T, path string, entries []transcriptEntry) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReader_GetTranscriptPath(t *testing.T) {
	r := &Reader{claudeDir: "/home/user/.claude"}
	got := r.getTranscriptPath("/Users/foo/bar", "abc-123")
	want := filepath.Join("/home/user/.claude", "projects", "-Users-foo-bar", "abc-123.jsonl")
	if got != want {
		t.Errorf("getTranscriptPath = %q, want %q", got, want)
	}
}

func TestReader_ReadLastMessage(t *testing.T) {
	tmpDir := t.TempDir()
	r := &Reader{claudeDir: tmpDir}

	workDir := "/test/project"
	sessionID := "sess-001"

	transcriptPath := r.getTranscriptPath(workDir, sessionID)

	entries := []transcriptEntry{
		{
			Type:      "user",
			Message:   msgObject{Role: "user", Content: "first question"},
			Timestamp: "2024-01-01T00:00:00Z",
		},
		{
			Type: "assistant",
			Message: msgObject{
				Role: "assistant",
				Content: []any{
					map[string]any{"type": "text", "text": "first answer"},
				},
			},
			Timestamp: "2024-01-01T00:00:01Z",
		},
		{
			Type:      "user",
			Message:   msgObject{Role: "user", Content: "second question"},
			Timestamp: "2024-01-01T00:00:02Z",
		},
	}
	writeJSONL(t, transcriptPath, entries)

	msg, err := r.readLastMessage(transcriptPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	// Last message overall is the second user message
	if msg.Type != "user" {
		t.Errorf("expected type %q, got %q", "user", msg.Type)
	}
	if msg.Content != "second question" {
		t.Errorf("expected content %q, got %q", "second question", msg.Content)
	}
	if msg.Timestamp != "2024-01-01T00:00:02Z" {
		t.Errorf("expected timestamp %q, got %q", "2024-01-01T00:00:02Z", msg.Timestamp)
	}
}

func TestReader_ReadLastMessages(t *testing.T) {
	tmpDir := t.TempDir()
	r := &Reader{claudeDir: tmpDir}

	workDir := "/test/project"
	sessionID := "sess-002"
	transcriptPath := r.getTranscriptPath(workDir, sessionID)

	entries := []transcriptEntry{
		{
			Type:      "user",
			Message:   msgObject{Role: "user", Content: "hello"},
			Timestamp: "2024-01-01T00:00:00Z",
		},
		{
			Type: "assistant",
			Message: msgObject{
				Role: "assistant",
				Content: []any{
					map[string]any{"type": "text", "text": "world"},
				},
			},
			Timestamp: "2024-01-01T00:00:01Z",
		},
		{
			Type:      "user",
			Message:   msgObject{Role: "user", Content: "follow up"},
			Timestamp: "2024-01-01T00:00:02Z",
		},
		{
			Type: "assistant",
			Message: msgObject{
				Role: "assistant",
				Content: []any{
					map[string]any{"type": "text", "text": "final response"},
				},
			},
			Timestamp: "2024-01-01T00:00:03Z",
		},
	}
	writeJSONL(t, transcriptPath, entries)

	msgs, err := r.readLastMessages(transcriptPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgs == nil {
		t.Fatal("expected non-nil LastMessages")
	}

	if msgs.User == nil {
		t.Fatal("expected non-nil User message")
	}
	if msgs.User.Content != "follow up" {
		t.Errorf("User.Content = %q, want %q", msgs.User.Content, "follow up")
	}

	if msgs.Assistant == nil {
		t.Fatal("expected non-nil Assistant message")
	}
	if msgs.Assistant.Content != "final response" {
		t.Errorf("Assistant.Content = %q, want %q", msgs.Assistant.Content, "final response")
	}
}

func TestReader_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	r := &Reader{claudeDir: tmpDir}

	msg, err := r.readLastMessage(filepath.Join(tmpDir, "nonexistent.jsonl"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if msg != nil {
		t.Errorf("expected nil message, got %+v", msg)
	}

	msgs, err := r.readLastMessages(filepath.Join(tmpDir, "nonexistent.jsonl"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if msgs != nil {
		t.Errorf("expected nil LastMessages, got %+v", msgs)
	}
}

func TestReader_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	r := &Reader{claudeDir: tmpDir}

	emptyFile := filepath.Join(tmpDir, "empty.jsonl")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	msg, err := r.readLastMessage(emptyFile)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if msg != nil {
		t.Errorf("expected nil message for empty file, got %+v", msg)
	}

	msgs, err := r.readLastMessages(emptyFile)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if msgs != nil {
		t.Errorf("expected nil LastMessages for empty file, got %+v", msgs)
	}
}

func TestReader_EmptySessionID(t *testing.T) {
	tmpDir := t.TempDir()
	r := &Reader{claudeDir: tmpDir}

	msg, err := r.GetLastMessage("/some/dir", "")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if msg != nil {
		t.Errorf("expected nil message for empty sessionID, got %+v", msg)
	}

	msgs, err := r.GetLastMessages("/some/dir", "")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if msgs != nil {
		t.Errorf("expected nil LastMessages for empty sessionID, got %+v", msgs)
	}
}

// --- Additional edge cases ---

func TestExtractContent_AssistantNonTextBlock(t *testing.T) {
	// Blocks that are not type "text" should be ignored
	blocks := []any{
		map[string]any{"type": "tool_use", "name": "read_file"},
		map[string]any{"type": "text", "text": "only text"},
	}
	entry := &transcriptEntry{
		Type: "assistant",
		Message: msgObject{
			Role:    "assistant",
			Content: blocks,
		},
	}
	got := extractContent(entry)
	if got != "only text" {
		t.Errorf("expected %q, got %q", "only text", got)
	}
}

func TestReader_GetLastMessage_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	r := &Reader{claudeDir: tmpDir}

	workDir := "/integration/test"
	sessionID := "sess-int"
	transcriptPath := r.getTranscriptPath(workDir, sessionID)

	entries := []transcriptEntry{
		{
			Type:      "user",
			Message:   msgObject{Role: "user", Content: "the question"},
			Timestamp: "2024-06-01T12:00:00Z",
		},
	}
	writeJSONL(t, transcriptPath, entries)

	msg, err := r.GetLastMessage(workDir, sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Content != "the question" {
		t.Errorf("Content = %q, want %q", msg.Content, "the question")
	}
}

func TestCleanContent_CarriageReturnNewline(t *testing.T) {
	got := cleanContent("line1\r\nline2")
	// \r is removed (replaced with ""), \n is replaced with space
	if !strings.Contains(got, "line1") || !strings.Contains(got, "line2") {
		t.Errorf("expected both lines, got %q", got)
	}
	if strings.ContainsAny(got, "\r\n") {
		t.Errorf("expected no CR/LF, got %q", got)
	}
}

func TestGetConversation(t *testing.T) {
	tmpDir := t.TempDir()
	r := &Reader{claudeDir: tmpDir}

	workDir := "/test/project"
	sessionID := "sess-conv"
	transcriptPath := r.getTranscriptPath(workDir, sessionID)

	entries := []transcriptEntry{
		{
			Type:      "user",
			Message:   msgObject{Role: "user", Content: "first question"},
			Timestamp: "2024-01-01T00:00:00Z",
		},
		{
			Type: "assistant",
			Message: msgObject{
				Role:    "assistant",
				Content: []any{map[string]any{"type": "text", "text": "first answer"}},
			},
			Timestamp: "2024-01-01T00:00:01Z",
		},
		{
			Type:      "user",
			Message:   msgObject{Role: "user", Content: "second question"},
			Timestamp: "2024-01-01T00:00:02Z",
		},
		{
			Type: "assistant",
			Message: msgObject{
				Role:    "assistant",
				Content: []any{map[string]any{"type": "text", "text": "second answer"}},
			},
			Timestamp: "2024-01-01T00:00:03Z",
		},
		{
			Type:      "user",
			Message:   msgObject{Role: "user", Content: "third question"},
			Timestamp: "2024-01-01T00:00:04Z",
		},
		{
			Type: "assistant",
			Message: msgObject{
				Role:    "assistant",
				Content: []any{map[string]any{"type": "text", "text": "third answer"}},
			},
			Timestamp: "2024-01-01T00:00:05Z",
		},
	}
	writeJSONL(t, transcriptPath, entries)

	t.Run("last 1 pair", func(t *testing.T) {
		msgs, err := r.GetConversation(workDir, sessionID, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msgs) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(msgs))
		}
		if msgs[0].Type != "user" || msgs[0].Content != "third question" {
			t.Errorf("unexpected first message: %+v", msgs[0])
		}
		if msgs[1].Type != "assistant" || msgs[1].Content != "third answer" {
			t.Errorf("unexpected second message: %+v", msgs[1])
		}
	})

	t.Run("last 2 pairs", func(t *testing.T) {
		msgs, err := r.GetConversation(workDir, sessionID, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msgs) != 4 {
			t.Fatalf("expected 4 messages, got %d", len(msgs))
		}
		if msgs[0].Content != "second question" {
			t.Errorf("expected %q, got %q", "second question", msgs[0].Content)
		}
	})

	t.Run("last N exceeds total", func(t *testing.T) {
		msgs, err := r.GetConversation(workDir, sessionID, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msgs) != 6 {
			t.Fatalf("expected 6 messages, got %d", len(msgs))
		}
	})

	t.Run("empty session ID", func(t *testing.T) {
		msgs, err := r.GetConversation(workDir, "", 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msgs != nil {
			t.Errorf("expected nil, got %v", msgs)
		}
	})
}

func TestExtractFullContent_PreservesNewlines(t *testing.T) {
	entry := &transcriptEntry{
		Type:    "user",
		Message: msgObject{Role: "user", Content: "line1\nline2\nline3"},
	}
	got := extractFullContent(entry)
	if !strings.Contains(got, "\n") {
		t.Errorf("expected newlines preserved, got %q", got)
	}
}
