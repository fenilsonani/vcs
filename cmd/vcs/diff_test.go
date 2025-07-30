package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewDiffCommand(t *testing.T) {
	cmd := newDiffCommand()

	if cmd.Use != "diff [flags] [commit] [commit] [-- path...]" {
		t.Errorf("Expected Use to be 'diff [flags] [commit] [commit] [-- path...]', got %s", cmd.Use)
	}

	if cmd.Short != "Show changes between commits, commit and working tree, etc" {
		t.Errorf("Expected Short description, got %s", cmd.Short)
	}

	// Check flags exist
	if cmd.Flags().Lookup("cached") == nil {
		t.Error("Expected --cached flag to exist")
	}
	if cmd.Flags().Lookup("name-only") == nil {
		t.Error("Expected --name-only flag to exist")
	}
	if cmd.Flags().Lookup("name-status") == nil {
		t.Error("Expected --name-status flag to exist")
	}
	if cmd.Flags().Lookup("unified") == nil {
		t.Error("Expected --unified flag to exist")
	}
}

func TestRunDiff(t *testing.T) {
	// Create temp directory for test repo
	tmpDir, err := os.MkdirTemp("", "diff-test-*")
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

	refManager := refs.NewRefManager(repo.GitDir())

	// Helper to create and commit a file
	createCommit := func(filename, content, message string) objects.ObjectID {
		// Write file
		os.WriteFile(filename, []byte(content), 0644)

		// Add to index
		idx := index.New()
		indexPath := filepath.Join(repo.GitDir(), "index")
		if _, err := os.Stat(indexPath); err == nil {
			idx.ReadFromFile(indexPath)
		}

		blob := repo.CreateBlobDirect([]byte(content))
		entry := &index.Entry{
			Mode: objects.ModeBlob,
			Size: uint32(len(content)),
			ID:   blob.ID(),
			Path: filename,
		}
		idx.Add(entry)
		idx.WriteToFile(indexPath)

		// Create tree from index
		tree, _ := createTreeFromIndex(repo, idx)

		// Create commit
		commit, _ := repo.CreateCommit(tree.ID(), nil, objects.Signature{
			Name: "Test", Email: "test@example.com", When: time.Now(),
		}, objects.Signature{
			Name: "Test", Email: "test@example.com", When: time.Now(),
		}, message)

		// Update HEAD
		refManager.CreateBranch("main", commit.ID())
		refManager.SetHEAD("refs/heads/main")

		return commit.ID()
	}

	tests := []struct {
		name         string
		setup        func() error
		args         []string
		flags        map[string]string
		wantErr      bool
		wantContains []string
	}{
		{
			name: "no changes",
			setup: func() error {
				createCommit("file.txt", "content", "Initial commit")
				return nil
			},
			args:    []string{},
			wantErr: false,
		},
		{
			name: "working tree changes",
			setup: func() error {
				createCommit("file.txt", "original content", "Initial commit")
				return os.WriteFile("file.txt", []byte("modified content"), 0644)
			},
			args:         []string{},
			wantErr:      false,
			wantContains: []string{"diff --git", "file.txt", "-original content", "+modified content"},
		},
		{
			name: "new file in working tree",
			setup: func() error {
				createCommit("file1.txt", "content1", "Initial commit")
				return os.WriteFile("file2.txt", []byte("new content"), 0644)
			},
			args:         []string{},
			wantErr:      false,
			wantContains: []string{"diff --git", "file2.txt", "new file mode", "+new content"},
		},
		{
			name: "name-only flag",
			setup: func() error {
				createCommit("file.txt", "original", "Initial commit")
				os.WriteFile("file.txt", []byte("modified"), 0644)
				return os.WriteFile("new.txt", []byte("new"), 0644)
			},
			args:         []string{},
			flags:        map[string]string{"name-only": "true"},
			wantErr:      false,
			wantContains: []string{"file.txt", "new.txt"},
		},
		{
			name: "name-status flag",
			setup: func() error {
				createCommit("file.txt", "original", "Initial commit")
				os.WriteFile("file.txt", []byte("modified"), 0644)
				return os.WriteFile("new.txt", []byte("new"), 0644)
			},
			args:         []string{},
			flags:        map[string]string{"name-status": "true"},
			wantErr:      false,
			wantContains: []string{"M\tfile.txt", "A\tnew.txt"},
		},
		{
			name: "cached diff",
			setup: func() error {
				commit1 := createCommit("file.txt", "original", "Initial commit")
				
				// Modify and stage
				os.WriteFile("file.txt", []byte("staged content"), 0644)
				idx := index.New()
				indexPath := filepath.Join(repo.GitDir(), "index")
				idx.ReadFromFile(indexPath)
				
				blob := repo.CreateBlobDirect([]byte("staged content"))
				entry := &index.Entry{
					Mode: objects.ModeBlob,
					Size: uint32(len("staged content")),
					ID:   blob.ID(),
					Path: "file.txt",
				}
				idx.Add(entry)
				idx.WriteToFile(indexPath)
				
				_ = commit1
				return nil
			},
			args:         []string{},
			flags:        map[string]string{"cached": "true"},
			wantErr:      false,
			wantContains: []string{"diff --git", "file.txt", "-original", "+staged content"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			files, _ := filepath.Glob(filepath.Join(tmpDir, "*.txt"))
			for _, file := range files {
				os.Remove(file)
			}
			os.Remove(filepath.Join(repo.GitDir(), "index"))
			os.Remove(filepath.Join(repo.GitDir(), "HEAD"))
			os.RemoveAll(filepath.Join(repo.GitDir(), "refs"))
			os.MkdirAll(filepath.Join(repo.GitDir(), "refs", "heads"), 0755)

			// Setup
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Create command
			cmd := newDiffCommand()

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

			if !tt.wantErr {
				// Check output contains expected strings
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

func TestDiffWorkingTreeToIndex(t *testing.T) {
	// Create temp directory for test repo
	tmpDir, err := os.MkdirTemp("", "diff-working-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repository
	repo, err := vcs.Init(tmpDir)
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	tests := []struct {
		name       string
		setup      func() error
		nameOnly   bool
		nameStatus bool
		wantOutput []string
		wantErr    bool
	}{
		{
			name: "no changes",
			setup: func() error {
				// Create empty index
				idx := index.New()
				return idx.WriteToFile(filepath.Join(repo.GitDir(), "index"))
			},
			wantErr: false,
		},
		{
			name: "new file in working tree",
			setup: func() error {
				// Create file in working tree
				return os.WriteFile(filepath.Join(tmpDir, "new.txt"), []byte("new content"), 0644)
			},
			wantOutput: []string{"diff --git", "new.txt", "new file mode"},
			wantErr:    false,
		},
		{
			name: "name-only output",
			setup: func() error {
				return os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)
			},
			nameOnly:   true,
			wantOutput: []string{"file.txt"},
			wantErr:    false,
		},
		{
			name: "name-status output",
			setup: func() error {
				return os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644)
			},
			nameStatus: true,
			wantOutput: []string{"A\tfile.txt"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			files, _ := filepath.Glob(filepath.Join(tmpDir, "*.txt"))
			for _, file := range files {
				os.Remove(file)
			}
			os.Remove(filepath.Join(repo.GitDir(), "index"))

			// Setup
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := diffWorkingTreeToIndex(repo, tt.nameOnly, tt.nameStatus, 3)

			w.Close()
			os.Stdout = oldStdout

			buf := make([]byte, 2048)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("diffWorkingTreeToIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check output
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing expected string %q\nGot: %s", want, output)
				}
			}
		})
	}
}

func TestPrintUnifiedDiff(t *testing.T) {
	tests := []struct {
		name        string
		oldContent  []byte
		newContent  []byte
		contextLines int
		wantContains []string
	}{
		{
			name:        "new file",
			oldContent:  nil,
			newContent:  []byte("line1\nline2\nline3"),
			contextLines: 3,
			wantContains: []string{"@@", "+line1", "+line2", "+line3"},
		},
		{
			name:        "deleted file",
			oldContent:  []byte("line1\nline2\nline3"),
			newContent:  nil,
			contextLines: 3,
			wantContains: []string{"@@", "-line1", "-line2", "-line3"},
		},
		{
			name:        "modified file",
			oldContent:  []byte("line1\noriginal\nline3"),
			newContent:  []byte("line1\nmodified\nline3"),
			contextLines: 1,
			wantContains: []string{"@@", " line1", "-original", "+modified", " line3"},
		},
		{
			name:        "no changes",
			oldContent:  []byte("same content"),
			newContent:  []byte("same content"),
			contextLines: 3,
			wantContains: []string{}, // Should output nothing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printUnifiedDiff(tt.oldContent, tt.newContent, tt.contextLines)

			w.Close()
			os.Stdout = oldStdout

			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing expected string %q\nGot: %s", want, output)
				}
			}
		})
	}
}

func TestGetObjectContent(t *testing.T) {
	// Create temp directory for test repo
	tmpDir, err := os.MkdirTemp("", "get-object-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repository
	repo, err := vcs.Init(tmpDir)
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	// Create a blob
	content := []byte("test content")
	blob := repo.CreateBlobDirect(content)

	// Test getting content
	retrievedContent := getObjectContent(repo, blob.ID())
	if !bytes.Equal(retrievedContent, content) {
		t.Errorf("Retrieved content = %q, want %q", retrievedContent, content)
	}

	// Test with zero ID
	zeroContent := getObjectContent(repo, objects.ObjectID{})
	if zeroContent != nil {
		t.Errorf("Expected nil for zero ID, got %q", zeroContent)
	}
}

