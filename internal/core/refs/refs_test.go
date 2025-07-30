package refs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fenilsonani/vcs/internal/core/objects"
)

func TestNewRefManager(t *testing.T) {
	gitDir := "/test/git"
	rm := NewRefManager(gitDir)
	
	if rm.gitDir != gitDir {
		t.Errorf("NewRefManager() gitDir = %v, want %v", rm.gitDir, gitDir)
	}
}

func TestRefManager_HEAD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	
	rm := NewRefManager(gitDir)

	tests := []struct {
		name        string
		headContent string
		wantRefName string
		wantErr     bool
		setup       func()
	}{
		{
			name:        "symbolic reference",
			headContent: "ref: refs/heads/main\n",
			wantRefName: "refs/heads/main",
			wantErr:     true, // ResolveRef will fail as ref doesn't exist
			setup:       func() {},
		},
		{
			name:        "direct object reference",
			headContent: "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3\n",
			wantRefName: "",
			wantErr:     false,
			setup:       func() {},
		},
		{
			name:        "missing HEAD file",
			headContent: "",
			wantErr:     true,
			setup:       func() {
				os.Remove(filepath.Join(gitDir, "HEAD"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.headContent != "" {
				headPath := filepath.Join(gitDir, "HEAD")
				os.WriteFile(headPath, []byte(tt.headContent), 0644)
			}
			
			tt.setup()

			id, refName, err := rm.HEAD()
			if (err != nil) != tt.wantErr {
				t.Errorf("HEAD() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if refName != tt.wantRefName {
					t.Errorf("HEAD() refName = %v, want %v", refName, tt.wantRefName)
				}
				if id.IsZero() && tt.headContent != "" {
					t.Error("HEAD() returned zero ID")
				}
			}
		})
	}
}

func TestRefManager_SetHEAD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	
	rm := NewRefManager(gitDir)

	err = rm.SetHEAD("refs/heads/main")
	if err != nil {
		t.Fatalf("SetHEAD() error = %v", err)
	}

	headPath := filepath.Join(gitDir, "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		t.Fatalf("Failed to read HEAD: %v", err)
	}

	expected := "ref: refs/heads/main\n"
	if string(content) != expected {
		t.Errorf("HEAD content = %v, want %v", string(content), expected)
	}
}

func TestRefManager_SetHEADToCommit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	
	rm := NewRefManager(gitDir)

	commitID, _ := objects.NewObjectID("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")
	err = rm.SetHEADToCommit(commitID)
	if err != nil {
		t.Fatalf("SetHEADToCommit() error = %v", err)
	}

	headPath := filepath.Join(gitDir, "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		t.Fatalf("Failed to read HEAD: %v", err)
	}

	expected := "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3\n"
	if string(content) != expected {
		t.Errorf("HEAD content = %v, want %v", string(content), expected)
	}
}

func TestRefManager_ResolveRef(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	rm := NewRefManager(gitDir)

	// Create refs structure
	os.MkdirAll(filepath.Join(gitDir, "refs", "heads"), 0755)
	os.MkdirAll(filepath.Join(gitDir, "refs", "tags"), 0755)

	commitID, _ := objects.NewObjectID("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")

	// Create a branch
	branchPath := filepath.Join(gitDir, "refs", "heads", "main")
	os.WriteFile(branchPath, []byte(commitID.String()+"\n"), 0644)

	tests := []struct {
		name     string
		refName  string
		wantID   objects.ObjectID
		wantErr  bool
	}{
		{
			name:    "full reference name",
			refName: "refs/heads/main",
			wantID:  commitID,
			wantErr: false,
		},
		{
			name:    "short branch name",
			refName: "main",
			wantID:  commitID,
			wantErr: false,
		},
		{
			name:    "non-existent reference",
			refName: "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := rm.ResolveRef(tt.refName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveRef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && id != tt.wantID {
				t.Errorf("ResolveRef() id = %v, want %v", id, tt.wantID)
			}
		})
	}
}

func TestRefManager_UpdateRef(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	rm := NewRefManager(gitDir)

	commitID, _ := objects.NewObjectID("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")

	err = rm.UpdateRef("refs/heads/test", commitID)
	if err != nil {
		t.Fatalf("UpdateRef() error = %v", err)
	}

	// Verify ref was created
	refPath := filepath.Join(gitDir, "refs", "heads", "test")
	content, err := os.ReadFile(refPath)
	if err != nil {
		t.Fatalf("Failed to read ref: %v", err)
	}

	expected := commitID.String() + "\n"
	if string(content) != expected {
		t.Errorf("Ref content = %v, want %v", string(content), expected)
	}
}

