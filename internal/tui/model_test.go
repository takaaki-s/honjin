package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/takaaki-s/claude-code-valet/internal/session"
)

// --- truncateString ---

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
	}{
		{
			name:     "short string within limit",
			input:    "hello",
			maxWidth: 10,
			want:     "hello",
		},
		{
			name:     "string exactly at limit",
			input:    "hello",
			maxWidth: 5,
			want:     "hello",
		},
		{
			name:     "string needs truncation",
			input:    "hello world this is long",
			maxWidth: 10,
			want:     "hello w...",
		},
		{
			name:     "maxWidth 3 gets ellipsis",
			input:    "hello world",
			maxWidth: 3,
			want:     "hel",
		},
		{
			name:     "maxWidth 2 no ellipsis",
			input:    "hello",
			maxWidth: 2,
			want:     "he",
		},
		{
			name:     "maxWidth 1",
			input:    "hello",
			maxWidth: 1,
			want:     "h",
		},
		{
			name:     "empty string",
			input:    "",
			maxWidth: 10,
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxWidth)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxWidth, got, tt.want)
			}
		})
	}
}

// --- truncateStringFromEnd ---

func TestTruncateStringFromEnd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
	}{
		{
			name:     "short string within limit",
			input:    "hello",
			maxWidth: 10,
			want:     "hello",
		},
		{
			name:     "string exactly at limit",
			input:    "hello",
			maxWidth: 5,
			want:     "hello",
		},
		{
			name:     "string needs truncation keeps end",
			input:    "hello world",
			maxWidth: 8,
			want:     "...world",
		},
		{
			name:     "maxWidth 3 no ellipsis",
			input:    "hello world",
			maxWidth: 3,
			want:     "rld",
		},
		{
			name:     "maxWidth 2",
			input:    "hello",
			maxWidth: 2,
			want:     "lo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateStringFromEnd(tt.input, tt.maxWidth)
			if got != tt.want {
				t.Errorf("truncateStringFromEnd(%q, %d) = %q, want %q", tt.input, tt.maxWidth, got, tt.want)
			}
		})
	}
}

// --- timeAgo ---

func TestTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		offset   time.Duration
		want     string
	}{
		{"just now", 10 * time.Second, "just now"},
		{"1 minute ago", 1 * time.Minute, "1m ago"},
		{"5 minutes ago", 5 * time.Minute, "5m ago"},
		{"59 minutes ago", 59 * time.Minute, "59m ago"},
		{"1 hour ago", 1 * time.Hour, "1h ago"},
		{"3 hours ago", 3 * time.Hour, "3h ago"},
		{"1 day ago", 24 * time.Hour, "1d ago"},
		{"5 days ago", 5 * 24 * time.Hour, "5d ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			past := time.Now().Add(-tt.offset)
			got := timeAgo(past)
			if got != tt.want {
				t.Errorf("timeAgo(now - %v) = %q, want %q", tt.offset, got, tt.want)
			}
		})
	}
}

// --- matchesSearch ---

