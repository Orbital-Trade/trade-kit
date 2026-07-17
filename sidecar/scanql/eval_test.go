package scanql

import "testing"

func TestEvaluate(t *testing.T) {
	data := map[string]float64{
		"rsi":     18.3,
		"price":   185.50,
		"volume":  2400000,
		"gap_pct": 8.5,
	}

	tests := []struct {
		name   string
		conds  []Condition
		expect bool
	}{
		{"rsi pass", []Condition{{Field: "rsi", Op: "<=", Value: 25}}, true},
		{"rsi fail", []Condition{{Field: "rsi", Op: "<=", Value: 15}}, false},
		{"volume pass", []Condition{{Field: "volume", Op: ">=", Value: 500000}}, true},
		{"volume fail", []Condition{{Field: "volume", Op: ">=", Value: 5000000}}, false},
		{"between pass", []Condition{{Field: "gap_pct", Op: "between", Value: 3, ValueEnd: 20}}, true},
		{"between fail", []Condition{{Field: "gap_pct", Op: "between", Value: 10, ValueEnd: 20}}, false},
		{"multiple pass", []Condition{
			{Field: "rsi", Op: "<=", Value: 25},
			{Field: "volume", Op: ">=", Value: 500000},
			{Field: "price", Op: ">=", Value: 5},
		}, true},
		{"multiple fail", []Condition{
			{Field: "rsi", Op: "<=", Value: 25},
			{Field: "volume", Op: ">=", Value: 5000000}, // fails
		}, false},
		{"missing field", []Condition{{Field: "macd", Op: ">=", Value: 0}}, false},
		{"equals", []Condition{{Field: "rsi", Op: "==", Value: 18.3}}, true},
		{"not equals", []Condition{{Field: "rsi", Op: "!=", Value: 50}}, true},
		{"greater", []Condition{{Field: "price", Op: ">", Value: 100}}, true},
		{"less", []Condition{{Field: "rsi", Op: "<", Value: 20}}, true},
	}

	for _, tt := range tests {
		pass, reason := Evaluate(tt.conds, data)
		if pass != tt.expect {
			t.Errorf("%s: got %v (reason: %s), want %v", tt.name, pass, reason, tt.expect)
		}
	}
}
