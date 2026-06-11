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

const attestHooksDir = ".git-attest/hooks"

// globalPrepareCommitMsg is the hook that runs on EVERY repo.
// It chains to any repo-local prepare-commit-msg hook so existing
// project hooks (linters, CI checks, etc.) are never broken.
const globalPrepareCommitMsg = `#!/bin/sh
# git-attest: global prepare-commit-msg hook
# Managed by git-attest. Do not edit manually.

# --- Bypass ---
if [ -n "$ATTEST_DEV_MODE" ]; then
  exit 0
fi

# --- 1. Run git-attest evidence capture ---
if command -v git-attest >/dev/null 2>&1; then
  git-attest internal-hook "$@"
fi

# --- 2. Chain to repo-local hook (if one exists) ---
local_hook="$(git rev-parse --git-dir)/hooks/prepare-commit-msg"
if [ -x "$local_hook" ]; then
  "$local_hook" "$@"
fi
`

// initCmd performs the one-time global setup.
// After this, every repo on the machine is automatically instrumented.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "One-time global setup — instruments every repo on this machine",
	Long: `Performs the one-time global setup for git-attest.

This:
  1. Creates a global hooks directory at ~/.git-attest/hooks/
  2. Installs a prepare-commit-msg hook that captures forensic evidence
  3. Sets git's core.hooksPath to use this global directory
  4. The hook automatically chains to any repo-local hooks

After running this command, every git commit on this machine will
automatically have a transparency certificate attached. No per-repo
setup is needed.

This command is idempotent and safe to run multiple times.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to determine home directory: %w", err)
	}

	// 1. Create ~/.git-attest/hooks/
	hooksDir := filepath.Join(home, attestHooksDir)
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// 2. Write the global prepare-commit-msg hook.
	hookPath := filepath.Join(hooksDir, "prepare-commit-msg")
	if err := os.WriteFile(hookPath, []byte(globalPrepareCommitMsg), 0755); err != nil {
		return fmt.Errorf("failed to write hook: %w", err)
	}
	fmt.Printf("✓ Hook installed at %s\n", hookPath)

	// 3. Set core.hooksPath globally.
	setCmd := exec.Command("git", "config", "--global", "core.hooksPath", hooksDir)
	if out, err := setCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set core.hooksPath: %w\n%s", err, string(out))
	}
	fmt.Printf("✓ Global core.hooksPath set to %s\n", hooksDir)

	fmt.Println()
	fmt.Println("🎉 git-attest is active on every repository on this machine.")
	fmt.Println()
	fmt.Println("Your workflow:")
	fmt.Println("  git commit -m \"message\"              → evidence captured automatically")
	fmt.Println("  git push                             → works as usual")
	fmt.Println("  git attest push origin feature-x     → pushes code + flight recordings")
	fmt.Println()
	fmt.Println("To disable temporarily:  ATTEST_DEV_MODE=1 git commit ...")
	fmt.Println("To uninstall:            git attest uninstall")

	return nil
}
