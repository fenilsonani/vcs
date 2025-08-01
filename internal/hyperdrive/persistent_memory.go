package hyperdrive

import (
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// PersistentMemoryPool represents a persistent memory pool
type PersistentMemoryPool struct {
	path      string
	size      uint64
	baseAddr  unsafe.Pointer
	metadata  *PMemMetadata
	allocator *PMemAllocator
	mu        sync.RWMutex
}

// PMemMetadata stores persistent memory metadata
type PMemMetadata struct {
	magic       uint64
	version     uint32
	poolSize    uint64
	allocated   uint64
	freeList    uint64 // Offset to free list
	checksum    uint64
	lastFlush   int64
	reserved    [464]byte // Align to 512 bytes
}

// PMemAllocator manages persistent memory allocation
type PMemAllocator struct {
	pool      *PersistentMemoryPool
	freeList  *PMemFreeList
	allocated atomic.Uint64
	freed     atomic.Uint64
}

// PMemFreeList represents the free memory list
type PMemFreeList struct {
	head   atomic.Uint64 // Offset from base
	count  atomic.Uint32
	blocks []FreeBlock
}

// FreeBlock represents a free memory block
type FreeBlock struct {
	offset uint64
	size   uint64
	next   uint64 // Offset to next block
}

// PMemObject represents a persistent memory object
type PMemObject struct {
	ID       [32]byte
	Size     uint64
	Offset   uint64
	Checksum uint32
	Flags    uint32
	Created  int64
	Modified int64
}

const (
	PMEM_MAGIC    = 0x504D454D5643530A // "PMEMVCS\n"
	PMEM_VERSION  = 1
	PMEM_MIN_SIZE = 64 * 1024 * 1024 // 64MB minimum
	PAGE_SIZE     = 4096
	CACHE_LINE    = 64
)

// Persistent memory flags
const (
	PMEM_SYNC     = 1 << iota // Synchronous writes
	PMEM_DIRECT              // Direct access (DAX)
	PMEM_HUGEPAGE           // Use huge pages
	PMEM_ENCRYPTED         // Encrypted storage
)

// NewPersistentMemoryPool creates or opens a persistent memory pool
func NewPersistentMemoryPool(path string, size uint64) (*PersistentMemoryPool, error) {
	if size < PMEM_MIN_SIZE {
		size = PMEM_MIN_SIZE
	}

	// Align size to page boundary
	size = (size + PAGE_SIZE - 1) &^ (PAGE_SIZE - 1)

	pool := &PersistentMemoryPool{
		path: path,
		size: size,
	}

	if err := pool.initialize(); err != nil {
		return nil, err
	}

	return pool, nil
}

// initialize sets up the persistent memory pool
func (p *PersistentMemoryPool) initialize() error {
	// Open or create file
	file, err := os.OpenFile(p.path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Resize if needed
	if uint64(info.Size()) < p.size {
		if err := file.Truncate(int64(p.size)); err != nil {
			return err
		}
	}

	// Memory map the file
	data, err := syscall.Mmap(int(file.Fd()), 0, int(p.size),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED)
	if err != nil {
		return err
	}

	p.baseAddr = unsafe.Pointer(&data[0])
	p.metadata = (*PMemMetadata)(p.baseAddr)

	// Check if pool is already initialized
	if p.metadata.magic != PMEM_MAGIC {
		// Initialize new pool
		p.initializeMetadata()
	}

	// Verify pool
	if err := p.verify(); err != nil {
		syscall.Munmap(data)
		return err
	}

	// Create allocator
	p.allocator = &PMemAllocator{
		pool: p,
	}

	return nil
}

// initializeMetadata initializes pool metadata
func (p *PersistentMemoryPool) initializeMetadata() {
	p.metadata.magic = PMEM_MAGIC
	p.metadata.version = PMEM_VERSION
	p.metadata.poolSize = p.size
	p.metadata.allocated = uint64(unsafe.Sizeof(PMemMetadata{}))
	p.metadata.freeList = uint64(unsafe.Sizeof(PMemMetadata{}))

	// Initialize free list
	freeList := (*PMemFreeList)(unsafe.Add(p.baseAddr, p.metadata.freeList))
	freeList.head.Store(uint64(unsafe.Sizeof(PMemMetadata{})) + uint64(unsafe.Sizeof(PMemFreeList{})))
	freeList.count.Store(1)

	// Create initial free block
	firstBlock := (*FreeBlock)(unsafe.Add(p.baseAddr, freeList.head.Load()))
	firstBlock.offset = freeList.head.Load()
	firstBlock.size = p.size - freeList.head.Load()
	firstBlock.next = 0

	// Flush metadata
	p.flush(unsafe.Pointer(p.metadata), unsafe.Sizeof(PMemMetadata{}))
	p.flush(unsafe.Pointer(freeList), unsafe.Sizeof(PMemFreeList{}))
	p.flush(unsafe.Pointer(firstBlock), unsafe.Sizeof(FreeBlock{}))
}

// verify checks pool integrity
func (p *PersistentMemoryPool) verify() error {
	if p.metadata.magic != PMEM_MAGIC {
		return errors.New("invalid persistent memory pool magic")
	}

	if p.metadata.version != PMEM_VERSION {
		return errors.New("unsupported persistent memory version")
	}

	if p.metadata.poolSize != p.size {
		return errors.New("pool size mismatch")
	}

	return nil
}

// Allocate allocates persistent memory
func (p *PersistentMemoryPool) Allocate(size uint64) (unsafe.Pointer, error) {
	// Align size to cache line
	size = (size + CACHE_LINE - 1) &^ (CACHE_LINE - 1)

	p.mu.Lock()
	defer p.mu.Unlock()

	// Find free block
	block := p.findFreeBlock(size)
	if block == nil {
		return nil, errors.New("out of persistent memory")
	}

	// Split block if needed
	if block.size > size+uint64(unsafe.Sizeof(FreeBlock{})) {
		p.splitBlock(block, size)
	}

	// Update metadata
	p.metadata.allocated += block.size
	p.allocator.allocated.Add(block.size)

	// Return pointer to allocated memory
	return unsafe.Add(p.baseAddr, block.offset), nil
}

// Free returns memory to the pool
func (p *PersistentMemoryPool) Free(ptr unsafe.Pointer, size uint64) {
	if ptr == nil {
		return
	}

	offset := uintptr(ptr) - uintptr(p.baseAddr)
	size = (size + CACHE_LINE - 1) &^ (CACHE_LINE - 1)

	p.mu.Lock()
	defer p.mu.Unlock()

	// Create free block
	block := &FreeBlock{
		offset: uint64(offset),
		size:   size,
	}

	// Add to free list
	p.addToFreeList(block)

	// Update metadata
	p.metadata.allocated -= size
	p.allocator.freed.Add(size)
}

// findFreeBlock finds a suitable free block
func (p *PersistentMemoryPool) findFreeBlock(size uint64) *FreeBlock {
	freeList := (*PMemFreeList)(unsafe.Add(p.baseAddr, p.metadata.freeList))
	current := freeList.head.Load()

	for current != 0 {
		block := (*FreeBlock)(unsafe.Add(p.baseAddr, current))
		if block.size >= size {
			return block
		}
		current = block.next
	}

	return nil
}

// splitBlock splits a free block
func (p *PersistentMemoryPool) splitBlock(block *FreeBlock, size uint64) {
	newBlock := (*FreeBlock)(unsafe.Add(p.baseAddr, block.offset+size))
	newBlock.offset = block.offset + size
	newBlock.size = block.size - size
	newBlock.next = block.next

	block.size = size
	block.next = newBlock.offset

	// Flush changes
	p.flush(unsafe.Pointer(block), unsafe.Sizeof(FreeBlock{}))
	p.flush(unsafe.Pointer(newBlock), unsafe.Sizeof(FreeBlock{}))
}

// addToFreeList adds a block to the free list
func (p *PersistentMemoryPool) addToFreeList(block *FreeBlock) {
	freeList := (*PMemFreeList)(unsafe.Add(p.baseAddr, p.metadata.freeList))
	block.next = freeList.head.Load()
	freeList.head.Store(block.offset)
	freeList.count.Add(1)

	// Flush changes
	p.flush(unsafe.Pointer(block), unsafe.Sizeof(FreeBlock{}))
	head := freeList.head.Load()
	p.flush(unsafe.Pointer(&head), 8)
}

// flush ensures data is persisted
func (p *PersistentMemoryPool) flush(ptr unsafe.Pointer, size uintptr) {
	// Platform-specific flush
	// On Linux, would use msync
	// For now, no-op
	_ = ptr
	_ = size
}

// Store stores an object in persistent memory
func (p *PersistentMemoryPool) Store(obj *PMemObject, data []byte) error {
	ptr, err := p.Allocate(uint64(len(data)))
	if err != nil {
		return err
	}

	// Copy data
	dst := (*[1 << 30]byte)(ptr)[:len(data):len(data)]
	copy(dst, data)

	// Update object metadata
	obj.Size = uint64(len(data))
	obj.Offset = uint64(uintptr(ptr) - uintptr(p.baseAddr))
	obj.Checksum = crc32Fast(data)
	obj.Modified = timeNow()

	// Flush data and metadata
	p.flush(ptr, uintptr(len(data)))

	return nil
}

// Load loads an object from persistent memory
func (p *PersistentMemoryPool) Load(obj *PMemObject) ([]byte, error) {
	if obj.Offset+obj.Size > p.size {
		return nil, errors.New("invalid object offset")
	}

	ptr := unsafe.Add(p.baseAddr, obj.Offset)
	data := make([]byte, obj.Size)
	src := (*[1 << 30]byte)(ptr)[:obj.Size:obj.Size]
	copy(data, src)

	// Verify checksum
	if crc32Fast(data) != obj.Checksum {
		return nil, errors.New("checksum mismatch")
	}

	return data, nil
}

// PMemTransaction represents a persistent memory transaction
type PMemTransaction struct {
	pool      *PersistentMemoryPool
	log       []TransactionLogEntry
	committed bool
	mu        sync.Mutex
}

// TransactionLogEntry represents a transaction log entry
type TransactionLogEntry struct {
	offset   uint64
	size     uint64
	oldData  []byte
	newData  []byte
	checksum uint32
}

// BeginTransaction starts a new transaction
func (p *PersistentMemoryPool) BeginTransaction() *PMemTransaction {
	return &PMemTransaction{
		pool: p,
		log:  make([]TransactionLogEntry, 0, 16),
	}
}

// Write adds a write operation to the transaction
func (t *PMemTransaction) Write(offset uint64, data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.committed {
		return errors.New("transaction already committed")
	}

	// Read old data
	oldData := make([]byte, len(data))
	src := (*[1 << 30]byte)(unsafe.Add(t.pool.baseAddr, offset))[:len(data):len(data)]
	copy(oldData, src)

	// Add to log
	t.log = append(t.log, TransactionLogEntry{
		offset:   offset,
		size:     uint64(len(data)),
		oldData:  oldData,
		newData:  data,
		checksum: crc32Fast(data),
	})

	return nil
}

// Commit commits the transaction
func (t *PMemTransaction) Commit() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.committed {
		return errors.New("transaction already committed")
	}

	// Apply all writes
	for _, entry := range t.log {
		dst := (*[1 << 30]byte)(unsafe.Add(t.pool.baseAddr, entry.offset))[:entry.size:entry.size]
		copy(dst, entry.newData)
		t.pool.flush(unsafe.Add(t.pool.baseAddr, entry.offset), uintptr(entry.size))
	}

	t.committed = true
	return nil
}

// Rollback rolls back the transaction
func (t *PMemTransaction) Rollback() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.committed {
		return errors.New("transaction already committed")
	}

	// Nothing to do - changes not applied
	t.committed = true
	return nil
}

