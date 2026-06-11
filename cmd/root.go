/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)



// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "git-attest",
	Short: "Transparency certificates for AI-assisted code contributions",
	Long: `git-attest captures forensic evidence about how code was written
and attaches it to your Git commits as immutable flight recordings.

After 'brew install git-attest', every repo is automatically instrumented.
No per-repo setup required.

Commands:
  git attest init         One-time global setup (run by brew automatically)
  git attest push         Push code + flight recordings together
  git attest preview      Preview the payload before committing
  git attest uninstall    Remove git-attest and restore defaults`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}


