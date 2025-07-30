package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestNewHashObjectCommand(t *testing.T) {
	cmd := newHashObjectCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "hash-object", cmd.Use)
	assert.Contains(t, cmd.Short, "Compute object ID")
}

func TestHashObjectCommandDetailed(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		input       string
		setupFunc   func(t *testing.T, tmpDir string) string
		expectError bool
		checkFunc   func(t *testing.T, output string, repoPath string)
	}{
		{
			name: "hash file content",
			args: []string{"test.txt"},
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Create test file
				testFile := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(testFile, []byte("Hello, World!\n"), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				// Check output is valid SHA-1
				output = strings.TrimSpace(output)
				assert.Len(t, output, 40)
				assert.Regexp(t, "^[0-9a-f]{40}$", output)
				
				// Known hash for "Hello, World!\n"
				assert.Equal(t, "8ab686eafeb1f44702738c8b0f24f2567c36da6d", output)
			},
		},
		{
			name: "hash and write object",
			args: []string{"test.txt"},
			flags: map[string]string{
				"w": "true",
			},
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Initialize repository
				repoPath := filepath.Join(tmpDir, "repo")
				_, err := vcs.Init(repoPath)
				require.NoError(t, err)
				
				// Create test file
				testFile := filepath.Join(repoPath, "test.txt")
				err = os.WriteFile(testFile, []byte("test content"), 0644)
				require.NoError(t, err)
				
				return repoPath
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				// Check output is valid SHA-1
				output = strings.TrimSpace(output)
				assert.Len(t, output, 40)
				
				// Check object was written
				objectPath := filepath.Join(repoPath, ".git", "objects", output[:2], output[2:])
				assert.FileExists(t, objectPath)
			},
		},
		{
			name: "hash from stdin",
			args: []string{},
			flags: map[string]string{
				"stdin": "true",
			},
			input: "Content from stdin\n",
			checkFunc: func(t *testing.T, output string, repoPath string) {
				output = strings.TrimSpace(output)
				assert.Len(t, output, 40)
				// The hash will depend on git object format (blob <size>\0<content>)
				// Just verify it's a valid hash
				assert.Regexp(t, "^[0-9a-f]{40}$", output)
			},
		},
		{
			name: "hash with type",
			args: []string{"test.txt"},
			flags: map[string]string{
				"t": "blob",
			},
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Create test file
				testFile := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(testFile, []byte("test"), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				output = strings.TrimSpace(output)
				assert.Len(t, output, 40)
			},
		},
		{
			name: "hash with literally flag",
			args: []string{},
			flags: map[string]string{
				"literally": "true",
			},
			input: "literal content",
			checkFunc: func(t *testing.T, output string, repoPath string) {
				output = strings.TrimSpace(output)
				assert.Len(t, output, 40)
			},
		},
		{
			name: "hash multiple files",
			args: []string{"file1.txt", "file2.txt"},
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Create test files
				err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				assert.Len(t, lines, 2)
				for _, line := range lines {
					assert.Len(t, line, 40)
					assert.Regexp(t, "^[0-9a-f]{40}$", line)
				}
			},
		},
		{
			name: "hash with path flag",
			args: []string{},
			flags: map[string]string{
				"path": "custom/path.txt",
			},
			input: "path test",
			checkFunc: func(t *testing.T, output string, repoPath string) {
				output = strings.TrimSpace(output)
				assert.Len(t, output, 40)
			},
		},
		{
			name: "hash with no-filters flag",
			args: []string{"test.txt"},
			flags: map[string]string{
				"no-filters": "true",
			},
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Create test file
				testFile := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(testFile, []byte("test\r\n"), 0644) // CRLF line ending
				require.NoError(t, err)
				return tmpDir
			},
			checkFunc: func(t *testing.T, output string, repoPath string) {
				output = strings.TrimSpace(output)
				assert.Len(t, output, 40)
			},
		},
		{
			name:        "hash non-existent file",
			args:        []string{"nonexistent.txt"},
			expectError: true,
		},
		{
			name: "write object outside repository",
			args: []string{"test.txt"},
			flags: map[string]string{
				"w": "true",
			},
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Create test file but don't initialize repo
				testFile := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(testFile, []byte("test"), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			expectError: true,
		},
		{
			name:        "invalid type",
			args:        []string{"test.txt"},
			flags:       map[string]string{"t": "invalid"},
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Create test file
				testFile := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(testFile, []byte("test"), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			expectError: true,
		},
		{
			name:        "no file and no stdin",
			args:        []string{},
			expectError: false, // Will read from stdin by default
			input:       "default stdin content",
			checkFunc: func(t *testing.T, output string, repoPath string) {
				output = strings.TrimSpace(output)
				assert.Len(t, output, 40)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			
			// Run setup if provided
			var repoPath string
			if tc.setupFunc != nil {
				repoPath = tc.setupFunc(t, tmpDir)
			} else {
				repoPath = tmpDir
			}
			
			// Change to test directory
			err := os.Chdir(repoPath)
			require.NoError(t, err)
			
			// Create command
			cmd := newHashObjectCommand()
			
			// Set flags
			for flag, value := range tc.flags {
				err := cmd.Flags().Set(flag, value)
				require.NoError(t, err)
			}
			
			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			
			// Set stdin if provided
			if tc.input != "" {
				cmd.SetIn(strings.NewReader(tc.input))
			}
			
			// Execute command
			cmd.SetArgs(tc.args)
			err = cmd.Execute()
			
			// Check error
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Check output
				if tc.checkFunc != nil {
					tc.checkFunc(t, buf.String(), repoPath)
				}
			}
		})
	}
}

