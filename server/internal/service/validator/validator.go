package validator

import (
	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

const maxInValues = 100

func Validate(q ir.Query, tables []dynamoschema.Table) (dynamoschema.Table, []ir.Rejection) {
	table, ok := findTable(tables, q.Entity)
	if !ok {
		return dynamoschema.Table{}, []ir.Rejection{{Code: constants.RejectCodeUnknownTable, Field: q.Entity}}
	}

	rejections := []ir.Rejection{}
	for _, c := range q.Conditions {
		if code, ok := checkCondition(c, table); !ok {
			rejections = append(rejections, ir.Rejection{Code: code, Field: c.Field})
		}
	}
	return table, rejections
}

func findTable(tables []dynamoschema.Table, name string) (dynamoschema.Table, bool) {
	for _, t := range tables {
		if t.Name == name {
			return t, true
		}
	}
	return dynamoschema.Table{}, false
}

func checkCondition(c ir.Condition, table dynamoschema.Table) (constants.RejectionCode, bool) {
	attr, ok := table.Attribute(c.Field)
	if !ok {
		return constants.RejectCodeUnknownField, false
	}
	rule, ok := c.Operator.Rule()
	if !ok {
		return constants.RejectCodeUnknownOperator, false
	}
	if rule.StringOnly && attr.Type != dynamoschema.TypeString {
		return constants.RejectCodeOperatorTypeMismatch, false
	}
	if !arityMatches(c.Value, rule.Arity) {
		return constants.RejectCodeInvalidValue, false
	}

	values := valuesOf(c.Value)
	if len(values) > maxInValues {
		return constants.RejectCodeTooManyValues, false
	}
	for _, v := range values {
		if !matchesType(v, attr.Type) {
			return constants.RejectCodeValueTypeMismatch, false
		}
	}
	return "", true
}

func arityMatches(value any, arity ir.Arity) bool {
	list, isList := value.([]any)
	switch arity {
	case ir.ArityNone:
		return value == nil
	case ir.ArityOne:
		return value != nil && !isList
	case ir.ArityTwo:
		return isList && len(list) == 2
	case ir.ArityList:
		return isList && len(list) > 0
	default:
		return false
	}
}

func valuesOf(value any) []any {
	switch v := value.(type) {
	case nil:
		return nil
	case []any:
		return v
	default:
		return []any{v}
	}
}

// Numbers are float64 because the IR is decoded from JSON with encoding/json defaults.
func matchesType(value any, attrType string) bool {
	switch attrType {
	case dynamoschema.TypeString:
		_, ok := value.(string)
		return ok
	case dynamoschema.TypeNumber:
		_, ok := value.(float64)
		return ok
	default:
		return false
	}
}
