package vcs

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fenilsonani/vcs/internal/core/objects"
)

func TestInit(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vcs-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize repository
	repo, err := Init(repoPath)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Verify repository structure
	if repo.Path() != repoPath {
		t.Errorf("Path() = %v, want %v", repo.Path(), repoPath)
	}

	gitDir := filepath.Join(repoPath, ".git")
	if repo.GitDir() != gitDir {
		t.Errorf("GitDir() = %v, want %v", repo.GitDir(), gitDir)
	}

	// Check directories exist
	dirs := []string{
		".git",
		".git/objects",
		".git/refs/heads",
		".git/refs/tags",
		".git/hooks",
		".git/info",
	}

	for _, dir := range dirs {
		path := filepath.Join(repoPath, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Directory %s does not exist", dir)
		}
	}

	// Check files exist
	files := []string{
		".git/HEAD",
		".git/config",
		".git/description",
	}

	for _, file := range files {
		path := filepath.Join(repoPath, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("File %s does not exist", file)
		}
	}

	// Verify HEAD content
	headContent, err := os.ReadFile(filepath.Join(repoPath, ".git/HEAD"))
	if err != nil {
		t.Fatalf("Failed to read HEAD: %v", err)
	}
	if !strings.Contains(string(headContent), "ref: refs/heads/main") {
		t.Errorf("HEAD content = %s, want to contain 'ref: refs/heads/main'", headContent)
	}
}

func TestInit_ExistingDirectory(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vcs-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repository
	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if repo.Path() != tmpDir {
		t.Errorf("Path() = %v, want %v", repo.Path(), tmpDir)
	}
}

func TestOpen(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vcs-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repository first
	repo1, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Open existing repository
	repo2, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if repo2.Path() != repo1.Path() {
		t.Errorf("Path() = %v, want %v", repo2.Path(), repo1.Path())
	}

	if repo2.GitDir() != repo1.GitDir() {
		t.Errorf("GitDir() = %v, want %v", repo2.GitDir(), repo1.GitDir())
	}
}

func TestOpen_NotRepository(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vcs-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Try to open non-repository
	_, err = Open(tmpDir)
	if err == nil {
		t.Error("Open() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("Open() error = %v, want 'not a git repository'", err)
	}
}

func TestOpen_MissingHEAD(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vcs-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .git directory without HEAD
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Try to open invalid repository
	_, err = Open(tmpDir)
	if err == nil {
		t.Error("Open() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "invalid git repository") {
		t.Errorf("Open() error = %v, want 'invalid git repository'", err)
	}
}

func TestRepository_HashObject(t *testing.T) {
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

	tests := []struct {
		name    string
		data    []byte
		write   bool
		wantErr bool
	}{
		{
			name:    "hash without writing",
			data:    []byte("test content"),
			write:   false,
			wantErr: false,
		},
		{
			name:    "hash with writing",
			data:    []byte("test content to store"),
			write:   true,
			wantErr: false,
		},
		{
			name:    "empty data",
			data:    []byte{},
			write:   true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := repo.HashObject(tt.data, objects.TypeBlob, tt.write)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if id.IsZero() {
					t.Error("HashObject() returned zero ID")
				}

				// Verify object exists if written
				if tt.write {
					if !repo.HasObject(id) {
						t.Error("Object was not written to storage")
					}
				}
			}
		})
	}
}

