package active

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SaaSAILabs/git-attest/pkg/util"
)

// codexRecord is one line of a Codex rollout log. Payload stays raw so we only
// pay for unmarshalling the record types we actually care about.
type codexRecord struct {
	Timestamp time.Time       `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// codexSessionMeta is the payload of the leading session_meta record.
type codexSessionMeta struct {
	Cwd        string `json:"cwd"`
	Originator string `json:"originator"`
}

// codexPayload covers the event_msg payloads we read.
type codexPayload struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// codexNoisePrefixes mark user_message records that Codex generates rather than
// the human typing them: interrupt markers, slash-command expansion and its
// stdout, compaction summaries, and the synthetic history fed to the approval
// assessor. Attesting these as human intent would be a false provenance claim.
var codexNoisePrefixes = []string{
	"[Request interrupted by user]",
	"<command-name>",
	"<local-command-stdout>",
	"This session is being continued from a previous",
	"The following is the Codex agent history",
}

// codexImagePlaceholder stands in for an attached image. It can lead an
// otherwise real prompt, so it is stripped per-line rather than used to drop
// the whole message.
const codexImagePlaceholder = "[external unsupported block: image]"

// codexReplayWindow collapses fan-out replicas. A multi-agent turn writes the
// same prompt into several rollout files milliseconds apart, each with a fresh
// timestamp, so identical text this close together is one human action.
// Measured on real logs: any window from 1s to 60s collapses the same set.
const codexReplayWindow = time.Second

// FindRelevantCodexSessions locates rollout logs under
// ~/.codex/sessions/YYYY/MM/DD/ that overlap with the given TimeWindow.
func FindRelevantCodexSessions(window TimeWindow) ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to resolve home directory: %w", err)
	}

	pattern := filepath.Join(home, ".codex", "sessions", "*", "*", "*", "*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob error: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no codex session files found matching %s", pattern)
	}

	var relevant []string
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		stamps := util.GetFileTimestamps(info)
		if window.Overlaps(stamps.Btime, stamps.Mtime) {
			relevant = append(relevant, path)
		}
	}

	if len(relevant) == 0 {
		return nil, fmt.Errorf("no overlapping codex session files found")
	}
	return relevant, nil
}

// ParseCodexSessionFile extracts human prompts from a Codex rollout log.
// Sessions are skipped entirely when their originator is not Codex or when
// their cwd falls outside repoRoot; an empty repoRoot disables repo scoping.
func ParseCodexSessionFile(path string, repoRoot string) ([]FlightEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open codex session file: %w", err)
	}
	defer f.Close()

	var events []FlightEvent
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		var record codexRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue // skip malformed lines
		}

		if record.Type == "session_meta" {
			var meta codexSessionMeta
			if err := json.Unmarshal(record.Payload, &meta); err != nil {
				return nil, nil
			}
			if !strings.Contains(strings.ToLower(meta.Originator), "codex") {
				return nil, nil // e.g. Claude Cowork, harvested by its own extractor
			}
			if !withinRepo(meta.Cwd, repoRoot) {
				return nil, nil
			}
			continue
		}

		if record.Type != "event_msg" {
			continue
		}

		var payload codexPayload
		if err := json.Unmarshal(record.Payload, &payload); err != nil {
			continue
		}
		if payload.Type != "user_message" {
			continue
		}

		prompt := cleanCodexMessage(payload.Message)
		if prompt == "" {
			continue
		}

		events = append(events, FlightEvent{
			Timestamp: record.Timestamp.UnixMilli(),
			Type:      "agent_prompt",
			Meta: map[string]interface{}{
				"source": "codex",
				"prompt": prompt,
			},
		})
	}

	if err := scanner.Err(); err != nil {
		return events, fmt.Errorf("scanner error: %w", err)
	}
	return events, nil
}

// cleanCodexMessage strips image placeholders and returns "" for any message
// that is machine-generated rather than typed by the human.
func cleanCodexMessage(message string) string {
	var kept []string
	for _, line := range strings.Split(message, "\n") {
		if strings.TrimSpace(line) == codexImagePlaceholder {
			continue
		}
		kept = append(kept, line)
	}

	body := strings.TrimSpace(strings.Join(kept, "\n"))
	for _, prefix := range codexNoisePrefixes {
		if strings.HasPrefix(body, prefix) {
			return ""
		}
	}
	return body
}

// dedupeReplayedPrompts drops repeats of the same text within
// codexReplayWindow, which is how multi-agent fan-out duplicates a single turn.
// Genuine repeats typed further apart are preserved. Events must already be
// sorted chronologically.
func dedupeReplayedPrompts(events []FlightEvent) []FlightEvent {
	lastSeen := make(map[string]int64, len(events))
	windowMillis := codexReplayWindow.Milliseconds()

	var unique []FlightEvent
	for _, event := range events {
		prompt, _ := event.Meta["prompt"].(string)
		if seen, ok := lastSeen[prompt]; ok && event.Timestamp-seen <= windowMillis {
			continue
		}
		lastSeen[prompt] = event.Timestamp
		unique = append(unique, event)
	}
	return unique
}

// --- Codex Extractor ---

type CodexExtractor struct{}

func (c *CodexExtractor) Name() string { return "codex" }

func (c *CodexExtractor) Extract(window TimeWindow) ([]FlightEvent, error) {
	repoRoot, err := currentRepoRoot()
	if err != nil {
		return nil, err
	}

	paths, err := FindRelevantCodexSessions(window)
	if err != nil {
		return nil, err
	}

	var allEvents []FlightEvent
	for _, path := range paths {
		events, err := ParseCodexSessionFile(path, repoRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[attest] warning: failed to parse %s: %v\n", path, err)
			continue
		}
		allEvents = append(allEvents, events...)
	}

	sortEventsByTimestamp(allEvents)
	return dedupeReplayedPrompts(allEvents), nil
}
