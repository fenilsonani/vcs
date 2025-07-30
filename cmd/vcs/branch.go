package main

import (
	"fmt"
	"strings"

	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
	"github.com/spf13/cobra"
)

func newBranchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch [flags] [branch_name] [start_point]",
		Short: "List, create, or delete branches",
		Long: `With no arguments, list existing branches. The current branch will be highlighted with an asterisk.
With one argument, create a new branch with that name.
With two arguments, create a new branch with the first name starting at the second commit.`,
		RunE: runBranch,
	}

	cmd.Flags().BoolP("delete", "d", false, "Delete a branch")
	cmd.Flags().BoolP("force", "f", false, "Force creation or deletion")
	cmd.Flags().BoolP("list", "l", false, "List branches (default)")
	cmd.Flags().BoolP("all", "a", false, "List both remote-tracking and local branches")
	cmd.Flags().BoolP("verbose", "v", false, "Show sha1 and commit subject line for each head")

	return cmd
}

func runBranch(cmd *cobra.Command, args []string) error {
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
	deleteBranch, _ := cmd.Flags().GetBool("delete")
	force, _ := cmd.Flags().GetBool("force")
	listBranches, _ := cmd.Flags().GetBool("list")
	showAll, _ := cmd.Flags().GetBool("all")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Get reference manager
	refManager := refs.NewRefManager(repo.GitDir())

	// Handle different operations
	switch {
	case deleteBranch:
		return deleteBranchOperation(refManager, args, force)
	case len(args) == 0 || listBranches:
		return listBranchesOperation(repo, refManager, showAll, verbose)
	case len(args) == 1:
		return createBranchOperation(repo, refManager, args[0], "")
	case len(args) == 2:
		return createBranchOperation(repo, refManager, args[0], args[1])
	default:
		return fmt.Errorf("too many arguments")
	}
}

func listBranchesOperation(repo *vcs.Repository, refManager *refs.RefManager, showAll bool, verbose bool) error {
	// Get current branch
	currentBranch, err := refManager.CurrentBranch()
	isDetached := err != nil

	// List local branches
	branches, err := refManager.ListBranches()
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	if len(branches) == 0 {
		fmt.Println("No branches found")
		return nil
	}

	for _, branchRef := range branches {
		branchName := strings.TrimPrefix(branchRef, "refs/heads/")
		
		// Mark current branch with asterisk
		prefix := "  "
		if !isDetached && branchName == currentBranch {
			prefix = "* "
		}

		if verbose {
			// Show commit info
			commitID, err := refManager.ResolveRef(branchRef)
			if err != nil {
				fmt.Printf("%s%s\n", prefix, branchName)
				continue
			}

			commitInfo := ""
			if obj, err := repo.ReadObject(commitID); err == nil {
				if commit, ok := obj.(*objects.Commit); ok {
					message := strings.Split(strings.TrimSpace(commit.Message()), "\n")[0]
					if len(message) > 50 {
						message = message[:47] + "..."
					}
					commitInfo = fmt.Sprintf(" %s %s", commitID.String()[:7], message)
				}
			}

			fmt.Printf("%s%s%s\n", prefix, branchName, commitInfo)
		} else {
			fmt.Printf("%s%s\n", prefix, branchName)
		}
	}

	// Show detached HEAD if applicable
	if isDetached {
		headCommitID, _, err := refManager.HEAD()
		if err == nil && !headCommitID.IsZero() {
			prefix := "* "
			if verbose {
				commitInfo := ""
				if obj, err := repo.ReadObject(headCommitID); err == nil {
					if commit, ok := obj.(*objects.Commit); ok {
						message := strings.Split(strings.TrimSpace(commit.Message()), "\n")[0]
						if len(message) > 50 {
							message = message[:47] + "..."
						}
						commitInfo = fmt.Sprintf(" %s", message)
					}
				}
				fmt.Printf("%s(HEAD detached at %s)%s\n", prefix, headCommitID.String()[:7], commitInfo)
			} else {
				fmt.Printf("%s(HEAD detached at %s)\n", prefix, headCommitID.String()[:7])
			}
		}
	}

	// TODO: List remote branches if showAll is true
	if showAll {
		// For now, we don't have remote branches implemented
	}

	return nil
}

func createBranchOperation(repo *vcs.Repository, refManager *refs.RefManager, branchName string, startPoint string) error {
	// Validate branch name
	if !refManager.IsValidRef("refs/heads/"+branchName) {
		return fmt.Errorf("invalid branch name: %s", branchName)
	}

	// Check if branch already exists
	if refManager.RefExists(branchName) {
		return fmt.Errorf("branch '%s' already exists", branchName)
	}

	// Determine starting commit
	var startCommitID objects.ObjectID
	var err error

	if startPoint == "" {
		// Start from current HEAD
		startCommitID, _, err = refManager.HEAD()
		if err != nil || startCommitID.IsZero() {
			return fmt.Errorf("no commits found to start branch from")
		}
	} else {
		// Resolve start point
		startCommitID, err = refManager.ResolveRef(startPoint)
		if err != nil {
			// Try to parse as object ID
			startCommitID, err = objects.NewObjectID(startPoint)
			if err != nil {
				return fmt.Errorf("invalid start point: %s", startPoint)
			}
		}

		// Verify it's a commit
		obj, err := repo.ReadObject(startCommitID)
		if err != nil {
			return fmt.Errorf("start point does not exist: %s", startPoint)
		}
		if _, ok := obj.(*objects.Commit); !ok {
			return fmt.Errorf("start point is not a commit: %s", startPoint)
		}
	}

	// Create the branch
	if err := refManager.CreateBranch(branchName, startCommitID); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	fmt.Printf("Created branch '%s'\n", branchName)
	return nil
}

func deleteBranchOperation(refManager *refs.RefManager, args []string, force bool) error {
	if len(args) == 0 {
		return fmt.Errorf("branch name required for deletion")
	}

	for _, branchName := range args {
		// Check if it's the current branch
		currentBranch, err := refManager.CurrentBranch()
		if err == nil && currentBranch == branchName {
			return fmt.Errorf("cannot delete the currently active branch '%s'", branchName)
		}

		// Check if branch exists
		if !refManager.RefExists(branchName) {
			if force {
				fmt.Printf("branch '%s' not found\n", branchName)
				continue
			}
			return fmt.Errorf("branch '%s' not found", branchName)
		}

		// Delete the branch
		if err := refManager.DeleteBranch(branchName); err != nil {
			if force {
				fmt.Printf("failed to delete branch '%s': %v\n", branchName, err)
				continue
			}
			return fmt.Errorf("failed to delete branch '%s': %w", branchName, err)
		}

		fmt.Printf("Deleted branch '%s'\n", branchName)
	}

	return nil
}