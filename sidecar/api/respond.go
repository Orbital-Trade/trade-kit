// Package api provides shared HTTP response helpers.
package api

import (
	"encoding/json"
	"net/http"
)

// WriteJSON marshals v to JSON and writes it to w with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// WriteOK writes a JSON success response.
func WriteOK(w http.ResponseWriter) {
	WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
