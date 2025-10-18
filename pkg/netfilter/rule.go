package netfilter

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Table string

const (
	TableFilter Table = "filter"
	TableNat    Table = "nat"
	TableMangle Table = "mangle"
	TableRaw    Table = "raw"
)

type Chain string

const (
	ChainInput       Chain = "INPUT"
	ChainOutput      Chain = "OUTPUT"
	ChainForward     Chain = "FORWARD"
	ChainPrerouting  Chain = "PREROUTING"
	ChainPostrouting Chain = "POSTROUTING"
)

type Target string

const (
	TargetAccept     Target = "ACCEPT"
	TargetDrop       Target = "DROP"
	TargetReject     Target = "REJECT"
	TargetRedirect   Target = "REDIRECT"
	TargetMasquerade Target = "MASQUERADE"
	TargetReturn     Target = "RETURN"
)

type Protocol string

const (
	ProtocolTCP  Protocol = "tcp"
	ProtocolUDP  Protocol = "udp"
	ProtocolICMP Protocol = "icmp"
	ProtocolAll  Protocol = "all"
)

type Rule struct {
	Table       Table
	Chain       Chain
	Protocol    Protocol
	Source      string
	Destination string
	SrcPort     string
	DstPort     string
	Target      Target
	ToPort      string
	Comment     string
}

func (r *Rule) Args() []string {
	args := []string{"-t", string(r.Table)}

	if r.Protocol != "" {
		args = append(args, "-p", string(r.Protocol))
	}

	if r.Source != "" {
		args = append(args, "-s", r.Source)
	}

	if r.Destination != "" {
		args = append(args, "-d", r.Destination)
	}

	if r.SrcPort != "" {
		args = append(args, "--sport", r.SrcPort)
	}

	if r.DstPort != "" {
		args = append(args, "--dport", r.DstPort)
	}

	if r.Comment != "" {
		args = append(args, "-m", "comment", "--comment", r.Comment)
	}

	args = append(args, "-j", string(r.Target))

	if r.Target == TargetRedirect && r.ToPort != "" {
		args = append(args, "--to-ports", r.ToPort)
	}

	return args
}

func (r *Rule) String() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("table=%s", r.Table))
	parts = append(parts, fmt.Sprintf("chain=%s", r.Chain))

	if r.Protocol != "" {
		parts = append(parts, fmt.Sprintf("proto=%s", r.Protocol))
	}

	if r.Source != "" {
		parts = append(parts, fmt.Sprintf("src=%s", r.Source))
	}

	if r.Destination != "" {
		parts = append(parts, fmt.Sprintf("dst=%s", r.Destination))
	}

	if r.SrcPort != "" {
		parts = append(parts, fmt.Sprintf("sport=%s", r.SrcPort))
	}

	if r.DstPort != "" {
		parts = append(parts, fmt.Sprintf("dport=%s", r.DstPort))
	}

	parts = append(parts, fmt.Sprintf("target=%s", r.Target))

	if r.ToPort != "" {
		parts = append(parts, fmt.Sprintf("toport=%s", r.ToPort))
	}

	if r.Comment != "" {
		parts = append(parts, fmt.Sprintf("comment=%q", r.Comment))
	}

	return strings.Join(parts, " ")
}

func (r *Rule) Validate() error {
	if r.Table == "" {
		return fmt.Errorf("table is required")
	}

	if r.Chain == "" {
		return fmt.Errorf("chain is required")
	}

	if r.Target == "" {
		return fmt.Errorf("target is required")
	}

	if r.Target == TargetRedirect && r.ToPort == "" {
		return fmt.Errorf("REDIRECT target requires --to-ports")
	}

	if (r.SrcPort != "" || r.DstPort != "") && r.Protocol == "" {
		return fmt.Errorf("port specifications require protocol")
	}

	if r.SrcPort != "" {
		if err := validatePort(r.SrcPort); err != nil {
			return fmt.Errorf("invalid source port: %w", err)
		}
	}

	if r.DstPort != "" {
		if err := validatePort(r.DstPort); err != nil {
			return fmt.Errorf("invalid destination port: %w", err)
		}
	}

	if r.ToPort != "" {
		if err := validatePort(r.ToPort); err != nil {
			return fmt.Errorf("invalid redirect port: %w", err)
		}
	}

	if r.Source != "" {
		if err := validateCIDR(r.Source); err != nil {
			return fmt.Errorf("invalid source CIDR: %w", err)
		}
	}

	if r.Destination != "" {
		if err := validateCIDR(r.Destination); err != nil {
			return fmt.Errorf("invalid destination CIDR: %w", err)
		}
	}

	return nil
}

func validateCIDR(cidr string) error {
	if strings.Contains(cidr, "/") {
		_, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid CIDR notation: %w", err)
		}
		return nil
	}

	ip := net.ParseIP(cidr)
	if ip == nil {
		return fmt.Errorf("not a valid IP address or CIDR: %s", cidr)
	}

	return nil
}

func validatePort(port string) error {
	if port == "" {
		return nil
	}

	if strings.Contains(port, ":") {
		parts := strings.Split(port, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid port range format: %s", port)
		}
		if err := validateSinglePort(parts[0]); err != nil {
			return err
		}
		return validateSinglePort(parts[1])
	}

	return validateSinglePort(port)
}

func validateSinglePort(port string) error {
	p, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("port must be numeric: %s", port)
	}
	if p < 1 || p > 65535 {
		return fmt.Errorf("port must be between 1 and 65535: %d", p)
	}
	return nil
}
