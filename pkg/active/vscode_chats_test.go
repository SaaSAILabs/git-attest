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

	// Test extraction
	events, err := ParseVSCodeChatFiles(tempDir, window, "copilot")
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
