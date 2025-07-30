package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestDiffCommandDetailed(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		setupFunc   func(t *testing.T, repo *vcs.Repository, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string)
	}{
		{
			name: "diff with no changes",
			args: []string{},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create and commit a file
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string) {
				// No changes, output should be empty
				assert.Empty(t, strings.TrimSpace(output))
			},
		},
		{
			name: "diff with unstaged changes",
			args: []string{},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create and commit a file
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Modify the file
				err = os.WriteFile(testFile, []byte("modified content\n"), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "diff --git a/test.txt b/test.txt")
				assert.Contains(t, output, "-initial content")
				assert.Contains(t, output, "+modified content")
			},
		},
		{
			name: "diff with staged changes",
			args: []string{},
			flags: map[string]string{
				"cached": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create and commit a file
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Modify and stage the file
				err = os.WriteFile(testFile, []byte("staged content\n"), 0644)
				require.NoError(t, err)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "diff --git a/test.txt b/test.txt")
				assert.Contains(t, output, "-initial content")
				assert.Contains(t, output, "+staged content")
			},
		},
		{
			name: "diff specific file",
			args: []string{"test.txt"},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create and commit multiple files
				testFile1 := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile1, []byte("test content\n"), 0644)
				require.NoError(t, err)
				
				testFile2 := filepath.Join(repoPath, "other.txt")
				err = os.WriteFile(testFile2, []byte("other content\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add(".")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Modify both files
				err = os.WriteFile(testFile1, []byte("modified test\n"), 0644)
				require.NoError(t, err)
				err = os.WriteFile(testFile2, []byte("modified other\n"), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string) {
				// Should only show diff for test.txt
				assert.Contains(t, output, "test.txt")
				assert.NotContains(t, output, "other.txt")
			},
		},
		{
			name: "diff with name-only",
			args: []string{},
			flags: map[string]string{
				"name-only": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create and commit a file
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Modify the file
				err = os.WriteFile(testFile, []byte("modified\n"), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string) {
				output = strings.TrimSpace(output)
				assert.Equal(t, "test.txt", output)
				assert.NotContains(t, output, "diff --git")
			},
		},
		{
			name: "diff with name-status",
			args: []string{},
			flags: map[string]string{
				"name-status": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create and commit a file
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("initial\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Modify the file
				err = os.WriteFile(testFile, []byte("modified\n"), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string) {
				output = strings.TrimSpace(output)
				assert.Equal(t, "M\ttest.txt", output)
			},
		},
		{
			name: "diff with stat",
			args: []string{},
			flags: map[string]string{
				"stat": "true",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create and commit a file
				testFile := filepath.Join(repoPath, "test.txt")
				err := os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Modify the file
				err = os.WriteFile(testFile, []byte("line1\nmodified\nline3\nnew line\n"), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "test.txt")
				assert.Contains(t, output, "|")
				assert.Contains(t, output, "+")
				assert.Contains(t, output, "-")
			},
		},
		{
			name: "diff with new file",
			args: []string{},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create and commit initial file
				testFile := filepath.Join(repoPath, "existing.txt")
				err := os.WriteFile(testFile, []byte("existing\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("existing.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Create new file
				newFile := filepath.Join(repoPath, "new.txt")
				err = os.WriteFile(newFile, []byte("new content\n"), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string) {
				// Untracked files appear in diff
				assert.Contains(t, output, "new.txt")
				assert.Contains(t, output, "new file mode")
				assert.Contains(t, output, "+new content")
			},
		},
		{
			name: "diff with deleted file",
			args: []string{},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create and commit a file
				testFile := filepath.Join(repoPath, "to-delete.txt")
				err := os.WriteFile(testFile, []byte("will be deleted\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("to-delete.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Delete the file
				err = os.Remove(testFile)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "diff --git a/to-delete.txt b/to-delete.txt")
				assert.Contains(t, output, "deleted file")
				assert.Contains(t, output, "-will be deleted")
			},
		},
		{
			name: "diff with unified context",
			args: []string{},
			flags: map[string]string{
				"unified": "1",
			},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create file with multiple lines
				testFile := filepath.Join(repoPath, "multiline.txt")
				content := []string{}
				for i := 1; i <= 10; i++ {
					content = append(content, fmt.Sprintf("line %d", i))
				}
				err := os.WriteFile(testFile, []byte(strings.Join(content, "\n")+"\n"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("multiline.txt")
				require.NoError(t, err)
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Modify line in the middle
				content[4] = "modified line 5"
				err = os.WriteFile(testFile, []byte(strings.Join(content, "\n")+"\n"), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string) {
				// With context=1, should show 1 line before and after
				assert.Contains(t, output, "line 4")
				assert.Contains(t, output, "line 6")
				assert.Contains(t, output, "-line 5")
				assert.Contains(t, output, "+modified line 5")
			},
		},
		{
			name:        "diff outside repository",
			args:        []string{},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "diff outside repository" {
				// Test outside repository
				tmpDir := t.TempDir()
				err := os.Chdir(tmpDir)
				require.NoError(t, err)
				
				cmd := newDiffCommand()
				var buf bytes.Buffer
				cmd.SetOut(&buf)
				cmd.SetErr(&buf)
				
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
			cmd := newDiffCommand()
			
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
				
				// Check output
				if tc.checkFunc != nil {
					tc.checkFunc(t, buf.String())
				}
			}
		})
	}
}

func TestDiffBinaryFilesDetailed(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create binary file
	binaryFile := filepath.Join(repoPath, "binary.dat")
	binaryContent := []byte{0x00, 0xFF, 0x42, 0x13, 0x37, 0x00, 0xAB, 0xCD}
	err = os.WriteFile(binaryFile, binaryContent, 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("binary.dat")
	require.NoError(t, err)
	_, err = testRepo.Commit("Add binary", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Modify binary file
	newBinaryContent := []byte{0xFF, 0x00, 0x13, 0x42, 0x00, 0x37, 0xCD, 0xAB}
	err = os.WriteFile(binaryFile, newBinaryContent, 0644)
	require.NoError(t, err)
	
	// Run diff
	cmd := newDiffCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Binary files")
	assert.Contains(t, output, "differ")
}

func TestDiffPermissionChanges(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create file with normal permissions
	testFile := filepath.Join(repoPath, "script.sh")
	err = os.WriteFile(testFile, []byte("#!/bin/bash\necho 'Hello'\n"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("script.sh")
	require.NoError(t, err)
	_, err = testRepo.Commit("Add script", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Make file executable
	err = os.Chmod(testFile, 0755)
	require.NoError(t, err)
	
	// Stage the permission change
	err = testRepo.Add("script.sh")
	require.NoError(t, err)
	
	// Run diff --cached to see staged changes
	cmd := newDiffCommand()
	err = cmd.Flags().Set("cached", "true")
	require.NoError(t, err)
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	// Should show mode change
	assert.Contains(t, output, "mode")
}