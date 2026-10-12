package routes

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/akshaymundra/ai-text-to-nosql/internal/handler"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
)

func WorkspaceRoutes(logger *zap.Logger) func(router chi.Router) {
	svc := service.NewWorkspaceService()
	h := handler.NewWorkspaceHandler(svc, logger)
	return func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/connection", h.UpdateConnection)
	}
}
