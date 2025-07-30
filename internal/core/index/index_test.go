package index

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fenilsonani/vcs/internal/core/objects"
)

func TestEntry_Stage(t *testing.T) {
	tests := []struct {
		name     string
		flags    uint16
		expected int
	}{
		{"stage 0", 0x0000, 0},
		{"stage 1", 0x1000, 1},
		{"stage 2", 0x2000, 2},
		{"stage 3", 0x3000, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Entry{Flags: tt.flags}
			if got := e.Stage(); got != tt.expected {
				t.Errorf("Stage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEntry_SetStage(t *testing.T) {
	tests := []struct {
		name     string
		stage    int
		expected uint16
	}{
		{"set stage 0", 0, 0x0000},
		{"set stage 1", 1, 0x1000},
		{"set stage 2", 2, 0x2000},
		{"set stage 3", 3, 0x3000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Entry{Flags: 0x0FFF} // Set other flags
			e.SetStage(tt.stage)
			if (e.Flags & FlagStageMask) != tt.expected {
				t.Errorf("SetStage(%d) flags = %04x, want stage bits %04x", tt.stage, e.Flags, tt.expected)
			}
			// Verify other flags preserved
			if (e.Flags & FlagNameMask) != 0x0FFF {
				t.Error("SetStage() should preserve other flags")
			}
		})
	}
}

func TestNew(t *testing.T) {
	idx := New()
	if idx == nil {
		t.Fatal("New() returned nil")
	}
	if idx.version != IndexVersion {
		t.Errorf("version = %v, want %v", idx.version, IndexVersion)
	}
	if len(idx.entries) != 0 {
		t.Errorf("entries length = %v, want 0", len(idx.entries))
	}
	if len(idx.cache) != 0 {
		t.Errorf("cache length = %v, want 0", len(idx.cache))
	}
}

func TestIndex_AddAndGet(t *testing.T) {
	idx := New()
	
	// Create test entry
	entry := &Entry{
		Path:  "test.txt",
		Mode:  objects.ModeBlob,
		ID:    objects.ObjectID{1, 2, 3, 4, 5},
		Size:  100,
		MTime: time.Now(),
		CTime: time.Now(),
	}

	// Add entry
	if err := idx.Add(entry); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Get entry
	got, ok := idx.Get("test.txt")
	if !ok {
		t.Fatal("Get() returned false, want true")
	}
	if got.Path != entry.Path {
		t.Errorf("Get() path = %v, want %v", got.Path, entry.Path)
	}

	// Test empty path
	if err := idx.Add(&Entry{}); err == nil {
		t.Error("Add() with empty path should return error")
	}
}

func TestIndex_AddUpdate(t *testing.T) {
	idx := New()
	
	// Add initial entry
	entry1 := &Entry{
		Path: "test.txt",
		ID:   objects.ObjectID{1, 2, 3},
		Size: 100,
	}
	idx.Add(entry1)

	// Update with new entry
	entry2 := &Entry{
		Path: "test.txt",
		ID:   objects.ObjectID{4, 5, 6},
		Size: 200,
	}
	idx.Add(entry2)

	// Verify update
	if len(idx.entries) != 1 {
		t.Errorf("entries length = %v, want 1", len(idx.entries))
	}
	
	got, _ := idx.Get("test.txt")
	if got.Size != 200 {
		t.Errorf("updated size = %v, want 200", got.Size)
	}
}

func TestIndex_Remove(t *testing.T) {
	idx := New()
	
	// Add entries
	idx.Add(&Entry{Path: "file1.txt"})
	idx.Add(&Entry{Path: "file2.txt"})
	idx.Add(&Entry{Path: "file3.txt"})

	// Remove middle entry
	if err := idx.Remove("file2.txt"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// Verify removal
	if len(idx.entries) != 2 {
		t.Errorf("entries length = %v, want 2", len(idx.entries))
	}
	
	if _, ok := idx.Get("file2.txt"); ok {
		t.Error("Get() should return false for removed entry")
	}

	// Remove non-existent
	if err := idx.Remove("nonexistent.txt"); err == nil {
		t.Error("Remove() should return error for non-existent entry")
	}
}

func TestIndex_Clear(t *testing.T) {
	idx := New()
	
	// Add entries
	idx.Add(&Entry{Path: "file1.txt"})
	idx.Add(&Entry{Path: "file2.txt"})

	// Clear
	idx.Clear()

	// Verify
	if len(idx.entries) != 0 {
		t.Errorf("entries length = %v, want 0", len(idx.entries))
	}
	if len(idx.cache) != 0 {
		t.Errorf("cache length = %v, want 0", len(idx.cache))
	}
}

func TestIndex_Sort(t *testing.T) {
	idx := New()
	
	// Add entries in reverse order
	idx.Add(&Entry{Path: "c.txt"})
	idx.Add(&Entry{Path: "a.txt"})
	idx.Add(&Entry{Path: "b.txt"})

	// Verify sorted order
	entries := idx.Entries()
	if len(entries) != 3 {
		t.Fatalf("entries length = %v, want 3", len(entries))
	}
	
	expectedPaths := []string{"a.txt", "b.txt", "c.txt"}
	for i, expected := range expectedPaths {
		if entries[i].Path != expected {
			t.Errorf("entries[%d].Path = %v, want %v", i, entries[i].Path, expected)
		}
	}
}

func TestIndex_WriteToAndReadFrom(t *testing.T) {
	// Create index with entries
	idx1 := New()
	now := time.Now().Truncate(time.Second)
	
	entries := []*Entry{
		{
			Path:  "file1.txt",
			Mode:  objects.ModeBlob,
			ID:    objects.ObjectID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			Size:  100,
			MTime: now,
			CTime: now,
			UID:   1000,
			GID:   1000,
			Dev:   64769,
			Ino:   123456,
			Flags: 10,
		},
		{
			Path:  "dir/file2.txt",
			Mode:  objects.ModeExec,
			ID:    objects.ObjectID{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
			Size:  200,
			MTime: now.Add(time.Hour),
			CTime: now.Add(time.Hour),
			UID:   1001,
			GID:   1001,
			Dev:   64770,
			Ino:   123457,
			Flags: 14,
		},
	}
	
	for _, e := range entries {
		if err := idx1.Add(e); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := idx1.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}

	// Read back
	idx2 := New()
	if err := idx2.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom() error = %v", err)
	}

	// Compare
	if idx2.version != idx1.version {
		t.Errorf("version = %v, want %v", idx2.version, idx1.version)
	}
	
	if len(idx2.entries) != len(idx1.entries) {
		t.Fatalf("entries length = %v, want %v", len(idx2.entries), len(idx1.entries))
	}

	for i, e1 := range idx1.entries {
		e2 := idx2.entries[i]
		
		if e2.Path != e1.Path {
			t.Errorf("entry[%d].Path = %v, want %v", i, e2.Path, e1.Path)
		}
		if e2.Mode != e1.Mode {
			t.Errorf("entry[%d].Mode = %v, want %v", i, e2.Mode, e1.Mode)
		}
		if e2.ID != e1.ID {
			t.Errorf("entry[%d].ID = %v, want %v", i, e2.ID, e1.ID)
		}
		if e2.Size != e1.Size {
			t.Errorf("entry[%d].Size = %v, want %v", i, e2.Size, e1.Size)
		}
		if !e2.MTime.Equal(e1.MTime.Truncate(time.Second)) {
			t.Errorf("entry[%d].MTime = %v, want %v", i, e2.MTime, e1.MTime.Truncate(time.Second))
		}
		if !e2.CTime.Equal(e1.CTime.Truncate(time.Second)) {
			t.Errorf("entry[%d].CTime = %v, want %v", i, e2.CTime, e1.CTime.Truncate(time.Second))
		}
	}
}

func TestIndex_ReadFromErrors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{
			name:    "short header",
			data:    []byte("DI"),
			wantErr: "failed to read header",
		},
		{
			name:    "invalid signature",
			data:    []byte("XXXX\x00\x00\x00\x02\x00\x00\x00\x00"),
			wantErr: "invalid index signature",
		},
		{
			name:    "invalid version",
			data:    []byte("DIRC\x00\x00\x00\x01\x00\x00\x00\x00"),
			wantErr: "unsupported index version",
		},
		{
			name:    "checksum mismatch",
			data:    append([]byte("DIRC\x00\x00\x00\x02\x00\x00\x00\x00"), make([]byte, 20)...),
			wantErr: "checksum mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := New()
			err := idx.ReadFrom(bytes.NewReader(tt.data))
			if err == nil {
				t.Fatal("ReadFrom() error = nil, want error")
			}
			if !bytes.Contains([]byte(err.Error()), []byte(tt.wantErr)) {
				t.Errorf("ReadFrom() error = %v, want error containing %v", err, tt.wantErr)
			}
		})
	}
}

func TestIndex_LongPath(t *testing.T) {
	idx1 := New()
	
	// Create entry with very long path
	longPath := ""
	for i := 0; i < 300; i++ {
		longPath += "a"
	}
	longPath += "/file.txt"
	
	entry := &Entry{
		Path:  longPath,
		Mode:  objects.ModeBlob,
		ID:    objects.ObjectID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		MTime: time.Now(),
		CTime: time.Now(),
	}
	
	if err := idx1.Add(entry); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Write and read back
	var buf bytes.Buffer
	if err := idx1.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}

	idx2 := New()
	if err := idx2.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom() error = %v", err)
	}

	// Verify long path preserved
	got, ok := idx2.Get(longPath)
	if !ok {
		t.Fatal("Get() returned false for long path")
	}
	if got.Path != longPath {
		t.Errorf("Path length = %v, want %v", len(got.Path), len(longPath))
	}
}

