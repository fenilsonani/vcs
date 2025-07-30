package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewCommitCommand(t *testing.T) {
	cmd := newCommitCommand()
	
	if cmd.Use != "commit" {
		t.Errorf("Expected Use to be 'commit', got %s", cmd.Use)
	}
	
	if cmd.Short != "Record changes to the repository" {
		t.Errorf("Expected Short description, got %s", cmd.Short)
	}
	
	// Check flags exist
	if cmd.Flags().Lookup("message") == nil {
		t.Error("Expected --message flag to exist")
	}
	if cmd.Flags().Lookup("file") == nil {
		t.Error("Expected --file flag to exist")
	}
}

func TestRunCommit(t *testing.T) {
	// Create temp directory for test repo
	tmpDir, err := os.MkdirTemp("", "commit-test-*")
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
		wantErr      bool
		wantContains []string
	}{
		{
			name: "no message",
			setup: func() error {
				return nil
			},
			args:    []string{},
			wantErr: true,
		},
		{
			name: "nothing to commit",
			setup: func() error {
				return nil
			},
			args:    []string{},
			flags:   map[string]string{"message": "test commit"},
			wantErr: true,
		},
		{
			name: "successful commit",
			setup: func() error {
				// Add file to index
				content := []byte("test content")
				os.WriteFile("test.txt", content, 0644)
				
				idx := index.New()
				blob := repo.CreateBlobDirect(content)
				entry := &index.Entry{
					Mode: 0644,
					Size: uint32(len(content)),
					ID:   blob.ID(),
					Path: "test.txt",
				}
				idx.Add(entry)
				return idx.WriteToFile(filepath.Join(repo.GitDir(), "index"))
			},
			args:         []string{},
			flags:        map[string]string{"message": "Test commit message"},
			wantErr:      false,
			wantContains: []string{"Test commit message", "1 file(s) changed"},
		},
		{
			name: "commit with file message",
			setup: func() error {
				// Create message file
				os.WriteFile("commit-msg.txt", []byte("File message"), 0644)
				
				// Add file to index
				content := []byte("content")
				os.WriteFile("file.txt", content, 0644)
				
				idx := index.New()
				blob := repo.CreateBlobDirect(content)
				entry := &index.Entry{
					Mode: 0644,
					Size: uint32(len(content)),
					ID:   blob.ID(),
					Path: "file.txt",
				}
				idx.Add(entry)
				return idx.WriteToFile(filepath.Join(repo.GitDir(), "index"))
			},
			args:         []string{},
			flags:        map[string]string{"file": "commit-msg.txt"},
			wantErr:      false,
			wantContains: []string{"File message", "1 file(s) changed"},
		},
		{
			name: "allow empty commit",
			setup: func() error {
				return nil
			},
			args:         []string{},
			flags:        map[string]string{"message": "Empty commit", "allow-empty": "true"},
			wantErr:      false,
			wantContains: []string{"Empty commit", "0 file(s) changed"},
		},
		{
			name: "custom author",
			setup: func() error {
				// Add file to index
				content := []byte("author test")
				os.WriteFile("author.txt", content, 0644)
				
				idx := index.New()
				blob := repo.CreateBlobDirect(content)
				entry := &index.Entry{
					Mode: 0644,
					Size: uint32(len(content)),
					ID:   blob.ID(),
					Path: "author.txt",
				}
				idx.Add(entry)
				return idx.WriteToFile(filepath.Join(repo.GitDir(), "index"))
			},
			args:         []string{},
			flags:        map[string]string{"message": "Custom author", "author": "John Doe <john@example.com>"},
			wantErr:      false,
			wantContains: []string{"Custom author", "1 file(s) changed"},
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
			cmd := newCommitCommand()
			
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
			if !tt.wantErr {
				output := buf.String()
				for _, want := range tt.wantContains {
					if !strings.Contains(output, want) {
						t.Errorf("Output missing expected string %q\nGot: %s", want, output)
					}
				}
			}
		})
	}
}

