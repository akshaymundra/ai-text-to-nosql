package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/compiler"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/extraction"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/planner"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/retrieval"
)

const runRequest = "Acme users in account state active"

func runnerEmbedder() *textEmbedder {
	vectors := map[string][]float32{
		retrieval.QueryText(runRequest):     {1, 0},
		retrieval.TermText("account state"): {0.6, 0.8},
	}
	for _, a := range usersTables[0].Attributes {
		vectors[retrieval.FieldText(a)] = []float32{1, 0}
	}
	vectors[retrieval.FieldText(usersTables[0].Attributes[3])] = []float32{0.6, 0.8}
	return &textEmbedder{vectors: vectors}
}

func usersSearcher() *fakePatternSearcher {
	return &fakePatternSearcher{matches: []model.AccessPatternMatch{
		{ID: "ap-1", TableName: "users", Definition: usersTables[0], Distance: 0.2},
	}}
}

func newTestRunner(searcher *fakePatternSearcher, chat *fakeChatter) *QueryRunner {
	embedder := runnerEmbedder()
	return NewQueryRunner(
		NewRetriever(searcher, embedder, testEmbeddingModel),
		NewExtractor(chat, "qwen3.5:9b", 42),
		NewFieldSuggester(embedder, testEmbeddingModel),
	)
}

func TestRun(t *testing.T) {
	retrieved := []model.RetrievedPattern{{AccessPatternID: "ap-1", Distance: 0.2}}

	tests := []struct {
		name     string
		searcher *fakePatternSearcher
		answer   string
		want     Run
	}{
		{
			name:     "valid query",
			searcher: usersSearcher(),
			answer: `{"entity": "users", "conditions": [` +
				`{"field": "organization_id", "operator": "eq", "value": "acme"},` +
				`{"field": "status", "operator": "eq", "value": "active"}], "unmapped_terms": []}`,
			want: Run{
				Result: QueryResult{
					Status: model.QueryStatusValid,
					Table:  "users",
					Index:  "organization-created-at-index",
					Query: &compiler.QueryInput{
						TableName:                 "users",
						IndexName:                 "organization-created-at-index",
						KeyConditionExpression:    "#n0 = :v0",
						FilterExpression:          "(#n1 = :v1)",
						ExpressionAttributeNames:  map[string]string{"#n0": "organization_id", "#n1": "status"},
						ExpressionAttributeValues: map[string]compiler.AttributeValue{":v0": {"S": "acme"}, ":v1": {"S": "active"}},
						Limit:                     DefaultQueryLimit,
					},
					Warnings: []planner.Warning{
						{Code: constants.WarnCodeFilterExpression, Field: "status"},
						{Code: constants.WarnCodeEventuallyConsistent},
					},
					Rejections: []ir.Rejection{},
				},
				Extraction: extraction.Extraction{
					Query: ir.Query{Entity: "users", Conditions: []ir.Condition{
						{Field: "organization_id", Operator: ir.OpEq, Value: "acme"},
						{Field: "status", Operator: ir.OpEq, Value: "active"},
					}},
					UnmappedTerms: []string{},
				},
				Retrieved: retrieved,
			},
		},
		{
			name:     "unknown term gets a suggestion",
			searcher: usersSearcher(),
			answer:   `{"entity": "users", "conditions": [], "unmapped_terms": ["account state"]}`,
			want: Run{
				Result: QueryResult{
					Status:     model.QueryStatusRejected,
					Warnings:   []planner.Warning{},
					Rejections: []ir.Rejection{{Code: constants.RejectCodeUnknownField, Field: "account state", Suggestion: "status"}},
				},
				Extraction: extraction.Extraction{
					Query:         ir.Query{Entity: "users", Conditions: []ir.Condition{}},
					UnmappedTerms: []string{"account state"},
				},
				Retrieved: retrieved,
			},
		},
		{
			name:     "unusable answer",
			searcher: usersSearcher(),
			answer:   `not json`,
			want: Run{
				Result: QueryResult{
					Status:     model.QueryStatusRejected,
					Warnings:   []planner.Warning{},
					Rejections: []ir.Rejection{{Code: constants.RejectCodeExtractionFailed}},
				},
				Retrieved: retrieved,
			},
		},
		{
			name:     "workspace without schemas",
			searcher: &fakePatternSearcher{},
			want: Run{
				Result: QueryResult{
					Status:     model.QueryStatusRejected,
					Warnings:   []planner.Warning{},
					Rejections: []ir.Rejection{{Code: constants.RejectCodeNoSchemas}},
				},
				Retrieved: []model.RetrievedPattern{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := newTestRunner(tc.searcher, &fakeChatter{answer: tc.answer}).
				Run(context.Background(), "ws-1", runRequest, extractNow)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("run mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunRetrievalFailure(t *testing.T) {
	failure := errors.New("connection refused")
	runner := newTestRunner(&fakePatternSearcher{err: failure}, &fakeChatter{})

	if _, err := runner.Run(context.Background(), "ws-1", runRequest, extractNow); !errors.Is(err, failure) {
		t.Fatalf("err = %v, want it to wrap %v", err, failure)
	}
}
