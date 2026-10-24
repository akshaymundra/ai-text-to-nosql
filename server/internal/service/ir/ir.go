package ir

import "github.com/akshaymundra/ai-text-to-nosql/internal/constants"

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
}

var operatorRules = map[Operator]OperatorRule{
	OpEq:         {Arity: ArityOne, PartitionKey: true, SortKey: true},
	OpLt:         {Arity: ArityOne, SortKey: true},
	OpLte:        {Arity: ArityOne, SortKey: true},
	OpGt:         {Arity: ArityOne, SortKey: true},
	OpGte:        {Arity: ArityOne, SortKey: true},
	OpBetween:    {Arity: ArityTwo, SortKey: true},
	OpBeginsWith: {Arity: ArityOne, SortKey: true, StringOnly: true},
	OpNe:         {Arity: ArityOne},
	OpIn:         {Arity: ArityList},
	OpContains:   {Arity: ArityOne, StringOnly: true},
	OpExists:     {Arity: ArityNone},
	OpNotExists:  {Arity: ArityNone},
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

type Rejection struct {
	Code  constants.RejectionCode `json:"code"`
	Field string                  `json:"field,omitempty"`
}
