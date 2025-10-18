//go:build linux

package network

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

const (
	// SO_ORIGINAL_DST retrieves original destination before iptables REDIRECT
	SO_ORIGINAL_DST = 80

	IP6T_SO_ORIGINAL_DST = 80
)

func GetOriginalDest(conn net.Conn) (string, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return "", fmt.Errorf("connection is not TCP")
	}

	file, err := tcpConn.File()
	if err != nil {
		return "", fmt.Errorf("failed to get connection file: %w", err)
	}
	defer file.Close()

	fd := int(file.Fd())

	var addr syscall.RawSockaddrInet4
	size := uint32(unsafe.Sizeof(addr))

	_, _, errno := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		uintptr(fd),
		syscall.SOL_IP,
		SO_ORIGINAL_DST,
		uintptr(unsafe.Pointer(&addr)),
		uintptr(unsafe.Pointer(&size)),
		0,
	)

	if errno != 0 {
		return "", fmt.Errorf("getsockopt SO_ORIGINAL_DST failed: %v", errno)
	}

	ip := net.IPv4(addr.Addr[0], addr.Addr[1], addr.Addr[2], addr.Addr[3])
	// Convert network byte order to host byte order
	port := uint16(addr.Port>>8) | uint16(addr.Port&0xff)<<8

	return net.JoinHostPort(ip.String(), fmt.Sprintf("%d", port)), nil
}

func GetOriginalDestIPv6(conn net.Conn) (string, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return "", fmt.Errorf("connection is not TCP")
	}

	file, err := tcpConn.File()
	if err != nil {
		return "", fmt.Errorf("failed to get connection file: %w", err)
	}
	defer file.Close()

	fd := int(file.Fd())

	var addr syscall.RawSockaddrInet6
	size := uint32(unsafe.Sizeof(addr))

	_, _, errno := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		uintptr(fd),
		syscall.SOL_IPV6,
		IP6T_SO_ORIGINAL_DST,
		uintptr(unsafe.Pointer(&addr)),
		uintptr(unsafe.Pointer(&size)),
		0,
	)

	if errno != 0 {
		return "", fmt.Errorf("getsockopt IP6T_SO_ORIGINAL_DST failed: %v", errno)
	}

	ip := net.IP(addr.Addr[:])
	port := uint16(addr.Port>>8) | uint16(addr.Port&0xff)<<8

	return net.JoinHostPort(ip.String(), fmt.Sprintf("%d", port)), nil
}

func GetOriginalDestAuto(conn net.Conn) (string, error) {
	originalDest, err := GetOriginalDest(conn)
	if err == nil {
		return originalDest, nil
	}

	originalDest, err = GetOriginalDestIPv6(conn)
	if err == nil {
		return originalDest, nil
	}

	return "", fmt.Errorf("failed to get original destination for both IPv4 and IPv6")
}

func GetOriginalDestIP(conn net.Conn) (net.IP, error) {
	addrStr, err := GetOriginalDest(conn)
	if err != nil {
		return nil, err
	}

	host, _, err := net.SplitHostPort(addrStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %w", err)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", host)
	}

	return ip, nil
}

func GetOriginalDestPort(conn net.Conn) (uint16, error) {
	addrStr, err := GetOriginalDest(conn)
	if err != nil {
		return 0, err
	}

	_, portStr, err := net.SplitHostPort(addrStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse address: %w", err)
	}

	var port uint16
	_, err = fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return 0, fmt.Errorf("invalid port: %s", portStr)
	}

	return port, nil
}

func IsRedirectedConnection(conn net.Conn) bool {
	originalDest, err := GetOriginalDest(conn)
	if err != nil {
		return false
	}

	currentDest := conn.LocalAddr().String()

	return originalDest != currentDest
}
