package network

import (
	"encoding/binary"
	"net"
)

// RFC 1071 - Computing the Internet Checksum
// https://www.ietf.org/rfc/rfc1071.txt
func CalculateChecksum(data []byte) uint16 {
	var sum uint32

	// Add 16-bit words (2 byte)
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}

	// Odd number addings
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	// Carry folding -> carry add to next 32-bit word.
	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	// Return one's complement
	return ^uint16(sum)
}

func CalculateIPChecksum(ipHeader []byte) uint16 {
	header := make([]byte, len(ipHeader))
	copy(header, ipHeader)
	header[10] = 0
	header[11] = 0

	return CalculateChecksum(header)
}

// RFC 793 - TRANSMISSION CONTROL PROTOCOL §3.1
// https://www.ietf.org/rfc/rfc793.txt
// ref) https://www.baeldung.com/cs/pseudo-header-tcp
//
// Pseudo Header Structure (12 bytes):
// +--------+--------+--------+--------+
// |          Source Address           |  (4 bytes)
// +--------+--------+--------+--------+
// |        Destination Address        |  (4 bytes)
// +--------+--------+--------+--------+
// |  zero  |  PTCL  |    TCP Length   |  (4 bytes)
// +--------+--------+--------+--------+
type pseudoHeader struct {
	SrcIP    [4]byte
	DstIP    [4]byte
	Zero     uint8
	Protocol uint8
	Length   uint16
}

func (ph *pseudoHeader) Serialize() []byte {
	buf := make([]byte, 12)
	copy(buf[0:4], ph.SrcIP[:])
	copy(buf[4:8], ph.DstIP[:])
	buf[8] = ph.Zero
	buf[9] = ph.Protocol
	binary.BigEndian.PutUint16(buf[10:12], ph.Length)
	return buf
}

func newPseudoHeader(srcIP, dstIP net.IP, protocol uint8, length uint16) *pseudoHeader {
	ph := &pseudoHeader{
		Zero:     0,
		Protocol: protocol,
		Length:   length,
	}
	copy(ph.SrcIP[:], srcIP.To4())
	copy(ph.DstIP[:], dstIP.To4())
	return ph
}

func CalculateTCPChecksum(srcIP, dstIP net.IP, tcpSegment []byte) uint16 {
	ph := newPseudoHeader(srcIP, dstIP, uint8(ProtocolTCP), uint16(len(tcpSegment)))

	segment := make([]byte, len(tcpSegment))
	copy(segment, tcpSegment)
	segment[16] = 0
	segment[17] = 0

	combined := append(ph.Serialize(), segment...)
	return CalculateChecksum(combined)
}

func CalculateUDPChecksum(srcIP, dstIP net.IP, udpSegment []byte) uint16 {
	ph := newPseudoHeader(srcIP, dstIP, uint8(ProtocolUDP), uint16(len(udpSegment)))

	segment := make([]byte, len(udpSegment))
	copy(segment, udpSegment)
	segment[6] = 0
	segment[7] = 0

	combined := append(ph.Serialize(), segment...)

	checksum := CalculateChecksum(combined)
	// UDP uses 0xFFFF for zero checksum (RFC 768)
	if checksum == 0 {
		return 0xFFFF
	}
	return checksum
}

func (p *IPPacket) RecalculateIPChecksum() {
	header := make([]byte, p.HeaderLen)
	header[0] = (p.Version << 4) | (p.HeaderLen / 4)
	header[1] = p.TOS
	binary.BigEndian.PutUint16(header[2:4], p.TotalLen)
	binary.BigEndian.PutUint16(header[4:6], p.ID)

	flagsAndOffset := (uint16(p.Flags) << 13) | (p.FragOffset & 0x1FFF)
	binary.BigEndian.PutUint16(header[6:8], flagsAndOffset)

	header[8] = p.TTL
	header[9] = uint8(p.Protocol)

	copy(header[12:16], p.SrcIP.To4())
	copy(header[16:20], p.DstIP.To4())

	if len(p.Options) > 0 {
		copy(header[20:], p.Options)
	}

	p.Checksum = CalculateIPChecksum(header)
}

func (p *IPPacket) RecalculateTCPChecksum() error {
	if p.Protocol != ProtocolTCP {
		return nil
	}

	p.Checksum = CalculateTCPChecksum(p.SrcIP, p.DstIP, p.Payload)
	return nil
}

func (p *IPPacket) RecalculateUDPChecksum() error {
	if p.Protocol != ProtocolUDP {
		return nil
	}

	p.Checksum = CalculateUDPChecksum(p.SrcIP, p.DstIP, p.Payload)
	return nil
}

func (p *IPPacket) RecalculateChecksums() error {
	switch p.Protocol {
	case ProtocolTCP:
		if err := p.RecalculateTCPChecksum(); err != nil {
			return err
		}
	case ProtocolUDP:
		if err := p.RecalculateUDPChecksum(); err != nil {
			return err
		}
	}

	p.RecalculateIPChecksum()

	return nil
}
