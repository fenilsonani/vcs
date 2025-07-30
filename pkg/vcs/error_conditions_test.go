package vcs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fenilsonani/vcs/internal/core/objects"
)

func TestRepository_ErrorConditions(t *testing.T) {
	// Test Init with invalid path
	t.Run("Init with permission denied", func(t *testing.T) {
		// Try to create repository in a directory we can't write to
		invalidPath := "/root/invalid-repo"
		_, err := Init(invalidPath)
		if err == nil {
			t.Error("Init() should fail with permission denied path")
		}
	})

	// Test Open with invalid repository
	t.Run("Open with missing git directory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "vcs-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		_, err = Open(tmpDir)
		if err == nil {
			t.Error("Open() should fail for directory without .git")
		}
		if !strings.Contains(err.Error(), "not a git repository") {
			t.Errorf("Open() error = %v, want 'not a git repository'", err)
		}
	})

	// Test Open with .git file instead of directory
	t.Run("Open with .git file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "vcs-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create .git as a file instead of directory
		gitFile := filepath.Join(tmpDir, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: ../other-repo"), 0644); err != nil {
			t.Fatalf("Failed to create .git file: %v", err)
		}

		_, err = Open(tmpDir)
		if err == nil {
			t.Error("Open() should fail when .git is a file")
		}
	})
}

