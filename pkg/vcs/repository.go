package vcs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fenilsonani/vcs/internal/core/objects"
)

// Repository represents a git repository
type Repository struct {
	path    string
	gitDir  string
	storage *objects.Storage
}

// Init initializes a new repository at the given path
func Init(path string) (*Repository, error) {
	// Create repository directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create repository directory: %w", err)
	}
	
	gitDir := filepath.Join(path, ".git")
	
	// Create .git directory
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .git directory: %w", err)
	}
	
	// Initialize object storage
	storage := objects.NewStorage(gitDir)
	if err := storage.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize object storage: %w", err)
	}
	
	// Create other necessary directories
	dirs := []string{"refs/heads", "refs/tags", "hooks", "info"}
	for _, dir := range dirs {
		fullPath := filepath.Join(gitDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}
	
	// Create HEAD file
	headPath := filepath.Join(gitDir, "HEAD")
	headContent := "ref: refs/heads/main\n"
	if err := os.WriteFile(headPath, []byte(headContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to create HEAD file: %w", err)
	}
	
	// Create config file
	configPath := filepath.Join(gitDir, "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
	logallrefupdates = true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to create config file: %w", err)
	}
	
	// Create description file
	descPath := filepath.Join(gitDir, "description")
	descContent := "Unnamed repository; edit this file 'description' to name the repository.\n"
	if err := os.WriteFile(descPath, []byte(descContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to create description file: %w", err)
	}
	
	return &Repository{
		path:    path,
		gitDir:  gitDir,
		storage: storage,
	}, nil
}

// Open opens an existing repository
func Open(path string) (*Repository, error) {
	// Find .git directory
	gitDir := filepath.Join(path, ".git")
	if info, err := os.Stat(gitDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("not a git repository: %s", path)
	}
	
	// Verify it's a valid repository
	headPath := filepath.Join(gitDir, "HEAD")
	if _, err := os.Stat(headPath); err != nil {
		return nil, fmt.Errorf("invalid git repository: missing HEAD")
	}
	
	storage := objects.NewStorage(gitDir)
	
	return &Repository{
		path:    path,
		gitDir:  gitDir,
		storage: storage,
	}, nil
}

// Path returns the repository path
func (r *Repository) Path() string {
	return r.path
}

// GitDir returns the .git directory path
func (r *Repository) GitDir() string {
	return r.gitDir
}

// HashObject hashes data and optionally writes it to the object store
func (r *Repository) HashObject(data []byte, objType objects.ObjectType, write bool) (objects.ObjectID, error) {
	var obj objects.Object
	
	switch objType {
	case objects.TypeBlob:
		obj = objects.NewBlob(data)
	default:
		return objects.ObjectID{}, fmt.Errorf("unsupported object type for hash-object: %s", objType)
	}
	
	if write {
		if err := r.storage.WriteObject(obj); err != nil {
			return objects.ObjectID{}, err
		}
	}
	
	return obj.ID(), nil
}

// HashObjectFromReader hashes data from a reader
func (r *Repository) HashObjectFromReader(reader io.Reader, objType objects.ObjectType, write bool) (objects.ObjectID, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return objects.ObjectID{}, fmt.Errorf("failed to read data: %w", err)
	}
	
	return r.HashObject(data, objType, write)
}

// ReadObject reads an object from the repository
func (r *Repository) ReadObject(id objects.ObjectID) (objects.Object, error) {
	return r.storage.ReadObject(id)
}

// WriteObject writes an object to the repository
func (r *Repository) WriteObject(obj objects.Object) error {
	return r.storage.WriteObject(obj)
}

// HasObject checks if an object exists in the repository
func (r *Repository) HasObject(id objects.ObjectID) bool {
	return r.storage.HasObject(id)
}

// CreateBlob creates a blob from data
func (r *Repository) CreateBlob(data []byte) (*objects.Blob, error) {
	blob := objects.NewBlob(data)
	if err := r.WriteObject(blob); err != nil {
		return nil, err
	}
	return blob, nil
}

// CreateTree creates a tree object
func (r *Repository) CreateTree(entries []objects.TreeEntry) (*objects.Tree, error) {
	tree := objects.NewTree()
	
	for _, entry := range entries {
		if err := tree.AddEntry(entry.Mode, entry.Name, entry.ID); err != nil {
			return nil, err
		}
	}
	
	if err := r.WriteObject(tree); err != nil {
		return nil, err
	}
	
	return tree, nil
}

// CreateCommit creates a commit object
func (r *Repository) CreateCommit(tree objects.ObjectID, parents []objects.ObjectID, author, committer objects.Signature, message string) (*objects.Commit, error) {
	commit := objects.NewCommit(tree, parents, author, committer, message)
	
	if err := r.WriteObject(commit); err != nil {
		return nil, err
	}
	
	return commit, nil
}

// CreateTag creates a tag object
func (r *Repository) CreateTag(object objects.ObjectID, objType objects.ObjectType, tag string, tagger objects.Signature, message string) (*objects.Tag, error) {
	tagObj := objects.NewTag(object, objType, tag, tagger, message)
	
	if err := r.WriteObject(tagObj); err != nil {
		return nil, err
	}
	
	return tagObj, nil
}