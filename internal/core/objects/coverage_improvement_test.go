package objects

import (
	"bytes"
	"compress/zlib"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test decompressData function (currently 0% coverage)
func TestDecompressData(t *testing.T) {
	// Test data to compress and decompress
	testData := []byte("Hello, World! This is test data for compression.")

	// Compress the data first
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write to zlib writer: %v", err)
	}
	w.Close()
	compressed := buf.Bytes()

	// Test decompressing valid data
	decompressed, err := decompressData(compressed)
	if err != nil {
		t.Errorf("decompressData() error = %v, want nil", err)
	}
	if !bytes.Equal(decompressed, testData) {
		t.Errorf("decompressData() = %v, want %v", decompressed, testData)
	}

	// Test decompressing invalid data
	invalidData := []byte("not compressed data")
	_, err = decompressData(invalidData)
	if err == nil {
		t.Error("decompressData() should return error for invalid data")
	}

	// Test decompressing empty data
	_, err = decompressData([]byte{})
	if err == nil {
		t.Error("decompressData() should return error for empty data")
	}
}

// Test ReadObject function more comprehensively (currently 26.3% coverage)
func TestReadObjectCoverage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)

	// Test reading non-existent object
	nonExistentID := ObjectID{}
	copy(nonExistentID[:], []byte("0123456789abcdef0123456789abcdef01234567"))
	_, err = storage.ReadObject(nonExistentID)
	if err == nil {
		t.Error("ReadObject() should return error for non-existent object")
	}

	// Test with various object types
	testCases := []struct {
		name   string
		object Object
	}{
		{
			name:   "blob object",
			object: NewBlob([]byte("test blob content")),
		},
		{
			name: "tree object",
			object: &Tree{entries: []TreeEntry{
				{Mode: ModeBlob, Name: "file.txt", ID: NewBlob([]byte("content")).ID()},
			}},
		},
		{
			name: "commit object",
			object: &Commit{
				tree: NewBlob([]byte("fake tree")).ID(),
				author: Signature{
					Name:  "Test Author",
					Email: "test@example.com",
					When:  time.Now(),
				},
				committer: Signature{
					Name:  "Test Committer", 
					Email: "committer@example.com",
					When:  time.Now(),
				},
				message: "Test commit message",
			},
		},
		{
			name: "tag object",
			object: &Tag{
				object: NewBlob([]byte("fake target")).ID(),
				typ:    TypeCommit,
				tag:    "test-tag",
				tagger: Signature{
					Name:  "Test Tagger",
					Email: "tagger@example.com",
					When:  time.Now(),
				},
				message: "Test tag message",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write the object first
			err := storage.WriteObject(tc.object)
			if err != nil {
				t.Fatalf("WriteObject() error = %v", err)
			}

			// Read it back
			readObj, err := storage.ReadObject(tc.object.ID())
			if err != nil {
				t.Errorf("ReadObject() error = %v", err)
				return
			}

			// Check it's the same type and ID
			if readObj.Type() != tc.object.Type() {
				t.Errorf("ReadObject() type = %v, want %v", readObj.Type(), tc.object.Type())
			}
			if readObj.ID() != tc.object.ID() {
				t.Errorf("ReadObject() ID = %v, want %v", readObj.ID(), tc.object.ID())
			}
		})
	}
}

// Test corrupted object file scenarios for ReadObject
func TestReadObjectCorruptedData(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage-corrupt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)

	// Create a blob and write it
	blob := NewBlob([]byte("test content"))
	err = storage.WriteObject(blob)
	if err != nil {
		t.Fatalf("WriteObject() error = %v", err)
	}

	// Get the object file path
	objPath := storage.objectPath(blob.ID())

	// Test corrupted file scenarios
	testCases := []struct {
		name        string
		corruptFunc func(string) error
	}{
		{
			name: "empty file",
			corruptFunc: func(path string) error {
				return os.WriteFile(path, []byte{}, 0644)
			},
		},
		{
			name: "invalid zlib data",
			corruptFunc: func(path string) error {
				return os.WriteFile(path, []byte("not zlib compressed"), 0644)
			},
		},
		{
			name: "malformed object header",
			corruptFunc: func(path string) error {
				// Write valid zlib data but with malformed content
				var buf bytes.Buffer
				w := zlib.NewWriter(&buf)
				w.Write([]byte("invalid header format"))
				w.Close()
				return os.WriteFile(path, buf.Bytes(), 0644)
			},
		},
		{
			name: "missing null byte in header",
			corruptFunc: func(path string) error {
				var buf bytes.Buffer
				w := zlib.NewWriter(&buf)
				w.Write([]byte("blob 12 content without null"))
				w.Close()
				return os.WriteFile(path, buf.Bytes(), 0644)
			},
		},
		{
			name: "invalid size in header",
			corruptFunc: func(path string) error {
				var buf bytes.Buffer
				w := zlib.NewWriter(&buf)
				w.Write([]byte("blob invalid_size\x00content"))
				w.Close()
				return os.WriteFile(path, buf.Bytes(), 0644)
			},
		},
		{
			name: "size mismatch",
			corruptFunc: func(path string) error {
				var buf bytes.Buffer
				w := zlib.NewWriter(&buf)
				w.Write([]byte("blob 100\x00short content"))
				w.Close()
				return os.WriteFile(path, buf.Bytes(), 0644)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Corrupt the file
			err := tc.corruptFunc(objPath)
			if err != nil {
				t.Fatalf("Failed to corrupt file: %v", err)
			}

			// Try to read the corrupted object
			_, err = storage.ReadObject(blob.ID())
			if err == nil {
				t.Error("ReadObject() should return error for corrupted data")
			}
		})
	}
}

