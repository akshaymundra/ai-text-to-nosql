package evals

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/extraction"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

var scanCase = Case{
	ID:     "031-users-name-starts-with",
	Tags:   []string{"reject", "scan"},
	IR:     ir.Query{Entity: "users", Conditions: []ir.Condition{{Field: "name", Operator: ir.OpBeginsWith, Value: "A"}}},
	Expect: Outcome{Status: model.QueryStatusRejected, Code: constants.RejectCodeScanRequired},
}

func scanRun(conditions []ir.Condition) service.Run {
	return service.Run{
		Result: service.QueryResult{
			Status:     model.QueryStatusRejected,
			Rejections: []ir.Rejection{{Code: constants.RejectCodeScanRequired}},
		},
		Extraction: extraction.Extraction{
			Query:         ir.Query{Entity: "users", Conditions: conditions},
			UnmappedTerms: []string{},
		},
	}
}

func TestEvaluate(t *testing.T) {
	t.Run("matching outcome and IR", func(t *testing.T) {
		got, err := Evaluate(scanCase, scanRun(scanCase.IR.Conditions))
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if !got.Passed || got.Diff != "" || got.IRDiff != "" {
			t.Errorf("passed %v, diff %q, IR diff %q; want a clean pass", got.Passed, got.Diff, got.IRDiff)
		}
	})

	t.Run("same outcome from a different IR still passes but shows the IR diff", func(t *testing.T) {
		other := []ir.Condition{{Field: "name", Operator: ir.OpContains, Value: "A"}}
		got, err := Evaluate(scanCase, scanRun(other))
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if !got.Passed || got.IRDiff == "" {
			t.Errorf("passed %v, IR diff %q; want a pass with an IR diff", got.Passed, got.IRDiff)
		}
	})

	t.Run("different outcome fails", func(t *testing.T) {
		run := scanRun(scanCase.IR.Conditions)
		run.Result.Rejections = []ir.Rejection{{Code: constants.RejectCodeUnknownField, Field: "name"}}
		got, err := Evaluate(scanCase, run)
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		if got.Passed || got.Diff == "" {
			t.Errorf("passed %v, diff %q; want a failure with a diff", got.Passed, got.Diff)
		}
	})
}

func TestScoreByTag(t *testing.T) {
	results := []CaseResult{
		{Case: Case{Tags: []string{"gsi", "filter"}}, Passed: true},
		{Case: Case{Tags: []string{"gsi"}}, Passed: false},
		{Case: Case{Tags: []string{"reject"}}, Passed: true},
	}

	want := []TagScore{
		{Tag: "filter", Passed: 1, Total: 1},
		{Tag: "gsi", Passed: 1, Total: 2},
		{Tag: "reject", Passed: 1, Total: 1},
	}
	if diff := cmp.Diff(want, ScoreByTag(results)); diff != "" {
		t.Errorf("scores mismatch (-want +got):\n%s", diff)
	}
}
