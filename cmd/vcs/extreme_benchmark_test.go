package main

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/fenilsonani/vcs/internal/hyperdrive"
)

// BenchmarkPersistentMemory tests persistent memory performance
func BenchmarkPersistentMemory(b *testing.B) {
	tmpDir := b.TempDir()
	pmemFile := filepath.Join(tmpDir, "pmem.dat")

	pool, err := hyperdrive.NewPersistentMemoryPool(pmemFile, 256*1024*1024) // 256MB
	if err != nil {
		b.Skip("Persistent memory not available:", err)
	}
	defer pool.Close()

	b.Run("Allocate_Small", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ptr, err := pool.Allocate(64)
			if err != nil {
				b.Fatal(err)
			}
			pool.Free(ptr, 64)
		}
	})

	b.Run("Allocate_Large", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			ptr, err := pool.Allocate(1024 * 1024) // 1MB
			if err != nil {
				b.Fatal(err)
			}
			pool.Free(ptr, 1024*1024)
		}
	})

	b.Run("Store_Load_Object", func(b *testing.B) {
		data := make([]byte, 4096)
		rand.Read(data)

		obj := &hyperdrive.PMemObject{}
		copy(obj.ID[:], "test-object-123")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := pool.Store(obj, data)
			if err != nil {
				b.Fatal(err)
			}

			loaded, err := pool.Load(obj)
			if err != nil {
				b.Fatal(err)
			}
			_ = loaded
		}
	})

	b.Run("Transaction", func(b *testing.B) {
		data := make([]byte, 1024)
		rand.Read(data)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tx := pool.BeginTransaction()

			// Write multiple locations
			for j := 0; j < 10; j++ {
				offset := uint64(j * 1024)
				err := tx.Write(offset, data)
				if err != nil {
					b.Fatal(err)
				}
			}

			err := tx.Commit()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkRDMA tests RDMA networking performance
func BenchmarkRDMA(b *testing.B) {
	if runtime.GOOS != "linux" {
		b.Skip("RDMA only available on Linux")
	}

	err := hyperdrive.InitRDMA()
	if err != nil {
		b.Skip("RDMA not available:", err)
	}

	conn, err := hyperdrive.NewRDMAConnection("localhost:5000", "localhost:5001")
	if err != nil {
		b.Skip("Cannot create RDMA connection:", err)
	}
	defer conn.Close()

	b.Run("Connect", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := conn.Connect()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Register_Memory", func(b *testing.B) {
		data := make([]byte, 1024*1024) // 1MB
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mr, err := conn.RegisterMemory(unsafe.Pointer(&data[0]), uint64(len(data)), 0)
			if err != nil {
				b.Fatal(err)
			}
			_ = mr
		}
	})

	b.Run("Zero_Copy_Write", func(b *testing.B) {
		data := make([]byte, 1024*1024) // 1MB
		rand.Read(data)

		mr, _ := conn.RegisterMemory(unsafe.Pointer(&data[0]), uint64(len(data)), 0)

		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Use 0 for rkey in test
			err := conn.Write(unsafe.Pointer(&data[0]), 0, uint32(len(data)), 0)
			if err != nil {
				b.Fatal(err)
			}
		}
		_ = mr
	})
}

// BenchmarkDPDK tests DPDK networking performance
func BenchmarkDPDK(b *testing.B) {
	if runtime.GOOS != "linux" {
		b.Skip("DPDK only available on Linux")
	}

	err := hyperdrive.InitDPDK([]string{})
	if err != nil {
		b.Skip("DPDK not available:", err)
	}

	port, err := hyperdrive.NewDPDKPort(0, 4)
	if err != nil {
		b.Skip("Cannot create DPDK port:", err)
	}

	b.Run("Packet_Alloc_Free", func(b *testing.B) {
		packets := make([]*hyperdrive.DPDKPacket, 32)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			n := port.AllocPackets(packets)
			if n > 0 {
				port.FreePackets(packets[:n])
			}
		}
	})

	b.Run("Burst_Send_Recv", func(b *testing.B) {
		packets := make([]*hyperdrive.DPDKPacket, 32)
		n := port.AllocPackets(packets)
		if n == 0 {
			b.Skip("No packets allocated")
		}

		// Fill packets with data (skip on non-Linux)
		if runtime.GOOS == "linux" {
			// Only set fields on Linux where DPDKPacket has fields
		}

		b.SetBytes(int64(n * 64))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Send burst
			sent := port.SendBurst(0, packets[:n], n)
			_ = sent

			// Receive burst
			recv := port.RecvBurst(0, packets[:n])
			_ = recv
		}

		port.FreePackets(packets[:n])
	})
}

