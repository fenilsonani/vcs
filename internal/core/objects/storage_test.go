package objects

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestStorage_Init(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "vcs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	gitDir := filepath.Join(tmpDir, ".git")
	storage := NewStorage(gitDir)
	
	if err := storage.Init(); err != nil {
		t.Fatalf("Storage.Init() error = %v", err)
	}
	
	// Verify directory structure
	objectsDir := filepath.Join(gitDir, "objects")
	if _, err := os.Stat(objectsDir); os.IsNotExist(err) {
		t.Error("objects directory not created")
	}
	
	// Check for subdirectories (00-ff)
	for i := 0; i < 256; i++ {
		dir := filepath.Join(objectsDir, fmt.Sprintf("%02x", i))
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("subdirectory %02x not created", i)
		}
	}
	
	// Check pack directory
	packDir := filepath.Join(objectsDir, "pack")
	if _, err := os.Stat(packDir); os.IsNotExist(err) {
		t.Error("pack directory not created")
	}
	
	// Check info directory
	infoDir := filepath.Join(objectsDir, "info")
	if _, err := os.Stat(infoDir); os.IsNotExist(err) {
		t.Error("info directory not created")
	}
}

func TestStorage_WriteAndReadObject(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "vcs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	gitDir := filepath.Join(tmpDir, ".git")
	storage := NewStorage(gitDir)
	
	if err := storage.Init(); err != nil {
		t.Fatalf("Storage.Init() error = %v", err)
	}
	
	// Test with different object types
	tests := []struct {
		name string
		obj  Object
	}{
		{
			name: "blob",
			obj:  NewBlob([]byte("test content")),
		},
		{
			name: "empty blob",
			obj:  NewBlob([]byte{}),
		},
		{
			name: "tree",
			obj: func() Object {
				tree := NewTree()
				id, _ := NewObjectID("1234567890abcdef1234567890abcdef12345678")
				tree.AddEntry(ModeBlob, "file.txt", id)
				return tree
			}(),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write object
			if err := storage.WriteObject(tt.obj); err != nil {
				t.Fatalf("Storage.WriteObject() error = %v", err)
			}
			
			// Verify object exists
			if !storage.HasObject(tt.obj.ID()) {
				t.Error("Storage.HasObject() = false, want true")
			}
			
			// Read object back
			read, err := storage.ReadObject(tt.obj.ID())
			if err != nil {
				t.Fatalf("Storage.ReadObject() error = %v", err)
			}
			
			// Verify object matches
			if read.ID() != tt.obj.ID() {
				t.Errorf("Read object ID = %v, want %v", read.ID(), tt.obj.ID())
			}
			
			if read.Type() != tt.obj.Type() {
				t.Errorf("Read object type = %v, want %v", read.Type(), tt.obj.Type())
			}
			
			if read.Size() != tt.obj.Size() {
				t.Errorf("Read object size = %v, want %v", read.Size(), tt.obj.Size())
			}
		})
	}
}

func TestStorage_WriteExistingObject(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "vcs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	gitDir := filepath.Join(tmpDir, ".git")
	storage := NewStorage(gitDir)
	
	if err := storage.Init(); err != nil {
		t.Fatalf("Storage.Init() error = %v", err)
	}
	
	// Write object
	blob := NewBlob([]byte("test content"))
	if err := storage.WriteObject(blob); err != nil {
		t.Fatalf("First WriteObject() error = %v", err)
	}
	
	// Write same object again (should succeed without error)
	if err := storage.WriteObject(blob); err != nil {
		t.Errorf("Second WriteObject() error = %v, want nil", err)
	}
}

func TestStorage_ReadNonExistentObject(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "vcs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	gitDir := filepath.Join(tmpDir, ".git")
	storage := NewStorage(gitDir)
	
	if err := storage.Init(); err != nil {
		t.Fatalf("Storage.Init() error = %v", err)
	}
	
	// Try to read non-existent object
	id, _ := NewObjectID("1234567890abcdef1234567890abcdef12345678")
	_, err = storage.ReadObject(id)
	if err == nil {
		t.Error("Storage.ReadObject() error = nil, want error")
	}
}