package validator

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

var testTables = []dynamoschema.Table{
	{
		Name: "users",
		Attributes: []dynamoschema.Attribute{
			{Name: "organization_id", Type: dynamoschema.TypeString},
			{Name: "created_at", Type: dynamoschema.TypeString},
			{Name: "status", Type: dynamoschema.TypeString},
			{Name: "login_count", Type: dynamoschema.TypeNumber},
		},
	},
	{
		Name: "projects",
		Attributes: []dynamoschema.Attribute{
			{Name: "project_id", Type: dynamoschema.TypeString},
		},
	},
}

func usersQuery(conditions ...ir.Condition) ir.Query {
	return ir.Query{Entity: "users", Conditions: conditions}
}

func numbers(n int) []any {
	values := make([]any, n)
	for i := range values {
		values[i] = float64(i)
	}
	return values
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		query     ir.Query
		wantTable string
		want      []ir.Rejection
	}{
		{
			name: "valid query with every value shape",
			query: usersQuery(
				ir.Condition{Field: "organization_id", Operator: ir.OpEq, Value: "acme"},
				ir.Condition{Field: "created_at", Operator: ir.OpBetween, Value: []any{"2026-09-01", "2026-09-30"}},
				ir.Condition{Field: "login_count", Operator: ir.OpGte, Value: float64(3)},
				ir.Condition{Field: "status", Operator: ir.OpIn, Value: []any{"active", "invited"}},
				ir.Condition{Field: "status", Operator: ir.OpBeginsWith, Value: "act"},
				ir.Condition{Field: "status", Operator: ir.OpNotExists},
			),
			wantTable: "users",
			want:      []ir.Rejection{},
		},
		{
			name:      "in with 100 values is the limit",
			query:     usersQuery(ir.Condition{Field: "login_count", Operator: ir.OpIn, Value: numbers(100)}),
			wantTable: "users",
			want:      []ir.Rejection{},
		},
		{
			name:  "unknown table",
			query: ir.Query{Entity: "people"},
			want:  []ir.Rejection{{Code: constants.RejectCodeUnknownTable, Field: "people"}},
		},
		{
			name:      "unknown field",
			query:     usersQuery(ir.Condition{Field: "email", Operator: ir.OpEq, Value: "a@b.c"}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeUnknownField, Field: "email"}},
		},
		{
			name:      "field from another table",
			query:     usersQuery(ir.Condition{Field: "project_id", Operator: ir.OpEq, Value: "p1"}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeUnknownField, Field: "project_id"}},
		},
		{
			name:      "unknown operator",
			query:     usersQuery(ir.Condition{Field: "status", Operator: "like", Value: "act"}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeUnknownOperator, Field: "status"}},
		},
		{
			name:      "begins_with on number attribute",
			query:     usersQuery(ir.Condition{Field: "login_count", Operator: ir.OpBeginsWith, Value: float64(1)}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeOperatorTypeMismatch, Field: "login_count"}},
		},
		{
			name:      "contains on number attribute",
			query:     usersQuery(ir.Condition{Field: "login_count", Operator: ir.OpContains, Value: float64(1)}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeOperatorTypeMismatch, Field: "login_count"}},
		},
		{
			name:      "exists with a value",
			query:     usersQuery(ir.Condition{Field: "status", Operator: ir.OpExists, Value: "active"}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeInvalidValue, Field: "status"}},
		},
		{
			name:      "eq without a value",
			query:     usersQuery(ir.Condition{Field: "status", Operator: ir.OpEq}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeInvalidValue, Field: "status"}},
		},
		{
			name:      "eq with a list",
			query:     usersQuery(ir.Condition{Field: "status", Operator: ir.OpEq, Value: []any{"active"}}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeInvalidValue, Field: "status"}},
		},
		{
			name:      "between with one value",
			query:     usersQuery(ir.Condition{Field: "created_at", Operator: ir.OpBetween, Value: []any{"2026-09-01"}}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeInvalidValue, Field: "created_at"}},
		},
		{
			name:      "between with a scalar",
			query:     usersQuery(ir.Condition{Field: "created_at", Operator: ir.OpBetween, Value: "2026-09-01"}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeInvalidValue, Field: "created_at"}},
		},
		{
			name:      "in with an empty list",
			query:     usersQuery(ir.Condition{Field: "status", Operator: ir.OpIn, Value: []any{}}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeInvalidValue, Field: "status"}},
		},
		{
			name:      "in with more than 100 values",
			query:     usersQuery(ir.Condition{Field: "login_count", Operator: ir.OpIn, Value: numbers(101)}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeTooManyValues, Field: "login_count"}},
		},
		{
			name:      "number value on string attribute",
			query:     usersQuery(ir.Condition{Field: "status", Operator: ir.OpEq, Value: float64(1)}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeValueTypeMismatch, Field: "status"}},
		},
		{
			name:      "string value on number attribute",
			query:     usersQuery(ir.Condition{Field: "login_count", Operator: ir.OpEq, Value: "3"}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeValueTypeMismatch, Field: "login_count"}},
		},
		{
			name:      "between with one wrong type",
			query:     usersQuery(ir.Condition{Field: "login_count", Operator: ir.OpBetween, Value: []any{float64(1), "9"}}),
			wantTable: "users",
			want:      []ir.Rejection{{Code: constants.RejectCodeValueTypeMismatch, Field: "login_count"}},
		},
		{
			name: "one rejection per bad condition, in condition order",
			query: usersQuery(
				ir.Condition{Field: "email", Operator: ir.OpEq, Value: "a@b.c"},
				ir.Condition{Field: "organization_id", Operator: ir.OpEq, Value: "acme"},
				ir.Condition{Field: "login_count", Operator: ir.OpEq, Value: "3"},
			),
			wantTable: "users",
			want: []ir.Rejection{
				{Code: constants.RejectCodeUnknownField, Field: "email"},
				{Code: constants.RejectCodeValueTypeMismatch, Field: "login_count"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			table, got := Validate(tc.query, testTables)
			if table.Name != tc.wantTable {
				t.Errorf("table = %q, want %q", table.Name, tc.wantTable)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("rejections mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
