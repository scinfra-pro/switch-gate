package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/scinfra-pro/switch-gate/internal/router"
)

// StatusResponse represents the /status response
type StatusResponse struct {
	Mode        string       `json:"mode"`
	Uptime      string       `json:"uptime"`
	Connections int          `json:"connections"`
	Traffic     TrafficStats `json:"traffic"`
	Home        HomeStats    `json:"home"`
	Available   []string     `json:"available_modes"`
}

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

	s.jsonResponse(w, http.StatusOK, resp)
}

func (s *Server) handleSetMode(w http.ResponseWriter, r *http.Request) {
	mode := r.PathValue("mode")

	if err := s.router.SetMode(router.Mode(mode)); err != nil {
		s.jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Printf("API: Mode switched to %s", mode)

	s.jsonResponse(w, http.StatusOK, map[string]string{
		"status": "ok",
		"mode":   mode,
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	stats := s.metrics.GetStats()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	fmt.Fprintf(w, "# HELP switch_gate_bytes_total Total bytes transferred\n")
	fmt.Fprintf(w, "# TYPE switch_gate_bytes_total counter\n")
	for mode, bytes := range stats.Bytes {
		fmt.Fprintf(w, "switch_gate_bytes_total{mode=\"%s\"} %d\n", mode, bytes)
	}

	fmt.Fprintf(w, "# HELP switch_gate_connections_active Active connections\n")
	fmt.Fprintf(w, "# TYPE switch_gate_connections_active gauge\n")
	fmt.Fprintf(w, "switch_gate_connections_active %d\n", s.proxy.ActiveConnections())

	fmt.Fprintf(w, "# HELP switch_gate_connections_total Total connections\n")
	fmt.Fprintf(w, "# TYPE switch_gate_connections_total counter\n")
	fmt.Fprintf(w, "switch_gate_connections_total %d\n", stats.TotalConns)

	fmt.Fprintf(w, "# HELP switch_gate_uptime_seconds Uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE switch_gate_uptime_seconds gauge\n")
	fmt.Fprintf(w, "switch_gate_uptime_seconds %.0f\n", stats.Uptime.Seconds())
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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) jsonError(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}

func roundTo2(f float64) float64 {
	return float64(int(f*100)) / 100
}