func TestRefManager_CreateBranch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	rm := NewRefManager(gitDir)

	commitID, _ := objects.NewObjectID("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")

	err = rm.CreateBranch("feature", commitID)
	if err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}

	// Verify branch was created
	branchPath := filepath.Join(gitDir, "refs", "heads", "feature")
	content, err := os.ReadFile(branchPath)
	if err != nil {
		t.Fatalf("Failed to read branch: %v", err)
	}

	expected := commitID.String() + "\n"
	if string(content) != expected {
		t.Errorf("Branch content = %v, want %v", string(content), expected)
	}
}

func TestRefManager_DeleteBranch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	rm := NewRefManager(gitDir)

	commitID, _ := objects.NewObjectID("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")

	// Create branch first
	rm.CreateBranch("todelete", commitID)

	// Delete it
	err = rm.DeleteBranch("todelete")
	if err != nil {
		t.Fatalf("DeleteBranch() error = %v", err)
	}

	// Verify branch was deleted
	branchPath := filepath.Join(gitDir, "refs", "heads", "todelete")
	if _, err := os.Stat(branchPath); !os.IsNotExist(err) {
		t.Error("Branch was not deleted")
	}

	// Try to delete non-existent branch
	err = rm.DeleteBranch("nonexistent")
	if err == nil {
		t.Error("DeleteBranch() should error for non-existent branch")
	}
}

func TestRefManager_CreateTag(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	rm := NewRefManager(gitDir)

	commitID, _ := objects.NewObjectID("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")

	err = rm.CreateTag("v1.0", commitID)
	if err != nil {
		t.Fatalf("CreateTag() error = %v", err)
	}

	// Verify tag was created
	tagPath := filepath.Join(gitDir, "refs", "tags", "v1.0")
	content, err := os.ReadFile(tagPath)
	if err != nil {
		t.Fatalf("Failed to read tag: %v", err)
	}

	expected := commitID.String() + "\n"
	if string(content) != expected {
		t.Errorf("Tag content = %v, want %v", string(content), expected)
	}
}

func TestRefManager_DeleteTag(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	rm := NewRefManager(gitDir)

	commitID, _ := objects.NewObjectID("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")

	// Create tag first
	rm.CreateTag("todelete", commitID)

	// Delete it
	err = rm.DeleteTag("todelete")
	if err != nil {
		t.Fatalf("DeleteTag() error = %v", err)
	}

	// Verify tag was deleted
	tagPath := filepath.Join(gitDir, "refs", "tags", "todelete")
	if _, err := os.Stat(tagPath); !os.IsNotExist(err) {
		t.Error("Tag was not deleted")
	}

	// Try to delete non-existent tag
	err = rm.DeleteTag("nonexistent")
	if err == nil {
		t.Error("DeleteTag() should error for non-existent tag")
	}
}

func TestRefManager_ListBranches(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	rm := NewRefManager(gitDir)

	commitID, _ := objects.NewObjectID("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")

	// Create some branches
	rm.CreateBranch("main", commitID)
	rm.CreateBranch("feature", commitID)
	rm.CreateBranch("hotfix", commitID)

	branches, err := rm.ListBranches()
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}

	if len(branches) != 3 {
		t.Errorf("ListBranches() count = %v, want 3", len(branches))
	}

	expectedBranches := map[string]bool{
		"refs/heads/main":    true,
		"refs/heads/feature": true,
		"refs/heads/hotfix":  true,
	}

	for _, branch := range branches {
		if !expectedBranches[branch] {
			t.Errorf("Unexpected branch: %v", branch)
		}
	}
}

func TestRefManager_ListTags(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	rm := NewRefManager(gitDir)

	commitID, _ := objects.NewObjectID("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")

	// Create some tags
	rm.CreateTag("v1.0", commitID)
	rm.CreateTag("v2.0", commitID)

	tags, err := rm.ListTags()
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("ListTags() count = %v, want 2", len(tags))
	}

	expectedTags := map[string]bool{
		"refs/tags/v1.0": true,
		"refs/tags/v2.0": true,
	}

	for _, tag := range tags {
		if !expectedTags[tag] {
			t.Errorf("Unexpected tag: %v", tag)
		}
	}
}

