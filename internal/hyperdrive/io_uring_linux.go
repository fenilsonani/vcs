// +build linux

package hyperdrive

import (
	"errors"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// io_uring constants
const (
	IORING_OP_NOP uint8 = iota
	IORING_OP_READV
	IORING_OP_WRITEV
	IORING_OP_FSYNC
	IORING_OP_READ_FIXED
	IORING_OP_WRITE_FIXED
	IORING_OP_POLL_ADD
	IORING_OP_POLL_REMOVE
	IORING_OP_SYNC_FILE_RANGE
	IORING_OP_SENDMSG
	IORING_OP_RECVMSG
	IORING_OP_TIMEOUT
	IORING_OP_TIMEOUT_REMOVE
	IORING_OP_ACCEPT
	IORING_OP_ASYNC_CANCEL
	IORING_OP_LINK_TIMEOUT
	IORING_OP_CONNECT
	IORING_OP_FALLOCATE
	IORING_OP_OPENAT
	IORING_OP_CLOSE
	IORING_OP_FILES_UPDATE
	IORING_OP_STATX
	IORING_OP_READ
	IORING_OP_WRITE
)

// IOUring represents an io_uring instance
type IOUring struct {
	fd      int
	sq      *SubmissionQueue
	cq      *CompletionQueue
	params  IOUringParams
	inFlight atomic.Int32
	mu      sync.Mutex
}

// SubmissionQueue represents the submission queue
type SubmissionQueue struct {
	head      *uint32
	tail      *uint32
	mask      uint32
	entries   uint32
	flags     *uint32
	dropped   *uint32
	array     unsafe.Pointer
	sqes      unsafe.Pointer
	sqeSize   uint32
	ringAddr  unsafe.Pointer
	ringSize  uint32
}

// CompletionQueue represents the completion queue
type CompletionQueue struct {
	head      *uint32
	tail      *uint32
	mask      uint32
	entries   uint32
	flags     *uint32
	overflow  *uint32
	cqes      unsafe.Pointer
	cqeSize   uint32
	ringAddr  unsafe.Pointer
	ringSize  uint32
}

// IOUringParams are parameters for io_uring setup
type IOUringParams struct {
	sqEntries    uint32
	cqEntries    uint32
	flags        uint32
	sqThreadCPU  uint32
	sqThreadIdle uint32
	features     uint32
	wqFD         uint32
	resv         [3]uint32
	sqOff        SubmissionQueueRingOffsets
	cqOff        CompletionQueueRingOffsets
}

// SubmissionQueueRingOffsets contains mmap offsets for SQ
type SubmissionQueueRingOffsets struct {
	head      uint32
	tail      uint32
	mask      uint32
	entries   uint32
	flags     uint32
	dropped   uint32
	array     uint32
	resv1     uint32
	resv2     uint64
}

// CompletionQueueRingOffsets contains mmap offsets for CQ
type CompletionQueueRingOffsets struct {
	head      uint32
	tail      uint32
	mask      uint32
	entries   uint32
	overflow  uint32
	cqes      uint32
	flags     uint32
	resv1     uint32
	resv2     uint64
}

// SubmissionQueueEntry represents an SQE
type SubmissionQueueEntry struct {
	opcode      uint8
	flags       uint8
	ioprio      uint16
	fd          int32
	off         uint64
	addr        uint64
	len         uint32
	userFlags   uint32
	userData    uint64
	bufIndex    uint16
	personality uint16
	fileIndex   uint32
	pad2        [2]uint64
}

// CompletionQueueEntry represents a CQE
type CompletionQueueEntry struct {
	userData uint64
	res      int32
	flags    uint32
}

var (
	ioUringPool     = sync.Pool{}
	globalIOUring   *IOUring
	ioUringInitOnce sync.Once
)

// GetIOUring returns a global io_uring instance
func GetIOUring() (*IOUring, error) {
	var err error
	ioUringInitOnce.Do(func() {
		globalIOUring, err = NewIOUring(256)
	})
	return globalIOUring, err
}

// NewIOUring creates a new io_uring instance
func NewIOUring(entries uint32) (*IOUring, error) {
	params := IOUringParams{
		sqEntries: entries,
	}

	// Setup io_uring
	fd, err := ioUringSetup(entries, &params)
	if err != nil {
		return nil, err
	}

	ring := &IOUring{
		fd:     fd,
		params: params,
	}

	// Map submission queue
	if err := ring.mapSubmissionQueue(); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	// Map completion queue
	if err := ring.mapCompletionQueue(); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	return ring, nil
}

// ReadAsync performs asynchronous read
func (ring *IOUring) ReadAsync(fd int, buf []byte, offset int64) (*IOUringFuture, error) {
	sqe := ring.getSQE()
	if sqe == nil {
		return nil, errors.New("submission queue full")
	}

	future := &IOUringFuture{
		done: make(chan struct{}),
	}

	sqe.opcode = IORING_OP_READ
	sqe.fd = int32(fd)
	sqe.addr = uint64(uintptr(unsafe.Pointer(&buf[0])))
	sqe.len = uint32(len(buf))
	sqe.off = uint64(offset)
	sqe.userData = uint64(uintptr(unsafe.Pointer(future)))

	ring.submit()
	return future, nil
}

// WriteAsync performs asynchronous write
func (ring *IOUring) WriteAsync(fd int, buf []byte, offset int64) (*IOUringFuture, error) {
	sqe := ring.getSQE()
	if sqe == nil {
		return nil, errors.New("submission queue full")
	}

	future := &IOUringFuture{
		done: make(chan struct{}),
	}

	sqe.opcode = IORING_OP_WRITE
	sqe.fd = int32(fd)
	sqe.addr = uint64(uintptr(unsafe.Pointer(&buf[0])))
	sqe.len = uint32(len(buf))
	sqe.off = uint64(offset)
	sqe.userData = uint64(uintptr(unsafe.Pointer(future)))

	ring.submit()
	return future, nil
}

// BatchRead performs multiple reads in a single syscall
func (ring *IOUring) BatchRead(requests []IORequest) ([]*IOUringFuture, error) {
	if len(requests) == 0 {
		return nil, nil
	}

	ring.mu.Lock()
	defer ring.mu.Unlock()

	futures := make([]*IOUringFuture, len(requests))

	for i, req := range requests {
		sqe := ring.getSQE()
		if sqe == nil {
			return futures[:i], errors.New("submission queue full")
		}

		future := &IOUringFuture{
			done: make(chan struct{}),
		}
		futures[i] = future

		sqe.opcode = IORING_OP_READ
		sqe.fd = int32(req.FD)
		sqe.addr = uint64(uintptr(unsafe.Pointer(&req.Buffer[0])))
		sqe.len = uint32(len(req.Buffer))
		sqe.off = uint64(req.Offset)
		sqe.userData = uint64(uintptr(unsafe.Pointer(future)))
	}

	ring.submit()
	return futures, nil
}

// getSQE gets next submission queue entry
func (ring *IOUring) getSQE() *SubmissionQueueEntry {
	head := atomic.LoadUint32(ring.sq.head)
	tail := atomic.LoadUint32(ring.sq.tail)

	if tail-head >= ring.sq.entries {
		return nil
	}

	idx := tail & ring.sq.mask
	sqe := (*SubmissionQueueEntry)(unsafe.Add(ring.sq.sqes, uintptr(idx)*unsafe.Sizeof(SubmissionQueueEntry{})))

	// Update array
	array := (*uint32)(unsafe.Add(ring.sq.array, uintptr(idx)*4))
	*array = idx

	atomic.StoreUint32(ring.sq.tail, tail+1)
	return sqe
}

// submit submits queued operations
func (ring *IOUring) submit() error {
	submitted := atomic.LoadUint32(ring.sq.tail) - atomic.LoadUint32(ring.sq.head)
	if submitted == 0 {
		return nil
	}

	_, _, errno := syscall.Syscall(syscall.SYS_IO_URING_ENTER,
		uintptr(ring.fd),
		uintptr(submitted),
		0, 0, 0, 0)

	if errno != 0 {
		return errno
	}

	ring.inFlight.Add(int32(submitted))
	return nil
}

// Wait waits for completions
func (ring *IOUring) Wait(count int) error {
	for ring.inFlight.Load() > 0 {
		err := ring.waitForCompletion(count)
		if err != nil {
			return err
		}
	}
	return nil
}

// waitForCompletion waits for at least one completion
func (ring *IOUring) waitForCompletion(minComplete int) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_IO_URING_ENTER,
		uintptr(ring.fd),
		0,
		uintptr(minComplete),
		1, // IORING_ENTER_GETEVENTS
		0, 0)

	if errno != 0 {
		return errno
	}

	// Process completions
	return ring.processCompletions()
}

