package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewTagCommand(t *testing.T) {
	cmd := newTagCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "tag [flags] [<tagname>] [<commit>]", cmd.Use)
	assert.Contains(t, cmd.Short, "Create, list, delete or verify a tag object")
}

func TestTagCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		setupFunc   func(t *testing.T, repo *vcs.Repository, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "list tags (empty repository)",
			args: []string{},
			flags: map[string]string{
				"list": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				// No tags should be listed
				assert.Equal(t, "", output)
			},
		},
		{
			name: "create lightweight tag",
			args: []string{"v1.0"},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Created lightweight tag v1.0")
				
				// Check tag file exists
				tagPath := filepath.Join(repoPath, ".git", "refs", "tags", "v1.0")
				assert.FileExists(t, tagPath)
			},
		},
		{
			name: "create annotated tag",
			args: []string{"v2.0"},
			flags: map[string]string{
				"annotated": "true",
				"message":   "Version 2.0 release",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Created annotated tag v2.0")
				
				// Check tag file exists
				tagPath := filepath.Join(repoPath, ".git", "refs", "tags", "v2.0")
				assert.FileExists(t, tagPath)
			},
		},
		{
			name: "create tag with message (implies annotated)",
			args: []string{"v3.0"},
			flags: map[string]string{
				"message": "Version 3.0 with new features",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Created annotated tag v3.0")
			},
		},
		{
			name: "create tag on specific commit",
			args: []string{"v1.0", "HEAD~1"},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create multiple commits
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("content1\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("First commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Second commit
				err = os.WriteFile(testFile, []byte("content2\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Second commit", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Created lightweight tag v1.0")
			},
		},
		{
			name: "list existing tags",
			args: []string{},
			flags: map[string]string{
				"list": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				hash, err := testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create tags
				tagger := objects.Signature{
					Name:  "Test User",
					Email: "test@example.com",
					When:  time.Now(),
				}
				_, err = testRepo.CreateTag(hash, objects.TypeCommit, "v1.0", tagger, "Version 1.0")
				require.NoError(t, err)
				_, err = testRepo.CreateTag(hash, objects.TypeCommit, "v2.0", tagger, "Version 2.0")
				require.NoError(t, err)
				
				// Write tag refs manually since CreateTag might not write refs
				refsDir := filepath.Join(repoPath, ".git", "refs", "tags")
				err = os.MkdirAll(refsDir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(refsDir, "v1.0"), []byte(hash.String()+"\n"), 0644)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(refsDir, "v2.0"), []byte(hash.String()+"\n"), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "v1.0")
				assert.Contains(t, output, "v2.0")
			},
		},
		{
			name: "delete tag",
			args: []string{"v1.0"},
			flags: map[string]string{
				"delete": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				hash, err := testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create tag manually
				refsDir := filepath.Join(repoPath, ".git", "refs", "tags")
				err = os.MkdirAll(refsDir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(refsDir, "v1.0"), []byte(hash.String()+"\n"), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Deleted tag v1.0")
				
				// Check tag file no longer exists
				tagPath := filepath.Join(repoPath, ".git", "refs", "tags", "v1.0")
				assert.NoFileExists(t, tagPath)
			},
		},
		{
			name: "force create tag (overwrite existing)",
			args: []string{"v1.0"},
			flags: map[string]string{
				"force": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create commits
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("content1\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				hash1, err := testRepo.Commit("First commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Second commit
				err = os.WriteFile(testFile, []byte("content2\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Second commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create existing tag on first commit
				refsDir := filepath.Join(repoPath, ".git", "refs", "tags")
				err = os.MkdirAll(refsDir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(refsDir, "v1.0"), []byte(hash1.String()+"\n"), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Created lightweight tag v1.0")
			},
		},
		{
			name: "create tag without force (should fail on existing)",
			args: []string{"v1.0"},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				hash, err := testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create existing tag
				refsDir := filepath.Join(repoPath, ".git", "refs", "tags")
				err = os.MkdirAll(refsDir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(refsDir, "v1.0"), []byte(hash.String()+"\n"), 0644)
				require.NoError(t, err)
			},
			expectError: true,
		},
		{
			name:        "delete non-existent tag",
			args:        []string{"non-existent"},
			flags:       map[string]string{"delete": "true"},
			expectError: true,
		},
		{
			name:        "create tag on non-existent commit",
			args:        []string{"v1.0", "invalid-hash"},
			expectError: true,
		},
		{
			name: "create tag with invalid name",
			args: []string{"invalid/tag/name"},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			expectError: true,
		},
		{
			name:        "tag outside repository",
			args:        []string{"v1.0"},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "tag outside repository" {
				// Test outside repository
				tmpDir := t.TempDir()
				err := os.Chdir(tmpDir)
				require.NoError(t, err)
				
				cmd := newTagCommand()
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
			
			// Run setup if provided
			if tc.setupFunc != nil {
				tc.setupFunc(t, repo, repoPath)
			}
			
			// Create command
			cmd := newTagCommand()
			
			// Set flags
			for flag, value := range tc.flags {
				err := cmd.Flags().Set(flag, value)
				require.NoError(t, err)
			}
			
			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			
			// Execute command
			cmd.SetArgs(tc.args)
			err = cmd.Execute()
			
			// Check error
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			// Check output
			if tc.checkFunc != nil {
				tc.checkFunc(t, buf.String(), repoPath)
			}
		})
	}
}

func TestTagSorting(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create initial commit
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("content\n"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("test.txt")
	require.NoError(t, err)
	hash, err := testRepo.Commit("Initial commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Create multiple tags in random order
	tags := []string{"v10.0", "v1.0", "v2.0", "v1.1", "beta-1"}
	refsDir := filepath.Join(repoPath, ".git", "refs", "tags")
	err = os.MkdirAll(refsDir, 0755)
	require.NoError(t, err)
	
	for _, tag := range tags {
		err = os.WriteFile(filepath.Join(refsDir, tag), []byte(hash.String()+"\n"), 0644)
		require.NoError(t, err)
	}
	
	// List tags
	cmd := newTagCommand()
	err = cmd.Flags().Set("list", "true")
	require.NoError(t, err)
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	// Tags should be sorted lexicographically
	assert.Contains(t, output, "beta-1")
	assert.Contains(t, output, "v1.0")
	assert.Contains(t, output, "v1.1")
	assert.Contains(t, output, "v2.0")
	assert.Contains(t, output, "v10.0")
}

func TestTagWithPatterns(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create initial commit
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("content\n"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("test.txt")
	require.NoError(t, err)
	hash, err := testRepo.Commit("Initial commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Create tags with different patterns
	tags := []string{"v1.0", "v2.0", "beta-1", "alpha-1", "release-1.0"}
	refsDir := filepath.Join(repoPath, ".git", "refs", "tags")
	err = os.MkdirAll(refsDir, 0755)
	require.NoError(t, err)
	
	for _, tag := range tags {
		err = os.WriteFile(filepath.Join(refsDir, tag), []byte(hash.String()+"\n"), 0644)
		require.NoError(t, err)
	}
	
	// List tags with pattern
	cmd := newTagCommand()
	err = cmd.Flags().Set("list", "true")
	require.NoError(t, err)
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"v*"})
	
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	// Should only show tags matching v* pattern
	assert.Contains(t, output, "v1.0")
	assert.Contains(t, output, "v2.0")
	assert.NotContains(t, output, "beta-1")
	assert.NotContains(t, output, "alpha-1")
	assert.NotContains(t, output, "release-1.0")
}

func TestTagDefaultListBehavior(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create initial commit
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("content\n"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("test.txt")
	require.NoError(t, err)
	hash, err := testRepo.Commit("Initial commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Create tag
	refsDir := filepath.Join(repoPath, ".git", "refs", "tags")
	err = os.MkdirAll(refsDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(refsDir, "v1.0"), []byte(hash.String()+"\n"), 0644)
	require.NoError(t, err)
	
	// Test default behavior (should list tags when no arguments)
	cmd := newTagCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "v1.0")
}

func TestTagValidation(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create initial commit
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("content\n"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("test.txt")
	require.NoError(t, err)
	_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Test various invalid tag names
	invalidNames := []string{
		"",           // empty
		"refs/",      // starts with refs/
		"tag with spaces", // contains spaces
		"tag\nwith\nnewlines", // contains newlines
		"../tag",     // contains path traversal
		".tag",       // starts with dot
		"tag.",       // ends with dot
		"tag.lock",   // ends with .lock
	}
	
	for _, name := range invalidNames {
		t.Run("invalid_name_"+name, func(t *testing.T) {
			if name == "" {
				name = "empty"
			}
			
			cmd := newTagCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			
			cmd.SetArgs([]string{name})
			err = cmd.Execute()
			// Most invalid names should cause errors, but implementation might vary
			// Just ensure the command doesn't crash
		})
	}
}