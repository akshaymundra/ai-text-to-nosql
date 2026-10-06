package routes

import (
	"github.com/akshaymundra/ai-text-to-nosql/internal/handler"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"

	"github.com/go-chi/chi/v5"
)

func QueryRoutes() func(router chi.Router) {
	svc := service.NewQueryService()
	h := handler.NewQueryHandler(svc)
	return func(r chi.Router) {
		_ = h
	}
}
