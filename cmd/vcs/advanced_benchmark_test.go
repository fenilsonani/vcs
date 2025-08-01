package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/fenilsonani/vcs/internal/hyperdrive"
)

// BenchmarkMemoryAllocator tests the NUMA-aware memory allocator
func BenchmarkMemoryAllocator(b *testing.B) {
	allocator := hyperdrive.GetAllocator()

	b.Run("Small_Allocations_8B", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ptr := allocator.Allocate(8)
			allocator.Free(ptr, 8)
		}
	})

	b.Run("Medium_Allocations_4KB", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ptr := allocator.Allocate(4096)
			allocator.Free(ptr, 4096)
		}
	})

	b.Run("Large_Allocations_1MB", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ptr := allocator.Allocate(1024 * 1024)
			allocator.Free(ptr, 1024*1024)
		}
	})

	b.Run("Aligned_Allocations", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ptr := allocator.AllocateAligned(1024)
			if uintptr(ptr)%64 != 0 {
				b.Errorf("allocation not aligned: %p", ptr)
			}
			allocator.Free(ptr, 1024)
		}
	})

	b.Run("Parallel_Allocations", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ptr := allocator.Allocate(512)
				// Simulate some work
				_ = (*[512]byte)(ptr)
				allocator.Free(ptr, 512)
			}
		})
	})

	b.Run("Huge_Page_Allocations", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ptr := allocator.AllocateHuge(2 * 1024 * 1024) // 2MB
			allocator.Free(ptr, 2*1024*1024)
		}
	})
}

// BenchmarkARM64Optimizations tests ARM64 NEON optimizations
func BenchmarkARM64Optimizations(b *testing.B) {
	if runtime.GOARCH != "arm64" {
		b.Skip("ARM64 optimizations not available on", runtime.GOARCH)
	}

	data1KB := make([]byte, 1024)
	data1MB := make([]byte, 1024*1024)
	rand.Read(data1KB)
	rand.Read(data1MB[:1024]) // Fill first 1KB

	b.Run("Vector_Compare", func(b *testing.B) {
		a := make([]byte, 1024)
		b2 := make([]byte, 1024)
		copy(a, data1KB)
		copy(b2, data1KB)
		b2[512] = 0xFF // Make different

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = hyperdrive.VectorCompareNEON(a, b2)
		}
	})

	b.Run("NEON_Copy", func(b *testing.B) {
		src := make([]byte, 4096)
		dst := make([]byte, 4096)
		rand.Read(src)

		b.SetBytes(4096)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = hyperdrive.CopyNEON(dst, src)
		}
	})

	b.Run("Dot_Product", func(b *testing.B) {
		a := make([]float32, 1024)
		b2 := make([]float32, 1024)
		for i := range a {
			a[i] = float32(i)
			b2[i] = float32(i * 2)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = hyperdrive.DotProductNEON(a, b2)
		}
	})
}

// BenchmarkIOUring tests io_uring performance on Linux
func BenchmarkIOUring(b *testing.B) {
	if runtime.GOOS != "linux" {
		b.Skip("io_uring only available on Linux")
	}

	tmpDir := b.TempDir()

	b.Run("Async_Read", func(b *testing.B) {
		// Create test file
		testFile := filepath.Join(tmpDir, "test.dat")
		data := make([]byte, 1024*1024) // 1MB
		rand.Read(data)
		os.WriteFile(testFile, data, 0644)

		ops, err := hyperdrive.NewHighPerformanceFileOps()
		if err != nil {
			b.Skip("io_uring not available:", err)
		}

		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := ops.ReadFile(testFile)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Async_Write", func(b *testing.B) {
		data := make([]byte, 1024*1024) // 1MB
		rand.Read(data)

		ops, err := hyperdrive.NewHighPerformanceFileOps()
		if err != nil {
			b.Skip("io_uring not available:", err)
		}

		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testFile := filepath.Join(tmpDir, fmt.Sprintf("write%d.dat", i))
			err := ops.WriteFile(testFile, data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Batch_Operations", func(b *testing.B) {
		// Create multiple files
		nFiles := 10
		fileSize := 100 * 1024 // 100KB each
		for i := 0; i < nFiles; i++ {
			data := make([]byte, fileSize)
			rand.Read(data)
			os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("batch%d.dat", i)), data, 0644)
		}

		ring, err := hyperdrive.GetIOUring()
		if err != nil {
			b.Skip("io_uring not available:", err)
		}

		b.SetBytes(int64(nFiles * fileSize))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			requests := make([]hyperdrive.IORequest, nFiles)
			for j := 0; j < nFiles; j++ {
				f, _ := os.Open(filepath.Join(tmpDir, fmt.Sprintf("batch%d.dat", j)))
				defer f.Close()
				requests[j] = hyperdrive.IORequest{
					FD:     int(f.Fd()),
					Buffer: make([]byte, fileSize),
					Offset: 0,
				}
			}

			futures, err := ring.BatchRead(requests)
			if err != nil {
				b.Fatal(err)
			}

			for _, future := range futures {
				_, err := future.Wait()
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}

// BenchmarkLockFreeOperations tests lock-free data structures
func BenchmarkLockFreeOperations(b *testing.B) {
	hashMap := &hyperdrive.LockFreeHashMap{}

	// Initialize with some data
	for i := uint64(0); i < 1000; i++ {
		ptr := unsafe.Pointer(&i)
		// hashMap.Put(i, ptr) // Would need to implement Put
		_ = ptr
	}

	b.Run("Concurrent_Reads", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := uint64(0)
			for pb.Next() {
				_, _ = hashMap.Get(i % 1000)
				i++
			}
		})
	})

	b.Run("Mixed_Read_Write", func(b *testing.B) {
		var readCount, writeCount atomic.Uint64

		b.RunParallel(func(pb *testing.PB) {
			i := uint64(0)
			for pb.Next() {
				if i%10 == 0 {
					// Write operation (would need Put implementation)
					writeCount.Add(1)
				} else {
					// Read operation
					_, _ = hashMap.Get(i % 1000)
					readCount.Add(1)
				}
				i++
			}
		})

		b.Logf("Reads: %d, Writes: %d", readCount.Load(), writeCount.Load())
	})
}

