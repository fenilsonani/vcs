package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func newDiffCommand() *cobra.Command {
	var (
		cached     bool
		nameOnly   bool
		nameStatus bool
		unified    int
	)

	cmd := &cobra.Command{
		Use:   "diff [flags] [commit] [commit] [-- path...]",
		Short: "Show changes between commits, commit and working tree, etc",
		Long: `Show changes between the working tree and the index or a tree, changes between
the index and a tree, changes between two trees, or changes between two files.`,
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

			return runDiff(vcsRepo, refManager, args, cached, nameOnly, nameStatus, unified)
		},
	}

	cmd.Flags().BoolVar(&cached, "cached", false, "Show diff between index and HEAD")
	cmd.Flags().BoolVar(&nameOnly, "name-only", false, "Show only names of changed files")
	cmd.Flags().BoolVar(&nameStatus, "name-status", false, "Show names and status of changed files")
	cmd.Flags().IntVarP(&unified, "unified", "u", 3, "Number of context lines")

	return cmd
}

func runDiff(repo *vcs.Repository, refManager *refs.RefManager, args []string, cached, nameOnly, nameStatus bool, unified int) error {
	if cached {
		return diffIndexToHEAD(repo, refManager, nameOnly, nameStatus, unified)
	}

	switch len(args) {
	case 0:
		return diffWorkingTreeToIndex(repo, nameOnly, nameStatus, unified)
	case 1:
		return diffCommitToWorkingTree(repo, refManager, args[0], nameOnly, nameStatus, unified)
	case 2:
		return diffCommitToCommit(repo, refManager, args[0], args[1], nameOnly, nameStatus, unified)
	default:
		return fmt.Errorf("too many arguments")
	}
}

