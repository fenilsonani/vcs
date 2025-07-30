package objects

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"
)

// Commit represents a git commit object
type Commit struct {
	id        ObjectID
	tree      ObjectID
	parents   []ObjectID
	author    Signature
	committer Signature
	message   string
}

// NewCommit creates a new commit object
func NewCommit(tree ObjectID, parents []ObjectID, author, committer Signature, message string) *Commit {
	c := &Commit{
		tree:      tree,
		parents:   parents,
		author:    author,
		committer: committer,
		message:   message,
	}
	c.computeID()
	return c
}

// Type returns the object type
func (c *Commit) Type() ObjectType {
	return TypeCommit
}

// Size returns the serialized size
func (c *Commit) Size() int64 {
	data, _ := c.Serialize()
	return int64(len(data))
}

// ID returns the object ID
func (c *Commit) ID() ObjectID {
	if c.id.IsZero() {
		c.computeID()
	}
	return c.id
}

// Tree returns the tree object ID
func (c *Commit) Tree() ObjectID {
	return c.tree
}

// Parents returns the parent commit IDs
func (c *Commit) Parents() []ObjectID {
	return c.parents
}

// Author returns the author signature
func (c *Commit) Author() Signature {
	return c.author
}

// Committer returns the committer signature
func (c *Commit) Committer() Signature {
	return c.committer
}

// Message returns the commit message
func (c *Commit) Message() string {
	return c.message
}

// Serialize serializes the commit object
func (c *Commit) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	
	// Tree
	fmt.Fprintf(&buf, "tree %s\n", c.tree)
	
	// Parents
	for _, parent := range c.parents {
		fmt.Fprintf(&buf, "parent %s\n", parent)
	}
	
	// Author
	fmt.Fprintf(&buf, "author %s\n", c.author)
	
	// Committer
	fmt.Fprintf(&buf, "committer %s\n", c.committer)
	
	// Empty line before message
	buf.WriteByte('\n')
	
	// Message
	buf.WriteString(c.message)
	
	return buf.Bytes(), nil
}

// computeID calculates the commit's object ID
func (c *Commit) computeID() {
	data, _ := c.Serialize()
	c.id = ComputeHash(TypeCommit, data)
}

// ParseCommit parses a commit from raw object data
func ParseCommit(id ObjectID, data []byte) (*Commit, error) {
	commit := &Commit{
		id:      id,
		parents: make([]ObjectID, 0),
	}
	
	scanner := bufio.NewScanner(bytes.NewReader(data))
	
	// Parse headers
	inHeaders := true
	var messageLines []string
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if inHeaders {
			if line == "" {
				inHeaders = false
				continue
			}
			
			parts := strings.SplitN(line, " ", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid commit header: %s", line)
			}
			
			key, value := parts[0], parts[1]
			
			switch key {
			case "tree":
				tree, err := NewObjectID(value)
				if err != nil {
					return nil, fmt.Errorf("invalid tree ID: %w", err)
				}
				commit.tree = tree
				
			case "parent":
				parent, err := NewObjectID(value)
				if err != nil {
					return nil, fmt.Errorf("invalid parent ID: %w", err)
				}
				commit.parents = append(commit.parents, parent)
				
			case "author":
				sig, err := parseSignatureLine(value)
				if err != nil {
					return nil, fmt.Errorf("invalid author: %w", err)
				}
				commit.author = *sig
				
			case "committer":
				sig, err := parseSignatureLine(value)
				if err != nil {
					return nil, fmt.Errorf("invalid committer: %w", err)
				}
				commit.committer = *sig
				
			default:
				// Ignore unknown headers
			}
		} else {
			messageLines = append(messageLines, line)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing commit: %w", err)
	}
	
	commit.message = strings.Join(messageLines, "\n")
	if len(messageLines) > 0 && !strings.HasSuffix(commit.message, "\n") {
		commit.message += "\n"
	}
	
	return commit, nil
}

// parseSignatureLine parses a signature from a line like "Name <email> timestamp timezone"
func parseSignatureLine(line string) (*Signature, error) {
	// Find email boundaries
	emailStart := strings.IndexByte(line, '<')
	emailEnd := strings.IndexByte(line, '>')
	
	if emailStart == -1 || emailEnd == -1 || emailStart >= emailEnd {
		return nil, fmt.Errorf("invalid signature format")
	}
	
	name := strings.TrimSpace(line[:emailStart])
	email := line[emailStart+1 : emailEnd]
	
	// Parse timestamp and timezone
	timeStr := strings.TrimSpace(line[emailEnd+1:])
	parts := strings.Fields(timeStr)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid timestamp format")
	}
	
	timestamp, err := parseUnixTimestamp(parts[0], parts[1])
	if err != nil {
		return nil, err
	}
	
	return &Signature{
		Name:  name,
		Email: email,
		When:  timestamp,
	}, nil
}

// parseUnixTimestamp parses a unix timestamp with timezone
func parseUnixTimestamp(unixStr, tzStr string) (time.Time, error) {
	var unix int64
	n, err := fmt.Sscanf(unixStr, "%d", &unix)
	if err != nil || n != 1 {
		return time.Time{}, fmt.Errorf("invalid timestamp")
	}
	
	// Parse timezone offset (e.g., "+0200" or "-0700")
	var tzOffset int
	n, err = fmt.Sscanf(tzStr, "%d", &tzOffset)
	if err != nil || n != 1 {
		return time.Time{}, fmt.Errorf("invalid timezone")
	}
	
	// Convert timezone offset to seconds
	hours := tzOffset / 100
	minutes := tzOffset % 100
	offsetSeconds := hours*3600 + minutes*60
	
	// Create location with the offset
	location := time.FixedZone("", offsetSeconds)
	
	return time.Unix(unix, 0).In(location), nil
}