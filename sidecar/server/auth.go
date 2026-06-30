package server

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"trade-kit-sidecar/api"
)

// AuthMiddleware returns an http.Handler that checks the Authorization header
// for a Bearer token matching the expected value. Returns 401 if missing/wrong.
func AuthMiddleware(token string, next http.Handler) http.Handler {
	tokenBytes := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Also accept token as query param (needed for SSE — EventSource can't set headers).
		got := ""
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			got = auth[7:]
		} else if q := r.URL.Query().Get("token"); q != "" {
			got = q
		}

		if got == "" {
			api.WriteError(w, http.StatusUnauthorized, "missing Authorization header")
			return
		}
		if subtle.ConstantTimeCompare(tokenBytes, []byte(got)) != 1 {
			api.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		next.ServeHTTP(w, r)
	})
}
