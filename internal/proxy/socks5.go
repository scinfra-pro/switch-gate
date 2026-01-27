package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	socks5Version = 0x05
	cmdConnect    = 0x01
	atypIPv4      = 0x01
	atypDomain    = 0x03
	atypIPv6      = 0x04
)

// socks5Handshake performs SOCKS5 handshake and returns target address
func (s *Server) socks5Handshake(conn net.Conn) (string, error) {
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	defer conn.SetDeadline(time.Time{})

	// Read: VER | NMETHODS | METHODS
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", fmt.Errorf("read version: %w", err)
	}

	if buf[0] != socks5Version {
		return "", fmt.Errorf("unsupported SOCKS version: %d", buf[0])
	}

	nMethods := int(buf[1])
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return "", fmt.Errorf("read methods: %w", err)
	}

	// Reply: VER | METHOD (no auth required)
	if _, err := conn.Write([]byte{socks5Version, 0x00}); err != nil {
		return "", fmt.Errorf("write method: %w", err)
	}

	// Read: VER | CMD | RSV | ATYP | DST.ADDR | DST.PORT
	buf = make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", fmt.Errorf("read request: %w", err)
	}

	if buf[1] != cmdConnect {
		s.socks5Reply(conn, 0x07) // Command not supported
		return "", fmt.Errorf("unsupported command: %d", buf[1])
	}

	var host string
	switch buf[3] {
	case atypIPv4:
		ip := make([]byte, 4)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", fmt.Errorf("read IPv4: %w", err)
		}
		host = net.IP(ip).String()

	case atypDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return "", fmt.Errorf("read domain length: %w", err)
		}
		domain := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", fmt.Errorf("read domain: %w", err)
		}
		host = string(domain)

	case atypIPv6:
		ip := make([]byte, 16)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", fmt.Errorf("read IPv6: %w", err)
		}
		host = net.IP(ip).String()

	default:
		s.socks5Reply(conn, 0x08) // Address type not supported
		return "", fmt.Errorf("unsupported address type: %d", buf[3])
	}

	// Read port
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return "", fmt.Errorf("read port: %w", err)
	}
	port := binary.BigEndian.Uint16(portBuf)

	return fmt.Sprintf("%s:%d", host, port), nil
}

// socks5Reply sends SOCKS5 reply
func (s *Server) socks5Reply(conn net.Conn, status byte) {
	// VER | REP | RSV | ATYP | BND.ADDR | BND.PORT
	reply := []byte{socks5Version, status, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0}
	conn.Write(reply)
}
