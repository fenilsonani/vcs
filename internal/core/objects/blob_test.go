package objects

import (
	"bytes"
	"io"
	"testing"
)

func TestNewBlob(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		expectedID   string
		expectedSize int64
	}{
		{
			name:         "empty blob",
			data:         []byte{},
			expectedID:   "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391",
			expectedSize: 0,
		},
		{
			name:         "hello world",
			data:         []byte("hello world\n"),
			expectedID:   "3b18e512dba79e4c8300dd08aeb37f8e728b8dad",
			expectedSize: 12,
		},
		{
			name:         "binary data",
			data:         []byte{0x00, 0x01, 0x02, 0x03, 0xff},
			expectedID:   "e2613b3caa99ff78f481d27639d5500710722310",
			expectedSize: 5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob := NewBlob(tt.data)
			
			if blob.Type() != TypeBlob {
				t.Errorf("Blob.Type() = %v, want %v", blob.Type(), TypeBlob)
			}
			
			if blob.Size() != tt.expectedSize {
				t.Errorf("Blob.Size() = %v, want %v", blob.Size(), tt.expectedSize)
			}
			
			if blob.ID().String() != tt.expectedID {
				t.Errorf("Blob.ID() = %v, want %v", blob.ID().String(), tt.expectedID)
			}
			
			if !bytes.Equal(blob.Data(), tt.data) {
				t.Errorf("Blob.Data() = %v, want %v", blob.Data(), tt.data)
			}
			
			serialized, err := blob.Serialize()
			if err != nil {
				t.Fatalf("Blob.Serialize() error = %v", err)
			}
			
			if !bytes.Equal(serialized, tt.data) {
				t.Errorf("Blob.Serialize() = %v, want %v", serialized, tt.data)
			}
		})
	}
}

func TestNewBlobFromReader(t *testing.T) {
	data := []byte("test content from reader")
	reader := bytes.NewReader(data)
	
	blob, err := NewBlobFromReader(reader)
	if err != nil {
		t.Fatalf("NewBlobFromReader() error = %v", err)
	}
	
	if !bytes.Equal(blob.Data(), data) {
		t.Errorf("Blob.Data() = %v, want %v", blob.Data(), data)
	}
}

func TestBlob_Reader(t *testing.T) {
	data := []byte("test content")
	blob := NewBlob(data)
	
	reader := blob.Reader()
	readData, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read from blob reader: %v", err)
	}
	
	if !bytes.Equal(readData, data) {
		t.Errorf("Read data = %v, want %v", readData, data)
	}
}

func TestParseBlob(t *testing.T) {
	data := []byte("parsed blob content")
	id := ComputeHash(TypeBlob, data)
	
	blob := ParseBlob(id, data)
	
	if blob.ID() != id {
		t.Errorf("ParseBlob ID = %v, want %v", blob.ID(), id)
	}
	
	if !bytes.Equal(blob.Data(), data) {
		t.Errorf("ParseBlob data = %v, want %v", blob.Data(), data)
	}
}