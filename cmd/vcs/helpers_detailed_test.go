package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureDirDetailed(t *testing.T) {
	tests := []struct {
		name    string
		path    func(string) string // Function to generate path relative to temp dir
		wantErr bool
	}{
		{
			name: "create single directory",
			path: func(tmpDir string) string {
				return filepath.Join(tmpDir, "newdir")
			},
			wantErr: false,
		},
		{
			name: "create nested directories",
			path: func(tmpDir string) string {
				return filepath.Join(tmpDir, "nested", "deep", "directories")
			},
			wantErr: false,
		},
		{
			name: "directory already exists",
			path: func(tmpDir string) string {
				existing := filepath.Join(tmpDir, "existing")
				os.MkdirAll(existing, 0755)
				return existing
			},
			wantErr: false,
		},
		{
			name: "empty path",
			path: func(tmpDir string) string {
				return ""
			},
			wantErr: false,
		},
		{
			name: "current directory",
			path: func(tmpDir string) string {
				return "."
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			oldWd, _ := os.Getwd()
			defer os.Chdir(oldWd)
			os.Chdir(tmpDir)

			path := tc.path(tmpDir)
			err := ensureDir(path)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if path != "" && path != "." {
					assert.DirExists(t, path)
				}
			}
		})
	}
}

func TestWriteFileDetailed(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(string) string // Returns file path to write to
		data    []byte
		wantErr bool
	}{
		{
			name: "write to new file",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "newfile.txt")
			},
			data:    []byte("hello world"),
			wantErr: false,
		},
		{
			name: "write to file in nested directory",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "nested", "deep", "file.txt")
			},
			data:    []byte("nested content"),
			wantErr: false,
		},
		{
			name: "overwrite existing file",
			setup: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "existing.txt")
				os.WriteFile(path, []byte("old content"), 0644)
				return path
			},
			data:    []byte("new content"),
			wantErr: false,
		},
		{
			name: "write empty content",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "empty.txt")
			},
			data:    []byte{},
			wantErr: false,
		},
		{
			name: "write binary data",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "binary.dat")
			},
			data:    []byte{0x00, 0xFF, 0x42, 0x13, 0x37},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tc.setup(tmpDir)

			err := writeFile(path, tc.data)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.FileExists(t, path)

				// Verify content
				content, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, tc.data, content)

				// Verify no temp file remains
				assert.NoFileExists(t, path+".tmp")
			}
		})
	}
}

func TestWriteFileAtomic(t *testing.T) {
	// Test the atomic nature of writeFile
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "atomic.txt")

	// Create initial file
	initialData := []byte("initial content")
	err := writeFile(path, initialData)
	require.NoError(t, err)

	// Verify initial content
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, initialData, content)

	// Overwrite with new content
	newData := []byte("new atomic content")
	err = writeFile(path, newData)
	require.NoError(t, err)

	// Verify new content
	content, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, newData, content)
}

func TestAppendToFileDetailed(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(string) string // Returns file path
		data    []byte
		wantErr bool
	}{
		{
			name: "append to new file",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "new.txt")
			},
			data:    []byte("first line\n"),
			wantErr: false,
		},
		{
			name: "append to existing file",
			setup: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "existing.txt")
				os.WriteFile(path, []byte("existing content\n"), 0644)
				return path
			},
			data:    []byte("appended content\n"),
			wantErr: false,
		},
		{
			name: "append empty data",
			setup: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "empty_append.txt")
				os.WriteFile(path, []byte("initial\n"), 0644)
				return path
			},
			data:    []byte{},
			wantErr: false,
		},
		{
			name: "append binary data",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "binary.dat")
			},
			data:    []byte{0x00, 0xFF, 0x42},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tc.setup(tmpDir)

			// Get original content if file exists
			var originalContent []byte
			if fileExists(path) {
				originalContent, _ = os.ReadFile(path)
			}

			err := appendToFile(path, tc.data)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.FileExists(t, path)

				// Verify content was appended
				content, err := os.ReadFile(path)
				require.NoError(t, err)

				expectedContent := append(originalContent, tc.data...)
				assert.Equal(t, expectedContent, content)
			}
		})
	}
}

func TestAppendToFileMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "multi_append.txt")

	// Append multiple times
	data1 := []byte("line 1\n")
	data2 := []byte("line 2\n")
	data3 := []byte("line 3\n")

	err := appendToFile(path, data1)
	require.NoError(t, err)

	err = appendToFile(path, data2)
	require.NoError(t, err)

	err = appendToFile(path, data3)
	require.NoError(t, err)

	// Verify all content is present
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	expected := append(append(data1, data2...), data3...)
	assert.Equal(t, expected, content)
}