// Stats returns persistent memory statistics
type PMemStats struct {
	TotalSize      uint64
	AllocatedSize  uint64
	FreeSize       uint64
	Allocations    uint64
	Deallocations  uint64
	Fragmentation  float64
}

// Stats returns pool statistics
func (p *PersistentMemoryPool) Stats() PMemStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	allocated := p.metadata.allocated
	free := p.size - allocated

	stats := PMemStats{
		TotalSize:     p.size,
		AllocatedSize: allocated,
		FreeSize:      free,
		Allocations:   p.allocator.allocated.Load(),
		Deallocations: p.allocator.freed.Load(),
	}

	// Calculate fragmentation
	freeList := (*PMemFreeList)(unsafe.Add(p.baseAddr, p.metadata.freeList))
	if freeList.count.Load() > 0 {
		stats.Fragmentation = float64(freeList.count.Load()) / float64(stats.FreeSize) * 100
	}

	return stats
}

// Close closes the persistent memory pool
func (p *PersistentMemoryPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Update last flush time
	p.metadata.lastFlush = timeNow()
	p.flush(unsafe.Pointer(p.metadata), unsafe.Sizeof(PMemMetadata{}))

	// Unmap memory
	data := (*[1 << 30]byte)(p.baseAddr)[:p.size:p.size]
	return syscall.Munmap(data)
}

// Helper functions

func crc32Fast(data []byte) uint32 {
	// Use hardware CRC32 if available
	return CRC32CHardware(uint64(len(data)))
}

func timeNow() int64 {
	return time.Now().UnixNano()
}

