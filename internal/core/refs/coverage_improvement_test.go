package refs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fenilsonani/vcs/internal/core/objects"
)

// Test readRefFile function more comprehensively (currently 77.8% coverage)
func TestReadRefFileCoverage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	refManager := NewRefManager(tmpDir)

	tests := []struct {
		name     string
		setup    func() string
		wantErr  bool
		checkID  bool
	}{
		{
			name: "valid ref with hash",
			setup: func() string {
				refPath := filepath.Join(tmpDir, "refs", "heads", "main")
				os.MkdirAll(filepath.Dir(refPath), 0755)
				hash := "abcdef1234567890abcdef1234567890abcdef12"
				os.WriteFile(refPath, []byte(hash+"\n"), 0644)
				return "refs/heads/main"
			},
			wantErr: false,
			checkID: true,
		},
		{
			name: "valid ref with symbolic ref",
			setup: func() string {
				refPath := filepath.Join(tmpDir, "refs", "heads", "feature")
				os.MkdirAll(filepath.Dir(refPath), 0755)
				os.WriteFile(refPath, []byte("ref: refs/heads/main\n"), 0644)
				
				// Also create the target
				mainPath := filepath.Join(tmpDir, "refs", "heads", "main")
				hash := "1234567890abcdef1234567890abcdef12345678"
				os.WriteFile(mainPath, []byte(hash), 0644)
				
				return "refs/heads/feature"
			},
			wantErr: false,
			checkID: true,
		},
		{
			name: "non-existent ref",
			setup: func() string {
				return "refs/heads/nonexistent"
			},
			wantErr: true,
			checkID: false,
		},
		{
			name: "empty ref file",
			setup: func() string {
				refPath := filepath.Join(tmpDir, "refs", "heads", "empty")
				os.MkdirAll(filepath.Dir(refPath), 0755)
				os.WriteFile(refPath, []byte(""), 0644)
				return "refs/heads/empty"
			},
			wantErr: true,
			checkID: false,
		},
		{
			name: "invalid hash in ref",
			setup: func() string {
				refPath := filepath.Join(tmpDir, "refs", "heads", "invalid")
				os.MkdirAll(filepath.Dir(refPath), 0755)
				os.WriteFile(refPath, []byte("invalid-hash\n"), 0644)
				return "refs/heads/invalid"
			},
			wantErr: true,
			checkID: false,
		},
		{
			name: "circular symbolic ref",
			setup: func() string {
				refPath1 := filepath.Join(tmpDir, "refs", "heads", "circular1")
				refPath2 := filepath.Join(tmpDir, "refs", "heads", "circular2")
				os.MkdirAll(filepath.Dir(refPath1), 0755)
				os.WriteFile(refPath1, []byte("ref: refs/heads/circular2\n"), 0644)
				os.WriteFile(refPath2, []byte("ref: refs/heads/circular1\n"), 0644)
				return "refs/heads/circular1"
			},
			wantErr: true,
			checkID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refName := tt.setup()
			
			id, err := refManager.readRefFile(refName)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("readRefFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkID && !tt.wantErr {
				if id.IsZero() {
					t.Error("readRefFile() returned zero ID for valid ref")
				}
			}
		})
	}
}

// Test UpdateRef function more comprehensively (currently 80% coverage)
func TestUpdateRefCoverage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-updateref-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	refManager := NewRefManager(tmpDir)
	testID := objects.NewBlob([]byte("test")).ID()

	tests := []struct {
		name    string
		refName string
		setup   func()
		wantErr bool
	}{
		{
			name:    "update new ref",
			refName: "refs/heads/new-branch",
			setup:   func() {},
			wantErr: false,
		},
		{
			name:    "update existing ref",
			refName: "refs/heads/existing",
			setup: func() {
				existingID := objects.NewBlob([]byte("existing")).ID()
				refManager.UpdateRef("refs/heads/existing", existingID)
			},
			wantErr: false,
		},
		{
			name:    "update with nested path",
			refName: "refs/heads/feature/long-branch-name",
			setup:   func() {},
			wantErr: false,
		},
		{
			name:    "invalid ref name",
			refName: "invalid..ref",
			setup:   func() {},
			wantErr: true,
		},
		{
			name:    "ref name with spaces",
			refName: "refs/heads/branch with spaces",
			setup:   func() {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			
			err := refManager.UpdateRef(tt.refName, testID)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If successful, verify the ref was actually updated
			if !tt.wantErr {
				readID, err := refManager.ResolveRef(tt.refName)
				if err != nil {
					t.Errorf("Failed to read back updated ref: %v", err)
				} else if readID != testID {
					t.Errorf("UpdateRef() stored ID = %v, want %v", readID, testID)
				}
			}
		})
	}
}

// Test listRefs function indirectly (currently 69.2% coverage)
func TestListRefsIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-list-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	refManager := NewRefManager(tmpDir)
	testID := objects.NewBlob([]byte("test")).ID()

	// Setup various refs to test listRefs indirectly through other functions
	refs := []string{
		"refs/heads/main",
		"refs/heads/feature/branch1", 
		"refs/tags/v1.0.0",
	}

	for _, ref := range refs {
		refManager.UpdateRef(ref, testID)
	}

	// Test that refs exist (this exercises listRefs internally)
	if !refManager.RefExists("main") {
		t.Error("RefExists should find main branch")
	}
	
	// Create some directories to test directory walking
	nestedDir := filepath.Join(tmpDir, "refs", "heads", "feature", "deep")
	os.MkdirAll(nestedDir, 0755)
	
	refManager.WriteRef("refs/heads/feature/deep/nested", testID, nil)
	
	if !refManager.RefExists("feature/deep/nested") {
		t.Error("RefExists should find nested branch")
	}
}

