package router

import "net"

// Dialer is the interface for different connection modes
type Dialer interface {
	Dial(network, address string) (net.Conn, error)
	Name() string
}
