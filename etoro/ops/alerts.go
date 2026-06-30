package ops

// Price alert operations — CRUD for eToro price alerts.
//
// Endpoints:
//   GET    /api/v1/price-alerts     — list active alerts
//   POST   /api/v1/price-alerts     — create alert
//   PATCH  /api/v1/price-alerts/{id} — update alert
//   DELETE /api/v1/price-alerts/{id} — delete alert

import (
	"encoding/json"
	"fmt"
)

// PriceAlert represents an eToro price alert.
type PriceAlert struct {
	ID           string  `json:"id"`
	InstrumentID int     `json:"instrument_id"`
	Symbol       string  `json:"symbol"`
	Price        float64 `json:"price"`
	Direction    string  `json:"direction"` // "above" or "below"
	Active       bool    `json:"active"`
}

// GetAlerts returns all active price alerts.
func GetAlerts(c Caller) ([]PriceAlert, error) {
	data, err := c.Get("/api/v1/price-alerts", nil)
	if err != nil {
		return nil, fmt.Errorf("get_alerts: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []PriceAlert{}, nil
	}

	var alerts []PriceAlert
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, fmt.Errorf("get_alerts: parse: %w", err)
	}
	return alerts, nil
}

// CreateAlert creates a new price alert.
func CreateAlert(c Caller, symbol string, price float64, direction string) (PriceAlert, error) {
	inst, err := ResolveInstrument(c, symbol)
	if err != nil {
		return PriceAlert{}, fmt.Errorf("create_alert %s: %w", symbol, err)
	}

	data, err := c.Post("/api/v1/price-alerts", map[string]interface{}{
		"InstrumentID": inst.ID,
		"Price":        price,
		"Direction":    direction,
	})
	if err != nil {
		return PriceAlert{}, fmt.Errorf("create_alert %s: %w", symbol, err)
	}

	var alert PriceAlert
	if err := json.Unmarshal(data, &alert); err != nil {
		return PriceAlert{}, fmt.Errorf("create_alert %s: parse: %w", symbol, err)
	}
	alert.Symbol = symbol
	return alert, nil
}

// DeleteAlert removes a price alert by ID.
func DeleteAlert(c Caller, alertID string) error {
	_, err := c.Delete(fmt.Sprintf("/api/v1/price-alerts/%s", alertID), nil)
	if err != nil {
		return fmt.Errorf("delete_alert: %w", err)
	}
	return nil
}
