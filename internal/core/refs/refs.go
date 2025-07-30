package refs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fenilsonani/vcs/internal/core/objects"
)

// RefManager manages Git references (branches, tags, HEAD)
type RefManager struct {
	gitDir string
}

// NewRefManager creates a new reference manager
func NewRefManager(gitDir string) *RefManager {
	return &RefManager{
		gitDir: gitDir,
	}
}

// HEAD returns the current HEAD reference
func (rm *RefManager) HEAD() (objects.ObjectID, string, error) {
	headPath := filepath.Join(rm.gitDir, "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		return objects.ObjectID{}, "", fmt.Errorf("failed to read HEAD: %w", err)
	}

	headStr := strings.TrimSpace(string(content))
	
	// Check if HEAD points to a reference
	if strings.HasPrefix(headStr, "ref: ") {
		refName := strings.TrimPrefix(headStr, "ref: ")
		id, err := rm.ResolveRef(refName)
		return id, refName, err
	}
	
	// HEAD points directly to an object
	id, err := objects.NewObjectID(headStr)
	return id, "", err
}

// SetHEAD sets the HEAD reference
func (rm *RefManager) SetHEAD(refName string) error {
	headPath := filepath.Join(rm.gitDir, "HEAD")
	content := fmt.Sprintf("ref: %s\n", refName)
	return os.WriteFile(headPath, []byte(content), 0644)
}

// SetHEADToCommit sets HEAD to point directly to a commit
func (rm *RefManager) SetHEADToCommit(commitID objects.ObjectID) error {
	headPath := filepath.Join(rm.gitDir, "HEAD")
	content := fmt.Sprintf("%s\n", commitID.String())
	return os.WriteFile(headPath, []byte(content), 0644)
}

// ResolveRef resolves a reference name to an object ID
func (rm *RefManager) ResolveRef(refName string) (objects.ObjectID, error) {
	// Try exact match first
	if id, err := rm.readRefFile(refName); err == nil {
		return id, nil
	}
	
	// Try common prefixes
	prefixes := []string{
		"refs/",
		"refs/heads/",
		"refs/tags/",
		"refs/remotes/",
		"refs/remotes/origin/",
	}
	
	for _, prefix := range prefixes {
		fullRef := prefix + refName
		if id, err := rm.readRefFile(fullRef); err == nil {
			return id, nil
		}
	}
	
	return objects.ObjectID{}, fmt.Errorf("reference not found: %s", refName)
}

// readRefFile reads a reference file and returns the object ID
func (rm *RefManager) readRefFile(refName string) (objects.ObjectID, error) {
	refPath := filepath.Join(rm.gitDir, refName)
	content, err := os.ReadFile(refPath)
	if err != nil {
		return objects.ObjectID{}, err
	}
	
	refStr := strings.TrimSpace(string(content))
	
	// Handle symbolic references
	if strings.HasPrefix(refStr, "ref: ") {
		targetRef := strings.TrimPrefix(refStr, "ref: ")
		return rm.ResolveRef(targetRef)
	}
	
	// Direct object reference
	return objects.NewObjectID(refStr)
}

// UpdateRef updates a reference to point to an object
func (rm *RefManager) UpdateRef(refName string, id objects.ObjectID) error {
	refPath := filepath.Join(rm.gitDir, refName)
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(refPath), 0755); err != nil {
		return fmt.Errorf("failed to create ref directory: %w", err)
	}
	
	content := fmt.Sprintf("%s\n", id.String())
	return os.WriteFile(refPath, []byte(content), 0644)
}

// ListBranches returns all local branches
func (rm *RefManager) ListBranches() ([]string, error) {
	branchesDir := filepath.Join(rm.gitDir, "refs", "heads")
	return rm.listRefs(branchesDir, "refs/heads/")
}

// ListTags returns all tags
func (rm *RefManager) ListTags() ([]string, error) {
	tagsDir := filepath.Join(rm.gitDir, "refs", "tags")
	return rm.listRefs(tagsDir, "refs/tags/")
}

// listRefs lists all references in a directory
func (rm *RefManager) listRefs(dir, prefix string) ([]string, error) {
	var refs []string
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Ignore missing directories
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		
		if !info.IsDir() {
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			refs = append(refs, prefix+filepath.ToSlash(relPath))
		}
		
		return nil
	})
	
	return refs, err
}

