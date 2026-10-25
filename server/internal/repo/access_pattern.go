package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

type AccessPatternRepo struct {
	db *pgxpool.Pool
}

func NewAccessPatternRepo(db *pgxpool.Pool) *AccessPatternRepo {
	return &AccessPatternRepo{db: db}
}

// Search returns the workspace's access patterns closest to the embedding by cosine distance. Only rows embedded
// with the same model are comparable, so the model is part of the filter.
func (r *AccessPatternRepo) Search(
	ctx context.Context,
	workspaceID string,
	embedding pgvector.Vector,
	embeddingModel string,
	limit int,
) ([]model.AccessPatternMatch, error) {
	rows, err := r.db.Query(ctx,
		`SELECT ap.id, ts.table_name, ap.index_name, ap.description, ts.definition,
			ap.embedding <=> $2 AS distance
		FROM access_patterns ap
		JOIN table_schemas ts ON ts.id = ap.table_schema_id
		WHERE ap.workspace_id = $1 AND ap.embedding_model = $3
		ORDER BY ap.embedding <=> $2
		LIMIT $4`,
		workspaceID, embedding, embeddingModel, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("searching access patterns: %w", err)
	}

	matches, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.AccessPatternMatch])
	if err != nil {
		return nil, fmt.Errorf("searching access patterns: %w", err)
	}
	return matches, nil
}
