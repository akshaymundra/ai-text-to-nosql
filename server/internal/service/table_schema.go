package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pgvector/pgvector-go"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/repo"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/retrieval"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
)

func (s *TableSchemaService) List(ctx context.Context, workspaceID string) ([]model.TableSchema, error) {
	return s.schemas.List(ctx, workspaceID)
}

func (s *TableSchemaService) Get(ctx context.Context, workspaceID, tableName string) (model.TableSchema, error) {
	schema, err := s.schemas.Get(ctx, workspaceID, tableName)
	if errors.Is(err, repo.ErrNotFound) {
		return model.TableSchema{}, tableSchemaNotFound(tableName)
	}
	return schema, err
}

// Put always re-embeds, so a change of embedding model can never leave a table with vectors that search skips.
func (s *TableSchemaService) Put(
	ctx context.Context,
	workspaceID, tableName string,
	req types.PutTableSchemaRequest,
) (model.TableSchema, error) {
	definition := req.Definition
	if definition.Name != tableName {
		return model.TableSchema{}, apperror.BadRequest(
			fmt.Sprintf("definition table %q does not match %q in the URL", definition.Name, tableName),
		)
	}
	if err := definition.Check(); err != nil {
		return model.TableSchema{}, apperror.BadRequest(strings.ReplaceAll(err.Error(), "\n", "; "))
	}

	hash, err := definition.ContentHash()
	if err != nil {
		return model.TableSchema{}, err
	}
	patterns, err := s.embedAccessPatterns(ctx, definition)
	if err != nil {
		return model.TableSchema{}, err
	}

	return s.schemas.Save(ctx, model.TableSchema{
		WorkspaceID: workspaceID,
		TableName:   tableName,
		Definition:  definition,
		ContentHash: hash,
	}, patterns)
}

func (s *TableSchemaService) Delete(ctx context.Context, workspaceID, tableName string) error {
	err := s.schemas.Delete(ctx, workspaceID, tableName)
	if errors.Is(err, repo.ErrNotFound) {
		return tableSchemaNotFound(tableName)
	}
	return err
}

func (s *TableSchemaService) embedAccessPatterns(
	ctx context.Context,
	definition dynamoschema.Table,
) ([]model.AccessPattern, error) {
	texts := make([]string, len(definition.AccessPatterns))
	for i, p := range definition.AccessPatterns {
		texts[i] = retrieval.DocumentText(definition, p)
	}
	vectors, err := s.embedder.Embed(ctx, s.embeddingModel, texts)
	if err != nil {
		return nil, fmt.Errorf("embedding access patterns of %q: %w", definition.Name, err)
	}

	patterns := make([]model.AccessPattern, len(definition.AccessPatterns))
	for i, p := range definition.AccessPatterns {
		patterns[i] = model.AccessPattern{
			IndexName:      indexName(p),
			Description:    p.Description,
			Embedding:      pgvector.NewVector(vectors[i]),
			EmbeddingModel: s.embeddingModel,
		}
	}
	return patterns, nil
}

func indexName(p dynamoschema.AccessPattern) *string {
	if p.Index == "" {
		return nil
	}
	return &p.Index
}

func tableSchemaNotFound(tableName string) error {
	return apperror.NotFound(fmt.Sprintf("table schema %q not found", tableName))
}
