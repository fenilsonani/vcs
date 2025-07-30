package objects

import (
	"fmt"
	"time"
)

// ObjectType represents the type of a git object
type ObjectType string

const (
	TypeBlob   ObjectType = "blob"
	TypeTree   ObjectType = "tree"
	TypeCommit ObjectType = "commit"
	TypeTag    ObjectType = "tag"
)

// IsValid returns true if the object type is valid
func (t ObjectType) IsValid() bool {
	switch t {
	case TypeBlob, TypeTree, TypeCommit, TypeTag:
		return true
	default:
		return false
	}
}

// Object is the base interface for all git objects
type Object interface {
	Type() ObjectType
	Size() int64
	ID() ObjectID
	Serialize() ([]byte, error)
}

// Signature represents author/committer information
type Signature struct {
	Name  string
	Email string
	When  time.Time
}

// String returns the signature in git format
func (s Signature) String() string {
	timestamp := s.When.Unix()
	tz := s.When.Format("-0700")
	return fmt.Sprintf("%s <%s> %d %s", s.Name, s.Email, timestamp, tz)
}

// ParseSignature parses a signature from git format
func ParseSignature(data []byte) (*Signature, error) {
	// Implementation will be added when needed
	return nil, fmt.Errorf("ParseSignature not yet implemented")
}