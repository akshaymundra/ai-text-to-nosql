package types

import "github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"

type PutTableSchemaRequest struct {
	Definition dynamoschema.Table `json:"definition" validate:"required"`
}

func (r PutTableSchemaRequest) Validate() error {
	return validateStruct(r)
}
