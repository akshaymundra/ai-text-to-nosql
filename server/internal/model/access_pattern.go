package model

import (
	"time"

	"github.com/pgvector/pgvector-go"

	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
)

type AccessPattern struct {
	ID             string          `db:"id"              json:"id"`
	WorkspaceID    string          `db:"workspace_id"    json:"workspace_id"`
	TableSchemaID  string          `db:"table_schema_id" json:"table_schema_id"`
	IndexName      *string         `db:"index_name"      json:"index_name"`
	Description    string          `db:"description"     json:"description"`
	Embedding      pgvector.Vector `db:"embedding"       json:"-"`
	EmbeddingModel string          `db:"embedding_model" json:"embedding_model"`
	CreatedAt      time.Time       `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at"      json:"updated_at"`
}

// AccessPatternMatch is one nearest-neighbour row: the pattern, the table it belongs to and its cosine distance
// to the request.
type AccessPatternMatch struct {
	ID          string             `db:"id"          json:"id"`
	TableName   string             `db:"table_name"  json:"table_name"`
	IndexName   *string            `db:"index_name"  json:"index_name"`
	Description string             `db:"description" json:"description"`
	Definition  dynamoschema.Table `db:"definition"  json:"definition"`
	Distance    float64            `db:"distance"    json:"distance"`
}
