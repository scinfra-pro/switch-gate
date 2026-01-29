package router

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/scinfra-pro/switch-gate/internal/config"
	"github.com/scinfra-pro/switch-gate/internal/metrics"
)

const (
	// testDialTimeout is the timeout for testing mode connectivity
	testDialTimeout = 5 * time.Second
)

// WebhookSender is an interface for sending webhook events
type WebhookSender interface {
	Send(event string, payload map[string]interface{})
}

// Router manages traffic routing through different modes
type Router struct {
	mu      sync.RWMutex
	mode    Mode
	dialers map[Mode]Dialer
	metrics *metrics.Metrics

	// Tunnel control (enable/disable)
	warpControl *WarpControl

	// Upstream proxy limits
	homeLimitBytes uint64
	homeAutoSwitch Mode

	// Webhook for event notifications
	webhook       WebhookSender
	webhookEvents config.EventsConfig
}

// New creates a new router with configured dialers
func New(cfg *config.Config, m *metrics.Metrics, webhook WebhookSender) (*Router, error) {
	r := &Router{
		mode:           ModeDirect,
		dialers:        make(map[Mode]Dialer),
		metrics:        m,
		homeLimitBytes: uint64(cfg.Limits.Home.MaxMB) * 1024 * 1024,
		homeAutoSwitch: Mode(cfg.Limits.Home.AutoSwitchTo),
		webhook:        webhook,
		webhookEvents:  cfg.Webhooks.Events,
	}

	// Always available: direct (bound to local IP if configured)
	r.dialers[ModeDirect] = NewDirectDialer(cfg.Modes.Direct.LocalIP)
	if cfg.Modes.Direct.LocalIP != "" {
		log.Printf("INFO: Direct dialer bound to %s", cfg.Modes.Direct.LocalIP)
	}

	// Tunnel mode: optional, depends on interface availability
	if cfg.Modes.Warp.Interface != "" {
		warpDialer, err := NewWarpDialer(cfg.Modes.Warp.Interface)
		if err != nil {
			log.Printf("WARN: Tunnel dialer not available: %v", err)
		} else {
			r.dialers[ModeWarp] = warpDialer
			r.warpControl = NewWarpControl()
			log.Printf("INFO: Tunnel dialer initialized on %s", cfg.Modes.Warp.Interface)
		}
	}

	// Home (upstream) proxy: optional, depends on config
	// Uses Direct.LocalIP to bypass tunnel when connecting to proxy
	if cfg.Modes.Home.Host != "" {
		homeDialer, err := NewSocks5Dialer(
			cfg.Modes.Home.Host,
			cfg.Modes.Home.Port,
			cfg.Modes.Home.Username,
			cfg.Modes.Home.Password,
			cfg.Modes.Direct.LocalIP, // Bypass tunnel for proxy connection
		)
		if err != nil {
			log.Printf("WARN: Home proxy dialer not available: %v", err)
		} else {
			r.dialers[ModeHome] = homeDialer
			log.Printf("INFO: Home proxy dialer initialized (%s:%d via %s)", cfg.Modes.Home.Host, cfg.Modes.Home.Port, cfg.Modes.Direct.LocalIP)
		}
	}

	return r, nil
}

// SetMode changes the current routing mode
func (r *Router) SetMode(mode Mode) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !mode.IsValid() {
		return fmt.Errorf("invalid mode: %s", mode)
	}

	if _, ok := r.dialers[mode]; !ok {
		return fmt.Errorf("mode %s is not available", mode)
	}

	if mode == ModeHome && r.isHomeExhaustedLocked() {
		return fmt.Errorf("home proxy limit exhausted (%d MB used)",
			r.metrics.GetBytes("home")/1024/1024)
	}

	oldMode := r.mode
	r.mode = mode
	log.Printf("INFO: Mode switched to %s", mode)

	// Send webhook notification (if enabled)
	if r.webhook != nil && oldMode != mode && r.webhookEvents.ModeChanged {
		r.webhook.Send("mode.changed", map[string]interface{}{
			"from":    oldMode.String(),
			"to":      mode.String(),
			"trigger": "manual",
		})
	}

	return nil
}

// GetMode returns the current routing mode
func (r *Router) GetMode() Mode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mode
}