func TestFileExistsDetailed(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		setup    func() string // Returns path to test
		expected bool
	}{
		{
			name: "existing file",
			setup: func() string {
				path := filepath.Join(tmpDir, "exists.txt")
				os.WriteFile(path, []byte("content"), 0644)
				return path
			},
			expected: true,
		},
		{
			name: "non-existent file",
			setup: func() string {
				return filepath.Join(tmpDir, "nonexistent.txt")
			},
			expected: false,
		},
		{
			name: "existing directory",
			setup: func() string {
				path := filepath.Join(tmpDir, "existingdir")
				os.MkdirAll(path, 0755)
				return path
			},
			expected: true,
		},
		{
			name: "empty path",
			setup: func() string {
				return ""
			},
			expected: false,
		},
		{
			name: "current directory",
			setup: func() string {
				return "."
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setup()
			result := fileExists(path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestReadFileDetailed(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(string) string // Returns file path
		expectData []byte
		wantErr    bool
		errMsg     string
	}{
		{
			name: "read existing file",
			setup: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "readable.txt")
				os.WriteFile(path, []byte("file content"), 0644)
				return path
			},
			expectData: []byte("file content"),
			wantErr:    false,
		},
		{
			name: "read empty file",
			setup: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "empty.txt")
				os.WriteFile(path, []byte{}, 0644)
				return path
			},
			expectData: []byte{},
			wantErr:    false,
		},
		{
			name: "read binary file",
			setup: func(tmpDir string) string {
				path := filepath.Join(tmpDir, "binary.dat")
				data := []byte{0x00, 0xFF, 0x42, 0x13, 0x37}
				os.WriteFile(path, data, 0644)
				return path
			},
			expectData: []byte{0x00, 0xFF, 0x42, 0x13, 0x37},
			wantErr:    false,
		},
		{
			name: "read non-existent file",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "nonexistent.txt")
			},
			wantErr: true,
			errMsg:  "file not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tc.setup(tmpDir)

			data, err := readFile(path)

			if tc.wantErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectData, data)
			}
		})
	}
}

func TestReadFileErrorMessages(t *testing.T) {
	tmpDir := t.TempDir()

	// Test file not found error
	_, err := readFile(filepath.Join(tmpDir, "notfound.txt"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")

	// Test permission error (on systems that support it)
	if os.Getuid() != 0 { // Don't run as root
		path := filepath.Join(tmpDir, "noperm.txt")
		os.WriteFile(path, []byte("content"), 0644)
		os.Chmod(path, 0000) // Remove all permissions
		defer os.Chmod(path, 0644) // Restore for cleanup

		_, err = readFile(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	}
}

func TestHelperFunctionsIntegration(t *testing.T) {
	// Test combining multiple helper functions
	tmpDir := t.TempDir()
	
	// Create a nested file using writeFile
	nestedPath := filepath.Join(tmpDir, "level1", "level2", "file.txt")
	data := []byte("integration test content")
	
	err := writeFile(nestedPath, data)
	require.NoError(t, err)
	
	// Verify file exists
	assert.True(t, fileExists(nestedPath))
	
	// Read the file back
	readData, err := readFile(nestedPath)
	require.NoError(t, err)
	assert.Equal(t, data, readData)
	
	// Append to the file
	appendData := []byte("\nappended line")
	err = appendToFile(nestedPath, appendData)
	require.NoError(t, err)
	
	// Read again and verify both contents
	finalData, err := readFile(nestedPath)
	require.NoError(t, err)
	
	expected := append(data, appendData...)
	assert.Equal(t, expected, finalData)
}

func TestHelperFunctionsConcurrency(t *testing.T) {
	// Test that helper functions work correctly under concurrent access
	tmpDir := t.TempDir()
	
	// Create multiple files concurrently
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			path := filepath.Join(tmpDir, fmt.Sprintf("concurrent_%d.txt", id))
			data := []byte(fmt.Sprintf("data from goroutine %d", id))
			
			err := writeFile(path, data)
			assert.NoError(t, err)
			
			// Verify the file
			assert.True(t, fileExists(path))
			
			readData, err := readFile(path)
			assert.NoError(t, err)
			assert.Equal(t, data, readData)
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// Verify all files were created
	for i := 0; i < numGoroutines; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("concurrent_%d.txt", i))
		assert.FileExists(t, path)
	}
}