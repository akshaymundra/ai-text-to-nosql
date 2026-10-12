package model

import (
	"encoding/json"
	"time"
)

type QueryStatus string

const (
	QueryStatusValid    QueryStatus = "valid"
	QueryStatusRejected QueryStatus = "rejected"
)

type RetrievedPattern struct {
	AccessPatternID string  `json:"access_pattern_id"`
	Distance        float64 `json:"distance"`
}

type Query struct {
	ID             string             `db:"id"              json:"id"`
	WorkspaceID    string             `db:"workspace_id"    json:"workspace_id"`
	TableSchemaID  *string            `db:"table_schema_id" json:"table_schema_id"`
	UserText       string             `db:"user_text"       json:"user_text"`
	Retrieved      []RetrievedPattern `db:"retrieved"       json:"retrieved"`
	IR             json.RawMessage    `db:"ir"              json:"ir"`
	Status         QueryStatus        `db:"status"          json:"status"`
	RejectionCode  *string            `db:"rejection_code"  json:"rejection_code"`
	RejectionField *string            `db:"rejection_field" json:"rejection_field"`
	TableName      *string            `db:"table_name"      json:"table_name"`
	IndexName      *string            `db:"index_name"      json:"index_name"`
	SchemaHash     *string            `db:"schema_hash"     json:"schema_hash"`
	QueryInput     json.RawMessage    `db:"query_input"     json:"query_input"`
	Warnings       []string           `db:"warnings"        json:"warnings"`
	Explanation    string             `db:"explanation"     json:"explanation"`
	LLMModel       string             `db:"llm_model"       json:"llm_model"`
	PromptVersion  string             `db:"prompt_version"  json:"prompt_version"`
	LatencyMS      int                `db:"latency_ms"      json:"latency_ms"`
	CreatedAt      time.Time          `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time          `db:"updated_at"      json:"updated_at"`
}
