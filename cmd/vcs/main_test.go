package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestMainRootCommand(t *testing.T) {
	// Test creating the root command structure (similar to main function but testable)
	rootCmd := &cobra.Command{
		Use:   "vcs",
		Short: "A high-performance custom git implementation",
		Long: `VCS is a high-performance version control system compatible with Git.
It provides optimized performance for large repositories and seamless GitHub integration.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", "test", "test-commit", "test-date"),
	}

	// Add commands like in main()
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
	)

	// Test command properties
	assert.Equal(t, "vcs", rootCmd.Use)
	assert.Equal(t, "A high-performance custom git implementation", rootCmd.Short)
	assert.Contains(t, rootCmd.Long, "VCS is a high-performance version control system")
	assert.Contains(t, rootCmd.Version, "test (commit: test-commit, built: test-date)")

	// Test that all expected subcommands are added
	expectedCommands := []string{
		"init", "clone", "hash-object", "cat-file", "status", "add", "commit",
		"log", "branch", "checkout", "diff", "merge", "reset", "tag",
		"remote", "fetch", "push", "pull", "stash",
	}

	for _, cmdName := range expectedCommands {
		cmd, _, err := rootCmd.Find([]string{cmdName})
		assert.NoError(t, err, "Command %s should be found", cmdName)
		assert.NotNil(t, cmd, "Command %s should not be nil", cmdName)
		assert.Equal(t, cmdName, cmd.Name(), "Command name should match")
	}
}

func TestRootCommandHelp(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:   "vcs",
		Short: "A high-performance custom git implementation",
		Long: `VCS is a high-performance version control system compatible with Git.
It provides optimized performance for large repositories and seamless GitHub integration.`,
	}

	rootCmd.AddCommand(newInitCommand())
	rootCmd.AddCommand(newStatusCommand())

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "A high-performance custom git implementation")
	assert.Contains(t, output, "Available Commands:")
	assert.Contains(t, output, "init")
	assert.Contains(t, output, "status")
}

func TestRootCommandVersion(t *testing.T) {
	rootCmd := &cobra.Command{
		Use:     "vcs",
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", "v1.0.0", "abc123", "2023-01-01"),
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--version"})

	err := rootCmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "vcs version v1.0.0 (commit: abc123, built: 2023-01-01)")
}

func TestRootCommandInvalidSubcommand(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "vcs",
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"nonexistent"})

	err := rootCmd.Execute()
	assert.Error(t, err)

	output := buf.String()
	assert.Contains(t, strings.ToLower(output), "unknown command")
}

func TestCommandRegistration(t *testing.T) {
	// Test that command constructors work properly
	commands := map[string]func() *cobra.Command{
		"init":        newInitCommand,
		"clone":       newCloneCommand,
		"hash-object": newHashObjectCommand,
		"cat-file":    newCatFileCommand,
		"status":      newStatusCommand,
		"add":         newAddCommand,
		"commit":      newCommitCommand,
		"log":         newLogCommand,
		"branch":      newBranchCommand,
		"checkout":    newCheckoutCommand,
		"diff":        newDiffCommand,
		"merge":       newMergeCommand,
		"reset":       newResetCommand,
		"tag":         newTagCommand,
		"remote":      newRemoteCommand,
		"fetch":       newFetchCommand,
		"push":        newPushCommand,
		"pull":        newPullCommand,
		"stash":       newStashCommand,
	}

	for name, constructor := range commands {
		t.Run(name, func(t *testing.T) {
			cmd := constructor()
			assert.NotNil(t, cmd, "Command constructor should return non-nil command")
			assert.NotEmpty(t, cmd.Use, "Command should have Use field set")
			assert.NotEmpty(t, cmd.Short, "Command should have Short description")
		})
	}
}

func TestGlobalFlags(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "vcs",
	}

	// Test adding some common global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().String("git-dir", "", "path to git directory")

	// Add a subcommand
	subCmd := &cobra.Command{
		Use: "status",
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			gitDir, _ := cmd.Flags().GetString("git-dir")
			
			if verbose {
				fmt.Fprintf(cmd.OutOrStdout(), "verbose mode enabled\n")
			}
			if gitDir != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "git-dir: %s\n", gitDir)
			}
			return nil
		},
	}
	rootCmd.AddCommand(subCmd)

	// Test verbose flag
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--verbose", "status"})

	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "verbose mode enabled")

	// Test git-dir flag
	buf.Reset()
	rootCmd.SetArgs([]string{"--git-dir", "/custom/git", "status"})

	err = rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "git-dir: /custom/git")
}

func TestCommandCompletion(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "vcs",
	}

	rootCmd.AddCommand(newInitCommand())
	rootCmd.AddCommand(newStatusCommand())

	// Test bash completion
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.GenBashCompletion(&buf)
	assert.NoError(t, err)

	completion := buf.String()
	assert.Contains(t, completion, "vcs")
	assert.Contains(t, completion, "complete")
}

// Test version variables (these are set at build time)
func TestVersionVariables(t *testing.T) {
	// Test default values
	assert.NotEmpty(t, version)
	assert.NotEmpty(t, commit)
	assert.NotEmpty(t, date)

	// Version should be "dev" by default
	assert.Equal(t, "dev", version)
	assert.Equal(t, "none", commit)
	assert.Equal(t, "unknown", date)
}

func TestCommandPanicRecovery(t *testing.T) {
	// Test that a panicking command doesn't crash the whole program
	panicCmd := &cobra.Command{
		Use: "panic",
		RunE: func(cmd *cobra.Command, args []string) error {
			panic("test panic")
		},
	}

	rootCmd := &cobra.Command{
		Use: "vcs",
	}
	rootCmd.AddCommand(panicCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"panic"})

	// This should not crash the test
	assert.Panics(t, func() {
		rootCmd.Execute()
	})
}

func TestCommandContextHandling(t *testing.T) {
	// Test that commands can handle context properly
	contextCmd := &cobra.Command{
		Use: "context",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				return fmt.Errorf("context is nil")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "context available\n")
			return nil
		},
	}

	rootCmd := &cobra.Command{
		Use: "vcs",
	}
	rootCmd.AddCommand(contextCmd)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"context"})

	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "context available")
}

func TestCommandErrorHandling(t *testing.T) {
	// Test various error scenarios
	errorCmd := &cobra.Command{
		Use: "error",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && args[0] == "fail" {
				return fmt.Errorf("command failed")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "command succeeded\n")
			return nil
		},
	}

	rootCmd := &cobra.Command{
		Use: "vcs",
	}
	rootCmd.AddCommand(errorCmd)

	// Test success case
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"error"})

	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "command succeeded")

	// Test error case
	buf.Reset()
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"error", "fail"})

	err = rootCmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command failed")
}

func TestAllCommandsHaveDescriptions(t *testing.T) {
	// Ensure all commands have proper descriptions
	constructors := []func() *cobra.Command{
		newInitCommand,
		newCloneCommand,
		newHashObjectCommand,
		newCatFileCommand,
		newStatusCommand,
		newAddCommand,
		newCommitCommand,
		newLogCommand,
		newBranchCommand,
		newCheckoutCommand,
		newDiffCommand,
		newMergeCommand,
		newResetCommand,
		newTagCommand,
		newRemoteCommand,
		newFetchCommand,
		newPushCommand,
		newPullCommand,
		newStashCommand,
	}

	for i, constructor := range constructors {
		t.Run(fmt.Sprintf("command_%d", i), func(t *testing.T) {
			cmd := constructor()
			assert.NotEmpty(t, cmd.Use, "Command Use should not be empty")
			assert.NotEmpty(t, cmd.Short, "Command Short description should not be empty")
			// Long description is optional but if present should not be empty
			if cmd.Long != "" {
				assert.NotEmpty(t, strings.TrimSpace(cmd.Long), "Command Long description should not be just whitespace")
			}
		})
	}
}

// Test the actual main function behavior (without calling os.Exit)
func TestMainFunctionBehavior(t *testing.T) {
	// We can't easily test the actual main() function because it calls os.Exit,
	// but we can test the core logic
	
	// Create a root command like main() does
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
	)

	// Test help output
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "A high-performance custom git implementation")
	assert.Contains(t, output, "Available Commands:")

	// Test version output
	buf.Reset()
	rootCmd.SetArgs([]string{"--version"})

	err = rootCmd.Execute()
	assert.NoError(t, err)

	versionOutput := buf.String()
	assert.Contains(t, versionOutput, fmt.Sprintf("vcs version %s", version))
	assert.Contains(t, versionOutput, fmt.Sprintf("commit: %s", commit))
	assert.Contains(t, versionOutput, fmt.Sprintf("built: %s", date))
}