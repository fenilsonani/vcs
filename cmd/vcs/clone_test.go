package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewCloneCommand(t *testing.T) {
	cmd := newCloneCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "clone", cmd.Use)
	assert.Contains(t, cmd.Short, "Clone a repository")
}

func TestCloneCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		setupFunc   func(t *testing.T, tmpDir string) string
		expectError bool
		checkFunc   func(t *testing.T, output string, targetPath string)
	}{
		{
			name: "clone to default directory",
			args: []string{"https://github.com/user/repo.git"},
			checkFunc: func(t *testing.T, output string, targetPath string) {
				assert.Contains(t, output, "Cloning into 'repo'")
				assert.Contains(t, output, "Note: This is a skeleton clone implementation")
				
				// Check directory was created
				expectedPath := filepath.Join(filepath.Dir(targetPath), "repo")
				assert.DirExists(t, expectedPath)
				assert.DirExists(t, filepath.Join(expectedPath, ".git"))
				
				// Check remote was configured
				configPath := filepath.Join(expectedPath, ".git", "config")
				content, err := os.ReadFile(configPath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "[remote \"origin\"]")
				assert.Contains(t, string(content), "url = https://github.com/user/repo.git")
			},
		},
		{
			name: "clone to specific directory",
			args: []string{"https://github.com/user/repo.git", "myrepo"},
			checkFunc: func(t *testing.T, output string, targetPath string) {
				assert.Contains(t, output, "Cloning into 'myrepo'")
				
				// Check directory was created
				expectedPath := filepath.Join(filepath.Dir(targetPath), "myrepo")
				assert.DirExists(t, expectedPath)
				assert.DirExists(t, filepath.Join(expectedPath, ".git"))
			},
		},
		{
			name: "clone with bare option",
			args: []string{"https://github.com/user/repo.git"},
			flags: map[string]string{
				"bare": "true",
			},
			checkFunc: func(t *testing.T, output string, targetPath string) {
				assert.Contains(t, output, "Cloning into bare repository 'repo'")
				
				// Check bare repository structure
				expectedPath := filepath.Join(filepath.Dir(targetPath), "repo")
				assert.DirExists(t, expectedPath)
				// In a bare repo, objects/ refs/ etc are at the top level
				assert.DirExists(t, filepath.Join(expectedPath, "objects"))
				assert.DirExists(t, filepath.Join(expectedPath, "refs"))
				assert.NoFileExists(t, filepath.Join(expectedPath, ".git"))
			},
		},
		{
			name: "clone with quiet option",
			args: []string{"https://github.com/user/repo.git"},
			flags: map[string]string{
				"quiet": "true",
			},
			checkFunc: func(t *testing.T, output string, targetPath string) {
				// With quiet option, output should be minimal
				assert.NotContains(t, output, "remote: Enumerating objects")
				assert.NotContains(t, output, "remote: Counting objects")
			},
		},
		{
			name: "clone with depth option",
			args: []string{"https://github.com/user/repo.git"},
			flags: map[string]string{
				"depth": "1",
			},
			checkFunc: func(t *testing.T, output string, targetPath string) {
				assert.Contains(t, output, "Cloning into 'repo'")
				// Would check shallow file in real implementation
			},
		},
		{
			name: "clone with branch option",
			args: []string{"https://github.com/user/repo.git"},
			flags: map[string]string{
				"branch": "develop",
			},
			checkFunc: func(t *testing.T, output string, targetPath string) {
				assert.Contains(t, output, "Cloning into 'repo'")
				// Would check that develop branch is checked out
			},
		},
		{
			name: "clone with no-checkout option",
			args: []string{"https://github.com/user/repo.git"},
			flags: map[string]string{
				"no-checkout": "true",
			},
			checkFunc: func(t *testing.T, output string, targetPath string) {
				assert.Contains(t, output, "Cloning into 'repo'")
				// Working directory should be empty except .git
			},
		},
		{
			name: "clone to existing non-empty directory",
			args: []string{"https://github.com/user/repo.git", "existing"},
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Create existing directory with file
				existingDir := filepath.Join(tmpDir, "existing")
				err := os.MkdirAll(existingDir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(existingDir, "file.txt"), []byte("content"), 0644)
				require.NoError(t, err)
				return existingDir
			},
			expectError: true,
		},
		{
			name:        "clone with no URL",
			args:        []string{},
			expectError: true,
		},
		{
			name: "clone with invalid URL",
			args: []string{"not-a-url"},
			checkFunc: func(t *testing.T, output string, targetPath string) {
				// Should still create directory structure
				assert.Contains(t, output, "Cloning into")
			},
		},
		{
			name: "clone SSH URL",
			args: []string{"git@github.com:user/repo.git"},
			checkFunc: func(t *testing.T, output string, targetPath string) {
				assert.Contains(t, output, "Cloning into 'repo'")
				
				// Check remote was configured with SSH URL
				expectedPath := filepath.Join(filepath.Dir(targetPath), "repo")
				configPath := filepath.Join(expectedPath, ".git", "config")
				content, err := os.ReadFile(configPath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "url = git@github.com:user/repo.git")
			},
		},
		{
			name: "clone with custom origin name",
			args: []string{"https://github.com/user/repo.git"},
			flags: map[string]string{
				"origin": "upstream",
			},
			checkFunc: func(t *testing.T, output string, targetPath string) {
				assert.Contains(t, output, "Cloning into 'repo'")
				
				// Check remote was configured with custom name
				expectedPath := filepath.Join(filepath.Dir(targetPath), "repo")
				configPath := filepath.Join(expectedPath, ".git", "config")
				content, err := os.ReadFile(configPath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "[remote \"upstream\"]")
				assert.NotContains(t, string(content), "[remote \"origin\"]")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			
			// Change to temp directory
			err := os.Chdir(tmpDir)
			require.NoError(t, err)
			
			// Run setup if provided
			var targetPath string
			if tc.setupFunc != nil {
				targetPath = tc.setupFunc(t, tmpDir)
			} else {
				targetPath = tmpDir
			}
			
			// Create command
			cmd := newCloneCommand()
			
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
				output := buf.String()
				if tc.checkFunc != nil {
					tc.checkFunc(t, output, targetPath)
				}
			}
		})
	}
}

