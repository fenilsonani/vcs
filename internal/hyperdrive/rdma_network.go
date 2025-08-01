// +build linux

package hyperdrive

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// RDMA (Remote Direct Memory Access) provides zero-copy networking
// This is a simplified implementation - real RDMA requires kernel modules

// RDMADevice represents an RDMA-capable network device
type RDMADevice struct {
	name        string
	type_       string // InfiniBand, RoCE, iWARP
	ports       []RDMAPort
	maxQP       uint32 // Max queue pairs
	maxCQ       uint32 // Max completion queues
	maxMR       uint32 // Max memory regions
	capabilities uint64
}

// RDMAPort represents a port on an RDMA device
type RDMAPort struct {
	number     uint8
	state      uint8
	mtu        uint32
	linkSpeed  uint64 // Gbps
	gid        [16]byte
	lid        uint16
}

// RDMAConnection represents an RDMA connection
type RDMAConnection struct {
	local      *RDMAEndpoint
	remote     *RDMAEndpoint
	qp         *QueuePair
	cq         *CompletionQueue
	mr         []MemoryRegion
	state      atomic.Uint32
	stats      RDMAStats
	mu         sync.RWMutex
}

// RDMAEndpoint represents an RDMA endpoint
type RDMAEndpoint struct {
	address    string
	port       uint16
	qpn        uint32 // Queue pair number
	psn        uint32 // Packet sequence number
	gid        [16]byte
	lid        uint16
}

// QueuePair represents an RDMA queue pair
type QueuePair struct {
	id         uint32
	sendQueue  *WorkQueue
	recvQueue  *WorkQueue
	state      atomic.Uint32
	type_      uint8 // RC, UC, UD
	maxSend    uint32
	maxRecv    uint32
	maxInline  uint32
}

// WorkQueue represents a work queue (send or receive)
type WorkQueue struct {
	head      atomic.Uint32
	tail      atomic.Uint32
	mask      uint32
	entries   []WorkRequest
	completed atomic.Uint64
}

// WorkRequest represents an RDMA work request
type WorkRequest struct {
	id        uint64
	opcode    uint8
	flags     uint32
	sge       []ScatterGatherElement
	remoteAddr uint64
	rkey      uint32
	immediate uint32
	userData  unsafe.Pointer
}

// ScatterGatherElement represents a scatter-gather element
type ScatterGatherElement struct {
	addr  uint64
	length uint32
	lkey  uint32
}

// CompletionQueue represents a completion queue
type CompletionQueue struct {
	id        uint32
	head      atomic.Uint32
	tail      atomic.Uint32
	mask      uint32
	entries   []WorkCompletion
	notifyChan chan struct{}
}

// WorkCompletion represents a work completion
type WorkCompletion struct {
	id        uint64
	status    uint8
	opcode    uint8
	bytesLen  uint32
	immediate uint32
	srcQP     uint32
	flags     uint32
}

// MemoryRegion represents a registered memory region
type MemoryRegion struct {
	addr      unsafe.Pointer
	length    uint64
	lkey      uint32 // Local key
	rkey      uint32 // Remote key
	access    uint32
	pinned    bool
	hugepages bool
}

// RDMAStats tracks RDMA performance statistics
type RDMAStats struct {
	bytessSent     atomic.Uint64
	bytesRecv     atomic.Uint64
	operationsSent atomic.Uint64
	operationsRecv atomic.Uint64
	completions    atomic.Uint64
	errors         atomic.Uint64
	retries        atomic.Uint64
}

// RDMA opcodes
const (
	RDMA_SEND = iota
	RDMA_SEND_WITH_IMM
	RDMA_RECV
	RDMA_RECV_WITH_IMM
	RDMA_WRITE
	RDMA_WRITE_WITH_IMM
	RDMA_READ
	RDMA_ATOMIC_CMP_AND_SWP
	RDMA_ATOMIC_FETCH_AND_ADD
)

