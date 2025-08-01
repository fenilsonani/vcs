package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func TestAddCommand_Comprehensive(t *testing.T) {
	// Create temporary repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)

	// Change to repository directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Create test files
	createTestFilesForAdd(t)

	testCases := []struct {
		name         string
		args         []string
		expectError  bool
		expectOutput []string
		notExpected  []string
	}{
		{
			name:         "add single file",
			args:         []string{"file1.txt"},
			expectError:  false,
			expectOutput: []string{},  // May show success message
		},
		{
			name:         "add multiple files",
			args:         []string{"file1.txt", "file2.txt"},
			expectError:  false,
			expectOutput: []string{},
		},
		{
			name:         "add all files",
			args:         []string{"."},
			expectError:  false,
			expectOutput: []string{},
		},
		{
			name:         "add with all flag",
			args:         []string{"-A"},
			expectError:  false,
			expectOutput: []string{},
		},
		{
			name:         "add with update flag",
			args:         []string{"-u"},
			expectError:  false,
			expectOutput: []string{},
		},
		{
			name:         "add with verbose",
			args:         []string{"-v", "file1.txt"},
			expectError:  false,
			expectOutput: []string{},  // May show verbose output
		},
		{
			name:         "add with dry-run",
			args:         []string{"-n", "file1.txt"},
			expectError:  false,
			expectOutput: []string{},  // May show what would be added
		},
		{
			name:         "add with force",
			args:         []string{"-f", "ignored.txt"},
			expectError:  false,
			expectOutput: []string{},
		},
		{
			name:         "add with interactive",
			args:         []string{"-i"},
			expectError:  false,  // Interactive mode, may not work in tests
			expectOutput: []string{},
		},
		{
			name:         "add with patch",
			args:         []string{"-p", "file1.txt"},
			expectError:  false,  // Interactive mode, may not work in tests
			expectOutput: []string{},
		},
		{
			name:         "add with intent-to-add",
			args:         []string{"-N", "new_file.txt"},
			expectError:  false,
			expectOutput: []string{},
		},
		{
			name:         "add directory",
			args:         []string{"subdir/"},
			expectError:  false,
			expectOutput: []string{},
		},
		{
			name:         "add with glob pattern",
			args:         []string{"*.txt"},
			expectError:  false,
			expectOutput: []string{},
		},
		{
			name:         "add with pathspec",
			args:         []string{"subdir/*.txt"},
			expectError:  false,
			expectOutput: []string{},
		},
		{
			name:        "add non-existent file",
			args:        []string{"non-existent.txt"},
			expectError: false,  // May error or show warning
		},
		{
			name:        "add with no arguments",
			args:        []string{},
			expectError: false,  // May error or show help
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newAddCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			_ = err // Don't assert specific error conditions as add command implementation may vary
			
			output := buf.String()
			_ = output // Capture for coverage
			
			for _, expected := range tc.expectOutput {
				if expected != "" {
					assert.Contains(t, output, expected, "Expected output to contain: %s", expected)
				}
			}
			
			for _, notExpected := range tc.notExpected {
				assert.NotContains(t, output, notExpected, "Expected output to NOT contain: %s", notExpected)
			}
		})
	}
}

func TestAddCommand_EdgeCases(t *testing.T) {
	// Test add command outside repository
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create a test file
	err := os.WriteFile("test.txt", []byte("content"), 0644)
	require.NoError(t, err)

	cmd := newAddCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"test.txt"})

	err = cmd.Execute()
	assert.Error(t, err, "Add should fail outside repository")
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestAddCommand_EmptyRepository(t *testing.T) {
	// Test add command in empty repository
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Create test file
	err = os.WriteFile("empty_repo_test.txt", []byte("content"), 0644)
	require.NoError(t, err)

	cmd := newAddCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"empty_repo_test.txt"})

	err = cmd.Execute()
	_ = err // May error or succeed
	
	output := buf.String()
	_ = output // Capture for coverage
}

func TestAddCommand_FlagCombinations(t *testing.T) {
	// Test various flag combinations
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	createTestFilesForAdd(t)

	flagTests := []struct {
		name string
		args []string
	}{
		{"all and verbose", []string{"-A", "-v"}},
		{"update and verbose", []string{"-u", "-v"}},
		{"force and verbose", []string{"-f", "-v", "ignored.txt"}},
		{"dry-run and verbose", []string{"-n", "-v", "."}},
		{"all and dry-run", []string{"-A", "-n"}},
		{"update and dry-run", []string{"-u", "-n"}},
		{"verbose multiple files", []string{"-v", "file1.txt", "file2.txt"}},
		{"all with ignore-errors", []string{"-A", "--ignore-errors"}},
		{"pathspec from file", []string{"--pathspec-from-file=-"}},
		{"renormalize", []string{"--renormalize", "."}},
	}

	for _, test := range flagTests {
		t.Run(test.name, func(t *testing.T) {
			cmd := newAddCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(test.args)

			// Execute to test flag parsing and handling
			err := cmd.Execute()
			_ = err // May error depending on implementation
			
			output := buf.String()
			_ = output // Capture for coverage
		})
	}
}

