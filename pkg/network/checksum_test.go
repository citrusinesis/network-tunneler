package network

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestCalculateChecksum(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want uint16
	}{
		{
			name: "RFC 1071 example 1",
			data: []byte{0x00, 0x01, 0xf2, 0x03, 0xf4, 0xf5, 0xf6, 0xf7},
			want: 0x220d,
		},
		{
			name: "empty data",
			data: []byte{},
			want: 0xffff,
		},
		{
			name: "single byte",
			data: []byte{0xAB},
			want: 0x54ff,
		},
		{
			name: "two bytes",
			data: []byte{0x12, 0x34},
			want: 0xedcb,
		},
		{
			name: "all zeros",
			data: []byte{0x00, 0x00, 0x00, 0x00},
			want: 0xffff,
		},
		{
			name: "all ones",
			data: []byte{0xff, 0xff, 0xff, 0xff},
			want: 0x0000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateChecksum(tt.data)
			if got != tt.want {
				t.Errorf("CalculateChecksum() = 0x%04x, want 0x%04x", got, tt.want)
			}
		})
	}
}

func TestCalculateIPChecksum(t *testing.T) {
	header := make([]byte, 20)
	header[0] = 0x45                               // Version 4, header length 20
	header[1] = 0x00                               // TOS
	binary.BigEndian.PutUint16(header[2:4], 60)    // Total length
	binary.BigEndian.PutUint16(header[4:6], 12345) // ID
	header[8] = 64                                 // TTL
	header[9] = uint8(ProtocolTCP)                 // Protocol
	copy(header[12:16], net.ParseIP("192.168.1.1").To4())
	copy(header[16:20], net.ParseIP("192.168.1.2").To4())

	checksum := CalculateIPChecksum(header)

	if checksum == 0 {
		t.Error("IP checksum should not be zero")
	}

	binary.BigEndian.PutUint16(header[10:12], checksum)

	verification := CalculateChecksum(header)
	if verification != 0xffff && verification != 0x0000 {
		t.Errorf("Checksum verification failed: got 0x%04x, want 0xffff or 0x0000", verification)
	}
}

func TestCalculateTCPChecksum(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.1").To4()
	dstIP := net.ParseIP("192.168.1.2").To4()

	tcpSegment := make([]byte, 20)
	binary.BigEndian.PutUint16(tcpSegment[0:2], 8080)   // Source port
	binary.BigEndian.PutUint16(tcpSegment[2:4], 80)     // Dest port
	binary.BigEndian.PutUint32(tcpSegment[4:8], 12345)  // Seq
	binary.BigEndian.PutUint32(tcpSegment[8:12], 67890) // Ack
	tcpSegment[12] = 0x50                               // Data offset
	tcpSegment[13] = uint8(FlagSYN)

	checksum := CalculateTCPChecksum(srcIP, dstIP, tcpSegment)

	if checksum == 0 {
		t.Error("TCP checksum should not be zero")
	}

	binary.BigEndian.PutUint16(tcpSegment[16:18], checksum)

	pseudoHeader := make([]byte, 12)
	copy(pseudoHeader[0:4], srcIP)
	copy(pseudoHeader[4:8], dstIP)
	pseudoHeader[9] = uint8(ProtocolTCP)
	binary.BigEndian.PutUint16(pseudoHeader[10:12], uint16(len(tcpSegment)))

	combined := append(pseudoHeader, tcpSegment...)
	verification := CalculateChecksum(combined)

	if verification != 0xffff && verification != 0x0000 {
		t.Errorf("TCP checksum verification failed: got 0x%04x, want 0xffff or 0x0000", verification)
	}
}

func TestCalculateUDPChecksum(t *testing.T) {
	srcIP := net.ParseIP("10.0.0.1").To4()
	dstIP := net.ParseIP("10.0.0.2").To4()

	// Create a minimal UDP datagram
	udpSegment := make([]byte, 8)
	binary.BigEndian.PutUint16(udpSegment[0:2], 53) // Source port
	binary.BigEndian.PutUint16(udpSegment[2:4], 53) // Dest port
	binary.BigEndian.PutUint16(udpSegment[4:6], 8)  // Length

	checksum := CalculateUDPChecksum(srcIP, dstIP, udpSegment)

	// UDP checksum can be 0xffff if calculated checksum is 0
	if checksum == 0 {
		t.Error("UDP checksum should not be zero (should be 0xffff if calculation yields 0)")
	}
}

