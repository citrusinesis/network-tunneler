//go:build !linux

package network

import (
	"fmt"
	"net"
)

func GetOriginalDest(conn net.Conn) (string, error) {
	return "", fmt.Errorf("SO_ORIGINAL_DST is not supported on this platform (Linux only)")
}

func GetOriginalDestIPv6(conn net.Conn) (string, error) {
	return "", fmt.Errorf("IP6T_SO_ORIGINAL_DST is not supported on this platform (Linux only)")
}

func GetOriginalDestAuto(conn net.Conn) (string, error) {
	return "", fmt.Errorf("SO_ORIGINAL_DST is not supported on this platform (Linux only)")
}

func GetOriginalDestIP(conn net.Conn) (net.IP, error) {
	return nil, fmt.Errorf("SO_ORIGINAL_DST is not supported on this platform (Linux only)")
}

func GetOriginalDestPort(conn net.Conn) (uint16, error) {
	return 0, fmt.Errorf("SO_ORIGINAL_DST is not supported on this platform (Linux only)")
}

func IsRedirectedConnection(conn net.Conn) bool {
	return false
}
