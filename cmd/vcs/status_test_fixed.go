package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewStatusCommand_Fixed(t *testing.T) {
	cmd := newStatusCommand()
	
	if cmd.Use != "status" {
		t.Errorf("Expected Use to be 'status', got %s", cmd.Use)
	}
	
	if cmd.Short != "Show working tree status" {
		t.Errorf("Expected Short description, got %s", cmd.Short)
	}
	
	// Check flags exist
	if cmd.Flags().Lookup("short") == nil {
		t.Error("Expected --short flag to exist")
	}
}

func TestRunStatus_Fixed(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()
	
	// Initialize repository
	helper.ChDir()
	repo, err := vcs.Init(helper.TmpDir())
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	tests := []struct {
		name         string
		setup        func()
		args         []string
		flags        map[string]string
		wantContains []string
		wantErr      bool
	}{
		{
			name: "empty repository",
			setup: func() {
				// No setup needed
			},
			args:         []string{},
			wantContains: []string{"nothing to commit, working tree clean"},
			wantErr:      false,
		},
		{
			name: "untracked files",
			setup: func() {
				helper.CreateFile("untracked.txt", "content")
			},
			args:         []string{},
			wantContains: []string{"Untracked files:", "untracked.txt"},
			wantErr:      false,
		},
		{
			name: "staged files",
			setup: func() {
				// Create file and add to index
				content := []byte("staged content")
				helper.CreateFile("staged.txt", string(content))
				
				// Add to index manually
				idx := index.New()
				blob := repo.CreateBlobDirect(content)
				entry := &index.Entry{
					Mode: 0644,
					Size: uint32(len(content)),
					ID:   blob.ID(),
					Path: "staged.txt",
				}
				idx.Add(entry)
				idx.WriteToFile(filepath.Join(repo.GitDir(), "index"))
			},
			args:         []string{},
			wantContains: []string{"Changes to be committed:", "new file:   staged.txt"},
			wantErr:      false,
		},
		{
			name: "short format",
			setup: func() {
				helper.CreateFile("short.txt", "content")
			},
			args:         []string{},
			flags:        map[string]string{"short": "true"},
			wantContains: []string{"?? short.txt"},
			wantErr:      false,
		},
		{
			name: "mixed status",
			setup: func() {
				// Create files in different states
				helper.CreateFile("untracked.txt", "untracked content")
				
				// Add a file to index
				helper.CreateFile("staged.txt", "staged content")
				addCmd := newAddCommand()
				addResult := helper.RunCommand(addCmd, []string{"staged.txt"}, nil)
				addResult.AssertError(t, false)
				
				// Modify the staged file
				helper.CreateFile("staged.txt", "modified staged content")
			},
			args: []string{},
			wantContains: []string{
				"Changes to be committed:",
				"Changes not staged for commit:",
				"Untracked files:",
			},
			wantErr: false,
		},
		{
			name: "short format with mixed status",
			setup: func() {
				helper.CreateFile("untracked.txt", "untracked")
				helper.CreateFile("modified.txt", "original")
				
				// Add file
				addCmd := newAddCommand()
				addResult := helper.RunCommand(addCmd, []string{"modified.txt"}, nil)
				addResult.AssertError(t, false)
				
				// Modify it
				helper.CreateFile("modified.txt", "modified content")
			},
			args:  []string{},
			flags: map[string]string{"short": "true"},
			wantContains: []string{
				"AM modified.txt", // Added and Modified
				"?? untracked.txt",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any previous files (but keep .git)
			files, _ := filepath.Glob(filepath.Join(helper.TmpDir(), "*"))
			for _, file := range files {
				if filepath.Base(file) != ".git" {
					os.RemoveAll(file)
				}
			}
			
			// Remove index if exists (fresh start)
			indexPath := filepath.Join(repo.GitDir(), "index")
			os.Remove(indexPath)
			
			// Setup
			if tt.setup != nil {
				tt.setup()
			}

			// Create and run command
			cmd := newStatusCommand()
			result := helper.RunCommand(cmd, tt.args, tt.flags)
			
			// Check error expectation
			result.AssertError(t, tt.wantErr)

			// Check output contains expected strings
			if len(tt.wantContains) > 0 {
				result.AssertContains(t, tt.wantContains...)
			}
		})
	}
}

func TestFindRepository_Fixed(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()
	
	// Test non-repository
	helper.ChDir()

	_, err := findRepository()
	if err == nil {
		t.Error("Expected error for non-repository")
	}

	// Initialize repository
	_, err = vcs.Init(helper.TmpDir())
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	// Test repository found
	repoPath, err := findRepository()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if repoPath != helper.TmpDir() {
		t.Errorf("Expected repo path %s, got %s", helper.TmpDir(), repoPath)
	}
}

func TestFileStatusInfo_Fixed(t *testing.T) {
	tests := []struct {
		status     FileStatus
		wantIndex  string
		wantWork   string
	}{
		{StatusUnmodified, " ", " "},
		{StatusStaged, "A", " "},
		{StatusModified, "M", "M"},
		{StatusUntracked, " ", "?"},
		{StatusDeleted, "D", "D"},
		{StatusIgnored, " ", "!"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", i), func(t *testing.T) {
			if got := tt.status.IndexChar(); got != tt.wantIndex {
				t.Errorf("IndexChar() = %v, want %v", got, tt.wantIndex)
			}
			if got := tt.status.WorkChar(); got != tt.wantWork {
				t.Errorf("WorkChar() = %v, want %v", got, tt.wantWork)
			}
		})
	}
}

func TestPrintShortStatus_Fixed(t *testing.T) {
	statusMap := map[string]*FileStatusInfo{
		"modified.txt": {
			Path:        "modified.txt",
			IndexStatus: StatusUnmodified,
			WorkStatus:  StatusModified,
		},
		"staged.txt": {
			Path:        "staged.txt",
			IndexStatus: StatusStaged,
			WorkStatus:  StatusUnmodified,
		},
		"untracked.txt": {
			Path:        "untracked.txt",
			IndexStatus: StatusUnmodified,
			WorkStatus:  StatusUntracked,
		},
	}

	// Test the status characters through the FileStatus methods
	expectedStatuses := []struct {
		file      string
		indexChar string
		workChar  string
	}{
		{"modified.txt", " ", "M"},
		{"staged.txt", "A", " "},
		{"untracked.txt", " ", "?"},
	}

	for _, expected := range expectedStatuses {
		info := statusMap[expected.file]
		if info.IndexStatus.IndexChar() != expected.indexChar {
			t.Errorf("File %s index char = %q, want %q", expected.file, info.IndexStatus.IndexChar(), expected.indexChar)
		}
		if info.WorkStatus.WorkChar() != expected.workChar {
			t.Errorf("File %s work char = %q, want %q", expected.file, info.WorkStatus.WorkChar(), expected.workChar)
		}
	}
}