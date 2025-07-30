package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fenilsonani/vcs/internal/core/index"
	"github.com/fenilsonani/vcs/internal/core/objects"
	"github.com/fenilsonani/vcs/pkg/vcs"
)

func BenchmarkRepositoryInit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tmpDir, err := os.MkdirTemp("", "bench-init-*")
		if err != nil {
			b.Fatal(err)
		}
		
		_, err = vcs.Init(tmpDir)
		if err != nil {
			b.Fatal(err)
		}
		
		os.RemoveAll(tmpDir)
	}
}

func BenchmarkBlobCreation(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-blob-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := vcs.Init(tmpDir)
	if err != nil {
		b.Fatal(err)
	}

	data := make([]byte, 1024) // 1KB file
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blob := repo.CreateBlobDirect(data)
		_ = blob
	}
}

func BenchmarkBlobCreationLarge(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-blob-large-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := vcs.Init(tmpDir)
	if err != nil {
		b.Fatal(err)
	}

	data := make([]byte, 1024*1024) // 1MB file
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blob := repo.CreateBlobDirect(data)
		_ = blob
	}
}

func BenchmarkHashComputation(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash := objects.ComputeHash(objects.TypeBlob, data)
		_ = hash
	}
}

func BenchmarkIndexOperations(b *testing.B) {
	idx := index.New()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := &index.Entry{
			Mode: objects.ModeBlob,
			Size: 1024,
			ID:   objects.ComputeHash(objects.TypeBlob, []byte(fmt.Sprintf("file%d", i))),
			Path: fmt.Sprintf("file%d.txt", i),
		}
		idx.Add(entry)
	}
}

func BenchmarkCommitCreation(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-commit-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := vcs.Init(tmpDir)
	if err != nil {
		b.Fatal(err)
	}

	// Create a simple tree
	blob := repo.CreateBlobDirect([]byte("test content"))
	tree, err := repo.CreateTree([]objects.TreeEntry{
		{Mode: objects.ModeBlob, Name: "test.txt", ID: blob.ID()},
	})
	if err != nil {
		b.Fatal(err)
	}

	sig := objects.Signature{
		Name:  "Benchmark",
		Email: "bench@example.com",
		When:  time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		message := fmt.Sprintf("Benchmark commit %d", i)
		_, err := repo.CreateCommit(tree.ID(), nil, sig, sig, message)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTreeCreation(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-tree-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := vcs.Init(tmpDir)
	if err != nil {
		b.Fatal(err)
	}

	// Pre-create blobs
	blobs := make([]*objects.Blob, 100)
	for i := range blobs {
		content := fmt.Sprintf("content for file %d", i)
		blobs[i] = repo.CreateBlobDirect([]byte(content))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entries := make([]objects.TreeEntry, len(blobs))
		for j, blob := range blobs {
			entries[j] = objects.TreeEntry{
				Mode: objects.ModeBlob,
				Name: fmt.Sprintf("file%d.txt", j),
				ID:   blob.ID(),
			}
		}
		
		_, err := repo.CreateTree(entries)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkObjectRetrieval(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-retrieve-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := vcs.Init(tmpDir)
	if err != nil {
		b.Fatal(err)
	}

	// Create objects to retrieve
	var objectIDs []objects.ObjectID
	for i := 0; i < 1000; i++ {
		content := fmt.Sprintf("test content %d", i)
		blob := repo.CreateBlobDirect([]byte(content))
		objectIDs = append(objectIDs, blob.ID())
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % len(objectIDs)
		_, err := repo.ReadObject(objectIDs[idx])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileOperations(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-file-ops-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo, err := vcs.Init(tmpDir)
	if err != nil {
		b.Fatal(err)
	}

	// Change to repo directory for file operations
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Create test files
	for i := 0; i < 100; i++ {
		filename := fmt.Sprintf("file%d.txt", i)
		content := fmt.Sprintf("content for file %d\nwith some additional lines\nto make it more realistic", i)
		err := os.WriteFile(filename, []byte(content), 0644)
		if err != nil {
			b.Fatal(err)
		}
	}

	idx := index.New()
	indexPath := filepath.Join(repo.GitDir(), "index")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		// Simulate git add operation
		for i := 0; i < 10; i++ { // Add 10 files per iteration
			filename := fmt.Sprintf("file%d.txt", i%100)
			content, err := os.ReadFile(filename)
			if err != nil {
				b.Fatal(err)
			}

			blob := repo.CreateBlobDirect(content)
			entry := &index.Entry{
				Mode: objects.ModeBlob,
				Size: uint32(len(content)),
				ID:   blob.ID(),
				Path: filename,
			}
			idx.Add(entry)
		}
		
		// Write index
		err := idx.WriteToFile(indexPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Run all benchmarks and show performance comparison
func BenchmarkAll(b *testing.B) {
	benchmarks := []struct {
		name string
		fn   func(*testing.B)
	}{
		{"RepoInit", BenchmarkRepositoryInit},
		{"BlobCreation", BenchmarkBlobCreation},
		{"HashComputation", BenchmarkHashComputation},
		{"IndexOps", BenchmarkIndexOperations},
		{"CommitCreation", BenchmarkCommitCreation},
		{"TreeCreation", BenchmarkTreeCreation},
		{"ObjectRetrieval", BenchmarkObjectRetrieval},
	}

	for _, bench := range benchmarks {
		b.Run(bench.name, bench.fn)
	}
}