// processCompletions processes completed operations
func (ring *IOUring) processCompletions() error {
	head := atomic.LoadUint32(ring.cq.head)
	tail := atomic.LoadUint32(ring.cq.tail)

	for head != tail {
		idx := head & ring.cq.mask
		cqe := (*CompletionQueueEntry)(unsafe.Add(ring.cq.cqes, uintptr(idx)*unsafe.Sizeof(CompletionQueueEntry{})))

		// Get future from user data
		future := (*IOUringFuture)(unsafe.Pointer(uintptr(cqe.userData)))
		if future != nil {
			future.result = int(cqe.res)
			future.flags = cqe.flags
			close(future.done)
		}

		head++
		ring.inFlight.Add(-1)
	}

	atomic.StoreUint32(ring.cq.head, head)
	return nil
}

// IOUringFuture represents an async I/O operation
type IOUringFuture struct {
	done   chan struct{}
	result int
	flags  uint32
}

// Wait waits for the operation to complete
func (f *IOUringFuture) Wait() (int, error) {
	<-f.done
	if f.result < 0 {
		return 0, syscall.Errno(-f.result)
	}
	return f.result, nil
}

// IORequest represents an I/O request
type IORequest struct {
	FD     int
	Buffer []byte
	Offset int64
}

// Platform-specific syscalls

