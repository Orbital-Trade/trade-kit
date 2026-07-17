package scanql

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Parse reads ScanQL text and returns a ScanPlan.
func Parse(input string) (*ScanPlan, error) {
	plan := &ScanPlan{}
	lines := strings.Split(input, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "--") {
			continue
		}
		upper := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(upper, "SCAN "):
			plan.Name = strings.TrimSpace(line[5:])

		case strings.HasPrefix(upper, "EVERY "):
			d, err := parseDuration(strings.TrimSpace(line[6:]))
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid EVERY duration: %w", i+1, err)
			}
			plan.Interval = d

		case strings.HasPrefix(upper, "SYMBOLS FROM "):
			plan.Symbols = []string{strings.TrimSpace(line[13:])} // file path marker
			// TODO: resolve file at runtime

		case strings.HasPrefix(upper, "SYMBOLS "):
			plan.Symbols = parseSymbolList(line[8:])

		case strings.HasPrefix(upper, "FETCH "):
			specs, err := parseFetchList(line[6:])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			plan.Fetch = specs

		case strings.HasPrefix(upper, "WHERE "):
			// WHERE may span multiple lines with AND
			whereText := line[6:]
			for i+1 < len(lines) {
				next := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(strings.ToUpper(next), "AND ") {
					whereText += " " + next
					i++
				} else {
					break
				}
			}
			conds, err := parseWhere(whereText)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			plan.Where = conds

		case strings.HasPrefix(upper, "ENTER "):
			action, err := parseAction(line[6:])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			// Parse indented action details
			for i+1 < len(lines) {
				next := strings.TrimSpace(lines[i+1])
				nextUpper := strings.ToUpper(next)
				if next == "" {
					i++
					continue
				}
				switch {
				case strings.HasPrefix(nextUpper, "STOP "):
					v, err := parsePercent(next[5:])
					if err != nil {
						return nil, fmt.Errorf("line %d: invalid STOP: %w", i+2, err)
					}
					action.StopPct = v
					i++
				case strings.HasPrefix(nextUpper, "TARGET "):
					pct, r, err := parseTarget(next[7:])
					if err != nil {
						return nil, fmt.Errorf("line %d: invalid TARGET: %w", i+2, err)
					}
					action.TargetPct = pct
					action.TargetR = r
					i++
				case strings.HasPrefix(nextUpper, "BUDGET "):
					v, err := strconv.ParseFloat(strings.TrimSpace(next[7:]), 64)
					if err != nil {
						return nil, fmt.Errorf("line %d: invalid BUDGET: %w", i+2, err)
					}
					action.Budget = v
					i++
				case strings.HasPrefix(nextUpper, "EXIT WHEN "):
					if plan.Exit == nil {
						plan.Exit = &ExitSpec{}
					}
					f, op, val, err := parseCondition(next[10:])
					if err != nil {
						return nil, fmt.Errorf("line %d: invalid EXIT WHEN: %w", i+2, err)
					}
					plan.Exit.WhenField = f
					plan.Exit.WhenOp = op
					plan.Exit.WhenValue = val
					i++
				case strings.HasPrefix(nextUpper, "HOLD MAX "):
					if plan.Exit == nil {
						plan.Exit = &ExitSpec{}
					}
					days, err := parseHoldMax(next[9:])
					if err != nil {
						return nil, fmt.Errorf("line %d: invalid HOLD MAX: %w", i+2, err)
					}
					plan.Exit.MaxHold = days
					i++
				case strings.HasPrefix(nextUpper, "EXIT BY "):
					if plan.Exit == nil {
						plan.Exit = &ExitSpec{}
					}
					plan.Exit.ExitBy = strings.TrimSuffix(strings.TrimSpace(next[8:]), "ET")
					plan.Exit.ExitBy = strings.TrimSpace(plan.Exit.ExitBy)
					i++
				default:
					goto doneAction
				}
			}
		doneAction:
			plan.Action = action
		}
	}

	if err := validate(plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func validate(p *ScanPlan) error {
	if p.Name == "" {
		return fmt.Errorf("missing SCAN name")
	}
	if p.Interval == 0 {
		return fmt.Errorf("missing EVERY interval")
	}
	if len(p.Symbols) == 0 {
		return fmt.Errorf("missing SYMBOLS")
	}
	if len(p.Fetch) == 0 {
		return fmt.Errorf("missing FETCH indicators")
	}
	if p.Action.Side == "" {
		return fmt.Errorf("missing ENTER direction")
	}
	return nil
}

func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasSuffix(s, "m") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * time.Minute, nil
	}
	if strings.HasSuffix(s, "s") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "s"))
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * time.Second, nil
	}
	return 0, fmt.Errorf("expected duration like 60s or 5m, got %q", s)
}

func parseSymbolList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		sym := strings.TrimSpace(strings.ToUpper(p))
		if sym != "" {
			out = append(out, sym)
		}
	}
	return out
}

func parseFetchList(s string) ([]FetchSpec, error) {
	// Split by comma, but respect parentheses: "quote, macd(12, 26, 9)" → ["quote", "macd(12, 26, 9)"]
	parts := splitRespectingParens(s)
	out := make([]FetchSpec, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p == "" {
			continue
		}
		spec := FetchSpec{}
		if idx := strings.Index(p, "("); idx >= 0 {
			spec.Name = p[:idx]
			paramStr := strings.TrimSuffix(p[idx+1:], ")")
			for _, param := range strings.Split(paramStr, ",") {
				v, err := strconv.ParseFloat(strings.TrimSpace(param), 64)
				if err != nil {
					return nil, fmt.Errorf("invalid parameter in %q: %w", p, err)
				}
				spec.Params = append(spec.Params, v)
			}
		} else {
			spec.Name = p
		}
		out = append(out, spec)
	}
	return out, nil
}

