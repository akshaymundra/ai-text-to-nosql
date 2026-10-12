package types

import (
	"encoding/json"

	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

type CreateQueryRequest struct {
	Query string `json:"query" validate:"required"`
}

func (r CreateQueryRequest) Validate() error {
	return validateStruct(r)
}

type ExecuteQueryRequest struct {
	StartKey json.RawMessage `json:"start_key,omitempty"`
}

func (r ExecuteQueryRequest) Validate() error {
	return validateStruct(r)
}

type ExecuteQueryResponse struct {
	Execution model.QueryExecution `json:"execution"`
	Items     []map[string]any     `json:"items"`
}
