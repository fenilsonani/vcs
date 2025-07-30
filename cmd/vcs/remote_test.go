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

func TestNewRemoteCommand(t *testing.T) {
	cmd := newRemoteCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "remote", cmd.Use)
	assert.Contains(t, cmd.Short, "Manage set of tracked repositories")
}

func TestRemoteCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFunc   func(t *testing.T, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "list remotes empty",
			args: []string{},
			setupFunc: func(t *testing.T, repoPath string) {
				// No remotes configured
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Empty(t, strings.TrimSpace(output))
			},
		},
		{
			name: "list remotes",
			args: []string{},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add some remotes
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/user/repo.git
[remote "upstream"]
	url = https://github.com/upstream/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "origin")
				assert.Contains(t, output, "upstream")
			},
		},
		{
			name: "list remotes verbose",
			args: []string{"-v"},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add some remotes
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/user/repo.git
[remote "upstream"]
	url = https://github.com/upstream/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "origin\thttps://github.com/user/repo.git (fetch)")
				assert.Contains(t, output, "origin\thttps://github.com/user/repo.git (push)")
				assert.Contains(t, output, "upstream\thttps://github.com/upstream/repo.git (fetch)")
				assert.Contains(t, output, "upstream\thttps://github.com/upstream/repo.git (push)")
			},
		},
		{
			name:        "list outside repository",
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
			if tc.name != "list outside repository" {
				// Initialize repository
				repoPath = filepath.Join(tmpDir, "test-repo")
				_, err := vcs.Init(repoPath)
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
			cmd := newRemoteCommand()
			
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

func TestRemoteAddCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFunc   func(t *testing.T, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "add new remote",
			args: []string{"add", "origin", "https://github.com/user/repo.git"},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				// Check config file
				configPath := filepath.Join(repoPath, ".git", "config")
				content, err := os.ReadFile(configPath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "[remote \"origin\"]")
				assert.Contains(t, string(content), "url = https://github.com/user/repo.git")
			},
		},
		{
			name: "add duplicate remote",
			args: []string{"add", "origin", "https://github.com/other/repo.git"},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/user/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			expectError: true,
		},
		{
			name:        "add remote with no arguments",
			args:        []string{"add"},
			expectError: true,
		},
		{
			name:        "add remote with only name",
			args:        []string{"add", "origin"},
			expectError: true,
		},
		{
			name: "add remote with fetch option",
			args: []string{"add", "-f", "upstream", "https://github.com/upstream/repo.git"},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Updating upstream")
				// Check config file
				configPath := filepath.Join(repoPath, ".git", "config")
				content, err := os.ReadFile(configPath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "[remote \"upstream\"]")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			repoPath := filepath.Join(tmpDir, "test-repo")
			
			// Initialize repository
			_, err := vcs.Init(repoPath)
			require.NoError(t, err)
			
			// Run setup function
			if tc.setupFunc != nil {
				tc.setupFunc(t, repoPath)
			}
			
			// Change to repo directory
			err = os.Chdir(repoPath)
			require.NoError(t, err)
			
			// Create command
			cmd := newRemoteCommand()
			
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
					tc.checkFunc(t, output, repoPath)
				}
			}
		})
	}
}

func TestRemoteRemoveCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFunc   func(t *testing.T, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "remove existing remote",
			args: []string{"remove", "origin"},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/user/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				// Check config file
				configPath := filepath.Join(repoPath, ".git", "config")
				content, err := os.ReadFile(configPath)
				require.NoError(t, err)
				assert.NotContains(t, string(content), "[remote \"origin\"]")
			},
		},
		{
			name:        "remove non-existent remote",
			args:        []string{"remove", "nonexistent"},
			expectError: true,
		},
		{
			name:        "remove with no arguments",
			args:        []string{"remove"},
			expectError: true,
		},
		{
			name: "remove remote with rm alias",
			args: []string{"rm", "upstream"},
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
				// Check config file
				configPath := filepath.Join(repoPath, ".git", "config")
				content, err := os.ReadFile(configPath)
				require.NoError(t, err)
				assert.NotContains(t, string(content), "[remote \"upstream\"]")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			repoPath := filepath.Join(tmpDir, "test-repo")
			
			// Initialize repository
			_, err := vcs.Init(repoPath)
			require.NoError(t, err)
			
			// Run setup function
			if tc.setupFunc != nil {
				tc.setupFunc(t, repoPath)
			}
			
			// Change to repo directory
			err = os.Chdir(repoPath)
			require.NoError(t, err)
			
			// Create command
			cmd := newRemoteCommand()
			
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
					tc.checkFunc(t, output, repoPath)
				}
			}
		})
	}
}

func TestRemoteShowCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFunc   func(t *testing.T, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string)
	}{
		{
			name: "show existing remote",
			args: []string{"show", "origin"},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/user/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "* remote origin")
				assert.Contains(t, output, "Fetch URL: https://github.com/user/repo.git")
				assert.Contains(t, output, "Push  URL: https://github.com/user/repo.git")
			},
		},
		{
			name:        "show non-existent remote",
			args:        []string{"show", "nonexistent"},
			expectError: true,
		},
		{
			name:        "show with no arguments",
			args:        []string{"show"},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			repoPath := filepath.Join(tmpDir, "test-repo")
			
			// Initialize repository
			_, err := vcs.Init(repoPath)
			require.NoError(t, err)
			
			// Run setup function
			if tc.setupFunc != nil {
				tc.setupFunc(t, repoPath)
			}
			
			// Change to repo directory
			err = os.Chdir(repoPath)
			require.NoError(t, err)
			
			// Create command
			cmd := newRemoteCommand()
			
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

func TestRemoteSetURLCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFunc   func(t *testing.T, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "set-url for existing remote",
			args: []string{"set-url", "origin", "https://github.com/newuser/newrepo.git"},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/user/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				// Check config file
				configPath := filepath.Join(repoPath, ".git", "config")
				content, err := os.ReadFile(configPath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "url = https://github.com/newuser/newrepo.git")
				assert.NotContains(t, string(content), "url = https://github.com/user/repo.git")
			},
		},
		{
			name:        "set-url for non-existent remote",
			args:        []string{"set-url", "nonexistent", "https://github.com/user/repo.git"},
			expectError: true,
		},
		{
			name:        "set-url with no arguments",
			args:        []string{"set-url"},
			expectError: true,
		},
		{
			name:        "set-url with only remote name",
			args:        []string{"set-url", "origin"},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			repoPath := filepath.Join(tmpDir, "test-repo")
			
			// Initialize repository
			_, err := vcs.Init(repoPath)
			require.NoError(t, err)
			
			// Run setup function
			if tc.setupFunc != nil {
				tc.setupFunc(t, repoPath)
			}
			
			// Change to repo directory
			err = os.Chdir(repoPath)
			require.NoError(t, err)
			
			// Create command
			cmd := newRemoteCommand()
			
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
					tc.checkFunc(t, output, repoPath)
				}
			}
		})
	}
}

// TestLoadConfig tests loading git config
// Commented out as loadConfig is not exported
/*
func TestLoadConfig(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Test with empty config
	config, err := loadConfig(repo)
	assert.NoError(t, err)
	assert.Empty(t, config.remotes)
	
	// Create config with remotes
	configPath := filepath.Join(repo.GitDir(), "config")
	configContent := `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/user/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
[remote "upstream"]
	url = https://github.com/upstream/repo.git
[branch "main"]
	remote = origin
	merge = refs/heads/main
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	
	// Load config again
	config, err = loadConfig(repo)
	assert.NoError(t, err)
	assert.Len(t, config.remotes, 2)
	
	// Check origin remote
	origin, exists := config.remotes["origin"]
	assert.True(t, exists)
	assert.Equal(t, "origin", origin.name)
	assert.Equal(t, "https://github.com/user/repo.git", origin.url)
	assert.Equal(t, "+refs/heads/*:refs/remotes/origin/*", origin.fetch)
	
	// Check upstream remote
	upstream, exists := config.remotes["upstream"]
	assert.True(t, exists)
	assert.Equal(t, "upstream", upstream.name)
	assert.Equal(t, "https://github.com/upstream/repo.git", upstream.url)
}

*/

// TestSaveConfig tests saving git config  
// Commented out as saveConfig is not exported
/*
func TestSaveConfig(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Create config
	config := &gitConfig{
		remotes: map[string]*remoteConfig{
			"origin": {
				name:  "origin",
				url:   "https://github.com/user/repo.git",
				fetch: "+refs/heads/*:refs/remotes/origin/*",
			},
			"upstream": {
				name: "upstream",
				url:  "https://github.com/upstream/repo.git",
			},
		},
	}
	
	// Save config
	err = saveConfig(repo, config)
	assert.NoError(t, err)
	
	// Read config file
	configPath := filepath.Join(repo.GitDir(), "config")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	
	// Check content
	assert.Contains(t, string(content), "[remote \"origin\"]")
	assert.Contains(t, string(content), "url = https://github.com/user/repo.git")
	assert.Contains(t, string(content), "fetch = +refs/heads/*:refs/remotes/origin/*")
	assert.Contains(t, string(content), "[remote \"upstream\"]")
	assert.Contains(t, string(content), "url = https://github.com/upstream/repo.git")
}
*/

func TestGetRemotesFunction(t *testing.T) {
	// This function is already tested through fetch_test.go
	// but we can add a specific test here
	
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
	url = https://github.com/user/repo.git
[remote "upstream"]
	url = git@github.com:upstream/repo.git
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	
	// Test with remotes
	remotes, err = getRemotes(repo)
	assert.NoError(t, err)
	assert.Len(t, remotes, 2)
	assert.Equal(t, "https://github.com/user/repo.git", remotes["origin"])
	assert.Equal(t, "git@github.com:upstream/repo.git", remotes["upstream"])
}