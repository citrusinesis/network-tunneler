package network

import (
	"encoding/binary"
	"fmt"
	"net"

	pkgnet "network-tunneler/pkg/network"
)

// RewriteSourceIP modifies the source IP address in an IP packet
// and recalculates both IP and transport layer checksums.
// This is used by the Implant to make packets appear to originate from itself.
func RewriteSourceIP(ipPkt *pkgnet.IPPacket, newSrcIP net.IP) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	newSrcIP4 := newSrcIP.To4()
	if newSrcIP4 == nil {
		return fmt.Errorf("invalid IPv4 address: %s", newSrcIP.String())
	}

	// Update source IP
	ipPkt.SrcIP = newSrcIP4

	// Recalculate checksums (transport layer first, then IP)
	return ipPkt.RecalculateChecksums()
}

// RewriteDestinationIP modifies the destination IP address in an IP packet
// and recalculates both IP and transport layer checksums.
func RewriteDestinationIP(ipPkt *pkgnet.IPPacket, newDstIP net.IP) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	newDstIP4 := newDstIP.To4()
	if newDstIP4 == nil {
		return fmt.Errorf("invalid IPv4 address: %s", newDstIP.String())
	}

	// Update destination IP
	ipPkt.DstIP = newDstIP4

	// Recalculate checksums
	return ipPkt.RecalculateChecksums()
}

// RewriteSourcePort modifies the source port in a TCP or UDP packet
// and recalculates the transport layer checksum.
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

	// Modify source port in payload (first 2 bytes for both TCP and UDP)
	binary.BigEndian.PutUint16(ipPkt.Payload[0:2], newSrcPort)

	// Recalculate transport layer checksum
	switch ipPkt.Protocol {
	case pkgnet.ProtocolTCP:
		return ipPkt.RecalculateTCPChecksum()
	case pkgnet.ProtocolUDP:
		return ipPkt.RecalculateUDPChecksum()
	}

	return nil
}

// RewriteDestinationPort modifies the destination port in a TCP or UDP packet
// and recalculates the transport layer checksum.
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

	// Modify destination port in payload (bytes 2-3 for both TCP and UDP)
	binary.BigEndian.PutUint16(ipPkt.Payload[2:4], newDstPort)

	// Recalculate transport layer checksum
	switch ipPkt.Protocol {
	case pkgnet.ProtocolTCP:
		return ipPkt.RecalculateTCPChecksum()
	case pkgnet.ProtocolUDP:
		return ipPkt.RecalculateUDPChecksum()
	}

	return nil
}

// RewriteEndpoints modifies both source and destination IP:port pairs
// and recalculates all checksums. This is useful for NAT operations.
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

	// Validate IPs
	newSrcIP4 := newSrcIP.To4()
	newDstIP4 := newDstIP.To4()
	if newSrcIP4 == nil || newDstIP4 == nil {
		return fmt.Errorf("invalid IPv4 addresses")
	}

	// Update all fields
	ipPkt.SrcIP = newSrcIP4
	ipPkt.DstIP = newDstIP4
	binary.BigEndian.PutUint16(ipPkt.Payload[0:2], newSrcPort)
	binary.BigEndian.PutUint16(ipPkt.Payload[2:4], newDstPort)

	// Recalculate all checksums
	return ipPkt.RecalculateChecksums()
}

// RewriteForImplantForward rewrites a packet from Agent to appear as if
// it originated from the Implant when forwarding to the internal network.
//
// This is the key transformation that allows the Implant to forward traffic:
// - Original: AgentIP:AgentPort -> TargetIP:TargetPort
// - Rewritten: ImplantIP:ImplantPort -> TargetIP:TargetPort
//
// The target server will see traffic from Implant, not the original Agent.
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

	// Validate IP
	implantIP4 := implantIP.To4()
	if implantIP4 == nil {
		return fmt.Errorf("invalid IPv4 address: %s", implantIP.String())
	}

	// Rewrite source to Implant
	ipPkt.SrcIP = implantIP4
	binary.BigEndian.PutUint16(ipPkt.Payload[0:2], implantPort)
	// Destination remains unchanged (internal target)

	// Recalculate checksums
	return ipPkt.RecalculateChecksums()
}

// RewriteForImplantReturn rewrites a response packet from the internal network
// to be sent back through the tunnel.
//
// This reverses the transformation done by RewriteForImplantForward:
// - Received: TargetIP:TargetPort -> ImplantIP:ImplantPort
// - Rewritten: TargetIP:TargetPort -> AgentIP:AgentPort
//
// The Agent will receive the response as if it came directly from the target.
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

	// Validate IP
	agentIP4 := agentIP.To4()
	if agentIP4 == nil {
		return fmt.Errorf("invalid IPv4 address: %s", agentIP.String())
	}

	// Rewrite destination back to Agent
	ipPkt.DstIP = agentIP4
	binary.BigEndian.PutUint16(ipPkt.Payload[2:4], agentPort)
	// Source remains unchanged (internal target)

	// Recalculate checksums
	return ipPkt.RecalculateChecksums()
}

// IncrementTTL increments the TTL (Time To Live) field and recalculates IP checksum.
// Useful when forwarding packets to prevent them from expiring.
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

// DecrementTTL decrements the TTL (Time To Live) field and recalculates IP checksum.
// Returns error if TTL would reach 0 (packet should be dropped).
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

// SetTTL sets the TTL to a specific value and recalculates IP checksum.
func SetTTL(ipPkt *pkgnet.IPPacket, ttl uint8) error {
	if ipPkt == nil {
		return fmt.Errorf("nil IP packet")
	}

	ipPkt.TTL = ttl
	ipPkt.RecalculateIPChecksum()

	return nil
}
