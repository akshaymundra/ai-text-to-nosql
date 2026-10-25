package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pgvector/pgvector-go"

	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

type fakePatternSearcher struct {
	gotWorkspaceID string
	gotEmbedding   pgvector.Vector
	gotModel       string
	gotLimit       int
	matches        []model.AccessPatternMatch
	err            error
}

func (f *fakePatternSearcher) Search(
	ctx context.Context,
	workspaceID string,
	embedding pgvector.Vector,
	embeddingModel string,
	limit int,
) ([]model.AccessPatternMatch, error) {
	f.gotWorkspaceID = workspaceID
	f.gotEmbedding = embedding
	f.gotModel = embeddingModel
	f.gotLimit = limit
	return f.matches, f.err
}

func match(id, table string, distance float64) model.AccessPatternMatch {
	return model.AccessPatternMatch{
		ID:         id,
		TableName:  table,
		Definition: dynamoschema.Table{Name: table},
		Distance:   distance,
	}
}

func TestRetrieve(t *testing.T) {
	searcher := &fakePatternSearcher{matches: []model.AccessPatternMatch{
		match("ap-1", "invoices", 0.21),
		match("ap-2", "invoices", 0.30),
		match("ap-3", "organizations", 0.42),
		match("ap-4", "users", 0.47),
		match("ap-5", "tasks", 0.51),
	}}
	embedder := &fakeEmbedder{}
	retriever := NewRetriever(searcher, embedder, testEmbeddingModel)

	got, err := retriever.Retrieve(context.Background(), "ws-1", "Show all overdue invoices")
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}

	wantInputs := []string{
		"Instruct: Given a request for data, retrieve the database table access patterns that can answer it\n" +
			"Query:Show all overdue invoices",
	}
	if diff := cmp.Diff(wantInputs, embedder.gotInputs); diff != "" {
		t.Errorf("embedded text mismatch (-want +got):\n%s", diff)
	}
	if embedder.gotModel != testEmbeddingModel || searcher.gotModel != testEmbeddingModel {
		t.Errorf("models = %q and %q, want %q for both", embedder.gotModel, searcher.gotModel, testEmbeddingModel)
	}
	if searcher.gotWorkspaceID != "ws-1" || searcher.gotLimit != retrievalPatternLimit {
		t.Errorf("search scope = %q limit %d, want ws-1 limit %d", searcher.gotWorkspaceID, searcher.gotLimit, retrievalPatternLimit)
	}
	if diff := cmp.Diff(pgvector.NewVector([]float32{0, 1}), searcher.gotEmbedding, cmp.AllowUnexported(pgvector.Vector{})); diff != "" {
		t.Errorf("searched vector mismatch (-want +got):\n%s", diff)
	}

	want := Retrieved{
		Tables: []dynamoschema.Table{{Name: "invoices"}, {Name: "organizations"}, {Name: "users"}},
		Patterns: []model.RetrievedPattern{
			{AccessPatternID: "ap-1", Distance: 0.21},
			{AccessPatternID: "ap-2", Distance: 0.30},
			{AccessPatternID: "ap-3", Distance: 0.42},
			{AccessPatternID: "ap-4", Distance: 0.47},
			{AccessPatternID: "ap-5", Distance: 0.51},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("retrieved mismatch (-want +got):\n%s", diff)
	}
}

func TestRetrieveErrors(t *testing.T) {
	failure := errors.New("boom")

	tests := []struct {
		name     string
		embedder *fakeEmbedder
		searcher *fakePatternSearcher
	}{
		{name: "embedding fails", embedder: &fakeEmbedder{err: failure}, searcher: &fakePatternSearcher{}},
		{name: "search fails", embedder: &fakeEmbedder{}, searcher: &fakePatternSearcher{err: failure}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewRetriever(tc.searcher, tc.embedder, testEmbeddingModel).
				Retrieve(context.Background(), "ws-1", "overdue invoices")
			if !errors.Is(err, failure) {
				t.Fatalf("err = %v, want it to wrap %v", err, failure)
			}
		})
	}
}
