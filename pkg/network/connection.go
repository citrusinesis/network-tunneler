package network

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
)

type ConnectionTuple struct {
	SrcIP   net.IP
	SrcPort uint16
	DstIP   net.IP
	DstPort uint16
}

func NewConnectionTuple(srcIP net.IP, srcPort uint16, dstIP net.IP, dstPort uint16) *ConnectionTuple {
	return &ConnectionTuple{
		SrcIP:   srcIP,
		SrcPort: srcPort,
		DstIP:   dstIP,
		DstPort: dstPort,
	}
}

func NewConnectionTupleFromPacket(ipPkt *IPPacket) (*ConnectionTuple, error) {
	if ipPkt.Protocol != ProtocolTCP && ipPkt.Protocol != ProtocolUDP {
		return nil, fmt.Errorf("unsupported protocol: %d", ipPkt.Protocol)
	}

	if len(ipPkt.Payload) < 4 {
		return nil, fmt.Errorf("payload too small to extract ports")
	}

	srcPort := binary.BigEndian.Uint16(ipPkt.Payload[0:2])
	dstPort := binary.BigEndian.Uint16(ipPkt.Payload[2:4])

	return &ConnectionTuple{
		SrcIP:   ipPkt.SrcIP,
		SrcPort: srcPort,
		DstIP:   ipPkt.DstIP,
		DstPort: dstPort,
	}, nil
}

func (ft *ConnectionTuple) String() string {
	return fmt.Sprintf("%s:%d -> %s:%d",
		ft.SrcIP.String(), ft.SrcPort,
		ft.DstIP.String(), ft.DstPort)
}

func (ft *ConnectionTuple) Reverse() *ConnectionTuple {
	return &ConnectionTuple{
		SrcIP:   ft.DstIP,
		SrcPort: ft.DstPort,
		DstIP:   ft.SrcIP,
		DstPort: ft.SrcPort,
	}
}

func GenerateConnectionID(srcIP net.IP, srcPort uint16, dstIP net.IP, dstPort uint16) string {
	srcIP4, dstIP4 := srcIP.To4(), dstIP.To4()
	if srcIP4 == nil || dstIP4 == nil {
		srcIP4 = srcIP
		dstIP4 = dstIP
	}

	ep1, ep2 := endpoint{ip: srcIP4, port: srcPort}, endpoint{ip: dstIP4, port: dstPort}
	if cmp := compareIPs(ep1.ip, ep2.ip); cmp > 0 || (cmp == 0 && ep1.port > ep2.port) {
		ep1, ep2 = ep2, ep1
	}

	hasher := sha256.New()
	portBytes := make([]byte, 2)

	hasher.Write(ep1.ip)
	binary.BigEndian.PutUint16(portBytes, ep1.port)
	hasher.Write(portBytes)

	hasher.Write(ep2.ip)
	binary.BigEndian.PutUint16(portBytes, ep2.port)
	hasher.Write(portBytes)

	hash := hasher.Sum(nil)
	return fmt.Sprintf("%x", hash[:16])
}

func GenerateConnectionIDFromTuple(ft *ConnectionTuple) string {
	return GenerateConnectionID(ft.SrcIP, ft.SrcPort, ft.DstIP, ft.DstPort)
}

func GenerateConnectionIDFromPacket(ipPkt *IPPacket) (string, error) {
	ft, err := NewConnectionTupleFromPacket(ipPkt)
	if err != nil {
		return "", err
	}
	return GenerateConnectionIDFromTuple(ft), nil
}

type endpoint struct {
	ip   net.IP
	port uint16
}

func compareIPs(a, b net.IP) int {
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}

	for i := range len(a) {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}

	return 0
}
