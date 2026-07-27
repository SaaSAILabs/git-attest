package active

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseVSCodeChatFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Mock window
	now := time.Now()
	window := TimeWindow{
		Start: now.Add(-1 * time.Hour),
		End:   now.Add(1 * time.Hour),
	}

	// 1. Create a workspace chatSessions json file
	workspaceDir := filepath.Join(tempDir, "workspaceStorage", "1234", "chatSessions")
	err := os.MkdirAll(workspaceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create mock workspace dir: %v", err)
	}

	jsonPath := filepath.Join(workspaceDir, "test.json")
	mockJSON := chatSessionJSON{
		Requests: []chatRequest{
			{
				Timestamp: now.UnixMilli(),
				Message: struct {
					Text     string `json:"text"`
					ChatText string `json:"chatText"`
				}{
					Text: "hello from json",
				},
			},
		},
	}
	data, _ := json.Marshal(mockJSON)
	os.WriteFile(jsonPath, data, 0644)

	// 2. Create an empty window chatSessions jsonl file
	globalDir := filepath.Join(tempDir, "globalStorage", "emptyWindowChatSessions")
	err = os.MkdirAll(globalDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create mock global dir: %v", err)
	}

	jsonlPath := filepath.Join(globalDir, "test.jsonl")

	reqData := []chatRequest{
		{
			Timestamp: now.UnixMilli(),
			Message: struct {
				Text     string `json:"text"`
				ChatText string `json:"chatText"`
			}{
				Text: "hello from jsonl",
			},
		},
	}
	reqDataBytes, _ := json.Marshal(reqData)

	mockJSONL := chatSessionJSONL{
		Kind: 2,
		K:    []string{"requests"},
		V:    reqDataBytes,
	}
	lineBytes, _ := json.Marshal(mockJSONL)
	os.WriteFile(jsonlPath, append(lineBytes, '\n'), 0644)

	// Test extraction (repoRoot "" disables scoping; this is the parse test).
	events, err := ParseVSCodeChatFiles(tempDir, window, "copilot", "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	foundJSON := false
	foundJSONL := false

	for _, e := range events {
		prompt := e.Meta["prompt"].(string)
		if prompt == "hello from json" {
			foundJSON = true
		}
		if prompt == "hello from jsonl" {
			foundJSONL = true
		}
	}

	if !foundJSON {
		t.Errorf("Did not extract prompt from .json file")
	}
	if !foundJSONL {
		t.Errorf("Did not extract prompt from .jsonl file")
	}
}

// writeVSCodeWorkspaceChat creates a workspaceStorage/<hash>/ with a
// workspace.json pointing at folder, plus one chatSessions prompt.
func writeVSCodeWorkspaceChat(t *testing.T, base, hash, folder, prompt string, ts int64) {
	t.Helper()
	hashDir := filepath.Join(base, "workspaceStorage", hash)
	chatDir := filepath.Join(hashDir, "chatSessions")
	if err := os.MkdirAll(chatDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if folder != "" {
		ws := []byte(`{"folder": "file://` + folder + `"}`)
		if err := os.WriteFile(filepath.Join(hashDir, "workspace.json"), ws, 0644); err != nil {
			t.Fatalf("write workspace.json: %v", err)
		}
	}
	session := chatSessionJSON{Requests: []chatRequest{{
		Timestamp: ts,
		Message: struct {
			Text     string `json:"text"`
			ChatText string `json:"chatText"`
		}{Text: prompt},
	}}}
	data, _ := json.Marshal(session)
	if err := os.WriteFile(filepath.Join(chatDir, "s.json"), data, 0644); err != nil {
		t.Fatalf("write session: %v", err)
	}
}

// TestParseVSCodeChatFiles_Scoping verifies chats are attributed to the repo by
// their workspace folder: the repo (and subdirectories) are kept; other repos
// and folderless sessions are excluded.
func TestParseVSCodeChatFiles_Scoping(t *testing.T) {
	base := t.TempDir()
	repo := "/Users/me/repo"
	now := time.Now().UnixMilli()

	writeVSCodeWorkspaceChat(t, base, "hash_repo", repo, "in-repo prompt", now)
	writeVSCodeWorkspaceChat(t, base, "hash_sub", repo+"/pkg", "in-subdir prompt", now)
	writeVSCodeWorkspaceChat(t, base, "hash_other", "/Users/me/other-repo", "other-repo prompt", now)
	writeVSCodeWorkspaceChat(t, base, "hash_none", "", "folderless prompt", now)

	window := TimeWindow{Start: time.UnixMilli(now).Add(-time.Hour), End: time.UnixMilli(now).Add(time.Hour)}
	events, err := ParseVSCodeChatFiles(base, window, "copilot", repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := map[string]bool{}
	for _, e := range events {
		got[e.Meta["prompt"].(string)] = true
	}
	if !got["in-repo prompt"] || !got["in-subdir prompt"] {
		t.Errorf("expected in-repo and in-subdir prompts kept, got %v", got)
	}
	if got["other-repo prompt"] {
		t.Error("other-repo prompt leaked past scoping")
	}
	if got["folderless prompt"] {
		t.Error("folderless (unattributable) prompt leaked past scoping")
	}
}
