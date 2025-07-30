package objects

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBlob_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty blob", []byte{}},
		{"single byte", []byte{0}},
		{"large blob", make([]byte, 100000)},
		{"binary data", []byte{0, 1, 2, 255, 254, 253}},
		{"unicode text", []byte("Hello 世界 🌍")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob := NewBlob(tt.data)
			if blob == nil {
				t.Fatal("NewBlob returned nil")
			}

			if !bytes.Equal(blob.Data(), tt.data) {
				t.Errorf("Data() = %v, want %v", blob.Data(), tt.data)
			}

			if blob.Type() != TypeBlob {
				t.Errorf("Type() = %v, want %v", blob.Type(), TypeBlob)
			}

			if blob.Size() != int64(len(tt.data)) {
				t.Errorf("Size() = %v, want %v", blob.Size(), len(tt.data))
			}

			// Test serialization
			serialized, err := blob.Serialize()
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}
			if len(serialized) != len(tt.data) {
				t.Errorf("Serialize() length = %v, want %v", len(serialized), len(tt.data))
			}

			// Test ID computation
			id := blob.ID()
			if id.IsZero() {
				t.Error("ID() returned zero ID")
			}
		})
	}
}

func TestTree_EdgeCases(t *testing.T) {
	tree := NewTree()

	// Test empty tree
	if len(tree.Entries()) != 0 {
		t.Errorf("Empty tree entries = %v, want 0", len(tree.Entries()))
	}

	if tree.Type() != TypeTree {
		t.Errorf("Type() = %v, want %v", tree.Type(), TypeTree)
	}

	// Test adding entries with various names
	testCases := []struct {
		name     string
		mode     FileMode
		id       ObjectID
		wantErr  bool
		errContains string
	}{
		{"normal.txt", ModeBlob, ObjectID{1}, false, ""},
		{"", ModeBlob, ObjectID{2}, true, "entry name cannot be empty"},
		{"with/slash", ModeBlob, ObjectID{3}, false, ""}, // Git allows path separators in names
		{"with space", ModeBlob, ObjectID{4}, false, ""}, // Git allows spaces in names
		{"with\ttab", ModeBlob, ObjectID{5}, false, ""}, // Git allows tabs in names
		{"with\nnewline", ModeBlob, ObjectID{6}, false, ""}, // Git allows newlines in names
		{"normal-exec.sh", ModeExec, ObjectID{7}, false, ""},
		{"subdir", ModeTree, ObjectID{8}, false, ""},
		{"duplicate.txt", ModeBlob, ObjectID{9}, false, ""}, // First occurrence
		{"duplicate.txt", ModeBlob, ObjectID{10}, true, "duplicate entry name"}, // Duplicate should fail
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tree.AddEntry(tc.mode, tc.name, tc.id)
			if tc.wantErr {
				if err == nil {
					t.Errorf("AddEntry() error = nil, want error")
				} else if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("AddEntry() error = %v, want error containing %q", err, tc.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("AddEntry() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestCommit_EdgeCases(t *testing.T) {
	treeID := ObjectID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	now := time.Now()

	tests := []struct {
		name     string
		parents  []ObjectID
		author   Signature
		committer Signature
		message  string
	}{
		{
			name:    "root commit",
			parents: nil,
			author:  Signature{Name: "Author", Email: "author@example.com", When: now},
			committer: Signature{Name: "Committer", Email: "committer@example.com", When: now},
			message: "Initial commit",
		},
		{
			name:    "merge commit",
			parents: []ObjectID{{1}, {2}},
			author:  Signature{Name: "Author", Email: "author@example.com", When: now},
			committer: Signature{Name: "Committer", Email: "committer@example.com", When: now},
			message: "Merge branch 'feature'",
		},
		{
			name:    "empty message",
			parents: []ObjectID{{1}},
			author:  Signature{Name: "Author", Email: "author@example.com", When: now},
			committer: Signature{Name: "Committer", Email: "committer@example.com", When: now},
			message: "",
		},
		{
			name:    "multiline message",
			parents: []ObjectID{{1}},
			author:  Signature{Name: "Author", Email: "author@example.com", When: now},
			committer: Signature{Name: "Committer", Email: "committer@example.com", When: now},
			message: "Short summary\n\nLonger description\nwith multiple lines\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commit := NewCommit(treeID, tt.parents, tt.author, tt.committer, tt.message)
			if commit == nil {
				t.Fatal("NewCommit returned nil")
			}

			if commit.Type() != TypeCommit {
				t.Errorf("Type() = %v, want %v", commit.Type(), TypeCommit)
			}

			if commit.Tree() != treeID {
				t.Errorf("Tree() = %v, want %v", commit.Tree(), treeID)
			}

			if len(commit.Parents()) != len(tt.parents) {
				t.Errorf("Parents() length = %v, want %v", len(commit.Parents()), len(tt.parents))
			}

			if commit.Message() != tt.message {
				t.Errorf("Message() = %q, want %q", commit.Message(), tt.message)
			}

			// Test serialization
			serialized, err := commit.Serialize()
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}
			if len(serialized) == 0 {
				t.Error("Serialize() returned empty data")
			}

			// Verify serialized content contains expected data
			serializedStr := string(serialized)
			if !strings.Contains(serializedStr, "tree "+treeID.String()) {
				t.Error("Serialized commit missing tree reference")
			}
			if !strings.Contains(serializedStr, "author "+tt.author.Name) {
				t.Error("Serialized commit missing author")
			}
			if !strings.Contains(serializedStr, "committer "+tt.committer.Name) {
				t.Error("Serialized commit missing committer")
			}
		})
	}
}

func TestTag_EdgeCases(t *testing.T) {
	objectID := ObjectID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	now := time.Now()
	tagger := Signature{Name: "Tagger", Email: "tagger@example.com", When: now}

	tests := []struct {
		name       string
		objType    ObjectType
		tagName    string
		message    string
	}{
		{"simple tag", TypeCommit, "v1.0.0", "Release version 1.0.0"},
		{"blob tag", TypeBlob, "important-file", "Important file tag"},
		{"tree tag", TypeTree, "snapshot", "Tree snapshot"},
		{"empty message", TypeCommit, "v1.0.1", ""},
		{"multiline message", TypeCommit, "v2.0.0", "Version 2.0.0\n\n* Major refactor\n* Breaking changes\n* New features"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := NewTag(objectID, tt.objType, tt.tagName, tagger, tt.message)
			if tag == nil {
				t.Fatal("NewTag returned nil")
			}

			if tag.Type() != TypeTag {
				t.Errorf("Type() = %v, want %v", tag.Type(), TypeTag)
			}

			if tag.Object() != objectID {
				t.Errorf("Object() = %v, want %v", tag.Object(), objectID)
			}

			if tag.ObjectType() != tt.objType {
				t.Errorf("ObjectType() = %v, want %v", tag.ObjectType(), tt.objType)
			}

			if tag.TagName() != tt.tagName {
				t.Errorf("TagName() = %q, want %q", tag.TagName(), tt.tagName)
			}

			if tag.Message() != tt.message {
				t.Errorf("Message() = %q, want %q", tag.Message(), tt.message)
			}

			// Test serialization
			serialized, err := tag.Serialize()
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}
			if len(serialized) == 0 {
				t.Error("Serialize() returned empty data")
			}

			serializedStr := string(serialized)
			if !strings.Contains(serializedStr, "object "+objectID.String()) {
				t.Error("Serialized tag missing object reference")
			}
			if !strings.Contains(serializedStr, "type "+string(tt.objType)) {
				t.Error("Serialized tag missing type")
			}
			if !strings.Contains(serializedStr, "tag "+tt.tagName) {
				t.Error("Serialized tag missing tag name")
			}
		})
	}
}

