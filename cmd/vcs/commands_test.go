package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommand(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vcs-cmd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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
			args:    []string{filepath.Join(tmpDir, "repo")},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newInitCommand()
			
			// Change to temp directory if no args
			if len(tt.args) == 0 {
				oldDir, _ := os.Getwd()
				os.Chdir(tmpDir)
				defer os.Chdir(oldDir)
			}
			
			err := cmd.RunE(cmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunE() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHashObjectCommand(t *testing.T) {
	// Create temp repository
	tmpDir, err := os.MkdirTemp("", "vcs-cmd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repository
	initCmd := newInitCommand()
	initCmd.RunE(initCmd, []string{tmpDir})

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	// Change to repo directory
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	tests := []struct {
		name    string
		args    []string
		stdin   string
		flags   map[string]string
		wantErr bool
	}{
		{
			name:    "hash file",
			args:    []string{"test.txt"},
			wantErr: false,
		},
		{
			name:    "hash stdin",
			stdin:   "stdin content",
			flags:   map[string]string{"stdin": "true"},
			wantErr: false,
		},
		{
			name:    "hash and write",
			args:    []string{"test.txt"},
			flags:   map[string]string{"write": "true"},
			wantErr: false,
		},
		{
			name:    "unsupported type",
			args:    []string{"test.txt"},
			flags:   map[string]string{"type": "tree"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newHashObjectCommand()
			
			// Set flags
			for key, value := range tt.flags {
				cmd.Flags().Set(key, value)
			}
			
			// Set stdin if provided
			if tt.stdin != "" {
				oldStdin := os.Stdin
				r, w, _ := os.Pipe()
				os.Stdin = r
				w.WriteString(tt.stdin)
				w.Close()
				defer func() { os.Stdin = oldStdin }()
			}
			
			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			
			err := cmd.RunE(cmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunE() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if !tt.wantErr && buf.Len() == 0 {
				t.Error("Expected output but got none")
			}
		})
	}
}

func TestCatFileCommand(t *testing.T) {
	// Create temp repository
	tmpDir, err := os.MkdirTemp("", "vcs-cmd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repository
	initCmd := newInitCommand()
	initCmd.RunE(initCmd, []string{tmpDir})

	// Change to repo directory
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Create and hash an object
	hashCmd := newHashObjectCommand()
	hashCmd.Flags().Set("write", "true")
	hashCmd.Flags().Set("stdin", "true")
	
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.WriteString("test content")
	w.Close()
	
	var hashBuf bytes.Buffer
	hashCmd.SetOut(&hashBuf)
	hashCmd.RunE(hashCmd, []string{})
	os.Stdin = oldStdin
	
	objectID := strings.TrimSpace(hashBuf.String())

	tests := []struct {
		name    string
		args    []string
		flags   map[string]string
		wantErr bool
	}{
		{
			name:    "show type",
			args:    []string{objectID},
			flags:   map[string]string{"type": "true"},
			wantErr: false,
		},
		{
			name:    "show size",
			args:    []string{objectID},
			flags:   map[string]string{"size": "true"},
			wantErr: false,
		},
		{
			name:    "show content",
			args:    []string{objectID},
			flags:   map[string]string{"pretty-print": "true"},
			wantErr: false,
		},
		{
			name:    "invalid object",
			args:    []string{"0000000000000000000000000000000000000000"},
			flags:   map[string]string{"type": "true"},
			wantErr: true,
		},
		{
			name:    "no flags",
			args:    []string{objectID},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCatFileCommand()
			
			// Set flags
			for key, value := range tt.flags {
				cmd.Flags().Set(key, value)
			}
			
			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			
			err := cmd.RunE(cmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunE() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}