// RDMA access flags
const (
	RDMA_ACCESS_LOCAL_WRITE  = 1 << iota
	RDMA_ACCESS_REMOTE_WRITE
	RDMA_ACCESS_REMOTE_READ
	RDMA_ACCESS_REMOTE_ATOMIC
	RDMA_ACCESS_MW_BIND
	RDMA_ACCESS_ZERO_BASED
	RDMA_ACCESS_ON_DEMAND
)

// Connection states
const (
	RDMA_STATE_IDLE = iota
	RDMA_STATE_CONNECTING
	RDMA_STATE_CONNECTED
	RDMA_STATE_DISCONNECTING
	RDMA_STATE_ERROR
)

var (
	rdmaDevices   []RDMADevice
	rdmaInitOnce  sync.Once
	rdmaSupported bool
)

// InitRDMA initializes RDMA subsystem
func InitRDMA() error {
	var err error
	rdmaInitOnce.Do(func() {
		rdmaSupported, rdmaDevices = detectRDMADevices()
		if !rdmaSupported {
			err = errors.New("RDMA not supported on this system")
		}
	})
	return err
}

// detectRDMADevices detects available RDMA devices
func detectRDMADevices() (bool, []RDMADevice) {
	// In a real implementation, this would query /sys/class/infiniband/
	// For now, return mock devices for testing
	return false, nil
}

// NewRDMAConnection creates a new RDMA connection
func NewRDMAConnection(localAddr, remoteAddr string) (*RDMAConnection, error) {
	if !rdmaSupported {
		return nil, errors.New("RDMA not supported")
	}

	conn := &RDMAConnection{
		local: &RDMAEndpoint{
			address: localAddr,
		},
		remote: &RDMAEndpoint{
			address: remoteAddr,
		},
	}

	// Create queue pair
	conn.qp = &QueuePair{
		id:        generateQPID(),
		sendQueue: newWorkQueue(256),
		recvQueue: newWorkQueue(256),
		type_:     0, // RC (Reliable Connection)
		maxSend:   256,
		maxRecv:   256,
		maxInline: 256,
	}

	// Create completion queue
	conn.cq = &CompletionQueue{
		id:         generateCQID(),
		mask:       511,
		entries:    make([]WorkCompletion, 512),
		notifyChan: make(chan struct{}, 1),
	}

	conn.state.Store(RDMA_STATE_IDLE)
	return conn, nil
}

// Connect establishes an RDMA connection
func (c *RDMAConnection) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state.Load() != RDMA_STATE_IDLE {
		return errors.New("connection not in idle state")
	}

	c.state.Store(RDMA_STATE_CONNECTING)

	// In real RDMA, this would:
	// 1. Resolve address to GID
	// 2. Exchange QP information
	// 3. Modify QP state to RTR (Ready to Receive)
	// 4. Modify QP state to RTS (Ready to Send)

	// For now, simulate connection
	c.state.Store(RDMA_STATE_CONNECTED)
	return nil
}

// RegisterMemory registers a memory region for RDMA
func (c *RDMAConnection) RegisterMemory(addr unsafe.Pointer, length uint64, access uint32) (*MemoryRegion, error) {
	if c.state.Load() != RDMA_STATE_CONNECTED {
		return nil, errors.New("not connected")
	}

	mr := &MemoryRegion{
		addr:   addr,
		length: length,
		lkey:   generateKey(),
		rkey:   generateKey(),
		access: access,
	}

	// In real RDMA, this would pin memory pages
	// and register with the RDMA device

	c.mu.Lock()
	c.mr = append(c.mr, *mr)
	c.mu.Unlock()

	return mr, nil
}

