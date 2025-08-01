package hyperdrive

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// NUMANode represents a NUMA memory node
type NUMANode struct {
	id       int
	distance []int // Distance to other nodes
	memory   uint64 // Available memory in bytes
}

// MemoryPool represents a thread-local memory pool
type MemoryPool struct {
	smallBlocks  [64]*Block      // 8B to 512B (8B increments)
	mediumBlocks [64]*Block      // 512B to 32KB (512B increments)
	largeBlocks  sync.Map        // 32KB+ (custom sizes)
	node         int             // NUMA node affinity
	totalAlloc   atomic.Uint64   // Total allocated bytes
	totalFree    atomic.Uint64   // Total freed bytes
}

// Block represents a memory block in the pool
type Block struct {
	data     unsafe.Pointer
	size     uint32
	next     *Block
	inUse    atomic.Bool
	padding  [20]byte // Align to 64 bytes (cache line)
}

// UltraFastAllocator is a NUMA-aware memory allocator
type UltraFastAllocator struct {
	nodes        []NUMANode
	pools        sync.Map // map[goroutineID]*MemoryPool
	globalPool   *MemoryPool
	hugepages    bool
	cacheLineSize int
}

var (
	defaultAllocator *UltraFastAllocator
	allocatorOnce   sync.Once
)

// GetAllocator returns the global ultra-fast allocator
func GetAllocator() *UltraFastAllocator {
	allocatorOnce.Do(func() {
		defaultAllocator = NewUltraFastAllocator()
	})
	return defaultAllocator
}

// NewUltraFastAllocator creates a new NUMA-aware allocator
func NewUltraFastAllocator() *UltraFastAllocator {
	a := &UltraFastAllocator{
		cacheLineSize: 64, // Most modern CPUs
		hugepages:     detectHugepageSupport(),
	}
	a.detectNUMATopology()
	a.globalPool = a.newMemoryPool(0)
	return a
}

// Allocate allocates memory with NUMA awareness
func (a *UltraFastAllocator) Allocate(size int) unsafe.Pointer {
	if size <= 0 {
		return nil
	}

	// Get thread-local pool
	pool := a.getThreadPool()

	// Small allocation (8B - 512B)
	if size <= 512 {
		bucket := (size + 7) / 8 - 1
		if bucket < 64 {
			if block := pool.smallBlocks[bucket]; block != nil {
				if !block.inUse.Swap(true) {
					pool.totalAlloc.Add(uint64(size))
					return block.data
				}
			}
		}
	}

	// Medium allocation (512B - 32KB)
	if size <= 32*1024 {
		bucket := (size - 512 + 511) / 512
		if bucket < 64 {
			if block := pool.mediumBlocks[bucket]; block != nil {
				if !block.inUse.Swap(true) {
					pool.totalAlloc.Add(uint64(size))
					return block.data
				}
			}
		}
	}

	// Large allocation or fallback
	return a.allocateLarge(pool, size)
}

// AllocateAligned allocates cache-line aligned memory
func (a *UltraFastAllocator) AllocateAligned(size int) unsafe.Pointer {
	alignedSize := (size + a.cacheLineSize - 1) &^ (a.cacheLineSize - 1)
	ptr := a.Allocate(alignedSize + a.cacheLineSize)
	if ptr == nil {
		return nil
	}

	// Align to cache line
	aligned := (uintptr(ptr) + uintptr(a.cacheLineSize) - 1) &^ (uintptr(a.cacheLineSize) - 1)
	return unsafe.Pointer(aligned)
}

// AllocateHuge allocates memory using huge pages if available
func (a *UltraFastAllocator) AllocateHuge(size int) unsafe.Pointer {
	if !a.hugepages || size < 2*1024*1024 { // 2MB minimum for huge pages
		return a.Allocate(size)
	}

	// Allocate using mmap with MAP_HUGETLB
	return allocateHugepage(size)
}

// Free returns memory to the pool
func (a *UltraFastAllocator) Free(ptr unsafe.Pointer, size int) {
	if ptr == nil {
		return
	}

	pool := a.getThreadPool()
	pool.totalFree.Add(uint64(size))

	// Try to return to appropriate pool
	if size <= 512 {
		bucket := (size + 7) / 8 - 1
		if bucket < 64 {
			// Create new block and add to pool
			block := &Block{
				data: ptr,
				size: uint32(size),
			}
			block.next = pool.smallBlocks[bucket]
			pool.smallBlocks[bucket] = block
			return
		}
	}

	// For larger sizes, add to large pool
	pool.largeBlocks.Store(ptr, size)
}

