package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/workdir"
	"github.com/fenilsonani/vcs/pkg/vcs"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show working tree status",
		Long: `Shows paths that have differences between the index file and the current HEAD commit,
paths that have differences between the working tree and the index file, and paths in the
working tree that are not tracked by Git.`,
		RunE: runStatus,
	}

	cmd.Flags().BoolP("short", "s", false, "Give the output in the short-format")
	cmd.Flags().Bool("porcelain", false, "Give the output in an easy-to-parse format for scripts")
	cmd.Flags().Bool("ignored", false, "Show ignored files as well")

	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
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
	shortFormat, _ := cmd.Flags().GetBool("short")
	porcelain, _ := cmd.Flags().GetBool("porcelain")
	showIgnored, _ := cmd.Flags().GetBool("ignored")

	// Create scanner for working directory
	scanner := workdir.NewScanner(repoPath, repo.GitDir())
	
	// Load .gitignore file if it exists
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	scanner.LoadIgnoreFile(gitignorePath)

	// Get index
	idx := index.New()
	indexPath := filepath.Join(repo.GitDir(), "index")
	if _, err := os.Stat(indexPath); err == nil {
		if err := idx.ReadFromFile(indexPath); err != nil {
			// If we can't read the index file, assume it's empty (e.g., different format)
			idx = index.New()
		}
	}

	// Scan working directory files
	files, err := scanner.ScanFiles()
	if err != nil {
		return fmt.Errorf("failed to scan working directory: %w", err)
	}

	// Analyze file statuses
	statusMap := make(map[string]*FileStatusInfo)

	// Check for staged files
	for _, entry := range idx.Entries() {
		statusMap[entry.Path] = &FileStatusInfo{
			Path:        entry.Path,
			IndexStatus: StatusStaged,
			WorkStatus:  StatusUnmodified,
		}
	}

	// Check working directory files
	for _, file := range files {
		if scanner.IsIgnored(file.Path) {
			if showIgnored {
				statusMap[file.Path] = &FileStatusInfo{
					Path:        file.Path,
					IndexStatus: StatusUnmodified,
					WorkStatus:  StatusIgnored,
				}
			}
			continue
		}

		entry, exists := idx.Get(file.Path)
		if !exists {
			// Untracked file
			statusMap[file.Path] = &FileStatusInfo{
				Path:        file.Path,
				IndexStatus: StatusUnmodified,
				WorkStatus:  StatusUntracked,
			}
		} else {
			// Check if file is modified
			content, err := scanner.GetFileContent(file.Path)
			if err != nil {
				continue
			}

			// Compare with index
			currentHash := repo.HashData(content)
			if currentHash != entry.ID {
				if existing, exists := statusMap[file.Path]; exists {
					existing.WorkStatus = StatusModified
				} else {
					statusMap[file.Path] = &FileStatusInfo{
						Path:        file.Path,
						IndexStatus: StatusUnmodified,
						WorkStatus:  StatusModified,
					}
				}
			}
		}
	}

	// Check for deleted files (in index but not in working directory)
	workFileMap := make(map[string]bool)
	for _, file := range files {
		workFileMap[file.Path] = true
	}

	for _, entry := range idx.Entries() {
		if !workFileMap[entry.Path] {
			statusMap[entry.Path] = &FileStatusInfo{
				Path:        entry.Path,
				IndexStatus: StatusStaged,
				WorkStatus:  StatusDeleted,
			}
		}
	}

	// Sort files for consistent output
	var sortedFiles []string
	for path := range statusMap {
		sortedFiles = append(sortedFiles, path)
	}
	sort.Strings(sortedFiles)

	// Output results
	if shortFormat || porcelain {
		printShortStatus(sortedFiles, statusMap)
	} else {
		printLongStatus(sortedFiles, statusMap)
	}

	return nil
}

type FileStatusInfo struct {
	Path        string
	IndexStatus FileStatus
	WorkStatus  FileStatus
}

type FileStatus int

const (
	StatusUnmodified FileStatus = iota
	StatusStaged
	StatusModified
	StatusUntracked
	StatusDeleted
	StatusIgnored
)

func (s FileStatus) IndexChar() string {
	switch s {
	case StatusStaged:
		return "A"
	case StatusModified:
		return "M"
	case StatusDeleted:
		return "D"
	default:
		return " "
	}
}

func (s FileStatus) WorkChar() string {
	switch s {
	case StatusModified:
		return "M"
	case StatusDeleted:
		return "D"
	case StatusUntracked:
		return "?"
	case StatusIgnored:
		return "!"
	default:
		return " "
	}
}

func printShortStatus(sortedFiles []string, statusMap map[string]*FileStatusInfo) {
	for _, path := range sortedFiles {
		status := statusMap[path]
		indexChar := status.IndexStatus.IndexChar()
		workChar := status.WorkStatus.WorkChar()
		
		if indexChar == " " && workChar == " " {
			continue // Skip unmodified files
		}
		
		fmt.Printf("%s%s %s\n", indexChar, workChar, path)
	}
}

func printLongStatus(sortedFiles []string, statusMap map[string]*FileStatusInfo) {
	var staged []string
	var modified []string
	var untracked []string
	var deleted []string
	var ignored []string

	for _, path := range sortedFiles {
		status := statusMap[path]
		
		switch {
		case status.IndexStatus == StatusStaged && status.WorkStatus == StatusUnmodified:
			staged = append(staged, path)
		case status.IndexStatus == StatusStaged && status.WorkStatus == StatusDeleted:
			deleted = append(deleted, path)
		case status.WorkStatus == StatusModified:
			modified = append(modified, path)
		case status.WorkStatus == StatusUntracked:
			untracked = append(untracked, path)
		case status.WorkStatus == StatusIgnored:
			ignored = append(ignored, path)
		}
	}

	// Print status sections
	if len(staged) > 0 {
		fmt.Println("Changes to be committed:")
		for _, path := range staged {
			fmt.Printf("  new file:   %s\n", path)
		}
		fmt.Println()
	}

	if len(modified) > 0 {
		fmt.Println("Changes not staged for commit:")
		for _, path := range modified {
			fmt.Printf("  modified:   %s\n", path)
		}
		fmt.Println()
	}

	if len(deleted) > 0 {
		fmt.Println("Changes not staged for commit:")
		for _, path := range deleted {
			fmt.Printf("  deleted:    %s\n", path)
		}
		fmt.Println()
	}

	if len(untracked) > 0 {
		fmt.Println("Untracked files:")
		for _, path := range untracked {
			fmt.Printf("  %s\n", path)
		}
		fmt.Println()
	}

	if len(ignored) > 0 {
		fmt.Println("Ignored files:")
		for _, path := range ignored {
			fmt.Printf("  %s\n", path)
		}
		fmt.Println()
	}

	// Print status summary
	if len(staged) == 0 && len(modified) == 0 && len(untracked) == 0 {
		fmt.Println("nothing to commit, working tree clean")
	}
}

func findRepository() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up directory tree looking for .git
	dir := cwd
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}

	return "", fmt.Errorf("not a git repository")
}