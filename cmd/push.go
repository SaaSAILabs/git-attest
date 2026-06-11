/*
Copyright © 2026 SaaSAILabs
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// pushCmd wraps `git push` to automatically include flight recordings (git notes).
// Usage: git attest push origin main
// Expands to: git push origin main refs/notes/commits
var pushCmd = &cobra.Command{
	Use:   "push [git push args...]",
	Short: "Push code and flight recordings together",
	Long: `A drop-in replacement for 'git push' that automatically includes
your attest flight recordings (git notes) alongside your code.

Examples:
  git attest push origin main        → git push origin main refs/notes/commits
  git attest push origin feature-x   → git push origin feature-x refs/notes/commits
  git attest push -u origin my-branch → git push -u origin my-branch refs/notes/commits

This ensures your transparency certificates always travel with your code
in a single network request, with a single authentication prompt.`,
	DisableFlagParsing: true, // Pass all flags through to git push
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPush(args)
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}

// runPush executes git push with the user's arguments plus the notes refspec.
func runPush(args []string) error {
	// Build the full command: git push <user args> refs/notes/commits
	gitArgs := []string{"push"}
	gitArgs = append(gitArgs, args...)

	// Only append notes refspec if local notes exist.
	checkCmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/notes/commits")
	if checkCmd.Run() == nil {
		gitArgs = append(gitArgs, "refs/notes/commits")
	}

	gitCmd := exec.Command("git", gitArgs...)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	gitCmd.Stdin = os.Stdin

	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}
	return nil
}