// Write performs RDMA write operation (zero-copy)
func (c *RDMAConnection) Write(localAddr unsafe.Pointer, remoteAddr uint64, length uint32, rkey uint32) error {
	if c.state.Load() != RDMA_STATE_CONNECTED {
		return errors.New("not connected")
	}

	// Find local memory region
	mr := c.findMemoryRegion(localAddr)
	if mr == nil {
		return errors.New("memory not registered")
	}

	// Create work request
	wr := WorkRequest{
		id:         generateWRID(),
		opcode:     RDMA_WRITE,
		remoteAddr: remoteAddr,
		rkey:       rkey,
		sge: []ScatterGatherElement{{
			addr:   uint64(uintptr(localAddr)),
			length: length,
			lkey:   mr.Lkey,
		}},
	}

	// Post to send queue
	if err := c.qp.sendQueue.post(&wr); err != nil {
		return err
	}

	// Update stats
	c.stats.operationsSent.Add(1)
	c.stats.bytessSent.Add(uint64(length))

	// In real RDMA, the NIC would handle the transfer
	// For simulation, immediately complete
	c.completeWork(wr.id, 0, RDMA_WRITE, length)

	return nil
}

// Read performs RDMA read operation (zero-copy)
func (c *RDMAConnection) Read(localAddr unsafe.Pointer, remoteAddr uint64, length uint32, rkey uint32) error {
	if c.state.Load() != RDMA_STATE_CONNECTED {
		return errors.New("not connected")
	}

	// Find local memory region
	mr := c.findMemoryRegion(localAddr)
	if mr == nil {
		return errors.New("memory not registered")
	}

	// Create work request
	wr := WorkRequest{
		id:         generateWRID(),
		opcode:     RDMA_READ,
		remoteAddr: remoteAddr,
		rkey:       rkey,
		sge: []ScatterGatherElement{{
			addr:   uint64(uintptr(localAddr)),
			length: length,
			lkey:   mr.Lkey,
		}},
	}

	// Post to send queue
	if err := c.qp.sendQueue.post(&wr); err != nil {
		return err
	}

	// Update stats
	c.stats.operationsRecv.Add(1)
	c.stats.bytesRecv.Add(uint64(length))

	// In real RDMA, the NIC would handle the transfer
	// For simulation, immediately complete
	c.completeWork(wr.id, 0, RDMA_READ, length)

	return nil
}

// Send performs RDMA send operation
func (c *RDMAConnection) Send(data []byte) error {
	if c.state.Load() != RDMA_STATE_CONNECTED {
		return errors.New("not connected")
	}

	// For small data, use inline send
	if len(data) <= int(c.qp.maxInline) {
		return c.sendInline(data)
	}

	// Register memory for larger data
	mr, err := c.RegisterMemory(unsafe.Pointer(&data[0]), uint64(len(data)), RDMA_ACCESS_LOCAL_WRITE)
	if err != nil {
		return err
	}

	// Create work request
	wr := WorkRequest{
		id:     generateWRID(),
		opcode: RDMA_SEND,
		sge: []ScatterGatherElement{{
			addr:   uint64(uintptr(unsafe.Pointer(&data[0]))),
			length: uint32(len(data)),
			lkey:   mr.Lkey,
		}},
	}

	// Post to send queue
	return c.qp.sendQueue.post(&wr)
}

// sendInline sends small data inline
func (c *RDMAConnection) sendInline(data []byte) error {
	wr := WorkRequest{
		id:     generateWRID(),
		opcode: RDMA_SEND,
		flags:  1, // INLINE flag
		sge: []ScatterGatherElement{{
			addr:   uint64(uintptr(unsafe.Pointer(&data[0]))),
			length: uint32(len(data)),
		}},
	}

	return c.qp.sendQueue.post(&wr)
}