func TestStorage_EdgeCases(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vcs-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)
	if err := storage.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Test with various object types and sizes
	testObjects := []Object{
		NewBlob([]byte{}),                    // Empty blob
		NewBlob([]byte("small")),             // Small blob
		NewBlob(make([]byte, 10000)),         // Large blob
		NewTree(),                            // Empty tree
		NewCommit(ObjectID{1}, nil, Signature{Name: "Test", Email: "test@example.com", When: time.Now()}, Signature{Name: "Test", Email: "test@example.com", When: time.Now()}, "Test"),
		NewTag(ObjectID{1}, TypeCommit, "test", Signature{Name: "Test", Email: "test@example.com", When: time.Now()}, "Test tag"),
	}

	for i, obj := range testObjects {
		t.Run(string(obj.Type())+"_"+string(rune('0'+i)), func(t *testing.T) {
			// Write object
			if err := storage.WriteObject(obj); err != nil {
				t.Fatalf("WriteObject() error = %v", err)
			}

			// Check existence
			if !storage.HasObject(obj.ID()) {
				t.Error("HasObject() returned false after WriteObject()")
			}

			// Read back
			readObj, err := storage.ReadObject(obj.ID())
			if err != nil {
				t.Fatalf("ReadObject() error = %v", err)
			}

			if readObj.Type() != obj.Type() {
				t.Errorf("ReadObject() type = %v, want %v", readObj.Type(), obj.Type())
			}

			if readObj.ID() != obj.ID() {
				t.Errorf("ReadObject() ID = %v, want %v", readObj.ID(), obj.ID())
			}

			// Compare serialized data
			readSerialized, err := readObj.Serialize()
			if err != nil {
				t.Fatalf("readObj.Serialize() error = %v", err)
			}
			objSerialized, err := obj.Serialize()
			if err != nil {
				t.Fatalf("obj.Serialize() error = %v", err)
			}
			if !bytes.Equal(readSerialized, objSerialized) {
				t.Error("ReadObject() returned different serialized data")
			}
		})
	}

	// Test reading non-existent object
	fakeID := ComputeHash(TypeBlob, []byte("nonexistent"))
	_, err = storage.ReadObject(fakeID)
	if err == nil {
		t.Error("ReadObject() should return error for non-existent object")
	}

	// Test HasObject for non-existent object
	if storage.HasObject(fakeID) {
		t.Error("HasObject() should return false for non-existent object")
	}
}

