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

// hookScript is the shell script injected into .git/hooks/prepare-commit-msg.
// It includes a bypass for ATTEST_DEV_MODE so developers can disable the hook
// without uninstalling it.
const hookScript = `#!/bin/sh
# attest-cli: prepare-commit-msg hook
# This hook is managed by attest-cli. Do not edit manually.

if [ -n "$ATTEST_DEV_MODE" ]; then
  exit 0
fi

attest internal-hook "$@"
`

// installCmd installs the prepare-commit-msg git hook into the current repository.
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the attest prepare-commit-msg git hook",
	Long: `Install the attest flight-recorder hook into the current Git repository.

This creates a prepare-commit-msg hook at .git/hooks/prepare-commit-msg
that automatically captures forensic evidence on every commit.

The hook can be bypassed at runtime by setting the ATTEST_DEV_MODE
environment variable to any non-empty value.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInstall()
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

// runInstall validates the git directory and writes the hook file.
func runInstall() error {
	// 1. Validate that we are inside a git repository.
	gitDir := ".git"
	info, err := os.Stat(gitDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("not a git repository: .git directory not found in the current working directory")
	}

	// 2. Ensure .git/hooks/ directory exists.
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// 3. Write the prepare-commit-msg hook with executable permissions.
	hookPath := filepath.Join(hooksDir, "prepare-commit-msg")
	if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
		return fmt.Errorf("failed to write hook file: %w", err)
	}

	fmt.Printf("✓ Hook installed at %s\n", hookPath)

	// 4. Automatically configure Git to push notes if origin exists
	if err := configureGitPush(); err != nil {
		fmt.Printf("⚠ Warning: Could not configure auto-push: %v\n", err)
	}

	return nil
}

// configureGitPush sets up the local Git repository to automatically push
// Git Notes alongside standard branch pushes, if an origin remote exists.
func configureGitPush() error {
	// Check if origin remote exists
	if err := exec.Command("git", "remote", "get-url", "origin").Run(); err != nil {
		return nil // No origin remote, skip gracefully
	}

	// Check if notes refspec already exists
	checkCmd := exec.Command("git", "config", "--get", "remote.origin.push", "\\+refs/notes/\\*:refs/notes/\\*")
	if err := checkCmd.Run(); err == nil {
		return nil // Already configured
	}

	// Configure HEAD first so standard branch pushing still works natively
	if err := exec.Command("git", "config", "--add", "remote.origin.push", "HEAD").Run(); err != nil {
		return fmt.Errorf("failed to configure HEAD push: %w", err)
	}
	// Configure notes refspec
	if err := exec.Command("git", "config", "--add", "remote.origin.push", "+refs/notes/*:refs/notes/*").Run(); err != nil {
		return fmt.Errorf("failed to configure notes push: %w", err)
	}
	
	fmt.Println("✓ Git configured to automatically push flight recordings to 'origin'")
	return nil
}
