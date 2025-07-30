package objects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStorage_ErrorConditions(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vcs-storage-error-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)
	if err := storage.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test WriteObject with read-only directory
	t.Run("WriteObject with read-only objects dir", func(t *testing.T) {
		objectsDir := filepath.Join(tmpDir, "objects")
		if err := os.Chmod(objectsDir, 0444); err != nil {
			t.Skip("Could not make objects directory read-only")
		}
		defer os.Chmod(objectsDir, 0755) // Restore permissions

		blob := NewBlob([]byte("test data"))
		err := storage.WriteObject(blob)
		if err == nil {
			t.Error("WriteObject() should fail with read-only objects directory")
		}
	})

	// Test ReadObject with corrupted object file
	t.Run("ReadObject with corrupted object", func(t *testing.T) {
		// Create a blob first
		blob := NewBlob([]byte("test data"))
		if err := storage.WriteObject(blob); err != nil {
			t.Skip("Could not write object for corruption test")
		}

		// Try to make the objects directory writable first
		objectDir := filepath.Join(tmpDir, "objects", blob.ID().String()[:2])
		if err := os.Chmod(objectDir, 0755); err != nil {
			t.Skip("Could not change directory permissions")
		}

		// Corrupt the object file
		objectPath := filepath.Join(objectDir, blob.ID().String()[2:])
		corruptData := []byte("corrupted data")
		if err := os.WriteFile(objectPath, corruptData, 0644); err != nil {
			t.Skip("Could not corrupt object file")
		}

		// Try to read the corrupted object
		_, err := storage.ReadObject(blob.ID())
		if err == nil {
			t.Error("ReadObject() should fail with corrupted object")
		}
	})

	// Test with invalid object directory structure
	t.Run("ReadObject with missing subdirectory", func(t *testing.T) {
		fakeID := ComputeHash(TypeBlob, []byte("nonexistent"))
		_, err := storage.ReadObject(fakeID)
		if err == nil {
			t.Error("ReadObject() should fail for nonexistent object")
		}
	})
}

func TestTree_ErrorConditions(t *testing.T) {
	// Test ParseTree with malformed data
	t.Run("ParseTree with no space separator", func(t *testing.T) {
		data := []byte("100644filename\x00" + string(make([]byte, 20)))
		_, err := ParseTree(ObjectID{}, data)
		if err == nil {
			t.Error("ParseTree() should fail with no space separator")
		}
		if !strings.Contains(err.Error(), "no space found") {
			t.Errorf("ParseTree() error = %v, want error containing 'no space found'", err)
		}
	})

	// Test AddEntry with very long name
	t.Run("AddEntry with very long name", func(t *testing.T) {
		tree := NewTree()
		longName := strings.Repeat("a", 1000) + ".txt"
		err := tree.AddEntry(ModeBlob, longName, ObjectID{1})
		if err != nil {
			t.Errorf("AddEntry() with long name should succeed, got error: %v", err)
		}
	})

	// Test tree serialization
	t.Run("Tree serialization", func(t *testing.T) {
		tree := NewTree()
		tree.AddEntry(ModeBlob, "file.txt", ObjectID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
		
		serialized, err := tree.Serialize()
		if err != nil {
			t.Fatalf("Serialize() error = %v", err)
		}
		if len(serialized) == 0 {
			t.Error("Serialize() returned empty data")
		}
	})
}

func TestCommit_ErrorConditions(t *testing.T) {
	// Test commit with many parents
	t.Run("Commit with many parents", func(t *testing.T) {
		treeID := ObjectID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
		parents := make([]ObjectID, 10)
		for i := range parents {
			parents[i] = ObjectID{byte(i + 1)}
		}
		
		author := Signature{Name: "Author", Email: "author@example.com", When: time.Now()}
		committer := Signature{Name: "Committer", Email: "committer@example.com", When: time.Now()}
		
		commit := NewCommit(treeID, parents, author, committer, "Commit with many parents")
		if commit == nil {
			t.Fatal("NewCommit returned nil")
		}
		
		if len(commit.Parents()) != 10 {
			t.Errorf("Commit parents = %d, want 10", len(commit.Parents()))
		}
	})

	// Test commit serialization
	t.Run("Commit serialization", func(t *testing.T) {
		treeID := ObjectID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
		author := Signature{Name: "Author", Email: "author@example.com", When: time.Now()}
		committer := Signature{Name: "Committer", Email: "committer@example.com", When: time.Now()}
		
		commit := NewCommit(treeID, nil, author, committer, "Test commit")
		serialized, err := commit.Serialize()
		if err != nil {
			t.Fatalf("Serialize() error = %v", err)
		}
		if len(serialized) == 0 {
			t.Error("Serialize() returned empty data")
		}
	})
}

func TestTag_ErrorConditions(t *testing.T) {
	// Test tag serialization
	t.Run("Tag serialization", func(t *testing.T) {
		objectID := ObjectID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
		tagger := Signature{Name: "Tagger", Email: "tagger@example.com", When: time.Now()}
		
		tag := NewTag(objectID, TypeCommit, "v1.0.0", tagger, "Release tag")
		serialized, err := tag.Serialize()
		if err != nil {
			t.Fatalf("Serialize() error = %v", err)
		}
		if len(serialized) == 0 {
			t.Error("Serialize() returned empty data")
		}
	})

	// Test tag with empty tagger name
	t.Run("Tag with empty tagger", func(t *testing.T) {
		objectID := ObjectID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
		tagger := Signature{Name: "", Email: "", When: time.Time{}}
		
		tag := NewTag(objectID, TypeCommit, "v1.0.0", tagger, "Tag with empty tagger")
		if tag == nil {
			t.Fatal("NewTag returned nil")
		}
		
		if tag.Tagger().Name != "" {
			t.Errorf("Tag tagger name = %q, want empty", tag.Tagger().Name)
		}
	})
}

func TestBlob_ErrorConditions(t *testing.T) {
	// Test blob with nil data
	t.Run("Blob with nil data", func(t *testing.T) {
		blob := NewBlob(nil)
		if blob == nil {
			t.Fatal("NewBlob returned nil for nil data")
		}
		if len(blob.Data()) != 0 {
			t.Errorf("Blob data length = %d, want 0", len(blob.Data()))
		}
	})

	// Test very large blob
	t.Run("Very large blob", func(t *testing.T) {
		largeData := make([]byte, 1024*1024) // 1MB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		
		blob := NewBlob(largeData)
		if blob == nil {
			t.Fatal("NewBlob returned nil for large data")
		}
		if blob.Size() != int64(len(largeData)) {
			t.Errorf("Blob size = %d, want %d", blob.Size(), len(largeData))
		}
	})
}

func TestNewObjectID_ErrorConditions(t *testing.T) {
	// Test with invalid hex strings
	tests := []struct {
		name string
		hex  string
		wantErr bool
	}{
		{"valid hex", "1234567890abcdef1234567890abcdef12345678", false},
		{"too short", "1234567890abcdef", true},
		{"too long", "1234567890abcdef1234567890abcdef123456789a", true},
		{"invalid characters", "1234567890abcdefg234567890abcdef12345678", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewObjectID(tt.hex)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewObjectID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && id.IsZero() {
				t.Error("NewObjectID() returned zero ID for valid input")
			}
		})
	}
}