func TestIndex_FileOperations(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vcs-index-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "index")

	// Create index
	idx1 := New()
	idx1.Add(&Entry{
		Path: "test.txt",
		Mode: objects.ModeBlob,
		ID:   objects.ObjectID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
	})

	// Write to file
	if err := idx1.WriteToFile(indexPath); err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("Index file not created: %v", err)
	}

	// Read from file
	idx2 := New()
	if err := idx2.ReadFromFile(indexPath); err != nil {
		t.Fatalf("ReadFromFile() error = %v", err)
	}

	// Verify content
	if len(idx2.entries) != 1 {
		t.Errorf("entries length = %v, want 1", len(idx2.entries))
	}

	// Test non-existent file
	if err := idx2.ReadFromFile("/nonexistent/path/index"); err == nil {
		t.Error("ReadFromFile() should return error for non-existent file")
	}
}

func TestIndex_EdgeCases(t *testing.T) {
	idx := New()

	// Test with empty path (should fail)
	emptyEntry := &Entry{
		Path: "",
		Mode: objects.ModeBlob,
		ID:   objects.ObjectID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
	}
	if err := idx.Add(emptyEntry); err == nil {
		t.Error("Add() with empty path should fail")
	}

	// Test Get with non-existent path
	_, exists := idx.Get("nonexistent.txt")
	if exists {
		t.Error("Get() should return false for non-existent path")
	}

	// Test Remove with non-existent path
	err := idx.Remove("nonexistent.txt")
	if err == nil {
		t.Error("Remove() should return error for non-existent path")
	}

	// Test Clear
	idx.Clear()
	if len(idx.entries) != 0 {
		t.Errorf("Clear() left %d entries, want 0", len(idx.entries))
	}
}

