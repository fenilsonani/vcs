package objects

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestObjectID_Short(t *testing.T) {
	tests := []struct {
		name     string
		id       ObjectID
		expected string
	}{
		{
			name:     "normal ID",
			id:       ObjectID{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			expected: "1234567",
		},
		{
			name:     "zero ID",
			id:       ObjectID{},
			expected: "0000000",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Short(); got != tt.expected {
				t.Errorf("Short() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHashReader(t *testing.T) {
	tests := []struct {
		name       string
		objectType ObjectType
		data       string
		size       int64
		wantErr    bool
	}{
		{
			name:       "valid blob",
			objectType: TypeBlob,
			data:       "hello world",
			size:       11,
			wantErr:    false,
		},
		{
			name:       "empty blob",
			objectType: TypeBlob,
			data:       "",
			size:       0,
			wantErr:    false,
		},
		{
			name:       "large blob",
			objectType: TypeBlob,
			data:       strings.Repeat("a", 1000),
			size:       1000,
			wantErr:    false,
		},
		{
			name:       "size mismatch",
			objectType: TypeBlob,
			data:       "hello",
			size:       10, // Wrong size  
			wantErr:    false, // HashReader doesn't validate size
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.data)
			id, err := HashReader(tt.objectType, tt.size, reader)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("HashReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				// For size mismatch test, the hash will be different
				if tt.name == "size mismatch" {
					// Just verify we got a hash
					if id.IsZero() {
						t.Error("HashReader() returned zero ID")
					}
				} else {
					// Verify hash matches ComputeHash
					expectedID := ComputeHash(tt.objectType, []byte(tt.data))
					if id != expectedID {
						t.Errorf("HashReader() = %v, want %v", id, expectedID)
					}
				}
			}
		})
	}
}

func TestHashReader_Error(t *testing.T) {
	// Test with a reader that returns an error
	errReader := &errorReader{err: io.ErrUnexpectedEOF}
	
	_, err := HashReader(TypeBlob, 10, errReader)
	if err == nil {
		t.Error("HashReader() with error reader should return error")
	}
	if !strings.Contains(err.Error(), "failed to hash reader") {
		t.Errorf("HashReader() error = %v, want error containing 'failed to hash reader'", err)
	}
}

func TestParseObjectID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid full hash",
			input:   "1234567890abcdef1234567890abcdef12345678",
			wantErr: false,
		},
		{
			name:    "hash with spaces",
			input:   "  1234567890abcdef1234567890abcdef12345678  ",
			wantErr: false,
		},
		{
			name:    "hash with newlines",
			input:   "\n1234567890abcdef1234567890abcdef12345678\n",
			wantErr: false,
		},
		{
			name:    "hash with tabs",
			input:   "\t1234567890abcdef1234567890abcdef12345678\t",
			wantErr: false,
		},
		{
			name:    "abbreviated hash",
			input:   "1234567",
			wantErr: true, // Not yet supported
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			input:   "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseObjectID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseObjectID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && got.IsZero() {
				t.Error("ParseObjectID() returned zero ID")
			}
		})
	}
}

func TestTrimWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no whitespace", "hello", "hello"},
		{"leading spaces", "  hello", "hello"},
		{"trailing spaces", "hello  ", "hello"},
		{"both spaces", "  hello  ", "hello"},
		{"tabs", "\thello\t", "hello"},
		{"newlines", "\nhello\n", "hello"},
		{"mixed", " \t\nhello \n\t", "hello"},
		{"only whitespace", " \t\n", ""},
		{"empty", "", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimWhitespace(tt.input); got != tt.want {
				t.Errorf("trimWhitespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsWhitespace(t *testing.T) {
	tests := []struct {
		b    byte
		want bool
	}{
		{' ', true},
		{'\t', true},
		{'\n', true},
		{'\r', true},
		{'a', false},
		{'0', false},
		{0, false},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.b), func(t *testing.T) {
			if got := isWhitespace(tt.b); got != tt.want {
				t.Errorf("isWhitespace(%q) = %v, want %v", tt.b, got, tt.want)
			}
		})
	}
}

// errorReader is a reader that always returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func TestParseSignature(t *testing.T) {
	// Test the unimplemented ParseSignature function
	_, err := ParseSignature([]byte("test data"))
	if err == nil {
		t.Error("ParseSignature() should return error (not implemented)")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("ParseSignature() error = %v, want 'not yet implemented'", err)
	}
}

// Test corner cases for ComputeHash
func TestComputeHash_EdgeCases(t *testing.T) {
	// Test with various object types
	types := []ObjectType{TypeBlob, TypeTree, TypeCommit, TypeTag}
	
	for _, objType := range types {
		t.Run(string(objType), func(t *testing.T) {
			// Test with nil data (should work like empty data)
			id1 := ComputeHash(objType, nil)
			id2 := ComputeHash(objType, []byte{})
			
			if id1 != id2 {
				t.Errorf("ComputeHash with nil != ComputeHash with empty slice")
			}
			
			// Test with large data
			largeData := bytes.Repeat([]byte("x"), 1024*1024) // 1MB
			id := ComputeHash(objType, largeData)
			if id.IsZero() {
				t.Error("ComputeHash with large data returned zero ID")
			}
		})
	}
}