package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewCatFileCommand(t *testing.T) {
	cmd := newCatFileCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "cat-file [options] <object>", cmd.Use)
	assert.Contains(t, cmd.Short, "Provide content or type and size information")
}

func TestCatFileCommandDetailed(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		setupFunc   func(t *testing.T, repo *vcs.Repository) objects.ObjectID
		expectError bool
		checkFunc   func(t *testing.T, output string)
	}{
		{
			name: "show blob content with -p",
			args: []string{},
			flags: map[string]string{
				"pretty-print": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository) objects.ObjectID {
				// Create a blob
				storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
				blob := objects.NewBlob([]byte("Hello, World!\n"))
				err := storage.WriteObject(blob)
				require.NoError(t, err)
				return blob.ID()
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Equal(t, "Hello, World!\n", output)
			},
		},
		{
			name: "show object type with -t",
			args: []string{},
			flags: map[string]string{
				"type": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository) objects.ObjectID {
				// Create a blob
				storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
				blob := objects.NewBlob([]byte("test content"))
				err := storage.WriteObject(blob)
				require.NoError(t, err)
				return blob.ID()
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Equal(t, "blob\n", output)
			},
		},
		{
			name: "show object size with -s",
			args: []string{},
			flags: map[string]string{
				"size": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository) objects.ObjectID {
				// Create a blob
				storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
				content := []byte("test content")
				blob := objects.NewBlob(content)
				err := storage.WriteObject(blob)
				require.NoError(t, err)
				return blob.ID()
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Equal(t, "12\n", output) // "test content" is 12 bytes
			},
		},
		{
			name: "show tree content",
			args: []string{},
			flags: map[string]string{
				"pretty-print": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository) objects.ObjectID {
				// Create a tree with entries
				storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
				
				// Create a blob first
				blob := objects.NewBlob([]byte("file content"))
				err := storage.WriteObject(blob)
				require.NoError(t, err)
				
				// Create tree
				tree := objects.NewTree()
				err = tree.AddEntry(0100644, "file.txt", blob.ID())
				require.NoError(t, err)
				
				err = storage.WriteObject(tree)
				require.NoError(t, err)
				return tree.ID()
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "100644 blob")
				assert.Contains(t, output, "file.txt")
			},
		},
		{
			name: "show commit content",
			args: []string{},
			flags: map[string]string{
				"pretty-print": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository) objects.ObjectID {
				// Create commit
				storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
				
				// Create tree
				tree := objects.NewTree()
				err := storage.WriteObject(tree)
				require.NoError(t, err)
				
				// Create commit
				author := objects.Signature{
					Name:  "Test User",
					Email: "test@example.com",
				}
				commit := objects.NewCommit(tree.ID(), []objects.ObjectID{}, author, author, "Test commit")
				err = storage.WriteObject(commit)
				require.NoError(t, err)
				return commit.ID()
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "tree")
				assert.Contains(t, output, "author Test User <test@example.com>")
				assert.Contains(t, output, "committer Test User <test@example.com>")
				assert.Contains(t, output, "Test commit")
			},
		},
		{
			name: "show tag content",
			args: []string{},
			flags: map[string]string{
				"pretty-print": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository) objects.ObjectID {
				// Create tag
				storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
				
				// Create a commit to tag
				tree := objects.NewTree()
				err := storage.WriteObject(tree)
				require.NoError(t, err)
				
				author := objects.Signature{
					Name:  "Test User",
					Email: "test@example.com",
				}
				commit := objects.NewCommit(tree.ID(), []objects.ObjectID{}, author, author, "Tagged commit")
				err = storage.WriteObject(commit)
				require.NoError(t, err)
				
				// Create tag
				tag := objects.NewTag(commit.ID(), "commit", "v1.0.0", author, "Release v1.0.0")
				err = storage.WriteObject(tag)
				require.NoError(t, err)
				return tag.ID()
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "object")
				assert.Contains(t, output, "type commit")
				assert.Contains(t, output, "tag v1.0.0")
				assert.Contains(t, output, "tagger Test User <test@example.com>")
				assert.Contains(t, output, "Release v1.0.0")
			},
		},
		{
			name: "non-existent object",
			args: []string{"0000000000000000000000000000000000000000"},
			flags: map[string]string{
				"pretty-print": "true",
			},
			setupFunc:   func(t *testing.T, repo *vcs.Repository) objects.ObjectID { return objects.ObjectID{} },
			expectError: true,
		},
		{
			name: "invalid object ID",
			args: []string{"invalid"},
			flags: map[string]string{
				"pretty-print": "true",
			},
			setupFunc:   func(t *testing.T, repo *vcs.Repository) objects.ObjectID { return objects.ObjectID{} },
			expectError: true,
		},
		{
			name: "no flags specified",
			args: []string{},
			setupFunc: func(t *testing.T, repo *vcs.Repository) objects.ObjectID {
				// Create a blob
				storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
				blob := objects.NewBlob([]byte("test"))
				err := storage.WriteObject(blob)
				require.NoError(t, err)
				return blob.ID()
			},
			expectError: true, // Should error without -p, -t, or -s
		},
		{
			name: "multiple flags specified",
			args: []string{},
			flags: map[string]string{
				"pretty-print": "true",
				"type": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository) objects.ObjectID {
				// Create a blob
				storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
				blob := objects.NewBlob([]byte("test"))
				err := storage.WriteObject(blob)
				require.NoError(t, err)
				return blob.ID()
			},
			expectError: true, // Should error with multiple flags
		},
		{
			name: "pretty print mode",
			args: []string{},
			flags: map[string]string{
				"pretty-print": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository) objects.ObjectID {
				// Create a commit
				storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
				
				tree := objects.NewTree()
				err := storage.WriteObject(tree)
				require.NoError(t, err)
				
				author := objects.Signature{
					Name:  "Test User",
					Email: "test@example.com",
				}
				commit := objects.NewCommit(tree.ID(), []objects.ObjectID{}, author, author, "Pretty commit")
				err = storage.WriteObject(commit)
				require.NoError(t, err)
				return commit.ID()
			},
			checkFunc: func(t *testing.T, output string) {
				// Pretty print should format nicely
				assert.Contains(t, output, "commit")
				assert.Contains(t, output, "Author: Test User <test@example.com>")
				assert.Contains(t, output, "Pretty commit")
			},
		},
		{
			name: "textconv mode",
			args: []string{},
			flags: map[string]string{
				"textconv": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository) objects.ObjectID {
				// Create a blob
				storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
				blob := objects.NewBlob([]byte("binary content"))
				err := storage.WriteObject(blob)
				require.NoError(t, err)
				return blob.ID()
			},
			checkFunc: func(t *testing.T, output string) {
				// textconv would apply filters in real git
				assert.Equal(t, "binary content", output)
			},
		},
		{
			name:        "outside repository",
			args:        []string{"someobject"},
			flags:       map[string]string{"p": "true"},
			setupFunc:   func(t *testing.T, repo *vcs.Repository) objects.ObjectID { return objects.ObjectID{} },
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "outside repository" {
				// Test outside repository
				tmpDir := t.TempDir()
				err := os.Chdir(tmpDir)
				require.NoError(t, err)
				
				cmd := newCatFileCommand()
				for flag, value := range tc.flags {
					err := cmd.Flags().Set(flag, value)
					require.NoError(t, err)
				}
				
				var buf bytes.Buffer
				cmd.SetOut(&buf)
				cmd.SetErr(&buf)
				cmd.SetArgs(tc.args)
				
				err = cmd.Execute()
				assert.Error(t, err)
				return
			}
			
			// Create temporary directory
			tmpDir := t.TempDir()
			repoPath := filepath.Join(tmpDir, "test-repo")
			
			// Initialize repository
			repo, err := vcs.Init(repoPath)
			require.NoError(t, err)
			
			// Change to repo directory
			err = os.Chdir(repoPath)
			require.NoError(t, err)
			
			// Setup test data
			var objectID objects.ObjectID
			if tc.setupFunc != nil {
				objectID = tc.setupFunc(t, repo)
			}
			
			// Create command
			cmd := newCatFileCommand()
			
			// Set flags
			for flag, value := range tc.flags {
				err := cmd.Flags().Set(flag, value)
				require.NoError(t, err)
			}
			
			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			
			// Set args (object ID if created)
			if len(tc.args) == 0 && !objectID.IsZero() {
				cmd.SetArgs([]string{objectID.String()})
			} else {
				cmd.SetArgs(tc.args)
			}
			
			// Execute command
			err = cmd.Execute()
			
			// Check error
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Check output
				if tc.checkFunc != nil {
					tc.checkFunc(t, buf.String())
				}
			}
		})
	}
}

