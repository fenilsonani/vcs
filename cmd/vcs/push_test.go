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

func TestNewPushCommand(t *testing.T) {
	cmd := newPushCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "push", cmd.Use)
	assert.Contains(t, cmd.Short, "Update remote refs")
}

func TestPushCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		setupFunc   func(t *testing.T, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "push to origin",
			args: []string{},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/example/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Pushing to https://github.com/example/repo.git")
				assert.Contains(t, output, "main -> main")
			},
		},
		{
			name: "push specific branch",
			args: []string{"origin", "feature"},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/example/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
				
				// Create feature branch
				repo, err := vcs.Open(repoPath)
				require.NoError(t, err)
				testRepo := WrapRepository(repo, repoPath)
				_, err = testRepo.CreateBranch("feature")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "feature -> feature")
			},
		},
		{
			name: "push with refspec",
			args: []string{"origin", "main:develop"},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/example/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "main -> develop")
			},
		},
		{
			name: "force push",
			args: []string{},
			flags: map[string]string{
				"force": "true",
			},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/example/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "forced update")
			},
		},
		{
			name: "push with set-upstream",
			args: []string{},
			flags: map[string]string{
				"set-upstream": "true",
			},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/example/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Branch 'main' set up to track remote branch")
				
				// Check config was updated
				configPath := filepath.Join(repoPath, ".git", "config")
				content, err := os.ReadFile(configPath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "[branch \"main\"]")
			},
		},
		{
			name: "dry run push",
			args: []string{},
			flags: map[string]string{
				"dry-run": "true",
			},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/example/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Dry run mode")
				assert.Contains(t, output, "[dry-run]")
			},
		},
		{
			name: "verbose push",
			args: []string{},
			flags: map[string]string{
				"verbose": "true",
			},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/example/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Enumerating objects")
				assert.Contains(t, output, "Counting objects")
				assert.Contains(t, output, "Writing objects")
			},
		},
		{
			name: "push non-existent branch",
			args: []string{"origin", "nonexistent"},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/example/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "[rejected]")
				assert.Contains(t, output, "no such ref")
			},
		},
		{
			name:        "push to non-existent remote",
			args:        []string{"nonexistent"},
			setupFunc:   func(t *testing.T, repoPath string) {},
			expectError: true,
		},
		{
			name:        "push outside repository",
			args:        []string{},
			setupFunc:   func(t *testing.T, repoPath string) {},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			
			var repoPath string
			if tc.name != "push outside repository" {
				// Initialize repository
				repoPath = filepath.Join(tmpDir, "test-repo")
				repo, err := vcs.Init(repoPath)
				require.NoError(t, err)
				
				// Make initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				err = os.WriteFile(testFile, []byte("test content"), 0644)
				require.NoError(t, err)
				
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
				require.NoError(t, err)
				
				_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
				require.NoError(t, err)
				
				// Run setup function
				if tc.setupFunc != nil {
					tc.setupFunc(t, repoPath)
				}
				
				// Change to repo directory
				err = os.Chdir(repoPath)
				require.NoError(t, err)
			} else {
				// Stay in temp directory (outside repository)
				err := os.Chdir(tmpDir)
				require.NoError(t, err)
			}
			
			// Create command
			cmd := newPushCommand()
			
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
			err := cmd.Execute()
			
			// Check error
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Check output
				output := buf.String()
				if tc.checkFunc != nil {
					tc.checkFunc(t, output, repoPath)
				}
			}
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Make initial commit
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("test.txt")
	require.NoError(t, err)
	
	_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Test getting current branch
	branch, err := getCurrentBranch(repo)
	assert.NoError(t, err)
	assert.Equal(t, "main", branch)
	
	// Create and checkout new branch
	_, err = testRepo.CreateBranch("feature")
	require.NoError(t, err)
	err = testRepo.Checkout("feature")
	require.NoError(t, err)
	
	// Test again
	branch, err = getCurrentBranch(repo)
	assert.NoError(t, err)
	assert.Equal(t, "feature", branch)
}

