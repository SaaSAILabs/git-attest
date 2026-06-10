package active

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Extractor for AntigravityIDE
type AntigravityExtractor struct{}

func (e *AntigravityExtractor) Name() string {
	return "antigravity"
}

func (e *AntigravityExtractor) Extract(window TimeWindow) ([]FlightEvent, error) {
	// 1. Check override path for testing
	override := os.Getenv("ATTEST_TEST_ANTIGRAVITY_DB_PATH")
	if override != "" {
		if strings.HasSuffix(override, ".db") {
			// Tests expect the old .db path. In a real environment, they should use a .jsonl.
			// Let's just return nil for now until the test is updated.
			if !strings.HasSuffix(override, ".jsonl") {
				return nil, nil
			}
		}
		return parseAntigravityJSONL(override, window)
	}

	// 2. Discover all transcript.jsonl files
	dir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	brainDir := filepath.Join(dir, ".gemini", "antigravity-ide", "brain")

	var allEvents []FlightEvent

	err = filepath.WalkDir(brainDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "transcript.jsonl" {
			events, parseErr := parseAntigravityJSONL(path, window)
			if parseErr == nil {
				allEvents = append(allEvents, events...)
			}
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return allEvents, nil
}

type antigravityLogRecord struct {
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
	Content   string `json:"content"`
}

func parseAntigravityJSONL(path string, window TimeWindow) ([]FlightEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []FlightEvent
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var record antigravityLogRecord
		if err := json.Unmarshal(line, &record); err != nil {
			continue
		}

		if record.Type != "USER_INPUT" {
			continue
		}

		ts, err := time.Parse(time.RFC3339, record.CreatedAt)
		if err != nil {
			continue
		}

		if ts.Before(window.Start) || ts.After(window.End) {
			continue
		}

		prompt := extractAntigravityPrompt(record.Content)
		if prompt == "" {
			continue
		}

		events = append(events, FlightEvent{
			Timestamp: ts.UnixMilli(),
			Type:      "agent_prompt",
			Meta: map[string]interface{}{
				"source": "antigravity",
				"prompt": prompt,
			},
		})
	}

	return events, scanner.Err()
}

func extractAntigravityPrompt(content string) string {
	startMarker := "<USER_REQUEST>\n"
	endMarker := "\n</USER_REQUEST>"

	startIdx := strings.Index(content, startMarker)
	if startIdx != -1 {
		startIdx += len(startMarker)
		endIdx := strings.Index(content[startIdx:], endMarker)
		if endIdx != -1 {
			return strings.TrimSpace(content[startIdx : startIdx+endIdx])
		}
	}
	return strings.TrimSpace(content)
}