func TestRepository_HashObject_UnsupportedType(t *testing.T) {
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

	// Try to hash with unsupported type
	_, err = repo.HashObject([]byte("test"), objects.TypeTree, false)
	if err == nil {
		t.Error("HashObject() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "unsupported object type") {
		t.Errorf("HashObject() error = %v, want 'unsupported object type'", err)
	}
}

func TestRepository_HashObjectFromReader(t *testing.T) {
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

	data := []byte("test content from reader")
	reader := bytes.NewReader(data)

	id, err := repo.HashObjectFromReader(reader, objects.TypeBlob, true)
	if err != nil {
		t.Fatalf("HashObjectFromReader() error = %v", err)
	}

	if id.IsZero() {
		t.Error("HashObjectFromReader() returned zero ID")
	}

	// Verify object was written
	if !repo.HasObject(id) {
		t.Error("Object was not written to storage")
	}

	// Read back and verify
	obj, err := repo.ReadObject(id)
	if err != nil {
		t.Fatalf("ReadObject() error = %v", err)
	}

	blob, ok := obj.(*objects.Blob)
	if !ok {
		t.Fatal("Object is not a blob")
	}

	if !bytes.Equal(blob.Data(), data) {
		t.Errorf("Blob data = %v, want %v", blob.Data(), data)
	}
}

func TestRepository_CreateBlob(t *testing.T) {
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

	data := []byte("blob content")
	blob, err := repo.CreateBlob(data)
	if err != nil {
		t.Fatalf("CreateBlob() error = %v", err)
	}

	if blob == nil {
		t.Fatal("CreateBlob() returned nil")
	}

	if !bytes.Equal(blob.Data(), data) {
		t.Errorf("Blob data = %v, want %v", blob.Data(), data)
	}

	// Verify stored
	if !repo.HasObject(blob.ID()) {
		t.Error("Blob was not stored")
	}
}

func TestRepository_CreateTree(t *testing.T) {
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

	// Create blobs first
	blob1, _ := repo.CreateBlob([]byte("file1 content"))
	blob2, _ := repo.CreateBlob([]byte("file2 content"))

	entries := []objects.TreeEntry{
		{Mode: objects.ModeBlob, Name: "file1.txt", ID: blob1.ID()},
		{Mode: objects.ModeExec, Name: "script.sh", ID: blob2.ID()},
	}

	tree, err := repo.CreateTree(entries)
	if err != nil {
		t.Fatalf("CreateTree() error = %v", err)
	}

	if tree == nil {
		t.Fatal("CreateTree() returned nil")
	}

	if len(tree.Entries()) != 2 {
		t.Errorf("Tree entries = %v, want 2", len(tree.Entries()))
	}

	// Verify stored
	if !repo.HasObject(tree.ID()) {
		t.Error("Tree was not stored")
	}
}

func TestRepository_CreateCommit(t *testing.T) {
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

	// Create tree
	tree, _ := repo.CreateTree([]objects.TreeEntry{})

	author := objects.Signature{
		Name:  "Test Author",
		Email: "author@example.com",
		When:  time.Now(),
	}
	committer := objects.Signature{
		Name:  "Test Committer",
		Email: "committer@example.com",
		When:  time.Now(),
	}

	commit, err := repo.CreateCommit(tree.ID(), nil, author, committer, "Initial commit\n")
	if err != nil {
		t.Fatalf("CreateCommit() error = %v", err)
	}

	if commit == nil {
		t.Fatal("CreateCommit() returned nil")
	}

	if commit.Tree() != tree.ID() {
		t.Errorf("Commit tree = %v, want %v", commit.Tree(), tree.ID())
	}

	if commit.Message() != "Initial commit\n" {
		t.Errorf("Commit message = %v, want 'Initial commit\\n'", commit.Message())
	}

	// Verify stored
	if !repo.HasObject(commit.ID()) {
		t.Error("Commit was not stored")
	}
}

func TestRepository_CreateTag(t *testing.T) {
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

	// Create commit to tag
	tree, _ := repo.CreateTree([]objects.TreeEntry{})
	commit, _ := repo.CreateCommit(tree.ID(), nil, objects.Signature{
		Name:  "Test",
		Email: "test@example.com",
		When:  time.Now(),
	}, objects.Signature{
		Name:  "Test",
		Email: "test@example.com",
		When:  time.Now(),
	}, "Tagged commit\n")

	tagger := objects.Signature{
		Name:  "Test Tagger",
		Email: "tagger@example.com",
		When:  time.Now(),
	}

	tag, err := repo.CreateTag(commit.ID(), objects.TypeCommit, "v1.0.0", tagger, "Release v1.0.0\n")
	if err != nil {
		t.Fatalf("CreateTag() error = %v", err)
	}

	if tag == nil {
		t.Fatal("CreateTag() returned nil")
	}

	if tag.Object() != commit.ID() {
		t.Errorf("Tag object = %v, want %v", tag.Object(), commit.ID())
	}

	if tag.TagName() != "v1.0.0" {
		t.Errorf("Tag name = %v, want 'v1.0.0'", tag.TagName())
	}

	// Verify stored
	if !repo.HasObject(tag.ID()) {
		t.Error("Tag was not stored")
	}
}

func TestRepository_WriteAndReadObject(t *testing.T) {
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

	// Create and write blob
	blob := objects.NewBlob([]byte("test content"))
	if err := repo.WriteObject(blob); err != nil {
		t.Fatalf("WriteObject() error = %v", err)
	}

	// Read back
	obj, err := repo.ReadObject(blob.ID())
	if err != nil {
		t.Fatalf("ReadObject() error = %v", err)
	}

	readBlob, ok := obj.(*objects.Blob)
	if !ok {
		t.Fatal("ReadObject() returned wrong type")
	}

	if !bytes.Equal(readBlob.Data(), blob.Data()) {
		t.Errorf("Read data = %v, want %v", readBlob.Data(), blob.Data())
	}
}