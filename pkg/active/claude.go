package active

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SaaSAILabs/attest-cli.git/pkg/util"
)

// ClaudeLogRecord is a targeted struct for unmarshalling Claude Code JSONL lines.
// It intentionally drops large assistant/tool payloads to optimize memory.
type ClaudeLogRecord struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Message   *struct {
		Role    string      `json:"role"`
		Content interface{} `json:"content"`
	} `json:"message,omitempty"`
}

// FlightEvent represents a single human-intent event extracted from a session log.
type FlightEvent struct {
	Timestamp int64                  `json:"timestamp"`
	Type      string                 `json:"type"`
	Meta      map[string]interface{} `json:"meta"`
}

// FindRelevantSessions locates .jsonl files under ~/.claude/projects/*/sessions/
// that overlap with the given TimeWindow.
func FindRelevantSessions(window TimeWindow) ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to resolve home directory: %w", err)
	}

	pattern := filepath.Join(home, ".claude", "projects", "*", "sessions", "*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob error: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no claude session files found matching %s", pattern)
	}

	var relevant []string
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		
		stamps := util.GetFileTimestamps(info)
		birth := stamps.Btime
		if birth.IsZero() {
			birth = time.Time{}
		}
		
		if window.Overlaps(birth, stamps.Mtime) {
			relevant = append(relevant, path)
		}
	}

	if len(relevant) == 0 {
		return nil, fmt.Errorf("no overlapping session files found")
	}
	return relevant, nil
}

// ParseSessionFile reads a Claude Code .jsonl file line-by-line and extracts
// user prompts as FlightEvent structs. Malformed lines are silently skipped.
func ParseSessionFile(path string) ([]FlightEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open session file: %w", err)
	}
	defer f.Close()

	var events []FlightEvent
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Bytes()

		var record ClaudeLogRecord
		if err := json.Unmarshal(line, &record); err != nil {
			continue // skip malformed lines
		}

		if record.Type != "user" || record.Message == nil {
			continue
		}

		prompt := extractPromptText(record.Message.Content)
		if prompt == "" {
			continue
		}

		events = append(events, FlightEvent{
			Timestamp: record.Timestamp.UnixMilli(),
			Type:      "agent_prompt",
			Meta: map[string]interface{}{
				"source": "claude_code",
				"prompt": prompt,
			},
		})
	}

	if err := scanner.Err(); err != nil {
		return events, fmt.Errorf("scanner error: %w", err)
	}
	return events, nil
}

// extractPromptText handles both plain string content and multimodal
// []interface{} arrays (where each block has "type":"text" and "text":"...").
func extractPromptText(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, block := range v {
			m, ok := block.(map[string]interface{})
			if !ok {
				continue
			}
			if m["type"] == "text" {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}
