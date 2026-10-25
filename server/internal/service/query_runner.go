package service

import (
	"context"
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/extraction"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

// DefaultQueryLimit caps the items one compiled Query evaluates; callers page with the returned cursor.
const DefaultQueryLimit int32 = 25

// Run is everything one request produced, kept so it can be stored and so evals can tell an extraction mistake
// from a pipeline bug.
type Run struct {
	Result     QueryResult
	Extraction extraction.Extraction
	Retrieved  []model.RetrievedPattern
}

// Run turns a request into a compiled query or a rejection: retrieve tables, extract the IR, run the
// deterministic stages, then suggest fields for any unknown ones.
func (r *QueryRunner) Run(ctx context.Context, workspaceID, request string, now time.Time) (Run, error) {
	retrieved, err := r.retriever.Retrieve(ctx, workspaceID, request)
	if err != nil {
		return Run{}, err
	}

	extracted, rejection, err := r.extractor.Extract(ctx, retrieved.Tables, request, now)
	if err != nil {
		return Run{}, err
	}
	if rejection != nil {
		return Run{Result: rejected([]ir.Rejection{*rejection}), Retrieved: retrieved.Patterns}, nil
	}

	result, err := BuildQuery(extracted, retrieved.Tables, DefaultQueryLimit)
	if err != nil {
		return Run{}, err
	}
	if table, ok := dynamoschema.FindTable(retrieved.Tables, extracted.Query.Entity); ok {
		result.Rejections, err = r.suggester.Suggest(ctx, result.Rejections, table)
		if err != nil {
			return Run{}, err
		}
	}

	return Run{Result: result, Extraction: extracted, Retrieved: retrieved.Patterns}, nil
}