func TestStorage_InitTwice(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vcs-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)

	// First init should succeed
	if err := storage.Init(); err != nil {
		t.Fatalf("First Init() error = %v", err)
	}

	// Second init should also succeed (idempotent)
	if err := storage.Init(); err != nil {
		t.Fatalf("Second Init() error = %v", err)
	}

	// Verify directories still exist
	objectsDir := filepath.Join(tmpDir, "objects")
	if _, err := os.Stat(objectsDir); err != nil {
		t.Errorf("Objects directory missing after second init: %v", err)
	}
}

func TestSignature_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		sig  Signature
	}{
		{
			name: "empty name",
			sig:  Signature{Name: "", Email: "test@example.com", When: time.Now()},
		},
		{
			name: "empty email",
			sig:  Signature{Name: "Test User", Email: "", When: time.Now()},
		},
		{
			name: "special characters",
			sig:  Signature{Name: "Test User <with> brackets", Email: "test+tag@example.co.uk", When: time.Now()},
		},
		{
			name: "unicode name",
			sig:  Signature{Name: "测试用户", Email: "test@example.com", When: time.Now()},
		},
		{
			name: "zero time",
			sig:  Signature{Name: "Test User", Email: "test@example.com", When: time.Time{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test string representation
			str := tt.sig.String()
			if len(str) == 0 {
				t.Error("String() returned empty string")
			}

			// Should contain name and email (even if empty)
			// Should contain timestamp
			if !strings.Contains(str, tt.sig.Name) && tt.sig.Name != "" {
				t.Errorf("String() missing name: %q", str)
			}
			if !strings.Contains(str, tt.sig.Email) && tt.sig.Email != "" {
				t.Errorf("String() missing email: %q", str)
			}
		})
	}
}

func TestParseTree_InvalidData(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: "", // Empty tree is valid
		},
		{
			name:    "incomplete entry",
			data:    []byte("100644 file"),
			wantErr: "no null byte found",
		},
		{
			name:    "invalid mode",
			data:    []byte("invalid filename\x00" + string(make([]byte, 20))),
			wantErr: "invalid file mode",
		},
		{
			name:    "missing null separator",
			data:    []byte("100644 filename"),
			wantErr: "no null byte found",
		},
		{
			name:    "truncated hash",
			data:    []byte("100644 filename\x00" + string(make([]byte, 10))),
			wantErr: "insufficient data for hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTree(ObjectID{}, tt.data)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ParseTree() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ParseTree() error = nil, want error containing %q", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ParseTree() error = %v, want error containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestObjectID_EdgeCases(t *testing.T) {
	// Test zero ID
	var zeroID ObjectID
	if !zeroID.IsZero() {
		t.Error("Zero ObjectID should return true for IsZero()")
	}

	// Test non-zero ID
	nonZeroID := ObjectID{1}
	if nonZeroID.IsZero() {
		t.Error("Non-zero ObjectID should return false for IsZero()")
	}

	// Test String representation
	if len(zeroID.String()) != 40 {
		t.Errorf("ObjectID String() length = %d, want 40", len(zeroID.String()))
	}

	// Test Short representation
	if len(nonZeroID.Short()) != 7 {
		t.Errorf("ObjectID Short() length = %d, want 7", len(nonZeroID.Short()))
	}

	// Test Equal
	if !zeroID.Equal(ObjectID{}) {
		t.Error("Equal ObjectIDs should return true")
	}
	if zeroID.Equal(nonZeroID) {
		t.Error("Different ObjectIDs should return false")
	}
}