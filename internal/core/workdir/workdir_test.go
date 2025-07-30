package workdir

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fenilsonani/vcs/internal/core/objects"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusUntracked, "untracked"},
		{StatusModified, "modified"},
		{StatusAdded, "added"},
		{StatusDeleted, "deleted"},
		{StatusRenamed, "renamed"},
		{StatusIgnored, "ignored"},
		{Status(999), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("Status.String() = %v, want %v", got, tt.want)
		}
	}
}

func TestNewScanner(t *testing.T) {
	repoPath := "/test/repo"
	gitDir := "/test/repo/.git"
	
	scanner := NewScanner(repoPath, gitDir)
	
	if scanner.repoPath != repoPath {
		t.Errorf("NewScanner() repoPath = %v, want %v", scanner.repoPath, repoPath)
	}
	
	if scanner.gitDir != gitDir {
		t.Errorf("NewScanner() gitDir = %v, want %v", scanner.gitDir, gitDir)
	}
	
	if scanner.ignores == nil {
		t.Error("NewScanner() ignores is nil")
	}
}

func TestScanner_ScanWorkingDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workdir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test directory structure
	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	
	// Create test files
	testFiles := map[string]string{
		"file1.txt":        "content1",
		"subdir/file2.txt": "content2",
		"script.sh":        "#!/bin/bash\necho hello",
	}
	
	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		os.WriteFile(fullPath, []byte(content), 0644)
	}
	
	// Make script executable
	os.Chmod(filepath.Join(tmpDir, "script.sh"), 0755)
	
	scanner := NewScanner(tmpDir, gitDir)
	files, err := scanner.ScanWorkingDirectory()
	if err != nil {
		t.Fatalf("ScanWorkingDirectory() error = %v", err)
	}
	
	// Should find 4 items: 3 files + 1 subdir
	if len(files) != 4 {
		t.Errorf("ScanWorkingDirectory() found %d files, want 4", len(files))
	}
	
	// Check that .git directory is not included
	for _, file := range files {
		if file.Path == ".git" || filepath.HasPrefix(file.Path, ".git/") {
			t.Errorf("ScanWorkingDirectory() included .git path: %v", file.Path)
		}
	}
	
	// Verify file info
	fileMap := make(map[string]FileInfo)
	for _, file := range files {
		fileMap[file.Path] = file
	}
	
	if file, exists := fileMap["file1.txt"]; exists {
		if file.IsDir {
			t.Error("file1.txt should not be a directory")
		}
		if file.Size != 8 { // "content1" is 8 bytes
			t.Errorf("file1.txt size = %d, want 8", file.Size)
		}
	} else {
		t.Error("file1.txt not found")
	}
	
	if file, exists := fileMap["subdir"]; exists {
		if !file.IsDir {
			t.Error("subdir should be a directory")
		}
	} else {
		t.Error("subdir not found")
	}
}

func TestScanner_ScanFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workdir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test structure
	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file2.txt"), []byte("content"), 0644)
	
	scanner := NewScanner(tmpDir, gitDir)
	files, err := scanner.ScanFiles()
	if err != nil {
		t.Fatalf("ScanFiles() error = %v", err)
	}
	
	// Should find only files, not directories
	if len(files) != 2 {
		t.Errorf("ScanFiles() found %d files, want 2", len(files))
	}
	
	for _, file := range files {
		if file.IsDir {
			t.Errorf("ScanFiles() returned directory: %v", file.Path)
		}
	}
}

func TestScanner_GetFileContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workdir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := "test file content"
	testFile := "test.txt"
	os.WriteFile(filepath.Join(tmpDir, testFile), []byte(testContent), 0644)
	
	scanner := NewScanner(tmpDir, filepath.Join(tmpDir, ".git"))
	content, err := scanner.GetFileContent(testFile)
	if err != nil {
		t.Fatalf("GetFileContent() error = %v", err)
	}
	
	if string(content) != testContent {
		t.Errorf("GetFileContent() = %v, want %v", string(content), testContent)
	}
}

func TestScanner_GetFileContent_NotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workdir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scanner := NewScanner(tmpDir, filepath.Join(tmpDir, ".git"))
	_, err = scanner.GetFileContent("nonexistent.txt")
	if err == nil {
		t.Error("GetFileContent() should error for non-existent file")
	}
}

func TestScanner_GetFileMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workdir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create regular file
	regularFile := "regular.txt"
	os.WriteFile(filepath.Join(tmpDir, regularFile), []byte("content"), 0644)
	
	// Create executable file
	execFile := "script.sh"
	os.WriteFile(filepath.Join(tmpDir, execFile), []byte("#!/bin/bash"), 0755)
	
	scanner := NewScanner(tmpDir, filepath.Join(tmpDir, ".git"))
	
	// Test regular file
	mode, err := scanner.GetFileMode(regularFile)
	if err != nil {
		t.Fatalf("GetFileMode() error = %v", err)
	}
	if mode != objects.ModeBlob {
		t.Errorf("GetFileMode() = %v, want %v", mode, objects.ModeBlob)
	}
	
	// Test executable file
	mode, err = scanner.GetFileMode(execFile)
	if err != nil {
		t.Fatalf("GetFileMode() error = %v", err)
	}
	if mode != objects.ModeExec {
		t.Errorf("GetFileMode() = %v, want %v", mode, objects.ModeExec)
	}
}

func TestScanner_GetFileMode_NotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workdir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scanner := NewScanner(tmpDir, filepath.Join(tmpDir, ".git"))
	_, err = scanner.GetFileMode("nonexistent.txt")
	if err == nil {
		t.Error("GetFileMode() should error for non-existent file")
	}
}

