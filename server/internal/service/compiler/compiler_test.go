package compiler

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/planner"
)

const testLimit = 25

var projectEq = ir.Condition{Field: "project_id", Operator: ir.OpEq, Value: "p1"}

func baseTablePath(filters ...ir.Condition) planner.AccessPath {
	return planner.AccessPath{Table: "tasks", PartitionKey: projectEq, Filters: filters}
}

func TestCompile(t *testing.T) {
	dueBetween := ir.Condition{Field: "due_date", Operator: ir.OpBetween, Value: []any{"2026-09-01", "2026-09-30"}}

	tests := []struct {
		name string
		path planner.AccessPath
		want QueryInput
	}{
		{
			name: "base table partition key only",
			path: baseTablePath(),
			want: QueryInput{
				TableName:                 "tasks",
				KeyConditionExpression:    "#n0 = :v0",
				ExpressionAttributeNames:  map[string]string{"#n0": "project_id"},
				ExpressionAttributeValues: map[string]AttributeValue{":v0": {"S": "p1"}},
				Limit:                     testLimit,
			},
		},
		{
			name: "GSI with sort key and filters",
			path: planner.AccessPath{
				Table:        "tasks",
				Index:        "assignee-due-date-index",
				PartitionKey: ir.Condition{Field: "assignee_id", Operator: ir.OpEq, Value: "u1"},
				SortKey:      &dueBetween,
				Filters: []ir.Condition{
					{Field: "priority", Operator: ir.OpGte, Value: float64(3)},
					{Field: "status", Operator: ir.OpIn, Value: []any{"open", "blocked"}},
					{Field: "deleted_at", Operator: ir.OpNotExists},
				},
			},
			want: QueryInput{
				TableName:              "tasks",
				IndexName:              "assignee-due-date-index",
				KeyConditionExpression: "#n0 = :v0 AND #n1 BETWEEN :v1 AND :v2",
				FilterExpression:       "(#n2 >= :v3) AND (#n3 IN (:v4, :v5)) AND (attribute_not_exists(#n4))",
				ExpressionAttributeNames: map[string]string{
					"#n0": "assignee_id",
					"#n1": "due_date",
					"#n2": "priority",
					"#n3": "status",
					"#n4": "deleted_at",
				},
				ExpressionAttributeValues: map[string]AttributeValue{
					":v0": {"S": "u1"},
					":v1": {"S": "2026-09-01"},
					":v2": {"S": "2026-09-30"},
					":v3": {"N": "3"},
					":v4": {"S": "open"},
					":v5": {"S": "blocked"},
				},
				Limit: testLimit,
			},
		},
		{
			name: "same attribute in two filters reuses its name placeholder",
			path: baseTablePath(
				ir.Condition{Field: "status", Operator: ir.OpNe, Value: "done"},
				ir.Condition{Field: "status", Operator: ir.OpBeginsWith, Value: "op"},
			),
			want: QueryInput{
				TableName:                "tasks",
				KeyConditionExpression:   "#n0 = :v0",
				FilterExpression:         "(#n1 <> :v1) AND (begins_with(#n1, :v2))",
				ExpressionAttributeNames: map[string]string{"#n0": "project_id", "#n1": "status"},
				ExpressionAttributeValues: map[string]AttributeValue{
					":v0": {"S": "p1"},
					":v1": {"S": "done"},
					":v2": {"S": "op"},
				},
				Limit: testLimit,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Compile(tc.path, testLimit)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("query input mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCompileOperatorExpressions(t *testing.T) {
	tests := []struct {
		filter ir.Condition
		want   string
	}{
		{filter: ir.Condition{Field: "status", Operator: ir.OpEq, Value: "a"}, want: "(#n1 = :v1)"},
		{filter: ir.Condition{Field: "status", Operator: ir.OpNe, Value: "a"}, want: "(#n1 <> :v1)"},
		{filter: ir.Condition{Field: "priority", Operator: ir.OpLt, Value: float64(1)}, want: "(#n1 < :v1)"},
		{filter: ir.Condition{Field: "priority", Operator: ir.OpLte, Value: float64(1)}, want: "(#n1 <= :v1)"},
		{filter: ir.Condition{Field: "priority", Operator: ir.OpGt, Value: float64(1)}, want: "(#n1 > :v1)"},
		{filter: ir.Condition{Field: "priority", Operator: ir.OpGte, Value: float64(1)}, want: "(#n1 >= :v1)"},
		{
			filter: ir.Condition{Field: "priority", Operator: ir.OpBetween, Value: []any{float64(1), float64(5)}},
			want:   "(#n1 BETWEEN :v1 AND :v2)",
		},
		{filter: ir.Condition{Field: "status", Operator: ir.OpBeginsWith, Value: "a"}, want: "(begins_with(#n1, :v1))"},
		{filter: ir.Condition{Field: "status", Operator: ir.OpContains, Value: "a"}, want: "(contains(#n1, :v1))"},
		{filter: ir.Condition{Field: "status", Operator: ir.OpIn, Value: []any{"a"}}, want: "(#n1 IN (:v1))"},
		{filter: ir.Condition{Field: "status", Operator: ir.OpExists}, want: "(attribute_exists(#n1))"},
		{filter: ir.Condition{Field: "status", Operator: ir.OpNotExists}, want: "(attribute_not_exists(#n1))"},
	}

	for _, tc := range tests {
		t.Run(string(tc.filter.Operator), func(t *testing.T) {
			got, err := Compile(baseTablePath(tc.filter), testLimit)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.FilterExpression != tc.want {
				t.Errorf("filter = %q, want %q", got.FilterExpression, tc.want)
			}
		})
	}
}

func TestCompileErrors(t *testing.T) {
	tests := []struct {
		name   string
		filter ir.Condition
	}{
		{name: "unknown operator", filter: ir.Condition{Field: "status", Operator: "like", Value: "a"}},
		{name: "unsupported value type", filter: ir.Condition{Field: "archived", Operator: ir.OpEq, Value: true}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Compile(baseTablePath(tc.filter), testLimit); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestAttributeValue(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  AttributeValue
	}{
		{name: "string", value: "acme", want: AttributeValue{"S": "acme"}},
		{name: "empty string", value: "", want: AttributeValue{"S": ""}},
		{name: "integer", value: float64(3), want: AttributeValue{"N": "3"}},
		{name: "decimal", value: 2.5, want: AttributeValue{"N": "2.5"}},
		{name: "negative", value: float64(-7), want: AttributeValue{"N": "-7"}},
		{name: "large number without exponent", value: 1e21, want: AttributeValue{"N": "1000000000000000000000"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := attributeValue(tc.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("attribute value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestQueryInputJSON(t *testing.T) {
	got, err := Compile(baseTablePath(), testLimit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"TableName":"tasks","KeyConditionExpression":"#n0 = :v0",` +
		`"ExpressionAttributeNames":{"#n0":"project_id"},` +
		`"ExpressionAttributeValues":{":v0":{"S":"p1"}},"Limit":25}`
	if string(data) != want {
		t.Errorf("json = %s\nwant   %s", data, want)
	}
}
