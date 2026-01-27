package router

import (
	"log"
	"os/exec"
	"strings"
)

// WarpControl manages tunnel service (enable/disable)
type WarpControl struct {
	enabled bool
}

// NewWarpControl creates a new tunnel controller
func NewWarpControl() *WarpControl {
	wc := &WarpControl{}
	wc.enabled = wc.isWarpRunning()
	log.Printf("INFO: Tunnel control initialized (currently %s)", wc.statusString())
	return wc
}

// isWarpRunning checks if tunnel service is active
func (wc *WarpControl) isWarpRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "warp-go")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "active"
}

// statusString returns human-readable status
func (wc *WarpControl) statusString() string {
	if wc.enabled {
		return "enabled"
	}
	return "disabled"
}

// Enable turns on tunnel (if not already on)
func (wc *WarpControl) Enable() error {
	if wc.enabled {
		log.Printf("DEBUG: Tunnel already enabled, skipping")
		return nil
	}

	log.Printf("INFO: Enabling tunnel...")
	cmd := exec.Command("warp-go", "o")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ERROR: Failed to enable tunnel: %v, output: %s", err, string(output))
		return err
	}

	wc.enabled = true
	log.Printf("INFO: Tunnel enabled successfully")
	return nil
}

// Disable turns off tunnel (if not already off)
func (wc *WarpControl) Disable() error {
	if !wc.enabled {
		log.Printf("DEBUG: Tunnel already disabled, skipping")
		return nil
	}

	log.Printf("INFO: Disabling tunnel...")
	cmd := exec.Command("warp-go", "o")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ERROR: Failed to disable tunnel: %v, output: %s", err, string(output))
		return err
	}

	wc.enabled = false
	log.Printf("INFO: Tunnel disabled successfully")
	return nil
}

// IsEnabled returns current tunnel state
func (wc *WarpControl) IsEnabled() bool {
	return wc.enabled
}