// CreateBranch creates a new branch pointing to the given commit
func (rm *RefManager) CreateBranch(branchName string, commitID objects.ObjectID) error {
	refName := "refs/heads/" + branchName
	return rm.UpdateRef(refName, commitID)
}

// DeleteBranch deletes a branch
func (rm *RefManager) DeleteBranch(branchName string) error {
	refPath := filepath.Join(rm.gitDir, "refs", "heads", branchName)
	err := os.Remove(refPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("branch does not exist: %s", branchName)
	}
	return err
}

// CreateTag creates a new tag pointing to the given object
func (rm *RefManager) CreateTag(tagName string, objectID objects.ObjectID) error {
	refName := "refs/tags/" + tagName
	return rm.UpdateRef(refName, objectID)
}

// DeleteTag deletes a tag
func (rm *RefManager) DeleteTag(tagName string) error {
	refPath := filepath.Join(rm.gitDir, "refs", "tags", tagName)
	err := os.Remove(refPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("tag does not exist: %s", tagName)
	}
	return err
}

// CurrentBranch returns the current branch name
func (rm *RefManager) CurrentBranch() (string, error) {
	_, refName, err := rm.HEAD()
	if err != nil {
		return "", err
	}
	
	if refName == "" {
		return "", fmt.Errorf("HEAD is detached")
	}
	
	if strings.HasPrefix(refName, "refs/heads/") {
		return strings.TrimPrefix(refName, "refs/heads/"), nil
	}
	
	return refName, nil
}

// IsValidRef checks if a reference name is valid
func (rm *RefManager) IsValidRef(refName string) bool {
	// Basic validation - Git ref names have complex rules
	if refName == "" {
		return false
	}
	
	// Cannot start or end with slash
	if strings.HasPrefix(refName, "/") || strings.HasSuffix(refName, "/") {
		return false
	}
	
	// Cannot contain double slashes
	if strings.Contains(refName, "//") {
		return false
	}
	
	// Cannot contain certain characters
	forbidden := []string{".", "..", " ", "~", "^", ":", "?", "*", "[", "\\"}
	for _, f := range forbidden {
		if strings.Contains(refName, f) {
			return false
		}
	}
	
	return true
}

// RefExists checks if a reference exists
func (rm *RefManager) RefExists(refName string) bool {
	_, err := rm.ResolveRef(refName)
	return err == nil
}

// WriteRef writes a reference with locking
func (rm *RefManager) WriteRef(refName string, id objects.ObjectID, oldID *objects.ObjectID) error {
	refPath := filepath.Join(rm.gitDir, refName)
	lockPath := refPath + ".lock"
	
	// Ensure directory exists before creating lock file
	if err := os.MkdirAll(filepath.Dir(refPath), 0755); err != nil {
		return fmt.Errorf("failed to create ref directory: %w", err)
	}
	
	// Create lock file
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer os.Remove(lockPath)
	defer lockFile.Close()
	
	// Verify old value if specified
	if oldID != nil {
		currentID, err := rm.readRefFile(refName)
		if err == nil && currentID != *oldID {
			return fmt.Errorf("reference has changed")
		}
	}
	
	// Write new value to lock file
	content := fmt.Sprintf("%s\n", id.String())
	if _, err := lockFile.WriteString(content); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}
	
	if err := lockFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync lock file: %w", err)
	}
	
	lockFile.Close()
	
	// Atomically rename lock file to reference file
	return os.Rename(lockPath, refPath)
}

// PackedRefs represents packed references
type PackedRefs struct {
	refs map[string]objects.ObjectID
}

// ReadPackedRefs reads the packed-refs file
func (rm *RefManager) ReadPackedRefs() (*PackedRefs, error) {
	packedPath := filepath.Join(rm.gitDir, "packed-refs")
	file, err := os.Open(packedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &PackedRefs{refs: make(map[string]objects.ObjectID)}, nil
		}
		return nil, err
	}
	defer file.Close()
	
	refs := make(map[string]objects.ObjectID)
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			if id, err := objects.NewObjectID(parts[0]); err == nil {
				refs[parts[1]] = id
			}
		}
	}
	
	return &PackedRefs{refs: refs}, nil
}