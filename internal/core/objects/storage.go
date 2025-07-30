package objects

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Storage handles reading and writing git objects
type Storage struct {
	basePath string
	mu       sync.RWMutex
	cache    map[ObjectID]Object // Simple in-memory cache
}

// NewStorage creates a new object storage
func NewStorage(gitDir string) *Storage {
	return &Storage{
		basePath: filepath.Join(gitDir, "objects"),
		cache:    make(map[ObjectID]Object),
	}
}

// Init initializes the object storage directory structure
func (s *Storage) Init() error {
	// Create objects directory
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create objects directory: %w", err)
	}
	
	// Create subdirectories for loose objects (00-ff)
	for i := 0; i < 256; i++ {
		dir := filepath.Join(s.basePath, fmt.Sprintf("%02x", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create object subdirectory: %w", err)
		}
	}
	
	// Create pack directory
	packDir := filepath.Join(s.basePath, "pack")
	if err := os.MkdirAll(packDir, 0755); err != nil {
		return fmt.Errorf("failed to create pack directory: %w", err)
	}
	
	// Create info directory
	infoDir := filepath.Join(s.basePath, "info")
	if err := os.MkdirAll(infoDir, 0755); err != nil {
		return fmt.Errorf("failed to create info directory: %w", err)
	}
	
	return nil
}

// WriteObject writes an object to storage
func (s *Storage) WriteObject(obj Object) error {
	id := obj.ID()
	
	// Check if object already exists
	if s.HasObject(id) {
		return nil
	}
	
	// Serialize object
	data, err := obj.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize object: %w", err)
	}
	
	// Create object header
	header := fmt.Sprintf("%s %d\x00", obj.Type(), len(data))
	fullData := append([]byte(header), data...)
	
	// Compress data
	compressed, err := compressData(fullData)
	if err != nil {
		return fmt.Errorf("failed to compress object: %w", err)
	}
	
	// Write to loose object file
	path := s.objectPath(id)
	dir := filepath.Dir(path)
	
	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create object directory: %w", err)
	}
	
	// Write atomically using a temporary file
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, compressed, 0444); err != nil {
		return fmt.Errorf("failed to write object file: %w", err)
	}
	
	// Rename to final location
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize object file: %w", err)
	}
	
	// Update cache
	s.mu.Lock()
	s.cache[id] = obj
	s.mu.Unlock()
	
	return nil
}

// ReadObject reads an object from storage
func (s *Storage) ReadObject(id ObjectID) (Object, error) {
	// Check cache first
	s.mu.RLock()
	if obj, ok := s.cache[id]; ok {
		s.mu.RUnlock()
		return obj, nil
	}
	s.mu.RUnlock()
	
	// Read from loose object
	path := s.objectPath(id)
	compressed, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// TODO: Check packfiles
			return nil, fmt.Errorf("object not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read object file: %w", err)
	}
	
	// Decompress data
	fullData, err := decompressData(compressed)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress object: %w", err)
	}
	
	// Parse header
	nullIdx := bytes.IndexByte(fullData, 0)
	if nullIdx == -1 {
		return nil, fmt.Errorf("invalid object format: no null byte")
	}
	
	header := string(fullData[:nullIdx])
	data := fullData[nullIdx+1:]
	
	// Parse object type and size from header
	var objType string
	var size int
	if _, err := fmt.Sscanf(header, "%s %d", &objType, &size); err != nil {
		return nil, fmt.Errorf("invalid object header: %s", header)
	}
	
	if len(data) != size {
		return nil, fmt.Errorf("object size mismatch: expected %d, got %d", size, len(data))
	}
	
	// Parse object based on type
	var obj Object
	switch ObjectType(objType) {
	case TypeBlob:
		obj = ParseBlob(id, data)
	case TypeTree:
		obj, err = ParseTree(id, data)
	case TypeCommit:
		obj, err = ParseCommit(id, data)
	case TypeTag:
		obj, err = ParseTag(id, data)
	default:
		return nil, fmt.Errorf("unknown object type: %s", objType)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Update cache
	s.mu.Lock()
	s.cache[id] = obj
	s.mu.Unlock()
	
	return obj, nil
}

// HasObject checks if an object exists in storage
func (s *Storage) HasObject(id ObjectID) bool {
	// Check cache
	s.mu.RLock()
	if _, ok := s.cache[id]; ok {
		s.mu.RUnlock()
		return true
	}
	s.mu.RUnlock()
	
	// Check loose object
	path := s.objectPath(id)
	if _, err := os.Stat(path); err == nil {
		return true
	}
	
	// TODO: Check packfiles
	return false
}

// objectPath returns the path to a loose object file
func (s *Storage) objectPath(id ObjectID) string {
	hex := id.String()
	return filepath.Join(s.basePath, hex[:2], hex[2:])
}

// compressData compresses data using zlib
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	
	if _, err := w.Write(data); err != nil {
		w.Close()
		return nil, err
	}
	
	if err := w.Close(); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// decompressData decompresses data using zlib
func decompressData(compressed []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	
	return io.ReadAll(r)
}