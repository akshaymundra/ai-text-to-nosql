package types

import (
	"encoding/json"

	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

type CreateQueryRequest struct {
	Query string `json:"query"`
}

type ExecuteQueryRequest struct {
	StartKey json.RawMessage `json:"start_key,omitempty"`
}

type ExecuteQueryResponse struct {
	Execution model.QueryExecution `json:"execution"`
	Items     []map[string]any     `json:"items"`
}