// Recv posts a receive buffer
func (c *RDMAConnection) Recv(buffer []byte) error {
	if c.state.Load() != RDMA_STATE_CONNECTED {
		return errors.New("not connected")
	}

	// Register memory
	mr, err := c.RegisterMemory(unsafe.Pointer(&buffer[0]), uint64(len(buffer)), RDMA_ACCESS_LOCAL_WRITE)
	if err != nil {
		return err
	}

	// Create work request
	wr := WorkRequest{
		id:     generateWRID(),
		opcode: RDMA_RECV,
		sge: []ScatterGatherElement{{
			addr:   uint64(uintptr(unsafe.Pointer(&buffer[0]))),
			length: uint32(len(buffer)),
			lkey:   mr.Lkey,
		}},
	}

	// Post to receive queue
	return c.qp.recvQueue.post(&wr)
}

// PollCompletion polls for work completions
func (c *RDMAConnection) PollCompletion(timeout time.Duration) (*WorkCompletion, error) {
	select {
	case <-c.cq.notifyChan:
		// Get completion from queue
		head := c.cq.head.Load()
		tail := c.cq.tail.Load()
		if head == tail {
			return nil, nil
		}

		wc := &c.cq.entries[head&c.cq.mask]
		c.cq.head.Store(head + 1)
		return wc, nil

	case <-time.After(timeout):
		return nil, nil
	}
}

// completeWork simulates work completion
func (c *RDMAConnection) completeWork(id uint64, status uint8, opcode uint8, length uint32) {
	tail := c.cq.tail.Load()
	wc := &c.cq.entries[tail&c.cq.mask]
	wc.id = id
	wc.status = status
	wc.opcode = opcode
	wc.bytesLen = length

	c.cq.tail.Store(tail + 1)
	c.stats.completions.Add(1)

	// Notify
	select {
	case c.cq.notifyChan <- struct{}{}:
	default:
	}
}

// findMemoryRegion finds a memory region containing the address
func (c *RDMAConnection) findMemoryRegion(addr unsafe.Pointer) *MemoryRegion {
	c.mu.RLock()
	defer c.mu.RUnlock()

	target := uintptr(addr)
	for i := range c.mr {
		mr := &c.mr[i]
		start := uintptr(mr.Addr)
		end := start + uintptr(mr.Length)
		if target >= start && target < end {
			return mr
		}
	}
	return nil
}

// Close closes the RDMA connection
func (c *RDMAConnection) Close() error {
	c.state.Store(RDMA_STATE_DISCONNECTING)

	// Deregister memory regions
	c.mu.Lock()
	c.mr = nil
	c.mu.Unlock()

	c.state.Store(RDMA_STATE_IDLE)
	return nil
}

// Stats returns connection statistics
func (c *RDMAConnection) Stats() RDMAStats {
	return RDMAStats{
		bytessSent:     c.stats.bytessSent,
		bytesRecv:     c.stats.bytesRecv,
		operationsSent: c.stats.operationsSent,
		operationsRecv: c.stats.operationsRecv,
		completions:    c.stats.completions,
		errors:         c.stats.errors,
		retries:        c.stats.retries,
	}
}

// Work queue implementation

func newWorkQueue(size uint32) *WorkQueue {
	return &WorkQueue{
		mask:    size - 1,
		entries: make([]WorkRequest, size),
	}
}

func (wq *WorkQueue) post(wr *WorkRequest) error {
	head := wq.head.Load()
	tail := wq.tail.Load()

	if tail-head >= uint32(len(wq.entries)) {
		return errors.New("work queue full")
	}

	wq.entries[tail&wq.mask] = *wr
	wq.tail.Store(tail + 1)
	return nil
}

// ID generators
var (
	qpIDCounter atomic.Uint32
	cqIDCounter atomic.Uint32
	keyCounter  atomic.Uint32
	wrIDCounter atomic.Uint64
)

func generateQPID() uint32  { return qpIDCounter.Add(1) }
func generateCQID() uint32  { return cqIDCounter.Add(1) }
func generateKey() uint32   { return keyCounter.Add(1) }
func generateWRID() uint64  { return wrIDCounter.Add(1) }

