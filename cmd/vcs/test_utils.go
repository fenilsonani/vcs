package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"time"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

// TestRepository wraps vcs.Repository and adds test helper methods
type TestRepository struct {
	*vcs.Repository
	path string
}

// WrapRepository wraps a vcs.Repository for testing
func WrapRepository(repo *vcs.Repository, path string) *TestRepository {
	return &TestRepository{
		Repository: repo,
		path:       path,
	}
}

// Add adds files to the repository index (test helper)
func (r *TestRepository) Add(pathspec string) error {
	// Get repository directories
	gitDir := r.GitDir()
	repoPath := filepath.Dir(gitDir)
	
	// Create or load index
	indexPath := filepath.Join(gitDir, "index")
	idx := index.New()
	
	// Load existing index if it exists
	if _, err := os.Stat(indexPath); err == nil {
		if err := idx.ReadFromFile(indexPath); err != nil {
			return fmt.Errorf("failed to read index: %w", err)
		}
	}
	
	// Create storage
	storage := objects.NewStorage(filepath.Join(gitDir, "objects"))
	
	// For simple test case, just add the specific file
	if pathspec != "" && pathspec != "." {
		// Read file content
		content, err := os.ReadFile(filepath.Join(repoPath, pathspec))
		if err != nil {
			return err
		}
		
		// Create blob
		blob := objects.NewBlob(content)
		err = storage.WriteObject(blob)
		if err != nil {
			return err
		}
		blobID := blob.ID()
		
		// Add to index
		entry := &index.Entry{
			Mode: 0100644, // Regular file
			Path: pathspec,
			ID:   blobID,
			Size: uint32(len(content)),
		}
		idx.Add(entry)
	}
	
	// Write index
	return idx.WriteToFile(indexPath)
}

// Commit creates a commit (test helper)
func (r *TestRepository) Commit(message, authorName, authorEmail string) (objects.ObjectID, error) {
	gitDir := r.GitDir()
	
	// Load index
	indexPath := filepath.Join(gitDir, "index")
	idx := index.New()
	err := idx.ReadFromFile(indexPath)
	if err != nil {
		return objects.ObjectID{}, fmt.Errorf("failed to read index: %w", err)
	}
	
	// Create tree from index
	storage := objects.NewStorage(filepath.Join(gitDir, "objects"))
	tree := objects.NewTree()
	
	for _, entry := range idx.Entries() {
		err = tree.AddEntry(objects.FileMode(entry.Mode), entry.Path, entry.ID)
		if err != nil {
			return objects.ObjectID{}, fmt.Errorf("failed to add tree entry: %w", err)
		}
	}
	
	// Write tree object
	err = storage.WriteObject(tree)
	if err != nil {
		return objects.ObjectID{}, fmt.Errorf("failed to write tree: %w", err)
	}
	treeID := tree.ID()
	
	// Get parent commit
	refManager := refs.NewRefManager(gitDir)
	var parentID *objects.ObjectID
	
	headCommitID, _, err := refManager.HEAD()
	if err == nil {
		parentID = &headCommitID
	}
	
	// Create commit
	var parents []objects.ObjectID
	if parentID != nil {
		parents = append(parents, *parentID)
	}
	
	// Create signatures
	now := time.Now()
	author := objects.Signature{
		Name:  authorName,
		Email: authorEmail,
		When:  now,
	}
	committer := author
	
	commit := objects.NewCommit(treeID, parents, author, committer, message)
	err = storage.WriteObject(commit)
	if err != nil {
		return objects.ObjectID{}, fmt.Errorf("failed to write commit: %w", err)
	}
	commitID := commit.ID()
	
	// Update HEAD
	currentBranch, err := refManager.CurrentBranch()
	if err != nil {
		// Create main branch if no current branch
		currentBranch = "main"
		if err := refManager.UpdateRef(fmt.Sprintf("refs/heads/%s", currentBranch), commitID); err != nil {
			return objects.ObjectID{}, fmt.Errorf("failed to update branch: %w", err)
		}
		
		// Update HEAD to point to main
		headPath := filepath.Join(gitDir, "HEAD")
		headContent := fmt.Sprintf("ref: refs/heads/%s\n", currentBranch)
		if err := os.WriteFile(headPath, []byte(headContent), 0644); err != nil {
			return objects.ObjectID{}, fmt.Errorf("failed to update HEAD: %w", err)
		}
	} else {
		// Update current branch
		if err := refManager.UpdateRef(fmt.Sprintf("refs/heads/%s", currentBranch), commitID); err != nil {
			return objects.ObjectID{}, fmt.Errorf("failed to update branch: %w", err)
		}
	}
	
	return commitID, nil
}