func ioUringSetup(entries uint32, params *IOUringParams) (int, error) {
	fd, _, errno := syscall.Syscall(426, // SYS_IO_URING_SETUP
		uintptr(entries),
		uintptr(unsafe.Pointer(params)),
		0)

	if errno != 0 {
		return 0, errno
	}
	return int(fd), nil
}

// mapSubmissionQueue maps the submission queue
func (ring *IOUring) mapSubmissionQueue() error {
	// Map SQ ring
	sqRingSize := uintptr(ring.params.sqOff.array) + uintptr(ring.params.sqEntries)*4
	sqRingPtr, err := syscall.Mmap(ring.fd, 0, int(sqRingSize),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED|syscall.MAP_POPULATE)
	if err != nil {
		return err
	}

	ring.sq = &SubmissionQueue{
		head:     (*uint32)(unsafe.Pointer(&sqRingPtr[ring.params.sqOff.head])),
		tail:     (*uint32)(unsafe.Pointer(&sqRingPtr[ring.params.sqOff.tail])),
		mask:     *(*uint32)(unsafe.Pointer(&sqRingPtr[ring.params.sqOff.mask])),
		entries:  *(*uint32)(unsafe.Pointer(&sqRingPtr[ring.params.sqOff.entries])),
		flags:    (*uint32)(unsafe.Pointer(&sqRingPtr[ring.params.sqOff.flags])),
		dropped:  (*uint32)(unsafe.Pointer(&sqRingPtr[ring.params.sqOff.dropped])),
		array:    unsafe.Pointer(&sqRingPtr[ring.params.sqOff.array]),
		ringAddr: unsafe.Pointer(&sqRingPtr[0]),
		ringSize: uint32(sqRingSize),
	}

	// Map SQEs
	sqeSize := uintptr(ring.params.sqEntries) * unsafe.Sizeof(SubmissionQueueEntry{})
	sqePtr, err := syscall.Mmap(ring.fd, 0x10000000, int(sqeSize),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED|syscall.MAP_POPULATE)
	if err != nil {
		return err
	}

	ring.sq.sqes = unsafe.Pointer(&sqePtr[0])
	ring.sq.sqeSize = uint32(sqeSize)

	return nil
}