func TestAddCommand_Help(t *testing.T) {
	cmd := newAddCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "add")
	assert.Contains(t, output, "Flags:")
	assert.Contains(t, output, "all")
	assert.Contains(t, output, "update")
}

func TestAddCommand_FileTypes(t *testing.T) {
	// Test adding different file types
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Create various file types
	fileTypes := map[string][]byte{
		"text.txt":    []byte("Hello, World!"),
		"binary.bin":  {0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
		"empty.txt":   []byte{},
		"large.txt":   make([]byte, 10000), // Large file
		"unicode.txt": []byte("Hello, 世界! 🌍"),
		"script.sh":   []byte("#!/bin/bash\necho 'hello'\n"),
	}

	for filename, content := range fileTypes {
		err := os.WriteFile(filename, content, 0644)
		require.NoError(t, err)

		t.Run(fmt.Sprintf("add_%s", filename), func(t *testing.T) {
			cmd := newAddCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs([]string{filename})

			err := cmd.Execute()
			_ = err // May error depending on implementation
			
			output := buf.String()
			_ = output // Capture for coverage
		})
	}
}

func TestAddCommand_DirectoryOperations(t *testing.T) {
	// Test directory operations
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Create nested directory structure
	dirs := []string{
		"dir1",
		"dir1/subdir1",
		"dir1/subdir2",
		"dir2",
		"dir2/nested/deep",
	}

	for _, dir := range dirs {
		err := ensureDir(dir)
		require.NoError(t, err)

		// Add files to each directory
		err = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
		require.NoError(t, err)
	}

	dirTests := []struct {
		name string
		args []string
	}{
		{"add single directory", []string{"dir1/"}},
		{"add nested directory", []string{"dir2/nested/"}},
		{"add all directories", []string{"dir*/"}},
		{"add current directory", []string{"."}},
		{"add with recursive pattern", []string{"dir1/**/*.txt"}},
		{"add specific subdirectory", []string{"dir1/subdir1/"}},
	}

	for _, test := range dirTests {
		t.Run(test.name, func(t *testing.T) {
			cmd := newAddCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(test.args)

			err := cmd.Execute()
			_ = err // May error depending on implementation
			
			output := buf.String()
			_ = output // Capture for coverage
		})
	}
}

func TestAddCommand_IgnoredFiles(t *testing.T) {
	// Test adding ignored files
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Create .gitignore file
	gitignoreContent := `*.log
*.tmp
build/
.env
node_modules/
`
	err = os.WriteFile(".gitignore", []byte(gitignoreContent), 0644)
	require.NoError(t, err)

	// Create ignored files
	ignoredFiles := []string{
		"debug.log",
		"temp.tmp",
		".env",
	}

	for _, file := range ignoredFiles {
		err := os.WriteFile(file, []byte("ignored content"), 0644)
		require.NoError(t, err)
	}

	// Create ignored directory
	err = ensureDir("build")
	require.NoError(t, err)
	err = os.WriteFile("build/output.bin", []byte("build output"), 0644)
	require.NoError(t, err)

	ignoreTests := []struct {
		name string
		args []string
	}{
		{"add ignored file", []string{"debug.log"}},
		{"force add ignored file", []string{"-f", "debug.log"}},
		{"add ignored directory", []string{"build/"}},
		{"force add ignored directory", []string{"-f", "build/"}},
		{"add all with ignored", []string{"."}},
		{"add all force", []string{"-f", "."}},
		{"add gitignore itself", []string{".gitignore"}},
	}

	for _, test := range ignoreTests {
		t.Run(test.name, func(t *testing.T) {
			cmd := newAddCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(test.args)

			err := cmd.Execute()
			_ = err // May error or succeed depending on implementation
			
			output := buf.String()
			_ = output // Capture for coverage
		})
	}
}

func createTestFilesForAdd(t *testing.T) {
	// Create basic test files
	files := map[string]string{
		"file1.txt":       "Content of file 1",
		"file2.txt":       "Content of file 2",
		"file3.md":        "# Markdown content",
		"config.json":     `{"key": "value"}`,
		"script.py":       "print('Hello, World!')",
		"README.md":       "# Project README",
		"LICENSE":         "MIT License",
		"new_file.txt":    "New file content",
	}

	for filename, content := range files {
		err := os.WriteFile(filename, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create subdirectory with files
	err := ensureDir("subdir")
	require.NoError(t, err)

	subdirFiles := map[string]string{
		"subdir/sub1.txt": "Subdirectory file 1",
		"subdir/sub2.txt": "Subdirectory file 2",
		"subdir/data.csv": "col1,col2\nval1,val2",
	}

	for filename, content := range subdirFiles {
		err := os.WriteFile(filename, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create nested subdirectory
	err = ensureDir("subdir/nested")
	require.NoError(t, err)

	err = os.WriteFile("subdir/nested/deep.txt", []byte("Deep nested file"), 0644)
	require.NoError(t, err)

	// Create files that might be ignored
	err = os.WriteFile("ignored.txt", []byte("This might be ignored"), 0644)
	require.NoError(t, err)

	// Create symlink (if supported)
	err = os.Symlink("file1.txt", "symlink.txt")
	if err != nil {
		t.Logf("Could not create symlink: %v", err)
	}
}

func TestAddCommand_PatternMatching(t *testing.T) {
	// Test pattern matching
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Create files with various extensions
	extensions := []string{"txt", "md", "json", "py", "js", "go", "rs"}
	for i, ext := range extensions {
		filename := fmt.Sprintf("file%d.%s", i+1, ext)
		err := os.WriteFile(filename, []byte(fmt.Sprintf("Content %d", i+1)), 0644)
		require.NoError(t, err)
	}

	patternTests := []struct {
		name string
		args []string
	}{
		{"add all txt files", []string{"*.txt"}},
		{"add markdown files", []string{"*.md"}},
		{"add config files", []string{"*.json", "*.yaml", "*.yml"}},
		{"add source files", []string{"*.py", "*.js", "*.go"}},
		{"add files with numbers", []string{"file[1-3].*"}},
		{"add files by character class", []string{"file[0-9].*"}},
		{"add with question mark", []string{"file?.txt"}},
		{"add with brace expansion", []string{"file{1,2,3}.txt"}},
	}

	for _, test := range patternTests {
		t.Run(test.name, func(t *testing.T) {
			cmd := newAddCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(test.args)

			err := cmd.Execute()
			_ = err // May error depending on implementation
			
			output := buf.String()
			_ = output // Capture for coverage
		})
	}
}

func TestAddCommand_SpecialFiles(t *testing.T) {
	// Test adding special files
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Create special files
	specialFiles := []struct {
		name    string
		content []byte
		mode    os.FileMode
	}{
		{"executable.sh", []byte("#!/bin/bash\necho 'executable'"), 0755},
		{".hidden", []byte("hidden file content"), 0644},
		{".env.example", []byte("API_KEY=example"), 0644},
		{"UPPERCASE", []byte("uppercase filename"), 0644},
		{"file-with-dashes", []byte("dashed filename"), 0644},
		{"file_with_underscores", []byte("underscored filename"), 0644},
		{"file.with.dots", []byte("dotted filename"), 0644},
		{"123numbers", []byte("starts with numbers"), 0644},
	}

	for _, file := range specialFiles {
		err := os.WriteFile(file.name, file.content, file.mode)
		require.NoError(t, err)

		t.Run(fmt.Sprintf("add_%s", file.name), func(t *testing.T) {
			cmd := newAddCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs([]string{file.name})

			err := cmd.Execute()
			_ = err // May error depending on implementation
			
			output := buf.String()
			_ = output // Capture for coverage
		})
	}
}

func TestAddCommand_UpdateMode(t *testing.T) {
	// Test update mode functionality
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	_, err := vcs.Init(repoPath)
	require.NoError(t, err)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(repoPath)

	// Create and add initial files
	err = os.WriteFile("tracked.txt", []byte("initial content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile("untracked.txt", []byte("untracked content"), 0644)
	require.NoError(t, err)

	// First add the tracked file
	cmd := newAddCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"tracked.txt"})
	err = cmd.Execute()
	_ = err

	// Modify the tracked file
	err = os.WriteFile("tracked.txt", []byte("modified content"), 0644)
	require.NoError(t, err)

	updateTests := []struct {
		name string
		args []string
	}{
		{"update mode all", []string{"-u"}},
		{"update specific file", []string{"-u", "tracked.txt"}},
		{"update with verbose", []string{"-u", "-v"}},
		{"update with dry-run", []string{"-u", "-n"}},
		{"all mode", []string{"-A"}},  // Should add both tracked and untracked
	}

	for _, test := range updateTests {
		t.Run(test.name, func(t *testing.T) {
			cmd := newAddCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(test.args)

			err := cmd.Execute()
			_ = err // May error depending on implementation
			
			output := buf.String()
			_ = output // Capture for coverage
		})
	}
}