func diffWorkingTreeToIndex(repo *vcs.Repository, nameOnly, nameStatus bool, unified int) error {
	idx := index.New()
	indexPath := filepath.Join(repo.GitDir(), "index")
	
	if _, err := os.Stat(indexPath); err == nil {
		if err := idx.ReadFromFile(indexPath); err != nil {
			return fmt.Errorf("failed to read index: %w", err)
		}
	}

	// Get working tree files
	workingFiles := make(map[string]*WorkingFile)
	err := filepath.Walk(repo.WorkDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(repo.WorkDir(), path)
		if err != nil {
			return err
		}

		if strings.HasPrefix(relPath, ".git") || info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		blob := repo.CreateBlobDirect(content)
		workingFiles[relPath] = &WorkingFile{
			Path:    relPath,
			Content: content,
			ID:      blob.ID(),
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Compare working tree to index
	changes := make(map[string]*DiffChange)
	
	// Check files in index
	for _, entry := range idx.Entries() {
		if workingFile, exists := workingFiles[entry.Path]; exists {
			if !entry.ID.Equal(workingFile.ID) {
				// File modified
				changes[entry.Path] = &DiffChange{
					Path:      entry.Path,
					Type:      DiffModified,
					OldID:     entry.ID,
					NewID:     workingFile.ID,
					OldContent: getObjectContent(repo, entry.ID),
					NewContent: workingFile.Content,
				}
			}
		} else {
			// File deleted
			changes[entry.Path] = &DiffChange{
				Path:       entry.Path,
				Type:       DiffDeleted,
				OldID:      entry.ID,
				OldContent: getObjectContent(repo, entry.ID),
			}
		}
	}

	// Check for new files
	for path, workingFile := range workingFiles {
		if _, exists := idx.Get(path); !exists {
			changes[path] = &DiffChange{
				Path:       path,
				Type:       DiffAdded,
				NewID:      workingFile.ID,
				NewContent: workingFile.Content,
			}
		}
	}

	return printDiff(changes, nameOnly, nameStatus, unified)
}

func diffIndexToHEAD(repo *vcs.Repository, refManager *refs.RefManager, nameOnly, nameStatus bool, unified int) error {
	// Get HEAD commit
	headID, err := refManager.ResolveRef("HEAD")
	if err != nil {
		return fmt.Errorf("failed to resolve HEAD: %w", err)
	}

	headCommit, err := repo.GetCommit(headID)
	if err != nil {
		return fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	headTree, err := repo.GetTree(headCommit.Tree())
	if err != nil {
		return fmt.Errorf("failed to get HEAD tree: %w", err)
	}

	// Get index
	idx := index.New()
	indexPath := filepath.Join(repo.GitDir(), "index")
	
	if _, err := os.Stat(indexPath); err == nil {
		if err := idx.ReadFromFile(indexPath); err != nil {
			return fmt.Errorf("failed to read index: %w", err)
		}
	}

	return diffTreeToIndex(repo, headTree, idx, nameOnly, nameStatus, unified)
}

func diffCommitToWorkingTree(repo *vcs.Repository, refManager *refs.RefManager, commitRef string, nameOnly, nameStatus bool, unified int) error {
	commitID, err := refManager.ResolveRef(commitRef)
	if err != nil {
		return fmt.Errorf("failed to resolve ref %q: %w", commitRef, err)
	}

	commit, err := repo.GetCommit(commitID)
	if err != nil {
		return fmt.Errorf("failed to get commit: %w", err)
	}

	tree, err := repo.GetTree(commit.Tree())
	if err != nil {
		return fmt.Errorf("failed to get tree: %w", err)
	}

	return diffTreeToWorkingTree(repo, tree, nameOnly, nameStatus, unified)
}

func diffCommitToCommit(repo *vcs.Repository, refManager *refs.RefManager, commit1Ref, commit2Ref string, nameOnly, nameStatus bool, unified int) error {
	commit1ID, err := refManager.ResolveRef(commit1Ref)
	if err != nil {
		return fmt.Errorf("failed to resolve ref %q: %w", commit1Ref, err)
	}

	commit2ID, err := refManager.ResolveRef(commit2Ref)
	if err != nil {
		return fmt.Errorf("failed to resolve ref %q: %w", commit2Ref, err)
	}

	commit1, err := repo.GetCommit(commit1ID)
	if err != nil {
		return fmt.Errorf("failed to get commit1: %w", err)
	}

	commit2, err := repo.GetCommit(commit2ID)
	if err != nil {
		return fmt.Errorf("failed to get commit2: %w", err)
	}

	tree1, err := repo.GetTree(commit1.Tree())
	if err != nil {
		return fmt.Errorf("failed to get tree1: %w", err)
	}

	tree2, err := repo.GetTree(commit2.Tree())
	if err != nil {
		return fmt.Errorf("failed to get tree2: %w", err)
	}

	return diffTreeToTree(repo, tree1, tree2, nameOnly, nameStatus, unified)
}

func diffTreeToIndex(repo *vcs.Repository, tree *objects.Tree, idx *index.Index, nameOnly, nameStatus bool, unified int) error {
	changes := make(map[string]*DiffChange)
	
	// Get tree entries
	treeEntries := make(map[string]objects.TreeEntry)
	for _, entry := range tree.Entries() {
		treeEntries[entry.Name] = entry
	}

	// Compare tree to index
	for _, entry := range idx.Entries() {
		if treeEntry, exists := treeEntries[entry.Path]; exists {
			if !entry.ID.Equal(treeEntry.ID) {
				changes[entry.Path] = &DiffChange{
					Path:       entry.Path,
					Type:       DiffModified,
					OldID:      treeEntry.ID,
					NewID:      entry.ID,
					OldContent: getObjectContent(repo, treeEntry.ID),
					NewContent: getObjectContent(repo, entry.ID),
				}
			}
		} else {
			changes[entry.Path] = &DiffChange{
				Path:       entry.Path,
				Type:       DiffAdded,
				NewID:      entry.ID,
				NewContent: getObjectContent(repo, entry.ID),
			}
		}
	}

	// Check for deleted files
	for path, treeEntry := range treeEntries {
		if _, exists := idx.Get(path); !exists {
			changes[path] = &DiffChange{
				Path:       path,
				Type:       DiffDeleted,
				OldID:      treeEntry.ID,
				OldContent: getObjectContent(repo, treeEntry.ID),
			}
		}
	}

	return printDiff(changes, nameOnly, nameStatus, unified)
}

func diffTreeToWorkingTree(repo *vcs.Repository, tree *objects.Tree, nameOnly, nameStatus bool, unified int) error {
	// Get working tree files
	workingFiles := make(map[string]*WorkingFile)
	err := filepath.Walk(repo.WorkDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(repo.WorkDir(), path)
		if err != nil {
			return err
		}

		if strings.HasPrefix(relPath, ".git") || info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		blob := repo.CreateBlobDirect(content)
		workingFiles[relPath] = &WorkingFile{
			Path:    relPath,
			Content: content,
			ID:      blob.ID(),
		}
		return nil
	})
	if err != nil {
		return err
	}

	changes := make(map[string]*DiffChange)
	
	// Get tree entries
	treeEntries := make(map[string]objects.TreeEntry)
	for _, entry := range tree.Entries() {
		treeEntries[entry.Name] = entry
	}

	// Compare tree to working tree
	for path, workingFile := range workingFiles {
		if treeEntry, exists := treeEntries[path]; exists {
			if !treeEntry.ID.Equal(workingFile.ID) {
				changes[path] = &DiffChange{
					Path:       path,
					Type:       DiffModified,
					OldID:      treeEntry.ID,
					NewID:      workingFile.ID,
					OldContent: getObjectContent(repo, treeEntry.ID),
					NewContent: workingFile.Content,
				}
			}
		} else {
			changes[path] = &DiffChange{
				Path:       path,
				Type:       DiffAdded,
				NewID:      workingFile.ID,
				NewContent: workingFile.Content,
			}
		}
	}

	// Check for deleted files
	for path, treeEntry := range treeEntries {
		if _, exists := workingFiles[path]; !exists {
			changes[path] = &DiffChange{
				Path:       path,
				Type:       DiffDeleted,
				OldID:      treeEntry.ID,
				OldContent: getObjectContent(repo, treeEntry.ID),
			}
		}
	}

	return printDiff(changes, nameOnly, nameStatus, unified)
}

func diffTreeToTree(repo *vcs.Repository, tree1, tree2 *objects.Tree, nameOnly, nameStatus bool, unified int) error {
	changes := make(map[string]*DiffChange)
	
	// Get tree entries
	tree1Entries := make(map[string]objects.TreeEntry)
	for _, entry := range tree1.Entries() {
		tree1Entries[entry.Name] = entry
	}

	tree2Entries := make(map[string]objects.TreeEntry)
	for _, entry := range tree2.Entries() {
		tree2Entries[entry.Name] = entry
	}

	// All unique paths
	allPaths := make(map[string]bool)
	for path := range tree1Entries {
		allPaths[path] = true
	}
	for path := range tree2Entries {
		allPaths[path] = true
	}

	for path := range allPaths {
		entry1, exists1 := tree1Entries[path]
		entry2, exists2 := tree2Entries[path]

		if exists1 && exists2 {
			if !entry1.ID.Equal(entry2.ID) {
				changes[path] = &DiffChange{
					Path:       path,
					Type:       DiffModified,
					OldID:      entry1.ID,
					NewID:      entry2.ID,
					OldContent: getObjectContent(repo, entry1.ID),
					NewContent: getObjectContent(repo, entry2.ID),
				}
			}
		} else if exists1 && !exists2 {
			changes[path] = &DiffChange{
				Path:       path,
				Type:       DiffDeleted,
				OldID:      entry1.ID,
				OldContent: getObjectContent(repo, entry1.ID),
			}
		} else if !exists1 && exists2 {
			changes[path] = &DiffChange{
				Path:       path,
				Type:       DiffAdded,
				NewID:      entry2.ID,
				NewContent: getObjectContent(repo, entry2.ID),
			}
		}
	}

	return printDiff(changes, nameOnly, nameStatus, unified)
}

type DiffType int

const (
	DiffAdded DiffType = iota
	DiffModified
	DiffDeleted
)

type DiffChange struct {
	Path       string
	Type       DiffType
	OldID      objects.ObjectID
	NewID      objects.ObjectID
	OldContent []byte
	NewContent []byte
}

type WorkingFile struct {
	Path    string
	Content []byte
	ID      objects.ObjectID
}

func getObjectContent(repo *vcs.Repository, id objects.ObjectID) []byte {
	if id.IsZero() {
		return nil
	}
	
	obj, err := repo.GetObject(id)
	if err != nil {
		return nil
	}
	
	blob, ok := obj.(*objects.Blob)
	if !ok {
		return nil
	}
	
	return blob.Data()
}

func printDiff(changes map[string]*DiffChange, nameOnly, nameStatus bool, unified int) error {
	if len(changes) == 0 {
		return nil
	}

	// Sort paths for consistent output
	paths := make([]string, 0, len(changes))
	for path := range changes {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	if nameOnly {
		for _, path := range paths {
			fmt.Println(path)
		}
		return nil
	}

	if nameStatus {
		for _, path := range paths {
			change := changes[path]
			var status string
			switch change.Type {
			case DiffAdded:
				status = "A"
			case DiffModified:
				status = "M"
			case DiffDeleted:
				status = "D"
			}
			fmt.Printf("%s\t%s\n", status, path)
		}
		return nil
	}

	// Full diff output
	for _, path := range paths {
		change := changes[path]
		
		switch change.Type {
		case DiffAdded:
			fmt.Printf("diff --git a/%s b/%s\n", path, path)
			fmt.Println("new file mode 100644")
			fmt.Printf("index 0000000..%s\n", change.NewID.String()[:7])
			fmt.Println("--- /dev/null")
			fmt.Printf("+++ b/%s\n", path)
			printUnifiedDiff(nil, change.NewContent, unified)
		case DiffDeleted:
			fmt.Printf("diff --git a/%s b/%s\n", path, path)
			fmt.Println("deleted file mode 100644")
			fmt.Printf("index %s..0000000\n", change.OldID.String()[:7])
			fmt.Printf("--- a/%s\n", path)
			fmt.Println("+++ /dev/null")
			printUnifiedDiff(change.OldContent, nil, unified)
		case DiffModified:
			fmt.Printf("diff --git a/%s b/%s\n", path, path)
			fmt.Printf("index %s..%s 100644\n", change.OldID.String()[:7], change.NewID.String()[:7])
			fmt.Printf("--- a/%s\n", path)
			fmt.Printf("+++ b/%s\n", path)
			printUnifiedDiff(change.OldContent, change.NewContent, unified)
		}
		fmt.Println()
	}

	return nil
}

func printUnifiedDiff(oldContent, newContent []byte, contextLines int) {
	oldLines := strings.Split(string(oldContent), "\n")
	newLines := strings.Split(string(newContent), "\n")
	
	if len(oldLines) == 1 && oldLines[0] == "" {
		oldLines = nil
	}
	if len(newLines) == 1 && newLines[0] == "" {
		newLines = nil
	}

	// Simple line-by-line diff
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	if maxLen == 0 {
		return
	}

	// Find first and last different lines
	firstDiff := -1
	lastDiff := -1
	
	for i := 0; i < maxLen; i++ {
		oldLine := ""
		newLine := ""
		
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}
		
		if oldLine != newLine {
			if firstDiff == -1 {
				firstDiff = i
			}
			lastDiff = i
		}
	}

	if firstDiff == -1 {
		return // No differences
	}

	// Calculate hunk boundaries
	hunkStart := firstDiff - contextLines
	if hunkStart < 0 {
		hunkStart = 0
	}
	
	hunkEnd := lastDiff + contextLines
	if hunkEnd >= maxLen {
		hunkEnd = maxLen - 1
	}

	// Count old and new lines in hunk
	oldCount := 0
	newCount := 0
	
	for i := hunkStart; i <= hunkEnd; i++ {
		if i < len(oldLines) {
			oldCount++
		}
		if i < len(newLines) {
			newCount++
		}
	}

	// Print hunk header
	fmt.Printf("@@ -%d,%d +%d,%d @@\n", hunkStart+1, oldCount, hunkStart+1, newCount)

	// Print hunk content
	for i := hunkStart; i <= hunkEnd; i++ {
		oldLine := ""
		newLine := ""
		
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine == newLine {
			fmt.Printf(" %s\n", oldLine)
		} else {
			if i < len(oldLines) {
				fmt.Printf("-%s\n", oldLine)
			}
			if i < len(newLines) {
				fmt.Printf("+%s\n", newLine)
			}
		}
	}
}