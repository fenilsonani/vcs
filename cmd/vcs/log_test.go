package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewLogCommand(t *testing.T) {
	cmd := newLogCommand()
	
	if cmd.Use != "log" {
		t.Errorf("Expected Use to be 'log', got %s", cmd.Use)
	}
	
	if cmd.Short != "Show commit logs" {
		t.Errorf("Expected Short description, got %s", cmd.Short)
	}
	
	// Check flags exist
	if cmd.Flags().Lookup("max-count") == nil {
		t.Error("Expected --max-count flag to exist")
	}
}

func TestRunLog(t *testing.T) {
	// Create temp directory for test repo
	tmpDir, err := os.MkdirTemp("", "log-test-*")
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

	// Create commits for testing
	setupCommits := func() (objects.ObjectID, objects.ObjectID) {
		// First commit
		content1 := []byte("first content")
		blob1 := repo.CreateBlobDirect(content1)
		tree1, _ := repo.CreateTree([]objects.TreeEntry{
			{Mode: objects.ModeBlob, Name: "file1.txt", ID: blob1.ID()},
		})
		commit1, _ := repo.CreateCommit(tree1.ID(), nil, objects.Signature{
			Name: "Author 1", Email: "author1@example.com", When: time.Now().Add(-time.Hour),
		}, objects.Signature{
			Name: "Author 1", Email: "author1@example.com", When: time.Now().Add(-time.Hour),
		}, "First commit")

		// Second commit
		content2 := []byte("second content")
		blob2 := repo.CreateBlobDirect(content2)
		tree2, _ := repo.CreateTree([]objects.TreeEntry{
			{Mode: objects.ModeBlob, Name: "file2.txt", ID: blob2.ID()},
		})
		commit2, _ := repo.CreateCommit(tree2.ID(), []objects.ObjectID{commit1.ID()}, objects.Signature{
			Name: "Author 2", Email: "author2@example.com", When: time.Now(),
		}, objects.Signature{
			Name: "Author 2", Email: "author2@example.com", When: time.Now(),
		}, "Second commit")

		// Set HEAD to second commit
		refManager.CreateBranch("main", commit2.ID())
		refManager.SetHEAD("refs/heads/main")

		return commit1.ID(), commit2.ID()
	}

	tests := []struct {
		name         string
		setup        func() (objects.ObjectID, objects.ObjectID)
		args         []string
		flags        map[string]string
		wantErr      bool
		wantContains []string
	}{
		{
			name: "no commits",
			setup: func() (objects.ObjectID, objects.ObjectID) {
				return objects.ObjectID{}, objects.ObjectID{}
			},
			args:         []string{},
			wantErr:      false,
			wantContains: []string{"No commits found"},
		},
		{
			name:         "full log",
			setup:        setupCommits,
			args:         []string{},
			wantErr:      false,
			wantContains: []string{"commit", "Author:", "Date:", "Second commit", "First commit"},
		},
		{
			name:         "oneline log",
			setup:        setupCommits,
			args:         []string{},
			flags:        map[string]string{"oneline": "true"},
			wantErr:      false,
			wantContains: []string{"Second commit", "First commit"},
		},
		{
			name:         "limited log",
			setup:        setupCommits,
			args:         []string{},
			flags:        map[string]string{"max-count": "1"},
			wantErr:      false,
			wantContains: []string{"Second commit"},
		},
		{
			name:         "pretty format short",
			setup:        setupCommits,
			args:         []string{},
			flags:        map[string]string{"pretty": "short"},
			wantErr:      false,
			wantContains: []string{"commit", "Author:", "Second commit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset repository state
			os.RemoveAll(filepath.Join(repo.GitDir(), "refs", "heads"))
			os.MkdirAll(filepath.Join(repo.GitDir(), "refs", "heads"), 0755)
			os.Remove(filepath.Join(repo.GitDir(), "HEAD"))
			
			// Setup
			if tt.setup != nil {
				tt.setup()
			}

			// Create command
			cmd := newLogCommand()
			
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

func TestPrintCommitOneline(t *testing.T) {
	// Create a test commit
	sig := objects.Signature{Name: "Test", Email: "test@example.com", When: time.Now()}
	commit := objects.NewCommit(
		objects.ObjectID{1, 2, 3}, // dummy tree ID
		nil,                       // no parents
		sig, sig,
		"Test commit message\nWith multiple lines",
	)

	commitID := objects.ObjectID{4, 5, 6} // dummy commit ID

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printCommitOneline(commitID, commit)

	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Should contain short hash and first line of message
	expectedHash := commitID.String()[:7]
	if !strings.Contains(output, expectedHash) {
		t.Errorf("Output missing expected hash %q\nGot: %s", expectedHash, output)
	}
	if !strings.Contains(output, "Test commit message") {
		t.Errorf("Output missing expected message\nGot: %s", output)
	}
	if strings.Contains(output, "With multiple lines") {
		t.Errorf("Output should not contain second line of message\nGot: %s", output)
	}
}

func TestPrintCommitFull(t *testing.T) {
	// Create a test commit with parents
	sig := objects.Signature{Name: "Test Author", Email: "test@example.com", When: time.Now()}
	parentID := objects.ObjectID{7, 8, 9}
	commit := objects.NewCommit(
		objects.ObjectID{1, 2, 3}, // dummy tree ID
		[]objects.ObjectID{parentID},
		sig, sig,
		"Test commit message",
	)

	commitID := objects.ObjectID{4, 5, 6} // dummy commit ID

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printCommitFull(commitID, commit, false, true)

	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check expected content
	expectedContent := []string{
		"commit " + commitID.String(),
		"Author: Test Author <test@example.com>",
		"Date:",
		"Test commit message",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Output missing expected content %q\nGot: %s", expected, output)
		}
	}
}

func TestFormatDate(t *testing.T) {
	// Test date formatting
	testTime := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
	formatted := formatDate(testTime)
	
	// Should contain day, month, time
	expectedParts := []string{"Mon", "Dec", "25", "15:30:45", "2023"}
	for _, part := range expectedParts {
		if !strings.Contains(formatted, part) {
			t.Errorf("Formatted date missing expected part %q\nGot: %s", part, formatted)
		}
	}
}