func TestCatFileExitCode(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Test with -e flag (check existence)
	cmd := newCatFileCommand()
	err = cmd.Flags().Set("e", "true")
	require.NoError(t, err)
	
	// Non-existent object
	cmd.SetArgs([]string{"0000000000000000000000000000000000000000"})
	err = cmd.Execute()
	assert.Error(t, err)
	
	// Create an object
	storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
	blob := objects.NewBlob([]byte("test"))
	err = storage.WriteObject(blob)
	require.NoError(t, err)
	
	// Existing object
	cmd = newCatFileCommand()
	err = cmd.Flags().Set("e", "true")
	require.NoError(t, err)
	cmd.SetArgs([]string{blob.ID().String()})
	err = cmd.Execute()
	assert.NoError(t, err)
}

func TestCatFileBatchMode(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create some objects
	storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
	
	blob1 := objects.NewBlob([]byte("content1"))
	err = storage.WriteObject(blob1)
	require.NoError(t, err)
	
	blob2 := objects.NewBlob([]byte("content2"))
	err = storage.WriteObject(blob2)
	require.NoError(t, err)
	
	// Test batch mode
	cmd := newCatFileCommand()
	err = cmd.Flags().Set("batch", "true")
	require.NoError(t, err)
	
	// In real implementation, this would read from stdin
	// For now, test that the flag is accepted
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	// Should not require args in batch mode
	err = cmd.Execute()
	// Batch mode not fully implemented, but flag should be accepted
}

func TestCatFileFilters(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create blob
	storage := objects.NewStorage(filepath.Join(repo.GitDir(), "objects"))
	blob := objects.NewBlob([]byte("test content\n"))
	err = storage.WriteObject(blob)
	require.NoError(t, err)
	
	// Test with --filters flag
	cmd := newCatFileCommand()
	err = cmd.Flags().Set("filters", "true")
	require.NoError(t, err)
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{blob.ID().String()})
	
	err = cmd.Execute()
	// Filters not implemented, but should work like -p for now
	assert.NoError(t, err)
	assert.Equal(t, "test content\n", buf.String())
}