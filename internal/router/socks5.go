package router

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/proxy"
)

// Socks5Dialer connects through a SOCKS5 upstream proxy
type Socks5Dialer struct {
	proxyAddr string
	auth      *proxy.Auth
	dialer    proxy.Dialer
}

// localIPDialer wraps net.Dialer to bind to specific local IP
// This ensures connection to proxy goes via server IP, not through tunnel
type localIPDialer struct {
	dialer net.Dialer
}

func (d *localIPDialer) Dial(network, addr string) (net.Conn, error) {
	return d.dialer.Dial(network, addr)
}

// NewSocks5Dialer creates a dialer that routes through a SOCKS5 proxy
func NewSocks5Dialer(host string, port int, username, password string, localIP string) (*Socks5Dialer, error) {
	proxyAddr := fmt.Sprintf("%s:%d", host, port)

	var auth *proxy.Auth
	if username != "" {
		auth = &proxy.Auth{
			User:     username,
			Password: password,
		}
	}

	// Use custom dialer bound to server IP to bypass tunnel routing
	var forward proxy.Dialer = proxy.Direct
	if localIP != "" {
		ip := net.ParseIP(localIP)
		if ip != nil {
			forward = &localIPDialer{
				dialer: net.Dialer{
					LocalAddr: &net.TCPAddr{IP: ip},
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				},
			}
		}
	}

	dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, forward)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
	}

	return &Socks5Dialer{
		proxyAddr: proxyAddr,
		auth:      auth,
		dialer:    dialer,
	}, nil
}

// Dial connects to the address through the SOCKS5 proxy
func (d *Socks5Dialer) Dial(network, address string) (net.Conn, error) {
	return d.dialer.Dial(network, address)
}

// Name returns the dialer name
func (d *Socks5Dialer) Name() string {
	return "home"
}
