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
	if err := checkConnectionType(req.ConnectionType, s.allowLocalConnection); err != nil {
		return model.Workspace{}, err
	}
	return model.Workspace{}, apperror.NotImplemented("update connection is not implemented")
}

// A local endpoint makes the server call a URL the client chose, so it stays a development-only option.
func checkConnectionType(connectionType model.ConnectionType, allowLocalConnection bool) error {
	if connectionType == model.ConnectionTypeLocal && !allowLocalConnection {
		return apperror.BadRequest("connection_type local is only available in development")
	}
	return nil
}
