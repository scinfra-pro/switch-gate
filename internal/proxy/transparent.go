//go:build linux

package proxy

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"syscall"
	"unsafe"

	"github.com/scinfra-pro/switch-gate/internal/metrics"
	"github.com/scinfra-pro/switch-gate/internal/router"
)

const (
	// SO_ORIGINAL_DST is the socket option to get original destination
	// from iptables REDIRECT (Linux only)
	SO_ORIGINAL_DST = 80
)

// TransparentServer handles connections redirected by iptables REDIRECT
type TransparentServer struct {
	listener net.Listener
	router   *router.Router
	metrics  *metrics.Metrics

	conns   map[net.Conn]struct{}
	connsMu sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
}

// NewTransparent creates a new transparent proxy server
func NewTransparent(addr string, r *router.Router, m *metrics.Metrics) (*TransparentServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TransparentServer{
		listener: listener,
		router:   r,
		metrics:  m,
		conns:    make(map[net.Conn]struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Serve starts accepting connections
func (s *TransparentServer) Serve() error {
	log.Printf("INFO: Transparent proxy listening on %s", s.listener.Addr())

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil
			default:
				log.Printf("ERROR: Transparent accept failed: %v", err)
				continue
			}
		}

		s.trackConn(conn, true)
		go s.handleConnection(conn)
	}
}

func (s *TransparentServer) handleConnection(clientConn net.Conn) {
	defer s.trackConn(clientConn, false)
	defer func() { _ = clientConn.Close() }()

	// Get original destination from SO_ORIGINAL_DST
	targetAddr, err := getOriginalDst(clientConn)
	if err != nil {
		log.Printf("ERROR: Failed to get original destination: %v", err)
		return
	}

	log.Printf("DEBUG: Transparent proxy: %s -> %s", clientConn.RemoteAddr(), targetAddr)

	// Dial target through router (uses current mode: direct/warp/home)
	targetConn, err := s.router.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("ERROR: Failed to dial %s: %v", targetAddr, err)
		return
	}
	defer func() { _ = targetConn.Close() }()

	// Bidirectional relay
	s.relay(clientConn, targetConn)
}

func (s *TransparentServer) relay(client, target net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(target, client)
		if tc, ok := target.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(client, target)
		if tc, ok := client.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
		done <- struct{}{}
	}()

	<-done
}

func (s *TransparentServer) trackConn(conn net.Conn, add bool) {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()

	if add {
		s.conns[conn] = struct{}{}
		s.metrics.ConnOpened()
	} else {
		delete(s.conns, conn)
		s.metrics.ConnClosed()
	}
}

// Shutdown stops the server
func (s *TransparentServer) Shutdown() {
	s.cancel()
	_ = s.listener.Close()

	s.connsMu.Lock()
	for conn := range s.conns {
		_ = conn.Close()
	}
	s.connsMu.Unlock()
}

// Addr returns the server address
func (s *TransparentServer) Addr() net.Addr {
	return s.listener.Addr()
}

// sockaddrIn is the raw sockaddr_in structure for IPv4
type sockaddrIn struct {
	Family uint16
	Port   uint16  // big-endian
	Addr   [4]byte // big-endian
	Zero   [8]byte
}

// getOriginalDst gets the original destination address from a connection
// that was redirected by iptables REDIRECT (Linux only)
func getOriginalDst(conn net.Conn) (string, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return "", fmt.Errorf("not a TCP connection")
	}

	file, err := tcpConn.File()
	if err != nil {
		return "", fmt.Errorf("failed to get file descriptor: %w", err)
	}
	defer func() { _ = file.Close() }()

	fd := int(file.Fd())

	// Get original destination using getsockopt SO_ORIGINAL_DST
	var addr sockaddrIn
	addrLen := uint32(unsafe.Sizeof(addr))

	_, _, errno := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT,
		uintptr(fd),
		uintptr(syscall.IPPROTO_IP),
		uintptr(SO_ORIGINAL_DST),
		uintptr(unsafe.Pointer(&addr)),
		uintptr(unsafe.Pointer(&addrLen)),
		0,
	)
	if errno != 0 {
		return "", fmt.Errorf("getsockopt SO_ORIGINAL_DST failed: %v", errno)
	}

	// Parse port (network byte order = big-endian)
	port := binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&addr.Port))[:])
	ip := net.IPv4(addr.Addr[0], addr.Addr[1], addr.Addr[2], addr.Addr[3])

	return fmt.Sprintf("%s:%d", ip.String(), port), nil
}
