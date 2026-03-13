package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/takaaki-s/claude-code-valet/internal/session"
)

func TestRenderWaitResultJSON(t *testing.T) {
	t.Run("outputs session info on wait completion", func(t *testing.T) {
		info := &session.Info{
			ID:        "abc-123",
			Name:      "my-session",
			Status:    session.StatusIdle,
			WorkDir:   "/home/user/project",
			CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		var buf bytes.Buffer
		if err := renderWaitResultJSON(&buf, info); err != nil {
			t.Fatalf("renderWaitResultJSON() error = %v", err)
		}
		var parsed session.Info
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
		}
		if parsed.Status != session.StatusIdle {
			t.Errorf("expected status %q, got %q", session.StatusIdle, parsed.Status)
		}
	})
}
