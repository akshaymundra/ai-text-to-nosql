package evals

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/compiler"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

const queryLimit = 25

// TestCases runs each case's reference IR through the pipeline, which proves the expect blocks are right
// before any LLM is involved.
func TestCases(t *testing.T) {
	tables, err := dynamoschema.LoadDir("../schemas-local")
	if err != nil {
		t.Fatalf("loading schemas: %v", err)
	}
	cases, err := LoadCases("cases")
	if err != nil {
		t.Fatalf("loading cases: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("no eval cases found")
	}

	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			result, err := service.BuildQuery(c.IR, tables, queryLimit)
			if err != nil {
				t.Fatalf("build query: %v", err)
			}
			got, err := OutcomeOf(result)
			if err != nil {
				t.Fatalf("outcome: %v", err)
			}
			if diff := cmp.Diff(c.Expect, got); diff != "" {
				t.Errorf("outcome mismatch (-expect +got):\n%s", diff)
			}
		})
	}
}

func TestOutcomeOf(t *testing.T) {
	tests := []struct {
		name   string
		result service.QueryResult
		want   Outcome
	}{
		{
			name: "valid result with string and number values",
			result: service.QueryResult{
				Status: model.QueryStatusValid,
				Table:  "invoices",
				Query: &compiler.QueryInput{
					KeyConditionExpression:   "#n0 = :v0",
					FilterExpression:         "(#n1 >= :v1)",
					ExpressionAttributeNames: map[string]string{"#n0": "organization_id", "#n1": "amount"},
					ExpressionAttributeValues: map[string]compiler.AttributeValue{
						":v0": {"S": "acme"},
						":v1": {"N": "4999.5"},
					},
				},
			},
			want: Outcome{
				Status: model.QueryStatusValid,
				Table:  "invoices",
				Query: &Query{
					KeyConditionExpression:    "#n0 = :v0",
					FilterExpression:          "(#n1 >= :v1)",
					ExpressionAttributeNames:  map[string]string{"#n0": "organization_id", "#n1": "amount"},
					ExpressionAttributeValues: map[string]any{":v0": "acme", ":v1": 4999.5},
				},
			},
		},
		{
			name: "rejected result keeps only the first rejection",
			result: service.QueryResult{
				Status: model.QueryStatusRejected,
				Rejections: []ir.Rejection{
					{Code: constants.RejectCodeUnknownField, Field: "email"},
					{Code: constants.RejectCodeValueTypeMismatch, Field: "amount"},
				},
			},
			want: Outcome{Status: model.QueryStatusRejected, Code: constants.RejectCodeUnknownField, Field: "email"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := OutcomeOf(tc.result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("outcome mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoadCasesRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	bad := []byte(`{"id": "x", "expct": {"status": "valid"}}`)
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), bad, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCases(dir); err == nil {
		t.Error("expected an error for a misspelled key, got nil")
	}
}