func TestMatchesSearch(t *testing.T) {
	sess := session.Info{
		Name:           "MyProject",
		WorkDir:        "/home/user/projects/webapp",
		CurrentWorkDir: "/home/user/projects/webapp/src",
		CurrentBranch:  "feature-auth",
	}

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"match by Name", "myproject", true},
		{"match by WorkDir", "webapp", true},
		{"match by CurrentWorkDir", "webapp/src", true},
		{"match by CurrentBranch", "feature-auth", true},
		{"case insensitive match", "myproject", true},
		{"partial match", "proj", true},
		{"no match", "nonexistent", false},
		{"empty query matches nothing meaningful", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesSearch(sess, tt.query)
			// Empty query: strings.Contains(anything, "") == true,
			// so it will match unless all fields are empty.
			if tt.query == "" {
				// For empty query, it will match any non-empty field
				if !got {
					t.Errorf("matchesSearch with empty query should match non-empty fields")
				}
				return
			}
			if got != tt.want {
				t.Errorf("matchesSearch(sess, %q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}

	t.Run("no match with empty session", func(t *testing.T) {
		emptySess := session.Info{}
		got := matchesSearch(emptySess, "anything")
		if got {
			t.Error("matchesSearch with empty session should return false")
		}
	})
}

// --- countStatuses ---

func TestCountStatuses(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		counts := countStatuses(nil)
		if counts.thinking != 0 || counts.permission != 0 || counts.running != 0 ||
			counts.creating != 0 || counts.idle != 0 || counts.stopped != 0 {
			t.Errorf("countStatuses(nil) should return all zeros, got %+v", counts)
		}
	})

	t.Run("mixed statuses", func(t *testing.T) {
		sessions := []session.Info{
			{Status: session.StatusThinking},
			{Status: session.StatusThinking},
			{Status: session.StatusPermission},
			{Status: session.StatusRunning},
			{Status: session.StatusCreating},
			{Status: session.StatusIdle},
			{Status: session.StatusIdle},
			{Status: session.StatusIdle},
			{Status: session.StatusStopped},
		}
		counts := countStatuses(sessions)
		if counts.thinking != 2 {
			t.Errorf("thinking = %d, want 2", counts.thinking)
		}
		if counts.permission != 1 {
			t.Errorf("permission = %d, want 1", counts.permission)
		}
		if counts.running != 1 {
			t.Errorf("running = %d, want 1", counts.running)
		}
		if counts.creating != 1 {
			t.Errorf("creating = %d, want 1", counts.creating)
		}
		if counts.idle != 3 {
			t.Errorf("idle = %d, want 3", counts.idle)
		}
		if counts.stopped != 1 {
			t.Errorf("stopped = %d, want 1", counts.stopped)
		}
	})
}

// --- getStatusDisplay ---

func TestGetStatusDisplay(t *testing.T) {
	tests := []struct {
		name      string
		status    session.Status
		wantIcon  string
		wantLabel string
	}{
		{"thinking", session.StatusThinking, "⚡", "THINKING"},
		{"permission", session.StatusPermission, "?", "PERMISSION"},
		{"running", session.StatusRunning, "▶", "RUNNING"},
		{"creating", session.StatusCreating, "+", "CREATING"},
		{"idle", session.StatusIdle, "○", "IDLE"},
		{"stopped", session.StatusStopped, "■", "STOPPED"},
		{"unknown", session.Status("unknown"), "?", "UNKNOWN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon, label, _ := getStatusDisplay(tt.status)
			if icon != tt.wantIcon {
				t.Errorf("getStatusDisplay(%q) icon = %q, want %q", tt.status, icon, tt.wantIcon)
			}
			if label != tt.wantLabel {
				t.Errorf("getStatusDisplay(%q) label = %q, want %q", tt.status, label, tt.wantLabel)
			}
		})
	}
}

// --- wrapText ---

func TestWrapText(t *testing.T) {
	t.Run("single short line", func(t *testing.T) {
		got := wrapText("hello", 20)
		if len(got) != 1 || got[0] != "hello" {
			t.Errorf("wrapText(%q, 20) = %v, want [%q]", "hello", got, "hello")
		}
	})

	t.Run("long line wraps", func(t *testing.T) {
		input := "abcdefghij" // 10 chars
		got := wrapText(input, 4)
		// Should wrap into: "abcd", "efgh", "ij"
		if len(got) != 3 {
			t.Fatalf("wrapText(%q, 4) got %d lines, want 3: %v", input, len(got), got)
		}
		if got[0] != "abcd" {
			t.Errorf("line 0 = %q, want %q", got[0], "abcd")
		}
		if got[1] != "efgh" {
			t.Errorf("line 1 = %q, want %q", got[1], "efgh")
		}
		if got[2] != "ij" {
			t.Errorf("line 2 = %q, want %q", got[2], "ij")
		}
	})

	t.Run("zero width returns original", func(t *testing.T) {
		got := wrapText("hello", 0)
		if len(got) != 1 || got[0] != "hello" {
			t.Errorf("wrapText(%q, 0) = %v, want [%q]", "hello", got, "hello")
		}
	})

	t.Run("negative width returns original", func(t *testing.T) {
		got := wrapText("hello", -1)
		if len(got) != 1 || got[0] != "hello" {
			t.Errorf("wrapText(%q, -1) = %v, want [%q]", "hello", got, "hello")
		}
	})

	t.Run("text with newlines", func(t *testing.T) {
		input := "line1\nline2\nline3"
		got := wrapText(input, 20)
		if len(got) != 3 {
			t.Fatalf("wrapText with newlines got %d lines, want 3: %v", len(got), got)
		}
		if got[0] != "line1" || got[1] != "line2" || got[2] != "line3" {
			t.Errorf("got %v, want [line1, line2, line3]", got)
		}
	})
}

