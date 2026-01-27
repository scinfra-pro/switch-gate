package api

import (
	"context"
	"net/http"

	"github.com/scinfra-pro/switch-gate/internal/metrics"
	"github.com/scinfra-pro/switch-gate/internal/proxy"
	"github.com/scinfra-pro/switch-gate/internal/router"
)

// Server is the HTTP API server
type Server struct {
	router  *router.Router
	metrics *metrics.Metrics
	proxy   *proxy.Server
	mux     *http.ServeMux
	server  *http.Server
}

// New creates a new API server
func New(r *router.Router, m *metrics.Metrics, p *proxy.Server) *Server {
	s := &Server{
		router:  r,
		metrics: m,
		proxy:   p,
		mux:     http.NewServeMux(),
	}

	// Register routes
	s.mux.HandleFunc("GET /status", s.handleStatus)
	s.mux.HandleFunc("POST /mode/{mode}", s.handleSetMode)
	s.mux.HandleFunc("GET /metrics", s.handleMetrics)
	s.mux.HandleFunc("POST /limit/home", s.handleSetLimit)
	s.mux.HandleFunc("GET /health", s.handleHealth)

	return s
}

// ListenAndServe starts the API server
func (s *Server) ListenAndServe(addr string) error {
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}
	return s.server.ListenAndServe()
}

// Shutdown stops the API server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}
