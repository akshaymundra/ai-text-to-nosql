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
	workspaces workspaceStore
	jwtSecret  []byte
	tokenTTL   time.Duration
}

func NewAuthService(workspaces workspaceStore, jwtSecret string, tokenTTL time.Duration) *AuthService {
	return &AuthService{
		workspaces: workspaces,
		jwtSecret:  []byte(jwtSecret),
		tokenTTL:   tokenTTL,
	}
}

type WorkspaceService struct{}

func NewWorkspaceService() *WorkspaceService {
	return &WorkspaceService{}
}

type TableSchemaService struct{}

func NewTableSchemaService() *TableSchemaService {
	return &TableSchemaService{}
}

type QueryService struct{}

func NewQueryService() *QueryService {
	return &QueryService{}
}
