package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewMergeCommand(t *testing.T) {
	cmd := newMergeCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "merge", cmd.Use)
	assert.Contains(t, cmd.Short, "Join two or more development histories")
}

func TestMergeCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		setupFunc   func(t *testing.T, repo *vcs.Repository, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "merge branch with no conflicts",
			args: []string{"feature-branch"},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit on main
				testFile := filepath.Join(repoPath, "main.txt")
				err := os.WriteFile(testFile, []byte("main content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("main.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create feature branch
				_, err = testRepo.CreateBranch("feature-branch")
				require.NoError(t, err)
				err = testRepo.Checkout("feature-branch")
				require.NoError(t, err)
				
				// Add feature file
				featureFile := filepath.Join(repoPath, "feature.txt")
				err = os.WriteFile(featureFile, []byte("feature content\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("feature.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Add feature", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Switch back to main
				err = testRepo.Checkout("main")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Merge made")
				// Check feature file exists after merge
				assert.FileExists(t, filepath.Join(repoPath, "feature.txt"))
			},
		},
		{
			name: "merge with fast-forward",
			args: []string{"feature-branch"},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create feature branch from current HEAD
				_, err = testRepo.CreateBranch("feature-branch")
				require.NoError(t, err)
				err = testRepo.Checkout("feature-branch")
				require.NoError(t, err)
				
				// Add commit on feature branch
				err = os.WriteFile(testFile, []byte("updated\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Update file", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Switch back to main (which is behind)
				err = testRepo.Checkout("main")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Fast-forward")
			},
		},
		{
			name: "merge with no-ff flag",
			args: []string{"feature-branch"},
			flags: map[string]string{
				"no-ff": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create and switch to feature branch
				_, err = testRepo.CreateBranch("feature-branch")
				require.NoError(t, err)
				err = testRepo.Checkout("feature-branch")
				require.NoError(t, err)
				
				// Add commit
				err = os.WriteFile(testFile, []byte("feature change\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Feature change", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Switch back to main
				err = testRepo.Checkout("main")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Merge made")
				assert.NotContains(t, output, "Fast-forward")
			},
		},
		{
			name: "merge with custom message",
			args: []string{"feature-branch"},
			flags: map[string]string{
				"message": "Custom merge message",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create feature branch
				_, err = testRepo.CreateBranch("feature-branch")
				require.NoError(t, err)
				err = testRepo.Checkout("feature-branch")
				require.NoError(t, err)
				
				// Add different file to avoid fast-forward
				newFile := filepath.Join(repoPath, "new.txt")
				err = os.WriteFile(newFile, []byte("new content\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("new.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Add new file", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Switch back to main and make a commit
				err = testRepo.Checkout("main")
				require.NoError(t, err)
				err = os.WriteFile(testFile, []byte("main update\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Update on main", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Custom merge message")
			},
		},
		{
			name: "merge with squash option",
			args: []string{"feature-branch"},
			flags: map[string]string{
				"squash": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create feature branch
				_, err = testRepo.CreateBranch("feature-branch")
				require.NoError(t, err)
				err = testRepo.Checkout("feature-branch")
				require.NoError(t, err)
				
				// Make multiple commits
				for i := 1; i <= 3; i++ {
					fileName := filepath.Join(repoPath, fmt.Sprintf("file%d.txt", i))
					err = os.WriteFile(fileName, []byte(fmt.Sprintf("content %d\n", i)), 0644)
					require.NoError(t, err)
					err = testRepo.Add(fileName)
					require.NoError(t, err)
					_, err = testRepo.Commit(fmt.Sprintf("Add file%d", i), "Test User", "test@example.com")
					require.NoError(t, err)
				}
				
				// Switch back to main
				err = testRepo.Checkout("main")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Squash commit")
			},
		},
		{
			name: "merge non-existent branch",
			args: []string{"non-existent"},
			expectError: true,
		},
		{
			name: "merge with no arguments",
			args: []string{},
			expectError: true,
		},
		{
			name: "merge with conflict",
			args: []string{"feature-branch"},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "conflict.txt")
				err := os.WriteFile(testFile, []byte("initial content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("conflict.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create feature branch
				_, err = testRepo.CreateBranch("feature-branch")
				require.NoError(t, err)
				err = testRepo.Checkout("feature-branch")
				require.NoError(t, err)
				
				// Modify file on feature branch
				err = os.WriteFile(testFile, []byte("feature change\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("conflict.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Feature change", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Switch back to main and make conflicting change
				err = testRepo.Checkout("main")
				require.NoError(t, err)
				err = os.WriteFile(testFile, []byte("main change\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("conflict.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Main change", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			expectError: true,
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "CONFLICT")
				assert.Contains(t, output, "Automatic merge failed")
			},
		},
		{
			name: "merge with strategy option",
			args: []string{"feature-branch"},
			flags: map[string]string{
				"strategy": "ours",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create feature branch
				_, err = testRepo.CreateBranch("feature-branch")
				require.NoError(t, err)
				err = testRepo.Checkout("feature-branch")
				require.NoError(t, err)
				
				// Make change on feature
				err = os.WriteFile(testFile, []byte("feature\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Feature change", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Switch back to main
				err = testRepo.Checkout("main")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "using the ours strategy")
			},
		},
		{
			name: "merge already up-to-date",
			args: []string{"feature-branch"},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create feature branch at same commit
				_, err = testRepo.CreateBranch("feature-branch")
				require.NoError(t, err)
				// Don't make any new commits
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Already up to date")
			},
		},
		{
			name:        "merge outside repository",
			args:        []string{"branch"},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "merge outside repository" {
				// Test outside repository
				tmpDir := t.TempDir()
				err := os.Chdir(tmpDir)
				require.NoError(t, err)
				
				cmd := newMergeCommand()
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
			cmd := newMergeCommand()
			
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

func TestMergeMultipleBranches(t *testing.T) {
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
	testFile := filepath.Join(repoPath, "base.txt")
	err = os.WriteFile(testFile, []byte("base content\n"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("base.txt")
	require.NoError(t, err)
	_, err = testRepo.Commit("Base commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Create first feature branch
	_, err = testRepo.CreateBranch("feature1")
	require.NoError(t, err)
	err = testRepo.Checkout("feature1")
	require.NoError(t, err)
	
	file1 := filepath.Join(repoPath, "feature1.txt")
	err = os.WriteFile(file1, []byte("feature1 content\n"), 0644)
	require.NoError(t, err)
	err = testRepo.Add("feature1.txt")
	require.NoError(t, err)
	_, err = testRepo.Commit("Add feature1", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Create second feature branch from main
	err = testRepo.Checkout("main")
	require.NoError(t, err)
	_, err = testRepo.CreateBranch("feature2")
	require.NoError(t, err)
	err = testRepo.Checkout("feature2")
	require.NoError(t, err)
	
	file2 := filepath.Join(repoPath, "feature2.txt")
	err = os.WriteFile(file2, []byte("feature2 content\n"), 0644)
	require.NoError(t, err)
	err = testRepo.Add("feature2.txt")
	require.NoError(t, err)
	_, err = testRepo.Commit("Add feature2", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Switch back to main
	err = testRepo.Checkout("main")
	require.NoError(t, err)
	
	// Try to merge multiple branches
	cmd := newMergeCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	cmd.SetArgs([]string{"feature1", "feature2"})
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Octopus merge")
}

func TestMergeAbort(t *testing.T) {
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
	testFile := filepath.Join(repoPath, "conflict.txt")
	err = os.WriteFile(testFile, []byte("initial\n"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("conflict.txt")
	require.NoError(t, err)
	_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Create conflicting branch
	_, err = testRepo.CreateBranch("conflict-branch")
	require.NoError(t, err)
	err = testRepo.Checkout("conflict-branch")
	require.NoError(t, err)
	
	err = os.WriteFile(testFile, []byte("branch change\n"), 0644)
	require.NoError(t, err)
	err = testRepo.Add("conflict.txt")
	require.NoError(t, err)
	_, err = testRepo.Commit("Branch change", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Make conflicting change on main
	err = testRepo.Checkout("main")
	require.NoError(t, err)
	err = os.WriteFile(testFile, []byte("main change\n"), 0644)
	require.NoError(t, err)
	err = testRepo.Add("conflict.txt")
	require.NoError(t, err)
	_, err = testRepo.Commit("Main change", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Try to merge - should fail with conflict
	cmd := newMergeCommand()
	cmd.SetArgs([]string{"conflict-branch"})
	err = cmd.Execute()
	assert.Error(t, err)
	
	// Create MERGE_HEAD to simulate merge in progress
	mergeHeadPath := filepath.Join(repoPath, ".git", "MERGE_HEAD")
	err = os.WriteFile(mergeHeadPath, []byte("dummy-hash\n"), 0644)
	require.NoError(t, err)
	
	// Test merge --abort
	cmd = newMergeCommand()
	err = cmd.Flags().Set("abort", "true")
	require.NoError(t, err)
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Merge aborted")
	
	// Check MERGE_HEAD was removed
	assert.NoFileExists(t, mergeHeadPath)
}

func TestMergeContinue(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create MERGE_HEAD to simulate merge in progress
	gitDir := filepath.Join(repoPath, ".git")
	mergeHeadPath := filepath.Join(gitDir, "MERGE_HEAD")
	err = os.WriteFile(mergeHeadPath, []byte("dummy-hash\n"), 0644)
	require.NoError(t, err)
	
	// Test merge --continue
	cmd := newMergeCommand()
	err = cmd.Flags().Set("continue", "true")
	require.NoError(t, err)
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	err = cmd.Execute()
	// Should fail because there's no real merge in progress
	assert.Error(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "No merge in progress")
}