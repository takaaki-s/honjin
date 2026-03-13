package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRenderActionResultJSON(t *testing.T) {
	t.Run("kill success", func(t *testing.T) {
		result := actionResult{
			Success: true,
			ID:      "abc-123",
			Name:    "my-session",
		}
		var buf bytes.Buffer
		if err := renderActionResultJSON(&buf, result); err != nil {
			t.Fatalf("renderActionResultJSON() error = %v", err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
		}
		if parsed["success"] != true {
			t.Errorf("expected success=true, got %v", parsed["success"])
		}
		if parsed["id"] != "abc-123" {
			t.Errorf("expected id %q, got %v", "abc-123", parsed["id"])
		}
		if parsed["name"] != "my-session" {
			t.Errorf("expected name %q, got %v", "my-session", parsed["name"])
		}
	})

	t.Run("delete success", func(t *testing.T) {
		result := actionResult{
			Success: true,
			ID:      "def-456",
			Name:    "other-session",
		}
		var buf bytes.Buffer
		if err := renderActionResultJSON(&buf, result); err != nil {
			t.Fatalf("renderActionResultJSON() error = %v", err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
			t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
		}
		if parsed["success"] != true {
			t.Errorf("expected success=true, got %v", parsed["success"])
		}
		if parsed["name"] != "other-session" {
			t.Errorf("expected name %q, got %v", "other-session", parsed["name"])
		}
	})
}
