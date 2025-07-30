package objects

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// Tag represents a git tag object (annotated tag)
type Tag struct {
	id       ObjectID
	object   ObjectID
	typ      ObjectType
	tag      string
	tagger   Signature
	message  string
}

// NewTag creates a new tag object
func NewTag(object ObjectID, typ ObjectType, tag string, tagger Signature, message string) *Tag {
	t := &Tag{
		object:  object,
		typ:     typ,
		tag:     tag,
		tagger:  tagger,
		message: message,
	}
	t.computeID()
	return t
}

// Type returns the object type
func (t *Tag) Type() ObjectType {
	return TypeTag
}

// Size returns the serialized size
func (t *Tag) Size() int64 {
	data, _ := t.Serialize()
	return int64(len(data))
}

// ID returns the object ID
func (t *Tag) ID() ObjectID {
	if t.id.IsZero() {
		t.computeID()
	}
	return t.id
}

// Object returns the tagged object ID
func (t *Tag) Object() ObjectID {
	return t.object
}

// ObjectType returns the type of the tagged object
func (t *Tag) ObjectType() ObjectType {
	return t.typ
}

// TagName returns the tag name
func (t *Tag) TagName() string {
	return t.tag
}

// Tagger returns the tagger signature
func (t *Tag) Tagger() Signature {
	return t.tagger
}

// Message returns the tag message
func (t *Tag) Message() string {
	return t.message
}

// Serialize serializes the tag object
func (t *Tag) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	
	// Object
	fmt.Fprintf(&buf, "object %s\n", t.object)
	
	// Type
	fmt.Fprintf(&buf, "type %s\n", t.typ)
	
	// Tag
	fmt.Fprintf(&buf, "tag %s\n", t.tag)
	
	// Tagger
	fmt.Fprintf(&buf, "tagger %s\n", t.tagger)
	
	// Empty line before message
	buf.WriteByte('\n')
	
	// Message
	buf.WriteString(t.message)
	
	return buf.Bytes(), nil
}

// computeID calculates the tag's object ID
func (t *Tag) computeID() {
	data, _ := t.Serialize()
	t.id = ComputeHash(TypeTag, data)
}

// ParseTag parses a tag from raw object data
func ParseTag(id ObjectID, data []byte) (*Tag, error) {
	tag := &Tag{
		id: id,
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
				return nil, fmt.Errorf("invalid tag header: %s", line)
			}
			
			key, value := parts[0], parts[1]
			
			switch key {
			case "object":
				object, err := NewObjectID(value)
				if err != nil {
					return nil, fmt.Errorf("invalid object ID: %w", err)
				}
				tag.object = object
				
			case "type":
				tag.typ = ObjectType(value)
				if !tag.typ.IsValid() {
					return nil, fmt.Errorf("invalid object type: %s", value)
				}
				
			case "tag":
				tag.tag = value
				
			case "tagger":
				sig, err := parseSignatureLine(value)
				if err != nil {
					return nil, fmt.Errorf("invalid tagger: %w", err)
				}
				tag.tagger = *sig
				
			default:
				// Ignore unknown headers
			}
		} else {
			messageLines = append(messageLines, line)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing tag: %w", err)
	}
	
	tag.message = strings.Join(messageLines, "\n")
	if len(messageLines) > 0 && !strings.HasSuffix(tag.message, "\n") {
		tag.message += "\n"
	}
	
	return tag, nil
}