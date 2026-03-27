package server

import (
	"fmt"
	"strings"
)

type JQLExpr interface {
	Match(t *Ticket) bool
}

type jqlMatchAll struct{}

func (jqlMatchAll) Match(t *Ticket) bool { return true }

type jqlCompare struct {
	Field string
	Value string
}

func (c jqlCompare) Match(t *Ticket) bool {
	switch strings.ToLower(c.Field) {
	case "status":
		return strings.EqualFold(t.Status, c.Value)
	case "assignee":
		return strings.EqualFold(t.Assignee, c.Value)
	case "reporter":
		return strings.EqualFold(t.Reporter, c.Value)
	case "labels":
		for _, l := range t.Labels {
			if strings.EqualFold(l, c.Value) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

type jqlAnd struct {
	Left, Right JQLExpr
}

func (a jqlAnd) Match(t *Ticket) bool {
	return a.Left.Match(t) && a.Right.Match(t)
}

type jqlOr struct {
	Left, Right JQLExpr
}

func (o jqlOr) Match(t *Ticket) bool {
	return o.Left.Match(t) || o.Right.Match(t)
}

func ParseJQL(input string) (JQLExpr, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return jqlMatchAll{}, nil
	}
	return parseOr(input)
}

func parseOr(input string) (JQLExpr, error) {
	parts := splitTopLevel(input, " OR ")
	if len(parts) == 1 {
		return parseAnd(input)
	}
	left, err := parseAnd(parts[0])
	if err != nil {
		return nil, err
	}
	for _, part := range parts[1:] {
		right, err := parseAnd(part)
		if err != nil {
			return nil, err
		}
		left = jqlOr{Left: left, Right: right}
	}
	return left, nil
}

func parseAnd(input string) (JQLExpr, error) {
	parts := splitTopLevel(input, " AND ")
	if len(parts) == 1 {
		return parseComparison(input)
	}
	left, err := parseComparison(parts[0])
	if err != nil {
		return nil, err
	}
	for _, part := range parts[1:] {
		right, err := parseComparison(part)
		if err != nil {
			return nil, err
		}
		left = jqlAnd{Left: left, Right: right}
	}
	return left, nil
}

func parseComparison(input string) (JQLExpr, error) {
	input = strings.TrimSpace(input)
	eqIdx := strings.Index(input, "=")
	if eqIdx < 0 {
		return nil, fmt.Errorf("expected '=' in: %s", input)
	}
	field := strings.TrimSpace(input[:eqIdx])
	value := strings.TrimSpace(input[eqIdx+1:])
	value = strings.Trim(value, `"'`)
	return jqlCompare{Field: field, Value: value}, nil
}

func splitTopLevel(input, sep string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(input); i++ {
		if !inQuote && i+len(sep) <= len(input) && strings.EqualFold(input[i:i+len(sep)], sep) {
			parts = append(parts, current.String())
			current.Reset()
			i += len(sep) - 1
			continue
		}
		if input[i] == '"' || input[i] == '\'' {
			if !inQuote {
				inQuote = true
				quoteChar = input[i]
			} else if input[i] == quoteChar {
				inQuote = false
			}
		}
		current.WriteByte(input[i])
	}
	parts = append(parts, current.String())
	return parts
}
