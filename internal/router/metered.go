package router

import (
	"net"

	"github.com/scinfra-pro/switch-gate/internal/metrics"
)

// MeteredConn wraps a connection to track bytes transferred
type MeteredConn struct {
	net.Conn
	mode    string
	metrics *metrics.Metrics
}

// NewMeteredConn creates a new metered connection
func NewMeteredConn(conn net.Conn, mode string, m *metrics.Metrics) *MeteredConn {
	return &MeteredConn{
		Conn:    conn,
		mode:    mode,
		metrics: m,
	}
}

// Read reads data and tracks bytes
func (m *MeteredConn) Read(b []byte) (int, error) {
	n, err := m.Conn.Read(b)
	if n > 0 {
		m.metrics.AddBytes(m.mode, int64(n))
	}
	return n, err
}

// Write writes data and tracks bytes
func (m *MeteredConn) Write(b []byte) (int, error) {
	n, err := m.Conn.Write(b)
	if n > 0 {
		m.metrics.AddBytes(m.mode, int64(n))
	}
	return n, err
}
