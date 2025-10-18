package network

import "network-tunneler/pkg/netfilter"

func RedirectToLocalPort(destinationCIDR, toPort string) *netfilter.Rule {
	return netfilter.NewRule().
		Table(netfilter.TableNat).
		Chain(netfilter.ChainOutput).
		Destination(destinationCIDR).
		Target(netfilter.TargetRedirect).
		ToPort(toPort).
		Comment("network-tunneler redirect").
		MustBuild()
}

func RedirectTCPToLocalPort(destinationCIDR, toPort string) *netfilter.Rule {
	return netfilter.NewRule().
		Table(netfilter.TableNat).
		Chain(netfilter.ChainOutput).
		Protocol(netfilter.ProtocolTCP).
		Destination(destinationCIDR).
		Target(netfilter.TargetRedirect).
		ToPort(toPort).
		Comment("network-tunneler TCP redirect").
		MustBuild()
}

func RedirectUDPToLocalPort(destinationCIDR, toPort string) *netfilter.Rule {
	return netfilter.NewRule().
		Table(netfilter.TableNat).
		Chain(netfilter.ChainOutput).
		Protocol(netfilter.ProtocolUDP).
		Destination(destinationCIDR).
		Target(netfilter.TargetRedirect).
		ToPort(toPort).
		Comment("network-tunneler UDP redirect").
		MustBuild()
}
