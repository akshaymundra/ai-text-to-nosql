package service

import (
	"context"
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
)

type workspaceStore interface {
	Create(ctx context.Context, ws model.Workspace) (model.Workspace, error)
	GetByName(ctx context.Context, name string) (model.Workspace, error)
}

type AuthService struct {
	workspaces           workspaceStore
	jwtSecret            []byte
	tokenTTL             time.Duration
	allowLocalConnection bool
}

func NewAuthService(
	workspaces workspaceStore,
	jwtSecret string,
	tokenTTL time.Duration,
	allowLocalConnection bool,
) *AuthService {
	return &AuthService{
		workspaces:           workspaces,
		jwtSecret:            []byte(jwtSecret),
		tokenTTL:             tokenTTL,
		allowLocalConnection: allowLocalConnection,
	}
}

type WorkspaceService struct {
	allowLocalConnection bool
}

func NewWorkspaceService(allowLocalConnection bool) *WorkspaceService {
	return &WorkspaceService{allowLocalConnection: allowLocalConnection}
}

type TableSchemaService struct{}

func NewTableSchemaService() *TableSchemaService {
	return &TableSchemaService{}
}

type QueryService struct{}

func NewQueryService() *QueryService {
	return &QueryService{}
}
