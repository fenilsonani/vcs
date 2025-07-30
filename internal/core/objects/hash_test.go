package objects

import (
	"testing"
)

func TestObjectID_String(t *testing.T) {
	tests := []struct {
		name     string
		id       ObjectID
		expected string
	}{
		{
			name:     "zero ID",
			id:       ObjectID{},
			expected: "0000000000000000000000000000000000000000",
		},
		{
			name:     "sample ID",
			id:       ObjectID{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc},
			expected: "123456789abcdef0112233445566778899aabbcc",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.String(); got != tt.expected {
				t.Errorf("ObjectID.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewObjectID(t *testing.T) {
	tests := []struct {
		name    string
		hexStr  string
		wantErr bool
	}{
		{
			name:    "valid ID",
			hexStr:  "123456789abcdef0112233445566778899aabbcc",
			wantErr: false,
		},
		{
			name:    "too short",
			hexStr:  "123456789abcdef",
			wantErr: true,
		},
		{
			name:    "too long",
			hexStr:  "123456789abcdef0112233445566778899aabbcc00",
			wantErr: true,
		},
		{
			name:    "invalid hex",
			hexStr:  "123456789abcdef0112233445566778899aabbcg",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObjectID(tt.hexStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewObjectID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.hexStr {
				t.Errorf("NewObjectID() = %v, want %v", got.String(), tt.hexStr)
			}
		})
	}
}

func TestComputeHash(t *testing.T) {
	tests := []struct {
		name     string
		objType  ObjectType
		data     []byte
		expected string
	}{
		{
			name:     "empty blob",
			objType:  TypeBlob,
			data:     []byte{},
			expected: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391",
		},
		{
			name:     "hello world blob",
			objType:  TypeBlob,
			data:     []byte("hello world\n"),
			expected: "3b18e512dba79e4c8300dd08aeb37f8e728b8dad",
		},
		{
			name:     "test content",
			objType:  TypeBlob,
			data:     []byte("test content"),
			expected: "08cf6101416f0ce0dda3c80e627f333854c4085c",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeHash(tt.objType, tt.data)
			if got.String() != tt.expected {
				t.Errorf("ComputeHash() = %v, want %v", got.String(), tt.expected)
			}
		})
	}
}

func TestObjectID_IsZero(t *testing.T) {
	tests := []struct {
		name string
		id   ObjectID
		want bool
	}{
		{
			name: "zero ID",
			id:   ObjectID{},
			want: true,
		},
		{
			name: "non-zero ID",
			id:   ObjectID{0x12},
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.IsZero(); got != tt.want {
				t.Errorf("ObjectID.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}