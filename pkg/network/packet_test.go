package network

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestParseIPPacket(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantErr  bool
		validate func(*testing.T, *IPPacket)
	}{
		{
			name: "valid IPv4 packet",
			data: func() []byte {
				pkt := make([]byte, 20)
				pkt[0] = 0x45                               // Version 4, header length 20 bytes
				pkt[1] = 0x00                               // TOS
				binary.BigEndian.PutUint16(pkt[2:4], 20)    // Total length
				binary.BigEndian.PutUint16(pkt[4:6], 12345) // ID
				pkt[8] = 64                                 // TTL
				pkt[9] = uint8(ProtocolTCP)                 // Protocol
				copy(pkt[12:16], net.ParseIP("192.168.1.1").To4())
				copy(pkt[16:20], net.ParseIP("192.168.1.2").To4())
				return pkt
			}(),
			wantErr: false,
			validate: func(t *testing.T, p *IPPacket) {
				if p.Version != 4 {
					t.Errorf("Version = %d, want 4", p.Version)
				}
				if p.Protocol != ProtocolTCP {
					t.Errorf("Protocol = %d, want %d", p.Protocol, ProtocolTCP)
				}
				if p.TTL != 64 {
					t.Errorf("TTL = %d, want 64", p.TTL)
				}
				if !p.SrcIP.Equal(net.ParseIP("192.168.1.1")) {
					t.Errorf("SrcIP = %s, want 192.168.1.1", p.SrcIP)
				}
				if !p.DstIP.Equal(net.ParseIP("192.168.1.2")) {
					t.Errorf("DstIP = %s, want 192.168.1.2", p.DstIP)
				}
			},
		},
		{
			name:    "packet too small",
			data:    []byte{0x45, 0x00, 0x00},
			wantErr: true,
		},
		{
			name: "unsupported IP version",
			data: func() []byte {
				pkt := make([]byte, 20)
				pkt[0] = 0x65 // Version 6 (IPv6)
				return pkt
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt, err := ParseIPPacket(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIPPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.validate != nil && pkt != nil {
				tt.validate(t, pkt)
			}
		})
	}
}

