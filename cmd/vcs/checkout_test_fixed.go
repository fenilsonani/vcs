package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/internal/core/refs"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewCheckoutCommand_Fixed(t *testing.T) {
	cmd := newCheckoutCommand()
	
	if cmd.Use != "checkout [flags] <branch|commit>" {
		t.Errorf("Expected Use to be 'checkout [flags] <branch|commit>', got %s", cmd.Use)
	}
	
	if cmd.Short != "Switch branches or restore working tree files" {
		t.Errorf("Expected Short description, got %s", cmd.Short)
	}
	
	// Check flags exist
	if cmd.Flags().Lookup("force") == nil {
		t.Error("Expected --force flag to exist")
	}
	if cmd.Flags().Lookup("create") == nil {
		t.Error("Expected --create flag to exist")
	}
}

func TestRunCheckout_Fixed(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Initialize repository
	helper.ChDir()
	repo, err := vcs.Init(helper.TmpDir())
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	refManager := refs.NewRefManager(repo.GitDir())

	// Create commits and branches for testing
	setupRepo := func() (objects.ObjectID, objects.ObjectID) {
		// Create first commit
		content1 := []byte("main content")
		blob1 := repo.CreateBlobDirect(content1)
		tree1, _ := repo.CreateTree([]objects.TreeEntry{
			{Mode: objects.ModeBlob, Name: "main.txt", ID: blob1.ID()},
		})
		commit1, _ := repo.CreateCommit(tree1.ID(), nil, objects.Signature{
			Name: "Test", Email: "test@example.com", When: time.Now(),
		}, objects.Signature{
			Name: "Test", Email: "test@example.com", When: time.Now(),
		}, "Main commit")

		// Create second commit for feature branch
		content2 := []byte("feature content")
		blob2 := repo.CreateBlobDirect(content2)
		tree2, _ := repo.CreateTree([]objects.TreeEntry{
			{Mode: objects.ModeBlob, Name: "feature.txt", ID: blob2.ID()},
		})
		commit2, _ := repo.CreateCommit(tree2.ID(), []objects.ObjectID{commit1.ID()}, objects.Signature{
			Name: "Test", Email: "test@example.com", When: time.Now(),
		}, objects.Signature{
			Name: "Test", Email: "test@example.com", When: time.Now(),
		}, "Feature commit")

		// Set up branches
		refManager.CreateBranch("main", commit1.ID())
		refManager.CreateBranch("feature", commit2.ID()) 
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
		checkBranch  string
	}{
		{
			name:    "no arguments",
			setup:   setupRepo,
			args:    []string{},
			wantErr: true,
		},
		{
			name:         "checkout existing branch",
			setup:        setupRepo,
			args:         []string{"feature"},
			wantErr:      false,
			wantContains: []string{"Switched to branch 'feature'"},
			checkBranch:  "feature",
		},
		{
			name:         "checkout commit by ID",
			setup:        setupRepo,
			args:         []string{}, // Will be set dynamically
			wantErr:      false,
			wantContains: []string{"HEAD is now at"},
			checkBranch:  "", // Detached HEAD
		},
		{
			name:         "create and checkout new branch",
			setup:        setupRepo,
			args:         []string{"newbranch"},
			flags:        map[string]string{"create": "true"},
			wantErr:      false,
			wantContains: []string{"Switched to a new branch 'newbranch'"},
			checkBranch:  "newbranch",
		},
		{
			name:    "checkout nonexistent branch",
			setup:   setupRepo,
			args:    []string{"nonexistent"},
			wantErr: true,
		},
		{
			name:    "checkout invalid commit",
			setup:   setupRepo,
			args:    []string{"invalidcommit"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset repository state
			os.RemoveAll(filepath.Join(repo.GitDir(), "refs", "heads"))
			os.MkdirAll(filepath.Join(repo.GitDir(), "refs", "heads"), 0755)
			os.Remove(filepath.Join(repo.GitDir(), "HEAD"))
			
			// Setup
			var commit1ID objects.ObjectID
			if tt.setup != nil {
				commit1ID, _ = tt.setup()
			}

			// Handle dynamic args (for commit ID test)
			args := tt.args
			if tt.name == "checkout commit by ID" && !commit1ID.IsZero() {
				args = []string{commit1ID.String()}
			}

			// Create command and run using TestHelper
			cmd := newCheckoutCommand()
			result := helper.RunCommand(cmd, args, tt.flags)
			
			// Check error expectation
			result.AssertError(t, tt.wantErr)

			if !tt.wantErr {
				// Check output contains expected strings
				for _, want := range tt.wantContains {
					result.AssertContains(t, want)
				}

				// Check current branch
				if tt.checkBranch != "" {
					currentBranch, err := refManager.CurrentBranch()
					if err != nil {
						t.Errorf("Failed to get current branch: %v", err)
					} else if currentBranch != tt.checkBranch {
						t.Errorf("Current branch = %v, want %v", currentBranch, tt.checkBranch)
					}
				}
			}
		})
	}
}

