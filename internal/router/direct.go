package router

import (
	"net"
	"time"
)

// DirectDialer connects directly using server's real IP (bypassing tunnel)
type DirectDialer struct {
	localIP net.IP
	dialer  net.Dialer
}

// NewDirectDialer creates a dialer bound to specific local IP
// If localIP is empty, uses default routing
func NewDirectDialer(localIP string) *DirectDialer {
	d := &DirectDialer{
		dialer: net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	}

	if localIP != "" {
		ip := net.ParseIP(localIP)
		if ip != nil {
			d.localIP = ip
			d.dialer.LocalAddr = &net.TCPAddr{IP: ip}
		}
	}

	return d
}

// Dial connects to the address using direct routing
func (d *DirectDialer) Dial(network, address string) (net.Conn, error) {
	return d.dialer.Dial(network, address)
}

// Name returns the dialer name
func (d *DirectDialer) Name() string {
	return "direct"
}

// LocalIP returns the bound local IP
func (d *DirectDialer) LocalIP() net.IP {
	return d.localIP
}
