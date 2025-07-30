package index

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fenilsonani/vcs/internal/core/objects"
)

func TestIndex_ErrorConditions(t *testing.T) {
	// Test ReadFrom with invalid data
	t.Run("ReadFrom with truncated entry", func(t *testing.T) {
		idx := New()
		
		// Create valid header but truncated entry
		var buf bytes.Buffer
		buf.WriteString("DIRC")                // Signature
		buf.Write([]byte{0, 0, 0, 2})          // Version 2
		buf.Write([]byte{0, 0, 0, 1})          // 1 entry
		buf.Write(make([]byte, 20))             // Partial entry data (too short)
		
		err := idx.ReadFrom(&buf)
		if err == nil {
			t.Error("ReadFrom() should fail with truncated entry")
		}
	})

	// Test ReadFrom with invalid entry count
	t.Run("ReadFrom with mismatched entry count", func(t *testing.T) {
		idx := New()
		
		// Create header claiming 1 entry but provide none
		var buf bytes.Buffer
		buf.WriteString("DIRC")                // Signature
		buf.Write([]byte{0, 0, 0, 2})          // Version 2
		buf.Write([]byte{0, 0, 0, 1})          // 1 entry (but none provided)
		// Add checksum
		checksum := make([]byte, 20)
		buf.Write(checksum)
		
		err := idx.ReadFrom(&buf)
		if err == nil {
			t.Error("ReadFrom() should fail with mismatched entry count")
		}
	})


	// Test with entry having invalid stage
	t.Run("Entry with invalid stage", func(t *testing.T) {
		entry := &Entry{Flags: 0xFFFF} // All flags set
		stage := entry.Stage()
		// Stage should be masked to valid range (0-3)
		if stage < 0 || stage > 3 {
			t.Errorf("Stage() = %d, should be in range 0-3", stage)
		}
	})

	// Test SetStage with out-of-range values
	t.Run("SetStage with invalid values", func(t *testing.T) {
		entry := &Entry{}
		
		// Test with negative stage - it uses bitwise operations so -1 becomes 3
		entry.SetStage(-1)
		actualStage := entry.Stage()
		if actualStage < 0 || actualStage > 3 {
			t.Errorf("SetStage(-1) stage = %d, should be in valid range 0-3", actualStage)
		}
		
		// Test with stage > 3 - should mask to 0-3 range
		entry.SetStage(5)
		actualStage = entry.Stage()
		if actualStage < 0 || actualStage > 3 {
			t.Errorf("SetStage(5) stage = %d, should be in valid range 0-3", actualStage)
		}
	})
}

func TestIndex_CorruptedData(t *testing.T) {
	// Test with corrupted signature
	t.Run("ReadFrom with corrupted signature", func(t *testing.T) {
		idx := New()
		data := []byte("XXXX\x00\x00\x00\x02\x00\x00\x00\x00")
		data = append(data, make([]byte, 20)...) // Add checksum
		
		err := idx.ReadFrom(bytes.NewReader(data))
		if err == nil {
			t.Error("ReadFrom() should fail with corrupted signature")
		}
		if !strings.Contains(err.Error(), "invalid index signature") {
			t.Errorf("ReadFrom() error = %v, want 'invalid index signature'", err)
		}
	})

	// Test with unsupported version
	t.Run("ReadFrom with version 1", func(t *testing.T) {
		idx := New()
		data := []byte("DIRC\x00\x00\x00\x01\x00\x00\x00\x00") // Version 1
		data = append(data, make([]byte, 20)...)                // Add checksum
		
		err := idx.ReadFrom(bytes.NewReader(data))
		if err == nil {
			t.Error("ReadFrom() should fail with unsupported version")
		}
		if !strings.Contains(err.Error(), "unsupported index version") {
			t.Errorf("ReadFrom() error = %v, want 'unsupported index version'", err)
		}
	})

}

