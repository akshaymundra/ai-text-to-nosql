package handler

import (
	"go.uber.org/zap"

	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
)

type AuthHandler struct {
	AuthService *service.AuthService
	Logger      *zap.Logger
}

func NewAuthHandler(svc *service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		AuthService: svc,
		Logger:      logger,
	}
}

type WorkspaceHandler struct {
	WorkspaceService *service.WorkspaceService
	Logger           *zap.Logger
}

func NewWorkspaceHandler(svc *service.WorkspaceService, logger *zap.Logger) *WorkspaceHandler {
	return &WorkspaceHandler{
		WorkspaceService: svc,
		Logger:           logger,
	}
}

type TableSchemaHandler struct {
	TableSchemaService *service.TableSchemaService
	Logger             *zap.Logger
}

func NewTableSchemaHandler(svc *service.TableSchemaService, logger *zap.Logger) *TableSchemaHandler {
	return &TableSchemaHandler{
		TableSchemaService: svc,
		Logger:             logger,
	}
}

type QueryHandler struct {
	QueryService *service.QueryService
	Logger       *zap.Logger
}

func NewQueryHandler(svc *service.QueryService, logger *zap.Logger) *QueryHandler {
	return &QueryHandler{
		QueryService: svc,
		Logger:       logger,
	}
}
