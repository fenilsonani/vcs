package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func newResetCommand() *cobra.Command {
	var (
		soft  bool
		mixed bool
		hard  bool
	)

	cmd := &cobra.Command{
		Use:   "reset [flags] [<commit>] [-- <paths>...]",
		Short: "Reset current HEAD to the specified state",
		Long: `Resets the index and optionally the working tree to match the specified commit.
The three modes are:
--soft: Only moves HEAD pointer
--mixed: Moves HEAD and resets index (default)
--hard: Moves HEAD, resets index, and working tree`,
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

			// Determine reset mode
			mode := ResetMixed // default
			if soft {
				mode = ResetSoft
			} else if hard {
				mode = ResetHard
			}

			// Determine target commit
			target := "HEAD"
			if len(args) > 0 {
				target = args[0]
			}

			return runReset(vcsRepo, refManager, target, mode)
		},
	}

	cmd.Flags().BoolVar(&soft, "soft", false, "Only move HEAD pointer")
	cmd.Flags().BoolVar(&mixed, "mixed", false, "Move HEAD and reset index (default)")
	cmd.Flags().BoolVar(&hard, "hard", false, "Move HEAD, reset index and working tree")

	return cmd
}

type ResetMode int

const (
	ResetSoft ResetMode = iota
	ResetMixed
	ResetHard
)

func runReset(repo *vcs.Repository, refManager *refs.RefManager, target string, mode ResetMode) error {
	// Resolve target commit
	targetID, err := refManager.ResolveRef(target)
	if err != nil {
		return fmt.Errorf("failed to resolve %q: %w", target, err)
	}

	targetCommit, err := repo.GetCommit(targetID)
	if err != nil {
		return fmt.Errorf("failed to get commit %s: %w", targetID.Short(), err)
	}

	// Get current branch
	currentBranch, err := refManager.CurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Update HEAD to point to target commit
	currentRef := "refs/heads/" + currentBranch
	if err := refManager.WriteRef(currentRef, targetID, nil); err != nil {
		return fmt.Errorf("failed to update %s: %w", currentRef, err)
	}

	switch mode {
	case ResetSoft:
		fmt.Printf("HEAD is now at %s\n", targetID.Short())
		return nil

	case ResetMixed:
		// Reset index to match target commit
		if err := resetIndex(repo, targetCommit); err != nil {
			return fmt.Errorf("failed to reset index: %w", err)
		}
		fmt.Printf("Unstaged changes after reset:\n")
		// Show what files are now modified in working tree
		return showUnstagedChanges(repo)

	case ResetHard:
		// Reset index and working tree
		if err := resetIndex(repo, targetCommit); err != nil {
			return fmt.Errorf("failed to reset index: %w", err)
		}
		if err := resetWorkingTree(repo, targetCommit); err != nil {
			return fmt.Errorf("failed to reset working tree: %w", err)
		}
		fmt.Printf("HEAD is now at %s %s\n", targetID.Short(), getCommitSubject(targetCommit))
		return nil

	default:
		return fmt.Errorf("unknown reset mode")
	}
}

func resetIndex(repo *vcs.Repository, commit *objects.Commit) error {
	// Get the tree from commit
	tree, err := repo.GetTree(commit.Tree())
	if err != nil {
		return fmt.Errorf("failed to get tree: %w", err)
	}

	// Create new index from tree
	idx := index.New()
	if err := populateIndexFromTree(repo, idx, tree, ""); err != nil {
		return fmt.Errorf("failed to populate index: %w", err)
	}

	// Write index
	indexPath := filepath.Join(repo.GitDir(), "index")
	if err := idx.WriteToFile(indexPath); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	return nil
}

func resetWorkingTree(repo *vcs.Repository, commit *objects.Commit) error {
	// Get the tree from commit
	tree, err := repo.GetTree(commit.Tree())
	if err != nil {
		return fmt.Errorf("failed to get tree: %w", err)
	}

	// Remove all files in working directory (except .git)
	if err := clearWorkingDirectory(repo); err != nil {
		return fmt.Errorf("failed to clear working directory: %w", err)
	}

	// Extract all files from tree
	return extractTreeToWorkingDirectory(repo, tree, repo.WorkDir())
}

func populateIndexFromTree(repo *vcs.Repository, idx *index.Index, tree *objects.Tree, prefix string) error {
	for _, entry := range tree.Entries() {
		fullPath := filepath.Join(prefix, entry.Name)
		
		if entry.Mode == objects.ModeTree {
			// Recursively handle subtree
			subtree, err := repo.GetTree(entry.ID)
			if err != nil {
				return fmt.Errorf("failed to get subtree %s: %w", entry.ID.Short(), err)
			}
			if err := populateIndexFromTree(repo, idx, subtree, fullPath); err != nil {
				return err
			}
		} else {
			// Add file to index
			blob, err := repo.GetBlob(entry.ID)
			if err != nil {
				return fmt.Errorf("failed to get blob %s: %w", entry.ID.Short(), err)
			}

			indexEntry := &index.Entry{
				Mode: entry.Mode,
				Size: uint32(len(blob.Data())),
				ID:   entry.ID,
				Path: fullPath,
			}
			
			if err := idx.Add(indexEntry); err != nil {
				return fmt.Errorf("failed to add entry to index: %w", err)
			}
		}
	}
	return nil
}

func clearWorkingDirectory(repo *vcs.Repository) error {
	// Walk through working directory and remove all files except .git
	return filepath.Walk(repo.WorkDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(repo.WorkDir(), path)
		if err != nil {
			return err
		}

		// Skip .git directory and its contents
		if relPath == ".git" || filepath.HasPrefix(relPath, ".git"+string(filepath.Separator)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Remove file
		if !info.IsDir() {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove %s: %w", path, err)
			}
		}

		return nil
	})
}

func extractTreeToWorkingDirectory(repo *vcs.Repository, tree *objects.Tree, basePath string) error {
	for _, entry := range tree.Entries() {
		fullPath := filepath.Join(basePath, entry.Name)
		
		if entry.Mode == objects.ModeTree {
			// Create directory and recursively extract subtree
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
			}
			
			subtree, err := repo.GetTree(entry.ID)
			if err != nil {
				return fmt.Errorf("failed to get subtree %s: %w", entry.ID.Short(), err)
			}
			
			if err := extractTreeToWorkingDirectory(repo, subtree, fullPath); err != nil {
				return err
			}
		} else {
			// Extract file
			blob, err := repo.GetBlob(entry.ID)
			if err != nil {
				return fmt.Errorf("failed to get blob %s: %w", entry.ID.Short(), err)
			}

			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", fullPath, err)
			}

			// Write file
			var fileMode os.FileMode = 0644
			if entry.Mode == objects.ModeExec {
				fileMode = 0755
			}

			if err := os.WriteFile(fullPath, blob.Data(), fileMode); err != nil {
				return fmt.Errorf("failed to write file %s: %w", fullPath, err)
			}
		}
	}
	return nil
}

func showUnstagedChanges(repo *vcs.Repository) error {
	// This is a simplified version - in reality would show detailed diff
	fmt.Println("(use \"git add <file>...\" to stage changes)")
	return nil
}

func getCommitSubject(commit *objects.Commit) string {
	message := commit.Message()
	if idx := len(message); idx > 50 {
		return message[:50] + "..."
	}
	return message
}