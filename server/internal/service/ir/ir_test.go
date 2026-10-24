package ir

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeQuery(t *testing.T) {
	raw := `{
		"entity": "users",
		"conditions": [
			{"field": "organization_id", "operator": "eq", "value": "acme"},
			{"field": "created_at", "operator": "between", "value": ["2026-09-01", "2026-09-30"]},
			{"field": "login_count", "operator": "gte", "value": 3},
			{"field": "deleted_at", "operator": "not_exists"}
		]
	}`
	want := Query{
		Entity: "users",
		Conditions: []Condition{
			{Field: "organization_id", Operator: OpEq, Value: "acme"},
			{Field: "created_at", Operator: OpBetween, Value: []any{"2026-09-01", "2026-09-30"}},
			{Field: "login_count", Operator: OpGte, Value: float64(3)},
			{Field: "deleted_at", Operator: OpNotExists},
		},
	}

	var got Query
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("query mismatch (-want +got):\n%s", diff)
	}
}

func TestOperatorRule(t *testing.T) {
	tests := []struct {
		op     Operator
		want   OperatorRule
		wantOK bool
	}{
		{op: OpEq, want: OperatorRule{Arity: ArityOne, PartitionKey: true, SortKey: true}, wantOK: true},
		{op: OpLt, want: OperatorRule{Arity: ArityOne, SortKey: true}, wantOK: true},
		{op: OpLte, want: OperatorRule{Arity: ArityOne, SortKey: true}, wantOK: true},
		{op: OpGt, want: OperatorRule{Arity: ArityOne, SortKey: true}, wantOK: true},
		{op: OpGte, want: OperatorRule{Arity: ArityOne, SortKey: true}, wantOK: true},
		{op: OpBetween, want: OperatorRule{Arity: ArityTwo, SortKey: true}, wantOK: true},
		{op: OpBeginsWith, want: OperatorRule{Arity: ArityOne, SortKey: true, StringOnly: true}, wantOK: true},
		{op: OpNe, want: OperatorRule{Arity: ArityOne}, wantOK: true},
		{op: OpIn, want: OperatorRule{Arity: ArityList}, wantOK: true},
		{op: OpContains, want: OperatorRule{Arity: ArityOne, StringOnly: true}, wantOK: true},
		{op: OpExists, want: OperatorRule{Arity: ArityNone}, wantOK: true},
		{op: OpNotExists, want: OperatorRule{Arity: ArityNone}, wantOK: true},
		{op: "like", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(string(tc.op), func(t *testing.T) {
			got, ok := tc.op.Rule()
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("rule mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
