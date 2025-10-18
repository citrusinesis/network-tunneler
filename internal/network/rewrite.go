package network

import (
	"encoding/binary"
	"fmt"
	"net"

	pkgnet "network-tunneler/pkg/network"
)

func RewriteSourceIP(ipPkt *pkgnet.IPPacket, newSrcIP net.IP) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	newSrcIP4 := newSrcIP.To4()
	if newSrcIP4 == nil {
		return fmt.Errorf("invalid IPv4 address: %s", newSrcIP.String())
	}
	ipPkt.SrcIP = newSrcIP4

	return ipPkt.RecalculateChecksums()
}

func RewriteDestinationIP(ipPkt *pkgnet.IPPacket, newDstIP net.IP) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	newDstIP4 := newDstIP.To4()
	if newDstIP4 == nil {
		return fmt.Errorf("invalid IPv4 address: %s", newDstIP.String())
	}
	ipPkt.DstIP = newDstIP4

	return ipPkt.RecalculateChecksums()
}

func RewriteSourcePort(ipPkt *pkgnet.IPPacket, newSrcPort uint16) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	if ipPkt.Protocol != pkgnet.ProtocolTCP && ipPkt.Protocol != pkgnet.ProtocolUDP {
		return fmt.Errorf("unsupported protocol: %d (only TCP/UDP)", ipPkt.Protocol)
	}

	if len(ipPkt.Payload) < 2 {
		return fmt.Errorf("payload too small to modify port")
	}

	binary.BigEndian.PutUint16(ipPkt.Payload[0:2], newSrcPort)

	switch ipPkt.Protocol {
	case pkgnet.ProtocolTCP:
		return ipPkt.RecalculateTCPChecksum()
	case pkgnet.ProtocolUDP:
		return ipPkt.RecalculateUDPChecksum()
	}

	return nil
}

func RewriteDestinationPort(ipPkt *pkgnet.IPPacket, newDstPort uint16) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	if ipPkt.Protocol != pkgnet.ProtocolTCP && ipPkt.Protocol != pkgnet.ProtocolUDP {
		return fmt.Errorf("unsupported protocol: %d (only TCP/UDP)", ipPkt.Protocol)
	}

	if len(ipPkt.Payload) < 4 {
		return fmt.Errorf("payload too small to modify port")
	}

	binary.BigEndian.PutUint16(ipPkt.Payload[2:4], newDstPort)

	switch ipPkt.Protocol {
	case pkgnet.ProtocolTCP:
		return ipPkt.RecalculateTCPChecksum()
	case pkgnet.ProtocolUDP:
		return ipPkt.RecalculateUDPChecksum()
	}

	return nil
}

func RewriteEndpoints(ipPkt *pkgnet.IPPacket, newSrcIP net.IP, newSrcPort uint16, newDstIP net.IP, newDstPort uint16) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	if ipPkt.Protocol != pkgnet.ProtocolTCP && ipPkt.Protocol != pkgnet.ProtocolUDP {
		return fmt.Errorf("unsupported protocol: %d (only TCP/UDP)", ipPkt.Protocol)
	}

	if len(ipPkt.Payload) < 4 {
		return fmt.Errorf("payload too small to modify ports")
	}

	newSrcIP4, newDstIP4 := newSrcIP.To4(), newDstIP.To4()
	if newSrcIP4 == nil || newDstIP4 == nil {
		return fmt.Errorf("invalid IPv4 addresses")
	}

	ipPkt.SrcIP, ipPkt.DstIP = newSrcIP4, newDstIP4
	binary.BigEndian.PutUint16(ipPkt.Payload[0:2], newSrcPort)
	binary.BigEndian.PutUint16(ipPkt.Payload[2:4], newDstPort)

	return ipPkt.RecalculateChecksums()
}

func RewriteForImplantForward(ipPkt *pkgnet.IPPacket, implantIP net.IP, implantPort uint16) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	if ipPkt.Protocol != pkgnet.ProtocolTCP && ipPkt.Protocol != pkgnet.ProtocolUDP {
		return fmt.Errorf("unsupported protocol: %d (only TCP/UDP)", ipPkt.Protocol)
	}

	if len(ipPkt.Payload) < 2 {
		return fmt.Errorf("payload too small to modify source port")
	}

	implantIP4 := implantIP.To4()
	if implantIP4 == nil {
		return fmt.Errorf("invalid IPv4 address: %s", implantIP.String())
	}

	ipPkt.SrcIP = implantIP4
	binary.BigEndian.PutUint16(ipPkt.Payload[0:2], implantPort)

	return ipPkt.RecalculateChecksums()
}

func RewriteForImplantReturn(ipPkt *pkgnet.IPPacket, agentIP net.IP, agentPort uint16) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	if ipPkt.Protocol != pkgnet.ProtocolTCP && ipPkt.Protocol != pkgnet.ProtocolUDP {
		return fmt.Errorf("unsupported protocol: %d (only TCP/UDP)", ipPkt.Protocol)
	}

	if len(ipPkt.Payload) < 4 {
		return fmt.Errorf("payload too small to modify destination port")
	}

	agentIP4 := agentIP.To4()
	if agentIP4 == nil {
		return fmt.Errorf("invalid IPv4 address: %s", agentIP.String())
	}

	ipPkt.DstIP = agentIP4
	binary.BigEndian.PutUint16(ipPkt.Payload[2:4], agentPort)

	return ipPkt.RecalculateChecksums()
}

func IncrementTTL(ipPkt *pkgnet.IPPacket) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	if ipPkt.TTL == 255 {
		return fmt.Errorf("TTL already at maximum")
	}

	ipPkt.TTL++
	ipPkt.RecalculateIPChecksum()

	return nil
}

func DecrementTTL(ipPkt *pkgnet.IPPacket) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	if ipPkt.TTL <= 1 {
		return fmt.Errorf("TTL expired")
	}

	ipPkt.TTL--
	ipPkt.RecalculateIPChecksum()

	return nil
}

func SetTTL(ipPkt *pkgnet.IPPacket, ttl uint8) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	ipPkt.TTL = ttl
	ipPkt.RecalculateIPChecksum()

	return nil
}