// Test CurrentBranch function more comprehensively (currently 75% coverage)
func TestCurrentBranchCoverage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-currentbranch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	refManager := NewRefManager(tmpDir)
	testID := objects.NewBlob([]byte("test")).ID()

	tests := []struct {
		name       string
		setup      func()
		wantBranch string
		wantErr    bool
	}{
		{
			name: "HEAD points to branch",
			setup: func() {
				// Create a branch
				refManager.CreateBranch("main", testID)
				// Set HEAD to point to it
				refManager.SetHEAD("refs/heads/main")
			},
			wantBranch: "main",
			wantErr:    false,
		},
		{
			name: "HEAD points to commit (detached)",
			setup: func() {
				refManager.SetHEADToCommit(testID)
			},
			wantBranch: "",
			wantErr:    true, // Detached HEAD
		},
		{
			name: "No HEAD file",
			setup: func() {
				// Remove HEAD if it exists
				headPath := filepath.Join(tmpDir, "HEAD")
				os.Remove(headPath)
			},
			wantBranch: "",
			wantErr:    true,
		},
		{
			name: "Invalid HEAD content",
			setup: func() {
				headPath := filepath.Join(tmpDir, "HEAD")
				os.WriteFile(headPath, []byte("invalid content"), 0644)
			},
			wantBranch: "",
			wantErr:    true,
		},
		{
			name: "HEAD points to non-existent branch",
			setup: func() {
				headPath := filepath.Join(tmpDir, "HEAD")
				os.WriteFile(headPath, []byte("ref: refs/heads/nonexistent"), 0644)
			},
			wantBranch: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			
			branch, err := refManager.CurrentBranch()
			
			if (err != nil) != tt.wantErr {
				t.Errorf("CurrentBranch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if branch != tt.wantBranch {
				t.Errorf("CurrentBranch() = %v, want %v", branch, tt.wantBranch)
			}
		})
	}
}

// Test WriteRef function more comprehensively (currently 80% coverage)
func TestWriteRefCoverage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-writeref-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	refManager := NewRefManager(tmpDir)
	testID := objects.NewBlob([]byte("test")).ID()

	tests := []struct {
		name     string
		refPath  string
		content  string
		setup    func()
		wantErr  bool
		checkErr func(error) bool
	}{
		{
			name:    "write new ref",
			refPath: "refs/heads/new",
			content: testID.String(),
			setup:   func() {},
			wantErr: false,
		},
		{
			name:    "overwrite existing ref",
			refPath: "refs/heads/existing",
			content: testID.String(),
			setup: func() {
				// Create the ref first
				existingID := objects.NewBlob([]byte("existing")).ID()
				refManager.WriteRef("refs/heads/existing", existingID, nil)
			},
			wantErr: false,
		},
		{
			name:    "write ref with nested directory",
			refPath: "refs/heads/feature/deep/nested",
			content: testID.String(),
			setup:   func() {},
			wantErr: false,
		},
		{
			name:    "write to read-only directory",
			refPath: "refs/readonly/test",
			content: testID.String(),
			setup: func() {
				// Create read-only directory
				readOnlyDir := filepath.Join(tmpDir, "refs", "readonly")
				os.MkdirAll(readOnlyDir, 0444)
			},
			wantErr: true,
		},
		{
			name:    "write symbolic ref",
			refPath: "refs/heads/symbolic",
			content: "ref: refs/heads/main",
			setup: func() {
				// Create target ref
				refManager.CreateBranch("main", testID)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			
			// For the WriteRef test, we need to provide ObjectID, not string
			var id objects.ObjectID
			if strings.HasPrefix(tt.content, "ref:") {
				// This is a symbolic ref, WriteRef doesn't handle these
				// Use the lower-level file operations instead
				refFile := filepath.Join(tmpDir, tt.refPath)
				os.MkdirAll(filepath.Dir(refFile), 0755)
				err := os.WriteFile(refFile, []byte(tt.content), 0644)
				if (err != nil) != tt.wantErr {
					t.Errorf("WriteFile() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			} else {
				var parseErr error
				id, parseErr = objects.NewObjectID(tt.content)
				if parseErr != nil {
					t.Fatalf("Invalid test object ID: %v", parseErr)
				}
			}
			err := refManager.WriteRef(tt.refPath, id, nil)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If successful, verify the ref was written correctly
			if !tt.wantErr {
				refFile := filepath.Join(tmpDir, tt.refPath)
				content, err := os.ReadFile(refFile)
				if err != nil {
					t.Errorf("Failed to read back written ref: %v", err)
				} else {
					written := strings.TrimSpace(string(content))
					if written != tt.content {
						t.Errorf("WriteRef() wrote %q, want %q", written, tt.content)
					}
				}
			}

			// Clean up read-only directories for next test
			if strings.Contains(tt.refPath, "readonly") {
				readOnlyDir := filepath.Join(tmpDir, "refs", "readonly")
				os.Chmod(readOnlyDir, 0755)
			}
		})
	}
}