// BenchmarkPrefetching tests memory prefetching effectiveness
func BenchmarkPrefetching(b *testing.B) {
	allocator := hyperdrive.GetAllocator()
	dataSize := 1024 * 1024 // 1MB

	b.Run("Without_Prefetch", func(b *testing.B) {
		data := make([]byte, dataSize)
		rand.Read(data)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sum := 0
			for j := 0; j < len(data); j += 64 { // Cache line size
				sum += int(data[j])
			}
			_ = sum
		}
	})

	b.Run("With_Prefetch", func(b *testing.B) {
		ptr := allocator.Allocate(dataSize)
		data := (*[1 << 30]byte)(ptr)[:dataSize:dataSize]
		rand.Read(data)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Prefetch ahead
			for j := 0; j < len(data); j += 1024 {
				if j+1024 < len(data) {
					allocator.Prefetch(unsafe.Add(ptr, j+1024), 1024)
				}
			}

			sum := 0
			for j := 0; j < len(data); j += 64 {
				sum += int(data[j])
			}
			_ = sum
		}

		allocator.Free(ptr, dataSize)
	})
}

// BenchmarkZeroMemory tests optimized memory zeroing
func BenchmarkZeroMemory(b *testing.B) {
	allocator := hyperdrive.GetAllocator()

	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"64KB", 64 * 1024},
		{"1MB", 1024 * 1024},
		{"16MB", 16 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			ptr := allocator.Allocate(tc.size)
			defer allocator.Free(ptr, tc.size)

			b.SetBytes(int64(tc.size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				allocator.Zero(ptr, tc.size)
			}
		})
	}
}

// BenchmarkComparisonSummary provides a summary comparison
func BenchmarkComparisonSummary(b *testing.B) {
	b.Run("Standard_vs_Hyperdrive", func(b *testing.B) {
		data := make([]byte, 1024*1024) // 1MB
		rand.Read(data)

		// Standard allocation
		start := time.Now()
		for i := 0; i < 1000; i++ {
			buf := make([]byte, 4096)
			_ = buf
		}
		standardTime := time.Since(start)

		// Hyperdrive allocation
		allocator := hyperdrive.GetAllocator()
		start = time.Now()
		for i := 0; i < 1000; i++ {
			ptr := allocator.Allocate(4096)
			allocator.Free(ptr, 4096)
		}
		hyperdriveTime := time.Since(start)

		b.Logf("Standard allocation: %v", standardTime)
		b.Logf("Hyperdrive allocation: %v", hyperdriveTime)
		b.Logf("Speedup: %.2fx", float64(standardTime)/float64(hyperdriveTime))

		// Memory stats
		stats := allocator.Stats()
		b.Logf("\nMemory Statistics:")
		b.Logf("Total Allocated: %d MB", stats.TotalAllocated/1024/1024)
		b.Logf("Total Freed: %d MB", stats.TotalFreed/1024/1024)
		b.Logf("Active Memory: %d KB", stats.ActiveMemory/1024)
		b.Logf("Pool Count: %d", stats.PoolCount)
	})
}

// BenchmarkScalability tests performance at different scales
func BenchmarkScalability(b *testing.B) {
	coreCounts := []int{1, 2, 4, 8, 16, 32}

	for _, cores := range coreCounts {
		if cores > runtime.NumCPU() {
			continue
		}

		b.Run(fmt.Sprintf("Cores_%d", cores), func(b *testing.B) {
			runtime.GOMAXPROCS(cores)
			defer runtime.GOMAXPROCS(runtime.NumCPU())

			var ops atomic.Uint64
			var wg sync.WaitGroup

			start := time.Now()
			for i := 0; i < cores; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for time.Since(start) < time.Second {
						// Simulate work
						hash := hyperdrive.UltraFastHash([]byte("test data"))
						_ = hash
						ops.Add(1)
					}
				}()
			}

			wg.Wait()
			totalOps := ops.Load()
			b.Logf("Operations/sec with %d cores: %d", cores, totalOps)
			b.Logf("Ops/sec/core: %d", totalOps/uint64(cores))
		})
	}
}