//go:build !amd64
// +build !amd64

package hyperdrive

// Assembly-optimized hash functions (stubs for non-amd64)

// SHA256Assembly uses hand-tuned assembly (fallback for non-amd64)
func SHA256Assembly(data []byte) [32]byte {
	return sha256Fallback(data)
}

// MemcpyAssembly uses optimized memory copy (fallback for non-amd64)
func MemcpyAssembly(dst, src []byte) {
	copy(dst, src)
}

// CRC32Assembly computes CRC32C using SSE4.2 (fallback for non-amd64)
func CRC32Assembly(data []byte) uint32 {
	return crc32cHardware(uint64(len(data)))
}