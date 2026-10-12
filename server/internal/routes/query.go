package routes

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/akshaymundra/ai-text-to-nosql/internal/handler"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
)

func QueryRoutes(logger *zap.Logger) func(router chi.Router) {
	svc := service.NewQueryService()
	h := handler.NewQueryHandler(svc, logger)
	return func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{queryID}", h.Get)
		r.Post("/{queryID}/execute", h.Execute)
		r.Get("/{queryID}/executions", h.ListExecutions)
	}
}
