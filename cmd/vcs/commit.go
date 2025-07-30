package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
	"github.com/spf13/cobra"
)

func newCommitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Record changes to the repository",
		Long: `Stores the current contents of the index in a new commit along with a log message 
from the user describing the changes.`,
		RunE: runCommit,
	}

	cmd.Flags().StringP("message", "m", "", "Use the given message as the commit message")
	cmd.Flags().StringP("file", "F", "", "Take the commit message from the given file")
	cmd.Flags().Bool("allow-empty", false, "Usually recording a commit that has the exact same tree as its sole parent commit is a mistake, and the command prevents you from making such a commit. This option bypasses the safety")
	cmd.Flags().StringP("author", "", "", "Override the commit author (format: Name <email>)")
	cmd.Flags().Bool("amend", false, "Replace the tip of the current branch by creating a new commit")

	return cmd
}

func runCommit(cmd *cobra.Command, args []string) error {
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
	message, _ := cmd.Flags().GetString("message")
	messageFile, _ := cmd.Flags().GetString("file")
	allowEmpty, _ := cmd.Flags().GetBool("allow-empty")
	authorStr, _ := cmd.Flags().GetString("author")
	amend, _ := cmd.Flags().GetBool("amend")

	// Get commit message
	if message == "" && messageFile == "" {
		return fmt.Errorf("no commit message provided (use -m or -F)")
	}

	if messageFile != "" {
		content, err := os.ReadFile(messageFile)
		if err != nil {
			return fmt.Errorf("failed to read message file: %w", err)
		}
		message = string(content)
	}

	// Ensure message ends with newline
	if !strings.HasSuffix(message, "\n") {
		message += "\n"
	}

	// Get index
	idx := index.New()
	indexPath := filepath.Join(repo.GitDir(), "index")
	if _, err := os.Stat(indexPath); err == nil {
		if err := idx.ReadFromFile(indexPath); err != nil {
			return fmt.Errorf("failed to read index: %w", err)
		}
	}

	// Check if there are changes to commit
	if len(idx.Entries()) == 0 && !allowEmpty {
		return fmt.Errorf("nothing to commit")
	}

	// Create tree from index
	tree, err := createTreeFromIndex(repo, idx)
	if err != nil {
		return fmt.Errorf("failed to create tree: %w", err)
	}

	// Get reference manager
	refManager := refs.NewRefManager(repo.GitDir())

	// Get current HEAD and parent commits
	var parents []objects.ObjectID
	if !amend {
		currentCommitID, _, err := refManager.HEAD()
		if err == nil && !currentCommitID.IsZero() {
			parents = append(parents, currentCommitID)
		}
	} else {
		// For amend, get the parents of the current commit
		currentCommitID, _, err := refManager.HEAD()
		if err == nil && !currentCommitID.IsZero() {
			currentCommit, err := repo.ReadObject(currentCommitID)
			if err == nil {
				if commit, ok := currentCommit.(*objects.Commit); ok {
					parents = commit.Parents()
				}
			}
		}
	}

	// Create author and committer signatures
	author, err := getSignature(authorStr)
	if err != nil {
		return fmt.Errorf("invalid author format: %w", err)
	}

	committer := author // For now, author and committer are the same

	// Create commit
	commit, err := repo.CreateCommit(tree.ID(), parents, author, committer, message)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	// Update HEAD to point to new commit
	currentBranch, err := refManager.CurrentBranch()
	if err != nil {
		// Detached HEAD, update HEAD directly
		if err := refManager.SetHEADToCommit(commit.ID()); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}
	} else {
		// Update current branch
		branchRef := "refs/heads/" + currentBranch
		if err := refManager.UpdateRef(branchRef, commit.ID()); err != nil {
			return fmt.Errorf("failed to update branch %s: %w", currentBranch, err)
		}
	}

	// Clear the index after successful commit
	fileCount := len(idx.Entries())
	idx.Clear()
	if err := idx.WriteToFile(indexPath); err != nil {
		return fmt.Errorf("failed to clear index: %w", err)
	}

	// Print commit summary
	if amend {
		fmt.Printf("[%s %s] %s", getCurrentBranchName(refManager), commit.ID().String()[:7], strings.TrimSpace(message))
	} else {
		commitCount := len(parents)
		if commitCount == 0 {
			fmt.Printf("[%s (root-commit) %s] %s", getCurrentBranchName(refManager), commit.ID().String()[:7], strings.TrimSpace(message))
		} else {
			fmt.Printf("[%s %s] %s", getCurrentBranchName(refManager), commit.ID().String()[:7], strings.TrimSpace(message))
		}
	}
	fmt.Printf("\n %d file(s) changed\n", fileCount)

	return nil
}

func createTreeFromIndex(repo *vcs.Repository, idx *index.Index) (*objects.Tree, error) {
	var entries []objects.TreeEntry

	for _, entry := range idx.Entries() {
		treeEntry := objects.TreeEntry{
			Mode: entry.Mode,
			Name: filepath.Base(entry.Path),
			ID:   entry.ID,
		}

		// For now, we only handle files in the root directory
		// A full implementation would need to handle subdirectories
		entries = append(entries, treeEntry)
	}

	return repo.CreateTree(entries)
}

func getSignature(authorStr string) (objects.Signature, error) {
	if authorStr != "" {
		// Parse author string (format: "Name <email>")
		parts := strings.Split(authorStr, " <")
		if len(parts) != 2 || !strings.HasSuffix(parts[1], ">") {
			return objects.Signature{}, fmt.Errorf("invalid author format, expected 'Name <email>'")
		}

		name := parts[0]
		email := strings.TrimSuffix(parts[1], ">")

		return objects.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		}, nil
	}

	// Use default signature
	return objects.Signature{
		Name:  getConfigValue("user.name", "VCS User"),
		Email: getConfigValue("user.email", "user@example.com"),
		When:  time.Now(),
	}, nil
}

func getConfigValue(key, defaultValue string) string {
	// For now, just return default values
	// A full implementation would read from .git/config
	return defaultValue
}

func getCurrentBranchName(refManager *refs.RefManager) string {
	branch, err := refManager.CurrentBranch()
	if err != nil {
		return "HEAD"
	}
	return branch
}