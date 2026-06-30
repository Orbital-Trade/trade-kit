// Package handler implements HTTP handlers for the sidecar REST API.
package handler

import (
	"trade-kit-sidecar/broker"
)

// EventBroadcaster is the interface for broadcasting SSE events.
type EventBroadcaster interface {
	Broadcast(eventType string, data any)
}

// Handlers holds all HTTP handler methods.
type Handlers struct {
	registry    *broker.Registry
	broadcaster EventBroadcaster
	srv         interface {
		Version() string
		Shutdown()
	}
}

// New creates a new Handlers instance.
func New(registry *broker.Registry, broadcaster EventBroadcaster, srv interface {
	Version() string
	Shutdown()
}) *Handlers {
	return &Handlers{
		registry:    registry,
		broadcaster: broadcaster,
		srv:         srv,
	}
}