// CreateBranch creates a new branch (test helper)
func (r *TestRepository) CreateBranch(name string) (objects.ObjectID, error) {
	gitDir := r.GitDir()
	refManager := refs.NewRefManager(gitDir)
	
	// Get current HEAD
	headCommitID, _, err := refManager.HEAD()
	if err != nil {
		return objects.ObjectID{}, fmt.Errorf("failed to get HEAD: %w", err)
	}
	
	// Create branch ref
	branchRef := fmt.Sprintf("refs/heads/%s", name)
	if err := refManager.UpdateRef(branchRef, headCommitID); err != nil {
		return objects.ObjectID{}, fmt.Errorf("failed to create branch: %w", err)
	}
	
	return headCommitID, nil
}

// Checkout switches to a branch or commit (test helper)
func (r *TestRepository) Checkout(target string) error {
	gitDir := r.GitDir()
	refManager := refs.NewRefManager(gitDir)
	
	// Check if target is a branch
	branchRef := fmt.Sprintf("refs/heads/%s", target)
	commitID, err := refManager.ResolveRef(branchRef)
	if err == nil {
		// It's a branch - update HEAD to point to it
		headPath := filepath.Join(gitDir, "HEAD")
		headContent := fmt.Sprintf("ref: %s\n", branchRef)
		return os.WriteFile(headPath, []byte(headContent), 0644)
	}
	
	// Try to parse as commit ID
	commitID, err = objects.ParseObjectID(target)
	if err != nil {
		return fmt.Errorf("invalid target: %s", target)
	}
	
	// Detached HEAD - write commit ID directly
	headPath := filepath.Join(gitDir, "HEAD")
	return os.WriteFile(headPath, []byte(commitID.String()+"\n"), 0644)
}

// Log returns commit history (test helper)
func (r *TestRepository) Log(limit int) ([]*objects.Commit, error) {
	gitDir := r.GitDir()
	refManager := refs.NewRefManager(gitDir)
	storage := objects.NewStorage(filepath.Join(gitDir, "objects"))
	
	// Get HEAD commit
	headCommitID, _, err := refManager.HEAD()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	
	// Walk commit history
	var commits []*objects.Commit
	currentID := headCommitID
	
	for len(commits) < limit {
		// Read commit object
		obj, err := storage.ReadObject(currentID)
		if err != nil {
			break
		}
		
		commit, ok := obj.(*objects.Commit)
		if !ok {
			break
		}
		
		commits = append(commits, commit)
		
		// Get parent
		parents := commit.Parents()
		if len(parents) == 0 {
			break
		}
		currentID = parents[0]
	}
	
	return commits, nil
}

// parseConfig parses git config format (test helper for fetch_test.go)
func parseConfig(data []byte) map[string]string {
	remotes := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	
	var currentRemote string
	inRemoteSection := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Check for section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := line[1 : len(line)-1]
			parts := strings.Fields(section)
			
			if len(parts) >= 2 && parts[0] == "remote" {
				// Extract remote name from quotes
				remoteName := strings.Trim(parts[1], "\"")
				currentRemote = remoteName
				inRemoteSection = true
			} else {
				inRemoteSection = false
			}
			continue
		}
		
		// Parse key-value pairs in remote section
		if inRemoteSection && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				
				if key == "url" && currentRemote != "" {
					remotes[currentRemote] = value
				}
			}
		}
	}
	
	return remotes
}