package active

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// codexSession builds a rollout log with the given session_meta plus one
// user_message per message, stamped a few milliseconds apart.
func codexSession(cwd, originator string, messages ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `{"timestamp":"2026-07-16T18:56:18.447Z","type":"session_meta","payload":{"cwd":%q,"originator":%q}}`+"\n", cwd, originator)
	for i, m := range messages {
		fmt.Fprintf(&b, `{"timestamp":"2026-07-16T18:56:%02d.000Z","type":"event_msg","payload":{"type":"user_message","message":%q}}`+"\n", 20+i, m)
	}
	b.WriteString(`{"corrupted": "json line that should not crash the parser`)
	return b.String()
}

func writeCodexFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture %s: %v", name, err)
	}
	return path
}

func TestParseCodexSessionFile(t *testing.T) {
	dir := t.TempDir()
	path := writeCodexFile(t, dir, "rollout.jsonl", codexSession(
		"/repo", "Codex Desktop",
		"Update the auth logic",
		"[external unsupported block: image]\n\nWhat does this screenshot show?",
	))

	events, err := ParseCodexSessionFile(path, "/repo")
	if err != nil {
		t.Fatalf("ParseCodexSessionFile returned error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Meta["prompt"] != "Update the auth logic" {
		t.Errorf("event[0] prompt = %q", events[0].Meta["prompt"])
	}
	if events[0].Meta["source"] != "codex" {
		t.Errorf("event[0] source = %q, want %q", events[0].Meta["source"], "codex")
	}
	if events[0].Type != "agent_prompt" {
		t.Errorf("event[0].Type = %q, want %q", events[0].Type, "agent_prompt")
	}
	if events[0].Timestamp == 0 {
		t.Error("event[0].Timestamp should be non-zero")
	}
	// The image placeholder is stripped but the real question survives.
	if events[1].Meta["prompt"] != "What does this screenshot show?" {
		t.Errorf("event[1] prompt = %q", events[1].Meta["prompt"])
	}
}

// TestParseCodexSessionFile_MachineGeneratedNoise guards the core provenance
// claim: text Codex generated must never be attested as human intent.
func TestParseCodexSessionFile_MachineGeneratedNoise(t *testing.T) {
	noise := []string{
		"[Request interrupted by user]",
		"<command-name>/model</command-name>",
		"<local-command-stdout>Set model to opus</local-command-stdout>",
		"This session is being continued from a previous conversation...",
		"The following is the Codex agent history whose request action you are assessing.",
		"[external unsupported block: image]",
		"   ",
	}
	dir := t.TempDir()
	path := writeCodexFile(t, dir, "rollout.jsonl", codexSession("/repo", "Codex Desktop", noise...))

	events, err := ParseCodexSessionFile(path, "/repo")
	if err != nil {
		t.Fatalf("ParseCodexSessionFile returned error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected all noise filtered, got %d: %v", len(events), events)
	}
}

func TestParseCodexSessionFile_Scoping(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name       string
		cwd        string
		originator string
		repoRoot   string
		want       int
	}{
		{"same repo", "/repo", "Codex Desktop", "/repo", 1},
		{"subdirectory of repo", "/repo/pkg/active", "Codex Desktop", "/repo", 1},
		{"different repo", "/other", "Codex Desktop", "/repo", 0},
		{"sibling path sharing a prefix", "/repo-other", "Codex Desktop", "/repo", 0},
		{"non-codex originator", "/repo", "Claude Cowork", "/repo", 0},
		{"scoping disabled", "/anywhere", "Codex Desktop", "", 1},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeCodexFile(t, dir, fmt.Sprintf("rollout-%d.jsonl", i),
				codexSession(tc.cwd, tc.originator, "Update the auth logic"))

			events, err := ParseCodexSessionFile(path, tc.repoRoot)
			if err != nil {
				t.Fatalf("ParseCodexSessionFile returned error: %v", err)
			}
			if len(events) != tc.want {
				t.Errorf("got %d events, want %d", len(events), tc.want)
			}
		})
	}
}

// TestDedupeReplayedPrompts pins the fan-out behaviour measured on real logs:
// multi-agent turns write the same prompt to several rollout files milliseconds
// apart with fresh timestamps, while genuine repeats arrive much later.
func TestDedupeReplayedPrompts(t *testing.T) {
	event := func(ms int64, prompt string) FlightEvent {
		return FlightEvent{Timestamp: ms, Type: "agent_prompt",
			Meta: map[string]interface{}{"source": "codex", "prompt": prompt}}
	}

	events := []FlightEvent{
		event(1_000_000, "Push the changes"),
		event(1_000_010, "Push the changes"), // fan-out replica, 10ms later
		event(1_000_050, "Different prompt"),
		event(1_000_200, "Push the changes"), // fan-out replica, 200ms later
		event(1_060_000, "Push the changes"), // genuine repeat a minute later
	}

	unique := dedupeReplayedPrompts(events)
	if len(unique) != 3 {
		t.Fatalf("expected 3 events, got %d: %v", len(unique), unique)
	}
	if unique[0].Timestamp != 1_000_000 {
		t.Errorf("expected first occurrence kept, got %d", unique[0].Timestamp)
	}
	if unique[2].Timestamp != 1_060_000 {
		t.Errorf("expected genuine repeat kept, got %d", unique[2].Timestamp)
	}
}

// TestFindRelevantCodexSessions_Layout pins the YYYY/MM/DD rollout layout.
func TestFindRelevantCodexSessions_Layout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	day := filepath.Join(home, ".codex", "sessions", "2026", "07", "16")
	if err := os.MkdirAll(day, 0755); err != nil {
		t.Fatalf("failed to build fixture tree: %v", err)
	}
	content := codexSession("/repo", "Codex Desktop", "Update the auth logic")
	rollout := writeCodexFile(t, day, "rollout-2026-07-16T11-56-18-019f6c49.jsonl", content)

	// A stray log at the sessions root must not be picked up.
	writeCodexFile(t, filepath.Join(home, ".codex", "sessions"), "stray.jsonl", content)

	window := TimeWindow{Start: time.Time{}, End: time.Now().Add(24 * time.Hour)}
	paths, err := FindRelevantCodexSessions(window)
	if err != nil {
		t.Fatalf("FindRelevantCodexSessions returned error: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 session, got %d: %v", len(paths), paths)
	}
	if paths[0] != rollout {
		t.Errorf("path = %q, want %q", paths[0], rollout)
	}
}
