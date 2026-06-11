/*
Copyright © 2026 SaaSAILabs
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/SaaSAILabs/git-attest/pkg/active"
	"github.com/SaaSAILabs/git-attest/pkg/passive"
	"github.com/SaaSAILabs/git-attest/pkg/payload"
	"github.com/SaaSAILabs/git-attest/pkg/privacy"
	"github.com/spf13/cobra"
)

var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview the forensic payload that would be attached to a commit right now",
	RunE: func(cmd *cobra.Command, args []string) error {
		commitTime := time.Now()

		stagedFiles, err := getStagedFiles()
		if err != nil {
			return fmt.Errorf("failed to get staged files: %w", err)
		}

		prevCommit, err := getPreviousCommitTime()
		if err != nil {
			return fmt.Errorf("failed to get previous commit time: %w", err)
		}

		window := active.TimeWindow{
			Start: prevCommit,
			End:   commitTime,
		}

		promptEvents := active.HarvestAll(window)

		redactor := privacy.NewRedactor()
		_ = redactor.LoadTraceFilter(".tracefilter")
		for i := range promptEvents {
			if prompt, ok := promptEvents[i].Meta["prompt"].(string); ok {
				promptEvents[i].Meta["prompt"] = redactor.Redact(prompt)
			}
		}

		var metrics *passive.TimelineMetrics
		if len(stagedFiles) > 0 {
			metrics, err = passive.CalculateTimelineMetrics(stagedFiles, commitTime)
			if err != nil {
				return fmt.Errorf("forensics failed: %w", err)
			}
		} else {
			metrics = &passive.TimelineMetrics{}
		}

		note := payload.Build(promptEvents, metrics, commitTime)
		
		raw, err := json.MarshalIndent(note, "", "  ")
		if err != nil {
			return err
		}
		
		fmt.Println(string(raw))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(previewCmd)
}
