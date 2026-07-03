package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"trade-kit-sidecar/broker"
	"trade-kit-sidecar/recipe"
)

// Config holds server configuration.
type Config struct {
	Port      int
	AuthToken string
	Version   string
}

// Server is the sidecar HTTP server.
type Server struct {
	config      Config
	registry    *broker.Registry
	broadcaster *Broadcaster
	runner      *recipe.Runner
	startTime   time.Time
	httpServer  *http.Server
	shutdown    chan struct{}
}

// New creates a new sidecar server.
func New(cfg Config) *Server {
	reg := broker.NewRegistry()
	bc := NewBroadcaster()
	return &Server{
		config:      cfg,
		registry:    reg,
		broadcaster: bc,
		runner:      recipe.NewRunner(reg, bc),
		startTime:   time.Now(),
		shutdown:    make(chan struct{}),
	}
}

// Registry returns the broker registry.
func (s *Server) Registry() *broker.Registry { return s.registry }

// Broadcaster returns the event broadcaster.
func (s *Server) Broadcaster() *Broadcaster { return s.broadcaster }

// StartTime returns when the server started.
func (s *Server) StartTime() time.Time { return s.startTime }

// Version returns the server version.
func (s *Server) Version() string { return s.config.Version }

// Shutdown signals the server to stop.
func (s *Server) Shutdown() { close(s.shutdown) }

// ListenAndServe starts the HTTP server and blocks until shutdown.
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// Wrap all routes with auth middleware.
	handler := AuthMiddleware(s.config.AuthToken, mux)

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.config.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful shutdown on SIGTERM/SIGINT.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		select {
		case sig := <-sigCh:
			log.Printf("[sidecar] received %s, shutting down", sig)
		case <-s.shutdown:
			log.Printf("[sidecar] shutdown requested")
		}
		s.gracefulShutdown()
	}()

	log.Printf("[sidecar] listening on :%d", s.config.Port)
	err := s.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) gracefulShutdown() {
	// Stop all running recipes first.
	s.runner.StopAll()

	// Disconnect all brokers.
	for _, b := range s.registry.List() {
		if b.Connected {
			adapter, _ := s.registry.Get(b.ID)
			if adapter != nil {
				adapter.Disconnect()
			}
		}
	}

	// Shutdown HTTP server with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.httpServer.Shutdown(ctx)
}
