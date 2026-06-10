package payload

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/SaaSAILabs/attest-cli.git/pkg/active"
	"github.com/SaaSAILabs/attest-cli.git/pkg/passive"
)

const payloadVersion = "0.1.0"

// GitNotePayload is the top-level schema attached to a Git commit via git notes.
type GitNotePayload struct {
	Version         string                  `json:"version"`
	Profile         string                  `json:"profile"`
	CommitTimestamp int64                   `json:"commit_timestamp"`
	Summary         SummaryMetrics          `json:"summary"`
	Forensics       passive.ForensicProfile `json:"forensics"`
	FlightRecorder  []active.FlightEvent    `json:"flight_recorder"`
}

// SummaryMetrics holds the computed forensic timing deltas.
type SummaryMetrics struct {
	ConstructionSpreadMs int64 `json:"first_to_last_file_mod_gap_ms"`
	DeliberationGapMs    int64 `json:"last_mod_to_commit_gap_ms"`
	TotalEvents          int   `json:"total_prompt_events"`
}

// Build assembles a GitNotePayload from the active telemetry events, passive
// forensics metrics, and the current commit time.
func Build(
	promptEvents []active.FlightEvent,
	metrics *passive.TimelineMetrics,
	commitTime time.Time,
) *GitNotePayload {
	// Merge prompt events and file modification events.
	merged := make([]active.FlightEvent, 0, len(promptEvents)+len(metrics.Events))
	merged = append(merged, promptEvents...)
	merged = append(merged, metrics.Events...)

	// Sort chronologically by timestamp.
	SortEvents(merged)

	// Dynamically determine the profile from the active daemons.
	sourceSet := make(map[string]struct{})
	for _, ev := range promptEvents {
		daemon, ok := ev.Meta["daemon"].(string)
		if !ok || daemon == "" {
			daemon, _ = ev.Meta["source"].(string)
		}
		if daemon != "" && !strings.HasPrefix(daemon, "/") {
			sourceSet[daemon] = struct{}{}
		}
	}
	var sources []string
	for src := range sourceSet {
		sources = append(sources, src)
	}
	profile := "unknown"
	if len(sources) > 0 {
		profile = sources[0]
		for i := 1; i < len(sources); i++ {
			profile += "," + sources[i]
		}
	}

	return &GitNotePayload{
		Version:         payloadVersion,
		Profile:         profile,
		CommitTimestamp: commitTime.UnixMilli(),
		Summary: SummaryMetrics{
			ConstructionSpreadMs: metrics.ConstructionSpread.Milliseconds(),
			DeliberationGapMs:    metrics.DeliberationGap.Milliseconds(),
			TotalEvents:          len(merged),
		},
		Forensics:      metrics.Forensics,
		FlightRecorder: merged,
	}
}

// Serialize marshals the payload to compact, deterministic JSON.
func (p *GitNotePayload) Serialize() (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("failed to serialize payload: %w", err)
	}
	return string(data), nil
}

// Attach writes the JSON payload as a git note on HEAD.
func Attach(jsonPayload string) error {
	cmd := exec.Command("git", "notes", "add", "-f", "-m", jsonPayload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git notes add failed: %w\n%s", err, string(output))
	}
	return nil
}