func TestCreateAndCheckoutBranch_Fixed(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Initialize repository
	helper.ChDir()
	repo, err := vcs.Init(helper.TmpDir())
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
		Name: "Test", Email: "test@example.com", When: time.Now(),
	}, objects.Signature{
		Name: "Test", Email: "test@example.com", When: time.Now(),
	}, "Test commit")

	refManager.CreateBranch("main", commit.ID())
	refManager.SetHEAD("refs/heads/main")

	tests := []struct {
		name       string
		branchName string
		force      bool
		wantErr    bool
		wantOut    []string
	}{
		{
			name:       "create valid branch",
			branchName: "newbranch",
			force:      false,
			wantErr:    false,
			wantOut:    []string{"Switched to a new branch 'newbranch'"},
		},
		{
			name:       "create existing branch without force",
			branchName: "main",
			force:      false,
			wantErr:    true,
		},
		{
			name:       "create existing branch with force",
			branchName: "main",
			force:      true,
			wantErr:    false,
			wantOut:    []string{"Switched to a new branch 'main'"},
		},
		{
			name:       "invalid branch name",
			branchName: "invalid..name",
			force:      false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCheckoutCommand()
			
			// Use TestHelper to capture output
			result := helper.RunCommand(cmd, []string{tt.branchName}, map[string]string{
				"create": "true",
				"force":  func() string { if tt.force { return "true" }; return "false" }(),
			})
			
			result.AssertError(t, tt.wantErr)

			if !tt.wantErr {
				// Check expected output
				if len(tt.wantOut) > 0 {
					result.AssertContains(t, tt.wantOut...)
				}

				// Check branch was created and checked out
				if !refManager.RefExists(tt.branchName) {
					t.Errorf("Branch %s was not created", tt.branchName)
				}
				
				currentBranch, err := refManager.CurrentBranch()
				if err != nil {
					t.Errorf("Failed to get current branch: %v", err)
				} else if currentBranch != tt.branchName {
					t.Errorf("Current branch = %v, want %v", currentBranch, tt.branchName)
				}
			}
		})
	}
}

func TestHasUncommittedChanges_Fixed(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Initialize repository
	helper.ChDir()
	repo, err := vcs.Init(helper.TmpDir())
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	refManager := refs.NewRefManager(repo.GitDir())

	tests := []struct {
		name         string
		setup        func() error
		wantChanges  bool
		wantErr      bool
	}{
		{
			name: "no index file",
			setup: func() error {
				return nil
			},
			wantChanges: false,
			wantErr:     false,
		},
		{
			name: "empty index",
			setup: func() error {
				idx := index.New()
				return idx.WriteToFile(filepath.Join(repo.GitDir(), "index"))
			},
			wantChanges: false,
			wantErr:     false,
		},
		{
			name: "index with entries",
			setup: func() error {
				idx := index.New()
				content := []byte("test")
				blob := repo.CreateBlobDirect(content)
				entry := &index.Entry{
					Mode: objects.ModeBlob,
					Size: uint32(len(content)),
					ID:   blob.ID(),
					Path: "test.txt",
				}
				idx.Add(entry)
				return idx.WriteToFile(filepath.Join(repo.GitDir(), "index"))
			},
			wantChanges: true,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			os.Remove(filepath.Join(repo.GitDir(), "index"))
			
			// Setup
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			hasChanges, err := hasUncommittedChanges(repo, refManager)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("hasUncommittedChanges() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if hasChanges != tt.wantChanges {
				t.Errorf("hasUncommittedChanges() = %v, want %v", hasChanges, tt.wantChanges)
			}
		})
	}
}

func TestExtractFile_Fixed(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Initialize repository
	helper.ChDir()
	repo, err := vcs.Init(helper.TmpDir())
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	// Create test blob
	content := []byte("test file content")
	blob := repo.CreateBlobDirect(content)

	tests := []struct {
		name     string
		entry    objects.TreeEntry
		wantErr  bool
		checkMode os.FileMode
	}{
		{
			name: "extract regular file",
			entry: objects.TreeEntry{
				Mode: objects.ModeBlob,
				Name: "regular.txt",
				ID:   blob.ID(),
			},
			wantErr:   false,
			checkMode: 0644,
		},
		{
			name: "extract executable file",
			entry: objects.TreeEntry{
				Mode: objects.ModeExec,
				Name: "executable.sh",
				ID:   blob.ID(),
			},
			wantErr:   false,
			checkMode: 0755,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing file
			filePath := filepath.Join(helper.TmpDir(), tt.entry.Name)
			os.Remove(filePath)

			err := extractFile(repo, tt.entry, helper.TmpDir())
			
			if (err != nil) != tt.wantErr {
				t.Errorf("extractFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check file was created
				info, err := os.Stat(filePath)
				if err != nil {
					t.Errorf("File was not created: %v", err)
					return
				}

				// Check file mode (permissions)
				if info.Mode().Perm() != tt.checkMode {
					t.Errorf("File mode = %v, want %v", info.Mode().Perm(), tt.checkMode)
				}

				// Check file content
				readContent, err := os.ReadFile(filePath)
				if err != nil {
					t.Errorf("Failed to read file: %v", err)
					return
				}

				if string(readContent) != string(content) {
					t.Errorf("File content = %v, want %v", string(readContent), string(content))
				}
			}
		})
	}
}