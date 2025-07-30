package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureDir(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name: "create new directory",
			path: filepath.Join(tmpDir, "new-dir"),
		},
		{
			name: "create nested directories",
			path: filepath.Join(tmpDir, "parent", "child", "grandchild"),
		},
		{
			name: "existing directory",
			path: tmpDir,
		},
		{
			name: "directory with permissions",
			path: filepath.Join(tmpDir, "perm-dir"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ensureDir(tc.path)
			
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.DirExists(t, tc.path)
				
				// Check permissions
				info, err := os.Stat(tc.path)
				require.NoError(t, err)
				assert.True(t, info.IsDir())
				// Check that directory has at least read and execute permissions for owner
				mode := info.Mode()
				assert.True(t, mode&0500 == 0500)
			}
		})
	}
}

func TestWriteFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		path        string
		data        []byte
		expectError bool
		checkFunc   func(t *testing.T, path string)
	}{
		{
			name: "write new file",
			path: filepath.Join(tmpDir, "test.txt"),
			data: []byte("test content"),
			checkFunc: func(t *testing.T, path string) {
				content, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, "test content", string(content))
			},
		},
		{
			name: "overwrite existing file",
			path: filepath.Join(tmpDir, "existing.txt"),
			data: []byte("new content"),
			checkFunc: func(t *testing.T, path string) {
				// Pre-create file
				err := os.WriteFile(path, []byte("old content"), 0644)
				require.NoError(t, err)
				
				// Write new content
				err = writeFile(path, []byte("new content"))
				require.NoError(t, err)
				
				// Check content
				content, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, "new content", string(content))
			},
		},
		{
			name: "write empty file",
			path: filepath.Join(tmpDir, "empty.txt"),
			data: []byte{},
			checkFunc: func(t *testing.T, path string) {
				info, err := os.Stat(path)
				require.NoError(t, err)
				assert.Equal(t, int64(0), info.Size())
			},
		},
		{
			name: "write binary data",
			path: filepath.Join(tmpDir, "binary.dat"),
			data: []byte{0x00, 0xFF, 0x42, 0x13, 0x37},
			checkFunc: func(t *testing.T, path string) {
				content, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, []byte{0x00, 0xFF, 0x42, 0x13, 0x37}, content)
			},
		},
		{
			name:        "write to non-existent directory",
			path:        filepath.Join(tmpDir, "nonexistent", "test.txt"),
			data:        []byte("test"),
			expectError: false, // writeFile creates parent directories
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Run pre-check if needed
			if tc.checkFunc != nil && tc.name == "overwrite existing file" {
				tc.checkFunc(t, tc.path)
				return
			}
			
			err := writeFile(tc.path, tc.data)
			
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.FileExists(t, tc.path)
				
				// Check permissions
				info, err := os.Stat(tc.path)
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(0644), info.Mode())
				
				// Run check function
				if tc.checkFunc != nil {
					tc.checkFunc(t, tc.path)
				}
			}
			
			// Verify temp file doesn't exist
			tmpPath := tc.path + ".tmp"
			assert.NoFileExists(t, tmpPath)
		})
	}
}

func TestAppendToFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		path         string
		initialData  string
		appendData   []byte
		expectedData string
		expectError  bool
	}{
		{
			name:         "append to existing file",
			path:         filepath.Join(tmpDir, "append.txt"),
			initialData:  "Hello",
			appendData:   []byte(" World"),
			expectedData: "Hello World",
		},
		{
			name:         "append to new file",
			path:         filepath.Join(tmpDir, "new-append.txt"),
			initialData:  "",
			appendData:   []byte("First line"),
			expectedData: "First line",
		},
		{
			name:         "append newline",
			path:         filepath.Join(tmpDir, "newline.txt"),
			initialData:  "Line 1",
			appendData:   []byte("\nLine 2"),
			expectedData: "Line 1\nLine 2",
		},
		{
			name:         "append empty data",
			path:         filepath.Join(tmpDir, "empty-append.txt"),
			initialData:  "Content",
			appendData:   []byte{},
			expectedData: "Content",
		},
		{
			name:        "append to non-existent directory",
			path:        filepath.Join(tmpDir, "nonexistent", "append.txt"),
			appendData:  []byte("test"),
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create initial file if needed
			if tc.initialData != "" && !tc.expectError {
				err := os.WriteFile(tc.path, []byte(tc.initialData), 0644)
				require.NoError(t, err)
			}
			
			// Append data
			err := appendToFile(tc.path, tc.appendData)
			
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Read and verify content
				content, err := os.ReadFile(tc.path)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedData, string(content))
			}
		})
	}
}

func TestReadFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		path        string
		createFile  bool
		fileContent string
		expectError bool
	}{
		{
			name:        "read existing file",
			path:        filepath.Join(tmpDir, "read.txt"),
			createFile:  true,
			fileContent: "File content to read",
		},
		{
			name:        "read non-existent file",
			path:        filepath.Join(tmpDir, "nonexistent.txt"),
			createFile:  false,
			expectError: true,
		},
		{
			name:        "read empty file",
			path:        filepath.Join(tmpDir, "empty.txt"),
			createFile:  true,
			fileContent: "",
		},
		{
			name:        "read file with newlines",
			path:        filepath.Join(tmpDir, "multiline.txt"),
			createFile:  true,
			fileContent: "Line 1\nLine 2\nLine 3",
		},
		{
			name:        "read file with special characters",
			path:        filepath.Join(tmpDir, "special.txt"),
			createFile:  true,
			fileContent: "Special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create file if needed
			if tc.createFile {
				err := os.WriteFile(tc.path, []byte(tc.fileContent), 0644)
				require.NoError(t, err)
			}
			
			// Read file
			data, err := readFile(tc.path)
			
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.fileContent, string(data))
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		path           string
		createFile     bool
		createDir      bool
		expectedExists bool
	}{
		{
			name:           "existing file",
			path:           filepath.Join(tmpDir, "exists.txt"),
			createFile:     true,
			expectedExists: true,
		},
		{
			name:           "non-existent file",
			path:           filepath.Join(tmpDir, "not-exists.txt"),
			createFile:     false,
			expectedExists: false,
		},
		{
			name:           "directory (not a file)",
			path:           filepath.Join(tmpDir, "subdir"),
			createDir:      true,
			expectedExists: true, // fileExists returns true for directories too
		},
		{
			name:           "empty path",
			path:           "",
			expectedExists: false,
		},
		{
			name:           "path in non-existent directory",
			path:           filepath.Join(tmpDir, "nonexistent", "file.txt"),
			expectedExists: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create file or directory if needed
			if tc.createFile {
				err := os.WriteFile(tc.path, []byte("content"), 0644)
				require.NoError(t, err)
			}
			if tc.createDir {
				err := os.MkdirAll(tc.path, 0755)
				require.NoError(t, err)
			}
			
			// Check existence
			exists := fileExists(tc.path)
			assert.Equal(t, tc.expectedExists, exists)
		})
	}
}

func TestAtomicWrite(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	
	// Test atomic write behavior
	path := filepath.Join(tmpDir, "atomic.txt")
	
	// Write initial content
	err := writeFile(path, []byte("initial"))
	require.NoError(t, err)
	
	// Simulate partial write by creating temp file
	tmpPath := path + ".tmp"
	err = os.WriteFile(tmpPath, []byte("partial"), 0644)
	require.NoError(t, err)
	
	// Write new content (should handle existing temp file)
	err = writeFile(path, []byte("final"))
	require.NoError(t, err)
	
	// Verify final content
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "final", string(content))
	
	// Verify temp file is cleaned up
	assert.NoFileExists(t, tmpPath)
}

func TestConcurrentFileOperations(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	
	// Test concurrent appends
	path := filepath.Join(tmpDir, "concurrent.txt")
	
	// Create initial file
	err := writeFile(path, []byte("Start\n"))
	require.NoError(t, err)
	
	// Simulate multiple appends
	for i := 0; i < 5; i++ {
		data := []byte(fmt.Sprintf("Line %d\n", i))
		err := appendToFile(path, data)
		assert.NoError(t, err)
	}
	
	// Read and verify all lines are present
	content, err := readFile(path)
	require.NoError(t, err)
	
	lines := strings.Split(string(content), "\n")
	assert.Contains(t, lines[0], "Start")
	for i := 0; i < 5; i++ {
		assert.Contains(t, string(content), fmt.Sprintf("Line %d", i))
	}
}

func TestEdgeCases(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	
	t.Run("write with subdirectory creation", func(t *testing.T) {
		// Ensure parent directory exists
		parentDir := filepath.Join(tmpDir, "parent")
		err := ensureDir(parentDir)
		require.NoError(t, err)
		
		// Write file in subdirectory
		filePath := filepath.Join(parentDir, "file.txt")
		err = writeFile(filePath, []byte("content"))
		assert.NoError(t, err)
		assert.FileExists(t, filePath)
	})
	
	t.Run("handle permission errors", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Running as root, skipping permission test")
		}
		
		// Create read-only directory
		readOnlyDir := filepath.Join(tmpDir, "readonly")
		err := os.MkdirAll(readOnlyDir, 0755)
		require.NoError(t, err)
		
		// Create file in directory
		filePath := filepath.Join(readOnlyDir, "file.txt")
		err = writeFile(filePath, []byte("content"))
		require.NoError(t, err)
		
		// Make directory read-only
		err = os.Chmod(readOnlyDir, 0555)
		require.NoError(t, err)
		defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup
		
		// Try to write file (should fail)
		err = writeFile(filePath, []byte("new content"))
		assert.Error(t, err)
	})
	
	t.Run("large file operations", func(t *testing.T) {
		// Create large data (1MB)
		largeData := make([]byte, 1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		
		path := filepath.Join(tmpDir, "large.dat")
		
		// Write large file
		err := writeFile(path, largeData)
		assert.NoError(t, err)
		
		// Read and verify
		readData, err := readFile(path)
		assert.NoError(t, err)
		assert.Equal(t, largeData, readData)
		
		// Append to large file
		appendData := []byte("\nAppended line")
		err = appendToFile(path, appendData)
		assert.NoError(t, err)
		
		// Verify append
		finalData, err := readFile(path)
		assert.NoError(t, err)
		assert.Equal(t, len(largeData)+len(appendData), len(finalData))
	})
}