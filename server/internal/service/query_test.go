package service

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/compiler"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/extraction"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/planner"
)

var usersTables = []dynamoschema.Table{
	{
		Name:         "users",
		PartitionKey: "user_id",
		Attributes: []dynamoschema.Attribute{
			{Name: "user_id", Type: dynamoschema.TypeString},
			{Name: "organization_id", Type: dynamoschema.TypeString},
			{Name: "created_at", Type: dynamoschema.TypeString, Format: dynamoschema.FormatTimestamp},
			{Name: "status", Type: dynamoschema.TypeString},
		},
		Indexes: []dynamoschema.Index{
			{Name: "organization-created-at-index", PartitionKey: "organization_id", SortKey: "created_at"},
		},
	},
}

func TestBuildQuery(t *testing.T) {
	tests := []struct {
		name  string
		query ir.Query
		want  QueryResult
	}{
		{
			name: "valid GSI query with widened dates and a filter",
			query: ir.Query{Entity: "users", Conditions: []ir.Condition{
				{Field: "organization_id", Operator: ir.OpEq, Value: "acme"},
				{Field: "created_at", Operator: ir.OpBetween, Value: []any{"2026-09-01", "2026-09-30"}},
				{Field: "status", Operator: ir.OpEq, Value: "active"},
			}},
			want: QueryResult{
				Status: model.QueryStatusValid,
				Table:  "users",
				Index:  "organization-created-at-index",
				Query: &compiler.QueryInput{
					TableName:              "users",
					IndexName:              "organization-created-at-index",
					KeyConditionExpression: "#n0 = :v0 AND #n1 BETWEEN :v1 AND :v2",
					FilterExpression:       "(#n2 = :v3)",
					ExpressionAttributeNames: map[string]string{
						"#n0": "organization_id",
						"#n1": "created_at",
						"#n2": "status",
					},
					ExpressionAttributeValues: map[string]compiler.AttributeValue{
						":v0": {"S": "acme"},
						":v1": {"S": "2026-09-01T00:00:00Z"},
						":v2": {"S": "2026-09-30T23:59:59Z"},
						":v3": {"S": "active"},
					},
					Limit: 25,
				},
				Warnings: []planner.Warning{
					{Code: constants.WarnCodeFilterExpression, Field: "status"},
					{Code: constants.WarnCodeEventuallyConsistent},
				},
				Rejections: []ir.Rejection{},
			},
		},
		{
			name: "rejected by validator",
			query: ir.Query{Entity: "users", Conditions: []ir.Condition{
				{Field: "email", Operator: ir.OpEq, Value: "a@b.c"},
			}},
			want: QueryResult{
				Status:     model.QueryStatusRejected,
				Warnings:   []planner.Warning{},
				Rejections: []ir.Rejection{{Code: constants.RejectCodeUnknownField, Field: "email"}},
			},
		},
		{
			name: "rejected by dates",
			query: ir.Query{Entity: "users", Conditions: []ir.Condition{
				{Field: "organization_id", Operator: ir.OpEq, Value: "acme"},
				{Field: "created_at", Operator: ir.OpGte, Value: "last month"},
			}},
			want: QueryResult{
				Status:     model.QueryStatusRejected,
				Warnings:   []planner.Warning{},
				Rejections: []ir.Rejection{{Code: constants.RejectCodeInvalidDateFormat, Field: "created_at"}},
			},
		},
		{
			name: "rejected by planner",
			query: ir.Query{Entity: "users", Conditions: []ir.Condition{
				{Field: "status", Operator: ir.OpEq, Value: "active"},
			}},
			want: QueryResult{
				Status:     model.QueryStatusRejected,
				Warnings:   []planner.Warning{},
				Rejections: []ir.Rejection{{Code: constants.RejectCodeScanRequired}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BuildQuery(extraction.Extraction{Query: tc.query}, usersTables, 25)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildQueryUnmappedTerms(t *testing.T) {
	ex := extraction.Extraction{
		Query:         ir.Query{Entity: "users", Conditions: []ir.Condition{{Field: "status", Operator: ir.OpEq, Value: "active"}}},
		UnmappedTerms: []string{"phone number", "salary"},
	}

	got, err := BuildQuery(ex, usersTables, 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := QueryResult{
		Status:   model.QueryStatusRejected,
		Warnings: []planner.Warning{},
		Rejections: []ir.Rejection{
			{Code: constants.RejectCodeUnknownField, Field: "phone number"},
			{Code: constants.RejectCodeUnknownField, Field: "salary"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