func TestSignature_String_ErrorConditions(t *testing.T) {
	// Test signature string formatting
	tests := []struct {
		name string
		sig  Signature
	}{
		{
			name: "normal signature",
			sig:  Signature{Name: "Test User", Email: "test@example.com", When: time.Unix(1234567890, 0)},
		},
		{
			name: "signature with special characters in name",
			sig:  Signature{Name: "Test <User>", Email: "test@example.com", When: time.Unix(1234567890, 0)},
		},
		{
			name: "signature with very long name",
			sig:  Signature{Name: strings.Repeat("Test User ", 100), Email: "test@example.com", When: time.Unix(1234567890, 0)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.sig.String()
			if len(str) == 0 {
				t.Error("String() returned empty string")
			}
			// Should contain the name (unless empty)
			if tt.sig.Name != "" && !strings.Contains(str, tt.sig.Name) {
				t.Errorf("String() = %q, should contain name %q", str, tt.sig.Name)
			}
		})
	}
}

func TestParseCommit_ErrorConditions(t *testing.T) {
	// Test with malformed commit data that actually causes errors
	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{
			name:    "invalid tree hash format",
			data:    []byte("tree invalid\n"),
			wantErr: "invalid tree ID",
		},
		{
			name:    "invalid parent hash format", 
			data:    []byte("tree 1234567890abcdef1234567890abcdef12345678\nparent invalid\n"),
			wantErr: "invalid parent ID",
		},
		{
			name:    "invalid author format",
			data:    []byte("tree 1234567890abcdef1234567890abcdef12345678\nauthor invalid-format\n"),
			wantErr: "invalid author",
		},
		{
			name:    "invalid committer format",
			data:    []byte("tree 1234567890abcdef1234567890abcdef12345678\nauthor Test <test@example.com> 1234567890 +0000\ncommitter invalid-format\n"),
			wantErr: "invalid committer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCommit(ObjectID{}, tt.data)
			if err == nil {
				t.Errorf("ParseCommit() error = nil, want error containing %q", tt.wantErr)
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ParseCommit() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}