func TestParseTCPPacket(t *testing.T) {
	tests := []struct {
		name     string
		payload  []byte
		wantErr  bool
		validate func(*testing.T, *TCPPacket)
	}{
		{
			name: "valid TCP packet",
			payload: func() []byte {
				pkt := make([]byte, 20)
				binary.BigEndian.PutUint16(pkt[0:2], 8080)   // Source port
				binary.BigEndian.PutUint16(pkt[2:4], 80)     // Dest port
				binary.BigEndian.PutUint32(pkt[4:8], 12345)  // Seq num
				binary.BigEndian.PutUint32(pkt[8:12], 67890) // Ack num
				pkt[12] = 0x50                               // Data offset = 20 bytes
				pkt[13] = uint8(FlagSYN | FlagACK)
				binary.BigEndian.PutUint16(pkt[14:16], 65535) // Window
				return pkt
			}(),
			wantErr: false,
			validate: func(t *testing.T, tcp *TCPPacket) {
				if tcp.SrcPort != 8080 {
					t.Errorf("SrcPort = %d, want 8080", tcp.SrcPort)
				}
				if tcp.DstPort != 80 {
					t.Errorf("DstPort = %d, want 80", tcp.DstPort)
				}
				if tcp.SeqNum != 12345 {
					t.Errorf("SeqNum = %d, want 12345", tcp.SeqNum)
				}
				if !tcp.HasFlag(FlagSYN) {
					t.Error("SYN flag not set")
				}
				if !tcp.HasFlag(FlagACK) {
					t.Error("ACK flag not set")
				}
				if tcp.HasFlag(FlagRST) {
					t.Error("RST flag should not be set")
				}
			},
		},
		{
			name:    "packet too small",
			payload: []byte{0x00, 0x50, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tcp, err := ParseTCPPacket(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTCPPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.validate != nil && tcp != nil {
				tt.validate(t, tcp)
			}
		})
	}
}

func TestIPPacketSerialize(t *testing.T) {
	// Create a packet
	pkt := &IPPacket{
		Version:    4,
		HeaderLen:  20,
		TOS:        0,
		TotalLen:   20,
		ID:         12345,
		Flags:      0,
		FragOffset: 0,
		TTL:        64,
		Protocol:   ProtocolTCP,
		Checksum:   0,
		SrcIP:      net.ParseIP("192.168.1.1").To4(),
		DstIP:      net.ParseIP("192.168.1.2").To4(),
		Payload:    []byte{},
	}

	// Serialize
	data := pkt.Serialize()

	// Parse it back
	parsed, err := ParseIPPacket(data)
	if err != nil {
		t.Fatalf("Failed to parse serialized packet: %v", err)
	}

	// Verify fields
	if parsed.Version != pkt.Version {
		t.Errorf("Version = %d, want %d", parsed.Version, pkt.Version)
	}
	if parsed.TTL != pkt.TTL {
		t.Errorf("TTL = %d, want %d", parsed.TTL, pkt.TTL)
	}
	if parsed.Protocol != pkt.Protocol {
		t.Errorf("Protocol = %d, want %d", parsed.Protocol, pkt.Protocol)
	}
	if !parsed.SrcIP.Equal(pkt.SrcIP) {
		t.Errorf("SrcIP = %s, want %s", parsed.SrcIP, pkt.SrcIP)
	}
	if !parsed.DstIP.Equal(pkt.DstIP) {
		t.Errorf("DstIP = %s, want %s", parsed.DstIP, pkt.DstIP)
	}
}

func TestTCPPacketSerialize(t *testing.T) {
	tcp := &TCPPacket{
		SrcPort:    8080,
		DstPort:    80,
		SeqNum:     12345,
		AckNum:     67890,
		DataOffset: 20,
		Flags:      FlagSYN | FlagACK,
		Window:     65535,
		Checksum:   0,
		Urgent:     0,
		Payload:    []byte("test data"),
	}

	// Serialize
	data := tcp.Serialize()

	// Parse it back
	parsed, err := ParseTCPPacket(data)
	if err != nil {
		t.Fatalf("Failed to parse serialized TCP packet: %v", err)
	}

	// Verify fields
	if parsed.SrcPort != tcp.SrcPort {
		t.Errorf("SrcPort = %d, want %d", parsed.SrcPort, tcp.SrcPort)
	}
	if parsed.DstPort != tcp.DstPort {
		t.Errorf("DstPort = %d, want %d", parsed.DstPort, tcp.DstPort)
	}
	if parsed.SeqNum != tcp.SeqNum {
		t.Errorf("SeqNum = %d, want %d", parsed.SeqNum, tcp.SeqNum)
	}
	if parsed.Flags != tcp.Flags {
		t.Errorf("Flags = %d, want %d", parsed.Flags, tcp.Flags)
	}
	if string(parsed.Payload) != "test data" {
		t.Errorf("Payload = %s, want 'test data'", parsed.Payload)
	}
}

func TestTCPFlags(t *testing.T) {
	tcp := &TCPPacket{
		Flags: FlagSYN | FlagACK,
	}

	tests := []struct {
		flag TCPFlags
		want bool
	}{
		{FlagSYN, true},
		{FlagACK, true},
		{FlagRST, false},
		{FlagFIN, false},
		{FlagPSH, false},
		{FlagURG, false},
	}

	for _, tt := range tests {
		if got := tcp.HasFlag(tt.flag); got != tt.want {
			t.Errorf("HasFlag(%v) = %v, want %v", tt.flag, got, tt.want)
		}
	}
}

func TestIPPacketWithPayload(t *testing.T) {
	// Create TCP payload
	tcpPayload := []byte("HTTP/1.1 200 OK\r\n\r\n")

	// Create TCP header
	tcp := &TCPPacket{
		SrcPort:    80,
		DstPort:    54321,
		SeqNum:     1000,
		AckNum:     2000,
		DataOffset: 20,
		Flags:      FlagACK | FlagPSH,
		Window:     65535,
		Payload:    tcpPayload,
	}

	tcpData := tcp.Serialize()

	// Create IP packet with TCP payload
	ipPkt := &IPPacket{
		Version:   4,
		HeaderLen: 20,
		TOS:       0,
		TotalLen:  uint16(20 + len(tcpData)),
		ID:        54321,
		TTL:       64,
		Protocol:  ProtocolTCP,
		SrcIP:     net.ParseIP("10.0.0.1").To4(),
		DstIP:     net.ParseIP("10.0.0.2").To4(),
		Payload:   tcpData,
	}

	// Serialize and parse
	data := ipPkt.Serialize()
	parsed, err := ParseIPPacket(data)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Parse TCP from IP payload
	parsedTCP, err := ParseTCPPacket(parsed.Payload)
	if err != nil {
		t.Fatalf("Failed to parse TCP: %v", err)
	}

	// Verify
	if parsedTCP.SrcPort != 80 {
		t.Errorf("TCP SrcPort = %d, want 80", parsedTCP.SrcPort)
	}
	if string(parsedTCP.Payload) != string(tcpPayload) {
		t.Errorf("TCP Payload = %s, want %s", parsedTCP.Payload, tcpPayload)
	}
}
