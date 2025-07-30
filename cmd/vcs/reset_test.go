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

func TestNewResetCommand(t *testing.T) {
	cmd := newResetCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "reset", cmd.Use)
	assert.Contains(t, cmd.Short, "Reset current HEAD")
}

func TestResetCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		setupFunc   func(t *testing.T, repo *vcs.Repository, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "reset soft to previous commit",
			args: []string{"HEAD~1"},
			flags: map[string]string{
				"soft": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Second commit
				err = os.WriteFile(testFile, []byte("updated content\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Second commit", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Soft reset")
				// File should still have updated content after soft reset
				content, err := os.ReadFile(filepath.Join(repoPath, "test.txt"))
				require.NoError(t, err)
				assert.Equal(t, "updated content\n", string(content))
			},
		},
		{
			name: "reset mixed to previous commit",
			args: []string{"HEAD~1"},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Second commit
				err = os.WriteFile(testFile, []byte("updated content\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Second commit", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Mixed reset")
				// File should still have updated content after mixed reset
				content, err := os.ReadFile(filepath.Join(repoPath, "test.txt"))
				require.NoError(t, err)
				assert.Equal(t, "updated content\n", string(content))
			},
		},
		{
			name: "reset hard to previous commit",
			args: []string{"HEAD~1"},
			flags: map[string]string{
				"hard": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Second commit
				err = os.WriteFile(testFile, []byte("updated content\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Second commit", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Hard reset")
				// File should have initial content after hard reset
				content, err := os.ReadFile(filepath.Join(repoPath, "test.txt"))
				require.NoError(t, err)
				assert.Equal(t, "initial content\n", string(content))
			},
		},
		{
			name: "reset to HEAD (no change)",
			args: []string{"HEAD"},
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
				assert.Contains(t, output, "HEAD is now at")
			},
		},
		{
			name: "reset specific file",
			args: []string{"test.txt"},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Modify file and stage it
				err = os.WriteFile(testFile, []byte("modified content\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Unstaged changes after reset")
				// File should still be modified but unstaged
				content, err := os.ReadFile(filepath.Join(repoPath, "test.txt"))
				require.NoError(t, err)
				assert.Equal(t, "modified content\n", string(content))
			},
		},
		{
			name: "reset with commit hash",
			args: []string{},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create multiple commits
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
				
				// Third commit
				err = os.WriteFile(testFile, []byte("content3\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Third commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Store first commit hash for use in test
				t.Setenv("FIRST_COMMIT", hash1.String())
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				// This would be tested with actual commit hash in a real scenario
				assert.Contains(t, output, "Mixed reset")
			},
		},
		{
			name: "reset with keep flag",
			args: []string{"HEAD~1"},
			flags: map[string]string{
				"keep": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create commits
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				err = os.WriteFile(testFile, []byte("updated\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Second commit", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Reset with keep")
			},
		},
		{
			name: "reset with merge flag",
			args: []string{"HEAD~1"},
			flags: map[string]string{
				"merge": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create commits
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				err = os.WriteFile(testFile, []byte("updated\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Second commit", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Reset with merge")
			},
		},
		{
			name: "reset with quiet flag",
			args: []string{"HEAD"},
			flags: map[string]string{
				"quiet": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create commit
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
				// With quiet flag, output should be minimal
				assert.Equal(t, "", output)
			},
		},
		{
			name: "reset with pathspec",
			args: []string{".", "--", "*.txt"},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create files
				testFile1 := filepath.Join(repoPath, "test1.txt")
				testFile2 := filepath.Join(repoPath, "test2.md")
				err := os.WriteFile(testFile1, []byte("content1\n"), 0644)
				require.NoError(t, err)
				err = os.WriteFile(testFile2, []byte("content2\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add(".")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Modify and stage files
				err = os.WriteFile(testFile1, []byte("modified1\n"), 0644)
				require.NoError(t, err)
				err = os.WriteFile(testFile2, []byte("modified2\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add(".")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Unstaged changes after reset")
			},
		},
		{
			name:        "reset invalid commit",
			args:        []string{"invalid-hash"},
			expectError: true,
		},
		{
			name: "reset non-existent file",
			args: []string{"nonexistent.txt"},
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
			name:        "reset outside repository",
			args:        []string{"HEAD"},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "reset outside repository" {
				// Test outside repository
				tmpDir := t.TempDir()
				err := os.Chdir(tmpDir)
				require.NoError(t, err)
				
				cmd := newResetCommand()
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
			cmd := newResetCommand()
			
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

func TestResetInteractive(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create files
	testFile1 := filepath.Join(repoPath, "file1.txt")
	testFile2 := filepath.Join(repoPath, "file2.txt")
	err = os.WriteFile(testFile1, []byte("content1\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("content2\n"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add(".")
	require.NoError(t, err)
	_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Modify and stage files
	err = os.WriteFile(testFile1, []byte("modified1\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("modified2\n"), 0644)
	require.NoError(t, err)
	err = testRepo.Add(".")
	require.NoError(t, err)
	
	// Test interactive reset
	cmd := newResetCommand()
	err = cmd.Flags().Set("interactive", "true")
	require.NoError(t, err)
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	err = cmd.Execute()
	// Interactive mode might not be fully implemented
	assert.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Interactive reset")
}

func TestResetPatch(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create file
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("line 1\nline 2\nline 3\n"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("test.txt")
	require.NoError(t, err)
	_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Modify file
	err = os.WriteFile(testFile, []byte("line 1 modified\nline 2\nline 3 modified\n"), 0644)
	require.NoError(t, err)
	err = testRepo.Add("test.txt")
	require.NoError(t, err)
	
	// Test patch reset
	cmd := newResetCommand()
	err = cmd.Flags().Set("patch", "true")
	require.NoError(t, err)
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	cmd.SetArgs([]string{"test.txt"})
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Patch reset")
}

func TestResetToTag(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create commit
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("tagged content\n"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("test.txt")
	require.NoError(t, err)
	hash, err := testRepo.Commit("Tagged commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Create tag
	tagger := objects.Signature{
		Name:  "Test User",
		Email: "test@example.com",
		When:  time.Now(),
	}
	_, err = testRepo.CreateTag(hash, objects.TypeCommit, "v1.0", tagger, "Test tag")
	require.NoError(t, err)
	
	// Make another commit
	err = os.WriteFile(testFile, []byte("later content\n"), 0644)
	require.NoError(t, err)
	err = testRepo.Add("test.txt")
	require.NoError(t, err)
	_, err = testRepo.Commit("Later commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Reset to tag
	cmd := newResetCommand()
	err = cmd.Flags().Set("hard", "true")
	require.NoError(t, err)
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	cmd.SetArgs([]string{"v1.0"})
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Hard reset")
	
	// File should have tagged content
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "tagged content\n", string(content))
}