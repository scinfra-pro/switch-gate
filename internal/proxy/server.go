package proxy

import (
	"context"
	"io"
	"log"
	"net"
	"sync"

	"github.com/scinfra-pro/switch-gate/internal/metrics"
	"github.com/scinfra-pro/switch-gate/internal/router"
)

// Server is a SOCKS5 proxy server
type Server struct {
	listener net.Listener
	router   *router.Router
	metrics  *metrics.Metrics

	conns   map[net.Conn]struct{}
	connsMu sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new SOCKS5 proxy server
func New(addr string, r *router.Router, m *metrics.Metrics) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		listener: listener,
		router:   r,
		metrics:  m,
		conns:    make(map[net.Conn]struct{}),
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Serve starts accepting connections
func (s *Server) Serve() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil
			default:
				log.Printf("ERROR: Accept failed: %v", err)
				continue
			}
		}

		s.trackConn(conn, true)
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(clientConn net.Conn) {
	defer s.trackConn(clientConn, false)
	defer func() { _ = clientConn.Close() }()

	// SOCKS5 handshake
	targetAddr, err := s.socks5Handshake(clientConn)
	if err != nil {
		log.Printf("DEBUG: SOCKS5 handshake failed: %v", err)
		return
	}

	// Dial target through router
	targetConn, err := s.router.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("DEBUG: Failed to dial %s: %v", targetAddr, err)
		s.socks5Reply(clientConn, 0x05) // Connection refused
		return
	}
	defer func() { _ = targetConn.Close() }()

	// Success reply
	s.socks5Reply(clientConn, 0x00)

	// Bidirectional relay
	s.relay(clientConn, targetConn)
}

func (s *Server) relay(client, target net.Conn) {
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

	// Wait for one direction to finish
	<-done
}

func (s *Server) trackConn(conn net.Conn, add bool) {
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
func (s *Server) Shutdown() {
	s.cancel()
	_ = s.listener.Close()

	// Close all active connections
	s.connsMu.Lock()
	for conn := range s.conns {
		_ = conn.Close()
	}
	s.connsMu.Unlock()
}

// ActiveConnections returns the number of active connections
func (s *Server) ActiveConnections() int {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	return len(s.conns)
}

// Addr returns the server address
func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}
