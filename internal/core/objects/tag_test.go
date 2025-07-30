package objects

import (
	"strings"
	"testing"
	"time"
)

func TestNewTag(t *testing.T) {
	object, _ := NewObjectID("1234567890abcdef1234567890abcdef12345678")
	
	tagger := Signature{
		Name:  "Test Tagger",
		Email: "tagger@example.com",
		When:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	
	tests := []struct {
		name    string
		object  ObjectID
		typ     ObjectType
		tag     string
		tagger  Signature
		message string
	}{
		{
			name:    "simple tag",
			object:  object,
			typ:     TypeCommit,
			tag:     "v1.0.0",
			tagger:  tagger,
			message: "Release version 1.0.0\n",
		},
		{
			name:    "tag with multi-line message",
			object:  object,
			typ:     TypeCommit,
			tag:     "v2.0.0",
			tagger:  tagger,
			message: "Release version 2.0.0\n\nThis release includes:\n- Feature A\n- Feature B\n",
		},
		{
			name:    "tag pointing to tree",
			object:  object,
			typ:     TypeTree,
			tag:     "tree-snapshot",
			tagger:  tagger,
			message: "Snapshot of tree state\n",
		},
		{
			name:    "tag pointing to blob",
			object:  object,
			typ:     TypeBlob,
			tag:     "important-file",
			tagger:  tagger,
			message: "Tagged important file\n",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := NewTag(tt.object, tt.typ, tt.tag, tt.tagger, tt.message)
			
			if tag.Type() != TypeTag {
				t.Errorf("Type() = %v, want %v", tag.Type(), TypeTag)
			}
			
			if tag.Object() != tt.object {
				t.Errorf("Object() = %v, want %v", tag.Object(), tt.object)
			}
			
			if tag.ObjectType() != tt.typ {
				t.Errorf("ObjectType() = %v, want %v", tag.ObjectType(), tt.typ)
			}
			
			if tag.TagName() != tt.tag {
				t.Errorf("TagName() = %v, want %v", tag.TagName(), tt.tag)
			}
			
			if tag.Tagger().Name != tt.tagger.Name {
				t.Errorf("Tagger().Name = %v, want %v", tag.Tagger().Name, tt.tagger.Name)
			}
			
			if tag.Message() != tt.message {
				t.Errorf("Message() = %v, want %v", tag.Message(), tt.message)
			}
			
			if tag.ID().IsZero() {
				t.Error("ID() should not be zero")
			}
			
			if tag.Size() == 0 {
				t.Error("Size() should not be zero")
			}
		})
	}
}

func TestTag_Serialize(t *testing.T) {
	object, _ := NewObjectID("1234567890abcdef1234567890abcdef12345678")
	
	tagger := Signature{
		Name:  "Test Tagger",
		Email: "tagger@example.com",
		When:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	
	tag := NewTag(object, TypeCommit, "v1.0.0", tagger, "Release version 1.0.0\n")
	
	data, err := tag.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	
	// Verify serialized format
	expected := []string{
		"object 1234567890abcdef1234567890abcdef12345678",
		"type commit",
		"tag v1.0.0",
		"tagger Test Tagger <tagger@example.com> 1704110400 +0000",
		"",
		"Release version 1.0.0",
	}
	
	serialized := string(data)
	for _, exp := range expected {
		if !strings.Contains(serialized, exp) {
			t.Errorf("Serialized data missing: %s", exp)
		}
	}
}

func TestParseTag(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name: "valid tag",
			data: `object 1234567890abcdef1234567890abcdef12345678
type commit
tag v1.0.0
tagger Test Tagger <tagger@example.com> 1704110400 +0000

Release version 1.0.0
`,
			wantErr: false,
		},
		{
			name: "tag with multi-line message",
			data: `object 1234567890abcdef1234567890abcdef12345678
type commit
tag v2.0.0
tagger Test Tagger <tagger@example.com> 1704110400 +0000

Release version 2.0.0

This release includes:
- Feature A
- Feature B
`,
			wantErr: false,
		},
		{
			name: "tag pointing to tree",
			data: `object 4b825dc642cb6eb9a060e54bf8d69288fbee4904
type tree
tag tree-snapshot
tagger Test Tagger <tagger@example.com> 1704110400 +0000

Snapshot of tree
`,
			wantErr: false,
		},
		{
			name: "invalid object ID",
			data: `object invalid
type commit
tag v1.0.0
tagger Test Tagger <tagger@example.com> 1704110400 +0000

Test
`,
			wantErr: true,
		},
		{
			name: "invalid object type",
			data: `object 1234567890abcdef1234567890abcdef12345678
type invalid
tag v1.0.0
tagger Test Tagger <tagger@example.com> 1704110400 +0000

Test
`,
			wantErr: true,
		},
		{
			name: "invalid header format",
			data: `object
type commit
tag v1.0.0
tagger Test Tagger <tagger@example.com> 1704110400 +0000

Test
`,
			wantErr: true,
		},
		{
			name: "invalid tagger format",
			data: `object 1234567890abcdef1234567890abcdef12345678
type commit
tag v1.0.0
tagger Invalid Format

Test
`,
			wantErr: true,
		},
		{
			name: "unknown header",
			data: `object 1234567890abcdef1234567890abcdef12345678
type commit
tag v1.0.0
unknown-header some value
tagger Test Tagger <tagger@example.com> 1704110400 +0000

Test
`,
			wantErr: false, // Should ignore unknown headers
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, _ := NewObjectID("0000000000000000000000000000000000000000")
			tag, err := ParseTag(id, []byte(tt.data))
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tag == nil {
				t.Error("ParseTag() returned nil tag")
			}
		})
	}
}

func TestObjectType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		typ  ObjectType
		want bool
	}{
		{"blob", TypeBlob, true},
		{"tree", TypeTree, true},
		{"commit", TypeCommit, true},
		{"tag", TypeTag, true},
		{"invalid", ObjectType("invalid"), false},
		{"empty", ObjectType(""), false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typ.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}