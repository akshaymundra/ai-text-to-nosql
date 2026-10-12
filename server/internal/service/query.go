package service

import (
	"context"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
)

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