func TestIgnorePatterns_AddPattern(t *testing.T) {
	ip := NewIgnorePatterns()
	
	tests := []struct {
		pattern string
		want    int
	}{
		{"*.txt", 1},
		{"  *.log  ", 2}, // should trim whitespace
		{"# comment", 2}, // should ignore comments
		{"", 2},          // should ignore empty lines
		{"build/", 3},
	}
	
	for _, tt := range tests {
		ip.AddPattern(tt.pattern)
		if len(ip.patterns) != tt.want {
			t.Errorf("After adding %q, patterns count = %d, want %d", tt.pattern, len(ip.patterns), tt.want)
		}
	}
}

func TestIgnorePatterns_LoadFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workdir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitignoreContent := `# This is a comment
*.log
*.tmp

# Another comment
build/
node_modules/
`
	
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	
	ip := NewIgnorePatterns()
	err = ip.LoadFile(gitignorePath)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	
	// Should have 4 patterns (comments and empty lines ignored)
	if len(ip.patterns) != 4 {
		t.Errorf("LoadFile() loaded %d patterns, want 4", len(ip.patterns))
	}
	
	expectedPatterns := []string{"*.log", "*.tmp", "build/", "node_modules/"}
	for i, expected := range expectedPatterns {
		if i >= len(ip.patterns) || ip.patterns[i] != expected {
			t.Errorf("Pattern %d = %v, want %v", i, ip.patterns[i], expected)
		}
	}
}

func TestIgnorePatterns_LoadFile_NotExists(t *testing.T) {
	ip := NewIgnorePatterns()
	err := ip.LoadFile("nonexistent.gitignore")
	if err != nil {
		t.Errorf("LoadFile() should not error for non-existent file, got: %v", err)
	}
}

func TestIgnorePatterns_Match(t *testing.T) {
	ip := NewIgnorePatterns()
	ip.AddPattern("*.log")
	ip.AddPattern("*.tmp")
	ip.AddPattern("build/")
	ip.AddPattern("node_modules/")
	ip.AddPattern("/config.json")
	ip.AddPattern("*.o")
	
	tests := []struct {
		path string
		want bool
	}{
		{"file.log", true},
		{"debug.log", true},
		{"logs/error.log", true},
		{"file.txt", false},
		{"build/output", true},
		{"src/build/output", true}, // should match build/ anywhere
		{"node_modules/package", true},
		{"config.json", true},
		{"src/config.json", false}, // /config.json should only match at root
		{"main.o", true},
		{"src/main.o", true},
		{"main.c", false},
		{"temp.tmp", true},
		{"data.tmp", true},
	}
	
	for _, tt := range tests {
		if got := ip.Match(tt.path); got != tt.want {
			t.Errorf("Match(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIgnorePatterns_wildcardMatch(t *testing.T) {
	ip := NewIgnorePatterns()
	
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"*.txt", "file.txt", true},
		{"*.txt", "file.log", false},
		{"test*", "test.txt", true},
		{"test*", "testing", true},
		{"test*", "file.txt", false},
		{"*test*", "mytest.txt", true},
		{"*test*", "testing", true},
		{"*test*", "file.txt", false},
		{"*.o", "main.o", true},
		{"*.o", "main.c", false},
	}
	
	for _, tt := range tests {
		if got := ip.wildcardMatch(tt.pattern, tt.path); got != tt.want {
			t.Errorf("wildcardMatch(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}

func TestScanner_LoadIgnoreFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workdir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gitignoreContent := "*.log\nbuild/\n"
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	
	scanner := NewScanner(tmpDir, filepath.Join(tmpDir, ".git"))
	err = scanner.LoadIgnoreFile(gitignorePath)
	if err != nil {
		t.Fatalf("LoadIgnoreFile() error = %v", err)
	}
	
	// Test that patterns were loaded
	if !scanner.IsIgnored("test.log") {
		t.Error("Scanner should ignore .log files")
	}
	
	if !scanner.IsIgnored("build/output") {
		t.Error("Scanner should ignore build/ directory")
	}
	
	if scanner.IsIgnored("test.txt") {
		t.Error("Scanner should not ignore .txt files")
	}
}

func TestScanner_IsIgnored(t *testing.T) {
	scanner := NewScanner("/test", "/test/.git")
	scanner.ignores.AddPattern("*.log")
	scanner.ignores.AddPattern("build/")
	
	tests := []struct {
		path string
		want bool
	}{
		{"file.log", true},
		{"file.txt", false},
		{"build/output", true},
		{"src/file.c", false},
	}
	
	for _, tt := range tests {
		if got := scanner.IsIgnored(tt.path); got != tt.want {
			t.Errorf("IsIgnored(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestScanner_FilterIgnored(t *testing.T) {
	scanner := NewScanner("/test", "/test/.git")
	scanner.ignores.AddPattern("*.log")
	scanner.ignores.AddPattern("*.tmp")
	
	files := []FileInfo{
		{Path: "file1.txt", Size: 10, ModTime: time.Now()},
		{Path: "file2.log", Size: 20, ModTime: time.Now()},
		{Path: "file3.tmp", Size: 30, ModTime: time.Now()},
		{Path: "file4.c", Size: 40, ModTime: time.Now()},
	}
	
	filtered := scanner.FilterIgnored(files)
	
	if len(filtered) != 2 {
		t.Errorf("FilterIgnored() returned %d files, want 2", len(filtered))
	}
	
	expectedPaths := map[string]bool{
		"file1.txt": true,
		"file4.c":   true,
	}
	
	for _, file := range filtered {
		if !expectedPaths[file.Path] {
			t.Errorf("FilterIgnored() included unexpected file: %v", file.Path)
		}
	}
}