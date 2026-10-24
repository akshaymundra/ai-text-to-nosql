package planner

import (
	"slices"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

type AccessPath struct {
	Table        string
	Index        string
	PartitionKey ir.Condition
	SortKey      *ir.Condition
	Filters      []ir.Condition
}

type Warning struct {
	Code  constants.WarningCode `json:"code"`
	Field string                `json:"field,omitempty"`
}

type keySchema struct {
	index        string
	partitionKey string
	sortKey      string
}

// Plan expects a query that already passed validator.Validate against the same table.
func Plan(q ir.Query, table dynamoschema.Table) (AccessPath, *ir.Rejection) {
	var paths []AccessPath
	var firstRejection *ir.Rejection
	for _, key := range keySchemas(table) {
		if !hasPartitionKeyEq(q.Conditions, key.partitionKey) {
			continue
		}
		path, rejection := buildPath(table.Name, key, q.Conditions)
		if rejection != nil {
			if firstRejection == nil {
				firstRejection = rejection
			}
			continue
		}
		paths = append(paths, path)
	}

	if len(paths) > 0 {
		return pickBest(paths), nil
	}
	if firstRejection != nil {
		return AccessPath{}, firstRejection
	}
	return AccessPath{}, &ir.Rejection{Code: constants.RejectCodeScanRequired}
}

func (p AccessPath) Warnings() []Warning {
	warnings := []Warning{}
	for _, f := range p.Filters {
		warnings = append(warnings, Warning{Code: constants.WarnCodeFilterExpression, Field: f.Field})
	}
	if p.Index != "" {
		warnings = append(warnings, Warning{Code: constants.WarnCodeEventuallyConsistent})
	}
	return warnings
}

func keySchemas(table dynamoschema.Table) []keySchema {
	keys := []keySchema{{partitionKey: table.PartitionKey, sortKey: table.SortKey}}
	for _, idx := range table.Indexes {
		keys = append(keys, keySchema{index: idx.Name, partitionKey: idx.PartitionKey, sortKey: idx.SortKey})
	}
	return keys
}

func hasPartitionKeyEq(conditions []ir.Condition, partitionKey string) bool {
	return slices.ContainsFunc(conditions, func(c ir.Condition) bool {
		return c.Field == partitionKey && c.Operator == ir.OpEq
	})
}

func buildPath(tableName string, key keySchema, conditions []ir.Condition) (AccessPath, *ir.Rejection) {
	path := AccessPath{Table: tableName, Index: key.index, Filters: []ir.Condition{}}
	partitionKeySet := false
	for _, c := range conditions {
		rule, _ := c.Operator.Rule()
		switch c.Field {
		case key.partitionKey:
			if code, ok := checkKeyCondition(c, partitionKeySet, rule.PartitionKey); !ok {
				return AccessPath{}, &ir.Rejection{Code: code, Field: c.Field}
			}
			path.PartitionKey = c
			partitionKeySet = true
		case key.sortKey:
			if code, ok := checkKeyCondition(c, path.SortKey != nil, rule.SortKey); !ok {
				return AccessPath{}, &ir.Rejection{Code: code, Field: c.Field}
			}
			path.SortKey = &c
		default:
			path.Filters = append(path.Filters, c)
		}
	}
	return path, nil
}

func checkKeyCondition(c ir.Condition, alreadySet, allowed bool) (constants.RejectionCode, bool) {
	if alreadySet {
		return constants.RejectCodeMultipleKeyConditions, false
	}
	if !allowed {
		return constants.RejectCodeInvalidKeyCondition, false
	}
	if hasEmptyString(c.Value) {
		return constants.RejectCodeEmptyKeyValue, false
	}
	return "", true
}

func hasEmptyString(value any) bool {
	values, ok := value.([]any)
	if !ok {
		values = []any{value}
	}
	return slices.Contains(values, any(""))
}

func pickBest(paths []AccessPath) AccessPath {
	for _, p := range paths {
		if p.SortKey != nil {
			return p
		}
	}
	return paths[0]
}
