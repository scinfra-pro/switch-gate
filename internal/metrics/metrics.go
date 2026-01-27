package metrics

import (
	"sync/atomic"
	"time"
)

// Metrics tracks traffic and connection statistics
type Metrics struct {
	startTime time.Time

	// Bytes per mode
	bytesDirect atomic.Uint64
	bytesWarp   atomic.Uint64
	bytesHome   atomic.Uint64

	// Connections
	activeConns atomic.Int32
	totalConns  atomic.Uint64
}

// New creates a new Metrics instance
func New() *Metrics {
	return &Metrics{
		startTime: time.Now(),
	}
}

// AddBytes adds bytes to the specified mode counter
func (m *Metrics) AddBytes(mode string, n int64) {
	if n <= 0 {
		return
	}

	switch mode {
	case "direct":
		m.bytesDirect.Add(uint64(n))
	case "warp":
		m.bytesWarp.Add(uint64(n))
	case "home":
		m.bytesHome.Add(uint64(n))
	}
}

// GetBytes returns bytes for the specified mode
func (m *Metrics) GetBytes(mode string) uint64 {
	switch mode {
	case "direct":
		return m.bytesDirect.Load()
	case "warp":
		return m.bytesWarp.Load()
	case "home":
		return m.bytesHome.Load()
	default:
		return 0
	}
}

// GetAllBytes returns bytes for all modes
func (m *Metrics) GetAllBytes() map[string]uint64 {
	return map[string]uint64{
		"direct": m.bytesDirect.Load(),
		"warp":   m.bytesWarp.Load(),
		"home":   m.bytesHome.Load(),
	}
}

// Uptime returns the time since start
func (m *Metrics) Uptime() time.Duration {
	return time.Since(m.startTime)
}

// ConnOpened increments connection counters
func (m *Metrics) ConnOpened() {
	m.activeConns.Add(1)
	m.totalConns.Add(1)
}

// ConnClosed decrements active connection counter
func (m *Metrics) ConnClosed() {
	m.activeConns.Add(-1)
}

// ActiveConnections returns the number of active connections
func (m *Metrics) ActiveConnections() int {
	return int(m.activeConns.Load())
}

// TotalConnections returns the total number of connections
func (m *Metrics) TotalConnections() uint64 {
	return m.totalConns.Load()
}

// Stats contains all metrics
type Stats struct {
	Bytes       map[string]uint64
	ActiveConns int
	TotalConns  uint64
	Uptime      time.Duration
}

// GetStats returns all metrics as a Stats struct
func (m *Metrics) GetStats() Stats {
	return Stats{
		Bytes:       m.GetAllBytes(),
		ActiveConns: m.ActiveConnections(),
		TotalConns:  m.TotalConnections(),
		Uptime:      m.Uptime(),
	}
}
