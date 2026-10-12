package model

import (
	"encoding/json"
	"time"
)

type QueryExecution struct {
	ID               string          `db:"id"                 json:"id"`
	QueryID          string          `db:"query_id"           json:"query_id"`
	StartKey         json.RawMessage `db:"start_key"          json:"start_key"`
	LastEvaluatedKey json.RawMessage `db:"last_evaluated_key" json:"last_evaluated_key"`
	ItemCount        int             `db:"item_count"         json:"item_count"`
	ScannedCount     int             `db:"scanned_count"      json:"scanned_count"`
	ConsumedCapacity *float64        `db:"consumed_capacity"  json:"consumed_capacity"`
	Error            *string         `db:"error"              json:"error"`
	DurationMS       int             `db:"duration_ms"        json:"duration_ms"`
	CreatedAt        time.Time       `db:"created_at"         json:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at"         json:"updated_at"`
}
