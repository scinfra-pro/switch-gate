package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/scinfra-pro/switch-gate/internal/router"
)

// SetModeResponse represents the POST /mode/{mode} response
type SetModeResponse struct {
	Success   bool   `json:"success"`
	Requested string `json:"requested"`
	Mode      string `json:"mode"`
	Error     string `json:"error,omitempty"`
	// Keep "status" for backward compatibility
	Status string `json:"status,omitempty"`
}

// Error codes for SetModeResponse
const (
	ErrModeNotConfigured = "mode_not_configured"
	ErrModeInvalid       = "mode_invalid"
	ErrHomeLimitReached  = "home_limit_reached"
	ErrInternal          = "internal_error"
)

// StatusResponse represents the /status response
type StatusResponse struct {
	Mode        string       `json:"mode"`
	ModeHealthy *bool        `json:"mode_healthy,omitempty"` // only with ?check=true
	ModeError   *string      `json:"mode_error,omitempty"`   // only if mode_healthy=false
	Uptime      string       `json:"uptime"`
	Connections int          `json:"connections"`
	Traffic     TrafficStats `json:"traffic"`
	Home        HomeStats    `json:"home"`
	Available   []string     `json:"available_modes"`
}

// Error codes for mode health check
const (
	ErrWarpUnreachable    = "warp_unreachable"
	ErrWarpTimeout        = "warp_timeout"
	ErrWarpInterfaceDown  = "warp_interface_down"
	ErrHomeUnreachable    = "home_unreachable"
	ErrHomeTimeout        = "home_timeout"
	ErrCheckFailed        = "check_failed"
)

// TrafficStats contains traffic statistics per mode
type TrafficStats struct {
	DirectMB float64 `json:"direct_mb"`
	WarpMB   float64 `json:"warp_mb"`
	HomeMB   float64 `json:"home_mb"`
	TotalMB  float64 `json:"total_mb"`
}

// HomeStats contains home mode statistics
type HomeStats struct {
	LimitMB     int     `json:"limit_mb"`
	UsedMB      float64 `json:"used_mb"`
	RemainingMB float64 `json:"remaining_mb"`
	CostUSD     float64 `json:"cost_usd"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	stats := s.metrics.GetStats()

	directMB := float64(stats.Bytes["direct"]) / 1024 / 1024
	warpMB := float64(stats.Bytes["warp"]) / 1024 / 1024
	homeMB := float64(stats.Bytes["home"]) / 1024 / 1024
	limitMB := s.router.GetHomeLimit()

	available := make([]string, 0)
	for _, m := range s.router.AvailableModes() {
		available = append(available, m.String())
	}

	resp := StatusResponse{
		Mode:        s.router.GetMode().String(),
		Uptime:      stats.Uptime.Round(time.Second).String(),
		Connections: s.proxy.ActiveConnections(),
		Traffic: TrafficStats{
			DirectMB: roundTo2(directMB),
			WarpMB:   roundTo2(warpMB),
			HomeMB:   roundTo2(homeMB),
			TotalMB:  roundTo2(directMB + warpMB + homeMB),
		},
		Home: HomeStats{
			LimitMB:     limitMB,
			UsedMB:      roundTo2(homeMB),
			RemainingMB: roundTo2(float64(limitMB) - homeMB),
			CostUSD:     roundTo2(homeMB / 1024 * 3.50),
		},
		Available: available,
	}

	// Health check only if requested via ?check=true
	if r.URL.Query().Get("check") == "true" {
		healthy, err := s.router.TestCurrentMode()
		resp.ModeHealthy = &healthy
		if err != nil {
			errCode := classifyModeError(err, s.router.GetMode().String())
			resp.ModeError = &errCode
			log.Printf("API: Mode health check failed: %s", err.Error())
		}
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

func (s *Server) handleSetMode(w http.ResponseWriter, r *http.Request) {
	requested := r.PathValue("mode")

	if err := s.router.SetMode(router.Mode(requested)); err != nil {
		// Mode switch failed — return current mode and error
		currentMode := s.router.GetMode().String()
		errorCode := classifySetModeError(err)

		log.Printf("API: Mode switch to %s failed: %s", requested, err.Error())

		s.jsonResponse(w, http.StatusOK, SetModeResponse{
			Success:   false,
			Requested: requested,
			Mode:      currentMode,
			Error:     errorCode,
		})
		return
	}

	currentMode := s.router.GetMode().String()
	log.Printf("API: Mode switched to %s", currentMode)

	s.jsonResponse(w, http.StatusOK, SetModeResponse{
		Success:   true,
		Requested: requested,
		Mode:      currentMode,
		Status:    "ok", // backward compatibility
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	stats := s.metrics.GetStats()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	_, _ = fmt.Fprintf(w, "# HELP switch_gate_bytes_total Total bytes transferred\n")
	_, _ = fmt.Fprintf(w, "# TYPE switch_gate_bytes_total counter\n")
	for mode, bytes := range stats.Bytes {
		_, _ = fmt.Fprintf(w, "switch_gate_bytes_total{mode=\"%s\"} %d\n", mode, bytes)
	}

	_, _ = fmt.Fprintf(w, "# HELP switch_gate_connections_active Active connections\n")
	_, _ = fmt.Fprintf(w, "# TYPE switch_gate_connections_active gauge\n")
	_, _ = fmt.Fprintf(w, "switch_gate_connections_active %d\n", s.proxy.ActiveConnections())

	_, _ = fmt.Fprintf(w, "# HELP switch_gate_connections_total Total connections\n")
	_, _ = fmt.Fprintf(w, "# TYPE switch_gate_connections_total counter\n")
	_, _ = fmt.Fprintf(w, "switch_gate_connections_total %d\n", stats.TotalConns)

	_, _ = fmt.Fprintf(w, "# HELP switch_gate_uptime_seconds Uptime in seconds\n")
	_, _ = fmt.Fprintf(w, "# TYPE switch_gate_uptime_seconds gauge\n")
	_, _ = fmt.Fprintf(w, "switch_gate_uptime_seconds %.0f\n", stats.Uptime.Seconds())
}

func (s *Server) handleSetLimit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LimitMB int `json:"limit_mb"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	s.router.SetHomeLimit(req.LimitMB)

	log.Printf("API: Home proxy limit set to %d MB", req.LimitMB)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"limit_mb": req.LimitMB,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) jsonError(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}

// classifySetModeError converts an error to an error code
func classifySetModeError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()

	switch {
	case strings.Contains(msg, "invalid mode"):
		return ErrModeInvalid
	case strings.Contains(msg, "not available"):
		return ErrModeNotConfigured
	case strings.Contains(msg, "limit exhausted"):
		return ErrHomeLimitReached
	default:
		return ErrInternal
	}
}

// classifyModeError converts a mode health check error to an error code
func classifyModeError(err error, mode string) string {
	if err == nil {
		return ""
	}

	msg := strings.ToLower(err.Error())
	isTimeout := strings.Contains(msg, "timeout")
	isInterfaceError := strings.Contains(msg, "interface")

	switch mode {
	case "warp":
		if isInterfaceError {
			return ErrWarpInterfaceDown
		}
		if isTimeout {
			return ErrWarpTimeout
		}
		return ErrWarpUnreachable
	case "home":
		if isTimeout {
			return ErrHomeTimeout
		}
		return ErrHomeUnreachable
	default:
		return ErrCheckFailed
	}
}

func roundTo2(f float64) float64 {
	return float64(int(f*100)) / 100
}