func TestIPPacketRecalculateIPChecksum(t *testing.T) {
	pkt := &IPPacket{
		Version:   4,
		HeaderLen: 20,
		TOS:       0,
		TotalLen:  40,
		ID:        12345,
		TTL:       64,
		Protocol:  ProtocolTCP,
		Checksum:  0, // Start with zero
		SrcIP:     net.ParseIP("192.168.1.1").To4(),
		DstIP:     net.ParseIP("192.168.1.2").To4(),
		Payload:   make([]byte, 20),
	}

	// Recalculate
	pkt.RecalculateIPChecksum()

	// Verify checksum is set
	if pkt.Checksum == 0 {
		t.Error("Checksum should be set after recalculation")
	}

	// Serialize and verify
	data := pkt.Serialize()
	verification := CalculateChecksum(data[:20])

	if verification != 0xffff && verification != 0x0000 {
		t.Errorf("IP checksum verification failed: got 0x%04x, want 0xffff or 0x0000", verification)
	}
}

func TestIPPacketRecalculateTCPChecksum(t *testing.T) {
	// Create TCP segment
	tcpSegment := make([]byte, 20)
	binary.BigEndian.PutUint16(tcpSegment[0:2], 8080) // Src port
	binary.BigEndian.PutUint16(tcpSegment[2:4], 80)   // Dst port
	tcpSegment[12] = 0x50                             // Data offset

	pkt := &IPPacket{
		Version:   4,
		HeaderLen: 20,
		Protocol:  ProtocolTCP,
		SrcIP:     net.ParseIP("192.168.1.1").To4(),
		DstIP:     net.ParseIP("192.168.1.2").To4(),
		Payload:   tcpSegment,
	}

	// Recalculate TCP checksum
	err := pkt.RecalculateTCPChecksum()
	if err != nil {
		t.Fatalf("RecalculateTCPChecksum failed: %v", err)
	}

	// The checksum should be stored in the IP packet's Checksum field
	if pkt.Checksum == 0 {
		t.Error("TCP checksum should be calculated")
	}
}

func TestIPPacketRecalculateChecksums(t *testing.T) {
	// Create TCP segment
	tcpSegment := make([]byte, 20)
	binary.BigEndian.PutUint16(tcpSegment[0:2], 8080)
	binary.BigEndian.PutUint16(tcpSegment[2:4], 80)
	tcpSegment[12] = 0x50

	pkt := &IPPacket{
		Version:   4,
		HeaderLen: 20,
		TOS:       0,
		TotalLen:  40,
		ID:        12345,
		TTL:       64,
		Protocol:  ProtocolTCP,
		SrcIP:     net.ParseIP("192.168.1.1").To4(),
		DstIP:     net.ParseIP("192.168.1.2").To4(),
		Payload:   tcpSegment,
	}

	// Recalculate all checksums
	err := pkt.RecalculateChecksums()
	if err != nil {
		t.Fatalf("RecalculateChecksums failed: %v", err)
	}

	// Both IP and TCP checksums should be set
	if pkt.Checksum == 0 {
		t.Error("IP checksum should be set")
	}

	// Verify IP checksum
	data := pkt.Serialize()
	ipVerification := CalculateChecksum(data[:20])
	if ipVerification != 0xffff && ipVerification != 0x0000 {
		t.Errorf("IP checksum verification failed: got 0x%04x, want 0xffff or 0x0000", ipVerification)
	}
}

func BenchmarkCalculateChecksum(b *testing.B) {
	data := make([]byte, 1500) // MTU-sized packet
	for i := range data {
		data[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateChecksum(data)
	}
}

func BenchmarkCalculateIPChecksum(b *testing.B) {
	header := make([]byte, 20)
	header[0] = 0x45
	header[8] = 64
	header[9] = uint8(ProtocolTCP)
	copy(header[12:16], net.ParseIP("192.168.1.1").To4())
	copy(header[16:20], net.ParseIP("192.168.1.2").To4())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateIPChecksum(header)
	}
}

func BenchmarkCalculateTCPChecksum(b *testing.B) {
	srcIP := net.ParseIP("192.168.1.1").To4()
	dstIP := net.ParseIP("192.168.1.2").To4()
	tcpSegment := make([]byte, 20)
	binary.BigEndian.PutUint16(tcpSegment[0:2], 8080)
	binary.BigEndian.PutUint16(tcpSegment[2:4], 80)
	tcpSegment[12] = 0x50

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateTCPChecksum(srcIP, dstIP, tcpSegment)
	}
}

func BenchmarkRecalculateChecksums(b *testing.B) {
	tcpSegment := make([]byte, 1460) // Typical TCP payload
	binary.BigEndian.PutUint16(tcpSegment[0:2], 8080)
	binary.BigEndian.PutUint16(tcpSegment[2:4], 80)
	tcpSegment[12] = 0x50

	pkt := &IPPacket{
		Version:   4,
		HeaderLen: 20,
		Protocol:  ProtocolTCP,
		TTL:       64,
		SrcIP:     net.ParseIP("192.168.1.1").To4(),
		DstIP:     net.ParseIP("192.168.1.2").To4(),
		Payload:   tcpSegment,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pkt.RecalculateChecksums()
	}
}
