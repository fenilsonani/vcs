package main

import (
	"path/filepath"
	"testing"
)

func TestInitCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "init current directory",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "init specified directory",
			args:    []string{"new-repo"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := NewTestHelper(t)
			defer helper.Cleanup()
			
			// For current directory test, change to temp dir
			if len(tt.args) == 0 {
				helper.ChDir()
			} else {
				// For specified directory, make it relative to temp dir
				tt.args[0] = filepath.Join(helper.TmpDir(), tt.args[0])
			}

			cmd := newInitCommand()
			result := helper.RunCommand(cmd, tt.args, nil)
			
			result.AssertError(t, tt.wantErr)
			if !tt.wantErr {
				result.AssertContains(t, "Initialized empty VCS repository")
			}
		})
	}
}

func TestHashObjectCommand(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()
	
	// Initialize repository
	helper.ChDir()
	initCmd := newInitCommand()
	initResult := helper.RunCommand(initCmd, []string{}, nil)
	initResult.AssertError(t, false)

	// Create test file
	helper.CreateFile("test.txt", "test content")

	tests := []struct {
		name    string
		args    []string
		stdin   string
		flags   map[string]string
		wantErr bool
		wantOut bool
	}{
		{
			name:    "hash file",
			args:    []string{"test.txt"},
			wantErr: false,
			wantOut: true, // Should output hash
		},
		{
			name:    "hash stdin",
			stdin:   "stdin content",
			flags:   map[string]string{"stdin": "true"},
			wantErr: false,
			wantOut: true,
		},
		{
			name:    "hash and write",
			args:    []string{"test.txt"},
			flags:   map[string]string{"write": "true"},
			wantErr: false,
			wantOut: true,
		},
		{
			name:    "unsupported type",
			args:    []string{"test.txt"},
			flags:   map[string]string{"type": "tree"},
			wantErr: true,
			wantOut: false,
		},
		{
			name:    "nonexistent file",
			args:    []string{"nonexistent.txt"},
			wantErr: true,
			wantOut: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set stdin if provided
			if tt.stdin != "" {
				helper.SetStdin(tt.stdin)
			}

			cmd := newHashObjectCommand()
			result := helper.RunCommand(cmd, tt.args, tt.flags)
			
			result.AssertError(t, tt.wantErr)
			if tt.wantOut && !tt.wantErr {
				if !result.HasOutput() {
					t.Error("Expected output but got none")
				}
			}
		})
	}
}

func TestCatFileCommand(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()
	
	// Initialize repository
	helper.ChDir()
	initCmd := newInitCommand()
	initResult := helper.RunCommand(initCmd, []string{}, nil)
	initResult.AssertError(t, false)

	// Create and hash an object to get a valid object ID
	helper.CreateFile("test.txt", "test content")
	
	hashCmd := newHashObjectCommand()
	hashResult := helper.RunCommand(hashCmd, []string{"test.txt"}, map[string]string{"write": "true"})
	hashResult.AssertError(t, false)
	
	objectID := hashResult.Output
	if objectID == "" {
		t.Fatal("No object ID returned from hash-object")
	}
	objectID = objectID[:40] // Take first 40 chars (SHA-1 hash length)

	tests := []struct {
		name      string
		args      []string
		flags     map[string]string
		wantErr   bool
		wantOut   bool
		wantEmpty bool
	}{
		{
			name:    "show type",
			args:    []string{objectID},
			flags:   map[string]string{"type": "true"},
			wantErr: false,
			wantOut: true,
		},
		{
			name:    "show size",
			args:    []string{objectID},
			flags:   map[string]string{"size": "true"},
			wantErr: false,
			wantOut: true,
		},
		{
			name:    "show content",
			args:    []string{objectID},
			flags:   map[string]string{"pretty-print": "true"},
			wantErr: false,
			wantOut: true,
		},
		{
			name:      "invalid object",
			args:      []string{"0000000000000000000000000000000000000000"},
			flags:     map[string]string{"type": "true"},
			wantErr:   true,
			wantEmpty: true,
		},
		{
			name:      "no flags",
			args:      []string{objectID},
			wantErr:   true,
			wantEmpty: true,
		},
		{
			name:      "missing object ID",
			args:      []string{},
			wantErr:   true,
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCatFileCommand()
			result := helper.RunCommand(cmd, tt.args, tt.flags)
			
			result.AssertError(t, tt.wantErr)
			
			if tt.wantOut && !tt.wantErr {
				if !result.HasOutput() {
					t.Error("Expected output but got none")
				}
			}
			
			if tt.wantEmpty && !result.HasOutput() {
				// This is expected for error cases
			}
		})
	}
}

