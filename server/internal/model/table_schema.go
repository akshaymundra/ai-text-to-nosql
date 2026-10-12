package model

import (
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
)

type TableSchema struct {
	ID          string             `db:"id"           json:"id"`
	WorkspaceID string             `db:"workspace_id" json:"workspace_id"`
	TableName   string             `db:"table_name"   json:"table_name"`
	Definition  dynamoschema.Table `db:"definition"   json:"definition"`
	ContentHash string             `db:"content_hash" json:"content_hash"`
	CreatedAt   time.Time          `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time          `db:"updated_at"   json:"updated_at"`
}
