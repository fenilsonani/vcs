package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
	"github.com/spf13/cobra"
)

func newCheckoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkout [flags] <branch|commit>",
		Short: "Switch branches or restore working tree files",
		Long: `Updates files in the working tree to match the version in the index or the specified tree.
If no pathspec is given, also updates HEAD to set the specified branch as the current branch.`,
		RunE: runCheckout,
	}

	cmd.Flags().BoolP("force", "f", false, "Force checkout (lose local changes)")
	cmd.Flags().BoolP("create", "b", false, "Create a new branch and switch to it")

	return cmd
}

func runCheckout(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("checkout requires exactly one argument")
	}

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

	// Get flags
	force, _ := cmd.Flags().GetBool("force")
	createBranch, _ := cmd.Flags().GetBool("create")

	target := args[0]

	// Get reference manager
	refManager := refs.NewRefManager(repo.GitDir())

	// Handle branch creation
	if createBranch {
		return createAndCheckoutBranch(repo, refManager, target, force)
	}

	// Check if target is a branch or commit
	var targetCommitID objects.ObjectID
	var isBranch bool

	// Try to resolve as branch first
	if refManager.RefExists(target) {
		targetCommitID, err = refManager.ResolveRef(target)
		if err != nil {
			return fmt.Errorf("failed to resolve branch %s: %w", target, err)
		}
		isBranch = true
	} else {
		// Try to parse as commit ID
		targetCommitID, err = objects.NewObjectID(target)
		if err != nil {
			return fmt.Errorf("invalid branch or commit: %s", target)
		}
		isBranch = false

		// Verify the commit exists
		_, err = repo.ReadObject(targetCommitID)
		if err != nil {
			return fmt.Errorf("commit does not exist: %s", target)
		}
	}

	// Check for uncommitted changes (unless force)
	if !force {
		hasChanges, err := hasUncommittedChanges(repo, refManager)
		if err != nil {
			return fmt.Errorf("failed to check for changes: %w", err)
		}
		if hasChanges {
			return fmt.Errorf("your local changes would be overwritten by checkout. Use -f to force")
		}
	}

	// Update working directory
	if err := updateWorkingDirectory(repo, targetCommitID, repoPath); err != nil {
		return fmt.Errorf("failed to update working directory: %w", err)
	}

	// Update HEAD
	if isBranch {
		if err := refManager.SetHEAD("refs/heads/" + target); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}
		fmt.Printf("Switched to branch '%s'\n", target)
	} else {
		if err := refManager.SetHEADToCommit(targetCommitID); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}
		fmt.Printf("HEAD is now at %s\n", targetCommitID.String()[:7])
	}

	// Clear index (for simplicity)
	idx := index.New()
	indexPath := filepath.Join(repo.GitDir(), "index")
	if err := idx.WriteToFile(indexPath); err != nil {
		return fmt.Errorf("failed to clear index: %w", err)
	}

	return nil
}

func createAndCheckoutBranch(repo *vcs.Repository, refManager *refs.RefManager, branchName string, force bool) error {
	// Validate branch name
	if !refManager.IsValidRef("refs/heads/"+branchName) {
		return fmt.Errorf("invalid branch name: %s", branchName)
	}

	// Check if branch already exists
	if refManager.RefExists(branchName) && !force {
		return fmt.Errorf("branch '%s' already exists", branchName)
	}

	// Get current HEAD for starting point
	currentCommitID, _, err := refManager.HEAD()
	if err != nil || currentCommitID.IsZero() {
		return fmt.Errorf("no commits found to start branch from")
	}

	// Create the branch
	if err := refManager.CreateBranch(branchName, currentCommitID); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Switch to the new branch
	if err := refManager.SetHEAD("refs/heads/" + branchName); err != nil {
		return fmt.Errorf("failed to switch to new branch: %w", err)
	}

	fmt.Printf("Switched to a new branch '%s'\n", branchName)
	return nil
}

func hasUncommittedChanges(repo *vcs.Repository, refManager *refs.RefManager) (bool, error) {
	// For simplicity, we'll just check if the index has entries
	// A full implementation would compare working directory with HEAD
	idx := index.New()
	indexPath := filepath.Join(repo.GitDir(), "index")
	if _, err := os.Stat(indexPath); err == nil {
		if err := idx.ReadFromFile(indexPath); err == nil {
			return len(idx.Entries()) > 0, nil
		}
	}
	return false, nil
}

func updateWorkingDirectory(repo *vcs.Repository, commitID objects.ObjectID, repoPath string) error {
	// Read the commit
	obj, err := repo.ReadObject(commitID)
	if err != nil {
		return err
	}

	commit, ok := obj.(*objects.Commit)
	if !ok {
		return fmt.Errorf("object is not a commit")
	}

	// Read the tree
	treeObj, err := repo.ReadObject(commit.Tree())
	if err != nil {
		return err
	}

	tree, ok := treeObj.(*objects.Tree)
	if !ok {
		return fmt.Errorf("commit tree is not a tree object")
	}

	// Clear working directory (except .git and untracked files)
	// For simplicity, we'll just remove files that exist in the tree
	for _, entry := range tree.Entries() {
		filePath := filepath.Join(repoPath, entry.Name)
		os.Remove(filePath) // Ignore errors
	}

	// Extract files from tree
	for _, entry := range tree.Entries() {
		if entry.Mode == objects.ModeBlob || entry.Mode == objects.ModeExec {
			if err := extractFile(repo, entry, repoPath); err != nil {
				return fmt.Errorf("failed to extract file %s: %w", entry.Name, err)
			}
		}
		// TODO: Handle subdirectories (trees within trees)
	}

	return nil
}

func extractFile(repo *vcs.Repository, entry objects.TreeEntry, repoPath string) error {
	// Read the blob
	obj, err := repo.ReadObject(entry.ID)
	if err != nil {
		return err
	}

	blob, ok := obj.(*objects.Blob)
	if !ok {
		return fmt.Errorf("tree entry is not a blob")
	}

	// Write file
	filePath := filepath.Join(repoPath, entry.Name)
	fileMode := os.FileMode(0644)
	if entry.Mode == objects.ModeExec {
		fileMode = os.FileMode(0755)
	}

	return os.WriteFile(filePath, blob.Data(), fileMode)
}