func TestHashObjectStdinBehavior(t *testing.T) {
	// Test reading from stdin without --stdin flag
	cmd := newHashObjectCommand()
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetIn(strings.NewReader("test input"))
	
	// Should error without --stdin flag
	err := cmd.Execute()
	assert.Error(t, err)
}

func TestHashObjectBatchMode(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create multiple files
	files := []struct {
		name    string
		content string
	}{
		{"file1.txt", "content1"},
		{"file2.txt", "content2"},
		{"file3.txt", "content3"},
	}
	
	for _, f := range files {
		err := os.WriteFile(f.name, []byte(f.content), 0644)
		require.NoError(t, err)
	}
	
	// Hash all files
	cmd := newHashObjectCommand()
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"file1.txt", "file2.txt", "file3.txt"})
	
	err = cmd.Execute()
	assert.NoError(t, err)
	
	// Check we got 3 hashes
	output := strings.TrimSpace(buf.String())
	lines := strings.Split(output, "\n")
	assert.Len(t, lines, 3)
}

func TestHashObjectWritePermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Running as root, skipping permission test")
	}
	
	// Create temporary directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	
	// Initialize repository
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)
	
	// Make objects directory read-only
	objectsDir := filepath.Join(repoPath, ".git", "objects")
	err = os.Chmod(objectsDir, 0555)
	require.NoError(t, err)
	defer os.Chmod(objectsDir, 0755)
	
	// Change to repo directory
	err = os.Chdir(repoPath)
	require.NoError(t, err)
	
	// Create test file
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)
	
	// Try to write object
	cmd := newHashObjectCommand()
	cmd.Flags().Set("w", "true")
	cmd.SetArgs([]string{"test.txt"})
	
	var buf bytes.Buffer
	cmd.SetErr(&buf)
	
	err = cmd.Execute()
	assert.Error(t, err)
}

func TestHashObjectKnownHashes(t *testing.T) {
	// Test known hash values to ensure compatibility
	tests := []struct {
		content      string
		expectedHash string
	}{
		{
			content:      "",
			expectedHash: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", // Empty blob
		},
		{
			content:      "Hello, World!\n",
			expectedHash: "8ab686eafeb1f44702738c8b0f24f2567c36da6d",
		},
		{
			content:      "test",
			expectedHash: "30d74d258442c7c65512eafab474568dd706c430",
		},
	}
	
	tmpDir := t.TempDir()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	for _, tc := range tests {
		t.Run(tc.content, func(t *testing.T) {
			// Create test file
			testFile := "test.txt"
			err := os.WriteFile(testFile, []byte(tc.content), 0644)
			require.NoError(t, err)
			
			// Hash it
			cmd := newHashObjectCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetArgs([]string{testFile})
			
			err = cmd.Execute()
			assert.NoError(t, err)
			
			output := strings.TrimSpace(buf.String())
			assert.Equal(t, tc.expectedHash, output)
		})
	}
}

func TestHashObjectLargeFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Create large file (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	
	testFile := "large.bin"
	err = os.WriteFile(testFile, largeContent, 0644)
	require.NoError(t, err)
	
	// Hash it
	cmd := newHashObjectCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{testFile})
	
	err = cmd.Execute()
	assert.NoError(t, err)
	
	output := strings.TrimSpace(buf.String())
	assert.Len(t, output, 40)
	assert.Regexp(t, "^[0-9a-f]{40}$", output)
}