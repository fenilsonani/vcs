package index

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fenilsonani/vcs/internal/core/objects"
)

const (
	// IndexSignature is the signature for index files
	IndexSignature = "DIRC"
	// IndexVersion is the current index format version
	IndexVersion = 2
	// EntrySize is the minimum size of an index entry
	EntrySize = 62
)

// Flags for index entries
const (
	FlagAssumeValid = 0x8000
	FlagExtended    = 0x4000
	FlagStageMask   = 0x3000
	FlagNameMask    = 0x0FFF
)

// Entry represents a single entry in the index
type Entry struct {
	CTime     time.Time
	MTime     time.Time
	Dev       uint32
	Ino       uint32
	Mode      objects.FileMode
	UID       uint32
	GID       uint32
	Size      uint32
	ID        objects.ObjectID
	Flags     uint16
	Path      string
	SkipWorktree bool
	IntentToAdd  bool
}

// Stage returns the merge stage of the entry (0-3)
func (e *Entry) Stage() int {
	return int((e.Flags & FlagStageMask) >> 12)
}

// SetStage sets the merge stage of the entry
func (e *Entry) SetStage(stage int) {
	e.Flags = (e.Flags &^ FlagStageMask) | uint16(stage<<12)
}

// Index represents the git index (staging area)
type Index struct {
	version int32
	entries []*Entry
	cache   map[string]*Entry
}

// New creates a new empty index
func New() *Index {
	return &Index{
		version: IndexVersion,
		entries: make([]*Entry, 0),
		cache:   make(map[string]*Entry),
	}
}

// Version returns the index version
func (idx *Index) Version() int32 {
	return idx.version
}

// Entries returns all entries in the index
func (idx *Index) Entries() []*Entry {
	return idx.entries
}

// Add adds or updates an entry in the index
func (idx *Index) Add(entry *Entry) error {
	if entry.Path == "" {
		return fmt.Errorf("entry path cannot be empty")
	}

	// Update cache
	idx.cache[entry.Path] = entry

	// Find existing entry
	found := false
	for i, e := range idx.entries {
		if e.Path == entry.Path {
			idx.entries[i] = entry
			found = true
			break
		}
	}

	if !found {
		idx.entries = append(idx.entries, entry)
	}

	// Keep entries sorted by path
	idx.sort()
	return nil
}