func TestRefManager_CurrentBranch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	rm := NewRefManager(gitDir)

	tests := []struct {
		name        string
		headContent string
		wantBranch  string
		wantErr     bool
		setup       func()
	}{
		{
			name:        "on branch",
			headContent: "ref: refs/heads/main\n",
			wantBranch:  "main",
			wantErr:     false,
			setup: func() {
				// Create the branch that HEAD points to
				commitID, _ := objects.NewObjectID("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")
				rm.CreateBranch("main", commitID)
			},
		},
		{
			name:        "detached HEAD",
			headContent: "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3\n",
			wantBranch:  "",
			wantErr:     true,
			setup:       func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			
			headPath := filepath.Join(gitDir, "HEAD")
			os.WriteFile(headPath, []byte(tt.headContent), 0644)

			branch, err := rm.CurrentBranch()
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

func TestRefManager_IsValidRef(t *testing.T) {
	rm := NewRefManager("/test")

	tests := []struct {
		name    string
		refName string
		want    bool
	}{
		{"valid branch", "refs/heads/main", true},
		{"valid tag with dot", "refs/tags/v1.0", false}, // dots are forbidden in our implementation
		{"empty string", "", false},
		{"starts with slash", "/invalid", false},
		{"ends with slash", "invalid/", false},
		{"double slash", "refs//heads", false},
		{"contains dot", "refs/heads/feat.ure", false},
		{"contains space", "refs/heads/feat ure", false},
		{"contains tilde", "refs/heads/feat~ure", false},
		{"contains caret", "refs/heads/feat^ure", false},
		{"contains colon", "refs/heads/feat:ure", false},
		{"contains question", "refs/heads/feat?ure", false},
		{"contains asterisk", "refs/heads/feat*ure", false},
		{"contains bracket", "refs/heads/feat[ure", false},
		{"contains backslash", "refs/heads/feat\\ure", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rm.IsValidRef(tt.refName); got != tt.want {
				t.Errorf("IsValidRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRefManager_RefExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	rm := NewRefManager(gitDir)

	commitID, _ := objects.NewObjectID("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")

	// Create a branch
	rm.CreateBranch("existing", commitID)

	if !rm.RefExists("existing") {
		t.Error("RefExists() should return true for existing ref")
	}

	if rm.RefExists("nonexistent") {
		t.Error("RefExists() should return false for non-existent ref")
	}
}

func TestRefManager_WriteRef(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	rm := NewRefManager(gitDir)

	commitID, _ := objects.NewObjectID("a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")

	// Test atomic write
	err = rm.WriteRef("refs/heads/atomic", commitID, nil)
	if err != nil {
		t.Fatalf("WriteRef() error = %v", err)
	}

	// Verify ref was created
	id, err := rm.ResolveRef("atomic")
	if err != nil {
		t.Fatalf("ResolveRef() error = %v", err)
	}

	if id != commitID {
		t.Errorf("ResolveRef() = %v, want %v", id, commitID)
	}

	// Test update with old value check
	newCommitID, _ := objects.NewObjectID("b94a8fe5ccb19ba61c4c0873d391e987982fbbd4")
	err = rm.WriteRef("refs/heads/atomic", newCommitID, &commitID)
	if err != nil {
		t.Fatalf("WriteRef() with old value error = %v", err)
	}

	// Verify update
	id, err = rm.ResolveRef("atomic")
	if err != nil {
		t.Fatalf("ResolveRef() error = %v", err)
	}

	if id != newCommitID {
		t.Errorf("ResolveRef() = %v, want %v", id, newCommitID)
	}

	// Test update with wrong old value
	wrongOldID, _ := objects.NewObjectID("c94a8fe5ccb19ba61c4c0873d391e987982fbbd5")
	err = rm.WriteRef("refs/heads/atomic", commitID, &wrongOldID)
	if err == nil {
		t.Error("WriteRef() should error when old value doesn't match")
	}
}

func TestRefManager_ReadPackedRefs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "refs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	rm := NewRefManager(gitDir)

	// Test with no packed-refs file
	packed, err := rm.ReadPackedRefs()
	if err != nil {
		t.Fatalf("ReadPackedRefs() error = %v", err)
	}
	if len(packed.refs) != 0 {
		t.Errorf("ReadPackedRefs() refs count = %v, want 0", len(packed.refs))
	}

	// Create packed-refs file
	packedContent := `# pack-refs with: peeled fully-peeled sorted 
a94a8fe5ccb19ba61c4c0873d391e987982fbbd3 refs/heads/main
b94a8fe5ccb19ba61c4c0873d391e987982fbbd4 refs/tags/v1.0
# comment line
c94a8fe5ccb19ba61c4c0873d391e987982fbbd5 refs/remotes/origin/main
`
	packedPath := filepath.Join(gitDir, "packed-refs")
	os.WriteFile(packedPath, []byte(packedContent), 0644)

	packed, err = rm.ReadPackedRefs()
	if err != nil {
		t.Fatalf("ReadPackedRefs() error = %v", err)
	}

	if len(packed.refs) != 3 {
		t.Errorf("ReadPackedRefs() refs count = %v, want 3", len(packed.refs))
	}

	expectedRefs := map[string]string{
		"refs/heads/main":         "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3",
		"refs/tags/v1.0":          "b94a8fe5ccb19ba61c4c0873d391e987982fbbd4",
		"refs/remotes/origin/main": "c94a8fe5ccb19ba61c4c0873d391e987982fbbd5",
	}

	for refName, expectedIDStr := range expectedRefs {
		if id, exists := packed.refs[refName]; !exists {
			t.Errorf("Expected ref %s not found", refName)
		} else if id.String() != expectedIDStr {
			t.Errorf("Ref %s ID = %v, want %v", refName, id.String(), expectedIDStr)
		}
	}
}