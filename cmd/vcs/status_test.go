package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewStatusCommand(t *testing.T) {
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

func TestRunStatus(t *testing.T) {
	// Create temp directory for test repo
	tmpDir, err := os.MkdirTemp("", "status-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repository
	repo, err := vcs.Init(tmpDir)
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	// Change to repo directory
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	tests := []struct {
		name         string
		setup        func() error
		args         []string
		flags        map[string]string
		wantContains []string
		wantErr      bool
	}{
		{
			name: "empty repository",
			setup: func() error {
				return nil
			},
			args:         []string{},
			wantContains: []string{"nothing to commit, working tree clean"},
			wantErr:      false,
		},
		{
			name: "untracked files",
			setup: func() error {
				return os.WriteFile("untracked.txt", []byte("content"), 0644)
			},
			args:         []string{},
			wantContains: []string{"Untracked files:", "untracked.txt"},
			wantErr:      false,
		},
		{
			name: "staged files",
			setup: func() error {
				// Create file and add to index
				content := []byte("staged content")
				if err := os.WriteFile("staged.txt", content, 0644); err != nil {
					return err
				}
				
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
				return idx.WriteToFile(filepath.Join(repo.GitDir(), "index"))
			},
			args:         []string{},
			wantContains: []string{"Changes to be committed:", "new file:   staged.txt"},
			wantErr:      false,
		},
		{
			name: "short format",
			setup: func() error {
				return os.WriteFile("short.txt", []byte("content"), 0644)
			},
			args:         []string{},
			flags:        map[string]string{"short": "true"},
			wantContains: []string{"?? short.txt"},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any previous files
			files, _ := filepath.Glob(filepath.Join(tmpDir, "*.txt"))
			for _, file := range files {
				os.Remove(file)
			}
			
			// Remove index if exists
			os.Remove(filepath.Join(repo.GitDir(), "index"))
			
			// Setup
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Create command
			cmd := newStatusCommand()
			
			// Set flags
			for key, value := range tt.flags {
				cmd.Flags().Set(key, value)
			}

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Run command
			err := cmd.RunE(cmd, tt.args)
			
			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("RunE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check output contains expected strings
			output := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing expected string %q\nGot: %s", want, output)
				}
			}
		})
	}
}

func TestFindRepository(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "find-repo-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test non-repository
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	_, err = findRepository()
	if err == nil {
		t.Error("Expected error for non-repository")
	}

	// Initialize repository
	_, err = vcs.Init(tmpDir)
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	// Test repository found
	repoPath, err := findRepository()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if repoPath != tmpDir {
		t.Errorf("Expected repo path %s, got %s", tmpDir, repoPath)
	}
}

func TestFileStatusInfo(t *testing.T) {
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

func TestPrintShortStatus(t *testing.T) {
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

	sortedFiles := []string{"modified.txt", "staged.txt", "untracked.txt"}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printShortStatus(sortedFiles, statusMap)

	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	expectedLines := []string{
		" M modified.txt",
		"A  staged.txt", 
		"?? untracked.txt",
	}

	for _, expected := range expectedLines {
		if !strings.Contains(output, expected) {
			t.Errorf("Output missing expected line %q\nGot: %s", expected, output)
		}
	}
}