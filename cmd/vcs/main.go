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

	// Add hardware check flag
	var checkHardware bool
	rootCmd.Flags().BoolVar(&checkHardware, "check-hardware", false, "Check hardware acceleration support")
	
	// Override run function to handle hardware check
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		if checkHardware {
			checkHardwareSupport()
			return
		}
		cmd.Help()
	}

	// Add commands
	rootCmd.AddCommand(
		newInitCommand(),
		newCloneCommand(),
		newHashObjectCommand(),
		newCatFileCommand(),
		newStatusCommand(),
		newAddCommand(),
		newCommitCommand(),
		newLogCommand(),
		newBranchCommand(),
		newCheckoutCommand(),
		newDiffCommand(),
		newMergeCommand(),
		newResetCommand(),
		newTagCommand(),
		newRemoteCommand(),
		newFetchCommand(),
		newPushCommand(),
		newPullCommand(),
		newStashCommand(),
		newBenchmarkCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}