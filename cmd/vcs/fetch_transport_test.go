package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestFetchCommand_HTTPTransport(t *testing.T) {
	// Create a mock HTTP server that responds like a Git server
	mockRefData := `# service=git-upload-pack
0000004895dc4b2c3e0ef0a5b7b2e4b3e1f2e3e4e5e6e7e8e9 HEAD
003f95dc4b2c3e0f0a5b7b2e4b3e1f2e3e4e5e6e7e8e9 refs/heads/main
0000`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/info/refs" && r.URL.Query().Get("service") == "git-upload-pack" {
			w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockRefData))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create a temporary repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Change to repository directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Add remote pointing to test server
	configPath := filepath.Join(repo.GitDir(), "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[remote "origin"]
	url = ` + server.URL + `
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	err = writeFile(configPath, []byte(configContent))
	require.NoError(t, err)

	// Test fetch command with HTTP transport
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"-v", "origin"})

	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Using HTTP transport")
	assert.Contains(t, output, "remote: Found")
	assert.Contains(t, output, "refs")
	assert.Contains(t, output, "HTTP transport fetch completed successfully")
}

func TestFetchCommand_HTTPTransportFallback(t *testing.T) {
	// Create a server that returns errors to test fallback
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
	}))
	defer server.Close()

	// Create a temporary repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Change to repository directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Add remote pointing to test server
	configPath := filepath.Join(repo.GitDir(), "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[remote "origin"]
	url = ` + server.URL + `
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	err = writeFile(configPath, []byte(configContent))
	require.NoError(t, err)

	// Test fetch command with fallback
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"-v", "origin"})

	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "HTTP transport failed")
	assert.Contains(t, output, "Falling back to basic implementation")
	assert.Contains(t, output, "This is a basic fetch implementation")
}

func TestFetchCommand_GitHubURL(t *testing.T) {
	// Create a temporary repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Change to repository directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Add GitHub remote (will fail to connect but tests URL parsing)
	configPath := filepath.Join(repo.GitDir(), "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[remote "origin"]
	url = git@github.com:user/repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	err = writeFile(configPath, []byte(configContent))
	require.NoError(t, err)

	// Test fetch command with GitHub URL
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"-v", "origin"})

	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	// Should attempt HTTP transport but fall back to basic implementation
	assert.Contains(t, output, "Falling back to basic implementation")
}

func TestIsHTTPURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"HTTPS URL", "https://github.com/user/repo.git", true},
		{"HTTP URL", "http://example.com/repo.git", true},
		{"GitHub SSH", "git@github.com:user/repo.git", true},
		{"Contains github.com", "https://github.com/user/repo", true},
		{"SSH format", "user@host.com:repo.git", true},
		{"Local path", "/path/to/repo", false},
		{"File URL", "file:///path/to/repo", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHTTPURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFetchCommand_NonHTTPURL(t *testing.T) {
	// Create a temporary repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Change to repository directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Add local file remote (non-HTTP)
	configPath := filepath.Join(repo.GitDir(), "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[remote "origin"]
	url = /path/to/local/repo
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	err = writeFile(configPath, []byte(configContent))
	require.NoError(t, err)

	// Test fetch command with non-HTTP URL
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"-v", "origin"})

	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	// Should skip HTTP transport and go directly to basic implementation
	assert.NotContains(t, output, "Using HTTP transport")
	assert.Contains(t, output, "This is a basic fetch implementation")
}

func TestFetchWithHTTPTransport_ParseError(t *testing.T) {
	// Create a temporary repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Test with invalid URL that should cause parse error
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Call fetchWithHTTPTransport directly with invalid URL
	err = fetchWithHTTPTransport(cmd, repo, "origin", "ht!tp://invalid-url", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse remote URL")
}

func TestFetchBasicImplementation(t *testing.T) {
	// Create a temporary repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Test basic implementation directly
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = fetchBasicImplementation(cmd, repo, "origin", "https://example.com/repo.git", true)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "remote: Enumerating objects")
	assert.Contains(t, output, "From https://example.com/repo.git")
	assert.Contains(t, output, "This is a basic fetch implementation")

	// Check that FETCH_HEAD was created
	fetchHeadPath := filepath.Join(repo.GitDir(), "FETCH_HEAD")
	assert.FileExists(t, fetchHeadPath)

	fetchHeadContent, err := readFile(fetchHeadPath)
	require.NoError(t, err)
	assert.Contains(t, string(fetchHeadContent), "origin")
}

func TestFetchCommand_AllFlag(t *testing.T) {
	// Create a temporary repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Change to repository directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Test fetch command with --all flag
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--all"})

	// This should error since there are no remotes configured
	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remote 'origin' does not exist")
}

func TestFetchCommand_DepthFlag(t *testing.T) {
	// Create a temporary repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Change to repository directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Add a remote
	configPath := filepath.Join(repo.GitDir(), "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[remote "origin"]
	url = /path/to/repo
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	err = writeFile(configPath, []byte(configContent))
	require.NoError(t, err)

	// Test fetch command with --depth flag
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--depth", "1", "origin"})

	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "From /path/to/repo")
}

func TestFetchCommand_TagsFlag(t *testing.T) {
	// Create a temporary repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Change to repository directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Add a remote
	configPath := filepath.Join(repo.GitDir(), "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[remote "origin"]
	url = /path/to/repo
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	err = writeFile(configPath, []byte(configContent))
	require.NoError(t, err)

	// Test fetch command with --tags flag
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--tags", "origin"})

	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "From /path/to/repo")
}

func TestFetchCommand_PruneFlag(t *testing.T) {
	// Create a temporary repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Change to repository directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Add a remote
	configPath := filepath.Join(repo.GitDir(), "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[remote "origin"]
	url = /path/to/repo
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	err = writeFile(configPath, []byte(configContent))
	require.NoError(t, err)

	// Test fetch command with --prune flag
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--prune", "origin"})

	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "From /path/to/repo")
}

func TestFetchCommand_NoRemotes(t *testing.T) {
	// Create a temporary repository with no remotes
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Change to repository directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Test fetch command with no remotes configured
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"origin"})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remote 'origin' does not exist")
}

func TestFetchCommand_CustomRemote(t *testing.T) {
	// Create a temporary repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Change to repository directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Add a custom remote
	configPath := filepath.Join(repo.GitDir(), "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[remote "upstream"]
	url = /path/to/upstream/repo
	fetch = +refs/heads/*:refs/remotes/upstream/*
`
	err = writeFile(configPath, []byte(configContent))
	require.NoError(t, err)

	// Test fetch command with custom remote
	cmd := newFetchCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"upstream"})

	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Fetching from upstream")
	assert.Contains(t, output, "From /path/to/upstream/repo")
}