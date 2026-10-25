package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/retrieval"
)

// Calibrated on qwen3-embedding:0.6b against the users table: real matches scored 0.64 to 0.77 ("first name" ->
// name, "signup date" -> created_at), unrelated terms 0.49 to 0.52 ("phone number", "salary"). Recalibrate when
// the embedding model changes.
const minSuggestionSimilarity = 0.6

// Suggest fills Suggestion on every UNKNOWN_FIELD rejection with the table's closest field, when one is close
// enough. Other rejections are returned unchanged.
func (s *FieldSuggester) Suggest(
	ctx context.Context,
	rejections []ir.Rejection,
	table dynamoschema.Table,
) ([]ir.Rejection, error) {
	var unknown []int
	for i, r := range rejections {
		if r.Code == constants.RejectCodeUnknownField {
			unknown = append(unknown, i)
		}
	}
	if len(unknown) == 0 || len(table.Attributes) == 0 {
		return rejections, nil
	}

	inputs := make([]string, 0, len(unknown)+len(table.Attributes))
	for _, i := range unknown {
		inputs = append(inputs, retrieval.TermText(rejections[i].Field))
	}
	for _, a := range table.Attributes {
		inputs = append(inputs, retrieval.FieldText(a))
	}
	vectors, err := s.embedder.Embed(ctx, s.embeddingModel, inputs)
	if err != nil {
		return nil, fmt.Errorf("embedding unknown fields: %w", err)
	}

	suggested := slices.Clone(rejections)
	fieldVectors := vectors[len(unknown):]
	for n, i := range unknown {
		best, similarity := retrieval.Closest(vectors[n], fieldVectors)
		if similarity >= minSuggestionSimilarity {
			suggested[i].Suggestion = table.Attributes[best].Name
		}
	}
	return suggested, nil
}
