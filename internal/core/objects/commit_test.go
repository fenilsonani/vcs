package objects

import (
	"strings"
	"testing"
	"time"
)

func TestNewCommit(t *testing.T) {
	tree, _ := NewObjectID("4b825dc642cb6eb9a060e54bf8d69288fbee4904")
	parent1, _ := NewObjectID("1234567890abcdef1234567890abcdef12345678")
	parent2, _ := NewObjectID("abcdef1234567890abcdef1234567890abcdef12")
	
	author := Signature{
		Name:  "Test Author",
		Email: "author@example.com",
		When:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	
	committer := Signature{
		Name:  "Test Committer",
		Email: "committer@example.com",
		When:  time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
	}
	
	tests := []struct {
		name     string
		tree     ObjectID
		parents  []ObjectID
		author   Signature
		committer Signature
		message  string
	}{
		{
			name:      "simple commit",
			tree:      tree,
			parents:   nil,
			author:    author,
			committer: committer,
			message:   "Initial commit\n",
		},
		{
			name:      "commit with one parent",
			tree:      tree,
			parents:   []ObjectID{parent1},
			author:    author,
			committer: committer,
			message:   "Second commit\n",
		},
		{
			name:      "merge commit",
			tree:      tree,
			parents:   []ObjectID{parent1, parent2},
			author:    author,
			committer: committer,
			message:   "Merge branch 'feature'\n",
		},
		{
			name:      "commit with multi-line message",
			tree:      tree,
			parents:   []ObjectID{parent1},
			author:    author,
			committer: committer,
			message:   "Fix bug #123\n\nThis commit fixes the issue where...\n",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commit := NewCommit(tt.tree, tt.parents, tt.author, tt.committer, tt.message)
			
			if commit.Type() != TypeCommit {
				t.Errorf("Type() = %v, want %v", commit.Type(), TypeCommit)
			}
			
			if commit.Tree() != tt.tree {
				t.Errorf("Tree() = %v, want %v", commit.Tree(), tt.tree)
			}
			
			if len(commit.Parents()) != len(tt.parents) {
				t.Errorf("Parents() length = %v, want %v", len(commit.Parents()), len(tt.parents))
			}
			
			for i, parent := range commit.Parents() {
				if parent != tt.parents[i] {
					t.Errorf("Parents()[%d] = %v, want %v", i, parent, tt.parents[i])
				}
			}
			
			if commit.Author().Name != tt.author.Name {
				t.Errorf("Author().Name = %v, want %v", commit.Author().Name, tt.author.Name)
			}
			
			if commit.Committer().Email != tt.committer.Email {
				t.Errorf("Committer().Email = %v, want %v", commit.Committer().Email, tt.committer.Email)
			}
			
			if commit.Message() != tt.message {
				t.Errorf("Message() = %v, want %v", commit.Message(), tt.message)
			}
			
			if commit.ID().IsZero() {
				t.Error("ID() should not be zero")
			}
			
			if commit.Size() == 0 {
				t.Error("Size() should not be zero")
			}
		})
	}
}

