package model

import "time"

type AccessPattern struct {
	ID             string    `db:"id"              json:"id"`
	WorkspaceID    string    `db:"workspace_id"    json:"workspace_id"`
	TableSchemaID  string    `db:"table_schema_id" json:"table_schema_id"`
	IndexName      *string   `db:"index_name"      json:"index_name"`
	Description    string    `db:"description"     json:"description"`
	Embedding      []float32 `db:"embedding"       json:"embedding"`
	EmbeddingModel string    `db:"embedding_model" json:"embedding_model"`
	CreatedAt      time.Time `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"      json:"updated_at"`
}