func TestIndex_DuplicateEntries(t *testing.T) {
	idx := New()

	// Add entry
	entry1 := &Entry{
		Path: "duplicate.txt",
		Mode: objects.ModeBlob,
		ID:   objects.ObjectID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		Size: 100,
	}
	if err := idx.Add(entry1); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Add same path again with different data
	entry2 := &Entry{
		Path: "duplicate.txt",
		Mode: objects.ModeExec,
		ID:   objects.ObjectID{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21},
		Size: 200,
	}
	if err := idx.Add(entry2); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Should only have one entry (updated)
	if len(idx.entries) != 1 {
		t.Errorf("entries length = %v, want 1", len(idx.entries))
	}

	// Should have updated data
	got, exists := idx.Get("duplicate.txt")
	if !exists {
		t.Fatal("Get() returned false for existing entry")
	}
	if got.Mode != objects.ModeExec {
		t.Errorf("Mode = %v, want %v", got.Mode, objects.ModeExec)
	}
	if got.Size != 200 {
		t.Errorf("Size = %v, want 200", got.Size)
	}
}

func TestIndex_InvalidData(t *testing.T) {
	idx := New()

	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{
			name:    "truncated header",
			data:    []byte("DIRC\x00\x00\x00\x02"),
			wantErr: "failed to read header",
		},
		{
			name:    "unsupported version 5",
			data:    []byte("DIRC\x00\x00\x00\x05\x00\x00\x00\x00"),
			wantErr: "unsupported index version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := idx.ReadFrom(bytes.NewReader(tt.data))
			if err == nil {
				t.Fatal("ReadFrom() should return error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ReadFrom() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}