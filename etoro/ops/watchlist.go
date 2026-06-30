package ops

// Watchlist operations — CRUD for eToro watchlists.
//
// Endpoints:
//   GET    /api/v1/watchlists          — list all watchlists
//   POST   /api/v1/watchlists          — create watchlist
//   DELETE /api/v1/watchlists/{id}     — delete watchlist
//   PUT    /api/v1/watchlists/{id}     — update watchlist (add/remove instruments)

import (
	"encoding/json"
	"fmt"
)

// Watchlist represents an eToro watchlist.
type Watchlist struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Instruments []int    `json:"instruments"` // instrument IDs
}

// GetWatchlists returns all user watchlists.
func GetWatchlists(c Caller) ([]Watchlist, error) {
	data, err := c.Get("/api/v1/watchlists", nil)
	if err != nil {
		return nil, fmt.Errorf("get_watchlists: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []Watchlist{}, nil
	}

	var lists []Watchlist
	if err := json.Unmarshal(data, &lists); err != nil {
		return nil, fmt.Errorf("get_watchlists: parse: %w", err)
	}
	return lists, nil
}

// CreateWatchlist creates a new watchlist.
func CreateWatchlist(c Caller, name string) (Watchlist, error) {
	data, err := c.Post("/api/v1/watchlists", map[string]interface{}{
		"Name": name,
	})
	if err != nil {
		return Watchlist{}, fmt.Errorf("create_watchlist: %w", err)
	}

	var wl Watchlist
	if err := json.Unmarshal(data, &wl); err != nil {
		return Watchlist{}, fmt.Errorf("create_watchlist: parse: %w", err)
	}
	return wl, nil
}

// DeleteWatchlist removes a watchlist by ID.
func DeleteWatchlist(c Caller, watchlistID string) error {
	_, err := c.Delete(fmt.Sprintf("/api/v1/watchlists/%s", watchlistID), nil)
	if err != nil {
		return fmt.Errorf("delete_watchlist: %w", err)
	}
	return nil
}

// AddToWatchlist adds instruments to a watchlist.
func AddToWatchlist(c Caller, watchlistID string, instrumentIDs []int) error {
	_, err := c.Put(fmt.Sprintf("/api/v1/watchlists/%s", watchlistID), map[string]interface{}{
		"InstrumentIDs": instrumentIDs,
	})
	if err != nil {
		return fmt.Errorf("add_to_watchlist: %w", err)
	}
	return nil
}
