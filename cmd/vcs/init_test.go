package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInitCommand(t *testing.T) {
	cmd := newInitCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "init", cmd.Use)
	assert.Contains(t, cmd.Short, "Create an empty Git repository")
}

func TestInitCommandDetailed(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		setupFunc   func(t *testing.T, tmpDir string) string
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "init in current directory",
			args: []string{},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Initialized empty Git repository")
				assert.Contains(t, output, repoPath)
				assert.Contains(t, output, ".git")
				
				// Check .git directory structure
				gitDir := filepath.Join(repoPath, ".git")
				assert.DirExists(t, gitDir)
				assert.DirExists(t, filepath.Join(gitDir, "objects"))
				assert.DirExists(t, filepath.Join(gitDir, "refs"))
				assert.DirExists(t, filepath.Join(gitDir, "refs", "heads"))
				assert.DirExists(t, filepath.Join(gitDir, "refs", "tags"))
				
				// Check files
				assert.FileExists(t, filepath.Join(gitDir, "HEAD"))
				assert.FileExists(t, filepath.Join(gitDir, "config"))
				assert.FileExists(t, filepath.Join(gitDir, "description"))
				
				// Check HEAD content
				headContent, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
				require.NoError(t, err)
				assert.Equal(t, "ref: refs/heads/main\n", string(headContent))
			},
		},
		{
			name: "init in specific directory",
			args: []string{"myrepo"},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				expectedPath := filepath.Join(filepath.Dir(repoPath), "myrepo")
				assert.Contains(t, output, "Initialized empty Git repository")
				assert.Contains(t, output, expectedPath)
				
				// Check directory was created
				assert.DirExists(t, expectedPath)
				assert.DirExists(t, filepath.Join(expectedPath, ".git"))
			},
		},
		{
			name: "init bare repository",
			args: []string{},
			flags: map[string]string{
				"bare": "true",
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Initialized empty Git repository")
				assert.NotContains(t, output, "/.git")
				
				// In bare repo, git files are at top level
				assert.DirExists(t, filepath.Join(repoPath, "objects"))
				assert.DirExists(t, filepath.Join(repoPath, "refs"))
				assert.FileExists(t, filepath.Join(repoPath, "HEAD"))
				assert.FileExists(t, filepath.Join(repoPath, "config"))
				
				// No .git subdirectory
				assert.NoDirExists(t, filepath.Join(repoPath, ".git"))
			},
		},
		{
			name: "init bare repository in specific directory",
			args: []string{"bare-repo"},
			flags: map[string]string{
				"bare": "true",
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				expectedPath := filepath.Join(filepath.Dir(repoPath), "bare-repo")
				assert.Contains(t, output, "Initialized empty Git repository")
				assert.Contains(t, output, expectedPath)
				assert.NotContains(t, output, "/.git")
				
				// Check bare structure
				assert.DirExists(t, filepath.Join(expectedPath, "objects"))
				assert.DirExists(t, filepath.Join(expectedPath, "refs"))
			},
		},
		{
			name: "init with custom branch name",
			args: []string{},
			flags: map[string]string{
				"initial-branch": "develop",
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Initialized empty Git repository")
				
				// Check HEAD points to custom branch
				headPath := filepath.Join(repoPath, ".git", "HEAD")
				headContent, err := os.ReadFile(headPath)
				require.NoError(t, err)
				assert.Equal(t, "ref: refs/heads/develop\n", string(headContent))
			},
		},
		{
			name: "init with quiet flag",
			args: []string{},
			flags: map[string]string{
				"quiet": "true",
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				// With quiet flag, no output
				assert.Empty(t, output)
				
				// But repository should still be created
				assert.DirExists(t, filepath.Join(repoPath, ".git"))
			},
		},
		{
			name: "init in existing repository",
			args: []string{},
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Create a repository first
				gitDir := filepath.Join(tmpDir, ".git")
				err := os.MkdirAll(gitDir, 0755)
				require.NoError(t, err)
				
				// Write HEAD file
				headPath := filepath.Join(gitDir, "HEAD")
				err = os.WriteFile(headPath, []byte("ref: refs/heads/main\n"), 0644)
				require.NoError(t, err)
				
				return tmpDir
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Reinitialized existing Git repository")
				assert.DirExists(t, filepath.Join(repoPath, ".git"))
			},
		},
		{
			name: "init in nested existing repository",
			args: []string{},
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Create parent repository
				parentGitDir := filepath.Join(tmpDir, ".git")
				err := os.MkdirAll(parentGitDir, 0755)
				require.NoError(t, err)
				
				// Create subdirectory
				subDir := filepath.Join(tmpDir, "subdir")
				err = os.MkdirAll(subDir, 0755)
				require.NoError(t, err)
				
				return subDir
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Initialized empty Git repository")
				// Should create new repo in subdir
				assert.DirExists(t, filepath.Join(repoPath, ".git"))
			},
		},
		{
			name: "init with invalid directory",
			args: []string{"/nonexistent/path/to/repo"},
			expectError: true,
		},
		{
			name: "init with template directory",
			args: []string{},
			flags: map[string]string{
				"template": "/path/to/template",
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Initialized empty Git repository")
				// Template functionality not implemented, but flag should be accepted
			},
		},
		{
			name: "init with shared option",
			args: []string{},
			flags: map[string]string{
				"shared": "group",
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Initialized empty Git repository")
				// Shared functionality not implemented, but flag should be accepted
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
			var repoPath string
			if tc.setupFunc != nil {
				repoPath = tc.setupFunc(t, tmpDir)
				err = os.Chdir(repoPath)
				require.NoError(t, err)
			} else {
				repoPath = tmpDir
			}
			
			// Create command
			cmd := newInitCommand()
			
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
					tc.checkFunc(t, output, repoPath)
				}
			}
		})
	}
}

func TestInitPermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Running as root, skipping permission test")
	}
	
	// Create temporary directory
	tmpDir := t.TempDir()
	
	// Create read-only directory
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.MkdirAll(readOnlyDir, 0755)
	require.NoError(t, err)
	
	// Make it read-only
	err = os.Chmod(readOnlyDir, 0555)
	require.NoError(t, err)
	defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup
	
	// Change to read-only directory
	err = os.Chdir(readOnlyDir)
	require.NoError(t, err)
	
	// Create command
	cmd := newInitCommand()
	
	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	// Execute command
	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestInitGitConfig(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create command
	cmd := newInitCommand()
	
	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	// Execute command
	err = cmd.Execute()
	assert.NoError(t, err)
	
	// Check config file content
	configPath := filepath.Join(tmpDir, ".git", "config")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	
	// Check basic config sections
	assert.Contains(t, string(content), "[core]")
	assert.Contains(t, string(content), "repositoryformatversion = 0")
	assert.Contains(t, string(content), "filemode = true")
	assert.Contains(t, string(content), "bare = false")
}

func TestInitHooksDirectory(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create command
	cmd := newInitCommand()
	
	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	// Execute command
	err = cmd.Execute()
	assert.NoError(t, err)
	
	// Check hooks directory
	hooksDir := filepath.Join(tmpDir, ".git", "hooks")
	assert.DirExists(t, hooksDir)
	
	// In a full implementation, sample hooks would be created
	// For now, just check the directory exists
}

func TestInitInfoDirectory(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create command
	cmd := newInitCommand()
	
	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	// Execute command
	err = cmd.Execute()
	assert.NoError(t, err)
	
	// Check info directory
	infoDir := filepath.Join(tmpDir, ".git", "info")
	assert.DirExists(t, infoDir)
	
	// Check exclude file
	excludePath := filepath.Join(infoDir, "exclude")
	assert.FileExists(t, excludePath)
	
	// Check exclude content
	content, err := os.ReadFile(excludePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# git ls-files --others --exclude-from=.git/info/exclude")
}

func TestInitMultipleTimes(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create command
	cmd := newInitCommand()
	
	// First init
	var buf1 bytes.Buffer
	cmd.SetOut(&buf1)
	err = cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf1.String(), "Initialized empty Git repository")
	
	// Second init (reinit)
	cmd = newInitCommand() // Create new command instance
	var buf2 bytes.Buffer
	cmd.SetOut(&buf2)
	err = cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf2.String(), "Reinitialized existing Git repository")
	
	// Repository should still be valid
	assert.DirExists(t, filepath.Join(tmpDir, ".git"))
}