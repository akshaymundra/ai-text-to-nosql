package service

import (
	"context"
	"fmt"

	"github.com/pgvector/pgvector-go"

	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/retrieval"
)

// Starting values; retune them from the retrieval score over the eval cases.
const (
	retrievalPatternLimit = 5
	retrievalMaxTables    = 3
)

type Retrieved struct {
	Tables   []dynamoschema.Table
	Patterns []model.RetrievedPattern
}

func (r *Retriever) Retrieve(ctx context.Context, workspaceID, request string) (Retrieved, error) {
	vectors, err := r.embedder.Embed(ctx, r.embeddingModel, []string{retrieval.QueryText(request)})
	if err != nil {
		return Retrieved{}, fmt.Errorf("embedding request: %w", err)
	}
	matches, err := r.patterns.Search(
		ctx,
		workspaceID,
		pgvector.NewVector(vectors[0]),
		r.embeddingModel,
		retrievalPatternLimit,
	)
	if err != nil {
		return Retrieved{}, err
	}

	ranked := make([]dynamoschema.Table, len(matches))
	patterns := make([]model.RetrievedPattern, len(matches))
	for i, m := range matches {
		ranked[i] = m.Definition
		patterns[i] = model.RetrievedPattern{AccessPatternID: m.ID, Distance: m.Distance}
	}
	return Retrieved{
		Tables:   retrieval.SelectTables(ranked, retrievalMaxTables),
		Patterns: patterns,
	}, nil
}
