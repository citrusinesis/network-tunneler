package network

import (
	"strings"
	"testing"
)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name       string
		byteLength int
		wantLen    int
	}{
		{"4 bytes", 4, 8},
		{"8 bytes", 8, 16},
		{"16 bytes", 16, 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := GenerateID(tt.byteLength)
			if err != nil {
				t.Errorf("GenerateID() error = %v", err)
				return
			}
			if len(id) != tt.wantLen {
				t.Errorf("GenerateID() length = %d, want %d", len(id), tt.wantLen)
			}
		})
	}
}

func TestGenerateAgentID(t *testing.T) {
	id, err := GenerateAgentID()
	if err != nil {
		t.Fatalf("GenerateAgentID() error = %v", err)
	}

	if !strings.HasPrefix(id, "agent-") {
		t.Errorf("GenerateAgentID() = %v, want prefix 'agent-'", id)
	}

	if len(id) != 22 {
		t.Errorf("GenerateAgentID() length = %d, want 22", len(id))
	}
}

func TestGenerateImplantID(t *testing.T) {
	id, err := GenerateImplantID()
	if err != nil {
		t.Fatalf("GenerateImplantID() error = %v", err)
	}

	if !strings.HasPrefix(id, "implant-") {
		t.Errorf("GenerateImplantID() = %v, want prefix 'implant-'", id)
	}

	if len(id) != 24 {
		t.Errorf("GenerateImplantID() length = %d, want 24", len(id))
	}
}

func TestGenerateRandomConnectionID(t *testing.T) {
	id, err := GenerateRandomConnectionID()
	if err != nil {
		t.Fatalf("GenerateRandomConnectionID() error = %v", err)
	}

	if !strings.HasPrefix(id, "conn-") {
		t.Errorf("GenerateRandomConnectionID() = %v, want prefix 'conn-'", id)
	}

	if len(id) != 21 {
		t.Errorf("GenerateRandomConnectionID() length = %d, want 21", len(id))
	}
}

func TestIDUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for _ = range 1000 {
		id, err := GenerateAgentID()
		if err != nil {
			t.Fatalf("GenerateAgentID() error = %v", err)
		}
		if seen[id] {
			t.Errorf("GenerateAgentID() produced duplicate: %s", id)
		}
		seen[id] = true
	}
}
