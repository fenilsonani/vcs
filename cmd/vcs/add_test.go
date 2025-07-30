package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewAddCommand(t *testing.T) {
	cmd := newAddCommand()
	
	if cmd.Use != "add [flags] [pathspec...]" {
		t.Errorf("Expected Use to be 'add [flags] [pathspec...]', got %s", cmd.Use)
	}
	
	if cmd.Short != "Add file contents to the index" {
		t.Errorf("Expected Short description, got %s", cmd.Short)
	}
	
	// Check flags exist
	if cmd.Flags().Lookup("all") == nil {
		t.Error("Expected --all flag to exist")
	}
	if cmd.Flags().Lookup("force") == nil {
		t.Error("Expected --force flag to exist")
	}
	if cmd.Flags().Lookup("dry-run") == nil {
		t.Error("Expected --dry-run flag to exist")
	}
	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("Expected --verbose flag to exist")
	}
}

func TestRunAdd(t *testing.T) {
	// Create temp directory for test repo
	tmpDir, err := os.MkdirTemp("", "add-test-*")
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
		name     string
		setup    func() error
		args     []string
		flags    map[string]string
		wantErr  bool
		checkIdx func(*index.Index) error
	}{
		{
			name: "add single file",
			setup: func() error {
				return os.WriteFile("test.txt", []byte("test content"), 0644)
			},
			args:    []string{"test.txt"},
			wantErr: false,
			checkIdx: func(idx *index.Index) error {
				if _, exists := idx.Get("test.txt"); !exists {
					return errors.New("file not found in index")
				}
				return nil
			},
		},
		{
			name: "add multiple files",
			setup: func() error {
				os.WriteFile("file1.txt", []byte("content1"), 0644)
				os.WriteFile("file2.txt", []byte("content2"), 0644)
				return nil
			},
			args:    []string{"file1.txt", "file2.txt"},
			wantErr: false,
			checkIdx: func(idx *index.Index) error {
				if _, exists := idx.Get("file1.txt"); !exists {
					return errors.New("file1.txt not found in index")
				}
				if _, exists := idx.Get("file2.txt"); !exists {
					return errors.New("file2.txt not found in index")
				}
				return nil
			},
		},
		{
			name: "add all files",
			setup: func() error {
				os.WriteFile("all1.txt", []byte("content1"), 0644)
				os.WriteFile("all2.txt", []byte("content2"), 0644)
				return nil
			},
			args:    []string{},
			flags:   map[string]string{"all": "true"},
			wantErr: false,
			checkIdx: func(idx *index.Index) error {
				if _, exists := idx.Get("all1.txt"); !exists {
					return errors.New("all1.txt not found in index")
				}
				if _, exists := idx.Get("all2.txt"); !exists {
					return errors.New("all2.txt not found in index")
				}
				return nil
			},
		},
		{
			name: "dry run",
			setup: func() error {
				return os.WriteFile("dryrun.txt", []byte("content"), 0644)
			},
			args:    []string{"dryrun.txt"},
			flags:   map[string]string{"dry-run": "true"},
			wantErr: false,
			checkIdx: func(idx *index.Index) error {
				if _, exists := idx.Get("dryrun.txt"); exists {
					return errors.New("file should not be in index during dry run")
				}
				return nil
			},
		},
		{
			name: "no files specified",
			setup: func() error {
				return nil
			},
			args:    []string{},
			wantErr: true,
		},
		{
			name: "nonexistent file",
			setup: func() error {
				return nil
			},
			args:    []string{"nonexistent.txt"},
			wantErr: false, // Should handle gracefully
			checkIdx: func(idx *index.Index) error {
				// Should not be in index
				if _, exists := idx.Get("nonexistent.txt"); exists {
					return errors.New("nonexistent file should not be in index")
				}
				return nil
			},
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
			indexPath := filepath.Join(repo.GitDir(), "index")
			os.Remove(indexPath)
			
			// Setup
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Create command
			cmd := newAddCommand()
			
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

			// Check index if specified
			if tt.checkIdx != nil && !tt.wantErr {
				idx := index.New()
				if _, err := os.Stat(indexPath); err == nil {
					idx.ReadFromFile(indexPath)
				}
				
				if err := tt.checkIdx(idx); err != nil {
					t.Errorf("Index check failed: %v", err)
				}
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "expand-path-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "test.log"), []byte("content"), 0644)

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
			paths, err := expandPath(tmpDir, tt.pattern)
			
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