func parseWhere(s string) ([]Condition, error) {
	// Split by AND (case-insensitive)
	parts := splitByAnd(s)
	out := make([]Condition, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		upperPart := strings.ToUpper(part)
		if strings.Contains(upperPart, " BETWEEN ") {
			c, err := parseBetween(part)
			if err != nil {
				return nil, err
			}
			out = append(out, c)
			continue
		}

		f, op, v, err := parseCondition(part)
		if err != nil {
			return nil, fmt.Errorf("bad condition %q: %w", part, err)
		}
		out = append(out, Condition{Field: f, Op: op, Value: v})
	}
	return out, nil
}

func splitByAnd(s string) []string {
	var parts []string
	upper := strings.ToUpper(s)
	for {
		idx := strings.Index(upper, " AND ")
		if idx < 0 {
			parts = append(parts, strings.TrimSpace(s))
			break
		}
		// Check if this AND belongs to a BETWEEN clause.
		before := strings.ToUpper(strings.TrimSpace(s[:idx]))
		if strings.Contains(before, " BETWEEN ") || strings.HasSuffix(before, "BETWEEN") {
			// This AND is part of BETWEEN X AND Y — skip it, find the next AND.
			nextStart := idx + 5
			nextIdx := strings.Index(upper[nextStart:], " AND ")
			if nextIdx < 0 {
				parts = append(parts, strings.TrimSpace(s))
				break
			}
			parts = append(parts, strings.TrimSpace(s[:nextStart+nextIdx]))
			s = s[nextStart+nextIdx+5:]
			upper = upper[nextStart+nextIdx+5:]
			continue
		}
		parts = append(parts, strings.TrimSpace(s[:idx]))
		s = s[idx+5:]
		upper = upper[idx+5:]
	}
	return parts
}

func splitRespectingParens(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func parseCondition(s string) (field, op string, value float64, err error) {
	s = strings.TrimSpace(s)
	ops := []string{">=", "<=", "!=", "==", ">", "<"}
	for _, o := range ops {
		if idx := strings.Index(s, o); idx >= 0 {
			field = strings.TrimSpace(strings.ToLower(s[:idx]))
			valStr := strings.TrimSpace(s[idx+len(o):])
			value, err = strconv.ParseFloat(valStr, 64)
			if err != nil {
				return "", "", 0, fmt.Errorf("invalid number %q", valStr)
			}
			return field, o, value, nil
		}
	}
	return "", "", 0, fmt.Errorf("no operator found in %q", s)
}

func parseBetween(s string) (Condition, error) {
	upper := strings.ToUpper(s)
	betIdx := strings.Index(upper, " BETWEEN ")
	andIdx := strings.Index(upper[betIdx+9:], " AND ")
	if andIdx < 0 {
		return Condition{}, fmt.Errorf("BETWEEN missing AND in %q", s)
	}
	andIdx += betIdx + 9

	field := strings.TrimSpace(strings.ToLower(s[:betIdx]))
	v1Str := strings.TrimSpace(s[betIdx+9 : andIdx])
	v2Str := strings.TrimSpace(s[andIdx+5:])

	v1, err := strconv.ParseFloat(v1Str, 64)
	if err != nil {
		return Condition{}, fmt.Errorf("invalid BETWEEN value %q", v1Str)
	}
	v2, err := strconv.ParseFloat(v2Str, 64)
	if err != nil {
		return Condition{}, fmt.Errorf("invalid BETWEEN value %q", v2Str)
	}
	return Condition{Field: field, Op: "between", Value: v1, ValueEnd: v2}, nil
}

func parseAction(s string) (ActionSpec, error) {
	parts := strings.Fields(strings.ToUpper(s))
	if len(parts) == 0 {
		return ActionSpec{}, fmt.Errorf("missing direction after ENTER")
	}

	action := ActionSpec{}
	switch parts[0] {
	case "LONG":
		action.Side = "long"
	case "SHORT":
		action.Side = "short"
	default:
		return ActionSpec{}, fmt.Errorf("expected LONG or SHORT, got %q", parts[0])
	}

	// Optional: ENTER LONG TQQQ 2 SHARES
	if len(parts) >= 2 {
		action.Symbol = parts[1]
	}
	if len(parts) >= 3 {
		if n, err := strconv.Atoi(parts[2]); err == nil {
			action.Shares = n
		}
	}

	return action, nil
}

func parsePercent(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	return strconv.ParseFloat(s, 64)
}

func parseTarget(s string) (pct, r float64, err error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if strings.HasSuffix(s, "R") {
		r, err = strconv.ParseFloat(strings.TrimSuffix(s, "R"), 64)
		return 0, r, err
	}
	if strings.HasSuffix(s, "%") {
		pct, err = strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		return pct, 0, err
	}
	pct, err = strconv.ParseFloat(s, 64)
	return pct, 0, err
}

func parseHoldMax(s string) (int, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	s = strings.TrimSuffix(s, " DAYS")
	s = strings.TrimSuffix(s, " DAY")
	return strconv.Atoi(strings.TrimSpace(s))
}
