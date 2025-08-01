// +build !linux

package hyperdrive

import (
	"errors"
	"os"
)

// Fallback implementations for non-Linux systems

// IOUring fallback for non-Linux systems
type IOUring struct{}

// IOUringFuture fallback
type IOUringFuture struct {
	result int
	err    error
}

// IORequest fallback
type IORequest struct {
	FD     int
	Buffer []byte
	Offset int64
}

// GetIOUring returns error on non-Linux systems
func GetIOUring() (*IOUring, error) {
	return nil, errors.New("io_uring not supported on this platform")
}

// NewIOUring returns error on non-Linux systems
func NewIOUring(entries uint32) (*IOUring, error) {
	return nil, errors.New("io_uring not supported on this platform")
}

// ReadAsync fallback using regular read
func (ring *IOUring) ReadAsync(fd int, buf []byte, offset int64) (*IOUringFuture, error) {
	f := os.NewFile(uintptr(fd), "")
	if f == nil {
		return nil, errors.New("invalid file descriptor")
	}

	n, err := f.ReadAt(buf, offset)
	return &IOUringFuture{result: n, err: err}, nil
}

// WriteAsync fallback using regular write
func (ring *IOUring) WriteAsync(fd int, buf []byte, offset int64) (*IOUringFuture, error) {
	f := os.NewFile(uintptr(fd), "")
	if f == nil {
		return nil, errors.New("invalid file descriptor")
	}

	n, err := f.WriteAt(buf, offset)
	return &IOUringFuture{result: n, err: err}, nil
}

// BatchRead fallback
func (ring *IOUring) BatchRead(requests []IORequest) ([]*IOUringFuture, error) {
	futures := make([]*IOUringFuture, len(requests))
	for i, req := range requests {
		futures[i], _ = ring.ReadAsync(req.FD, req.Buffer, req.Offset)
	}
	return futures, nil
}

// Wait fallback
func (ring *IOUring) Wait(count int) error {
	return nil
}

// Close fallback
func (ring *IOUring) Close() error {
	return nil
}

// Wait fallback for future
func (f *IOUringFuture) Wait() (int, error) {
	return f.result, f.err
}

// HighPerformanceFileOps fallback
type HighPerformanceFileOps struct{}

// NewHighPerformanceFileOps fallback
func NewHighPerformanceFileOps() (*HighPerformanceFileOps, error) {
	return &HighPerformanceFileOps{}, nil
}

// ReadFile fallback
func (h *HighPerformanceFileOps) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile fallback
func (h *HighPerformanceFileOps) WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// writeIOUring is defined in cpu_features.go