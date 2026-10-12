package service

import (
	"context"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
)

func (s *WorkspaceService) Get(ctx context.Context, workspaceID string) (model.Workspace, error) {
	return model.Workspace{}, apperror.NotImplemented("get workspace is not implemented")
}

func (s *WorkspaceService) UpdateConnection(
	ctx context.Context,
	workspaceID string,
	req types.UpdateConnectionRequest,
) (model.Workspace, error) {
	return model.Workspace{}, apperror.NotImplemented("update connection is not implemented")
}