// mapCompletionQueue maps the completion queue
func (ring *IOUring) mapCompletionQueue() error {
	// Map CQ ring
	cqRingSize := uintptr(ring.params.cqOff.cqes) + uintptr(ring.params.cqEntries)*unsafe.Sizeof(CompletionQueueEntry{})
	cqRingPtr, err := syscall.Mmap(ring.fd, 0x8000000, int(cqRingSize),
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED|syscall.MAP_POPULATE)
	if err != nil {
		return err
	}

	ring.cq = &CompletionQueue{
		head:     (*uint32)(unsafe.Pointer(&cqRingPtr[ring.params.cqOff.head])),
		tail:     (*uint32)(unsafe.Pointer(&cqRingPtr[ring.params.cqOff.tail])),
		mask:     *(*uint32)(unsafe.Pointer(&cqRingPtr[ring.params.cqOff.mask])),
		entries:  *(*uint32)(unsafe.Pointer(&cqRingPtr[ring.params.cqOff.entries])),
		overflow: (*uint32)(unsafe.Pointer(&cqRingPtr[ring.params.cqOff.overflow])),
		cqes:     unsafe.Pointer(&cqRingPtr[ring.params.cqOff.cqes]),
		ringAddr: unsafe.Pointer(&cqRingPtr[0]),
		ringSize: uint32(cqRingSize),
	}

	return nil
}

// Close closes the io_uring instance
func (ring *IOUring) Close() error {
	// Unmap memory
	if ring.sq != nil {
		syscall.Munmap((*[1 << 30]byte)(ring.sq.ringAddr)[:ring.sq.ringSize:ring.sq.ringSize])
		syscall.Munmap((*[1 << 30]byte)(ring.sq.sqes)[:ring.sq.sqeSize:ring.sq.sqeSize])
	}

	if ring.cq != nil {
		syscall.Munmap((*[1 << 30]byte)(ring.cq.ringAddr)[:ring.cq.ringSize:ring.cq.ringSize])
	}

	return syscall.Close(ring.fd)
}

// HighPerformanceFileOps provides io_uring-based file operations
type HighPerformanceFileOps struct {
	ring *IOUring
}

// NewHighPerformanceFileOps creates optimized file operations handler
func NewHighPerformanceFileOps() (*HighPerformanceFileOps, error) {
	ring, err := GetIOUring()
	if err != nil {
		return nil, err
	}

	return &HighPerformanceFileOps{
		ring: ring,
	}, nil
}

// ReadFile reads entire file using io_uring
func (h *HighPerformanceFileOps) ReadFile(path string) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(fd)

	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil {
		return nil, err
	}

	buf := make([]byte, stat.Size)
	future, err := h.ring.ReadAsync(fd, buf, 0)
	if err != nil {
		return nil, err
	}

	n, err := future.Wait()
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}

// WriteFile writes data to file using io_uring
func (h *HighPerformanceFileOps) WriteFile(path string, data []byte) error {
	fd, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)

	future, err := h.ring.WriteAsync(fd, data, 0)
	if err != nil {
		return err
	}

	_, err = future.Wait()
	return err
}