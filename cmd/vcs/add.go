package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/workdir"
	"github.com/fenilsonani/vcs/pkg/vcs"
	"github.com/spf13/cobra"
)

func newAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [flags] [pathspec...]",
		Short: "Add file contents to the index",
		Long: `Updates the index using the current content found in the working tree, 
to prepare the content staged for the next commit.`,
		RunE: runAdd,
	}

	cmd.Flags().BoolP("all", "A", false, "Add changes from all tracked and untracked files")
	cmd.Flags().Bool("update", false, "Update the index just where it already has an entry matching pathspec")
	cmd.Flags().BoolP("force", "f", false, "Allow adding otherwise ignored files")
	cmd.Flags().BoolP("dry-run", "n", false, "Don't actually add the file(s), just show if they exist and/or will be ignored")
	cmd.Flags().BoolP("verbose", "v", false, "Be verbose")

	return cmd
}

func runAdd(cmd *cobra.Command, args []string) error {
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
	addAll, _ := cmd.Flags().GetBool("all")
	updateOnly, _ := cmd.Flags().GetBool("update")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Get index
	idx := index.New()
	indexPath := filepath.Join(repo.GitDir(), "index")
	if _, err := os.Stat(indexPath); err == nil {
		if err := idx.ReadFromFile(indexPath); err != nil {
			// If we can't read the index file, start with empty index
			idx = index.New()
		}
	}

	// Create scanner for working directory
	scanner := workdir.NewScanner(repoPath, repo.GitDir())
	
	// Load .gitignore file if it exists
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	scanner.LoadIgnoreFile(gitignorePath)

	var pathsToAdd []string

	if addAll {
		// Add all files
		files, err := scanner.ScanFiles()
		if err != nil {
			return fmt.Errorf("failed to scan working directory: %w", err)
		}

		for _, file := range files {
			if !force && scanner.IsIgnored(file.Path) {
				continue
			}
			pathsToAdd = append(pathsToAdd, file.Path)
		}
	} else if len(args) == 0 {
		return fmt.Errorf("nothing specified, nothing added")
	} else {
		// Add specified paths
		for _, arg := range args {
			paths, err := expandPath(repoPath, arg)
			if err != nil {
				return fmt.Errorf("failed to expand path %s: %w", arg, err)
			}
			pathsToAdd = append(pathsToAdd, paths...)
		}
	}

	// Process each path
	modified := false
	for _, path := range pathsToAdd {
		// Convert to relative path from repo root
		absPath := filepath.Join(repoPath, path)
		relPath, err := filepath.Rel(repoPath, absPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}
		
		// Normalize path separators
		relPath = filepath.ToSlash(relPath)

		// Check if file exists
		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				// File doesn't exist, check if it's in the index to remove it
				if _, exists := idx.Get(relPath); exists {
					if !dryRun {
						idx.Remove(relPath)
						modified = true
					}
					if verbose {
						fmt.Printf("remove '%s'\n", relPath)
					}
				}
				continue
			}
			return fmt.Errorf("failed to stat file %s: %w", path, err)
		}

		// Skip directories
		if info.IsDir() {
			continue
		}

		// Check if file is ignored
		if !force && scanner.IsIgnored(relPath) {
			if verbose {
				fmt.Printf("The following paths are ignored by one of your .gitignore files:\n%s\n", relPath)
			}
			continue
		}

		// Check update-only mode
		if updateOnly {
			if _, exists := idx.Get(relPath); !exists {
				continue // Skip files not already in index
			}
		}

		if dryRun {
			fmt.Printf("add '%s'\n", relPath)
			continue
		}

		// Read file content
		content, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Get file mode
		fileMode, err := scanner.GetFileMode(relPath)
		if err != nil {
			return fmt.Errorf("failed to get file mode for %s: %w", relPath, err)
		}

		// Create blob and write to object store
		blob := objects.NewBlob(content)
		if err := repo.WriteObject(blob); err != nil {
			return fmt.Errorf("failed to write blob for %s: %w", relPath, err)
		}

		// Create index entry
		entry := &index.Entry{
			CTime: info.ModTime(),
			MTime: info.ModTime(),
			Dev:   0, // Not used in our implementation
			Ino:   0, // Not used in our implementation
			Mode:  fileMode,
			UID:   0, // Not used in our implementation
			GID:   0, // Not used in our implementation
			Size:  uint32(info.Size()),
			ID:    blob.ID(),
			Flags: 0,
			Path:  relPath,
		}

		// Add to index
		if err := idx.Add(entry); err != nil {
			return fmt.Errorf("failed to add entry to index: %w", err)
		}

		modified = true

		if verbose {
			fmt.Printf("add '%s'\n", relPath)
		}
	}

	// Write index if modified and not dry run
	if modified && !dryRun {
		if err := idx.WriteToFile(indexPath); err != nil {
			return fmt.Errorf("failed to write index: %w", err)
		}
	}

	return nil
}

// expandPath expands a path pattern to matching files
func expandPath(repoPath, pattern string) ([]string, error) {
	var paths []string

	// Handle absolute paths
	if filepath.IsAbs(pattern) {
		relPath, err := filepath.Rel(repoPath, pattern)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(relPath, "..") {
			return nil, fmt.Errorf("path outside repository: %s", pattern)
		}
		pattern = relPath
	}

	// Check if pattern contains wildcards
	if strings.ContainsAny(pattern, "*?[") {
		// Use glob matching
		fullPattern := filepath.Join(repoPath, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, err
		}

		for _, match := range matches {
			relPath, err := filepath.Rel(repoPath, match)
			if err != nil {
				continue
			}
			paths = append(paths, relPath)
		}
	} else {
		// Direct path
		fullPath := filepath.Join(repoPath, pattern)
		if _, err := os.Stat(fullPath); err != nil {
			if os.IsNotExist(err) {
				// Path doesn't exist, might be for removal
				paths = append(paths, pattern)
			} else {
				return nil, err
			}
		} else {
			paths = append(paths, pattern)
		}
	}

	return paths, nil
}