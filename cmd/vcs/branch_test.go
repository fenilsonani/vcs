package main

import (
	"os"
	"strings"
	"testing"

	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewBranchCommand(t *testing.T) {
	cmd := newBranchCommand()
	
	if cmd.Use != "branch [flags] [branch_name] [start_point]" {
		t.Errorf("Expected Use to be 'branch [flags] [branch_name] [start_point]', got %s", cmd.Use)
	}
	
	if cmd.Short != "List, create, or delete branches" {
		t.Errorf("Expected Short description, got %s", cmd.Short)
	}
	
	// Check flags exist
	if cmd.Flags().Lookup("delete") == nil {
		t.Error("Expected --delete flag to exist")
	}
	if cmd.Flags().Lookup("force") == nil {
		t.Error("Expected --force flag to exist")
	}
	if cmd.Flags().Lookup("all") == nil {
		t.Error("Expected --all flag to exist")
	}
	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("Expected --verbose flag to exist")
	}
}

func TestListBranchesOperation(t *testing.T) {
	// Create temp directory for test repo  
	tmpDir, err := os.MkdirTemp("", "branch-list-test-*")
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

	// Create a commit to have branches point to
	content := []byte("test content")
	blob := repo.CreateBlobDirect(content)
	tree, _ := repo.CreateTree([]objects.TreeEntry{
		{Mode: objects.ModeBlob, Name: "test.txt", ID: blob.ID()},
	})
	commit, _ := repo.CreateCommit(tree.ID(), nil, objects.Signature{
		Name: "Test", Email: "test@example.com",
	}, objects.Signature{
		Name: "Test", Email: "test@example.com",
	}, "Test commit")

	// Create some branches
	refManager.CreateBranch("main", commit.ID())
	refManager.CreateBranch("feature", commit.ID())
	refManager.CreateBranch("develop", commit.ID())
	refManager.SetHEAD("refs/heads/main")

	tests := []struct {
		name         string
		showAll      bool
		verbose      bool
		wantContains []string
	}{
		{
			name:         "simple list",
			showAll:      false,
			verbose:      false,
			wantContains: []string{"* main", "  feature", "  develop"},
		},
		{
			name:         "verbose list",
			showAll:      false,
			verbose:      true,
			wantContains: []string{"* main", "Test commit", commit.ID().String()[:7]},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := listBranchesOperation(repo, refManager, tt.showAll, tt.verbose)
			if err != nil {
				t.Errorf("listBranchesOperation() error = %v", err)
			}

			w.Close()
			os.Stdout = oldStdout

			buf := make([]byte, 2048)
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

func TestCreateBranchOperation(t *testing.T) {
	// Create temp directory for test repo
	tmpDir, err := os.MkdirTemp("", "branch-create-test-*")
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

	// Create a commit
	content := []byte("test content")
	blob := repo.CreateBlobDirect(content)
	tree, _ := repo.CreateTree([]objects.TreeEntry{
		{Mode: objects.ModeBlob, Name: "test.txt", ID: blob.ID()},
	})
	commit, _ := repo.CreateCommit(tree.ID(), nil, objects.Signature{
		Name: "Test", Email: "test@example.com",
	}, objects.Signature{
		Name: "Test", Email: "test@example.com",
	}, "Test commit")

	// Set up main branch
	refManager.CreateBranch("main", commit.ID())
	refManager.SetHEAD("refs/heads/main")

	tests := []struct {
		name        string
		branchName  string
		startPoint  string
		wantErr     bool
		checkExists bool
	}{
		{
			name:        "create branch from HEAD",
			branchName:  "feature",
			startPoint:  "",
			wantErr:     false,
			checkExists: true,
		},
		{
			name:        "create branch from commit",
			branchName:  "hotfix",
			startPoint:  commit.ID().String(),
			wantErr:     false,
			checkExists: true,
		},
		{
			name:        "create branch from branch",
			branchName:  "develop", 
			startPoint:  "main",
			wantErr:     false,
			checkExists: true,
		},
		{
			name:        "invalid branch name",
			branchName:  "invalid..name",
			startPoint:  "",
			wantErr:     true,
			checkExists: false,
		},
		{
			name:        "branch already exists",
			branchName:  "main",
			startPoint:  "",
			wantErr:     true,
			checkExists: false,
		},
		{
			name:        "invalid start point",
			branchName:  "newbranch",
			startPoint:  "nonexistent",
			wantErr:     true,
			checkExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := createBranchOperation(repo, refManager, tt.branchName, tt.startPoint)

			w.Close()
			os.Stdout = oldStdout

			bufData := make([]byte, 1024)
			_, _ = r.Read(bufData)
			output := string(bufData)

			if (err != nil) != tt.wantErr {
				t.Errorf("createBranchOperation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkExists && !tt.wantErr {
				if !refManager.RefExists(tt.branchName) {
					t.Errorf("Branch %s was not created", tt.branchName)
				}
				if !strings.Contains(output, "Created branch") {
					t.Errorf("Expected success message, got: %s", output)
				}
			}
		})
	}
}

func TestDeleteBranchOperation(t *testing.T) {
	// Create temp directory for test repo
	tmpDir, err := os.MkdirTemp("", "branch-delete-test-*")
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

	// Create a commit and branches
	content := []byte("test content")
	blob := repo.CreateBlobDirect(content)
	tree, _ := repo.CreateTree([]objects.TreeEntry{
		{Mode: objects.ModeBlob, Name: "test.txt", ID: blob.ID()},
	})
	commit, _ := repo.CreateCommit(tree.ID(), nil, objects.Signature{
		Name: "Test", Email: "test@example.com",
	}, objects.Signature{
		Name: "Test", Email: "test@example.com",
	}, "Test commit")

	refManager.CreateBranch("main", commit.ID())
	refManager.CreateBranch("feature", commit.ID())
	refManager.CreateBranch("todelete", commit.ID())
	refManager.SetHEAD("refs/heads/main")

	tests := []struct {
		name        string
		args        []string
		force       bool
		wantErr     bool
		checkGone   []string
	}{
		{
			name:      "delete existing branch",
			args:      []string{"todelete"},
			force:     false,
			wantErr:   false,
			checkGone: []string{"todelete"},
		},
		{
			name:    "delete current branch",
			args:    []string{"main"},
			force:   false,
			wantErr: true,
		},
		{
			name:    "delete nonexistent branch",
			args:    []string{"nonexistent"},
			force:   false,
			wantErr: true,
		},
		{
			name:      "force delete nonexistent branch",
			args:      []string{"nonexistent"},
			force:     true,
			wantErr:   false,
		},
		{
			name:      "delete multiple branches",
			args:      []string{"feature"},
			force:     false,
			wantErr:   false,
			checkGone: []string{"feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := deleteBranchOperation(refManager, tt.args, tt.force)

			w.Close()
			os.Stdout = oldStdout

			bufData := make([]byte, 1024)
			_, _ = r.Read(bufData)

			if (err != nil) != tt.wantErr {
				t.Errorf("deleteBranchOperation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check branches were deleted
			for _, branchName := range tt.checkGone {
				if refManager.RefExists(branchName) {
					t.Errorf("Branch %s still exists after deletion", branchName)
				}
			}
		})
	}
}