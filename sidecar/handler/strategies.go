package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"trade-kit-sidecar/api"
)

// strategyDirs maps recipe IDs to their strategy source directories.
var strategyDirs = map[string]string{
	"daytrader": "daytrader/strategy",
	"bounce":    "bounce/strategy",
	"earnings":  "earnings/strategy",
	"index":     "index/strategy",
}

// GetStrategy returns strategy source files for a recipe.
// Useful for the copilot to understand what a strategy does.
//
//	GET /v1/strategies/{id}
func (h *Handlers) GetStrategy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rel, ok := strategyDirs[id]
	if !ok {
		api.WriteError(w, http.StatusNotFound, "unknown strategy: "+id)
		return
	}

	dir := filepath.Join(h.baseDir, rel)
	entries, err := os.ReadDir(dir)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "strategy dir not found: "+rel)
		return
	}

	files := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		files[entry.Name()] = string(data)
	}

	api.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"recipe": id,
		"path":   rel,
		"files":  files,
	})
}

// ListStrategies returns which strategies are available.
//
//	GET /v1/strategies
func (h *Handlers) ListStrategies(w http.ResponseWriter, r *http.Request) {
	out := make([]map[string]string, 0, len(strategyDirs))
	for id, dir := range strategyDirs {
		out = append(out, map[string]string{
			"id":   id,
			"path": dir,
		})
	}
	api.WriteJSON(w, http.StatusOK, out)
}
