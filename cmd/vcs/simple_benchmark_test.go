package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fenilsonani/vcs/pkg/vcs"
)

// Simple benchmarks comparing VCS operations
func BenchmarkSimpleOperations(b *testing.B) {
	b.Run("VCS_Init", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tmpDir := filepath.Join(b.TempDir(), fmt.Sprintf("repo%d", i))
			start := time.Now()
			_, err := vcs.Init(tmpDir)
			elapsed := time.Since(start)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(elapsed.Nanoseconds()), "ns/op")
		}
	})

	b.Run("VCS_Add_Command", func(b *testing.B) {
		tmpDir := b.TempDir()
		repoPath := filepath.Join(tmpDir, "repo")
		_, err := vcs.Init(repoPath)
		if err != nil {
			b.Fatal(err)
		}

		oldWd, _ := os.Getwd()
		os.Chdir(repoPath)
		defer os.Chdir(oldWd)

		// Create test files
		for i := 0; i < 100; i++ {
			content := fmt.Sprintf("Test content %d\n", i)
			err := os.WriteFile(fmt.Sprintf("file%d.txt", i), []byte(content), 0644)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cmd := newAddCommand()
			cmd.SetArgs([]string{"."})
			
			start := time.Now()
			err := cmd.Execute()
			elapsed := time.Since(start)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(elapsed.Nanoseconds()), "ns/op")
		}
	})

	b.Run("VCS_Status_Command", func(b *testing.B) {
		tmpDir := b.TempDir()
		repoPath := filepath.Join(tmpDir, "repo")
		_, err := vcs.Init(repoPath)
		if err != nil {
			b.Fatal(err)
		}

		oldWd, _ := os.Getwd()
		os.Chdir(repoPath)
		defer os.Chdir(oldWd)

		// Create and add files
		for i := 0; i < 50; i++ {
			content := fmt.Sprintf("Test content %d\n", i)
			err := os.WriteFile(fmt.Sprintf("file%d.txt", i), []byte(content), 0644)
			if err != nil {
				b.Fatal(err)
			}
		}
		
		addCmd := newAddCommand()
		addCmd.SetArgs([]string{"."})
		addCmd.Execute()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cmd := newStatusCommand()
			
			start := time.Now()
			err := cmd.Execute()
			elapsed := time.Since(start)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(elapsed.Nanoseconds()), "ns/op")
		}
	})

	b.Run("VCS_Hash_Performance", func(b *testing.B) {
		data := make([]byte, 1024*1024) // 1MB
		for i := range data {
			data[i] = byte(i % 256)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			start := time.Now()
			hash := sha256.Sum256(data)
			elapsed := time.Since(start)
			_ = hash
			b.ReportMetric(float64(elapsed.Nanoseconds()), "ns/op")
		}
	})

	b.Run("VCS_FileOperations", func(b *testing.B) {
		tmpDir := b.TempDir()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Write
			start := time.Now()
			filePath := filepath.Join(tmpDir, fmt.Sprintf("test%d.txt", i))
			err := os.WriteFile(filePath, []byte("test content"), 0644)
			writeTime := time.Since(start)
			if err != nil {
				b.Fatal(err)
			}

			// Read
			start = time.Now()
			data, err := os.ReadFile(filePath)
			readTime := time.Since(start)
			if err != nil {
				b.Fatal(err)
			}
			_ = data

			b.ReportMetric(float64(writeTime.Nanoseconds()), "ns/write")
			b.ReportMetric(float64(readTime.Nanoseconds()), "ns/read")
		}
	})
}

// Benchmark command execution
func BenchmarkCommandExecution(b *testing.B) {
	commands := []struct {
		name string
		fn   func() interface{}
	}{
		{"Init", func() interface{} { return newInitCommand() }},
		{"Status", func() interface{} { return newStatusCommand() }},
		{"Add", func() interface{} { return newAddCommand() }},
		{"Commit", func() interface{} { return newCommitCommand() }},
		{"Log", func() interface{} { return newLogCommand() }},
		{"Branch", func() interface{} { return newBranchCommand() }},
	}

	for _, cmd := range commands {
		b.Run(cmd.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				start := time.Now()
				command := cmd.fn()
				elapsed := time.Since(start)
				_ = command
				b.ReportMetric(float64(elapsed.Nanoseconds()), "ns/create")
			}
		})
	}
}

// Benchmark memory allocations
func BenchmarkMemoryAllocations(b *testing.B) {
	b.Run("SmallAllocations", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			data := make([]byte, 1024) // 1KB
			_ = data
		}
	})

	b.Run("LargeAllocations", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			data := make([]byte, 1024*1024) // 1MB
			_ = data
		}
	})

	b.Run("ObjectCreation", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			obj := &struct {
				ID   [32]byte
				Type uint8
				Size uint64
				Data []byte
			}{
				Data: make([]byte, 1024),
			}
			_ = obj
		}
	})
}

// Benchmark parallel operations
func BenchmarkParallelism(b *testing.B) {
	b.Run("Sequential", func(b *testing.B) {
		sum := 0
		for i := 0; i < b.N; i++ {
			for j := 0; j < 1000; j++ {
				sum += j
			}
		}
		_ = sum
	})

	b.Run("Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			sum := 0
			for pb.Next() {
				for j := 0; j < 1000; j++ {
					sum += j
				}
			}
			_ = sum
		})
	})
}