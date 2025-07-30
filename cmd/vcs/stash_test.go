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

func TestNewStashCommand(t *testing.T) {
	cmd := newStashCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "stash", cmd.Use)
	assert.Contains(t, cmd.Short, "Stash the changes")
}

func TestStashSubcommands(t *testing.T) {
	cmd := newStashCommand()
	
	// Check all subcommands are registered
	subcommands := []string{"push", "list", "show", "pop", "apply", "drop", "clear"}
	for _, subcmd := range subcommands {
		found := false
		for _, c := range cmd.Commands() {
			if c.Name() == subcmd {
				found = true
				break
			}
		}
		assert.True(t, found, "Subcommand %s not found", subcmd)
	}
}

func TestStashSave(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFunc   func(t *testing.T, repo *vcs.Repository, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "stash with changes",
			args: []string{},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create a file and stage it
				testFile := filepath.Join(repoPath, "modified.txt")
				err := os.WriteFile(testFile, []byte("modified content"), 0644)
				require.NoError(t, err)
				testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("modified.txt")
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "Saved working directory and index state")
				assert.Contains(t, output, "WIP on main:")
				
				// Check stash directory was created
				stashDir := filepath.Join(repoPath, ".git", "stash")
				assert.DirExists(t, stashDir)
				
				// Check stash list file exists
				stashFile := filepath.Join(stashDir, "stash_list")
				assert.FileExists(t, stashFile)
			},
		},
		{
			name:      "stash without changes",
			args:      []string{},
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				assert.Contains(t, output, "No local changes to save")
			},
		},
		{
			name:        "stash outside repository",
			args:        []string{},
			setupFunc:   func(t *testing.T, repo *vcs.Repository, repoPath string) {},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			
			var repoPath string
			var repo *vcs.Repository
			
			if tc.name != "stash outside repository" {
				// Initialize repository
				repoPath = filepath.Join(tmpDir, "test-repo")
				var err error
				repo, err = vcs.Init(repoPath)
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
					tc.setupFunc(t, repo, repoPath)
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
			cmd := newStashCommand()
			
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

func TestStashPush(t *testing.T) {
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
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create changes
	modifiedFile := filepath.Join(repoPath, "modified.txt")
	err = os.WriteFile(modifiedFile, []byte("modified content"), 0644)
	require.NoError(t, err)
				err = testRepo.Add("modified.txt")
	require.NoError(t, err)
	
	// Create stash push command
	cmd := newStashPushCommand()
	
	// Test with message flag
	err = cmd.Flags().Set("message", "My stash message")
	require.NoError(t, err)
	
	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	// Execute command
	err = cmd.Execute()
	assert.NoError(t, err)
	
	// Check output
	output := buf.String()
	assert.Contains(t, output, "Saved working directory")
}

func TestStashList(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T, repo *vcs.Repository, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string)
	}{
		{
			name: "list with stashes",
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {
				// Create stash directory and list
				stashDir := filepath.Join(repoPath, ".git", "stash")
				err := os.MkdirAll(stashDir, 0755)
				require.NoError(t, err)
				
				stashFile := filepath.Join(stashDir, "stash_list")
				stashContent := `2024-01-01T10:00:00Z main WIP on main: abc1234
2024-01-01T11:00:00Z feature WIP on feature: def5678
2024-01-01T12:00:00Z main Stash before merge
`
				err = os.WriteFile(stashFile, []byte(stashContent), 0644)
				require.NoError(t, err)
			},
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "stash@{0}: Stash before merge")
				assert.Contains(t, output, "stash@{1}: WIP on feature: def5678")
				assert.Contains(t, output, "stash@{2}: WIP on main: abc1234")
			},
		},
		{
			name:      "list with no stashes",
			setupFunc: func(t *testing.T, repo *vcs.Repository, repoPath string) {},
			checkFunc: func(t *testing.T, output string) {
				assert.Empty(t, strings.TrimSpace(output))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
			
			// Run setup function
			if tc.setupFunc != nil {
				tc.setupFunc(t, repo, repoPath)
			}
			
			// Change to repo directory
			err = os.Chdir(repoPath)
			require.NoError(t, err)
			
			// Create command
			cmd := newStashCommand()
			
			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			
			// Execute list subcommand
			cmd.SetArgs([]string{"list"})
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

func TestStashShow(t *testing.T) {
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
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create command
	cmd := newStashCommand()
	
	// Test default (stash@{0})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"show"})
	err = cmd.Execute()
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Showing stash stash@{0}")
	assert.Contains(t, output, "Full stash show would display the diff")
	
	// Test specific stash
	buf.Reset()
	cmd.SetArgs([]string{"show", "stash@{2}"})
	err = cmd.Execute()
	assert.NoError(t, err)
	output = buf.String()
	assert.Contains(t, output, "Showing stash stash@{2}")
}

func TestStashPop(t *testing.T) {
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
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create command
	cmd := newStashCommand()
	
	// Test pop
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"pop"})
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Applying stash stash@{0}")
	assert.Contains(t, output, "Dropped stash@{0}")
	
	// Test pop specific stash
	buf.Reset()
	cmd.SetArgs([]string{"pop", "stash@{1}"})
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output = buf.String()
	assert.Contains(t, output, "Applying stash stash@{1}")
	assert.Contains(t, output, "Dropped stash@{1}")
}