func TestParseRefspec(t *testing.T) {
	tests := []struct {
		name         string
		refspec      string
		expectedLocal string
		expectedRemote string
	}{
		{
			name:          "simple refspec",
			refspec:       "main:main",
			expectedLocal: "main",
			expectedRemote: "main",
		},
		{
			name:          "different names",
			refspec:       "local:remote",
			expectedLocal: "local",
			expectedRemote: "remote",
		},
		{
			name:          "force refspec",
			refspec:       "+main:main",
			expectedLocal: "main",
			expectedRemote: "main",
		},
		{
			name:          "no colon",
			refspec:       "main",
			expectedLocal: "main",
			expectedRemote: "main",
		},
		{
			name:          "full refs",
			refspec:       "refs/heads/main:refs/heads/develop",
			expectedLocal: "refs/heads/main",
			expectedRemote: "refs/heads/develop",
		},
		{
			name:          "HEAD to branch",
			refspec:       "HEAD:refs/heads/feature",
			expectedLocal: "HEAD",
			expectedRemote: "refs/heads/feature",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			local, remote := parseRefspec(tc.refspec)
			assert.Equal(t, tc.expectedLocal, local)
			assert.Equal(t, tc.expectedRemote, remote)
		})
	}
}

func TestSetUpstreamBranch(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Test setting upstream
	err = setUpstreamBranch(repo, "main", "origin", "main")
	assert.NoError(t, err)
	
	// Check config was updated
	configPath := filepath.Join(repo.GitDir(), "config")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	
	assert.Contains(t, string(content), "[branch \"main\"]")
	assert.Contains(t, string(content), "remote = origin")
	assert.Contains(t, string(content), "merge = refs/heads/main")
	
	// Test setting upstream for different branch
	err = setUpstreamBranch(repo, "feature", "upstream", "develop")
	assert.NoError(t, err)
	
	// Check config again
	content, err = os.ReadFile(configPath)
	require.NoError(t, err)
	
	assert.Contains(t, string(content), "[branch \"feature\"]")
	assert.Contains(t, string(content), "remote = upstream")
	assert.Contains(t, string(content), "merge = refs/heads/develop")
}

func TestPushToRemote(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Make initial commit
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("test.txt")
	require.NoError(t, err)
	
	_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Create command for output
	cmd := newPushCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	// Test push
	err = pushToRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
		[]string{"main"}, false, false, false, false, false, true)
	assert.NoError(t, err)
	
	// Check output
	output := buf.String()
	assert.Contains(t, output, "Enumerating objects")
	assert.Contains(t, output, "To https://github.com/example/repo.git")
	assert.Contains(t, output, "main -> main")
	
	// Test dry run
	buf.Reset()
	err = pushToRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
		[]string{"main"}, false, false, false, false, true, false)
	assert.NoError(t, err)
	
	output = buf.String()
	assert.Contains(t, output, "Dry run mode")
	assert.Contains(t, output, "[dry-run]")
	
	// Test force push
	buf.Reset()
	err = pushToRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
		[]string{"main"}, true, false, false, false, false, false)
	assert.NoError(t, err)
	
	output = buf.String()
	assert.Contains(t, output, "forced update")
	
	// Test with non-existent branch
	buf.Reset()
	err = pushToRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
		[]string{"nonexistent"}, false, false, false, false, false, false)
	assert.NoError(t, err) // No error at command level
	
	output = buf.String()
	assert.Contains(t, output, "[rejected]")
	assert.Contains(t, output, "no such ref")
}

func TestMultipleRefspecs(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Make initial commit
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
	err = testRepo.Add("test.txt")
	require.NoError(t, err)
	
	_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Create additional branches
	_, err = testRepo.CreateBranch("feature1")
	require.NoError(t, err)
	_, err = testRepo.CreateBranch("feature2")
	require.NoError(t, err)
	
	// Create command for output
	cmd := newPushCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	// Test push with multiple refspecs
	err = pushToRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
		[]string{"main", "feature1:feat1", "feature2"}, false, false, false, false, false, false)
	assert.NoError(t, err)
	
	// Check output
	output := buf.String()
	lines := strings.Split(output, "\n")
	
	// Count push results
	pushCount := 0
	for _, line := range lines {
		if strings.Contains(line, "->") && !strings.Contains(line, "[rejected]") {
			pushCount++
		}
	}
	assert.Equal(t, 3, pushCount)
	
	assert.Contains(t, output, "main -> main")
	assert.Contains(t, output, "feature1 -> feat1")
	assert.Contains(t, output, "feature2 -> feature2")
}