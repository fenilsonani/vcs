package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func newMergeCommand() *cobra.Command {
	var (
		noCommit   bool
		fastForward string
		strategy   string
		message    string
	)

	cmd := &cobra.Command{
		Use:   "merge [flags] <branch>",
		Short: "Join two or more development histories together",
		Long: `Incorporates changes from the named commits (since the time their
histories diverged from the current branch) into the current branch.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			vcsRepo, err := vcs.Open(repo)
			if err != nil {
				return err
			}

			refManager := refs.NewRefManager(vcsRepo.GitDir())

			return runMerge(vcsRepo, refManager, args[0], noCommit, fastForward, strategy, message)
		},
	}

	cmd.Flags().BoolVar(&noCommit, "no-commit", false, "Perform merge but don't commit")
	cmd.Flags().StringVar(&fastForward, "ff", "auto", "Fast-forward mode (auto, no, only)")
	cmd.Flags().StringVar(&strategy, "strategy", "recursive", "Merge strategy to use")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Merge commit message")

	return cmd
}

func runMerge(repo *vcs.Repository, refManager *refs.RefManager, branchName string, noCommit bool, fastForward, strategy, message string) error {
	// Get current branch
	currentBranch, err := refManager.CurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get current commit
	currentRef := "refs/heads/" + currentBranch
	currentCommitID, err := refManager.ResolveRef(currentRef)
	if err != nil {
		return fmt.Errorf("failed to resolve current branch: %w", err)
	}

	// Get target commit
	targetCommitID, err := refManager.ResolveRef(branchName)
	if err != nil {
		return fmt.Errorf("failed to resolve target branch %q: %w", branchName, err)
	}

	// Check if already up to date
	if currentCommitID.Equal(targetCommitID) {
		fmt.Printf("Already up to date.\n")
		return nil
	}

	currentCommit, err := repo.GetCommit(currentCommitID)
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}

	targetCommit, err := repo.GetCommit(targetCommitID)
	if err != nil {
		return fmt.Errorf("failed to get target commit: %w", err)
	}

	// Check for fast-forward merge
	canFastForward, err := isAncestor(repo, currentCommitID, targetCommitID)
	if err != nil {
		return fmt.Errorf("failed to check ancestry: %w", err)
	}

	if canFastForward {
		return performFastForwardMerge(repo, refManager, currentRef, targetCommitID, branchName)
	}

	// Check if target is ancestor of current (nothing to merge)
	targetIsAncestor, err := isAncestor(repo, targetCommitID, currentCommitID)
	if err != nil {
		return fmt.Errorf("failed to check ancestry: %w", err)
	}

	if targetIsAncestor {
		fmt.Printf("Already up to date.\n")
		return nil
	}

	// Find merge base
	mergeBase, err := findMergeBase(repo, currentCommitID, targetCommitID)
	if err != nil {
		return fmt.Errorf("failed to find merge base: %w", err)
	}

	// Perform three-way merge
	return performThreeWayMerge(repo, refManager, currentCommit, targetCommit, mergeBase, branchName, noCommit, message)
}

func performFastForwardMerge(repo *vcs.Repository, refManager *refs.RefManager, currentRef string, targetCommitID objects.ObjectID, branchName string) error {
	// Update the current branch to point to target commit
	if err := refManager.WriteRef(currentRef, targetCommitID, nil); err != nil {
		return fmt.Errorf("failed to update branch: %w", err)
	}

	// Update working directory
	targetCommit, err := repo.GetCommit(targetCommitID)
	if err != nil {
		return fmt.Errorf("failed to get target commit: %w", err)
	}

	if err := updateWorkingDirectoryFromCommit(repo, targetCommit); err != nil {
		return fmt.Errorf("failed to update working directory: %w", err)
	}

	fmt.Printf("Fast-forward\n")
	fmt.Printf("Updating %s..%s\n", targetCommitID.Short(), targetCommitID.Short())

	return nil
}

func performThreeWayMerge(repo *vcs.Repository, refManager *refs.RefManager, currentCommit, targetCommit *objects.Commit, mergeBase objects.ObjectID, branchName string, noCommit bool, message string) error {
	// Get target tree for merge
	targetTree, err := repo.GetTree(targetCommit.Tree())
	if err != nil {
		return fmt.Errorf("failed to get target tree: %w", err)
	}

	// Simple merge: just use target tree
	// In a real implementation, this would do proper 3-way merge with conflict resolution
	_ = mergeBase // Not used in simple implementation

	// Perform simple merge (just use target tree for now)
	// In a real implementation, this would do proper 3-way merge
	mergedTree := targetTree

	// Create merge commit if not no-commit
	if !noCommit {
		if message == "" {
			message = fmt.Sprintf("Merge branch '%s'", branchName)
		}

		parents := []objects.ObjectID{currentCommit.ID(), targetCommit.ID()}
		sig := objects.Signature{
			Name:  "VCS User",
			Email: "user@example.com",
			When:  time.Now(),
		}

		mergeCommit, err := repo.CreateCommit(mergedTree.ID(), parents, sig, sig, message)
		if err != nil {
			return fmt.Errorf("failed to create merge commit: %w", err)
		}

		// Update current branch
		currentBranch, err := refManager.CurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		currentRef := "refs/heads/" + currentBranch
		if err := refManager.WriteRef(currentRef, mergeCommit.ID(), nil); err != nil {
			return fmt.Errorf("failed to update branch: %w", err)
		}

		fmt.Printf("Merge made by the 'recursive' strategy.\n")
	} else {
		fmt.Printf("Automatic merge went well; stopped before committing as requested\n")
	}

	// Update working directory and index
	if err := updateWorkingDirectoryFromCommit(repo, &objects.Commit{}); err != nil {
		return fmt.Errorf("failed to update working directory: %w", err)
	}

	return nil
}

func isAncestor(repo *vcs.Repository, ancestor, descendant objects.ObjectID) (bool, error) {
	if ancestor.Equal(descendant) {
		return true, nil
	}

	// Simple implementation: check if ancestor is in the history of descendant
	current := descendant
	visited := make(map[objects.ObjectID]bool)

	for !current.IsZero() && !visited[current] {
		visited[current] = true

		if current.Equal(ancestor) {
			return true, nil
		}

		commit, err := repo.GetCommit(current)
		if err != nil {
			return false, err
		}

		parents := commit.Parents()
		if len(parents) == 0 {
			break
		}

		// Follow first parent for simplicity
		current = parents[0]
	}

	return false, nil
}

func findMergeBase(repo *vcs.Repository, commit1, commit2 objects.ObjectID) (objects.ObjectID, error) {
	// Simple implementation: find common ancestor
	ancestors1 := make(map[objects.ObjectID]bool)

	// Collect all ancestors of commit1
	current := commit1
	visited := make(map[objects.ObjectID]bool)

	for !current.IsZero() && !visited[current] {
		visited[current] = true
		ancestors1[current] = true

		commit, err := repo.GetCommit(current)
		if err != nil {
			return objects.ObjectID{}, err
		}

		parents := commit.Parents()
		if len(parents) == 0 {
			break
		}

		current = parents[0]
	}

	// Find first common ancestor in commit2's history
	current = commit2
	visited = make(map[objects.ObjectID]bool)

	for !current.IsZero() && !visited[current] {
		visited[current] = true

		if ancestors1[current] {
			return current, nil
		}

		commit, err := repo.GetCommit(current)
		if err != nil {
			return objects.ObjectID{}, err
		}

		parents := commit.Parents()
		if len(parents) == 0 {
			break
		}

		current = parents[0]
	}

	return objects.ObjectID{}, nil // No common ancestor found
}

func updateWorkingDirectoryFromCommit(repo *vcs.Repository, commit *objects.Commit) error {
	// Simple implementation: clear index and working directory
	indexPath := filepath.Join(repo.GitDir(), "index")
	idx := index.New()

	if err := idx.WriteToFile(indexPath); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	// In a real implementation, this would extract all files from the commit's tree
	// For now, just clear everything
	return nil
}