package compiler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/planner"
)

type AttributeValue map[string]string

type QueryInput struct {
	TableName                 string                    `json:"TableName"`
	IndexName                 string                    `json:"IndexName,omitempty"`
	KeyConditionExpression    string                    `json:"KeyConditionExpression"`
	FilterExpression          string                    `json:"FilterExpression,omitempty"`
	ExpressionAttributeNames  map[string]string         `json:"ExpressionAttributeNames"`
	ExpressionAttributeValues map[string]AttributeValue `json:"ExpressionAttributeValues"`
	Limit                     int32                     `json:"Limit"`
}

type aliases struct {
	names      map[string]string
	nameByAttr map[string]string
	values     map[string]AttributeValue
}

func Compile(path planner.AccessPath, limit int32) (QueryInput, error) {
	a := aliases{
		names:      map[string]string{},
		nameByAttr: map[string]string{},
		values:     map[string]AttributeValue{},
	}

	keyConditions := []ir.Condition{path.PartitionKey}
	if path.SortKey != nil {
		keyConditions = append(keyConditions, *path.SortKey)
	}
	keyExpressions, err := a.expressions(keyConditions)
	if err != nil {
		return QueryInput{}, err
	}
	filterExpressions, err := a.expressions(path.Filters)
	if err != nil {
		return QueryInput{}, err
	}

	return QueryInput{
		TableName:                 path.Table,
		IndexName:                 path.Index,
		KeyConditionExpression:    strings.Join(keyExpressions, " AND "),
		FilterExpression:          joinFilters(filterExpressions),
		ExpressionAttributeNames:  a.names,
		ExpressionAttributeValues: a.values,
		Limit:                     limit,
	}, nil
}

func (a *aliases) expressions(conditions []ir.Condition) ([]string, error) {
	expressions := make([]string, 0, len(conditions))
	for _, c := range conditions {
		expression, err := a.expression(c)
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expression)
	}
	return expressions, nil
}

func (a *aliases) expression(c ir.Condition) (string, error) {
	rule, ok := c.Operator.Rule()
	if !ok {
		return "", fmt.Errorf("compiling %q: unknown operator %q", c.Field, c.Operator)
	}

	args := []any{a.name(c.Field)}
	placeholders := []string{}
	for _, v := range c.Values() {
		placeholder, err := a.value(v)
		if err != nil {
			return "", fmt.Errorf("compiling %q: %w", c.Field, err)
		}
		placeholders = append(placeholders, placeholder)
	}

	if rule.Arity == ir.ArityList {
		args = append(args, strings.Join(placeholders, ", "))
	} else {
		for _, p := range placeholders {
			args = append(args, p)
		}
	}
	return fmt.Sprintf(rule.Expression, args...), nil
}

func (a *aliases) name(attr string) string {
	if placeholder, ok := a.nameByAttr[attr]; ok {
		return placeholder
	}
	placeholder := "#n" + strconv.Itoa(len(a.names))
	a.names[placeholder] = attr
	a.nameByAttr[attr] = placeholder
	return placeholder
}

func (a *aliases) value(v any) (string, error) {
	av, err := attributeValue(v)
	if err != nil {
		return "", err
	}
	placeholder := ":v" + strconv.Itoa(len(a.values))
	a.values[placeholder] = av
	return placeholder, nil
}

func attributeValue(v any) (AttributeValue, error) {
	switch v := v.(type) {
	case string:
		return AttributeValue{"S": v}, nil
	case float64:
		return AttributeValue{"N": strconv.FormatFloat(v, 'f', -1, 64)}, nil
	default:
		return nil, fmt.Errorf("unsupported value type %T", v)
	}
}

func joinFilters(expressions []string) string {
	if len(expressions) == 0 {
		return ""
	}
	return "(" + strings.Join(expressions, ") AND (") + ")"
}
