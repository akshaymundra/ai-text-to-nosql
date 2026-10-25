package ir

import (
	"maps"
	"slices"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
)

type Operator string

const (
	OpEq         Operator = "eq"
	OpLt         Operator = "lt"
	OpLte        Operator = "lte"
	OpGt         Operator = "gt"
	OpGte        Operator = "gte"
	OpBetween    Operator = "between"
	OpBeginsWith Operator = "begins_with"
	OpNe         Operator = "ne"
	OpIn         Operator = "in"
	OpContains   Operator = "contains"
	OpExists     Operator = "exists"
	OpNotExists  Operator = "not_exists"
)

// the number of operands or parameters
type Arity int

const (
	ArityNone Arity = iota
	ArityOne
	ArityTwo
	ArityList
)

type OperatorRule struct {
	Arity        Arity
	PartitionKey bool
	SortKey      bool
	StringOnly   bool
	Expression   string
}

var operatorRules = map[Operator]OperatorRule{
	OpEq:         {Arity: ArityOne, PartitionKey: true, SortKey: true, Expression: "%s = %s"},
	OpLt:         {Arity: ArityOne, SortKey: true, Expression: "%s < %s"},
	OpLte:        {Arity: ArityOne, SortKey: true, Expression: "%s <= %s"},
	OpGt:         {Arity: ArityOne, SortKey: true, Expression: "%s > %s"},
	OpGte:        {Arity: ArityOne, SortKey: true, Expression: "%s >= %s"},
	OpBetween:    {Arity: ArityTwo, SortKey: true, Expression: "%s BETWEEN %s AND %s"},
	OpBeginsWith: {Arity: ArityOne, SortKey: true, StringOnly: true, Expression: "begins_with(%s, %s)"},
	OpNe:         {Arity: ArityOne, Expression: "%s <> %s"},
	OpIn:         {Arity: ArityList, Expression: "%s IN (%s)"},
	OpContains:   {Arity: ArityOne, StringOnly: true, Expression: "contains(%s, %s)"},
	OpExists:     {Arity: ArityNone, Expression: "attribute_exists(%s)"},
	OpNotExists:  {Arity: ArityNone, Expression: "attribute_not_exists(%s)"},
}

// Operators lists every operator in a fixed order, so anything built from it (like a JSON schema enum) is stable.
func Operators() []Operator {
	return slices.Sorted(maps.Keys(operatorRules))
}

func (o Operator) Rule() (OperatorRule, bool) {
	rule, ok := operatorRules[o]
	return rule, ok
}

type Query struct {
	Entity     string      `json:"entity"`
	Conditions []Condition `json:"conditions"`
}

type Condition struct {
	Field    string   `json:"field"`
	Operator Operator `json:"operator"`
	Value    any      `json:"value,omitempty"`
}

func (c Condition) Values() []any {
	switch v := c.Value.(type) {
	case nil:
		return nil
	case []any:
		return v
	default:
		return []any{v}
	}
}

type Rejection struct {
	Code       constants.RejectionCode `json:"code"`
	Field      string                  `json:"field,omitempty"`
	Suggestion string                  `json:"suggestion,omitempty"`
}
