package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/scinfra-pro/switch-gate/internal/api"
	"github.com/scinfra-pro/switch-gate/internal/config"
	"github.com/scinfra-pro/switch-gate/internal/metrics"
	"github.com/scinfra-pro/switch-gate/internal/proxy"
	"github.com/scinfra-pro/switch-gate/internal/router"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "/etc/switch-gate/config.yaml", "config file path")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		log.Printf("switch-gate %s", version)
		os.Exit(0)
	}

	log.Printf("switch-gate %s starting...", version)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize components
	met := metrics.New()

	rtr, err := router.New(cfg, met)
	if err != nil {
		log.Fatalf("Failed to create router: %v", err)
	}

	// SOCKS5 Proxy server
	proxyServer, err := proxy.New(cfg.Server.Listen, rtr, met)
	if err != nil {
		log.Fatalf("Failed to create proxy server: %v", err)
	}

	// Transparent proxy server (optional, for iptables REDIRECT, Linux only)
	var transparentServer *proxy.TransparentServer
	if cfg.Server.Transparent != "" {
		transparentServer, err = proxy.NewTransparent(cfg.Server.Transparent, rtr, met)
		if err != nil {
			log.Printf("WARN: Transparent proxy not available: %v", err)
			transparentServer = nil
		}
	}

	// API server
	apiServer := api.New(rtr, met, proxyServer)

	// Graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	g, gCtx := errgroup.WithContext(ctx)

	// SOCKS5 Proxy server
	g.Go(func() error {
		log.Printf("SOCKS5 proxy listening on %s", cfg.Server.Listen)
		return proxyServer.Serve()
	})

	// Transparent proxy server
	if transparentServer != nil {
		g.Go(func() error {
			log.Printf("Transparent proxy listening on %s", cfg.Server.Transparent)
			return transparentServer.Serve()
		})
	}

	// API server
	g.Go(func() error {
		log.Printf("API server listening on %s", cfg.Server.API)
		return apiServer.ListenAndServe(cfg.Server.API)
	})

	// Limit checker
	g.Go(func() error {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				rtr.CheckLimits()
			case <-gCtx.Done():
				return nil
			}
		}
	})

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(), 10*time.Second)
	defer shutdownCancel()

	proxyServer.Shutdown()
	if transparentServer != nil {
		transparentServer.Shutdown()
	}
	_ = apiServer.Shutdown(shutdownCtx)

	log.Println("Goodbye!")
}