func TestIndex_BoundaryConditions(t *testing.T) {
	// Test with maximum number of entries
	t.Run("Index with many entries", func(t *testing.T) {
		idx := New()
		
		// Add many entries
		for i := 0; i < 1000; i++ {
			entry := &Entry{
				Path: fmt.Sprintf("file%04d.txt", i),
				Mode: objects.ModeBlob,
				ID:   objects.ObjectID{byte(i), byte(i >> 8)},
			}
			if err := idx.Add(entry); err != nil {
				t.Fatalf("Add() entry %d error = %v", i, err)
			}
		}
		
		if len(idx.entries) != 1000 {
			t.Errorf("entries length = %d, want 1000", len(idx.entries))
		}
		
		// Test serialization/deserialization
		var buf bytes.Buffer
		if err := idx.WriteTo(&buf); err != nil {
			t.Fatalf("WriteTo() error = %v", err)
		}
		
		idx2 := New()
		if err := idx2.ReadFrom(&buf); err != nil {
			t.Fatalf("ReadFrom() error = %v", err)
		}
		
		if len(idx2.entries) != 1000 {
			t.Errorf("ReadFrom entries length = %d, want 1000", len(idx2.entries))
		}
	})

	// Test with entry having maximum path length
	t.Run("Entry with very long path", func(t *testing.T) {
		idx := New()
		
		// Create path with maximum reasonable length
		longPath := strings.Repeat("dir/", 100) + "file.txt"
		entry := &Entry{
			Path: longPath,
			Mode: objects.ModeBlob,
			ID:   objects.ObjectID{1, 2, 3},
		}
		
		if err := idx.Add(entry); err != nil {
			t.Errorf("Add() with long path should succeed, got error: %v", err)
		}
		
		// Test that we can retrieve it
		got, exists := idx.Get(longPath)
		if !exists {
			t.Error("Get() should find entry with long path")
		}
		if got.Path != longPath {
			t.Errorf("Get() path = %s, want %s", got.Path, longPath)
		}
	})

	// Test with entries having identical timestamps
	t.Run("Entries with identical timestamps", func(t *testing.T) {
		idx := New()
		now := time.Now().Truncate(time.Second)
		
		entries := []*Entry{
			{Path: "file1.txt", Mode: objects.ModeBlob, MTime: now, CTime: now},
			{Path: "file2.txt", Mode: objects.ModeBlob, MTime: now, CTime: now},
			{Path: "file3.txt", Mode: objects.ModeBlob, MTime: now, CTime: now},
		}
		
		for _, entry := range entries {
			if err := idx.Add(entry); err != nil {
				t.Fatalf("Add() error = %v", err)
			}
		}
		
		// All entries should be present
		for _, entry := range entries {
			if _, exists := idx.Get(entry.Path); !exists {
				t.Errorf("Get(%s) should find entry", entry.Path)
			}
		}
	})

	// Test with zero ObjectID
	t.Run("Entry with zero ObjectID", func(t *testing.T) {
		idx := New()
		entry := &Entry{
			Path: "zero-id.txt",
			Mode: objects.ModeBlob,
			ID:   objects.ObjectID{}, // Zero ID
		}
		
		if err := idx.Add(entry); err != nil {
			t.Errorf("Add() with zero ID should succeed, got error: %v", err)
		}
		
		got, exists := idx.Get("zero-id.txt")
		if !exists {
			t.Error("Get() should find entry with zero ID")
		}
		if !got.ID.IsZero() {
			t.Error("Entry ID should be zero")
		}
	})
}

func TestIndex_ConcurrencyAndThreadSafety(t *testing.T) {
	// Note: The current index implementation is not thread-safe
	// These tests document the current behavior
	
	t.Run("Multiple operations on same index", func(t *testing.T) {
		idx := New()
		
		// Add entries sequentially (simulating concurrent access patterns)
		for i := 0; i < 100; i++ {
			entry := &Entry{
				Path: fmt.Sprintf("concurrent%d.txt", i),
				Mode: objects.ModeBlob,
				ID:   objects.ObjectID{byte(i)},
			}
			if err := idx.Add(entry); err != nil {
				t.Fatalf("Add() entry %d error = %v", i, err)
			}
			
			// Immediately try to get it
			if _, exists := idx.Get(entry.Path); !exists {
				t.Errorf("Get() should find entry %d", i)
			}
		}
		
		// Final count should be correct
		if len(idx.entries) != 100 {
			t.Errorf("Final entries count = %d, want 100", len(idx.entries))
		}
	})
}