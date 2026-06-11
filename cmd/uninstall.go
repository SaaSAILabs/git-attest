/*
Copyright © 2026 SaaSAILabs
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove git-attest global hooks and restore default Git behavior",
	Long: `Cleanly removes the git-attest global setup:

  1. Unsets core.hooksPath so Git reverts to per-repo .git/hooks/
  2. Removes the ~/.git-attest/hooks/ directory

Existing git notes on your commits are NOT removed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUninstall()
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall() error {
	// 1. Unset core.hooksPath.
	unsetCmd := exec.Command("git", "config", "--global", "--unset", "core.hooksPath")
	_ = unsetCmd.Run() // Ignore error if already unset.
	fmt.Println("✓ Restored default Git hooks behavior")

	// 2. Remove ~/.git-attest/hooks/ directory.
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine home directory: %w", err)
	}
	attestDir := filepath.Join(home, ".git-attest")
	if err := os.RemoveAll(attestDir); err != nil {
		return fmt.Errorf("failed to remove %s: %w", attestDir, err)
	}
	fmt.Printf("✓ Removed %s\n", attestDir)

	fmt.Println()
	fmt.Println("git-attest has been cleanly uninstalled.")
	fmt.Println("Your existing git notes on commits are preserved.")

	return nil
}