// Remove removes an entry from the index
func (idx *Index) Remove(path string) error {
	delete(idx.cache, path)

	for i, e := range idx.entries {
		if e.Path == path {
			idx.entries = append(idx.entries[:i], idx.entries[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("entry not found: %s", path)
}

// Get returns an entry by path
func (idx *Index) Get(path string) (*Entry, bool) {
	entry, ok := idx.cache[path]
	return entry, ok
}

// Clear removes all entries from the index
func (idx *Index) Clear() {
	idx.entries = idx.entries[:0]
	idx.cache = make(map[string]*Entry)
}

// sort sorts entries by path
func (idx *Index) sort() {
	sort.Slice(idx.entries, func(i, j int) bool {
		return idx.entries[i].Path < idx.entries[j].Path
	})
}

// WriteTo writes the index to a writer
func (idx *Index) WriteTo(w io.Writer) error {
	// Sort entries before writing
	idx.sort()

	// Write header
	header := make([]byte, 12)
	copy(header[0:4], IndexSignature)
	binary.BigEndian.PutUint32(header[4:8], uint32(idx.version))
	binary.BigEndian.PutUint32(header[8:12], uint32(len(idx.entries)))

	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Hash writer to calculate checksum
	h := sha1.New()
	mw := io.MultiWriter(w, h)

	// Write header to hash
	h.Write(header)

	// Write entries
	for _, entry := range idx.entries {
		if err := idx.writeEntry(mw, entry); err != nil {
			return fmt.Errorf("failed to write entry: %w", err)
		}
	}

	// Write checksum
	checksum := h.Sum(nil)
	if _, err := w.Write(checksum); err != nil {
		return fmt.Errorf("failed to write checksum: %w", err)
	}

	return nil
}

// writeEntry writes a single entry
func (idx *Index) writeEntry(w io.Writer, entry *Entry) error {
	// Create entry buffer
	buf := new(bytes.Buffer)

	// Write fixed-size fields
	binary.Write(buf, binary.BigEndian, uint32(entry.CTime.Unix()))
	binary.Write(buf, binary.BigEndian, uint32(entry.CTime.Nanosecond()))
	binary.Write(buf, binary.BigEndian, uint32(entry.MTime.Unix()))
	binary.Write(buf, binary.BigEndian, uint32(entry.MTime.Nanosecond()))
	binary.Write(buf, binary.BigEndian, entry.Dev)
	binary.Write(buf, binary.BigEndian, entry.Ino)
	binary.Write(buf, binary.BigEndian, uint32(entry.Mode))
	binary.Write(buf, binary.BigEndian, entry.UID)
	binary.Write(buf, binary.BigEndian, entry.GID)
	binary.Write(buf, binary.BigEndian, entry.Size)
	buf.Write(entry.ID[:])

	// Calculate flags
	flags := entry.Flags
	nameLen := len(entry.Path)
	if nameLen > FlagNameMask {
		nameLen = FlagNameMask
	}
	flags = (flags &^ FlagNameMask) | uint16(nameLen)
	binary.Write(buf, binary.BigEndian, flags)

	// Write path
	buf.WriteString(entry.Path)
	buf.WriteByte(0) // null terminator

	// Pad to 8-byte boundary
	entrySize := EntrySize + len(entry.Path) + 1
	padding := (8 - (entrySize % 8)) % 8
	for i := 0; i < padding; i++ {
		buf.WriteByte(0)
	}

	_, err := w.Write(buf.Bytes())
	return err
}

// ReadFrom reads the index from a reader
func (idx *Index) ReadFrom(r io.Reader) error {
	// Read header
	header := make([]byte, 12)
	if _, err := io.ReadFull(r, header); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	// Verify signature
	if string(header[0:4]) != IndexSignature {
		return fmt.Errorf("invalid index signature")
	}

	// Read version
	idx.version = int32(binary.BigEndian.Uint32(header[4:8]))
	if idx.version < 2 || idx.version > 4 {
		return fmt.Errorf("unsupported index version: %d", idx.version)
	}

	// Read entry count
	entryCount := binary.BigEndian.Uint32(header[8:12])

	// Hash reader to verify checksum
	h := sha1.New()
	h.Write(header)
	tr := io.TeeReader(r, h)

	// Clear existing entries
	idx.Clear()

	// Read entries
	for i := uint32(0); i < entryCount; i++ {
		entry, err := idx.readEntry(tr)
		if err != nil {
			return fmt.Errorf("failed to read entry %d: %w", i, err)
		}
		idx.entries = append(idx.entries, entry)
		idx.cache[entry.Path] = entry
	}

	// Read and verify checksum
	expectedChecksum := make([]byte, 20)
	if _, err := io.ReadFull(r, expectedChecksum); err != nil {
		return fmt.Errorf("failed to read checksum: %w", err)
	}

	actualChecksum := h.Sum(nil)
	if !bytes.Equal(expectedChecksum, actualChecksum) {
		return fmt.Errorf("checksum mismatch")
	}

	return nil
}

// readEntry reads a single entry
func (idx *Index) readEntry(r io.Reader) (*Entry, error) {
	entry := &Entry{}

	// Read fixed-size fields
	var cTimeSec, cTimeNsec uint32
	var mTimeSec, mTimeNsec uint32
	var mode uint32

	binary.Read(r, binary.BigEndian, &cTimeSec)
	binary.Read(r, binary.BigEndian, &cTimeNsec)
	binary.Read(r, binary.BigEndian, &mTimeSec)
	binary.Read(r, binary.BigEndian, &mTimeNsec)
	binary.Read(r, binary.BigEndian, &entry.Dev)
	binary.Read(r, binary.BigEndian, &entry.Ino)
	binary.Read(r, binary.BigEndian, &mode)
	binary.Read(r, binary.BigEndian, &entry.UID)
	binary.Read(r, binary.BigEndian, &entry.GID)
	binary.Read(r, binary.BigEndian, &entry.Size)
	
	if _, err := io.ReadFull(r, entry.ID[:]); err != nil {
		return nil, err
	}
	
	binary.Read(r, binary.BigEndian, &entry.Flags)

	entry.CTime = time.Unix(int64(cTimeSec), int64(cTimeNsec))
	entry.MTime = time.Unix(int64(mTimeSec), int64(mTimeNsec))
	entry.Mode = objects.FileMode(mode)

	// Read path
	nameLen := int(entry.Flags & FlagNameMask)
	if nameLen == FlagNameMask {
		// Long path, read until null
		var pathBuf bytes.Buffer
		for {
			b := make([]byte, 1)
			if _, err := r.Read(b); err != nil {
				return nil, err
			}
			if b[0] == 0 {
				break
			}
			pathBuf.WriteByte(b[0])
		}
		entry.Path = pathBuf.String()
		nameLen = len(entry.Path) + 1
	} else {
		// Normal path
		pathBuf := make([]byte, nameLen)
		if _, err := io.ReadFull(r, pathBuf); err != nil {
			return nil, err
		}
		entry.Path = string(pathBuf)
		
		// Read null terminator
		if _, err := r.Read(make([]byte, 1)); err != nil {
			return nil, err
		}
		nameLen++
	}

	// Skip padding
	entrySize := EntrySize + nameLen
	padding := (8 - (entrySize % 8)) % 8
	if padding > 0 {
		if _, err := r.Read(make([]byte, padding)); err != nil {
			return nil, err
		}
	}

	return entry, nil
}

// WriteToFile writes the index to a file
func (idx *Index) WriteToFile(path string) error {
	// Create temporary file
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".index-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	// Write index
	if err := idx.WriteTo(tmp); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomically replace
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// ReadFromFile reads the index from a file
func (idx *Index) ReadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open index file: %w", err)
	}
	defer file.Close()

	return idx.ReadFrom(file)
}