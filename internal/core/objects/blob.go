package objects

import (
	"bytes"
	"io"
)

// Blob represents a git blob object (file content)
type Blob struct {
	id   ObjectID
	data []byte
}

// NewBlob creates a new blob object from data
func NewBlob(data []byte) *Blob {
	b := &Blob{
		data: data,
	}
	b.id = ComputeHash(TypeBlob, data)
	return b
}

// NewBlobFromReader creates a new blob object from a reader
func NewBlobFromReader(r io.Reader) (*Blob, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return NewBlob(data), nil
}

// Type returns the object type
func (b *Blob) Type() ObjectType {
	return TypeBlob
}

// Size returns the size of the blob data
func (b *Blob) Size() int64 {
	return int64(len(b.data))
}

// ID returns the object ID
func (b *Blob) ID() ObjectID {
	return b.id
}

// Data returns the blob data
func (b *Blob) Data() []byte {
	return b.data
}

// Reader returns a reader for the blob data
func (b *Blob) Reader() io.Reader {
	return bytes.NewReader(b.data)
}

// Serialize returns the raw blob data
func (b *Blob) Serialize() ([]byte, error) {
	return b.data, nil
}

// ParseBlob parses a blob from raw object data
func ParseBlob(id ObjectID, data []byte) *Blob {
	return &Blob{
		id:   id,
		data: data,
	}
}