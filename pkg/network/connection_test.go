package network

import (
	"net"
	"testing"
)

func TestNewConnectionTuple(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.1")
	dstIP := net.ParseIP("192.168.1.2")

	ft := NewConnectionTuple(srcIP, 8080, dstIP, 80)

	if !ft.SrcIP.Equal(srcIP) {
		t.Errorf("SrcIP = %s, want %s", ft.SrcIP, srcIP)
	}
	if ft.SrcPort != 8080 {
		t.Errorf("SrcPort = %d, want 8080", ft.SrcPort)
	}
	if !ft.DstIP.Equal(dstIP) {
		t.Errorf("DstIP = %s, want %s", ft.DstIP, dstIP)
	}
	if ft.DstPort != 80 {
		t.Errorf("DstPort = %d, want 80", ft.DstPort)
	}
}

func TestConnectionTupleString(t *testing.T) {
	ft := NewConnectionTuple(
		net.ParseIP("192.168.1.1"), 8080,
		net.ParseIP("192.168.1.2"), 80,
	)

	str := ft.String()
	expected := "192.168.1.1:8080 -> 192.168.1.2:80"

	if str != expected {
		t.Errorf("String() = %s, want %s", str, expected)
	}
}

func TestConnectionTupleReverse(t *testing.T) {
	ft := NewConnectionTuple(
		net.ParseIP("192.168.1.1"), 8080,
		net.ParseIP("192.168.1.2"), 80,
	)

	reversed := ft.Reverse()

	if !reversed.SrcIP.Equal(ft.DstIP) {
		t.Errorf("Reversed SrcIP = %s, want %s", reversed.SrcIP, ft.DstIP)
	}
	if reversed.SrcPort != ft.DstPort {
		t.Errorf("Reversed SrcPort = %d, want %d", reversed.SrcPort, ft.DstPort)
	}
	if !reversed.DstIP.Equal(ft.SrcIP) {
		t.Errorf("Reversed DstIP = %s, want %s", reversed.DstIP, ft.SrcIP)
	}
	if reversed.DstPort != ft.SrcPort {
		t.Errorf("Reversed DstPort = %d, want %d", reversed.DstPort, ft.SrcPort)
	}
}

func TestGenerateConnectionID(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.1")
	dstIP := net.ParseIP("192.168.1.2")

	id1 := GenerateConnectionID(srcIP, 8080, dstIP, 80)
	id2 := GenerateConnectionID(srcIP, 8080, dstIP, 80)

	// Same inputs should produce same ID
	if id1 != id2 {
		t.Errorf("IDs should be identical for same inputs: %s != %s", id1, id2)
	}

	// ID should not be empty
	if id1 == "" {
		t.Error("Connection ID should not be empty")
	}

	// ID should be 32 characters (16 bytes in hex)
	if len(id1) != 32 {
		t.Errorf("Connection ID length = %d, want 32", len(id1))
	}
}

func TestGenerateConnectionIDBidirectional(t *testing.T) {
	// This is the critical test: IDs must be the same regardless of direction
	srcIP := net.ParseIP("192.168.1.1")
	dstIP := net.ParseIP("192.168.1.2")

	// Forward direction: 192.168.1.1:8080 -> 192.168.1.2:80
	forwardID := GenerateConnectionID(srcIP, 8080, dstIP, 80)

	// Reverse direction: 192.168.1.2:80 -> 192.168.1.1:8080
	reverseID := GenerateConnectionID(dstIP, 80, srcIP, 8080)

	if forwardID != reverseID {
		t.Errorf("Bidirectional IDs should match:\nForward: %s\nReverse: %s",
			forwardID, reverseID)
	}
}

func TestGenerateConnectionIDUnique(t *testing.T) {
	// Different connections should have different IDs
	id1 := GenerateConnectionID(
		net.ParseIP("192.168.1.1"), 8080,
		net.ParseIP("192.168.1.2"), 80,
	)

	id2 := GenerateConnectionID(
		net.ParseIP("192.168.1.1"), 8081, // Different port
		net.ParseIP("192.168.1.2"), 80,
	)

	id3 := GenerateConnectionID(
		net.ParseIP("192.168.1.1"), 8080,
		net.ParseIP("192.168.1.3"), 80, // Different IP
	)

	if id1 == id2 {
		t.Error("Different source ports should produce different IDs")
	}

	if id1 == id3 {
		t.Error("Different destination IPs should produce different IDs")
	}
}

func TestGenerateConnectionIDFromTuple(t *testing.T) {
	ft := NewConnectionTuple(
		net.ParseIP("192.168.1.1"), 8080,
		net.ParseIP("192.168.1.2"), 80,
	)

	id := GenerateConnectionIDFromTuple(ft)

	if id == "" {
		t.Error("Connection ID should not be empty")
	}

	// Should match direct generation
	directID := GenerateConnectionID(ft.SrcIP, ft.SrcPort, ft.DstIP, ft.DstPort)
	if id != directID {
		t.Errorf("ID from tuple (%s) != direct ID (%s)", id, directID)
	}
}

func TestCompareIPs(t *testing.T) {
	tests := []struct {
		name string
		a    net.IP
		b    net.IP
		want int
	}{
		{
			name: "equal IPs",
			a:    net.ParseIP("192.168.1.1"),
			b:    net.ParseIP("192.168.1.1"),
			want: 0,
		},
		{
			name: "a < b",
			a:    net.ParseIP("192.168.1.1"),
			b:    net.ParseIP("192.168.1.2"),
			want: -1,
		},
		{
			name: "a > b",
			a:    net.ParseIP("192.168.1.2"),
			b:    net.ParseIP("192.168.1.1"),
			want: 1,
		},
		{
			name: "different length - a shorter",
			a:    net.ParseIP("192.168.1.1").To4(),
			b:    net.ParseIP("::1"),
			want: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareIPs(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareIPs(%s, %s) = %d, want %d",
					tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestConnectionIDConsistency(t *testing.T) {
	// Test that connection IDs remain consistent across multiple calls
	srcIP := net.ParseIP("10.1.1.1")
	dstIP := net.ParseIP("10.2.2.2")

	ids := make(map[string]bool)
	for _ = range 100 {
		id := GenerateConnectionID(srcIP, 12345, dstIP, 80)
		ids[id] = true
	}

	if len(ids) != 1 {
		t.Errorf("Got %d unique IDs, want 1 (should be consistent)", len(ids))
	}
}

func TestConnectionIDCollisionResistance(t *testing.T) {
	// Generate IDs for many different connections
	ids := make(map[string]bool)
	collisions := 0

	baseIP := "10.0"
	for i := range 100 {
		for j := range 100 {
			srcIP := net.ParseIP(baseIP + "." + string(rune(i)) + "." + string(rune(j)))
			dstIP := net.ParseIP(baseIP + "." + string(rune(j)) + "." + string(rune(i)))

			id := GenerateConnectionID(srcIP, uint16(i*256+j), dstIP, 80)
			if ids[id] {
				collisions++
			}
			ids[id] = true
		}
	}

	t.Logf("Generated %d unique IDs with %d collisions", len(ids), collisions)

	// With SHA256, collisions should be extremely rare
	if collisions > 10 {
		t.Errorf("Too many collisions: %d", collisions)
	}
}