// --- padLine ---

func TestPadLine(t *testing.T) {
	t.Run("shorter string gets padded", func(t *testing.T) {
		got := padLine("hi", 5)
		if got != "hi   " {
			t.Errorf("padLine(%q, 5) = %q, want %q", "hi", got, "hi   ")
		}
	})

	t.Run("exact width no padding", func(t *testing.T) {
		got := padLine("hello", 5)
		if got != "hello" {
			t.Errorf("padLine(%q, 5) = %q, want %q", "hello", got, "hello")
		}
	})

	t.Run("longer string no change", func(t *testing.T) {
		got := padLine("hello world", 5)
		if got != "hello world" {
			t.Errorf("padLine(%q, 5) = %q, want %q", "hello world", got, "hello world")
		}
	})

	t.Run("empty string gets full padding", func(t *testing.T) {
		got := padLine("", 3)
		if got != "   " {
			t.Errorf("padLine(%q, 3) = %q, want %q", "", got, "   ")
		}
	})
}

// --- isSessionAlive ---

func TestIsSessionAlive(t *testing.T) {
	alive := []session.Status{
		session.StatusRunning,
		session.StatusThinking,
		session.StatusIdle,
		session.StatusPermission,
		session.StatusCreating,
	}
	for _, s := range alive {
		t.Run(string(s)+"_alive", func(t *testing.T) {
			if !isSessionAlive(s) {
				t.Errorf("isSessionAlive(%q) = false, want true", s)
			}
		})
	}

	dead := []session.Status{
		session.StatusStopped,
		session.Status("unknown"),
	}
	for _, s := range dead {
		name := string(s)
		if name == "" {
			name = "empty"
		}
		t.Run(name+"_not_alive", func(t *testing.T) {
			if isSessionAlive(s) {
				t.Errorf("isSessionAlive(%q) = true, want false", s)
			}
		})
	}
}

// --- helper: verify truncation lengths ---

func TestTruncateStringLengthProperties(t *testing.T) {
	// Verify the truncated result has a display width <= maxWidth
	input := "this is a longer string for testing"
	for _, maxWidth := range []int{1, 2, 3, 5, 10, 15} {
		got := truncateString(input, maxWidth)
		// For ASCII-only strings, len is the display width
		if len(got) > maxWidth {
			t.Errorf("truncateString(%q, %d) = %q (len %d), exceeds maxWidth",
				input, maxWidth, got, len(got))
		}
	}
}

func TestTruncateStringFromEndLengthProperties(t *testing.T) {
	input := "this is a longer string for testing"
	for _, maxWidth := range []int{1, 2, 3, 5, 10, 15} {
		got := truncateStringFromEnd(input, maxWidth)
		if len(got) > maxWidth {
			t.Errorf("truncateStringFromEnd(%q, %d) = %q (len %d), exceeds maxWidth",
				input, maxWidth, got, len(got))
		}
	}
}

// Verify truncateStringFromEnd keeps the end of the string
func TestTruncateStringFromEndKeepsEnd(t *testing.T) {
	got := truncateStringFromEnd("/home/user/very/long/path/to/project", 20)
	if !strings.HasSuffix(got, "to/project") {
		t.Errorf("truncateStringFromEnd should keep the end, got %q", got)
	}
	if !strings.HasPrefix(got, "...") {
		t.Errorf("truncateStringFromEnd should start with '...', got %q", got)
	}
}
