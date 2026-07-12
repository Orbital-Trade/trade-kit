// Package handler implements HTTP handlers for the sidecar REST API.
package handler

import (
	"os"
	"path/filepath"

	"trade-kit-sidecar/broker"
	"trade-kit-sidecar/recipe"
)

// EventBroadcaster is the interface for broadcasting SSE events.
type EventBroadcaster interface {
	Broadcast(eventType string, data any)
}

// Handlers holds all HTTP handler methods.
type Handlers struct {
	registry    *broker.Registry
	broadcaster EventBroadcaster
	runner      *recipe.Runner
	baseDir     string // trade-kit root directory
	srv         interface {
		Version() string
		Shutdown()
	}
}

// New creates a new Handlers instance.
func New(registry *broker.Registry, broadcaster EventBroadcaster, runner *recipe.Runner, srv interface {
	Version() string
	Shutdown()
}) *Handlers {
	baseDir, _ := os.Getwd()
	// Try to find trade-kit root (look for Makefile).
	if _, err := os.Stat(filepath.Join(baseDir, "Makefile")); err != nil {
		parent := filepath.Dir(baseDir)
		if _, err := os.Stat(filepath.Join(parent, "Makefile")); err == nil {
			baseDir = parent
		}
	}
	return &Handlers{
		registry:    registry,
		broadcaster: broadcaster,
		runner:      runner,
		baseDir:     baseDir,
		srv:         srv,
	}
}
