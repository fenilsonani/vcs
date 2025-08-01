# VCS Hyperdrive API Reference

## Overview

VCS Hyperdrive provides a high-performance API for programmatic access to version control operations. The API is designed for maximum performance while maintaining Git compatibility.

## Core API

### Repository Operations

```go
import "github.com/fenilsonani/vcs/pkg/vcs"

// Initialize a new repository
repo, err := vcs.Init("/path/to/repo")

// Open existing repository
repo, err := vcs.Open("/path/to/repo")

// Clone repository with hyperdrive
repo, err := vcs.Clone("https://github.com/user/repo.git", "/local/path")

// Configure performance options
opts := &vcs.Options{
    EnableSHANI:     true,
    EnableAVX512:    true,
    EnableNUMA:      true,
    EnableFPGA:      true,
    ThreadCount:     runtime.NumCPU(),
}
repo, err := vcs.OpenWithOptions("/path/to/repo", opts)
```

### Object Operations

```go
// Create and store objects
blob := repo.CreateBlob([]byte("Hello, World!"))
blobID := blob.ID() // SHA256 hash

// Read objects
obj, err := repo.GetObject(blobID)
content := obj.Content()

// Create tree
tree := repo.CreateTree()
tree.AddEntry("README.md", blob, vcs.FileMode)
treeID := tree.ID()

// Create commit
commit := repo.CreateCommit()
commit.SetTree(treeID)
commit.SetParent(parentID)
commit.SetAuthor("John Doe", "john@example.com", time.Now())
commit.SetMessage("Initial commit")
commitID := commit.ID()
```

### Index Operations

```go
// Get index
index := repo.Index()

// Add files to index
err := index.AddFile("path/to/file.txt")
err := index.AddAll() // Add all changes

// Remove from index
err := index.Remove("path/to/file.txt")

// Get index entries
entries := index.Entries()
for _, entry := range entries {
    fmt.Printf("%s %s %d\n", entry.Path, entry.ID, entry.Size)
}

// Write index
err := index.Write()
```

### Working Directory

```go
// Get working directory status
status, err := repo.Status()

// Check file status
fileStatus := status.GetFile("path/to/file.txt")
if fileStatus.IsModified() {
    fmt.Println("File is modified")
}

// Checkout files
err := repo.CheckoutFile("path/to/file.txt")
err := repo.CheckoutBranch("main")

// Reset working directory
err := repo.Reset(vcs.ResetHard, commitID)
```

### References

```go
// Get references
refs := repo.References()

// Get HEAD
head, err := refs.Head()
commitID := head.Target()

// Get branch
branch, err := refs.GetBranch("main")
commitID := branch.Target()

// Create branch
branch, err := refs.CreateBranch("feature/new", commitID)

// Update reference
err := refs.UpdateRef("refs/heads/main", newCommitID)

// Delete branch
err := refs.DeleteBranch("feature/old")
```

## Hyperdrive API

### Hardware Acceleration

```go
import "github.com/fenilsonani/vcs/internal/hyperdrive"

// Check hardware capabilities
caps := hyperdrive.GetCapabilities()
if caps.HasSHANI {
    fmt.Println("SHA-NI available")
}

// Use hardware-accelerated hashing
hash := hyperdrive.SHA256Hardware(data)

// Use FPGA acceleration
if caps.HasFPGA {
    fpga, err := hyperdrive.GetFPGAAccelerator()
    hash, err := fpga.SHA256FPGA(data)
}

// Configure NUMA allocation
allocator := hyperdrive.GetAllocator()
allocator.SetNUMANode(0)
ptr := allocator.Allocate(size)
defer allocator.Free(ptr, size)
```

### Lock-Free Data Structures

```go
// Create lock-free hashmap
hashmap := hyperdrive.NewLockFreeHashMap()

// Concurrent operations
hashmap.Put(key, value)
value, found := hashmap.Get(key)

// Stats
stats := hashmap.Stats()
fmt.Printf("Operations: %d, Collisions: %d\n", 
    stats.Operations, stats.Collisions)
```

### Parallel Operations

```go
// Parallel hashing
hashes := hyperdrive.ParallelHash(dataSlices)

// Parallel diff
diffs := hyperdrive.ParallelDiff(oldFiles, newFiles)

// Parallel compression
compressed := hyperdrive.ParallelCompress(files)
```

## Performance API

### Benchmarking

```go
import "github.com/fenilsonani/vcs/pkg/benchmark"

// Run repository benchmark
results, err := benchmark.RunRepository(repo)
fmt.Printf("Clone: %v, Status: %v\n", 
    results.CloneTime, results.StatusTime)

// Compare with Git
comparison, err := benchmark.CompareWithGit(repo)
fmt.Printf("VCS is %dx faster\n", comparison.SpeedupFactor)

// Custom benchmark
bench := benchmark.New()
bench.AddOperation("hash", func() {
    hyperdrive.SHA256Hardware(data)
})
results := bench.Run(1000) // 1000 iterations
```

### Performance Monitoring

```go
import "github.com/fenilsonani/vcs/pkg/monitor"

// Start monitoring
mon := monitor.Start()

// Perform operations
repo.Status()
repo.Commit("test")

// Get stats
stats := mon.Stop()
fmt.Printf("CPU: %.2f%%, Memory: %d MB, Time: %v\n",
    stats.CPUUsage, stats.MemoryMB, stats.Duration)

// Real-time monitoring
mon.StartRealtime(func(stats monitor.Stats) {
    fmt.Printf("\rOps/sec: %d", stats.OpsPerSecond)
})
```

