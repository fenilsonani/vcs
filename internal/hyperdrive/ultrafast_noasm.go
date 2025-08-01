// +build !amd64

package hyperdrive

import (
	"crypto/sha256"
	"hash/crc32"
	"unsafe"
)

// Fallback implementations for non-AMD64 architectures

func sha256Hardware(data []byte) [32]byte {
	return sha256.Sum256(data)
}

func sha256AVX512(data []byte) [32]byte {
	return sha256.Sum256(data)
}

func parallelSHA256AVX512(inputs [][]byte, outputs [][32]byte) {
	for i, input := range inputs {
		if i < len(outputs) {
			outputs[i] = sha256.Sum256(input)
		}
	}
}

func crc32cHardware(key uint64) uint32 {
	data := (*[8]byte)(unsafe.Pointer(&key))[:]
	return crc32.ChecksumIEEE(data)
}

func nonTemporalCopy(dst, src unsafe.Pointer, size int) {
	// Regular copy for non-AMD64
	dstSlice := (*[1 << 30]byte)(dst)[:size:size]
	srcSlice := (*[1 << 30]byte)(src)[:size:size]
	copy(dstSlice, srcSlice)
}

func sfence() {
	// No-op on non-AMD64
}

func clwb(addr unsafe.Pointer) {
	// No-op on non-AMD64
}

// prefetchT0 and prefetchNTA are defined in cpu_features.go