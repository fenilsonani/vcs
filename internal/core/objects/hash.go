package objects

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
)

// ObjectID represents a SHA-1 hash used to identify git objects
type ObjectID [20]byte

// String returns the hexadecimal string representation of the ObjectID
func (id ObjectID) String() string {
	return hex.EncodeToString(id[:])
}

// Short returns the first 7 characters of the hash
func (id ObjectID) Short() string {
	return id.String()[:7]
}

// IsZero returns true if the ObjectID is all zeros
func (id ObjectID) IsZero() bool {
	for _, b := range id {
		if b != 0 {
			return false
		}
	}
	return true
}

// NewObjectID creates an ObjectID from a hexadecimal string
func NewObjectID(hexStr string) (ObjectID, error) {
	var id ObjectID
	
	if len(hexStr) != 40 {
		return id, fmt.Errorf("invalid object ID length: expected 40, got %d", len(hexStr))
	}
	
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return id, fmt.Errorf("invalid hex string: %w", err)
	}
	
	copy(id[:], bytes)
	return id, nil
}

// ComputeHash calculates the SHA-1 hash of the given data with the object type prefix
func ComputeHash(objectType ObjectType, data []byte) ObjectID {
	h := sha1.New()
	fmt.Fprintf(h, "%s %d\x00", objectType, len(data))
	h.Write(data)
	
	var id ObjectID
	copy(id[:], h.Sum(nil))
	return id
}

// HashReader calculates the SHA-1 hash while reading from an io.Reader
func HashReader(objectType ObjectType, size int64, r io.Reader) (ObjectID, error) {
	h := sha1.New()
	fmt.Fprintf(h, "%s %d\x00", objectType, size)
	
	if _, err := io.Copy(h, r); err != nil {
		return ObjectID{}, fmt.Errorf("failed to hash reader: %w", err)
	}
	
	var id ObjectID
	copy(id[:], h.Sum(nil))
	return id, nil
}

// ParseObjectID attempts to parse an ObjectID from various formats
func ParseObjectID(input string) (ObjectID, error) {
	// Remove any whitespace
	input = trimWhitespace(input)
	
	// Handle full 40-character hash
	if len(input) == 40 {
		return NewObjectID(input)
	}
	
	// For abbreviated hashes, we would need access to the object database
	// This is a placeholder for now
	return ObjectID{}, fmt.Errorf("abbreviated object IDs not yet supported: %s", input)
}

func trimWhitespace(s string) string {
	// Fast whitespace trimming
	start := 0
	end := len(s)
	
	for start < end && isWhitespace(s[start]) {
		start++
	}
	
	for end > start && isWhitespace(s[end-1]) {
		end--
	}
	
	return s[start:end]
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}