package ops

// ClosePosition / SellLimit — close positions via eToro Trading API.
//
// eToro closes positions by position ID, not by symbol+quantity like Tiger.
// ClosePosition closes an existing open position.
//
// Demo: DELETE /api/v1/trading/demo/positions/{positionId}
// Real: DELETE /api/v1/trading/real/positions/{positionId}

import (
	"encoding/json"
	"fmt"
)

// CloseResult is returned when closing a position.
type CloseResult struct {
	PositionID string `json:"position_id"`
	Mode       string `json:"mode"`
	Symbol     string `json:"symbol"`
	Status     string `json:"status"`
}

// ClosePosition closes an open position by position ID.
func ClosePosition(c Caller, positionID string) (CloseResult, error) {
	if c.IsPaper() {
		return CloseResult{PositionID: positionID, Mode: "PAPER", Status: "CLOSED"}, nil
	}

	prefix := "/api/v1/trading/demo"
	if !c.IsPaper() {
		prefix = "/api/v1/trading/real"
	}

	path := fmt.Sprintf("%s/positions/%s", prefix, positionID)
	_, err := c.Delete(path, nil)
	if err != nil {
		return CloseResult{}, fmt.Errorf("close_position %s: %w", positionID, err)
	}

	return CloseResult{
		PositionID: positionID,
		Mode:       "LIVE",
		Status:     "CLOSED",
	}, nil
}

// SellBySymbol finds all open positions for a symbol and closes them.
// This provides a tiger-cli-compatible "sell AAPL 10" interface.
func SellBySymbol(c Caller, symbol string, amount float64) ([]CloseResult, error) {
	if c.IsPaper() {
		return []CloseResult{{
			PositionID: "PAPER",
			Mode:       "PAPER",
			Symbol:     symbol,
			Status:     "CLOSED",
		}}, nil
	}

	// Get all positions.
	positions, err := GetPositions(c)
	if err != nil {
		return nil, fmt.Errorf("sell %s: %w", symbol, err)
	}

	// Find matching positions.
	var results []CloseResult
	remaining := amount
	for _, p := range positions {
		if p.Symbol != symbol || !p.IsBuy {
			continue
		}
		if remaining <= 0 {
			break
		}

		res, err := ClosePosition(c, p.PositionID)
		if err != nil {
			return results, fmt.Errorf("sell %s: close position %s: %w", symbol, p.PositionID, err)
		}
		res.Symbol = symbol
		results = append(results, res)
		remaining -= p.Units
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("sell %s: no open buy positions found", symbol)
	}
	return results, nil
}

// ModifyPosition updates stop-loss and/or take-profit on an open position.
func ModifyPosition(c Caller, positionID string, stopLoss, takeProfit float64) error {
	if c.IsPaper() {
		return nil
	}

	prefix := "/api/v1/trading/demo"
	if !c.IsPaper() {
		prefix = "/api/v1/trading/real"
	}

	payload := map[string]interface{}{}
	if stopLoss > 0 {
		payload["StopLossRate"] = stopLoss
	}
	if takeProfit > 0 {
		payload["TakeProfitRate"] = takeProfit
	}

	path := fmt.Sprintf("%s/positions/%s", prefix, positionID)
	_, err := c.Patch(path, payload)
	if err != nil {
		return fmt.Errorf("modify_position %s: %w", positionID, err)
	}
	return nil
}

// CancelOrder cancels a pending order by order ID.
func CancelOrder(c Caller, orderID string) (CloseResult, error) {
	if c.IsPaper() {
		return CloseResult{PositionID: orderID, Mode: "PAPER", Status: "CANCELLED"}, nil
	}

	prefix := "/api/v1/trading/demo"
	if !c.IsPaper() {
		prefix = "/api/v1/trading/real"
	}

	path := fmt.Sprintf("%s/orders/%s", prefix, orderID)
	_, err := c.Delete(path, nil)
	if err != nil {
		return CloseResult{}, fmt.Errorf("cancel_order %s: %w", orderID, err)
	}

	return CloseResult{
		PositionID: orderID,
		Mode:       "LIVE",
		Status:     "CANCELLED",
	}, nil
}

// ModifyOrder modifies a pending order's stop-loss and/or take-profit.
func ModifyOrder(c Caller, orderID string, stopLoss, takeProfit, rate float64) error {
	if c.IsPaper() {
		return nil
	}

	prefix := "/api/v1/trading/demo"
	if !c.IsPaper() {
		prefix = "/api/v1/trading/real"
	}

	payload := map[string]interface{}{}
	if stopLoss > 0 {
		payload["StopLossRate"] = stopLoss
	}
	if takeProfit > 0 {
		payload["TakeProfitRate"] = takeProfit
	}
	if rate > 0 {
		payload["Rate"] = rate
	}

	path := fmt.Sprintf("%s/orders/%s", prefix, orderID)
	data, err := c.Patch(path, payload)
	if err != nil {
		return fmt.Errorf("modify_order %s: %w", orderID, err)
	}
	_ = data
	return nil
}

// etoroCloseResult is a helper for parsing the close response.
type etoroCloseResult struct {
	PositionID json.Number `json:"PositionID"`
	Status     string      `json:"Status"`
}