// BenchmarkTransactionalMemory tests HTM performance
func BenchmarkTransactionalMemory(b *testing.B) {
	tm := hyperdrive.NewTransactionalMap(1024)

	// Pre-populate map
	for i := uint64(0); i < 1000; i++ {
		val := i * 2
		tm.Put(i, unsafe.Pointer(&val))
	}

	b.Run("HTM_Get", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := uint64(0)
			for pb.Next() {
				_, found := tm.Get(i % 1000)
				if !found {
					b.Fatal("key not found")
				}
				i++
			}
		})
	})

	b.Run("HTM_Put", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := uint64(1000)
			for pb.Next() {
				val := i
				err := tm.Put(i, unsafe.Pointer(&val))
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})

	b.Run("HTM_Mixed_Operations", func(b *testing.B) {
		var reads, writes, deletes atomic.Uint64

		b.RunParallel(func(pb *testing.PB) {
			i := uint64(0)
			for pb.Next() {
				switch i % 10 {
				case 0, 1: // 20% writes
					val := i
					tm.Put(i%1000, unsafe.Pointer(&val))
					writes.Add(1)
				case 2: // 10% deletes
					tm.Delete(i % 1000)
					deletes.Add(1)
				default: // 70% reads
					tm.Get(i % 1000)
					reads.Add(1)
				}
				i++
			}
		})

		b.Logf("Operations - Reads: %d, Writes: %d, Deletes: %d",
			reads.Load(), writes.Load(), deletes.Load())

		// Report HTM statistics
		stats := tm.GetStats()
		b.Logf("HTM Stats - Started: %d, Committed: %d, Aborted: %d, Conflicts: %d",
			stats.Started.Load(), stats.Committed.Load(),
			stats.Aborted.Load(), stats.Conflicts.Load())
	})
}

// BenchmarkOptimisticLocking tests optimistic locking with HTM
func BenchmarkOptimisticLocking(b *testing.B) {
	lock := hyperdrive.NewOptimisticLock()
	sharedData := make([]int, 1000)

	b.Run("Optimistic_Read", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := lock.OptimisticRead(func() error {
					// Read shared data
					sum := 0
					for i := range sharedData {
						sum += sharedData[i]
					}
					_ = sum
					return nil
				})
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})

	b.Run("Optimistic_Write", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				err := lock.OptimisticWrite(func() error {
					// Write shared data
					sharedData[i%len(sharedData)]++
					return nil
				})
				if err != nil {
					b.Fatal(err)
				}
				i++
			}
		})
	})
}

// BenchmarkExtremeConcurrency tests extreme concurrency scenarios
func BenchmarkExtremeConcurrency(b *testing.B) {
	coreCounts := []int{1, 2, 4, 8, 16, 32, 64, 128}

	for _, cores := range coreCounts {
		if cores > runtime.NumCPU()*2 {
			continue
		}

		b.Run(fmt.Sprintf("Cores_%d", cores), func(b *testing.B) {
			var wg sync.WaitGroup
			var ops atomic.Uint64

			// Create workload
			workload := func() {
				defer wg.Done()
				allocator := hyperdrive.GetAllocator()

				for i := 0; i < b.N/cores; i++ {
					// Allocate
					ptr := allocator.Allocate(1024)

					// Do some work
					data := (*[1024]byte)(ptr)
					for j := range data {
						data[j] = byte(j)
					}

					// Hash
					hash := hyperdrive.UltraFastHash(data[:])
					_ = hash

					// Free
					allocator.Free(ptr, 1024)

					ops.Add(1)
				}
			}

			// Launch goroutines
			start := time.Now()
			for i := 0; i < cores; i++ {
				wg.Add(1)
				go workload()
			}

			wg.Wait()
			elapsed := time.Since(start)

			totalOps := ops.Load()
			opsPerSec := float64(totalOps) / elapsed.Seconds()
			b.Logf("Total ops: %d, Ops/sec: %.0f, Ops/sec/core: %.0f",
				totalOps, opsPerSec, opsPerSec/float64(cores))
		})
	}
}

// BenchmarkPerformanceSummary provides overall performance summary
func BenchmarkPerformanceSummary(b *testing.B) {
	b.Run("Complete_Stack", func(b *testing.B) {
		// Simulate complete operation using all optimizations
		allocator := hyperdrive.GetAllocator()

		start := time.Now()
		for i := 0; i < 1000; i++ {
			// 1. Allocate memory (NUMA-aware)
			ptr := allocator.Allocate(4096)
			data := (*[4096]byte)(ptr)

			// 2. Fill with random data
			for j := 0; j < 4096; j++ {
				data[j] = byte(j ^ i)
			}

			// 3. Hash using hardware acceleration
			hash := hyperdrive.UltraFastHash(data[:])

			// 4. Compress (simulated)
			compressed := hyperdrive.CompressUltraFast(data[:])

			// 5. Store to persistent memory (simulated)
			// 6. Network transfer (simulated)

			// 7. Free memory
			allocator.Free(ptr, 4096)

			_ = hash
			_ = compressed
		}
		elapsed := time.Since(start)

		b.Logf("\nComplete Stack Performance:")
		b.Logf("1000 operations in %v", elapsed)
		b.Logf("Average latency: %v per operation", elapsed/1000)
		b.Logf("Throughput: %.0f ops/sec", 1000/elapsed.Seconds())

		// Memory statistics
		stats := allocator.Stats()
		b.Logf("\nMemory Statistics:")
		b.Logf("Total Allocated: %d MB", stats.TotalAllocated/1024/1024)
		b.Logf("Total Freed: %d MB", stats.TotalFreed/1024/1024)
		b.Logf("Pool Count: %d", stats.PoolCount)
	})
}