// getThreadPool gets or creates a thread-local memory pool
func (a *UltraFastAllocator) getThreadPool() *MemoryPool {
	id := getGoroutineID()
	if pool, ok := a.pools.Load(id); ok {
		return pool.(*MemoryPool)
	}

	// Create new pool with NUMA affinity
	node := getCurrentNUMANode()
	pool := a.newMemoryPool(node)
	a.pools.Store(id, pool)
	return pool
}

// newMemoryPool creates a new memory pool
func (a *UltraFastAllocator) newMemoryPool(node int) *MemoryPool {
	return &MemoryPool{
		node: node,
	}
}

// allocateLarge handles large allocations
func (a *UltraFastAllocator) allocateLarge(pool *MemoryPool, size int) unsafe.Pointer {
	// Check large block cache
	var found unsafe.Pointer
	pool.largeBlocks.Range(func(key, value interface{}) bool {
		if blockSize := value.(int); blockSize >= size {
			found = key.(unsafe.Pointer)
			pool.largeBlocks.Delete(key)
			return false
		}
		return true
	})

	if found != nil {
		pool.totalAlloc.Add(uint64(size))
		return found
	}

	// Allocate new memory
	return allocateMemory(size, pool.node)
}

// detectNUMATopology detects NUMA topology
func (a *UltraFastAllocator) detectNUMATopology() {
	// On systems without NUMA, create single node
	a.nodes = []NUMANode{
		{
			id:       0,
			distance: []int{10}, // Local access latency
			memory:   getSystemMemory(),
		},
	}

	// On Linux, would read from /sys/devices/system/node/
	if runtime.GOOS == "linux" {
		// TODO: Parse NUMA topology from sysfs
	}
}

// MemoryStats returns allocator statistics
type MemoryStats struct {
	TotalAllocated uint64
	TotalFreed     uint64
	ActiveMemory   uint64
	PoolCount      int
	HugepagesUsed  uint64
}

// Stats returns current memory statistics
func (a *UltraFastAllocator) Stats() MemoryStats {
	stats := MemoryStats{}
	count := 0

	a.pools.Range(func(key, value interface{}) bool {
		pool := value.(*MemoryPool)
		stats.TotalAllocated += pool.totalAlloc.Load()
		stats.TotalFreed += pool.totalFree.Load()
		count++
		return true
	})

	stats.PoolCount = count
	stats.ActiveMemory = stats.TotalAllocated - stats.TotalFreed
	return stats
}

// Prefetch brings memory into CPU cache
func (a *UltraFastAllocator) Prefetch(ptr unsafe.Pointer, size int) {
	// Prefetch cache lines
	for offset := 0; offset < size; offset += a.cacheLineSize {
		addr := unsafe.Add(ptr, offset)
		prefetchT0(addr)
	}
}

// PrefetchWrite prefetches memory for writing
func (a *UltraFastAllocator) PrefetchWrite(ptr unsafe.Pointer, size int) {
	// Prefetch with intent to write
	for offset := 0; offset < size; offset += a.cacheLineSize {
		addr := unsafe.Add(ptr, offset)
		prefetchT0(addr) // Would use PREFETCHW on x86
	}
}

// Zero zeroes memory using optimized instructions
func (a *UltraFastAllocator) Zero(ptr unsafe.Pointer, size int) {
	// Use non-temporal stores for large buffers
	if size >= 32*1024 {
		zeroMemoryNonTemporal(ptr, size)
	} else {
		zeroMemory(ptr, size)
	}
}

// Platform-specific functions

func getGoroutineID() uint64 {
	// This is a hack - in production would use runtime support
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	var id uint64
	for i := 0; i < n; i++ {
		id = id*10 + uint64(buf[i])
	}
	return id
}

func getCurrentNUMANode() int {
	// Would use syscall to get current CPU and map to NUMA node
	return 0
}

func getSystemMemory() uint64 {
	// Would read from /proc/meminfo on Linux
	return 16 * 1024 * 1024 * 1024 // 16GB default
}

func detectHugepageSupport() bool {
	// Would check /sys/kernel/mm/transparent_hugepage/enabled
	return runtime.GOOS == "linux"
}

func allocateMemory(size int, node int) unsafe.Pointer {
	// Basic allocation - would use mmap with NUMA hints
	buf := make([]byte, size)
	return unsafe.Pointer(&buf[0])
}

func allocateHugepage(size int) unsafe.Pointer {
	// Would use mmap with MAP_HUGETLB
	return allocateMemory(size, 0)
}

func zeroMemory(ptr unsafe.Pointer, size int) {
	slice := (*[1 << 30]byte)(ptr)[:size:size]
	for i := range slice {
		slice[i] = 0
	}
}

func zeroMemoryNonTemporal(ptr unsafe.Pointer, size int) {
	// Would use MOVNTDQ on x86 or DC ZVA on ARM64
	zeroMemory(ptr, size)
}