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

// chatSessionJSON represents the schema for a workspace chatSession .json file.
type chatSessionJSON struct {
	Requests []chatRequest `json:"requests"`
}

// chatSessionJSONL represents the schema for a single line in a .jsonl file.
type chatSessionJSONL struct {
	Kind int             `json:"kind"`
	K    []string        `json:"k"`
	V    json.RawMessage `json:"v"`
}

// chatRequest represents an individual prompt/response pair inside the JSON.
type chatRequest struct {
	Timestamp int64 `json:"timestamp"`
	Message   struct {
		Text     string `json:"text"`
		ChatText string `json:"chatText"` // Sometimes Cursor uses chatText or something else, but we will check Text first
	} `json:"message"`
	Result struct {
		Metadata struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		} `json:"metadata"`
	} `json:"result"`
}

// findChatSessionFiles recursively searches the basePath for .json and .jsonl files
// within the chatSessions and emptyWindowChatSessions directories, returning those
// modified within the time window.
func findChatSessionFiles(basePath string, window TimeWindow) ([]string, error) {
	var files []string

	err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't access
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Only look at .json and .jsonl files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".json" && ext != ".jsonl" {
			return nil
		}

		// Only look inside directories named "chatSessions" or "emptyWindowChatSessions"
		parentDir := filepath.Base(filepath.Dir(path))
		if parentDir != "chatSessions" && parentDir != "emptyWindowChatSessions" {
			return nil
		}

		// Check mod time
		info, err := d.Info()
		if err != nil {
			return nil
		}

		// If the file was modified before the window start, we can still parse it
		// because the session might have started earlier but the last modified time
		// could be within the window. But to be safe, let's just parse it if its ModTime
		// is after the window start.
		if info.ModTime().After(window.Start) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// extractText from a chatRequest tries to find the prompt text
func extractText(req chatRequest) string {
	if req.Message.Text != "" {
		return req.Message.Text
	}
	if req.Message.ChatText != "" {
		return req.Message.ChatText
	}
	// Fallback to result metadata messages
	for _, msg := range req.Result.Metadata.Messages {
		if msg.Role == "user" && msg.Content != "" {
			return msg.Content
		}
	}
	return ""
}

// parseChatJSON reads a .json file and extracts valid events within the TimeWindow
func parseChatJSON(path string, window TimeWindow, daemon string) ([]FlightEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session chatSessionJSON
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return extractEvents(session.Requests, window, daemon, path), nil
}

// parseChatJSONL reads a .jsonl file and extracts valid events within the TimeWindow
func parseChatJSONL(path string, window TimeWindow, daemon string) ([]FlightEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var events []FlightEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		var entry chatSessionJSONL
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		// We are looking for kind == 2 (append) and k containing "requests"
		isRequests := false
		for _, key := range entry.K {
			if key == "requests" {
				isRequests = true
				break
			}
		}

		if entry.Kind == 2 && isRequests && len(entry.V) > 0 {
			var requests []chatRequest
			if err := json.Unmarshal(entry.V, &requests); err == nil {
				events = append(events, extractEvents(requests, window, daemon, path)...)
			}
		}
	}

	return events, nil
}

func extractEvents(requests []chatRequest, window TimeWindow, daemon string, sourceFile string) []FlightEvent {
	var events []FlightEvent
	for _, req := range requests {
		if req.Timestamp == 0 {
			continue
		}

		ts := time.UnixMilli(req.Timestamp)
		if ts.Before(window.Start) || ts.After(window.End) {
			continue
		}

		text := extractText(req)
		if text == "" {
			continue
		}

		events = append(events, FlightEvent{
			Timestamp: req.Timestamp,
			Type:      "prompt",
			Meta: map[string]interface{}{
				"daemon": daemon,
				"prompt": text,
				"source": sourceFile,
			},
		})
	}
	return events
}

// ParseVSCodeChatFiles discovers and parses all relevant .json and .jsonl files in basePath
func ParseVSCodeChatFiles(basePath string, window TimeWindow, daemon string) ([]FlightEvent, error) {
	files, err := findChatSessionFiles(basePath, window)
	if err != nil {
		return nil, err
	}

	var allEvents []FlightEvent
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file))
		var events []FlightEvent
		var err error

		if ext == ".json" {
			events, err = parseChatJSON(file, window, daemon)
		} else if ext == ".jsonl" {
			events, err = parseChatJSONL(file, window, daemon)
		}

		if err == nil && len(events) > 0 {
			allEvents = append(allEvents, events...)
		}
	}

	return allEvents, nil
}
