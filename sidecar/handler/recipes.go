package handler

import (
	"net/http"

	"trade-kit-sidecar/api"
)

type recipeInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Available recipes (stub — will be wired to subprocess manager).
var availableRecipes = []recipeInfo{
	{ID: "daytrader", Name: "Gap-Up Day Trader", Status: "stopped"},
	{ID: "earnings", Name: "Earnings Play Scanner", Status: "stopped"},
	{ID: "bounce", Name: "RSI Bounce Scanner", Status: "stopped"},
	{ID: "index", Name: "QQQ/VIX Index Trader", Status: "stopped"},
}

// ListRecipes returns all available recipes and their status.
func (h *Handlers) ListRecipes(w http.ResponseWriter, r *http.Request) {
	api.WriteJSON(w, http.StatusOK, availableRecipes)
}

// StartRecipe starts a recipe by ID (stub).
func (h *Handlers) StartRecipe(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for i, rec := range availableRecipes {
		if rec.ID == id {
			availableRecipes[i].Status = "running"
			api.WriteOK(w)
			h.broadcaster.Broadcast("recipe_state", map[string]interface{}{
				"recipe_id": id,
				"status":    "running",
			})
			return
		}
	}
	api.WriteError(w, http.StatusNotFound, "unknown recipe: "+id)
}

// StopRecipe stops a recipe by ID (stub).
func (h *Handlers) StopRecipe(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for i, rec := range availableRecipes {
		if rec.ID == id {
			availableRecipes[i].Status = "stopped"
			api.WriteOK(w)
			h.broadcaster.Broadcast("recipe_state", map[string]interface{}{
				"recipe_id": id,
				"status":    "stopped",
			})
			return
		}
	}
	api.WriteError(w, http.StatusNotFound, "unknown recipe: "+id)
}
