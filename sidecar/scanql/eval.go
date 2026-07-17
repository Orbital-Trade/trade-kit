package scanql

import "fmt"

// Evaluate checks all conditions against a data map.
// Returns true if all conditions pass, false with reason if any fails.
func Evaluate(conditions []Condition, data map[string]float64) (bool, string) {
	for _, c := range conditions {
		val, ok := data[c.Field]
		if !ok {
			return false, fmt.Sprintf("%s: no data", c.Field)
		}
		switch c.Op {
		case ">=":
			if !(val >= c.Value) {
				return false, fmt.Sprintf("%s %.2f < %.2f", c.Field, val, c.Value)
			}
		case ">":
			if !(val > c.Value) {
				return false, fmt.Sprintf("%s %.2f <= %.2f", c.Field, val, c.Value)
			}
		case "<=":
			if !(val <= c.Value) {
				return false, fmt.Sprintf("%s %.2f > %.2f", c.Field, val, c.Value)
			}
		case "<":
			if !(val < c.Value) {
				return false, fmt.Sprintf("%s %.2f >= %.2f", c.Field, val, c.Value)
			}
		case "==":
			if !(val == c.Value) {
				return false, fmt.Sprintf("%s %.2f != %.2f", c.Field, val, c.Value)
			}
		case "!=":
			if !(val != c.Value) {
				return false, fmt.Sprintf("%s %.2f == %.2f", c.Field, val, c.Value)
			}
		case "between":
			if !(val >= c.Value && val <= c.ValueEnd) {
				return false, fmt.Sprintf("%s %.2f not in [%.2f, %.2f]", c.Field, val, c.Value, c.ValueEnd)
			}
		default:
			return false, fmt.Sprintf("unknown operator %q", c.Op)
		}
	}
	return true, ""
}
