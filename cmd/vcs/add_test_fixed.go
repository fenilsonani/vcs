package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewAddCommand_Fixed(t *testing.T) {
	cmd := newAddCommand()
	
	if cmd.Use != "add [flags] [pathspec...]" {
		t.Errorf("Expected Use to be 'add [flags] [pathspec...]', got %s", cmd.Use)
	}
	
	if cmd.Short != "Add file contents to the index" {
		t.Errorf("Expected Short description, got %s", cmd.Short)
	}
	
	// Check flags exist
	flags := []string{"all", "force", "dry-run", "verbose"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected --%s flag to exist", flag)
		}
	}
}

func TestRunAdd_Fixed(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()
	
	// Initialize repository
	helper.ChDir()
	repo, err := vcs.Init(helper.TmpDir())
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	tests := []struct {
		name     string
		setup    func()
		args     []string
		flags    map[string]string
		wantErr  bool
		checkIdx func(*index.Index) error
		wantOut  []string // Expected output strings
	}{
		{
			name: "add single file",
			setup: func() {
				helper.CreateFile("test.txt", "test content")
			},
			args:    []string{"test.txt"},
			wantErr: false,
			checkIdx: func(idx *index.Index) error {
				if _, exists := idx.Get("test.txt"); !exists {
					t.Error("file not found in index")
				}
				return nil
			},
			wantOut: []string{}, // Add command typically has no output on success
		},
		{
			name: "add multiple files",
			setup: func() {
				helper.CreateFile("file1.txt", "content1")
				helper.CreateFile("file2.txt", "content2")
			},
			args:    []string{"file1.txt", "file2.txt"},
			wantErr: false,
			checkIdx: func(idx *index.Index) error {
				if _, exists := idx.Get("file1.txt"); !exists {
					t.Error("file1.txt not found in index")
				}
				if _, exists := idx.Get("file2.txt"); !exists {
					t.Error("file2.txt not found in index")
				}
				return nil
			},
		},
		{
			name: "add all files",
			setup: func() {
				helper.CreateFile("all1.txt", "content1")
				helper.CreateFile("all2.txt", "content2")
			},
			args:    []string{},
			flags:   map[string]string{"all": "true"},
			wantErr: false,
			checkIdx: func(idx *index.Index) error {
				if _, exists := idx.Get("all1.txt"); !exists {
					t.Error("all1.txt not found in index")
				}
				if _, exists := idx.Get("all2.txt"); !exists {
					t.Error("all2.txt not found in index")
				}
				return nil
			},
		},
		{
			name: "dry run",
			setup: func() {
				helper.CreateFile("dryrun.txt", "content")
			},
			args:    []string{"dryrun.txt"},
			flags:   map[string]string{"dry-run": "true"},
			wantErr: false,
			checkIdx: func(idx *index.Index) error {
				if _, exists := idx.Get("dryrun.txt"); exists {
					t.Error("file should not be in index during dry run")
				}
				return nil
			},
			wantOut: []string{"add 'dryrun.txt'"}, // Dry run should show what would be added
		},
		{
			name: "verbose mode",
			setup: func() {
				helper.CreateFile("verbose.txt", "content")
			},
			args:    []string{"verbose.txt"},
			flags:   map[string]string{"verbose": "true"},
			wantErr: false,
			checkIdx: func(idx *index.Index) error {
				if _, exists := idx.Get("verbose.txt"); !exists {
					t.Error("verbose.txt not found in index")
				}
				return nil
			},
			wantOut: []string{"add 'verbose.txt'"}, // Verbose should show what was added
		},
		{
			name:    "no files specified without --all",
			setup:   func() {},
			args:    []string{},
			wantErr: true,
		},
		{
			name: "nonexistent file",
			setup: func() {
				// Don't create the file
			},
			args:    []string{"nonexistent.txt"},
			wantErr: false, // Should handle gracefully
			checkIdx: func(idx *index.Index) error {
				// Should not be in index
				if _, exists := idx.Get("nonexistent.txt"); exists {
					t.Error("nonexistent file should not be in index")
				}
				return nil
			},
		},
		{
			name: "wildcard pattern",
			setup: func() {
				helper.CreateFile("pattern1.txt", "content1")
				helper.CreateFile("pattern2.txt", "content2")
				helper.CreateFile("other.log", "log content")
			},
			args:    []string{"*.txt"},
			wantErr: false,
			checkIdx: func(idx *index.Index) error {
				if _, exists := idx.Get("pattern1.txt"); !exists {
					t.Error("pattern1.txt not found in index")
				}
				if _, exists := idx.Get("pattern2.txt"); !exists {
					t.Error("pattern2.txt not found in index")
				}
				// Should not include .log file
				if _, exists := idx.Get("other.log"); exists {
					t.Error("other.log should not be in index")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any previous files
			files, _ := filepath.Glob(filepath.Join(helper.TmpDir(), "*"))
			for _, file := range files {
				if filepath.Base(file) != ".git" {
					os.RemoveAll(file)
				}
			}
			
			// Remove index if exists
			indexPath := filepath.Join(repo.GitDir(), "index")
			os.Remove(indexPath)
			
			// Setup
			if tt.setup != nil {
				tt.setup()
			}

			// Create command and run
			cmd := newAddCommand()
			result := helper.RunCommand(cmd, tt.args, tt.flags)
			
			// Check error expectation
			result.AssertError(t, tt.wantErr)

			// Check expected output
			if len(tt.wantOut) > 0 {
				result.AssertContains(t, tt.wantOut...)
			}

			// Check index if specified and no error expected
			if tt.checkIdx != nil && !tt.wantErr {
				idx := index.New()
				if _, err := os.Stat(indexPath); err == nil {
					if err := idx.ReadFromFile(indexPath); err != nil {
						t.Errorf("Failed to read index: %v", err)
					}
				}
				
				tt.checkIdx(idx)
			}
		})
	}
}

func TestExpandPath_Fixed(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()
	
	helper.ChDir()

	// Create test files
	helper.CreateFile("file1.txt", "content")
	helper.CreateFile("file2.txt", "content")
	helper.CreateFile("test.log", "content")

	tests := []struct {
		name     string
		pattern  string
		wantLen  int
		wantErr  bool
	}{
		{
			name:    "single file",
			pattern: "file1.txt",
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "wildcard pattern",
			pattern: "*.txt",
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "nonexistent file",
			pattern: "nonexistent.txt",
			wantLen: 1, // Should still return the path for potential removal
			wantErr: false,
		},
		{
			name:    "absolute path outside repo",
			pattern: "/etc/passwd",
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, err := expandPath(helper.TmpDir(), tt.pattern)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("expandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(paths) != tt.wantLen {
				t.Errorf("expandPath() returned %d paths, want %d", len(paths), tt.wantLen)
			}
		})
	}
}