package routes

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/akshaymundra/ai-text-to-nosql/internal/handler"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
)

func TableSchemaRoutes(svc *service.TableSchemaService, logger *zap.Logger) func(router chi.Router) {
	h := handler.NewTableSchemaHandler(svc, logger)
	return func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{tableName}", h.Get)
		r.Put("/{tableName}", h.Put)
		r.Delete("/{tableName}", h.Delete)
	}
}
