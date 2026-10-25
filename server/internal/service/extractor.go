package service

import (
	"context"
	"errors"
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/integrations/ollama"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/extraction"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

// Extract asks the model for the request's IR. An unusable answer is a rejection, never a retry: retries would
// make results depend on luck and hide prompt problems the eval set should show.
func (e *Extractor) Extract(
	ctx context.Context,
	tables []dynamoschema.Table,
	request string,
	now time.Time,
) (extraction.Extraction, *ir.Rejection, error) {
	if len(tables) == 0 {
		return extraction.Extraction{}, &ir.Rejection{Code: constants.RejectCodeNoSchemas}, nil
	}
	schema, err := extraction.Schema(tables)
	if err != nil {
		return extraction.Extraction{}, nil, err
	}

	answer, err := e.chat.Chat(ctx, ollama.ChatRequest{
		Model: e.model,
		Messages: []ollama.Message{
			{Role: ollama.RoleSystem, Content: extraction.SystemPrompt(tables, now)},
			{Role: ollama.RoleUser, Content: request},
		},
		Format:  schema,
		Options: ollama.Options{Temperature: 0, Seed: e.seed},
	})
	if errors.Is(err, ollama.ErrIncompleteAnswer) {
		return extraction.Extraction{}, &ir.Rejection{Code: constants.RejectCodeExtractionFailed}, nil
	}
	if err != nil {
		return extraction.Extraction{}, nil, err
	}

	extracted, err := extraction.Parse(answer)
	if err != nil {
		return extraction.Extraction{}, &ir.Rejection{Code: constants.RejectCodeExtractionFailed}, nil
	}
	return extracted, nil, nil
}
