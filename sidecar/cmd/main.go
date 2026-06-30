// trade-kit sidecar — HTTP server bridging the OrbitalTrade desktop app
// to Tiger, Moomoo, and eToro broker CLIs.
//
// Spawned by the Electron shell (SidecarManager) at app startup:
//
//	trade-kit serve --port 19090
//
// Environment:
//
//	ORBITAL_AUTH_TOKEN — Bearer token for all requests
//	ORBITAL_PORT       — override port (used by SidecarManager)
//
// Build:
//
//	cd sidecar && go build -o trade-kit ./cmd/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"trade-kit-sidecar/server"
)

var Version = "dev"

const usage = `trade-kit — OrbitalTrade sidecar server

Usage:
  trade-kit serve [--port PORT]

Commands:
  serve    Start the HTTP sidecar server

Flags:
  --port   Port to listen on (default: 19090, overridden by ORBITAL_PORT)

Environment:
  ORBITAL_AUTH_TOKEN   Bearer token for authentication (required)
  ORBITAL_PORT         Override port number
`

func main() {
	if len(os.Args) < 2 || os.Args[1] != "serve" {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 19090, "Port to listen on")
	fs.Parse(os.Args[2:])

	// ORBITAL_PORT env overrides --port flag.
	if envPort := os.Getenv("ORBITAL_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			*port = p
		}
	}

	authToken := os.Getenv("ORBITAL_AUTH_TOKEN")
	if authToken == "" {
		log.Fatal("[sidecar] ORBITAL_AUTH_TOKEN not set — refusing to start without auth")
	}

	srv := server.New(server.Config{
		Port:      *port,
		AuthToken: authToken,
		Version:   Version,
	})

	log.Printf("[sidecar] trade-kit %s starting on :%d", Version, *port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("[sidecar] %v", err)
	}
}
