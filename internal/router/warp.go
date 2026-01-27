package router

import (
	"fmt"
	"net"
	"time"
)

// WarpDialer uses default routing which goes through tunnel (via policy routing)
// It does NOT bind to tunnel interface IP - that doesn't work with TUN devices
type WarpDialer struct {
	interfaceName string
	dialer        net.Dialer
}

// NewWarpDialer creates a dialer that routes through the tunnel interface
func NewWarpDialer(interfaceName string) (*WarpDialer, error) {
	// Verify the interface exists (tunnel is installed)
	_, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("interface %s not found: %w", interfaceName, err)
	}

	// Use default dialer WITHOUT LocalAddr - traffic will go through tunnel via policy routing
	return &WarpDialer{
		interfaceName: interfaceName,
		dialer: net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	}, nil
}

// Dial connects to the address through the tunnel
func (d *WarpDialer) Dial(network, address string) (net.Conn, error) {
	return d.dialer.Dial(network, address)
}

// Name returns the dialer name
func (d *WarpDialer) Name() string {
	return "warp"
}

// LocalIP returns nil as tunnel uses default routing
func (d *WarpDialer) LocalIP() net.IP {
	return nil
}
