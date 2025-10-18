package network

import (
	"encoding/binary"
	"fmt"
	"net"
)

type IPProtocol uint8

const (
	ProtocolICMP IPProtocol = 1
	ProtocolTCP  IPProtocol = 6
	ProtocolUDP  IPProtocol = 17
)

type TCPFlags uint8

const (
	FlagFIN TCPFlags = 1 << 0
	FlagSYN TCPFlags = 1 << 1
	FlagRST TCPFlags = 1 << 2
	FlagPSH TCPFlags = 1 << 3
	FlagACK TCPFlags = 1 << 4
	FlagURG TCPFlags = 1 << 5
)

type IPPacket struct {
	Version    uint8
	HeaderLen  uint8
	TOS        uint8
	TotalLen   uint16
	ID         uint16
	Flags      uint8
	FragOffset uint16
	TTL        uint8
	Protocol   IPProtocol
	Checksum   uint16
	SrcIP      net.IP
	DstIP      net.IP
	Options    []byte
	Payload    []byte
}

type TCPPacket struct {
	SrcPort    uint16
	DstPort    uint16
	SeqNum     uint32
	AckNum     uint32
	DataOffset uint8
	Flags      TCPFlags
	Window     uint16
	Checksum   uint16
	Urgent     uint16
	Options    []byte
	Payload    []byte
}

func ParseIPPacket(data []byte) (*IPPacket, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("packet too small for IP header: %d bytes", len(data))
	}

	pkt := &IPPacket{}
	pkt.Version = data[0] >> 4
	if pkt.Version != 4 {
		return nil, fmt.Errorf("unsupported IP version: %d", pkt.Version)
	}

	headerLen := (data[0] & 0x0F) * 4 // 0x0F => 0b00001111, 4byte chunk
	pkt.HeaderLen = headerLen

	if len(data) < int(headerLen) {
		return nil, fmt.Errorf("packet smaller than header length: %d < %d", len(data), headerLen)
	}

	pkt.TOS = data[1]
	pkt.TotalLen = binary.BigEndian.Uint16(data[2:4])
	pkt.ID = binary.BigEndian.Uint16(data[4:6])

	flagsAndOffset := binary.BigEndian.Uint16(data[6:8])
	pkt.Flags = uint8(flagsAndOffset >> 13)
	pkt.FragOffset = flagsAndOffset & 0x1FFF

	pkt.TTL = data[8]
	pkt.Protocol = IPProtocol(data[9])
	pkt.Checksum = binary.BigEndian.Uint16(data[10:12])
	pkt.SrcIP = net.IP(data[12:16])
	pkt.DstIP = net.IP(data[16:20])

	if headerLen > 20 {
		pkt.Options = make([]byte, headerLen-20)
		copy(pkt.Options, data[20:headerLen])
	}

	if int(pkt.TotalLen) <= len(data) {
		pkt.Payload = data[headerLen:pkt.TotalLen]
	} else {
		pkt.Payload = data[headerLen:]
	}

	return pkt, nil
}

func ParseTCPPacket(payload []byte) (*TCPPacket, error) {
	if len(payload) < 20 {
		return nil, fmt.Errorf("payload too small for TCP header: %d bytes", len(payload))
	}

	tcp := &TCPPacket{}
	tcp.SrcPort = binary.BigEndian.Uint16(payload[0:2])
	tcp.DstPort = binary.BigEndian.Uint16(payload[2:4])
	tcp.SeqNum = binary.BigEndian.Uint32(payload[4:8])
	tcp.AckNum = binary.BigEndian.Uint32(payload[8:12])
	tcp.DataOffset = (payload[12] >> 4) * 4
	tcp.Flags = TCPFlags(payload[13] & 0x3F) // 0x3F => 0b00111111
	tcp.Window = binary.BigEndian.Uint16(payload[14:16])
	tcp.Checksum = binary.BigEndian.Uint16(payload[16:18])
	tcp.Urgent = binary.BigEndian.Uint16(payload[18:20])

	if tcp.DataOffset > 20 {
		if len(payload) < int(tcp.DataOffset) {
			return nil, fmt.Errorf("payload smaller than TCP data offset: %d < %d", len(payload), tcp.DataOffset)
		}
		tcp.Options = make([]byte, tcp.DataOffset-20)
		copy(tcp.Options, payload[20:tcp.DataOffset])
	}

	if len(payload) > int(tcp.DataOffset) {
		tcp.Payload = payload[tcp.DataOffset:]
	}

	return tcp, nil
}

func (p *IPPacket) Serialize() []byte {
	totalLen := int(p.HeaderLen) + len(p.Payload)
	data := make([]byte, totalLen)

	data[0] = (p.Version << 4) | (p.HeaderLen / 4)
	data[1] = p.TOS
	binary.BigEndian.PutUint16(data[2:4], uint16(totalLen))
	binary.BigEndian.PutUint16(data[4:6], p.ID)

	flagsAndOffset := (uint16(p.Flags) << 13) | (p.FragOffset & 0x1FFF)
	binary.BigEndian.PutUint16(data[6:8], flagsAndOffset)

	data[8] = p.TTL
	data[9] = uint8(p.Protocol)
	binary.BigEndian.PutUint16(data[10:12], p.Checksum)

	copy(data[12:16], p.SrcIP.To4())
	copy(data[16:20], p.DstIP.To4())

	if len(p.Options) > 0 {
		copy(data[20:], p.Options)
	}

	copy(data[p.HeaderLen:], p.Payload)
	return data
}

func (t *TCPPacket) Serialize() []byte {
	totalLen := int(t.DataOffset) + len(t.Payload)
	data := make([]byte, totalLen)

	binary.BigEndian.PutUint16(data[0:2], t.SrcPort)
	binary.BigEndian.PutUint16(data[2:4], t.DstPort)
	binary.BigEndian.PutUint32(data[4:8], t.SeqNum)
	binary.BigEndian.PutUint32(data[8:12], t.AckNum)

	data[12] = (t.DataOffset / 4) << 4
	data[13] = uint8(t.Flags)
	binary.BigEndian.PutUint16(data[14:16], t.Window)
	binary.BigEndian.PutUint16(data[16:18], t.Checksum)
	binary.BigEndian.PutUint16(data[18:20], t.Urgent)

	if len(t.Options) > 0 {
		copy(data[20:], t.Options)
	}

	copy(data[t.DataOffset:], t.Payload)
	return data
}

func (t *TCPPacket) HasFlag(flag TCPFlags) bool {
	return t.Flags&flag != 0
}
