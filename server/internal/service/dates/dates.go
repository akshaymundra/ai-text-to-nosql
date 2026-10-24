package dates

import (
	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

const (
	startOfDay = "T00:00:00Z"
	endOfDay   = "T23:59:59Z"
)

// Normalize expects a query that already passed validator.Validate against the same table.
// Timestamps sort as strings, so a pure date bound must be widened to the right time of day
// or ranges silently drop items from the last day.
func Normalize(q ir.Query, table dynamoschema.Table) (ir.Query, []ir.Rejection) {
	normalized := ir.Query{Entity: q.Entity, Conditions: make([]ir.Condition, 0, len(q.Conditions))}
	rejections := []ir.Rejection{}
	for _, c := range q.Conditions {
		attr, _ := table.Attribute(c.Field)
		nc, ok := normalize(c, attr.Format)
		if !ok {
			rejections = append(rejections, ir.Rejection{Code: constants.RejectCodeInvalidDateFormat, Field: c.Field})
		}
		normalized.Conditions = append(normalized.Conditions, nc)
	}
	return normalized, rejections
}

func normalize(c ir.Condition, format string) (ir.Condition, bool) {
	isPartialMatch := c.Operator == ir.OpBeginsWith || c.Operator == ir.OpContains
	if isPartialMatch {
		return c, true
	}
	switch format {
	case dynamoschema.FormatDate:
		return c, allMatch(c.Values(), dynamoschema.IsDate)
	case dynamoschema.FormatTimestamp:
		return normalizeTimestamp(c)
	default:
		return c, true
	}
}

func normalizeTimestamp(c ir.Condition) (ir.Condition, bool) {
	values := c.Values()
	switch c.Operator {
	case ir.OpEq:
		if s, _ := values[0].(string); dynamoschema.IsDate(s) {
			c.Operator = ir.OpBeginsWith
			return c, true
		}
	case ir.OpGte, ir.OpLt:
		ts, ok := toTimestamp(values[0], startOfDay)
		c.Value = ts
		return c, ok
	case ir.OpGt, ir.OpLte:
		ts, ok := toTimestamp(values[0], endOfDay)
		c.Value = ts
		return c, ok
	case ir.OpBetween:
		low, lowOK := toTimestamp(values[0], startOfDay)
		high, highOK := toTimestamp(values[1], endOfDay)
		c.Value = []any{low, high}
		return c, lowOK && highOK
	}
	return c, allMatch(values, dynamoschema.IsTimestamp)
}

func toTimestamp(v any, timeOfDay string) (string, bool) {
	s, _ := v.(string)
	switch {
	case dynamoschema.IsTimestamp(s):
		return s, true
	case dynamoschema.IsDate(s):
		return s + timeOfDay, true
	default:
		return s, false
	}
}

func allMatch(values []any, match func(string) bool) bool {
	for _, v := range values {
		s, ok := v.(string)
		if !ok || !match(s) {
			return false
		}
	}
	return true
}
