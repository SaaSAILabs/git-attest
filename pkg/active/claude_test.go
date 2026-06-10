package active

import (
	"os"
	"path/filepath"
	"testing"
)

const mockSessionPayload = `{"type": "system", "uuid": "sys-1", "timestamp": "2025-02-20T09:10:00.000Z"}
{"type": "user", "uuid": "usr-1", "timestamp": "2025-02-20T09:14:28.000Z", "message": {"role": "user", "content": "Update the auth logic"}}
{"type": "assistant", "uuid": "ast-1", "timestamp": "2025-02-20T09:14:30.000Z", "message": {"role": "assistant", "content": [{"type": "text", "text": "Thinking..."}]}}
{"type": "user", "uuid": "usr-2", "timestamp": "2025-02-20T09:15:00.000Z", "message": {"role": "user", "content": [{"type": "text", "text": "Fix this error too"}]}}
{"corrupted": "json line that should not crash the parser`

func TestParseSessionFile(t *testing.T) {
	dir := t.TempDir()
	mockFile := filepath.Join(dir, "mock_session.jsonl")
	if err := os.WriteFile(mockFile, []byte(mockSessionPayload), 0644); err != nil {
		t.Fatalf("failed to write mock fixture: %v", err)
	}

	events, err := ParseSessionFile(mockFile)
	if err != nil {
		t.Fatalf("ParseSessionFile returned error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// First event: plain string prompt.
	if events[0].Type != "agent_prompt" {
		t.Errorf("event[0].Type = %q, want %q", events[0].Type, "agent_prompt")
	}
	if events[0].Meta["prompt"] != "Update the auth logic" {
		t.Errorf("event[0] prompt = %q, want %q", events[0].Meta["prompt"], "Update the auth logic")
	}
	if events[0].Meta["source"] != "claude_code" {
		t.Errorf("event[0] source = %q, want %q", events[0].Meta["source"], "claude_code")
	}

	// Second event: multimodal array prompt.
	if events[1].Type != "agent_prompt" {
		t.Errorf("event[1].Type = %q, want %q", events[1].Type, "agent_prompt")
	}
	if events[1].Meta["prompt"] != "Fix this error too" {
		t.Errorf("event[1] prompt = %q, want %q", events[1].Meta["prompt"], "Fix this error too")
	}
	if events[1].Meta["source"] != "claude_code" {
		t.Errorf("event[1] source = %q, want %q", events[1].Meta["source"], "claude_code")
	}

	// Verify timestamps are non-zero Unix millis.
	if events[0].Timestamp == 0 {
		t.Error("event[0].Timestamp should be non-zero")
	}
	if events[1].Timestamp == 0 {
		t.Error("event[1].Timestamp should be non-zero")
	}
}

func TestParseSessionFile_FileNotFound(t *testing.T) {
	_, err := ParseSessionFile("/nonexistent/path.jsonl")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestParseSessionFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(emptyFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write empty fixture: %v", err)
	}

	events, err := ParseSessionFile(emptyFile)
	if err != nil {
		t.Fatalf("ParseSessionFile returned error on empty file: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events from empty file, got %d", len(events))
	}
}
