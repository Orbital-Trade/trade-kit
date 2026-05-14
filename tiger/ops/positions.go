package ops

// GetPositions — MCP tool: tiger_positions
//
// Returns all open stock positions with cost basis and unrealized P&L.
// Calls Tiger REST method: positions (sec_type=STK).
//
// Tiger API quirks handled here:
//   - Response is double-encoded (client.Call unwraps the outer JSON string).
//   - After unwrapping, data is {"items":[...], "summary":{...}}.
//   - Field names are camelCase: position (qty), averageCost, latestPrice, etc.

import (
	"encoding/json"
	"fmt"
)

// Position represents a single open stock position.
type Position struct {
	Symbol        string  `json:"symbol"`
	Shares        int     `json:"shares"`
	AvgCost       float64 `json:"avg_cost"`
	MarketPrice   float64 `json:"market_price"`
	MarketValue   float64 `json:"market_value"`
	UnrealizedPnL float64 `json:"unrealized_pnl"`
	RealizedPnL   float64 `json:"realized_pnl"`
}

// tigerPosition maps the raw Tiger API field names (camelCase) to Go fields.
type tigerPosition struct {
	Symbol        string  `json:"symbol"`
	Position      int     `json:"position"`      // quantity / number of shares
	PositionQty   float64 `json:"positionQty"`   // float version; used if Position=0
	AverageCost   float64 `json:"averageCost"`
	LatestPrice   float64 `json:"latestPrice"`   // current market price
	MarketValue   float64 `json:"marketValue"`
	UnrealizedPnl float64 `json:"unrealizedPnl"`
	RealizedPnl   float64 `json:"realizedPnl"`
}

func (r tigerPosition) shares() int {
	if r.Position != 0 {
		return r.Position
	}
	return int(r.PositionQty)
}

// GetPositions returns all open stock positions for the account.
func GetPositions(c Caller) ([]Position, error) {
	data, err := c.Call("positions", map[string]interface{}{
		"account":  c.Account(),
		"sec_type": "STK",
		"lang":     "en_US",
	})
	if err != nil {
		return nil, fmt.Errorf("get_positions: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []Position{}, nil
	}

	raw, err := parsePositions(data)
	if err != nil {
		return nil, fmt.Errorf("get_positions: parse response: %w", err)
	}

	out := make([]Position, 0, len(raw))
	for _, r := range raw {
		if r.Symbol == "" {
			continue
		}
		out = append(out, Position{
			Symbol:        r.Symbol,
			Shares:        r.shares(),
			AvgCost:       r.AverageCost,
			MarketPrice:   r.LatestPrice,
			MarketValue:   r.MarketValue,
			UnrealizedPnL: r.UnrealizedPnl,
			RealizedPnL:   r.RealizedPnl,
		})
	}
	return out, nil
}

// parsePositions handles both response shapes Tiger may return:
//   - Direct array:              [{...}, {...}]
//   - Wrapped in items object:   {"items":[{...}], "summary":{...}}
func parsePositions(data json.RawMessage) ([]tigerPosition, error) {
	// Try direct array first.
	var direct []tigerPosition
	if err := json.Unmarshal(data, &direct); err == nil {
		return direct, nil
	}

	// Try object wrapper — Tiger wraps list responses in {"items":[...]}.
	var wrapper struct {
		Items []tigerPosition `json:"items"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("neither array nor {items:[]} format: %w", err)
	}
	return wrapper.Items, nil
}