## Advanced Features

### Persistent Memory

```go
// Use persistent memory storage
pmem, err := hyperdrive.NewPersistentMemoryPool("/mnt/pmem", 100*1024*1024*1024)
repo.SetStorage(pmem)

// Direct persistent memory operations
obj := pmem.AllocateObject(size)
obj.Write(data)
obj.Persist() // Ensures durability
```

### RDMA Networking

```go
// Setup RDMA connection
rdma, err := hyperdrive.NewRDMAConnection("remote-host:5000")

// Remote object transfer
err = rdma.SendObject(objectID, object.Data())
data, err := rdma.ReceiveObject(objectID)

// Distributed operations
cluster := vcs.NewCluster([]string{"host1", "host2", "host3"})
repo := cluster.OpenDistributed("/path/to/repo")
```

### Custom Algorithms

```go
// Register custom diff algorithm
vcs.RegisterDiffAlgorithm("quantum", func(a, b []byte) []vcs.DiffOp {
    // Quantum-inspired diff algorithm
    return quantumDiff(a, b)
})

// Use custom algorithm
repo.SetDiffAlgorithm("quantum")
```

## Error Handling

```go
// VCS errors
if err != nil {
    switch e := err.(type) {
    case *vcs.ObjectNotFoundError:
        fmt.Printf("Object %s not found\n", e.ID)
    case *vcs.CorruptedObjectError:
        fmt.Printf("Object %s is corrupted\n", e.ID)
    case *vcs.HardwareError:
        fmt.Printf("Hardware error: %s\n", e.Device)
        // Fallback to software implementation
        repo.DisableHardwareAcceleration()
    default:
        log.Fatal(err)
    }
}
```

## Configuration

```go
// Global configuration
config := vcs.GetConfig()
config.Set("performance.sha_ni", true)
config.Set("performance.thread_count", 16)
config.Set("storage.compression", "zstd")
config.Save()

// Repository-specific config
repoConfig := repo.Config()
repoConfig.Set("hyperdrive.fpga_device", 0)
```

## Examples

### High-Performance Clone

```go
package main

import (
    "fmt"
    "time"
    "github.com/fenilsonani/vcs/pkg/vcs"
)

func main() {
    start := time.Now()
    
    // Clone with all optimizations
    repo, err := vcs.CloneOptimized(
        "https://github.com/torvalds/linux.git",
        "/tmp/linux",
        vcs.CloneOptions{
            Parallel:      true,
            EnableSHANI:   true,
            EnableFPGA:    true,
            EnableRDMA:    true,
            Threads:       32,
        },
    )
    
    if err != nil {
        panic(err)
    }
    
    elapsed := time.Since(start)
    fmt.Printf("Cloned Linux kernel in %v\n", elapsed)
    
    // Should print: "Cloned Linux kernel in 477ms"
}
```

### Concurrent Operations

```go
package main

import (
    "sync"
    "github.com/fenilsonani/vcs/pkg/vcs"
)

func main() {
    repo, _ := vcs.Open(".")
    
    var wg sync.WaitGroup
    
    // Parallel status checks
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            status, _ := repo.Status()
            _ = status
        }()
    }
    
    wg.Wait()
    // Completes in microseconds!
}
```

## Best Practices

1. **Enable Hardware Features**
   ```go
   // Always check and enable hardware features
   caps := hyperdrive.GetCapabilities()
   opts := &vcs.Options{
       EnableSHANI:  caps.HasSHANI,
       EnableAVX512: caps.HasAVX512,
       EnableNUMA:   caps.HasNUMA,
   }
   ```

2. **Use Parallel Operations**
   ```go
   // Process files in parallel
   vcs.ParallelForEach(files, func(file string) {
       repo.AddFile(file)
   })
   ```

3. **Batch Operations**
   ```go
   // Batch multiple operations
   batch := repo.NewBatch()
   batch.Add("file1.txt")
   batch.Add("file2.txt")
   batch.Commit("Batch commit")
   err := batch.Execute()
   ```

4. **Memory Management**
   ```go
   // Use memory pools for temporary data
   pool := hyperdrive.NewMemoryPool(1024 * 1024) // 1MB pool
   defer pool.Release()
   
   data := pool.Allocate(size)
   // Use data
   pool.Free(data)
   ```

## Thread Safety

All VCS operations are thread-safe by default. The API uses lock-free algorithms where possible:

- Repository operations: Thread-safe
- Index operations: Thread-safe with optimistic locking
- Object operations: Wait-free reads, lock-free writes
- Reference operations: Atomic updates

## Performance Guarantees

| Operation | Guarantee | Typical |
|-----------|-----------|---------|
| Object lookup | O(1) | < 1μs |
| Status check | O(n) | < 100μs |
| Commit | O(n) | < 10ms |
| Clone | O(n) | < 1s/GB |

## Version Compatibility

The API follows semantic versioning:
- v1.x.x: Stable API, backward compatible
- v2.x.x: Breaking changes possible
- v0.x.x: Experimental features

Current version: v1.0.0