// Test NewBlobFromReader function (currently 75% coverage)
func TestNewBlobFromReaderCoverage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid reader",
			input:   "test content for blob",
			wantErr: false,
		},
		{
			name:    "empty reader",
			input:   "",
			wantErr: false,
		},
		{
			name:    "large content",
			input:   strings.Repeat("large content ", 1000),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			blob, err := NewBlobFromReader(reader)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBlobFromReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if string(blob.Data()) != tt.input {
					t.Errorf("NewBlobFromReader() data = %v, want %v", string(blob.Data()), tt.input)
				}
				if blob.Type() != TypeBlob {
					t.Errorf("NewBlobFromReader() type = %v, want %v", blob.Type(), TypeBlob)
				}
			}
		})
	}

	// Test error case with failing reader
	failingReader := &failingReader{}
	_, err := NewBlobFromReader(failingReader)
	if err == nil {
		t.Error("NewBlobFromReader() should return error for failing reader")
	}
}

// Helper type that always returns an error when reading
type failingReader struct{}

func (f *failingReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

// Test WriteObject error conditions (currently 80% coverage)
func TestWriteObjectErrorConditions(t *testing.T) {
	// Test with read-only directory
	tmpDir, err := os.MkdirTemp("", "storage-readonly-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewStorage(tmpDir)
	blob := NewBlob([]byte("test"))

	// Create the objects directory structure first
	objDir := filepath.Join(tmpDir, "objects", blob.ID().String()[:2])
	err = os.MkdirAll(objDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create object directory: %v", err)
	}

	// Make the directory read-only
	err = os.Chmod(objDir, 0444)
	if err != nil {
		t.Fatalf("Failed to make directory read-only: %v", err)
	}
	defer os.Chmod(objDir, 0755) // Restore permissions for cleanup

	// Try to write object to read-only directory
	err = storage.WriteObject(blob)
	if err == nil {
		t.Error("WriteObject() should return error for read-only directory")
	}
}

// Test compressData function error conditions (currently 62.5% coverage)
func TestCompressDataErrorConditions(t *testing.T) {
	// Test normal compression
	data := []byte("test data for compression")
	compressed, err := compressData(data)
	if err != nil {
		t.Errorf("compressData() error = %v, want nil", err)
	}
	if len(compressed) == 0 {
		t.Error("compressData() should return non-empty compressed data")
	}

	// Test with empty data
	compressed, err = compressData([]byte{})
	if err != nil {
		t.Errorf("compressData() error = %v, want nil", err)
	}
	if len(compressed) == 0 {
		t.Error("compressData() should return non-empty compressed data even for empty input")
	}

	// Test with large data
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	compressed, err = compressData(largeData)
	if err != nil {
		t.Errorf("compressData() error = %v, want nil", err)
	}
	if len(compressed) == 0 {
		t.Error("compressData() should return non-empty compressed data for large input")
	}
}

// Test Storage.Init error conditions (currently 69.2% coverage)
func TestStorageInitErrorConditions(t *testing.T) {
	// Test with invalid path (file instead of directory)
	tmpFile, err := os.CreateTemp("", "storage-init-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	storage := NewStorage(tmpFile.Name())
	err = storage.Init()
	if err == nil {
		t.Error("Init() should return error when gitDir is a file, not directory")
	}
}