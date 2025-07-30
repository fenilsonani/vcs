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

func TestNewPullCommand(t *testing.T) {
	cmd := newPullCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "pull", cmd.Use)
	assert.Contains(t, cmd.Short, "Fetch from and integrate")
}

func TestPullCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		setupFunc   func(t *testing.T, repoPath string)
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "pull from origin",
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
				assert.Contains(t, output, "Pulling from https://github.com/example/repo.git")
				assert.Contains(t, output, "Fetching from remote")
				assert.Contains(t, output, "Merging")
				assert.Contains(t, output, "Already up to date")
				
				// Check FETCH_HEAD was created
				fetchHeadPath := filepath.Join(repoPath, ".git", "FETCH_HEAD")
				assert.FileExists(t, fetchHeadPath)
			},
		},
		{
			name: "pull from specific remote and branch",
			args: []string{"upstream", "develop"},
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
				assert.Contains(t, output, "Pulling from https://github.com/upstream/repo.git")
				assert.Contains(t, output, "Fetching from remote")
			},
		},
		{
			name: "pull with rebase",
			args: []string{},
			flags: map[string]string{
				"rebase": "true",
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
				assert.Contains(t, output, "Rebasing")
				assert.Contains(t, output, "Successfully rebased")
			},
		},
		{
			name: "pull with no-commit",
			args: []string{},
			flags: map[string]string{
				"no-commit": "true",
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
				assert.Contains(t, output, "Merging")
				// With no-commit, merge commit message shouldn't appear
			},
		},
		{
			name: "pull with verbose",
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
				assert.Contains(t, output, "From https://github.com/example/repo.git")
				assert.Contains(t, output, "branch")
				assert.Contains(t, output, "FETCH_HEAD")
			},
		},
		{
			name: "pull with custom strategy",
			args: []string{},
			flags: map[string]string{
				"strategy": "ours",
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
				assert.Contains(t, output, "Merge made by the 'recursive' strategy")
			},
		},
		{
			name:        "pull from non-existent remote",
			args:        []string{"nonexistent"},
			setupFunc:   func(t *testing.T, repoPath string) {},
			expectError: true,
		},
		{
			name:        "pull outside repository",
			args:        []string{},
			setupFunc:   func(t *testing.T, repoPath string) {},
			expectError: true,
		},
		{
			name: "pull on detached HEAD",
			args: []string{},
			setupFunc: func(t *testing.T, repoPath string) {
				// Add origin remote
				configPath := filepath.Join(repoPath, ".git", "config")
				configContent := `[remote "origin"]
	url = https://github.com/example/repo.git
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
				
				// Checkout commit directly (detached HEAD)
				repo, err := vcs.Open(repoPath)
				require.NoError(t, err)
				testRepo := WrapRepository(repo, repoPath)
				commits, err := testRepo.Log(1)
				require.NoError(t, err)
				require.Len(t, commits, 1)
				err = testRepo.Checkout(commits[0].ID().String())
				require.NoError(t, err)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			
			var repoPath string
			if tc.name != "pull outside repository" {
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
			cmd := newPullCommand()
			
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

func TestPullFromRemote(t *testing.T) {
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
	cmd := newPullCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	// Test pull without rebase
	err = pullFromRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
		"main", "main", false, false, false, false, "recursive")
	assert.NoError(t, err)
	
	// Check output
	output := buf.String()
	assert.Contains(t, output, "Fetching from remote")
	assert.Contains(t, output, "Merging")
	assert.Contains(t, output, "Already up to date")
	
	// Check FETCH_HEAD
	fetchHeadPath := filepath.Join(repo.GitDir(), "FETCH_HEAD")
	assert.FileExists(t, fetchHeadPath)
	
	// Test pull with rebase
	buf.Reset()
	err = pullFromRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
		"main", "main", true, false, false, false, "recursive")
	assert.NoError(t, err)
	
	output = buf.String()
	assert.Contains(t, output, "Rebasing")
	assert.Contains(t, output, "Successfully rebased")
	
	// Test verbose pull
	buf.Reset()
	err = pullFromRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
		"main", "main", false, false, false, true, "recursive")
	assert.NoError(t, err)
	
	output = buf.String()
	assert.Contains(t, output, "From https://github.com/example/repo.git")
	assert.Contains(t, output, "branch")
	assert.Contains(t, output, "FETCH_HEAD")
}

func TestPullMergeScenarios(t *testing.T) {
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
	cmd := newPullCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	// Test fast-forward merge (simulated)
	err = pullFromRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
		"main", "main", false, false, false, false, "recursive")
	assert.NoError(t, err)
	
	output := buf.String()
	// In our simulation, fast-forward is false, so we see regular merge
	assert.Contains(t, output, "Merge made by the 'recursive' strategy")
	
	// Test merge with no-commit
	buf.Reset()
	err = pullFromRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
		"main", "main", false, true, false, false, "recursive")
	assert.NoError(t, err)
	
	output = buf.String()
	assert.Contains(t, output, "Merging")
	// With no-commit, the merge commit message should still appear in simulation
	assert.Contains(t, output, "Merge made by the 'recursive' strategy")
}

func TestPullImplementationNotes(t *testing.T) {
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
	cmd := newPullCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	
	// Run pull to see implementation notes
	err = pullFromRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
		"main", "main", false, false, false, false, "recursive")
	assert.NoError(t, err)
	
	// Check implementation notes appear
	output := buf.String()
	assert.Contains(t, output, "Note: This is a basic pull implementation")
	assert.Contains(t, output, "Full implementation would require:")
	assert.Contains(t, output, "Actual network fetch operation")
	assert.Contains(t, output, "Three-way merge algorithm")
	assert.Contains(t, output, "Conflict resolution")
	assert.Contains(t, output, "Working tree updates")
}

func TestPullEdgeCases(t *testing.T) {
	t.Run("pull with different local and remote branches", func(t *testing.T) {
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
		
		// Create feature branch
		_, err = testRepo.CreateBranch("feature")
		require.NoError(t, err)
		err = testRepo.Checkout("feature")
		require.NoError(t, err)
		
		// Create command for output
		cmd := newPullCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		
		// Pull from different remote branch
		err = pullFromRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
			"feature", "develop", false, false, false, true, "recursive")
		assert.NoError(t, err)
		
		// Check output shows correct branches
		output := buf.String()
		assert.Contains(t, output, "develop")
		assert.Contains(t, output, "FETCH_HEAD")
	})
	
	t.Run("pull with squash option", func(t *testing.T) {
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
		cmd := newPullCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		
		// Pull with squash
		err = pullFromRemote(cmd, repo, "origin", "https://github.com/example/repo.git", 
			"main", "main", false, false, true, false, "recursive")
		assert.NoError(t, err)
		
		// Squash option is passed but not used in basic implementation
		output := buf.String()
		assert.Contains(t, output, "Merge made by the 'recursive' strategy")
	})
}