// Dial connects to the address using the current mode
func (r *Router) Dial(network, address string) (net.Conn, error) {
	r.mu.RLock()
	mode := r.mode
	dialer := r.dialers[mode]
	r.mu.RUnlock()

	conn, err := dialer.Dial(network, address)
	if err != nil {
		// Fallback to direct if tunnel fails
		if mode == ModeWarp {
			log.Printf("WARN: Tunnel dial failed, falling back to direct: %v", err)
			r.mu.RLock()
			dialer = r.dialers[ModeDirect]
			r.mu.RUnlock()
			conn, err = dialer.Dial(network, address)
			mode = ModeDirect
		}
		if err != nil {
			return nil, err
		}
	}

	return NewMeteredConn(conn, mode.String(), r.metrics), nil
}

// AvailableModes returns all available modes
func (r *Router) AvailableModes() []Mode {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modes := make([]Mode, 0, len(r.dialers))
	for mode := range r.dialers {
		modes = append(modes, mode)
	}
	return modes
}

// SetHomeLimit sets the home proxy traffic limit in MB
func (r *Router) SetHomeLimit(mb int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.homeLimitBytes = uint64(mb) * 1024 * 1024
}

// GetHomeLimit returns the home proxy traffic limit in MB
func (r *Router) GetHomeLimit() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int(r.homeLimitBytes / 1024 / 1024)
}

func (r *Router) isHomeExhaustedLocked() bool {
	if r.homeLimitBytes == 0 {
		return false
	}
	return r.metrics.GetBytes("home") >= r.homeLimitBytes
}

// IsHomeExhausted checks if home proxy limit is exhausted
func (r *Router) IsHomeExhausted() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isHomeExhaustedLocked()
}

// CheckLimits checks if any limits are exceeded and switches mode if needed
func (r *Router) CheckLimits() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.mode == ModeHome && r.isHomeExhaustedLocked() {
		oldMode := r.mode
		newMode := r.homeAutoSwitch
		if _, ok := r.dialers[newMode]; !ok {
			newMode = ModeDirect
		}

		log.Printf("WARN: Home proxy limit reached, switching to %s", newMode)
		r.mode = newMode

		// Send webhook notifications (if enabled)
		if r.webhook != nil {
			usedMB := r.metrics.GetBytes("home") / 1024 / 1024
			limitMB := r.homeLimitBytes / 1024 / 1024

			if r.webhookEvents.LimitReached {
				r.webhook.Send("limit.reached", map[string]interface{}{
					"mode":        "home",
					"used_mb":     usedMB,
					"limit_mb":    limitMB,
					"switched_to": newMode.String(),
				})
			}

			if r.webhookEvents.ModeChanged {
				r.webhook.Send("mode.changed", map[string]interface{}{
					"from":    oldMode.String(),
					"to":      newMode.String(),
					"trigger": "limit_reached",
				})
			}
		}
	}
}

// TestCurrentMode tests if the current mode is working by attempting a test connection.
// Returns (healthy, error). For direct mode, always returns (true, nil).
func (r *Router) TestCurrentMode() (bool, error) {
	r.mu.RLock()
	mode := r.mode
	dialer, ok := r.dialers[mode]
	r.mu.RUnlock()

	if !ok {
		return false, fmt.Errorf("%s dialer not available", mode)
	}

	// Direct mode is always considered healthy
	if mode == ModeDirect {
		return true, nil
	}

	// For WARP: first check if interface exists and is up
	if mode == ModeWarp {
		if warpDialer, ok := dialer.(*WarpDialer); ok {
			if err := checkInterfaceUp(warpDialer.InterfaceName()); err != nil {
				return false, err
			}
		}
	}

	// TCP test - verify actual connectivity
	endpoints := []string{"1.1.1.1:443", "8.8.8.8:443"}

	for _, ep := range endpoints {
		conn, err := dialWithTimeout(dialer, "tcp", ep, testDialTimeout)
		if err == nil {
			_ = conn.Close()
			return true, nil
		}
	}

	return false, fmt.Errorf("%s unreachable", mode)
}

// checkInterfaceUp verifies that a network interface exists and is up
func checkInterfaceUp(name string) error {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return fmt.Errorf("%s interface not found", name)
	}
	if iface.Flags&net.FlagUp == 0 {
		return fmt.Errorf("%s interface down", name)
	}
	return nil
}

// dialWithTimeout wraps dialer.Dial with a timeout
func dialWithTimeout(dialer Dialer, network, address string, timeout time.Duration) (net.Conn, error) {
	type dialResult struct {
		conn net.Conn
		err  error
	}

	result := make(chan dialResult, 1)
	go func() {
		conn, err := dialer.Dial(network, address)
		result <- dialResult{conn, err}
	}()

	select {
	case r := <-result:
		return r.conn, r.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout after %v", timeout)
	}
}
