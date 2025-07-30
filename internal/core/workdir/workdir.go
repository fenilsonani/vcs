package workdir

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fenilsonani/vcs/internal/core/objects"
)

// FileInfo represents a file in the working directory
type FileInfo struct {
	Path     string
	Size     int64
	Mode     os.FileMode
	ModTime  time.Time
	IsDir    bool
}

// Status represents the status of a file
type Status int

const (
	StatusUntracked Status = iota
	StatusModified
	StatusAdded
	StatusDeleted
	StatusRenamed
	StatusIgnored
)

func (s Status) String() string {
	switch s {
	case StatusUntracked:
		return "untracked"
	case StatusModified:
		return "modified"
	case StatusAdded:
		return "added"
	case StatusDeleted:
		return "deleted"
	case StatusRenamed:
		return "renamed"
	case StatusIgnored:
		return "ignored"
	default:
		return "unknown"
	}
}

// FileStatus represents the status of a file in the working directory
type FileStatus struct {
	Path         string
	Status       Status
	IndexStatus  Status
	WorkStatus   Status
}

// Scanner scans the working directory for changes
type Scanner struct {
	repoPath string
	gitDir   string
	ignores  *IgnorePatterns
}

// NewScanner creates a new working directory scanner
func NewScanner(repoPath, gitDir string) *Scanner {
	return &Scanner{
		repoPath: repoPath,
		gitDir:   gitDir,
		ignores:  NewIgnorePatterns(),
	}
}

// LoadIgnoreFile loads patterns from a .gitignore file
func (s *Scanner) LoadIgnoreFile(path string) error {
	return s.ignores.LoadFile(path)
}

// ScanWorkingDirectory scans the working directory and returns file info
func (s *Scanner) ScanWorkingDirectory() ([]FileInfo, error) {
	var files []FileInfo
	
	err := filepath.WalkDir(s.repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// Skip .git directory
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		
		// Get relative path from repo root
		relPath, err := filepath.Rel(s.repoPath, path)
		if err != nil {
			return err
		}
		
		// Skip root directory
		if relPath == "." {
			return nil
		}
		
		info, err := d.Info()
		if err != nil {
			return err
		}
		
		fileInfo := FileInfo{
			Path:    filepath.ToSlash(relPath),
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		}
		
		files = append(files, fileInfo)
		return nil
	})
	
	return files, err
}

// ScanFiles scans only files (not directories)
func (s *Scanner) ScanFiles() ([]FileInfo, error) {
	files, err := s.ScanWorkingDirectory()
	if err != nil {
		return nil, err
	}
	
	var fileList []FileInfo
	for _, file := range files {
		if !file.IsDir {
			fileList = append(fileList, file)
		}
	}
	
	return fileList, nil
}

// IsIgnored checks if a path should be ignored
func (s *Scanner) IsIgnored(path string) bool {
	return s.ignores.Match(path)
}

// FilterIgnored filters out ignored files from the list
func (s *Scanner) FilterIgnored(files []FileInfo) []FileInfo {
	var filtered []FileInfo
	for _, file := range files {
		if !s.IsIgnored(file.Path) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// GetFileContent reads the content of a file
func (s *Scanner) GetFileContent(path string) ([]byte, error) {
	fullPath := filepath.Join(s.repoPath, path)
	return os.ReadFile(fullPath)
}

// GetFileMode gets the file mode for a path
func (s *Scanner) GetFileMode(path string) (objects.FileMode, error) {
	fullPath := filepath.Join(s.repoPath, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return objects.ModeBlob, err
	}
	
	mode := info.Mode()
	if mode&0111 != 0 {
		return objects.ModeExec, nil
	}
	
	return objects.ModeBlob, nil
}

// IgnorePatterns manages .gitignore patterns
type IgnorePatterns struct {
	patterns []string
}

// NewIgnorePatterns creates a new ignore patterns manager
func NewIgnorePatterns() *IgnorePatterns {
	return &IgnorePatterns{
		patterns: make([]string, 0),
	}
}

// AddPattern adds a pattern to ignore
func (ip *IgnorePatterns) AddPattern(pattern string) {
	pattern = strings.TrimSpace(pattern)
	if pattern != "" && !strings.HasPrefix(pattern, "#") {
		ip.patterns = append(ip.patterns, pattern)
	}
}

// LoadFile loads patterns from a file
func (ip *IgnorePatterns) LoadFile(filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // .gitignore doesn't exist, that's OK
		}
		return err
	}
	
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		ip.AddPattern(line)
	}
	
	return nil
}

// Match checks if a path matches any ignore pattern
func (ip *IgnorePatterns) Match(path string) bool {
	path = filepath.ToSlash(path)
	
	for _, pattern := range ip.patterns {
		if ip.matchPattern(pattern, path) {
			return true
		}
	}
	return false
}

// matchPattern checks if a path matches a specific pattern
func (ip *IgnorePatterns) matchPattern(pattern, path string) bool {
	pattern = filepath.ToSlash(pattern)
	
	// Handle negation (patterns starting with !)
	if strings.HasPrefix(pattern, "!") {
		return false // TODO: implement negation properly
	}
	
	// Handle directory patterns (ending with /)
	if strings.HasSuffix(pattern, "/") {
		pattern = strings.TrimSuffix(pattern, "/")
		// Check if path starts with the pattern or contains it as a directory
		return strings.HasPrefix(path, pattern+"/") || path == pattern || strings.Contains(path, "/"+pattern+"/")
	}
	
	// Handle patterns starting with /
	if strings.HasPrefix(pattern, "/") {
		pattern = strings.TrimPrefix(pattern, "/")
		return ip.simpleMatch(pattern, path)
	}
	
	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		return ip.wildcardMatch(pattern, path)
	}
	
	// Simple substring match
	return strings.Contains(path, pattern) || path == pattern
}

// simpleMatch performs simple pattern matching
func (ip *IgnorePatterns) simpleMatch(pattern, path string) bool {
	if strings.Contains(pattern, "*") {
		return ip.wildcardMatch(pattern, path)
	}
	return path == pattern || strings.HasPrefix(path, pattern+"/")
}

// wildcardMatch performs wildcard pattern matching
func (ip *IgnorePatterns) wildcardMatch(pattern, path string) bool {
	// Simple wildcard matching - convert * to regexp equivalent
	// This is a simplified implementation
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return path == pattern
	}
	
	// Check if path starts with first part
	if parts[0] != "" && !strings.HasPrefix(path, parts[0]) {
		return false
	}
	
	// Check if path ends with last part
	if parts[len(parts)-1] != "" && !strings.HasSuffix(path, parts[len(parts)-1]) {
		return false
	}
	
	// For more complex patterns, we'd need proper regexp matching
	// This is a basic implementation
	current := path
	for i, part := range parts {
		if i == 0 {
			if part != "" {
				current = strings.TrimPrefix(current, part)
			}
			continue
		}
		
		if part == "" {
			continue
		}
		
		idx := strings.Index(current, part)
		if idx == -1 {
			return false
		}
		current = current[idx+len(part):]
	}
	
	return true
}