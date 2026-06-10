/*
Copyright © 2026 SaaSAILabs
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/SaaSAILabs/attest-cli.git/pkg/active"
	"github.com/SaaSAILabs/attest-cli.git/pkg/passive"
	"github.com/SaaSAILabs/attest-cli.git/pkg/payload"
	"github.com/SaaSAILabs/attest-cli.git/pkg/privacy"
	"github.com/spf13/cobra"
)

// internalHookCmd is the entry point called by the prepare-commit-msg git hook.
// It orchestrates all five modules to gather forensic evidence and attach it.
var internalHookCmd = &cobra.Command{
	Use:    "internal-hook",
	Short:  "Gather forensic evidence and attach it to the current commit",
	Long:   `This command is invoked automatically by the prepare-commit-msg git hook. Do not run manually.`,
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runInternalHook(); err != nil {
			// Never block a commit — log the error and exit cleanly.
			fmt.Fprintf(os.Stderr, "[attest] warning: %v\n", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(internalHookCmd)
}

func runInternalHook() error {
	commitTime := time.Now()

	// --- 1. Get staged files via git diff --cached --name-only ---
	stagedFiles, err := getStagedFiles()
	if err != nil {
		return fmt.Errorf("failed to get staged files: %w", err)
	}
	if len(stagedFiles) == 0 {
		return nil // nothing staged, nothing to record
	}

	prevCommit, err := getPreviousCommitTime()
	if err != nil {
		return fmt.Errorf("failed to get previous commit time: %w", err)
	}

	window := active.TimeWindow{
		Start: prevCommit,
		End:   commitTime,
	}

	// --- 2. Active Telemetry: harvest prompts from all registered extractors ---
	promptEvents := active.HarvestAll(window)

	// --- 3. Privacy Pipeline: redact sensitive content from prompts ---
	redactor := privacy.NewRedactor()
	_ = redactor.LoadTraceFilter(".tracefilter")
	for i := range promptEvents {
		if prompt, ok := promptEvents[i].Meta["prompt"].(string); ok {
			promptEvents[i].Meta["prompt"] = redactor.Redact(prompt)
		}
	}


	// --- 4. Passive Forensics: calculate timeline metrics ---
	metrics, err := passive.CalculateTimelineMetrics(stagedFiles, commitTime)
	if err != nil {
		return fmt.Errorf("forensics failed: %w", err)
	}

	// --- 5. Build, serialize, and attach the payload ---
	note := payload.Build(promptEvents, metrics, commitTime)
	jsonStr, err := note.Serialize()
	if err != nil {
		return fmt.Errorf("serialization failed: %w", err)
	}

	if err := payload.Attach(jsonStr); err != nil {
		return fmt.Errorf("git notes attach failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[attest] ✓ flight recorder attached (%d events)\n", len(note.FlightRecorder))
	return nil
}

// getStagedFiles returns the list of staged file paths from git.
func getStagedFiles() ([]string, error) {
	out, err := exec.Command("git", "diff", "--cached", "--name-only").Output()
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

// getPreviousCommitTime gets the timestamp of the last commit.
// Returns zero time if there are no previous commits.
func getPreviousCommitTime() (time.Time, error) {
	out, err := exec.Command("git", "log", "-1", "--format=%aI").Output()
	if err != nil {
		return time.Time{}, nil // likely no commits yet
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, raw)
}
