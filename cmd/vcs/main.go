package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "vcs",
		Short: "A high-performance custom git implementation",
		Long: `VCS is a high-performance version control system compatible with Git.
It provides optimized performance for large repositories and seamless GitHub integration.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Add commands
	rootCmd.AddCommand(
		newInitCommand(),
		newHashObjectCommand(),
		newCatFileCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}