func TestRepository_WriteObjectErrors(t *testing.T) {
	// Create temp repository
	tmpDir, err := os.MkdirTemp("", "vcs-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test WriteObject with read-only objects directory
	t.Run("WriteObject with read-only objects dir", func(t *testing.T) {
		objectsDir := filepath.Join(repo.GitDir(), "objects")
		if err := os.Chmod(objectsDir, 0444); err != nil {
			t.Skip("Could not make objects directory read-only")
		}
		defer os.Chmod(objectsDir, 0755) // Restore permissions

		blob := objects.NewBlob([]byte("test data"))
		err := repo.WriteObject(blob)
		if err == nil {
			t.Error("WriteObject() should fail with read-only objects directory")
		}
	})
}

func TestRepository_HashObjectErrors(t *testing.T) {
	// Create temp repository
	tmpDir, err := os.MkdirTemp("", "vcs-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test with unsupported object types
	unsupportedTypes := []objects.ObjectType{
		objects.TypeTree,
		objects.TypeCommit,
		objects.TypeTag,
		objects.ObjectType("invalid"),
	}

	for _, objType := range unsupportedTypes {
		t.Run("HashObject with "+string(objType), func(t *testing.T) {
			_, err := repo.HashObject([]byte("test"), objType, false)
			if err == nil {
				t.Errorf("HashObject() should fail with unsupported type %s", objType)
			}
			if !strings.Contains(err.Error(), "unsupported object type") {
				t.Errorf("HashObject() error = %v, want 'unsupported object type'", err)
			}
		})
	}
}

func TestRepository_CreateTreeErrors(t *testing.T) {
	// Create temp repository
	tmpDir, err := os.MkdirTemp("", "vcs-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test CreateTree with invalid entries
	t.Run("CreateTree with empty name", func(t *testing.T) {
		entries := []objects.TreeEntry{
			{Mode: objects.ModeBlob, Name: "", ID: objects.ObjectID{1}},
		}

		_, err := repo.CreateTree(entries)
		if err == nil {
			t.Error("CreateTree() should fail with empty name")
		}
		if !strings.Contains(err.Error(), "entry name cannot be empty") {
			t.Errorf("CreateTree() error = %v, want 'entry name cannot be empty'", err)
		}
	})

	// Test CreateTree with duplicate names
	t.Run("CreateTree with duplicate names", func(t *testing.T) {
		entries := []objects.TreeEntry{
			{Mode: objects.ModeBlob, Name: "file.txt", ID: objects.ObjectID{1}},
			{Mode: objects.ModeBlob, Name: "file.txt", ID: objects.ObjectID{2}},
		}

		_, err := repo.CreateTree(entries)
		if err == nil {
			t.Error("CreateTree() should fail with duplicate names")
		}
		if !strings.Contains(err.Error(), "duplicate entry name") {
			t.Errorf("CreateTree() error = %v, want 'duplicate entry name'", err)
		}
	})
}

func TestRepository_ReadObjectErrors(t *testing.T) {
	// Create temp repository
	tmpDir, err := os.MkdirTemp("", "vcs-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test ReadObject with non-existent object
	t.Run("ReadObject non-existent", func(t *testing.T) {
		fakeID := objects.ComputeHash(objects.TypeBlob, []byte("nonexistent"))
		_, err := repo.ReadObject(fakeID)
		if err == nil {
			t.Error("ReadObject() should fail for non-existent object")
		}
	})
}

func TestRepository_CreateCommitWithErrors(t *testing.T) {
	// Create temp repository
	tmpDir, err := os.MkdirTemp("", "vcs-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Create a tree for testing
	tree, err := repo.CreateTree([]objects.TreeEntry{})
	if err != nil {
		t.Fatalf("CreateTree() error = %v", err)
	}

	// Test CreateCommit with invalid signatures
	t.Run("CreateCommit with empty author", func(t *testing.T) {
		author := objects.Signature{Name: "", Email: "", When: time.Time{}}
		committer := objects.Signature{Name: "Committer", Email: "committer@example.com", When: time.Now()}

		commit, err := repo.CreateCommit(tree.ID(), nil, author, committer, "Test commit")
		if err != nil {
			t.Errorf("CreateCommit() should succeed with empty author, got error: %v", err)
		}
		if commit == nil {
			t.Error("CreateCommit() returned nil")
		}
	})

	// Test CreateCommit with very long message
	t.Run("CreateCommit with long message", func(t *testing.T) {
		author := objects.Signature{Name: "Author", Email: "author@example.com", When: time.Now()}
		committer := objects.Signature{Name: "Committer", Email: "committer@example.com", When: time.Now()}
		longMessage := strings.Repeat("This is a very long commit message. ", 1000)

		commit, err := repo.CreateCommit(tree.ID(), nil, author, committer, longMessage)
		if err != nil {
			t.Errorf("CreateCommit() should succeed with long message, got error: %v", err)
		}
		if commit == nil {
			t.Error("CreateCommit() returned nil")
		}
		if commit.Message() != longMessage {
			t.Error("CreateCommit() did not preserve long message")
		}
	})
}

func TestRepository_CreateTagWithErrors(t *testing.T) {
	// Create temp repository
	tmpDir, err := os.MkdirTemp("", "vcs-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Create a commit to tag
	tree, _ := repo.CreateTree([]objects.TreeEntry{})
	commit, _ := repo.CreateCommit(tree.ID(), nil, objects.Signature{
		Name: "Test", Email: "test@example.com", When: time.Now(),
	}, objects.Signature{
		Name: "Test", Email: "test@example.com", When: time.Now(),
	}, "Test commit")

	// Test CreateTag with various tag names
	t.Run("CreateTag with special characters", func(t *testing.T) {
		tagger := objects.Signature{Name: "Tagger", Email: "tagger@example.com", When: time.Now()}
		tagName := "v1.0.0-alpha+beta"

		tag, err := repo.CreateTag(commit.ID(), objects.TypeCommit, tagName, tagger, "Tag with special chars")
		if err != nil {
			t.Errorf("CreateTag() should succeed with special characters, got error: %v", err)
		}
		if tag == nil {
			t.Error("CreateTag() returned nil")
		}
		if tag.TagName() != tagName {
			t.Errorf("CreateTag() tag name = %q, want %q", tag.TagName(), tagName)
		}
	})

	// Test CreateTag with empty message
	t.Run("CreateTag with empty message", func(t *testing.T) {
		tagger := objects.Signature{Name: "Tagger", Email: "tagger@example.com", When: time.Now()}

		tag, err := repo.CreateTag(commit.ID(), objects.TypeCommit, "empty-msg", tagger, "")
		if err != nil {
			t.Errorf("CreateTag() should succeed with empty message, got error: %v", err)
		}
		if tag == nil {
			t.Error("CreateTag() returned nil")
		}
		if tag.Message() != "" {
			t.Errorf("CreateTag() message = %q, want empty", tag.Message())
		}
	})
}

func TestRepository_BoundaryConditions(t *testing.T) {
	// Create temp repository
	tmpDir, err := os.MkdirTemp("", "vcs-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test with zero-length blob
	t.Run("CreateBlob with zero length", func(t *testing.T) {
		blob, err := repo.CreateBlob([]byte{})
		if err != nil {
			t.Errorf("CreateBlob() with zero length should succeed, got error: %v", err)
		}
		if blob == nil {
			t.Error("CreateBlob() returned nil")
		}
		if blob.Size() != 0 {
			t.Errorf("CreateBlob() size = %d, want 0", blob.Size())
		}
	})

	// Test with large blob
	t.Run("CreateBlob with large data", func(t *testing.T) {
		largeData := make([]byte, 100000) // 100KB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		blob, err := repo.CreateBlob(largeData)
		if err != nil {
			t.Errorf("CreateBlob() with large data should succeed, got error: %v", err)
		}
		if blob == nil {
			t.Error("CreateBlob() returned nil")
		}
		if blob.Size() != int64(len(largeData)) {
			t.Errorf("CreateBlob() size = %d, want %d", blob.Size(), len(largeData))
		}
	})

	// Test HasObject with zero ID
	t.Run("HasObject with zero ID", func(t *testing.T) {
		zeroID := objects.ObjectID{}
		exists := repo.HasObject(zeroID)
		if exists {
			t.Error("HasObject() should return false for zero ID")
		}
	})
}