// extractRepoName extracts repository name from URL
func extractRepoName(url string) string {
	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")
	
	// Get last part of path
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return "repo"
	}
	
	name := parts[len(parts)-1]
	
	// Handle git@github.com:user/repo.git format
	if strings.Contains(name, ":") {
		colonParts := strings.Split(name, ":")
		if len(colonParts) > 1 {
			name = colonParts[len(colonParts)-1]
		}
	}
	
	// Remove .git suffix
	name = strings.TrimSuffix(name, ".git")
	
	if name == "" {
		return "repo"
	}
	
	return name
}

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTPS URL with .git",
			url:      "https://github.com/user/repo.git",
			expected: "repo",
		},
		{
			name:     "HTTPS URL without .git",
			url:      "https://github.com/user/repo",
			expected: "repo",
		},
		{
			name:     "SSH URL",
			url:      "git@github.com:user/repo.git",
			expected: "repo",
		},
		{
			name:     "URL with subdirectories",
			url:      "https://github.com/org/team/repo.git",
			expected: "repo",
		},
		{
			name:     "URL with port",
			url:      "https://github.com:8080/user/repo.git",
			expected: "repo",
		},
		{
			name:     "Local path",
			url:      "/path/to/repo.git",
			expected: "repo",
		},
		{
			name:     "Local path without .git",
			url:      "/path/to/repo",
			expected: "repo",
		},
		{
			name:     "URL with trailing slash",
			url:      "https://github.com/user/repo.git/",
			expected: "repo",
		},
		{
			name:     "Just repo name",
			url:      "repo",
			expected: "repo",
		},
		{
			name:     "Repo name with .git",
			url:      "repo.git",
			expected: "repo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractRepoName(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCloneWithProgress(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	
	// Change to temp directory
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create command
	cmd := newCloneCommand()
	
	// Enable progress
	err = cmd.Flags().Set("progress", "true")
	require.NoError(t, err)
	
	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	// Execute command
	cmd.SetArgs([]string{"https://github.com/user/repo.git"})
	err = cmd.Execute()
	assert.NoError(t, err)
	
	// Check output includes progress indicators
	output := buf.String()
	assert.Contains(t, output, "Enumerating objects")
	assert.Contains(t, output, "Counting objects")
	assert.Contains(t, output, "Receiving objects")
}

func TestCloneEmptyRepository(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	
	// Create a source bare repository
	sourceRepo := filepath.Join(tmpDir, "source.git")
	_, err := vcs.Init(sourceRepo)
	require.NoError(t, err)
	
	// Make it bare
	err = os.Rename(filepath.Join(sourceRepo, ".git"), filepath.Join(tmpDir, "temp"))
	require.NoError(t, err)
	err = os.RemoveAll(sourceRepo)
	require.NoError(t, err)
	err = os.Rename(filepath.Join(tmpDir, "temp"), sourceRepo)
	require.NoError(t, err)
	
	// Change to temp directory
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create command
	cmd := newCloneCommand()
	
	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	// Execute command
	cmd.SetArgs([]string{sourceRepo, "cloned"})
	err = cmd.Execute()
	assert.NoError(t, err)
	
	// Check output
	output := buf.String()
	assert.Contains(t, output, "Cloning into 'cloned'")
	assert.Contains(t, output, "warning: You appear to have cloned an empty repository")
}

func TestCloneValidation(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	
	// Change to temp directory
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Test with both depth and single-branch
	cmd := newCloneCommand()
	err = cmd.Flags().Set("depth", "1")
	require.NoError(t, err)
	err = cmd.Flags().Set("single-branch", "true")
	require.NoError(t, err)
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	cmd.SetArgs([]string{"https://github.com/user/repo.git"})
	err = cmd.Execute()
	assert.NoError(t, err)
	
	// Both flags should work together
	output := buf.String()
	assert.Contains(t, output, "Cloning into 'repo'")
}