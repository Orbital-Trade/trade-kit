package ops

// BuyMarket / BuyLimit — place buy orders via eToro Trading API.
//
// eToro uses instrument IDs (not ticker symbols) for order placement.
// Symbol resolution happens automatically via the Asset Explorer API.
//
// Demo: POST /api/v1/trading/demo/orders
// Real: POST /api/v1/trading/real/orders

import (
	"encoding/json"
	"fmt"
)

// BuyMarket places a market buy order at current price.
func BuyMarket(c Caller, symbol string, amount float64) (OrderResult, error) {
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: symbol, Action: "BUY", Type: "MKT", Qty: int(amount)}, nil
	}

	inst, err := ResolveInstrument(c, symbol)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_market %s: %w", symbol, err)
	}

	prefix := "/api/v1/trading/demo"
	if !c.IsPaper() {
		prefix = "/api/v1/trading/real"
	}

	payload := map[string]interface{}{
		"InstrumentID": inst.ID,
		"IsBuy":        true,
		"Amount":       amount,
	}

	data, err := c.Post(prefix+"/orders", payload)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_market %s: %w", symbol, err)
	}

	return parseEtoroOrderResult(data, symbol, "BUY", "MKT", int(amount), 0, inst.ID)
}

// BuyLimit places a limit buy order at the specified price.
func BuyLimit(c Caller, symbol string, amount float64, limitPrice float64) (OrderResult, error) {
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: symbol, Action: "BUY", Type: "LMT", Qty: int(amount), Price: limitPrice}, nil
	}

	inst, err := ResolveInstrument(c, symbol)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_limit %s: %w", symbol, err)
	}

	prefix := "/api/v1/trading/demo"
	if !c.IsPaper() {
		prefix = "/api/v1/trading/real"
	}

	payload := map[string]interface{}{
		"InstrumentID": inst.ID,
		"IsBuy":        true,
		"Amount":       amount,
		"Rate":         limitPrice,
	}

	data, err := c.Post(prefix+"/orders", payload)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_limit %s: %w", symbol, err)
	}

	return parseEtoroOrderResult(data, symbol, "BUY", "LMT", int(amount), limitPrice, inst.ID)
}

// BuyWithStops places a buy order with optional stop-loss and take-profit.
func BuyWithStops(c Caller, symbol string, amount float64, limitPrice, stopLoss, takeProfit float64) (OrderResult, error) {
	if c.IsPaper() {
		tp := "MKT"
		if limitPrice > 0 {
			tp = "LMT"
		}
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: symbol, Action: "BUY", Type: tp, Qty: int(amount), Price: limitPrice}, nil
	}

	inst, err := ResolveInstrument(c, symbol)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy %s: %w", symbol, err)
	}

	prefix := "/api/v1/trading/demo"
	if !c.IsPaper() {
		prefix = "/api/v1/trading/real"
	}

	payload := map[string]interface{}{
		"InstrumentID": inst.ID,
		"IsBuy":        true,
		"Amount":       amount,
	}
	if limitPrice > 0 {
		payload["Rate"] = limitPrice
	}
	if stopLoss > 0 {
		payload["StopLossRate"] = stopLoss
	}
	if takeProfit > 0 {
		payload["TakeProfitRate"] = takeProfit
	}

	data, err := c.Post(prefix+"/orders", payload)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy %s: %w", symbol, err)
	}

	tp := "MKT"
	if limitPrice > 0 {
		tp = "LMT"
	}
	return parseEtoroOrderResult(data, symbol, "BUY", tp, int(amount), limitPrice, inst.ID)
}

func parseEtoroOrderResult(data json.RawMessage, symbol, action, orderType string, qty int, price float64, instrID int) (OrderResult, error) {
	var res struct {
		OrderID    json.Number `json:"OrderID"`
		PositionID json.Number `json:"PositionID"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return OrderResult{}, fmt.Errorf("place_order %s: parse response: %w", symbol, err)
	}

	id := res.OrderID.String()
	if id == "" || id == "0" {
		id = res.PositionID.String()
	}
	if id == "" || id == "0" {
		return OrderResult{}, fmt.Errorf("place_order %s: API returned no order/position ID — order likely rejected", symbol)
	}

	return OrderResult{
		OrderID: id,
		Mode:    "LIVE",
		Symbol:  symbol,
		Action:  action,
		Type:    orderType,
		Qty:     qty,
		Price:   price,
		InstrID: instrID,
	}, nil
}
