package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewFetchCommand(t *testing.T) {
	cmd := newFetchCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "fetch", cmd.Use)
	assert.Contains(t, cmd.Short, "Download objects and refs")
}

func TestFetchCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		setupFunc   func(t *testing.T, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "fetch from origin",
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
				assert.Contains(t, output, "Fetching from https://github.com/example/repo.git")
				assert.Contains(t, output, "remote: Enumerating objects")
				
				// Check FETCH_HEAD was created
				fetchHeadPath := filepath.Join(repoPath, ".git", "FETCH_HEAD")
				assert.FileExists(t, fetchHeadPath)
				
				// Check remote refs directory was created
				remoteRefsDir := filepath.Join(repoPath, ".git", "refs", "remotes", "origin")
				assert.DirExists(t, remoteRefsDir)
			},
		},
		{
			name: "fetch from specific remote",
			args: []string{"upstream"},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add upstream remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "upstream"]
	url = https://github.com/upstream/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Fetching from https://github.com/upstream/repo.git")
				
				// Check remote refs directory was created
				remoteRefsDir := filepath.Join(repoPath, ".git", "refs", "remotes", "upstream")
				assert.DirExists(t, remoteRefsDir)
			},
		},
		{
			name: "fetch with all branches",
			args: []string{},
			flags: map[string]string{
				"all": "true",
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
				assert.Contains(t, output, "Fetching all remotes")
			},
		},
		{
			name: "fetch with verbose",
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
				assert.Contains(t, output, "POST")
				assert.Contains(t, output, "git-upload-pack")
			},
		},
		{
			name: "fetch with prune",
			args: []string{},
			flags: map[string]string{
				"prune": "true",
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
				assert.Contains(t, output, "Pruning remote references")
			},
		},
		{
			name: "fetch with depth",
			args: []string{},
			flags: map[string]string{
				"depth": "1",
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
				assert.Contains(t, output, "Shallow fetch with depth 1")
			},
		},
		{
			name:        "fetch from non-existent remote",
			args:        []string{"nonexistent"},
			setupFunc:   func(t *testing.T, repoPath string) {},
			expectError: true,
		},
		{
			name:        "fetch outside repository",
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
			if tc.name != "fetch outside repository" {
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
			cmd := newFetchCommand()
			
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

func TestFetchFromRemote(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Create command for output
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	// Test fetch
	err = fetchFromRemote(cmd, repo, "origin", "https://github.com/example/repo.git", false, false, false, 0, true)
	assert.NoError(t, err)
	
	// Check output
	output := buf.String()
	assert.Contains(t, output, "POST https://github.com/example/repo.git/git-upload-pack")
	assert.Contains(t, output, "remote: Enumerating objects")
	
	// Check FETCH_HEAD
	fetchHeadPath := filepath.Join(repo.GitDir(), "FETCH_HEAD")
	assert.FileExists(t, fetchHeadPath)
	
	// Check remote refs directory
	remoteRefsDir := filepath.Join(repo.GitDir(), "refs", "remotes", "origin")
	assert.DirExists(t, remoteRefsDir)
}

func TestGetRemotes(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Test with no remotes
	remotes, err := getRemotes(repo)
	assert.NoError(t, err)
	assert.Empty(t, remotes)
	
	// Add remotes to config
	configPath := filepath.Join(repo.GitDir(), "config")
	configContent := `[remote "origin"]
	url = https://github.com/example/repo.git
[remote "upstream"]
	url = https://github.com/upstream/repo.git
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	
	// Test with remotes
	remotes, err = getRemotes(repo)
	assert.NoError(t, err)
	assert.Len(t, remotes, 2)
	assert.Equal(t, "https://github.com/example/repo.git", remotes["origin"])
	assert.Equal(t, "https://github.com/upstream/repo.git", remotes["upstream"])
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectedRemotes map[string]string
	}{
		{
			name: "single remote",
			configContent: `[remote "origin"]
	url = https://github.com/example/repo.git
`,
			expectedRemotes: map[string]string{
				"origin": "https://github.com/example/repo.git",
			},
		},
		{
			name: "multiple remotes",
			configContent: `[remote "origin"]
	url = https://github.com/example/repo.git
[remote "upstream"]
	url = https://github.com/upstream/repo.git
[remote "fork"]
	url = git@github.com:user/fork.git
`,
			expectedRemotes: map[string]string{
				"origin":   "https://github.com/example/repo.git",
				"upstream": "https://github.com/upstream/repo.git",
				"fork":     "git@github.com:user/fork.git",
			},
		},
		{
			name: "remote with fetch config",
			configContent: `[remote "origin"]
	url = https://github.com/example/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`,
			expectedRemotes: map[string]string{
				"origin": "https://github.com/example/repo.git",
			},
		},
		{
			name: "mixed config sections",
			configContent: `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/example/repo.git
[branch "main"]
	remote = origin
[remote "backup"]
	url = /path/to/backup.git
`,
			expectedRemotes: map[string]string{
				"origin": "https://github.com/example/repo.git",
				"backup": "/path/to/backup.git",
			},
		},
		{
			name:            "empty config",
			configContent:   "",
			expectedRemotes: map[string]string{},
		},
		{
			name: "malformed remote section",
			configContent: `[remote "origin"
	url = https://github.com/example/repo.git
`,
			expectedRemotes: map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			remotes := parseConfig([]byte(tc.configContent))
			assert.Equal(t, tc.expectedRemotes, remotes)
		})
	}
}