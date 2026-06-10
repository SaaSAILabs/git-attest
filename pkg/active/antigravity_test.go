package active

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func init() {
	loadTestEnv()
}

func loadTestEnv() {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	f, err := os.Open(filepath.Join(root, ".env.test"))
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		os.Setenv(key, val)
	}
}

func TestAntigravityExtractor_MockJSONL(t *testing.T) {
	// Create a temporary JSONL file with valid Antigravity log format
	dir := t.TempDir()
	mockFile := filepath.Join(dir, "transcript.jsonl")
	
	payload := `{"type":"SYSTEM","content":"system message"}
{"type":"USER_INPUT","created_at":"2026-06-08T21:36:35Z","content":"<USER_REQUEST>\n[Ticket 06] Implementation: The Extractor Registry\n</USER_REQUEST>"}
{"type":"USER_INPUT","created_at":"invalid_time","content":"<USER_REQUEST>\nbad time\n</USER_REQUEST>"}
{"type":"USER_INPUT","created_at":"2026-06-08T21:40:00Z","content":"Plain text prompt without tags"}`

	err := os.WriteFile(mockFile, []byte(payload), 0644)
	if err != nil {
		t.Fatalf("failed to create mock file: %v", err)
	}

	window := TimeWindow{
		Start: time.Time{},
		End:   time.Now().Add(24 * time.Hour),
	}

	events, err := parseAntigravityJSONL(mockFile, window)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// First event should be the tagged prompt
	if events[0].Meta["prompt"] != "[Ticket 06] Implementation: The Extractor Registry" {
		t.Errorf("expected tagged prompt, got %q", events[0].Meta["prompt"])
	}

	// Second event should be the untagged plain prompt
	if events[1].Meta["prompt"] != "Plain text prompt without tags" {
		t.Errorf("expected plain prompt, got %q", events[1].Meta["prompt"])
	}
}

func TestExtractAntigravityPrompt(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "With Tags",
			input:    "<USER_REQUEST>\nHello world\n</USER_REQUEST>\n<ADDITIONAL_METADATA>\nmeta\n</ADDITIONAL_METADATA>",
			expected: "Hello world",
		},
		{
			name:     "Without Tags",
			input:    "Just a normal prompt",
			expected: "Just a normal prompt",
		},
		{
			name:     "Empty",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractAntigravityPrompt(tc.input)
			if result != tc.expected {
				t.Errorf("got %q, want %q", result, tc.expected)
			}
		})
	}
}
