package index

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fenilsonani/vcs/internal/core/objects"
)

// Test Version function (currently 0% coverage)
func TestVersion(t *testing.T) {
	idx := New()
	version := idx.Version()
	
	// Should return version 2 by default
	if version != 2 {
		t.Errorf("Version() = %v, want 2", version)
	}
}

// Test WriteTo function more comprehensively (currently 82.4% coverage)  
func TestWriteToCoverage(t *testing.T) {
	idx := New()
	
	// Test writing empty index
	var buf bytes.Buffer
	err := idx.WriteTo(&buf)
	if err != nil {
		t.Errorf("WriteTo() error = %v", err)
	}
	if buf.Len() <= 0 {
		t.Errorf("WriteTo() wrote %d bytes, want > 0", buf.Len())
	}

	// Verify the header was written correctly
	written := buf.Bytes()
	if len(written) < 12 { // Minimum header size
		t.Errorf("WriteTo() wrote %d bytes, want at least 12", len(written))
	}

	// Check magic signature
	if !bytes.Equal(written[:4], []byte("DIRC")) {
		t.Errorf("WriteTo() magic = %v, want DIRC", written[:4])
	}

	// Test with entries
	idx = New()
	entry1 := &Entry{
		Mode: objects.ModeBlob,
		Size: 10,
		ID:   objects.NewBlob([]byte("test data")).ID(),
		Path: "file1.txt",
	}
	entry2 := &Entry{
		Mode: objects.ModeBlob,
		Size: 15,
		ID:   objects.NewBlob([]byte("more test data")).ID(),
		Path: "file2.txt",
	}
	idx.Add(entry1)
	idx.Add(entry2)

	buf.Reset()
	err = idx.WriteTo(&buf)
	if err != nil {
		t.Errorf("WriteTo() with entries error = %v", err)
	}
	if buf.Len() <= 12 { // Should be more than just header
		t.Errorf("WriteTo() with entries wrote %d bytes, want > 12", buf.Len())
	}

	// Test writing to failing writer
	failingWriter := &failingWriter{}
	err = idx.WriteTo(failingWriter)
	if err == nil {
		t.Error("WriteTo() should return error for failing writer")
	}
}

// Test reading entries indirectly through ReadFromFile (readEntry is unexported)
func TestReadEntryCoverage(t *testing.T) {
	// Create a simple index file and test reading it to cover readEntry code paths
	tmpDir, err := os.MkdirTemp("", "index-read-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create index with entries and write it
	idx := New()
	entry := &Entry{
		Mode: objects.ModeBlob,
		Size: 10,
		ID:   objects.NewBlob([]byte("test data")).ID(),
		Path: "test.txt",
		CTime: time.Now(),
		MTime: time.Now(),
	}
	idx.Add(entry)

	indexPath := filepath.Join(tmpDir, "index")
	err = idx.WriteToFile(indexPath)
	if err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}

	// Now test reading it back (this exercises readEntry)
	newIdx := New()
	err = newIdx.ReadFromFile(indexPath)
	if err != nil {
		t.Errorf("ReadFromFile() error = %v", err)
	}

	// Verify the entry was read correctly
	if len(newIdx.Entries()) != 1 {
		t.Errorf("ReadFromFile() entries count = %d, want 1", len(newIdx.Entries()))
	}

	readEntry, exists := newIdx.Get("test.txt")
	if !exists {
		t.Error("ReadFromFile() failed to read entry")
	} else if readEntry.Path != "test.txt" {
		t.Errorf("ReadFromFile() entry path = %s, want test.txt", readEntry.Path)
	}

	// Test with corrupted index file
	corruptPath := filepath.Join(tmpDir, "corrupt")
	err = os.WriteFile(corruptPath, []byte("DIRC\x00\x00\x00\x02\x00\x00\x00\x01corrupt"), 0644)
	if err != nil {
		t.Fatalf("Failed to create corrupt file: %v", err)
	}

	corruptIdx := New()
	err = corruptIdx.ReadFromFile(corruptPath)
	if err == nil {
		t.Error("ReadFromFile() should return error for corrupted file")
	}
}

// Test WriteToFile error conditions (currently 64.3% coverage)
func TestWriteToFileErrorConditions(t *testing.T) {
	idx := New()
	
	// Add some entries
	entry := &Entry{
		Mode: objects.ModeBlob,
		Size: 10,
		ID:   objects.NewBlob([]byte("test")).ID(),
		Path: "test.txt",
	}
	idx.Add(entry)

	// Test writing to valid file
	tmpDir, err := os.MkdirTemp("", "index-write-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "index")
	err = idx.WriteToFile(indexPath)
	if err != nil {
		t.Errorf("WriteToFile() error = %v", err)
	}

	// Verify file was created and has content
	info, err := os.Stat(indexPath)
	if err != nil {
		t.Errorf("WriteToFile() failed to create file: %v", err)
	}
	if info.Size() <= 0 {
		t.Errorf("WriteToFile() created empty file")
	}

	// Test writing to read-only directory
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err = os.MkdirAll(readOnlyDir, 0444)
	if err != nil {
		t.Fatalf("Failed to create read-only dir: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0755) // Restore for cleanup

	readOnlyPath := filepath.Join(readOnlyDir, "index")
	err = idx.WriteToFile(readOnlyPath)
	if err == nil {
		t.Error("WriteToFile() should return error for read-only directory")
	}

	// Test with invalid path (directory instead of file)
	err = idx.WriteToFile(tmpDir) // Directory path instead of file
	if err == nil {
		t.Error("WriteToFile() should return error when path is a directory")
	}
}

// Test checksum calculation and validation
func TestChecksumHandling(t *testing.T) {
	idx := New()
	
	// Add entries
	entry1 := &Entry{
		Mode:  objects.ModeBlob,
		Size:  5,
		ID:    objects.NewBlob([]byte("test1")).ID(),
		Path:  "file1.txt",
		CTime: time.Now(),
		MTime: time.Now(),
	}
	entry2 := &Entry{
		Mode:  objects.ModeBlob,
		Size:  5,
		ID:    objects.NewBlob([]byte("test2")).ID(),
		Path:  "file2.txt",
		CTime: time.Now(),
		MTime: time.Now(),
	}
	
	idx.Add(entry1)
	idx.Add(entry2)

	// Write to buffer
	var buf bytes.Buffer
	err := idx.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}

	// Verify checksum is present at the end
	written := buf.Bytes()
	if len(written) < 20 { // Should have at least 20-byte checksum
		t.Errorf("WriteTo() output too short, missing checksum")
	}

	// The last 20 bytes should be the SHA-1 checksum
	checksum := written[len(written)-20:]
	if bytes.Equal(checksum, make([]byte, 20)) {
		t.Error("WriteTo() checksum appears to be all zeros")
	}
}

// Helper type that always returns an error when writing
type failingWriter struct{}

func (f *failingWriter) Write(p []byte) (n int, err error) {
	return 0, os.ErrInvalid
}