package service

import (
	"context"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/compiler"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/dates"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/planner"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/validator"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
)

type QueryResult struct {
	Status     model.QueryStatus
	Table      string
	Index      string
	Query      *compiler.QueryInput
	Warnings   []planner.Warning
	Rejections []ir.Rejection
}

func (s *QueryService) Create(ctx context.Context, workspaceID string, req types.CreateQueryRequest) (model.Query, error) {
	return model.Query{}, apperror.NotImplemented("create query is not implemented")
}

func (s *QueryService) List(ctx context.Context, workspaceID string) ([]model.Query, error) {
	return nil, apperror.NotImplemented("list queries is not implemented")
}

func (s *QueryService) Get(ctx context.Context, workspaceID, queryID string) (model.Query, error) {
	return model.Query{}, apperror.NotImplemented("get query is not implemented")
}

func (s *QueryService) Execute(
	ctx context.Context,
	workspaceID, queryID string,
	req types.ExecuteQueryRequest,
) (types.ExecuteQueryResponse, error) {
	return types.ExecuteQueryResponse{}, apperror.NotImplemented("execute query is not implemented")
}

func (s *QueryService) ListExecutions(ctx context.Context, workspaceID, queryID string) ([]model.QueryExecution, error) {
	return nil, apperror.NotImplemented("list query executions is not implemented")
}

// BuildQuery runs the deterministic stages on an extracted IR. It does no I/O, so the eval harness calls it directly.
func BuildQuery(q ir.Query, tables []dynamoschema.Table, limit int32) (QueryResult, error) {
	table, rejections := validator.Validate(q, tables)
	if len(rejections) > 0 {
		return rejected(rejections), nil
	}
	q, rejections = dates.Normalize(q, table)
	if len(rejections) > 0 {
		return rejected(rejections), nil
	}
	path, rejection := planner.Plan(q, table)
	if rejection != nil {
		return rejected([]ir.Rejection{*rejection}), nil
	}
	input, err := compiler.Compile(path, limit)
	if err != nil {
		return QueryResult{}, err
	}

	return QueryResult{
		Status:     model.QueryStatusValid,
		Table:      path.Table,
		Index:      path.Index,
		Query:      &input,
		Warnings:   path.Warnings(),
		Rejections: []ir.Rejection{},
	}, nil
}

func rejected(rejections []ir.Rejection) QueryResult {
	return QueryResult{
		Status:     model.QueryStatusRejected,
		Warnings:   []planner.Warning{},
		Rejections: rejections,
	}
}
