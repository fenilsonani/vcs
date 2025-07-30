package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

type stashEntry struct {
	ID       objects.ObjectID
	Message  string
	Branch   string
	Parent   objects.ObjectID
	Tree     objects.ObjectID
	Date     time.Time
}

func newStashCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stash",
		Short: "Stash the changes in a dirty working directory away",
		Long:  `Use git stash when you want to record the current state of the working directory and the index, but want to go back to a clean working directory.`,
		RunE:  runStashSave,
	}

	// Subcommands
	cmd.AddCommand(
		newStashListCommand(),
		newStashShowCommand(),
		newStashPopCommand(),
		newStashApplyCommand(),
		newStashDropCommand(),
		newStashClearCommand(),
		newStashPushCommand(),
	)

	return cmd
}

func newStashPushCommand() *cobra.Command {
	var (
		message      string
		keepIndex    bool
		includeUntracked bool
	)

	cmd := &cobra.Command{
		Use:   "push [-m <message>] [--] [<pathspec>...]",
		Short: "Save your local modifications to a new stash entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStashSave(cmd, args)
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "Stash message")
	cmd.Flags().BoolVarP(&keepIndex, "keep-index", "k", false, "Keep changes in the index")
	cmd.Flags().BoolVarP(&includeUntracked, "include-untracked", "u", false, "Include untracked files")

	return cmd
}

func newStashListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the stash entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStashList(cmd)
		},
	}
}

func newStashShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show [<stash>]",
		Short: "Show the changes recorded in the stash entry",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stashRef := "stash@{0}"
			if len(args) > 0 {
				stashRef = args[0]
			}
			return runStashShow(cmd, stashRef)
		},
	}
}

func newStashPopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pop [<stash>]",
		Short: "Apply a stash entry and remove it from the stash list",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stashRef := "stash@{0}"
			if len(args) > 0 {
				stashRef = args[0]
			}
			return runStashPop(cmd, stashRef)
		},
	}
}

func newStashApplyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "apply [<stash>]",
		Short: "Apply a stash entry on top of the current working tree",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stashRef := "stash@{0}"
			if len(args) > 0 {
				stashRef = args[0]
			}
			return runStashApply(cmd, stashRef)
		},
	}
}

func newStashDropCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "drop [<stash>]",
		Short: "Remove a single stash entry from the list",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stashRef := "stash@{0}"
			if len(args) > 0 {
				stashRef = args[0]
			}
			return runStashDrop(cmd, stashRef)
		},
	}
}

func newStashClearCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all the stash entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStashClear(cmd)
		},
	}
}

func runStashSave(cmd *cobra.Command, args []string) error {
	// Find repository
	repoPath, err := findRepository()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	// Open repository
	repo, err := vcs.Open(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Check if there are changes to stash
	hasChanges, err := hasLocalChanges(repo)
	if err != nil {
		return fmt.Errorf("failed to check for changes: %w", err)
	}

	if !hasChanges {
		fmt.Fprintln(cmd.OutOrStdout(), "No local changes to save")
		return nil
	}

	// Get current branch
	refManager := refs.NewRefManager(repo.GitDir())
	currentBranch, _ := refManager.CurrentBranch()
	if currentBranch == "" {
		currentBranch = "HEAD"
	}

	// Get current commit
	currentCommitID, _, err := refManager.HEAD()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Create stash message
	message := fmt.Sprintf("WIP on %s: %s", currentBranch, currentCommitID.String()[:7])
	
	// In a real implementation, we would:
	// 1. Create a tree object with current working tree state
	// 2. Create a commit object with that tree
	// 3. Store it in refs/stash
	// 4. Update the stash reflog
	// 5. Reset working tree to HEAD state

	// For now, create a simple stash structure
	stashDir := filepath.Join(repo.GitDir(), "stash")
	if err := ensureDir(stashDir); err != nil {
		return fmt.Errorf("failed to create stash directory: %w", err)
	}

	// Save stash metadata
	stashFile := filepath.Join(stashDir, "stash_list")
	stashEntry := fmt.Sprintf("%s %s %s\n", time.Now().Format(time.RFC3339), currentBranch, message)
	if err := appendToFile(stashFile, []byte(stashEntry)); err != nil {
		return fmt.Errorf("failed to save stash: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Saved working directory and index state %s\n", message)
	fmt.Fprintln(cmd.OutOrStdout(), "\nNote: This is a basic stash implementation.")
	fmt.Fprintln(cmd.OutOrStdout(), "Full implementation would:")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Create tree objects for working directory and index")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Create stash commits")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Reset working directory")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Maintain stash reflog")

	return nil
}

func runStashList(cmd *cobra.Command) error {
	// Find repository
	repoPath, err := findRepository()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	// Open repository
	repo, err := vcs.Open(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Read stash list
	stashFile := filepath.Join(repo.GitDir(), "stash", "stash_list")
	if !fileExists(stashFile) {
		return nil // No stashes
	}

	data, err := readFile(stashFile)
	if err != nil {
		return fmt.Errorf("failed to read stash list: %w", err)
	}

	// Display stashes
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) >= 3 {
			fmt.Fprintf(cmd.OutOrStdout(), "stash@{%d}: %s\n", i, parts[2])
		}
	}

	return nil
}

func runStashShow(cmd *cobra.Command, stashRef string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "Showing stash %s\n", stashRef)
	fmt.Fprintln(cmd.OutOrStdout(), "Note: Full stash show would display the diff of the stashed changes")
	return nil
}

func runStashPop(cmd *cobra.Command, stashRef string) error {
	// Apply the stash
	if err := runStashApply(cmd, stashRef); err != nil {
		return err
	}

	// Then drop it
	return runStashDrop(cmd, stashRef)
}

func runStashApply(cmd *cobra.Command, stashRef string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "Applying stash %s\n", stashRef)
	fmt.Fprintln(cmd.OutOrStdout(), "Note: Full stash apply would:")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Retrieve stashed tree objects")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Apply changes to working directory")
	fmt.Fprintln(cmd.OutOrStdout(), "  - Handle conflicts if any")
	return nil
}

func runStashDrop(cmd *cobra.Command, stashRef string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "Dropped %s\n", stashRef)
	fmt.Fprintln(cmd.OutOrStdout(), "Note: Full stash drop would remove the stash entry from refs/stash")
	return nil
}

func runStashClear(cmd *cobra.Command) error {
	// Find repository
	repoPath, err := findRepository()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	// Open repository
	repo, err := vcs.Open(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// Clear stash file
	stashFile := filepath.Join(repo.GitDir(), "stash", "stash_list")
	if fileExists(stashFile) {
		if err := os.Remove(stashFile); err != nil {
			return fmt.Errorf("failed to clear stash: %w", err)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Cleared all stashes")
	return nil
}

func hasLocalChanges(repo *vcs.Repository) (bool, error) {
	// Check index for staged changes
	indexPath := filepath.Join(repo.GitDir(), "index")
	if fileExists(indexPath) {
		idx := index.New()
		if err := idx.ReadFromFile(indexPath); err == nil && len(idx.Entries()) > 0 {
			return true, nil
		}
	}

	// In a real implementation, we would also check:
	// - Working tree changes
	// - Untracked files (if --include-untracked)
	
	return false, nil
}