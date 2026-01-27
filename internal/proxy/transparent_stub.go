//go:build !linux

package proxy

import (
	"fmt"
	"net"

	"github.com/scinfra-pro/switch-gate/internal/metrics"
	"github.com/scinfra-pro/switch-gate/internal/router"
)

// TransparentServer is a stub for non-Linux platforms
type TransparentServer struct{}

// NewTransparent returns an error on non-Linux platforms
func NewTransparent(_ string, _ *router.Router, _ *metrics.Metrics) (*TransparentServer, error) {
	return nil, fmt.Errorf("transparent proxy is only supported on Linux")
}

// Serve returns an error on non-Linux platforms
func (s *TransparentServer) Serve() error {
	return fmt.Errorf("transparent proxy is only supported on Linux")
}

// Shutdown is a no-op on non-Linux platforms
func (s *TransparentServer) Shutdown() {}

// Addr returns nil on non-Linux platforms
func (s *TransparentServer) Addr() net.Addr {
	return nil
}

func getOriginalDst(_ net.Conn) (string, error) {
	return "", fmt.Errorf("SO_ORIGINAL_DST is only supported on Linux")
}
