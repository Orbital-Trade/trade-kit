// Package db manages the trade journal SQLite database.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS trades (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	symbol      TEXT    NOT NULL,
	side        TEXT    NOT NULL CHECK(side IN ('BUY','SELL')),
	qty         INTEGER NOT NULL CHECK(qty > 0),
	price       REAL    NOT NULL CHECK(price > 0),
	filled_at   TEXT    NOT NULL,
	broker      TEXT    NOT NULL DEFAULT '',
	order_id    TEXT    NOT NULL DEFAULT '',
	strategy    TEXT    NOT NULL DEFAULT '',
	note        TEXT    NOT NULL DEFAULT '',
	created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS trades_broker_order
	ON trades(broker, order_id) WHERE order_id != '';
`

// Trade represents a single filled order recorded in the journal.
type Trade struct {
	ID         int64
	Symbol     string
	Side       string
	Qty        int
	Price      float64
	FilledAt   time.Time
	Broker     string
	OrderID    string
	Strategy   string
	Note       string
	CreatedAt  time.Time
}

// DB wraps the SQLite connection.
type DB struct {
	conn *sql.DB
}

// DefaultPath returns ~/.trade-kit/journal/trades.db
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "trades.db"
	}
	return filepath.Join(home, ".trade-kit", "journal", "trades.db")
}

// Open opens (or creates) the journal database at the given path.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (d *DB) Close() error { return d.conn.Close() }

// Add inserts a trade. Duplicate (broker, order_id) pairs are silently ignored.
func (d *DB) Add(t Trade) (int64, error) {
	res, err := d.conn.Exec(
		`INSERT OR IGNORE INTO trades (symbol, side, qty, price, filled_at, broker, order_id, strategy, note)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.Symbol, t.Side, t.Qty, t.Price,
		t.FilledAt.UTC().Format(time.RFC3339),
		t.Broker, t.OrderID, t.Strategy, t.Note,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// List returns trades filtered by symbol and/or recency. Pass empty symbol for all.
func (d *DB) List(symbol string, days int) ([]Trade, error) {
	query := `SELECT id, symbol, side, qty, price, filled_at, broker, order_id, strategy, note, created_at
	          FROM trades WHERE 1=1`
	var args []interface{}

	if symbol != "" {
		query += " AND symbol = ?"
		args = append(args, symbol)
	}
	if days > 0 {
		cutoff := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
		query += " AND filled_at >= ?"
		args = append(args, cutoff)
	}
	query += " ORDER BY filled_at DESC"

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []Trade
	for rows.Next() {
		var t Trade
		var filledAt, createdAt string
		var err error
		if err = rows.Scan(&t.ID, &t.Symbol, &t.Side, &t.Qty, &t.Price,
			&filledAt, &t.Broker, &t.OrderID, &t.Strategy, &t.Note, &createdAt); err != nil {
			return nil, err
		}
		if t.FilledAt, err = time.Parse(time.RFC3339, filledAt); err != nil {
			return nil, fmt.Errorf("parse filled_at %q: %w", filledAt, err)
		}
		if t.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
			return nil, fmt.Errorf("parse created_at %q: %w", createdAt, err)
		}
		trades = append(trades, t)
	}
	return trades, rows.Err()
}

// PnL holds realized P&L for a symbol calculated using FIFO cost basis.
type PnL struct {
	Symbol    string
	Realized  float64
	WinCount  int
	LoseCount int
	Trades    int
}

// ComputePnL calculates realized P&L per symbol using FIFO matching.
func (d *DB) ComputePnL(symbol string) ([]PnL, error) {
	trades, err := d.List(symbol, 0)
	if err != nil {
		return nil, err
	}

	// Group by symbol, sort ascending by filled_at for FIFO
	bySymbol := map[string][]Trade{}
	for _, t := range trades {
		bySymbol[t.Symbol] = append(bySymbol[t.Symbol], t)
	}

	var results []PnL
	for sym, ts := range bySymbol {
		// reverse to get ascending order (List returns descending)
		for i, j := 0, len(ts)-1; i < j; i, j = i+1, j-1 {
			ts[i], ts[j] = ts[j], ts[i]
		}
		pnl := PnL{Symbol: sym}
		type lot struct {
			qty   int
			price float64
		}
		var lots []lot
		for _, t := range ts {
			pnl.Trades++
			if t.Side == "BUY" {
				lots = append(lots, lot{t.Qty, t.Price})
			} else {
				remaining := t.Qty
				for remaining > 0 && len(lots) > 0 {
					take := remaining
					if take > lots[0].qty {
						take = lots[0].qty
					}
					gain := float64(take) * (t.Price - lots[0].price)
					pnl.Realized += gain
					if gain >= 0 {
						pnl.WinCount++
					} else {
						pnl.LoseCount++
					}
					lots[0].qty -= take
					remaining -= take
					if lots[0].qty == 0 {
						lots = lots[1:]
					}
				}
			}
		}
		results = append(results, pnl)
	}
	return results, nil
}
