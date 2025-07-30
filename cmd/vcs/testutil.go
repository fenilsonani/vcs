package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestHelper provides utilities for testing CLI commands
type TestHelper struct {
	t         *testing.T
	tmpDir    string
	oldDir    string
	oldStdout *os.File
	oldStderr *os.File
	oldStdin  *os.File
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	tmpDir, err := os.MkdirTemp("", "vcs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	return &TestHelper{
		t:      t,
		tmpDir: tmpDir,
	}
}

// TmpDir returns the temporary directory path
func (h *TestHelper) TmpDir() string {
	return h.tmpDir
}

// Cleanup cleans up test resources
func (h *TestHelper) Cleanup() {
	if h.oldDir != "" {
		os.Chdir(h.oldDir)
	}
	if h.tmpDir != "" {
		os.RemoveAll(h.tmpDir)
	}
	h.restoreStdio()
}

// ChDir changes to the temporary directory
func (h *TestHelper) ChDir() {
	if h.oldDir == "" {
		h.oldDir, _ = os.Getwd()
	}
	if err := os.Chdir(h.tmpDir); err != nil {
		h.t.Fatalf("Failed to change directory: %v", err)
	}
}

// CreateFile creates a file with given content in the temp directory
func (h *TestHelper) CreateFile(name, content string) {
	if err := os.WriteFile(name, []byte(content), 0644); err != nil {
		h.t.Fatalf("Failed to create file %s: %v", name, err)
	}
}

// SetStdin sets the stdin content for the command
func (h *TestHelper) SetStdin(content string) {
	if h.oldStdin == nil {
		h.oldStdin = os.Stdin
	}
	
	r, w, err := os.Pipe()
	if err != nil {
		h.t.Fatalf("Failed to create pipe: %v", err)
	}
	
	os.Stdin = r
	
	go func() {
		defer w.Close()
		w.WriteString(content)
	}()
}

// restoreStdio restores original stdio
func (h *TestHelper) restoreStdio() {
	if h.oldStdout != nil {
		os.Stdout = h.oldStdout
		h.oldStdout = nil
	}
	if h.oldStderr != nil {
		os.Stderr = h.oldStderr
		h.oldStderr = nil
	}
	if h.oldStdin != nil {
		os.Stdin = h.oldStdin
		h.oldStdin = nil
	}
}

// CommandResult contains the result of a command execution
type CommandResult struct {
	Output string
	Error  error
}

// RunCommand runs a command and captures its output
func (h *TestHelper) RunCommand(cmd *cobra.Command, args []string, flags map[string]string) *CommandResult {
	// Set flags
	for key, value := range flags {
		if flag := cmd.Flags().Lookup(key); flag != nil {
			if err := cmd.Flags().Set(key, value); err != nil {
				h.t.Fatalf("Failed to set flag %s=%s: %v", key, value, err)
			}
		}
	}

	// Capture output using cmd.SetOut instead of os.Stdout redirection
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Run the command
	err := cmd.RunE(cmd, args)

	// Reset output streams
	cmd.SetOut(nil)
	cmd.SetErr(nil)

	return &CommandResult{
		Output: buf.String(),
		Error:  err,
	}
}

// AssertError checks if error expectation matches
func (r *CommandResult) AssertError(t *testing.T, wantErr bool) {
	if (r.Error != nil) != wantErr {
		t.Errorf("Command error = %v, wantErr %v", r.Error, wantErr)
	}
}

// AssertContains checks if output contains expected strings
func (r *CommandResult) AssertContains(t *testing.T, expected ...string) {
	for _, want := range expected {
		if !strings.Contains(r.Output, want) {
			t.Errorf("Output missing expected string %q\nGot: %s", want, r.Output)
		}
	}
}

// AssertNotContains checks if output does not contain unwanted strings
func (r *CommandResult) AssertNotContains(t *testing.T, unwanted ...string) {
	for _, dont := range unwanted {
		if strings.Contains(r.Output, dont) {
			t.Errorf("Output contains unwanted string %q\nGot: %s", dont, r.Output)
		}
	}
}

// AssertOutputEquals checks if output exactly matches expected
func (r *CommandResult) AssertOutputEquals(t *testing.T, expected string) {
	if strings.TrimSpace(r.Output) != strings.TrimSpace(expected) {
		t.Errorf("Output mismatch\nExpected: %q\nGot: %q", expected, r.Output)
	}
}

// AssertOutputEmpty checks if output is empty
func (r *CommandResult) AssertOutputEmpty(t *testing.T) {
	if strings.TrimSpace(r.Output) != "" {
		t.Errorf("Expected empty output, got: %q", r.Output)
	}
}

// HasOutput returns true if there is any output
func (r *CommandResult) HasOutput() bool {
	return strings.TrimSpace(r.Output) != ""
}