package service

import (
	"context"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
)

func (s *TableSchemaService) List(ctx context.Context, workspaceID string) ([]model.TableSchema, error) {
	return nil, apperror.NotImplemented("list table schemas is not implemented")
}

func (s *TableSchemaService) Get(ctx context.Context, workspaceID, tableName string) (model.TableSchema, error) {
	return model.TableSchema{}, apperror.NotImplemented("get table schema is not implemented")
}

func (s *TableSchemaService) Put(
	ctx context.Context,
	workspaceID, tableName string,
	req types.PutTableSchemaRequest,
) (model.TableSchema, error) {
	return model.TableSchema{}, apperror.NotImplemented("put table schema is not implemented")
}

func (s *TableSchemaService) Delete(ctx context.Context, workspaceID, tableName string) error {
	return apperror.NotImplemented("delete table schema is not implemented")
}