func TestGetSignature(t *testing.T) {
	tests := []struct {
		name      string
		authorStr string
		wantName  string
		wantEmail string
		wantErr   bool
	}{
		{
			name:      "valid author string",
			authorStr: "John Doe <john@example.com>",
			wantName:  "John Doe",
			wantEmail: "john@example.com",
			wantErr:   false,
		},
		{
			name:      "empty author string",
			authorStr: "",
			wantName:  "VCS User",
			wantEmail: "user@example.com",
			wantErr:   false,
		},
		{
			name:      "invalid author format",
			authorStr: "invalid format",
			wantName:  "",
			wantEmail: "",
			wantErr:   true,
		},
		{
			name:      "missing email bracket",
			authorStr: "John Doe john@example.com",
			wantName:  "",
			wantEmail: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig, err := getSignature(tt.authorStr)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("getSignature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if sig.Name != tt.wantName {
					t.Errorf("getSignature() name = %v, want %v", sig.Name, tt.wantName)
				}
				if sig.Email != tt.wantEmail {
					t.Errorf("getSignature() email = %v, want %v", sig.Email, tt.wantEmail)
				}
			}
		})
	}
}

func TestCreateTreeFromIndex(t *testing.T) {
	// Create temp directory for test repo
	tmpDir, err := os.MkdirTemp("", "tree-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repository
	repo, err := vcs.Init(tmpDir)
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	// Create index with entries
	idx := index.New()
	
	// Add entries to index
	content1 := []byte("content 1")
	content2 := []byte("content 2")
	
	blob1 := repo.CreateBlobDirect(content1)
	blob2 := repo.CreateBlobDirect(content2)
	
	entry1 := &index.Entry{
		Mode: objects.ModeBlob,
		Size: uint32(len(content1)),
		ID:   blob1.ID(),
		Path: "file1.txt",
	}
	
	entry2 := &index.Entry{
		Mode: objects.ModeExec,
		Size: uint32(len(content2)),
		ID:   blob2.ID(),
		Path: "script.sh",
	}
	
	idx.Add(entry1)
	idx.Add(entry2)

	// Create tree from index
	tree, err := createTreeFromIndex(repo, idx)
	if err != nil {
		t.Fatalf("createTreeFromIndex() error = %v", err)
	}

	if tree == nil {
		t.Fatal("createTreeFromIndex() returned nil tree")
	}

	// Verify tree entries
	entries := tree.Entries()
	if len(entries) != 2 {
		t.Errorf("Expected 2 tree entries, got %d", len(entries))
	}

	// Check entries are present
	entryMap := make(map[string]objects.TreeEntry)
	for _, entry := range entries {
		entryMap[entry.Name] = entry
	}

	if entry, exists := entryMap["file1.txt"]; !exists {
		t.Error("file1.txt not found in tree")
	} else if entry.Mode != objects.ModeBlob {
		t.Errorf("file1.txt mode = %v, want %v", entry.Mode, objects.ModeBlob)
	}

	if entry, exists := entryMap["script.sh"]; !exists {
		t.Error("script.sh not found in tree")
	} else if entry.Mode != objects.ModeExec {
		t.Errorf("script.sh mode = %v, want %v", entry.Mode, objects.ModeExec)
	}
}

func TestGetCurrentBranchName(t *testing.T) {
	// Create temp directory for test repo
	tmpDir, err := os.MkdirTemp("", "branch-name-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repository
	repo, err := vcs.Init(tmpDir)
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	refManager := refs.NewRefManager(repo.GitDir())

	// Test default branch
	branchName := getCurrentBranchName(refManager)
	if branchName != "main" {
		t.Errorf("getCurrentBranchName() = %v, want 'main'", branchName)
	}

	// Test detached HEAD (simulate by setting HEAD to a commit ID)
	commitID := objects.ObjectID{1, 2, 3} // dummy ID
	refManager.SetHEADToCommit(commitID)
	
	branchName = getCurrentBranchName(refManager)
	if branchName != "HEAD" {
		t.Errorf("getCurrentBranchName() for detached HEAD = %v, want 'HEAD'", branchName)
	}
}