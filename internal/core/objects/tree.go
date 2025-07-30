package objects

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
)

// FileMode represents the mode of a file in a tree
type FileMode uint32

const (
	ModeTree    FileMode = 0040000
	ModeBlob    FileMode = 0100644
	ModeExec    FileMode = 0100755
	ModeSymlink FileMode = 0120000
	ModeCommit  FileMode = 0160000 // Submodule
)

// TreeEntry represents an entry in a tree object
type TreeEntry struct {
	Mode FileMode
	Name string
	ID   ObjectID
}

// Tree represents a git tree object (directory listing)
type Tree struct {
	id      ObjectID
	entries []TreeEntry
}

// NewTree creates a new tree object
func NewTree() *Tree {
	return &Tree{
		entries: make([]TreeEntry, 0),
	}
}

// AddEntry adds an entry to the tree
func (t *Tree) AddEntry(mode FileMode, name string, id ObjectID) error {
	if name == "" {
		return fmt.Errorf("entry name cannot be empty")
	}
	
	// Check for duplicate names
	for _, e := range t.entries {
		if e.Name == name {
			return fmt.Errorf("duplicate entry name: %s", name)
		}
	}
	
	t.entries = append(t.entries, TreeEntry{
		Mode: mode,
		Name: name,
		ID:   id,
	})
	
	// Recompute hash after modification
	t.computeID()
	return nil
}

// Entries returns all tree entries
func (t *Tree) Entries() []TreeEntry {
	return t.entries
}

// Type returns the object type
func (t *Tree) Type() ObjectType {
	return TypeTree
}

// Size returns the serialized size
func (t *Tree) Size() int64 {
	data, _ := t.Serialize()
	return int64(len(data))
}

// ID returns the object ID
func (t *Tree) ID() ObjectID {
	if t.id.IsZero() {
		t.computeID()
	}
	return t.id
}

// Serialize serializes the tree object
func (t *Tree) Serialize() ([]byte, error) {
	// Sort entries by name for consistent hashing
	sorted := make([]TreeEntry, len(t.entries))
	copy(sorted, t.entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	
	var buf bytes.Buffer
	for _, entry := range sorted {
		// Format: <mode> <name>\0<20-byte SHA-1>
		fmt.Fprintf(&buf, "%o %s\x00", entry.Mode, entry.Name)
		buf.Write(entry.ID[:])
	}
	
	return buf.Bytes(), nil
}

// computeID calculates the tree's object ID
func (t *Tree) computeID() {
	data, _ := t.Serialize()
	t.id = ComputeHash(TypeTree, data)
}

// ParseTree parses a tree from raw object data
func ParseTree(id ObjectID, data []byte) (*Tree, error) {
	tree := &Tree{
		id:      id,
		entries: make([]TreeEntry, 0),
	}
	
	for len(data) > 0 {
		// Find the space separating mode and name
		spaceIdx := bytes.IndexByte(data, ' ')
		if spaceIdx == -1 {
			return nil, fmt.Errorf("invalid tree format: no space found")
		}
		
		// Parse mode
		modeStr := string(data[:spaceIdx])
		mode, err := strconv.ParseUint(modeStr, 8, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid file mode: %s", modeStr)
		}
		
		data = data[spaceIdx+1:]
		
		// Find the null byte separating name and hash
		nullIdx := bytes.IndexByte(data, 0)
		if nullIdx == -1 {
			return nil, fmt.Errorf("invalid tree format: no null byte found")
		}
		
		name := string(data[:nullIdx])
		data = data[nullIdx+1:]
		
		// Read the 20-byte SHA-1 hash
		if len(data) < 20 {
			return nil, fmt.Errorf("invalid tree format: insufficient data for hash")
		}
		
		var objID ObjectID
		copy(objID[:], data[:20])
		data = data[20:]
		
		tree.entries = append(tree.entries, TreeEntry{
			Mode: FileMode(mode),
			Name: name,
			ID:   objID,
		})
	}
	
	return tree, nil
}