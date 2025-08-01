// +build !linux

package hyperdrive

import (
	"errors"
	"time"
	"unsafe"
)

// RDMA fallback for non-Linux systems

type RDMAConnection struct{}
type MemoryRegion struct{}
type WorkCompletion struct{}
type RDMAStats struct{}

func InitRDMA() error {
	return errors.New("RDMA not supported on this platform")
}

func NewRDMAConnection(localAddr, remoteAddr string) (*RDMAConnection, error) {
	return nil, errors.New("RDMA not supported on this platform")
}

func (c *RDMAConnection) Connect() error {
	return errors.New("RDMA not supported")
}

func (c *RDMAConnection) RegisterMemory(addr unsafe.Pointer, length uint64, access uint32) (*MemoryRegion, error) {
	return nil, errors.New("RDMA not supported")
}

func (c *RDMAConnection) Write(localAddr unsafe.Pointer, remoteAddr uint64, length uint32, rkey uint32) error {
	return errors.New("RDMA not supported")
}

func (c *RDMAConnection) Read(localAddr unsafe.Pointer, remoteAddr uint64, length uint32, rkey uint32) error {
	return errors.New("RDMA not supported")
}

func (c *RDMAConnection) Send(data []byte) error {
	return errors.New("RDMA not supported")
}

func (c *RDMAConnection) Recv(buffer []byte) error {
	return errors.New("RDMA not supported")
}

func (c *RDMAConnection) PollCompletion(timeout time.Duration) (*WorkCompletion, error) {
	return nil, errors.New("RDMA not supported")
}

func (c *RDMAConnection) Close() error {
	return nil
}

func (c *RDMAConnection) Stats() RDMAStats {
	return RDMAStats{}
}