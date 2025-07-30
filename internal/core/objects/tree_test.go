package objects

import (
	"testing"
)

func TestNewTree(t *testing.T) {
	tree := NewTree()
	
	if tree.Type() != TypeTree {
		t.Errorf("Tree.Type() = %v, want %v", tree.Type(), TypeTree)
	}
	
	if len(tree.Entries()) != 0 {
		t.Errorf("New tree should have no entries, got %d", len(tree.Entries()))
	}
}

func TestTree_AddEntry(t *testing.T) {
	tree := NewTree()
	
	// Create some object IDs for testing
	blobID1, _ := NewObjectID("1234567890abcdef1234567890abcdef12345678")
	blobID2, _ := NewObjectID("abcdef1234567890abcdef1234567890abcdef12")
	treeID, _ := NewObjectID("fedcba0987654321fedcba0987654321fedcba09")
	
	tests := []struct {
		name    string
		mode    FileMode
		fname   string
		id      ObjectID
		wantErr bool
	}{
		{
			name:    "add blob",
			mode:    ModeBlob,
			fname:   "file.txt",
			id:      blobID1,
			wantErr: false,
		},
		{
			name:    "add executable",
			mode:    ModeExec,
			fname:   "script.sh",
			id:      blobID2,
			wantErr: false,
		},
		{
			name:    "add subtree",
			mode:    ModeTree,
			fname:   "subdir",
			id:      treeID,
			wantErr: false,
		},
		{
			name:    "duplicate name",
			mode:    ModeBlob,
			fname:   "file.txt",
			id:      blobID2,
			wantErr: true,
		},
		{
			name:    "empty name",
			mode:    ModeBlob,
			fname:   "",
			id:      blobID1,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tree.AddEntry(tt.mode, tt.fname, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Tree.AddEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	
	// Verify entries were added correctly
	entries := tree.Entries()
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}
}

func TestTree_Serialize(t *testing.T) {
	tree := NewTree()
	
	// Add entries in non-alphabetical order to test sorting
	id1, _ := NewObjectID("1234567890abcdef1234567890abcdef12345678")
	id2, _ := NewObjectID("abcdef1234567890abcdef1234567890abcdef12")
	
	tree.AddEntry(ModeBlob, "zebra.txt", id1)
	tree.AddEntry(ModeBlob, "apple.txt", id2)
	
	data, err := tree.Serialize()
	if err != nil {
		t.Fatalf("Tree.Serialize() error = %v", err)
	}
	
	// The serialized data should have entries sorted by name
	// We can't easily verify the exact bytes, but we can parse it back
	parsed, err := ParseTree(tree.ID(), data)
	if err != nil {
		t.Fatalf("ParseTree() error = %v", err)
	}
	
	entries := parsed.Entries()
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
	
	// Verify entries are in alphabetical order
	if entries[0].Name != "apple.txt" {
		t.Errorf("First entry should be apple.txt, got %s", entries[0].Name)
	}
	if entries[1].Name != "zebra.txt" {
		t.Errorf("Second entry should be zebra.txt, got %s", entries[1].Name)
	}
}

func TestParseTree(t *testing.T) {
	// Create a tree and serialize it
	tree := NewTree()
	id1, _ := NewObjectID("1234567890abcdef1234567890abcdef12345678")
	id2, _ := NewObjectID("abcdef1234567890abcdef1234567890abcdef12")
	
	tree.AddEntry(ModeBlob, "file1.txt", id1)
	tree.AddEntry(ModeExec, "script.sh", id2)
	
	data, err := tree.Serialize()
	if err != nil {
		t.Fatalf("Tree.Serialize() error = %v", err)
	}
	
	// Parse it back
	parsed, err := ParseTree(tree.ID(), data)
	if err != nil {
		t.Fatalf("ParseTree() error = %v", err)
	}
	
	if parsed.ID() != tree.ID() {
		t.Errorf("ParseTree ID = %v, want %v", parsed.ID(), tree.ID())
	}
	
	entries := parsed.Entries()
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
	
	// Verify entries
	if entries[0].Name != "file1.txt" || entries[0].Mode != ModeBlob || entries[0].ID != id1 {
		t.Errorf("First entry mismatch: %+v", entries[0])
	}
	if entries[1].Name != "script.sh" || entries[1].Mode != ModeExec || entries[1].ID != id2 {
		t.Errorf("Second entry mismatch: %+v", entries[1])
	}
}