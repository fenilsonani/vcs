package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func isGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func BenchmarkGitComparison(b *testing.B) {
	if !isGitAvailable() {
		b.Skip("Git not available")
	}

	b.Run("Git_Init", func(b *testing.B) {
		tmpDir := b.TempDir()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gitRepo := filepath.Join(tmpDir, fmt.Sprintf("git-repo-%d", i))
			
			start := time.Now()
			cmd := exec.Command("git", "init", gitRepo)
			err := cmd.Run()
			elapsed := time.Since(start)
			
			if err != nil {
				b.Fatal(err)
			}
			
			b.ReportMetric(float64(elapsed.Nanoseconds()), "ns/op")
			os.RemoveAll(gitRepo) // Clean up
		}
	})
}