func TestCommit_Serialize(t *testing.T) {
	tree, _ := NewObjectID("4b825dc642cb6eb9a060e54bf8d69288fbee4904")
	parent, _ := NewObjectID("1234567890abcdef1234567890abcdef12345678")
	
	author := Signature{
		Name:  "Test Author",
		Email: "author@example.com",
		When:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	
	committer := Signature{
		Name:  "Test Committer",
		Email: "committer@example.com",
		When:  time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
	}
	
	commit := NewCommit(tree, []ObjectID{parent}, author, committer, "Test commit\n")
	
	data, err := commit.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	
	// Verify serialized format
	expected := []string{
		"tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904",
		"parent 1234567890abcdef1234567890abcdef12345678",
		"author Test Author <author@example.com> 1704110400 +0000",
		"committer Test Committer <committer@example.com> 1704114000 +0000",
		"",
		"Test commit",
	}
	
	serialized := string(data)
	for _, exp := range expected {
		if !strings.Contains(serialized, exp) {
			t.Errorf("Serialized data missing: %s", exp)
		}
	}
}

func TestParseCommit(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name: "valid commit",
			data: `tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904
parent 1234567890abcdef1234567890abcdef12345678
author Test Author <author@example.com> 1704110400 +0000
committer Test Committer <committer@example.com> 1704114000 +0000

Test commit
`,
			wantErr: false,
		},
		{
			name: "commit without parent",
			data: `tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904
author Test Author <author@example.com> 1704110400 +0000
committer Test Committer <committer@example.com> 1704114000 +0000

Initial commit
`,
			wantErr: false,
		},
		{
			name: "commit with multiple parents",
			data: `tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904
parent 1234567890abcdef1234567890abcdef12345678
parent abcdef1234567890abcdef1234567890abcdef12
author Test Author <author@example.com> 1704110400 +0000
committer Test Committer <committer@example.com> 1704114000 +0000

Merge commit
`,
			wantErr: false,
		},
		{
			name: "invalid tree ID",
			data: `tree invalid
author Test Author <author@example.com> 1704110400 +0000
committer Test Committer <committer@example.com> 1704114000 +0000

Test
`,
			wantErr: true,
		},
		{
			name: "invalid parent ID",
			data: `tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904
parent invalid
author Test Author <author@example.com> 1704110400 +0000
committer Test Committer <committer@example.com> 1704114000 +0000

Test
`,
			wantErr: true,
		},
		{
			name: "invalid header format",
			data: `tree
author Test Author <author@example.com> 1704110400 +0000
committer Test Committer <committer@example.com> 1704114000 +0000

Test
`,
			wantErr: true,
		},
		{
			name: "invalid author format",
			data: `tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904
author Invalid Format
committer Test Committer <committer@example.com> 1704114000 +0000

Test
`,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, _ := NewObjectID("0000000000000000000000000000000000000000")
			commit, err := ParseCommit(id, []byte(tt.data))
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && commit == nil {
				t.Error("ParseCommit() returned nil commit")
			}
		})
	}
}

func TestSignature_String(t *testing.T) {
	tests := []struct {
		name string
		sig  Signature
		want string
	}{
		{
			name: "UTC timezone",
			sig: Signature{
				Name:  "Test User",
				Email: "test@example.com",
				When:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			want: "Test User <test@example.com> 1704110400 +0000",
		},
		{
			name: "positive timezone",
			sig: Signature{
				Name:  "Test User",
				Email: "test@example.com",
				When:  time.Date(2024, 1, 1, 10, 0, 0, 0, time.FixedZone("", 2*3600)),
			},
			want: "Test User <test@example.com> 1704096000 +0200",
		},
		{
			name: "negative timezone",
			sig: Signature{
				Name:  "Test User",
				Email: "test@example.com",
				When:  time.Date(2024, 1, 1, 17, 0, 0, 0, time.FixedZone("", -5*3600)),
			},
			want: "Test User <test@example.com> 1704146400 -0500",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sig.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSignatureLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    Signature
		wantErr bool
	}{
		{
			name: "valid signature",
			line: "Test User <test@example.com> 1704110400 +0000",
			want: Signature{
				Name:  "Test User",
				Email: "test@example.com",
				When:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "signature with spaces in name",
			line: "First Middle Last <email@example.com> 1704110400 +0000",
			want: Signature{
				Name:  "First Middle Last",
				Email: "email@example.com",
				When:  time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name:    "missing email brackets",
			line:    "Test User test@example.com 1704110400 +0000",
			wantErr: true,
		},
		{
			name:    "missing timestamp",
			line:    "Test User <test@example.com>",
			wantErr: true,
		},
		{
			name:    "invalid timestamp format",
			line:    "Test User <test@example.com> invalid +0000",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSignatureLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSignatureLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if got.Name != tt.want.Name {
					t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
				}
				if got.Email != tt.want.Email {
					t.Errorf("Email = %v, want %v", got.Email, tt.want.Email)
				}
				if !got.When.Equal(tt.want.When) {
					t.Errorf("When = %v, want %v", got.When, tt.want.When)
				}
			}
		})
	}
}