package extraction

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

var testTables = []dynamoschema.Table{
	{
		Name:         "users",
		Description:  "People who can log in.",
		PartitionKey: "user_id",
		Attributes: []dynamoschema.Attribute{
			{Name: "user_id", Type: dynamoschema.TypeString, Description: "User ID, e.g. u_001."},
			{Name: "status", Type: dynamoschema.TypeString, Description: "Account status: active or invited."},
			{
				Name:        "created_at",
				Type:        dynamoschema.TypeString,
				Format:      dynamoschema.FormatTimestamp,
				Description: "Signup time.",
			},
		},
		Indexes: []dynamoschema.Index{{Name: "status-index", PartitionKey: "status"}},
	},
	{
		Name:        "invoices",
		Description: "Invoices billed to organizations.",
		Attributes: []dynamoschema.Attribute{
			{Name: "status", Type: dynamoschema.TypeString, Description: "Payment status: paid, open or overdue."},
			{Name: "amount", Type: dynamoschema.TypeNumber, Description: "Amount in cents."},
			{Name: "due_date", Type: dynamoschema.TypeString, Format: dynamoschema.FormatDate, Description: "Due date."},
		},
	},
}

func TestSchema(t *testing.T) {
	raw, err := Schema(testTables)
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	scalar := []any{map[string]any{"type": "string"}, map[string]any{"type": "number"}}
	want := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"entity": map[string]any{"type": "string", "enum": []any{"users", "invoices"}},
			"conditions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"field": map[string]any{
							"type": "string",
							"enum": []any{"user_id", "status", "created_at", "amount", "due_date"},
						},
						"operator": map[string]any{
							"type": "string",
							"enum": []any{
								"begins_with", "between", "contains", "eq", "exists", "gt",
								"gte", "in", "lt", "lte", "ne", "not_exists",
							},
						},
						"value": map[string]any{
							"anyOf": append(scalar, map[string]any{"type": "array", "items": map[string]any{"anyOf": scalar}}),
						},
					},
					"required": []any{"field", "operator"},
				},
			},
			"unmapped_terms": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []any{"entity", "conditions", "unmapped_terms"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("schema mismatch (-want +got):\n%s", diff)
	}
}

func TestSystemPrompt(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

	got := SystemPrompt(testTables, now)

	wantParts := []string{
		"Today is Wednesday, 2026-10-07 (UTC).",
		"## users\nPeople who can log in.\nFields:\n" +
			"- user_id (string): User ID, e.g. u_001.\n" +
			"- status (string): Account status: active or invited.\n" +
			"- created_at (string, timestamp): Signup time.\n",
		"## invoices\nInvoices billed to organizations.\nFields:\n" +
			"- status (string): Payment status: paid, open or overdue.\n" +
			"- amount (number): Amount in cents.\n" +
			"- due_date (string, date): Due date.\n",
		"unmapped_terms:",
	}
	for _, part := range wantParts {
		if !strings.Contains(got, part) {
			t.Errorf("prompt is missing %q", part)
		}
	}
	for _, hidden := range []string{"status-index", "partition", "sort key"} {
		if strings.Contains(strings.ToLower(got), hidden) {
			t.Errorf("prompt mentions %q; keys and indexes are the planner's job", hidden)
		}
	}
}

func TestParse(t *testing.T) {
	answer := `{
		"entity": "users",
		"conditions": [
			{"field": "status", "operator": "eq", "value": "active"},
			{"field": "created_at", "operator": "between", "value": ["2026-09-01", "2026-09-30"]},
			{"field": "last_login_at", "operator": "not_exists", "value": ""}
		],
		"unmapped_terms": ["phone number"]
	}`

	got, err := Parse(answer)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	want := Extraction{
		Query: ir.Query{Entity: "users", Conditions: []ir.Condition{
			{Field: "status", Operator: ir.OpEq, Value: "active"},
			{Field: "created_at", Operator: ir.OpBetween, Value: []any{"2026-09-01", "2026-09-30"}},
			{Field: "last_login_at", Operator: ir.OpNotExists},
		}},
		UnmappedTerms: []string{"phone number"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("extraction mismatch (-want +got):\n%s", diff)
	}
}

func TestParseRejects(t *testing.T) {
	tests := []struct {
		name   string
		answer string
	}{
		{name: "not json", answer: `entity: users`},
		{name: "unknown key", answer: `{"entity": "users", "conditions": [], "unmapped_terms": [], "index": "status-index"}`},
		{name: "missing entity", answer: `{"conditions": [], "unmapped_terms": []}`},
		{name: "trailing content", answer: `{"entity": "users", "conditions": [], "unmapped_terms": []} {"entity": "x"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(tc.answer); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}