func TestStashApply(t *testing.T) {
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
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create command
	cmd := newStashCommand()
	
	// Test apply
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"apply"})
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Applying stash stash@{0}")
	assert.Contains(t, output, "Note: Full stash apply would:")
	assert.NotContains(t, output, "Dropped") // Apply doesn't drop
}

func TestStashDrop(t *testing.T) {
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
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create command
	cmd := newStashCommand()
	
	// Test drop
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"drop"})
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Dropped stash@{0}")
	assert.Contains(t, output, "Note: Full stash drop would remove")
}

func TestStashClear(t *testing.T) {
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
	
	// Create stash file
	stashDir := filepath.Join(repoPath, ".git", "stash")
	err = os.MkdirAll(stashDir, 0755)
	require.NoError(t, err)
	
	stashFile := filepath.Join(stashDir, "stash_list")
	err = os.WriteFile(stashFile, []byte("some stash entries"), 0644)
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create command
	cmd := newStashCommand()
	
	// Test clear
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"clear"})
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := buf.String()
	assert.Contains(t, output, "Cleared all stashes")
	
	// Verify file was removed
	assert.NoFileExists(t, stashFile)
}

func TestHasLocalChanges(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")
	
	// Initialize repository
	repo, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Test with no changes
	hasChanges, err := hasLocalChanges(repo)
	assert.NoError(t, err)
	assert.False(t, hasChanges)
	
	// Make initial commit
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	
	testRepo := WrapRepository(repo, repoPath)
				err = testRepo.Add("test.txt")
	require.NoError(t, err)
	
	// Test with staged changes (before commit)
	hasChanges, err = hasLocalChanges(repo)
	assert.NoError(t, err)
	assert.True(t, hasChanges)
	
	// Commit the changes
	_, err = testRepo.Commit("Initial commit", "Test User", "test@example.com")
	require.NoError(t, err)
	
	// Test after commit (no changes)
	hasChanges, err = hasLocalChanges(repo)
	assert.NoError(t, err)
	assert.False(t, hasChanges)
	
	// Add new file and stage it
	newFile := filepath.Join(repoPath, "new.txt")
	err = os.WriteFile(newFile, []byte("new content"), 0644)
	require.NoError(t, err)
	
				err = testRepo.Add("new.txt")
	require.NoError(t, err)
	
	// Test with new staged changes
	hasChanges, err = hasLocalChanges(repo)
	assert.NoError(t, err)
	assert.True(t, hasChanges)
}

func TestStashImplementationNotes(t *testing.T) {
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
	
	// Create changes to stash
	modifiedFile := filepath.Join(repoPath, "modified.txt")
	err = os.WriteFile(modifiedFile, []byte("modified content"), 0644)
	require.NoError(t, err)
				err = testRepo.Add("modified.txt")
	require.NoError(t, err)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Run stash save
	cmd := newStashCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	err = cmd.Execute()
	assert.NoError(t, err)
	
	// Check implementation notes
	output := buf.String()
	assert.Contains(t, output, "Note: This is a basic stash implementation")
	assert.Contains(t, output, "Full implementation would:")
	assert.Contains(t, output, "Create tree objects for working directory and index")
	assert.Contains(t, output, "Create stash commits")
	assert.Contains(t, output, "Reset working directory")
	assert.Contains(t, output, "Maintain stash reflog")
}